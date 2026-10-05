#!/usr/bin/env python3
"""Ticket overview, and the next ticket to take (AGENTS.md, "Taking the next ticket").

    scripts/tickets.py          table of all tickets: state, dependencies, PR or worktree
    scripts/tickets.py --open   the same, without done tickets
    scripts/tickets.py --next   only the next ticket: "<NNN> new" or "<NNN> resume <path>";
                                exit 1 if no ticket is ready

Ticket status comes from origin/main. Work in progress comes from open PRs, local worktrees and
their `.owner` PID, and `ticket/*` branches on origin.

States, in the order AGENTS.md cares about them:
  done       status: done on main
  review     an open PR exists
  working    a worktree whose .owner process is alive
  abandoned  a worktree with no live owner, or a ticket branch with no worktree and no PR
  ready      todo, every dependency done, nothing in progress
  blocked    todo, waiting for the listed dependencies
"""

import json
import os
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(
    subprocess.run(
        ["git", "-C", str(Path(__file__).parent), "rev-parse", "--show-toplevel"],
        capture_output=True, text=True, check=True,
    ).stdout.strip()
)


def run(*args: str) -> str:
    return subprocess.run(args, cwd=ROOT, capture_output=True, text=True, check=True).stdout


def tickets_on_main() -> dict[str, dict]:
    tickets = {}
    for path in run("git", "ls-tree", "--name-only", "origin/main", "tickets/").split():
        m = re.match(r"tickets/(\d{3}[a-z]?)-.*\.md$", path)
        if not m:
            continue
        text = run("git", "show", f"origin/main:{path}")
        front = text.split("---")[1] if text.startswith("---") else ""
        status = re.search(r"^status:\s*(\S+)", front, re.M)
        deps = re.search(r"^depends_on:\s*\[(.*)\]", front, re.M)
        title = re.search(r"^# (.+)$", text, re.M)
        tickets[m[1]] = {
            "status": status[1] if status else "?",
            "deps": [d.strip() for d in deps[1].split(",") if d.strip()] if deps else [],
            "title": title[1] if title else path,
        }
    return tickets


def ticket_of(branch: str) -> str | None:
    m = re.match(r"(?:refs/heads/)?ticket/(\d{3}[a-z]?)-", branch)
    return m[1] if m else None


def open_prs() -> dict[str, int]:
    prs = json.loads(run("gh", "pr", "list", "--state", "open", "--json", "number,headRefName"))
    return {t: p["number"] for p in prs if (t := ticket_of(p["headRefName"]))}


def worktrees() -> dict[str, dict]:
    result = {}
    for block in run("git", "worktree", "list", "--porcelain").strip().split("\n\n"):
        fields = dict(line.split(" ", 1) for line in block.splitlines() if " " in line)
        t = ticket_of(fields.get("branch", ""))
        if not t:
            continue
        path = Path(fields["worktree"])
        owner = None
        try:
            pid = int((path / ".owner").read_text().strip())
            os.kill(pid, 0)
            owner = pid
        except (OSError, ValueError):
            pass
        result[t] = {"path": os.path.relpath(path, ROOT), "owner": owner}
    return result


def remote_branches() -> set[str]:
    out = run("git", "for-each-ref", "--format=%(refname:short)", "refs/remotes/origin/ticket/")
    return {t for b in out.split() if (t := ticket_of(b.removeprefix("origin/")))}


def main() -> int:
    args = set(sys.argv[1:])
    run("git", "fetch", "-q", "--prune", "origin")
    tickets = tickets_on_main()
    prs, trees, branches = open_prs(), worktrees(), remote_branches()

    for num, t in tickets.items():
        missing = [d for d in t["deps"] if tickets.get(d, {}).get("status") != "done"]
        tree = trees.get(num)
        if t["status"] == "done":
            state, note = "done", ""
        elif num in prs:
            state, note = "review", f"PR #{prs[num]}"
        elif tree and tree["owner"]:
            state, note = "working", f"{tree['path']} (pid {tree['owner']})"
        elif tree:
            state, note = "abandoned", f"{tree['path']}, no live owner"
        elif num in branches:
            state, note = "abandoned", f"branch origin/ticket/{num}-*, no PR"
        elif missing:
            state, note = "blocked", "waits for " + ", ".join(missing)
        else:
            state, note = "ready", ""
        t.update(state=state, note=note, missing=missing)

    # AGENTS.md: resume abandoned work first, else the lowest-numbered ready ticket.
    nxt = next((n for n, t in tickets.items() if t["state"] == "abandoned"), None)
    nxt = nxt or next((n for n, t in tickets.items() if t["state"] == "ready"), None)

    if "--next" in args:
        if not nxt:
            return 1
        t = tickets[nxt]
        print(f"{nxt} resume {trees[nxt]['path']}" if t["state"] == "abandoned" and nxt in trees
              else f"{nxt} resume origin/{nxt}" if t["state"] == "abandoned"
              else f"{nxt} new")
        return 0

    def dep_list(t: dict) -> str:
        return " ".join(d + ("" if d not in t["missing"] else "*") for d in t["deps"])

    rows = [(n, t) for n, t in tickets.items() if not ("--open" in args and t["state"] == "done")]
    width = min(max(len(t["title"]) for _, t in rows), 46)
    print(f"{'#':<4} {'TITLE':<{width}}  {'STATE':<9}  {'DEPENDS ON':<14}  NOTE")
    for n, t in rows:
        title = t["title"] if len(t["title"]) <= width else t["title"][: width - 1] + "…"
        print(f"{n:<4} {title:<{width}}  {t['state']:<9}  {dep_list(t):<14}  {t['note']}")
    counts = {}
    for t in tickets.values():
        counts[t["state"]] = counts.get(t["state"], 0) + 1
    print("\n" + ", ".join(f"{v} {k}" for k, v in counts.items()) + "   (* = dependency not done)")
    print(f"next: {nxt} ({tickets[nxt]['state']})" if nxt else "next: none ready")
    return 0


if __name__ == "__main__":
    sys.exit(main())
