#!/usr/bin/env python3
"""
backlog.py — parsing and validation rules shared by everything that reads docs/backlog.md.

SINGLE SOURCE of the backlog's structural rules: imported by `sync-backlog-to-issues.py`
(which opens/updates the GitHub issues), by `backlog-lint.py` (which validates in CI) and by
`generate-roadmap.py` (which generates the milestone page). No regex or convention is
duplicated across the scripts.

A backlog item is:

    ### `stable-id` — Title
    - **status**: open|done          (default: open)
    - **priority**: low|medium|high  (optional)
    - **labels**: a, b, c            (optional)
    - **milestone**: <title>         (optional — created on GitHub when missing)
    - **owner**: <GitHub login>      (optional)
    - **ref**: <link>                (optional)

    prose … (becomes the issue body)

`parse_backlog()` is PURE (no sys.exit): it returns the list of items. Besides the fields the
sync uses, each item carries two technical keys for the linter:
  - `_line`      : line number (1-based) of the `### ...` heading
  - `_meta_seen` : list of {key, line, value} for the `- **k**: v` bullets found in the
                   *meta block* (the bullets right under the heading, before the first line
                   of prose) — used to catch unknown keys (typos).

This module is the twin of `docs/scripts/lib/backlog.py` in devops_hiway: same rules, same item
format. If you change it here, consider aligning it there too.
"""
import re

# --- structural regexes (single source) -----------------------------------------------
# item: "### `id` — Title"  (id in backticks; optional — or - separator)
ITEM_RE = re.compile(r"^###\s+`([^`]+)`\s*[—\-]*\s*(.*)$")
META_RE = re.compile(r"^[-*]\s+\*\*(\w+)\*\*:\s*(.+)$")
FP_RE = re.compile(r"<!--\s*backlog-id:\s*([a-zA-Z0-9._-]+)")        # id (compat: with or without | hash:)
HASH_RE = re.compile(r"backlog-id:\s*[\w.-]+\s*\|\s*hash:\s*([0-9a-f]+)")

# --- allowed vocabulary ----------------------------------------------------------------
KNOWN_META = {"status", "priority", "labels", "milestone", "owner", "ref"}
VALID_STATUS = {"open", "done"}
VALID_PRIORITY = {"low", "medium", "high"}
ID_RE = re.compile(r"^[a-z0-9][a-z0-9._-]*$")   # lowercase kebab-case


def _new_item(item_id, title, line):
    return {"id": item_id, "title": title,
            "labels": [], "status": "open", "priority": "", "ref": "", "owner": "",
            "milestone": "", "body": [], "_line": line, "_meta_seen": []}


def parse_backlog(path):
    """Parse docs/backlog.md → list of items (PURE, no sys.exit).

    A `- **k**: v` bullet with a known `k` sets the field; any other line (blank, prose,
    checklist, bullet with an unknown key) goes into the body. It also records in `_meta_seen`
    the bullets met in the initial meta block (until prose starts), so the linter can report
    key typos.
    """
    items, cur, in_meta = [], None, False
    with open(path, encoding="utf-8") as f:
        lines = f.read().splitlines()
    for i, ln in enumerate(lines, start=1):
        m = ITEM_RE.match(ln)
        if m:
            if cur:
                items.append(cur)
            cur = _new_item(m.group(1).strip(), m.group(2).strip(), i)
            in_meta = True
            continue
        if cur is None:
            continue
        if re.match(r"^#{1,3}\s", ln):   # new (non-item) section → closes the current item
            items.append(cur)
            cur, in_meta = None, False
            continue
        mm = META_RE.match(ln)
        if mm:
            k, v = mm.group(1).lower(), mm.group(2).strip()
            if in_meta:
                cur["_meta_seen"].append({"key": mm.group(1), "line": i, "value": v})
            if k == "labels":
                cur["labels"] = [x.strip().strip("`") for x in v.split(",") if x.strip()]
            elif k in ("status", "priority", "ref", "owner", "milestone"):
                cur[k] = v.strip().lstrip("@")
            else:
                cur["body"].append(ln)   # unknown key → body
        else:
            if ln.strip():               # first non-blank line of prose → end of the meta block
                in_meta = False
            cur["body"].append(ln)
    if cur:
        items.append(cur)
    return items


def item_hash(item):
    """Stable hash of the synced content (title/labels/priority/ref/owner/milestone/body).
    Embedded in the issue fingerprint: when it changes, the issue needs updating."""
    import hashlib
    canon = "\x1f".join([
        item["title"],
        ",".join(sorted(item["labels"])),
        item.get("priority", ""),
        item.get("ref", ""),
        item.get("owner", ""),
        item.get("milestone", ""),
        "\n".join(item["body"]).strip(),
    ])
    return hashlib.sha1(canon.encode("utf-8")).hexdigest()[:12]


def _norm_milestone(title):
    """Normalise a milestone title for the 'same milestone' match (case + collapsed spaces)."""
    return re.sub(r"\s+", " ", title).strip().casefold()


def milestones(items, include_done=False):
    """Distinct milestone titles, sorted. Milestones are DYNAMIC: there is no hardcoded list,
    only the titles the items declare."""
    src = items if include_done else [it for it in items if it["status"].lower() != "done"]
    return sorted({it["milestone"] for it in src if it["milestone"]})


def lint(items):
    """Validate the parsed items. Returns (errors, warnings): two lists of formatted strings
    (with line numbers). Non-empty `errors` ⇒ the linter must exit with a non-zero status."""
    errors, warnings = [], []

    # 1) duplicate ids (fatal: the sync itself would refuse them)
    by_id = {}
    for it in items:
        by_id.setdefault(it["id"], []).append(it["_line"])
    for iid, ls in by_id.items():
        if len(ls) > 1:
            errors.append(f"duplicate id `{iid}` (lines {', '.join(map(str, ls))})")

    for it in items:
        line, iid = it["_line"], it["id"]
        loc = f"line {line} [{iid}]"

        # 2) id format
        if not ID_RE.match(iid):
            errors.append(f"{loc}: invalid id (expected lowercase kebab-case `[a-z0-9._-]`)")

        # 3) non-empty title
        if not it["title"].strip():
            errors.append(f"{loc}: empty title")

        # 4) status / priority in the vocabulary
        if it["status"] and it["status"].lower() not in VALID_STATUS:
            errors.append(f"{loc}: invalid status `{it['status']}` (expected {sorted(VALID_STATUS)})")
        if it["priority"] and it["priority"].lower() not in VALID_PRIORITY:
            errors.append(f"{loc}: invalid priority `{it['priority']}` (expected {sorted(VALID_PRIORITY)})")

        # 5) unknown meta keys in the initial block (typos such as `- **lables**:`)
        for meta in it["_meta_seen"]:
            if meta["key"].lower() not in KNOWN_META:
                errors.append(f"line {meta['line']} [{iid}]: unknown meta key "
                              f"`{meta['key']}` (known: {sorted(KNOWN_META)}) — typo?")

    # 6) consistent milestone titles: same characters everywhere (reports case/space variants)
    variants = {}
    for it in items:
        if it["milestone"]:
            variants.setdefault(_norm_milestone(it["milestone"]), set()).add(it["milestone"])
    for norm, raws in variants.items():
        if len(raws) > 1:
            warnings.append("milestone written in different ways (same milestone?): "
                            + " · ".join(f"«{r}»" for r in sorted(raws))
                            + " → make the title identical (the match is exact, character by character)")

    return errors, warnings
