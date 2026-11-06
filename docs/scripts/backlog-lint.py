#!/usr/bin/env python3
"""
backlog-lint.py — validates docs/backlog.md WITHOUT touching GitHub (standalone lint for CI/PRs).

Immediate feedback on PRs and pushes: validation (unique ids, valid status/priority, known meta
keys, consistent milestone titles) does not wait for a sync run. The rules live in
`lib/backlog.py` (single source, also used by sync-backlog-to-issues.py and generate-roadmap.py).

Exit code:
  0 → no errors (warnings are printed but do not block; with --strict warnings fail)
  1 → at least one error (or warning with --strict)
  2 → usage error (missing file, etc.)

Requires: stdlib only.
"""
import argparse
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, "lib"))
import backlog as B   # noqa: E402


def main():
    ap = argparse.ArgumentParser(description="Lint docs/backlog.md (standalone, no GitHub).")
    ap.add_argument("--backlog", default=os.path.join(HERE, "..", "backlog.md"))
    ap.add_argument("--strict", action="store_true", help="treat warnings as errors (exit 1)")
    args = ap.parse_args()

    path = os.path.normpath(args.backlog)
    if not os.path.isfile(path):
        print(f"ERROR: backlog not found: {path}", file=sys.stderr)
        return 2

    items = B.parse_backlog(path)
    errors, warnings = B.lint(items)

    n_open = sum(1 for it in items if it["status"].lower() != "done")
    ms = B.milestones(items, include_done=True)
    print(f"backlog-lint: {len(items)} items ({n_open} open) - {len(ms)} distinct milestones")
    for title in ms:
        n = sum(1 for it in items if it["milestone"] == title and it["status"].lower() != "done")
        print(f"    - {title} ({n} open)")

    for w in warnings:
        print(f"  ! {w}")
    for e in errors:
        print(f"  x {e}")

    fail = bool(errors) or (args.strict and bool(warnings))
    if fail:
        n = len(errors) + (len(warnings) if args.strict else 0)
        print(f"\nbacklog-lint FAILED: {n} blocking problem(s).")
        return 1
    print("\nbacklog-lint OK" + (f" ({len(warnings)} non-blocking warnings)" if warnings else ""))
    return 0


if __name__ == "__main__":
    sys.exit(main())
