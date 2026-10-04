# AGENTS.md

OpenTofu provider for Home Assistant (Go, terraform-plugin-framework).

- `GLOSSARY.md`: the vocabulary. Code and docs use these terms.
- `adr/`: decisions and why they were made. Accepted ADRs are immutable. Decisions belong to the
  human: an agent writes a new ADR, or one that supersedes another, only after putting the
  decision and its options to the human and getting an answer.
- `spec/`: what the provider does. Start with `spec/overview.md`.
- `tickets/`: the work, as vertical slices. `tickets/README.md` holds the definition of done.

## Worktrees and ownership

Every PR or ticket is worked on in its own worktree, `../tofu-ha-NNN`, on branch
`ticket/NNN-<slug>`. A worktree is **owned** while the PID in its `.owner` file is alive.

- **Claim:** create or reuse the worktree, then immediately run `echo $PPID > ../tofu-ha-NNN/.owner`
  (in the tool shell, `$PPID` is your own Claude process).
- **Owned by someone else** (`kill -0 $(cat ../tofu-ha-NNN/.owner)` succeeds): skip it.
- **Abandoned** (no `.owner`, or a dead PID): take it over and continue from its current state,
  including uncommitted changes. A `ticket/*` branch with no worktree and no PR is abandoned too.
- **Release:** `git worktree remove ../tofu-ha-NNN` when done.

Run every command in the foreground and wait for it. A session ends as soon as it replies without
a tool call, and its background tasks die with it.

## Tending open PRs

Do this before taking a new ticket. For each open PR (`gh pr list`), oldest first:

1. **Claim** its worktree (see above), or skip the PR if someone else owns it.
2. **Rebase** onto `origin/main` and resolve every conflict. Re-run the definition of done,
   then `git push --force-with-lease`.
3. **Comments:** address every unresolved review comment and thread. Fix it, or reply with
   the reason you didn't. A comment that asks for a decision follows the ADR rule above. If it
   needs the human, reply on the PR with options and a recommendation, and leave it open.
4. **CI:** first check `gh pr view <pr> --json mergeable`. If it is `CONFLICTING`, `main` has
   moved: go back to step 2, because GitHub runs no checks on a conflicting PR. Otherwise
   `gh pr checks --watch` until every check is green. Fix failures and push again.
5. **Release** the worktree.
6. **Merge** when `scripts/lgtm-check.sh <pr>` passes: `gh pr merge <pr> --merge`. GitHub deletes
   the branch itself and retargets PRs stacked on it, so don't delete it yourself.
   The check passes once toelke has commented a bare "LGTM", nothing but rebases has changed
   since (resolving `CHANGELOG.md` conflicts counts as a rebase), and all checks are green.
   Agents post as toelke too, so your own comments always carry the Claude Code footer and are
   never just "LGTM".

## Taking the next ticket

1. **Pick:** first resume any abandoned ticket worktree or branch. Otherwise choose the
   lowest-numbered ticket with `status: todo` whose `depends_on` tickets are all `done`, and
   that has no open PR or branch named `ticket/NNN-*` (`gh pr list --search "head:ticket/NNN"`).
2. **Claim:** for a new ticket, `git fetch && git worktree add ../tofu-ha-NNN -b
   ticket/NNN-<slug> origin/main`, then write `.owner`.
3. **Read:** the ticket, plus every ADR and spec file in its frontmatter.
4. **Build:** keep going until every acceptance criterion and the definition of done in
   `tickets/README.md` hold. If the implementation deviates from the spec, update the spec in the
   same branch. If the ticket needs a decision that no ADR covers, or contradicts an ADR, stop
   and ask the human (options plus your recommendation). Record the answer as an ADR.
5. **Ship:** set the ticket to `status: done`, then commit, push, and run `gh pr create`. The PR
   body links the ticket and lists any spec changes.
6. **Release** the worktree.

Locally, build and test for the host platform only, e.g.
`goreleaser build --snapshot --clean --single-target`. Cross-compiling is CI's job; a local
build of every target overloads the machine.

Agents may create branches, commit, push, and open PRs in this repo without asking. They merge
only through step 6 of "Tending open PRs".
