#!/usr/bin/env python3
"""
sync-backlog-to-issues.py — sincronizza docs/backlog.md → issue GitHub (IDEMPOTENTE).

Sorgente di verità UNICA: docs/backlog.md. Ogni item ha un `id` stabile. Per ogni item con
status != done lo script garantisce una issue aperta; gli item rimossi o `status: done` chiudono
la issue corrispondente. Re-run senza duplicati grazie al fingerprint nascosto nel corpo issue:
    <!-- backlog-id: <id> | hash: <sha> -->
+ label `backlog-sync` su tutte le issue gestite.

Matrice (idempotente):
  item open  + nessuna issue            → CREATE
  item open  + issue aperta, hash ==    → SKIP (già allineata)
  item open  + issue aperta, hash !=    → UPDATE (titolo/labels/corpo divergenti)
  item open  + issue chiusa             → REOPEN (+ riallinea contenuto)
  item done/rimosso + issue aperta      → CLOSE (commento auto + estratto del corpo dal backlog)

Rilevamento modifiche: il fingerprint include un hash del contenuto; se l'hash del backlog
differisce da quello embeddato nella issue → UPDATE.

MILESTONE DINAMICHE: l'item può avere `- **milestone**: <titolo>`. Non esiste un elenco di
milestone hardcoded: lo script cerca la milestone del repo per titolo esatto e la **CREA se
manca** (idempotente), poi assegna la issue. Il titolo è incluso nell'hash → cambiarlo/toglierlo
riallinea la issue al prossimo sync. Non rimuove mai una milestone assegnata a mano se l'item
non ne dichiara una.

Differenza rispetto al gemello GitLab: l'API GitHub non ha `add_labels`, la PATCH **sostituisce**
l'intero set di label. Per non perdere le label aggiunte a mano in UI, lo script invia sempre
l'UNIONE fra le label esistenti sulla issue e quelle desiderate.

Sicurezza: opera SOLO sulle issue che portano la label `backlog-sync`. Le issue di terzi
(Renovate/Dependabot, segnalazioni utenti) non vengono mai lette né chiuse. Le pull request,
che l'endpoint issues restituisce insieme alle issue, sono filtrate via campo `pull_request`.

Auth: token con scope `repo` (`issues: write` su repo pubblico basta `public_repo`).
  - $GITHUB_TOKEN (in CI: `${{ secrets.GITHUB_TOKEN }}`)
  - oppure `gh auth token` in locale (rilevato automaticamente)
  - oppure --token-file <path>

Default: --dry-run (mostra il piano, NON scrive). Usa --apply per eseguire davvero.

Requires: solo stdlib (urllib, json, re, argparse) + `gh` opzionale per il token in locale.
"""
import argparse
import json
import os
import re
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, "lib"))
import backlog as B   # noqa: E402  — parsing/regole condivisi con backlog-lint e generate-roadmap

DEFAULT_REPO = "Allan-Nava/Go-DDD-Scaffold"
DEFAULT_BRANCH = "main"
SYNC_LABEL = "backlog-sync"
API = "https://api.github.com"

FP_RE = B.FP_RE
HASH_RE = B.HASH_RE
item_hash = B.item_hash


# ----------------------------------------------------------------------------- parsing
def parse_backlog(path):
    """Parsa il backlog (via lib/backlog.py) e rifiuta gli id duplicati (fatale per il sync)."""
    items = B.parse_backlog(path)
    counts = {}
    for it in items:
        counts[it["id"]] = counts.get(it["id"], 0) + 1
    dups = sorted(k for k, n in counts.items() if n > 1)
    if dups:
        sys.exit(f"ERRORE: id duplicati in backlog.md: {dups}")
    return items


def build_body(item, backlog_url):
    parts = []
    body = "\n".join(item["body"]).strip()
    if body:
        parts.append(body)
    meta = []
    if item["priority"]:
        meta.append(f"**Priorità**: {item['priority']}")
    if item.get("milestone"):
        meta.append(f"**Milestone**: {item['milestone']}")
    if item.get("owner"):
        meta.append(f"**Owner**: @{item['owner']}")
    if item["ref"]:
        meta.append(f"**Ref**: {item['ref']}")
    if meta:
        parts.append("\n".join(meta))
    parts.append(f"_Issue generata automaticamente da [`docs/backlog.md`]({backlog_url}) "
                 f"(id: `{item['id']}`). Non editare il titolo a mano: modifica il backlog._")
    parts.append(f"<!-- backlog-id: {item['id']} | hash: {item_hash(item)} -->")
    return "\n\n".join(parts)


CHECKLIST_RE = re.compile(r"^[-*]\s*\[[ xX]\]")   # voce checklist markdown: - [ ] / - [x]


def close_note(item, max_len=700):
    """Estratto del corpo dell'item da allegare al commento di chiusura, per dare contesto/motivo.
    Preferisce il primo blocco *blockquote* (`> …`, per convenzione la nota di risoluzione),
    altrimenti il primo blocco di prosa (saltando le checklist). None se l'item è stato rimosso
    dal backlog (nessun corpo disponibile) o il corpo è vuoto."""
    if not item:
        return None
    blocks, cur = [], []
    for ln in item.get("body", []):
        if ln.strip():
            cur.append(ln.strip())
        elif cur:
            blocks.append(cur)
            cur = []
    if cur:
        blocks.append(cur)
    if not blocks:
        return None
    chosen = next((b for b in blocks if all(l.startswith(">") for l in b)), None)          # 1) blockquote
    if chosen is None:
        chosen = next((b for b in blocks if not all(CHECKLIST_RE.match(l) for l in b)), None)  # 2) prosa
    if not chosen:
        return None
    text = " ".join(re.sub(r"^>\s*", "", l) for l in chosen).strip()
    if len(text) > max_len:
        text = text[:max_len].rsplit(" ", 1)[0].rstrip() + "…"
    return text or None


# ----------------------------------------------------------------------------- API
class GitHub:
    def __init__(self, repo, token):
        self.repo = repo
        self.token = token
        self._milestone_cache = {}   # titolo → number (per `- **milestone**:`)
        self._labels = None          # set dei nomi label esistenti sul repo

    def _req(self, method, path, params=None, data=None):
        url = f"{API}{path}"
        if params:
            url += "?" + urllib.parse.urlencode(params)
        body = json.dumps(data).encode() if data is not None else None
        headers = {
            "Authorization": f"Bearer {self.token}",
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
            "User-Agent": "backlog-sync",
            # La lista issue di GitHub può servire una risposta di qualche secondo prima:
            # subito dopo una CREATE un re-run vedrebbe uno stato incompleto e creerebbe
            # doppioni. Questo riduce la finestra; la rete di sicurezza vera è il dedup
            # per backlog-id nel piano (vedi main()).
            "Cache-Control": "no-cache",
        }
        if body is not None:
            headers["Content-Type"] = "application/json"
        req = urllib.request.Request(url, data=body, method=method, headers=headers)
        try:
            with urllib.request.urlopen(req, timeout=30) as r:
                raw = r.read().decode()
                return json.loads(raw) if raw else None
        except urllib.error.HTTPError as e:
            detail = e.read().decode(errors="replace")[:300]
            if e.code == 404 and "/issues" in path:
                sys.exit("ERRORE 404 sulle issue: l'Issue Tracker è disabilitato sul repo? "
                         "Abilitalo (Settings ▸ General ▸ Features ▸ Issues) e riprova.")
            if e.code == 403 and "rate limit" in detail.lower():
                sys.exit(f"ERRORE: rate limit GitHub esaurito — {detail}")
            sys.exit(f"ERRORE API {method} {path}: HTTP {e.code} — {detail}")

    # --- milestone dinamiche ---------------------------------------------------
    def ensure_milestone(self, title):
        """Titolo milestone → number. La cerca sul repo (match esatto, anche fra le chiuse);
        se non esiste la CREA. Cache per non ripetere le GET/POST. None se `title` è vuoto."""
        if not title:
            return None
        if title in self._milestone_cache:
            return self._milestone_cache[title]

        num = None
        page = 1
        while True:
            data = self._req("GET", f"/repos/{self.repo}/milestones",
                             params={"state": "all", "per_page": 100, "page": page}) or []
            num = next((m["number"] for m in data if m.get("title") == title), None)
            if num is not None or len(data) < 100:
                break
            page += 1

        if num is None:
            print(f"      + milestone GitHub mancante -> CREATE «{title}»")
            res = self._req("POST", f"/repos/{self.repo}/milestones",
                            data={"title": title,
                                  "description": "Milestone gestita da docs/backlog.md "
                                                 "(campo `- **milestone**:` degli item)."})
            num = res["number"]
        self._milestone_cache[title] = num
        return num

    # --- label ----------------------------------------------------------------
    def ensure_labels(self, names):
        """Crea le label mancanti sul repo. GitHub le creerebbe implicitamente alla POST della
        issue, ma non alla PATCH: crearle esplicitamente rende UPDATE e CREATE simmetrici."""
        if self._labels is None:
            self._labels, page = set(), 1
            while True:
                data = self._req("GET", f"/repos/{self.repo}/labels",
                                 params={"per_page": 100, "page": page}) or []
                self._labels.update(l["name"] for l in data)
                if len(data) < 100:
                    break
                page += 1
        for name in names:
            if name not in self._labels:
                print(f"      + label mancante -> CREATE «{name}»")
                self._req("POST", f"/repos/{self.repo}/labels",
                          data={"name": name, "color": "ededed",
                                "description": "Gestita da docs/backlog.md"})
                self._labels.add(name)

    def list_sync_issues(self):
        """Solo le issue con label `backlog-sync` (le altre non sono affare di questo script).
        L'endpoint issues restituisce anche le PR: filtrate via campo `pull_request`."""
        out, page = [], 1
        while True:
            data = self._req("GET", f"/repos/{self.repo}/issues",
                             params={"labels": SYNC_LABEL, "state": "all",
                                     "per_page": 100, "page": page}) or []
            out.extend(i for i in data if "pull_request" not in i)
            if len(data) < 100:
                break
            page += 1
        return out

    def create_issue(self, title, body, labels, assignee=None, milestone=None):
        data = {"title": title, "body": body, "labels": sorted(labels)}
        if assignee:
            data["assignees"] = [assignee]
        if milestone:
            data["milestone"] = milestone
        return self._req("POST", f"/repos/{self.repo}/issues", data=data)

    def update_issue(self, number, title, body, labels, assignee=None, milestone=None):
        # labels è già l'UNIONE con quelle presenti: la PATCH sostituisce il set.
        data = {"title": title, "body": body, "labels": sorted(labels)}
        if assignee:
            data["assignees"] = [assignee]
        if milestone:
            data["milestone"] = milestone
        return self._req("PATCH", f"/repos/{self.repo}/issues/{number}", data=data)

    def comment(self, number, body):
        return self._req("POST", f"/repos/{self.repo}/issues/{number}/comments",
                         data={"body": body})

    def set_state(self, number, state, reason=None):   # 'closed' | 'open'
        data = {"state": state}
        if reason:
            data["state_reason"] = reason
        return self._req("PATCH", f"/repos/{self.repo}/issues/{number}", data=data)


def discover_token(token_file):
    """$GITHUB_TOKEN → --token-file → `gh auth token`. Vuoto se nessuno disponibile."""
    tok = os.environ.get("GITHUB_TOKEN", "").strip()
    if tok:
        return tok
    if token_file:
        with open(token_file) as f:
            return f.read().strip()
    try:
        out = subprocess.run(["gh", "auth", "token"], capture_output=True, text=True, timeout=10)
        if out.returncode == 0:
            return out.stdout.strip()
    except (OSError, subprocess.SubprocessError):
        pass
    return ""


# ----------------------------------------------------------------------------- main
def main():
    ap = argparse.ArgumentParser(description="Sincronizza docs/backlog.md → issue GitHub (idempotente).")
    ap.add_argument("--backlog", default=os.path.join(HERE, "..", "backlog.md"))
    ap.add_argument("--repo", default=os.environ.get("GITHUB_REPOSITORY") or DEFAULT_REPO,
                    help="owner/repo (default: $GITHUB_REPOSITORY o %(default)s)")
    ap.add_argument("--token-file", help="file con il token (alternativa a $GITHUB_TOKEN / gh)")
    ap.add_argument("--apply", action="store_true", help="esegue davvero (default: dry-run)")
    args = ap.parse_args()

    backlog_path = os.path.normpath(args.backlog)
    items = parse_backlog(backlog_path)
    desired = {it["id"]: it for it in items if it["status"].lower() != "done"}
    done_ids = {it["id"] for it in items if it["status"].lower() == "done"}
    items_by_id = {it["id"]: it for it in items}   # per l'estratto nel commento di chiusura

    ms_titles = B.milestones(items)
    print(f"backlog: {len(items)} item totali - {len(desired)} attivi (open) - {len(done_ids)} done")
    print(f"milestone dichiarate dagli item open: {len(ms_titles)}")
    for t in ms_titles:
        n = sum(1 for it in desired.values() if it["milestone"] == t)
        print(f"    - «{t}» ({n} item)")

    token = discover_token(args.token_file)
    backlog_url = f"https://github.com/{args.repo}/blob/{DEFAULT_BRANCH}/docs/backlog.md"

    if not token:
        if args.apply:
            sys.exit("ERRORE: --apply richiede un token ($GITHUB_TOKEN, --token-file o `gh auth login`).")
        print("\n! Nessun token -> dry-run OFFLINE (non confronto con le issue esistenti).")
        print("  Item che verrebbero garantiti OPEN:")
        for it in desired.values():
            ms = f"  milestone=«{it['milestone']}»" if it.get("milestone") else ""
            print(f"      - [{it['id']}] {it['title']}  labels={sorted(set(it['labels'] + [SYNC_LABEL]))}{ms}")
        return 0

    gh = GitHub(args.repo, token)

    existing = gh.list_sync_issues()
    by_fp = {}
    for iss in existing:
        m = FP_RE.search(iss.get("body") or "")
        if m:
            by_fp.setdefault(m.group(1), []).append(iss)

    to_create, to_update, to_reopen, to_close, to_dedup, skip = [], [], [], [], [], []
    for iid, it in desired.items():
        matches = by_fp.get(iid, [])
        # numero crescente = ordine di creazione: la prima creata è la canonica, così il
        # riferimento a una issue non cambia sotto i piedi di chi la sta leggendo.
        opened = sorted((i for i in matches if i["state"] == "open"), key=lambda i: i["number"])
        if opened:
            iss = opened[0]
            hm = HASH_RE.search(iss.get("body") or "")
            cur_hash = hm.group(1) if hm else None
            if cur_hash != item_hash(it):    # contenuto divergente → allinea
                to_update.append((it, iss))
            else:
                skip.append((it, iss))
            # Doppioni sullo stesso backlog-id: succede se un run legge la lista issue
            # prima che una CREATE appena fatta sia visibile. Li chiude, quindi il sync
            # si autoripara al run successivo invece di accumulare rumore.
            for dup in opened[1:]:
                to_dedup.append((iss, dup))
        elif matches:                        # esiste ma chiusa → riapri
            to_reopen.append((it, matches[0]))
        else:
            to_create.append(it)
    for iss in existing:
        m = FP_RE.search(iss.get("body") or "")
        fp = m.group(1) if m else None
        if iss["state"] == "open" and (fp is None or fp not in desired):
            to_close.append(iss)

    mode = "APPLY" if args.apply else "DRY-RUN"
    print(f"\n=== Piano sync ({mode}) su {args.repo} ===")
    print(f"  CREATE : {len(to_create)}")
    print(f"  UPDATE : {len(to_update)}")
    print(f"  REOPEN : {len(to_reopen)}"
          + ("   ! DRIFT: chiuse in UI ma ancora `status: open` nel backlog" if to_reopen else ""))
    print(f"  CLOSE  : {len(to_close)}")
    print(f"  DEDUP  : {len(to_dedup)}"
          + ("   ! doppioni sullo stesso backlog-id -> chiusi" if to_dedup else ""))
    print(f"  SKIP   : {len(skip)} (già allineate)\n")

    def ms_tag(it):
        return f"  milestone=«{it['milestone']}»" if it.get("milestone") else ""

    def milestone_for(it):
        """`- **milestone**:` → milestone number, creandola se manca. Solo in --apply."""
        return gh.ensure_milestone(it.get("milestone", ""))

    def labels_for(it, iss=None):
        """Label desiderate + `backlog-sync`, unite a quelle già sulla issue: la PATCH GitHub
        sostituisce il set, quindi senza l'unione perderemmo le label messe a mano."""
        want = set(it["labels"]) | {SYNC_LABEL}
        if iss:
            want |= {l["name"] for l in (iss.get("labels") or [])}
        return sorted(want)

    for it in to_create:
        own = f"  owner=@{it['owner']}" if it.get("owner") else ""
        print(f"  + CREATE [{it['id']}] {it['title']}{own}{ms_tag(it)}")
        if args.apply:
            labels = labels_for(it)
            gh.ensure_labels(labels)
            res = gh.create_issue(it["title"], build_body(it, backlog_url), labels,
                                  assignee=it.get("owner") or None, milestone=milestone_for(it))
            print(f"      -> #{res['number']} {res['html_url']}")

    for it, iss in to_update:
        labels = labels_for(it, iss)
        added = [l for l in labels if l not in {x["name"] for x in (iss.get("labels") or [])}]
        own = f"  owner=@{it['owner']}" if it.get("owner") else ""
        print(f"  ~ UPDATE #{iss['number']} [{it['id']}] {it['title']}"
              + (f"  +labels={added}" if added else "") + own + ms_tag(it))
        if args.apply:
            gh.ensure_labels(labels)
            gh.update_issue(iss["number"], it["title"], build_body(it, backlog_url), labels,
                            assignee=it.get("owner") or None, milestone=milestone_for(it))

    for it, iss in to_reopen:
        print(f"  ^ REOPEN #{iss['number']} [{it['id']}] {it['title']}"
              "   ! DRIFT: era chiusa in UI ma `status: open` nel backlog -> riaperta (il backlog comanda)")
        if args.apply:
            labels = labels_for(it, iss)
            gh.ensure_labels(labels)
            gh.set_state(iss["number"], "open")
            gh.update_issue(iss["number"], it["title"], build_body(it, backlog_url), labels,
                            assignee=it.get("owner") or None, milestone=milestone_for(it))
            gh.comment(iss["number"],
                       f"Riaperta: l'item `{it['id']}` è di nuovo attivo in `docs/backlog.md`. "
                       "Se volevi chiuderla davvero, metti `status: done` sull'item "
                       "(non chiuderla solo in UI).")

    for iss in to_close:
        m = FP_RE.search(iss.get("body") or "")
        fp = m.group(1) if m else "?"
        note = close_note(items_by_id.get(fp))
        print(f"  - CLOSE  #{iss['number']} (backlog-id: {fp}) {iss['title']}")
        if note:
            print(f"      estratto: {note if len(note) <= 120 else note[:120] + '…'}")
        if args.apply:
            body = ("Chiusa automaticamente: l'item non è più attivo "
                    "(rimosso o `status: done`) in `docs/backlog.md`.")
            if note:
                body += f"\n\n**Stato/motivo dal backlog:**\n\n> {note}"
            gh.comment(iss["number"], body)
            gh.set_state(iss["number"], "closed", reason="completed")

    for keep, dup in to_dedup:
        print(f"  ! DEDUP  #{dup['number']} doppione di #{keep['number']} -> CLOSE")
        if args.apply:
            gh.comment(dup["number"],
                       f"Chiusa come **doppione** di #{keep['number']}: stesso `backlog-id` in "
                       "`docs/backlog.md`. Nasce da una lista issue letta prima che la issue "
                       "originale fosse visibile lato API; la issue di riferimento è "
                       f"#{keep['number']}.")
            gh.set_state(dup["number"], "closed", reason="not_planned")

    for it, iss in skip:
        print(f"  = SKIP   #{iss['number']} [{it['id']}]")

    if not args.apply:
        print("\n(DRY-RUN: nessuna modifica. Rilancia con --apply per applicare.)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
