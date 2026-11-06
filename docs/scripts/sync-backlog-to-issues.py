#!/usr/bin/env python3
"""
sync-backlog-to-issues.py — syncs docs/backlog.md → GitHub issues (IDEMPOTENT).

SINGLE source of truth: docs/backlog.md. Every item has a stable `id`. For every item with
status != done the script guarantees an open issue; removed items or `status: done` close the
matching issue. Re-runs create no duplicates thanks to the fingerprint hidden in the issue body:
    <!-- backlog-id: <id> | hash: <sha> -->
plus the `backlog-sync` label on every managed issue.

Matrix (idempotent):
  open item  + no issue                 → CREATE
  open item  + open issue, hash ==      → SKIP (already aligned)
  open item  + open issue, hash !=      → UPDATE (title/labels/body differ)
  open item  + closed issue             → REOPEN (+ realign the content)
  done/removed item + open issue        → CLOSE (automatic comment + excerpt of the backlog body)

Change detection: the fingerprint includes a hash of the content; when the backlog hash differs
from the one embedded in the issue → UPDATE.

DYNAMIC MILESTONES: an item can carry `- **milestone**: <title>`. There is no hardcoded list of
milestones: the script looks the repo's milestone up by exact title and **CREATES it when
missing** (idempotent), then assigns the issue. The title is part of the hash → changing or
removing it realigns the issue on the next sync. It never removes a milestone assigned by hand
when the item declares none.

Difference from the GitLab twin: the GitHub API has no `add_labels`, the PATCH **replaces** the
whole label set. To keep the labels added by hand in the UI, the script always sends the UNION of
the labels already on the issue and the desired ones.

Safety: it ONLY touches issues carrying the `backlog-sync` label. Third-party issues
(Renovate/Dependabot, user reports) are never read or closed. Pull requests, which the issues
endpoint returns together with issues, are filtered out via the `pull_request` field.

Auth: a token with the `repo` scope (`issues: write`; on a public repo `public_repo` is enough).
  - $GITHUB_TOKEN (in CI: `${{ secrets.GITHUB_TOKEN }}`)
  - or `gh auth token` locally (detected automatically)
  - or --token-file <path>

Default: --dry-run (prints the plan, does NOT write). Use --apply to actually run it.

Requires: stdlib only (urllib, json, re, argparse) + optional `gh` for the local token.
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
import backlog as B   # noqa: E402  — parsing/rules shared with backlog-lint and generate-roadmap

DEFAULT_REPO = "Allan-Nava/Go-DDD-Scaffold"
DEFAULT_BRANCH = "main"
SYNC_LABEL = "backlog-sync"
API = "https://api.github.com"

FP_RE = B.FP_RE
HASH_RE = B.HASH_RE
item_hash = B.item_hash


# ----------------------------------------------------------------------------- parsing
def parse_backlog(path):
    """Parse the backlog (through lib/backlog.py) and refuse duplicate ids (fatal for the sync)."""
    items = B.parse_backlog(path)
    counts = {}
    for it in items:
        counts[it["id"]] = counts.get(it["id"], 0) + 1
    dups = sorted(k for k, n in counts.items() if n > 1)
    if dups:
        sys.exit(f"ERROR: duplicate ids in backlog.md: {dups}")
    return items


def build_body(item, backlog_url):
    parts = []
    body = "\n".join(item["body"]).strip()
    if body:
        parts.append(body)
    meta = []
    if item["priority"]:
        meta.append(f"**Priority**: {item['priority']}")
    if item.get("milestone"):
        meta.append(f"**Milestone**: {item['milestone']}")
    if item.get("owner"):
        meta.append(f"**Owner**: @{item['owner']}")
    if item["ref"]:
        meta.append(f"**Ref**: {item['ref']}")
    if meta:
        parts.append("\n".join(meta))
    parts.append(f"_Issue generated automatically from [`docs/backlog.md`]({backlog_url}) "
                 f"(id: `{item['id']}`). Do not edit the title by hand: change the backlog._")
    parts.append(f"<!-- backlog-id: {item['id']} | hash: {item_hash(item)} -->")
    return "\n\n".join(parts)


CHECKLIST_RE = re.compile(r"^[-*]\s*\[[ xX]\]")   # markdown checklist entry: - [ ] / - [x]


def close_note(item, max_len=700):
    """Excerpt of the item body attached to the closing comment, to give context/reason.
    Prefers the first *blockquote* block (`> …`, by convention the resolution note), otherwise
    the first block of prose (skipping checklists). None if the item was removed from the
    backlog (no body available) or the body is empty."""
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
        chosen = next((b for b in blocks if not all(CHECKLIST_RE.match(l) for l in b)), None)  # 2) prose
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
        self._milestone_cache = {}   # title → number (for `- **milestone**:`)
        self._labels = None          # set of the label names existing on the repo

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
            # GitHub's issue list can serve a response a few seconds old: right after a
            # CREATE a re-run would see an incomplete state and create duplicates. This
            # narrows the window; the real safety net is the dedup by backlog-id in the
            # plan (see main()).
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
                sys.exit("ERROR 404 on issues: is the Issue Tracker disabled on the repo? "
                         "Enable it (Settings ▸ General ▸ Features ▸ Issues) and try again.")
            if e.code == 403 and "rate limit" in detail.lower():
                sys.exit(f"ERROR: GitHub rate limit exhausted — {detail}")
            sys.exit(f"ERROR API {method} {path}: HTTP {e.code} — {detail}")

    # --- dynamic milestones ----------------------------------------------------
    def ensure_milestone(self, title):
        """Milestone title → number. Looks it up on the repo (exact match, closed ones too);
        CREATES it when it does not exist. Cached so GETs/POSTs are not repeated. None if
        `title` is empty."""
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
            print(f"      + GitHub milestone missing -> CREATE «{title}»")
            res = self._req("POST", f"/repos/{self.repo}/milestones",
                            data={"title": title,
                                  "description": "Milestone managed from docs/backlog.md "
                                                 "(the items' `- **milestone**:` field)."})
            num = res["number"]
        self._milestone_cache[title] = num
        return num

    # --- labels ---------------------------------------------------------------
    def ensure_labels(self, names):
        """Create the labels missing on the repo. GitHub would create them implicitly on the
        issue POST, but not on the PATCH: creating them explicitly makes UPDATE and CREATE
        symmetric."""
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
                print(f"      + label missing -> CREATE «{name}»")
                self._req("POST", f"/repos/{self.repo}/labels",
                          data={"name": name, "color": "ededed",
                                "description": "Managed from docs/backlog.md"})
                self._labels.add(name)

    def list_sync_issues(self):
        """Only the issues labelled `backlog-sync` (the others are none of this script's business).
        The issues endpoint also returns PRs: filtered out via the `pull_request` field."""
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
        # labels is already the UNION with the existing ones: the PATCH replaces the set.
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
    """$GITHUB_TOKEN → --token-file → `gh auth token`. Empty when none is available."""
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
    ap = argparse.ArgumentParser(description="Sync docs/backlog.md → GitHub issues (idempotent).")
    ap.add_argument("--backlog", default=os.path.join(HERE, "..", "backlog.md"))
    ap.add_argument("--repo", default=os.environ.get("GITHUB_REPOSITORY") or DEFAULT_REPO,
                    help="owner/repo (default: $GITHUB_REPOSITORY or %(default)s)")
    ap.add_argument("--token-file", help="file holding the token (alternative to $GITHUB_TOKEN / gh)")
    ap.add_argument("--apply", action="store_true", help="actually run it (default: dry-run)")
    args = ap.parse_args()

    backlog_path = os.path.normpath(args.backlog)
    items = parse_backlog(backlog_path)
    desired = {it["id"]: it for it in items if it["status"].lower() != "done"}
    done_ids = {it["id"] for it in items if it["status"].lower() == "done"}
    items_by_id = {it["id"]: it for it in items}   # for the excerpt in the closing comment

    ms_titles = B.milestones(items)
    print(f"backlog: {len(items)} items in total - {len(desired)} active (open) - {len(done_ids)} done")
    print(f"milestones declared by open items: {len(ms_titles)}")
    for t in ms_titles:
        n = sum(1 for it in desired.values() if it["milestone"] == t)
        print(f"    - «{t}» ({n} items)")

    token = discover_token(args.token_file)
    backlog_url = f"https://github.com/{args.repo}/blob/{DEFAULT_BRANCH}/docs/backlog.md"

    if not token:
        if args.apply:
            sys.exit("ERROR: --apply needs a token ($GITHUB_TOKEN, --token-file or `gh auth login`).")
        print("\n! No token -> OFFLINE dry-run (no comparison with the existing issues).")
        print("  Items that would be guaranteed OPEN:")
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
        # increasing number = creation order: the first one created is canonical, so the
        # reference to an issue does not change under the feet of whoever is reading it.
        opened = sorted((i for i in matches if i["state"] == "open"), key=lambda i: i["number"])
        if opened:
            iss = opened[0]
            hm = HASH_RE.search(iss.get("body") or "")
            cur_hash = hm.group(1) if hm else None
            if cur_hash != item_hash(it):    # content differs → realign
                to_update.append((it, iss))
            else:
                skip.append((it, iss))
            # Duplicates on the same backlog-id: it happens when a run reads the issue list
            # before a CREATE that was just made is visible. They get closed, so the sync
            # repairs itself on the next run instead of piling up noise.
            for dup in opened[1:]:
                to_dedup.append((iss, dup))
        elif matches:                        # exists but closed → reopen
            to_reopen.append((it, matches[0]))
        else:
            to_create.append(it)
    for iss in existing:
        m = FP_RE.search(iss.get("body") or "")
        fp = m.group(1) if m else None
        if iss["state"] == "open" and (fp is None or fp not in desired):
            to_close.append(iss)

    mode = "APPLY" if args.apply else "DRY-RUN"
    print(f"\n=== Sync plan ({mode}) on {args.repo} ===")
    print(f"  CREATE : {len(to_create)}")
    print(f"  UPDATE : {len(to_update)}")
    print(f"  REOPEN : {len(to_reopen)}"
          + ("   ! DRIFT: closed in the UI but still `status: open` in the backlog" if to_reopen else ""))
    print(f"  CLOSE  : {len(to_close)}")
    print(f"  DEDUP  : {len(to_dedup)}"
          + ("   ! duplicates on the same backlog-id -> closed" if to_dedup else ""))
    print(f"  SKIP   : {len(skip)} (already aligned)\n")

    def ms_tag(it):
        return f"  milestone=«{it['milestone']}»" if it.get("milestone") else ""

    def milestone_for(it):
        """`- **milestone**:` → milestone number, creating it when missing. Only with --apply."""
        return gh.ensure_milestone(it.get("milestone", ""))

    def labels_for(it, iss=None):
        """Desired labels + `backlog-sync`, joined with the ones already on the issue: the GitHub
        PATCH replaces the set, so without the union we would lose the labels added by hand."""
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
              "   ! DRIFT: closed in the UI but `status: open` in the backlog -> reopened (the backlog rules)")
        if args.apply:
            labels = labels_for(it, iss)
            gh.ensure_labels(labels)
            gh.set_state(iss["number"], "open")
            gh.update_issue(iss["number"], it["title"], build_body(it, backlog_url), labels,
                            assignee=it.get("owner") or None, milestone=milestone_for(it))
            gh.comment(iss["number"],
                       f"Reopened: item `{it['id']}` is active again in `docs/backlog.md`. "
                       "If you really meant to close it, set `status: done` on the item "
                       "(do not close it only in the UI).")

    for iss in to_close:
        m = FP_RE.search(iss.get("body") or "")
        fp = m.group(1) if m else "?"
        note = close_note(items_by_id.get(fp))
        print(f"  - CLOSE  #{iss['number']} (backlog-id: {fp}) {iss['title']}")
        if note:
            print(f"      excerpt: {note if len(note) <= 120 else note[:120] + '…'}")
        if args.apply:
            body = ("Closed automatically: the item is no longer active "
                    "(removed or `status: done`) in `docs/backlog.md`.")
            if note:
                body += f"\n\n**Status/reason from the backlog:**\n\n> {note}"
            gh.comment(iss["number"], body)
            gh.set_state(iss["number"], "closed", reason="completed")

    for keep, dup in to_dedup:
        print(f"  ! DEDUP  #{dup['number']} duplicate of #{keep['number']} -> CLOSE")
        if args.apply:
            gh.comment(dup["number"],
                       f"Closed as a **duplicate** of #{keep['number']}: same `backlog-id` in "
                       "`docs/backlog.md`. It comes from an issue list read before the original "
                       "issue was visible through the API; the reference issue is "
                       f"#{keep['number']}.")
            gh.set_state(dup["number"], "closed", reason="not_planned")

    for it, iss in skip:
        print(f"  = SKIP   #{iss['number']} [{it['id']}]")

    if not args.apply:
        print("\n(DRY-RUN: no changes. Run again with --apply to apply them.)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
