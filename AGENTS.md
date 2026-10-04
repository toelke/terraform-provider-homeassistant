# AGENTS.md

OpenTofu provider for Home Assistant (Go, terraform-plugin-framework).

- `GLOSSARY.md`: the vocabulary. Code and docs use these terms.
- `adr/`: decisions and why they were made. Accepted ADRs are immutable. Decisions belong to the
  human: an agent writes a new ADR, or one that supersedes another, only after putting the
  decision and its options to the human and getting an answer.
- `spec/`: what the provider does. Start with `spec/overview.md`.
- `tickets/`: the work, as vertical slices. `tickets/README.md` holds the definition of done.

## Tending open PRs

Do this before taking a new ticket. For each open PR (`gh pr list`), oldest first:

1. **Claim:** `git worktree add ../tofu-ha-NNN ticket/NNN-<slug>`. If the branch is already
   checked out in another worktree, another session owns it; skip the PR.
2. **Rebase** onto `origin/main` and resolve every conflict. Re-run the definition of done,
   then `git push --force-with-lease`.
3. **Comments:** address every unresolved review comment and thread. Fix it, or reply with
   the reason you didn't. A comment that asks for a decision follows the ADR rule above. If it
   needs the human, reply on the PR with options and a recommendation, and leave it open.
4. **CI:** `gh pr checks --watch` until every check is green. Fix failures and push again.
5. **Clean up:** `git worktree remove ../tofu-ha-NNN`.

## Taking the next ticket

1. **Pick:** choose the lowest-numbered ticket with `status: todo` whose `depends_on` tickets are
   all `done`, and that has no open PR or branch named `ticket/NNN-*`
   (`gh pr list --search "head:ticket/NNN"`).
2. **Isolate:** work in a private worktree, branched from up-to-date `origin/main`:
   `git fetch && git worktree add ../tofu-ha-NNN -b ticket/NNN-<slug> origin/main`.
3. **Read:** the ticket, plus every ADR and spec file in its frontmatter.
4. **Build:** keep going until every acceptance criterion and the definition of done in
   `tickets/README.md` hold. If the implementation deviates from the spec, update the spec in the
   same branch. If the ticket needs a decision that no ADR covers, or contradicts an ADR, stop
   and ask the human (options plus your recommendation). Record the answer as an ADR.
5. **Ship:** set the ticket to `status: done`, then commit, push, and run `gh pr create`. The PR
   body links the ticket and lists any spec changes.
6. **Clean up:** `git worktree remove ../tofu-ha-NNN`.

Locally, build and test for the host platform only, e.g.
`goreleaser build --snapshot --clean --single-target`. Cross-compiling is CI's job; a local
build of every target overloads the machine.

Agents may create branches, commit, push, and open PRs in this repo without asking. Merging is
left to the human.
