#!/usr/bin/env bash
# Exit 0 if PR <number> may be merged by an agent:
#   - toelke's newest verdict is an LGTM (an approving review, or a review or comment whose whole
#     body is "LGTM"), and no later review requests changes. Agents post as toelke too, but never
#     a bare "LGTM", so the exact match tells the human apart;
#   - the PR has not changed for real since then: rebases are fine, so the diff against its
#     merge-base must be identical (same `git patch-id`) to the diff that was LGTM'd;
#   - GitHub reports it mergeable, and every check has passed.
# Prints the reason when it exits 1.
set -euo pipefail

pr="${1:?usage: lgtm-check.sh <pr-number>}"
reviewer="toelke"
repo="$(gh repo view --json nameWithOwner --jq .nameWithOwner)"
fail() { echo "not mergeable: $*"; exit 1; }

info="$(gh pr view "$pr" --json state,mergeable,headRefOid,baseRefName,statusCheckRollup,reviews,comments,commits)"
[[ "$(jq -r .state <<<"$info")" == OPEN ]] || fail "PR is not open"
[[ "$(jq -r .mergeable <<<"$info")" == MERGEABLE ]] || fail "GitHub does not report it mergeable"
jq -e '.statusCheckRollup | length > 0' <<<"$info" >/dev/null || fail "no checks have run"
bad="$(jq -r '.statusCheckRollup[] | select((.conclusion // .state // "") | test("^(SUCCESS|SKIPPED|NEUTRAL)$") | not) | .name // .context' <<<"$info")"
[[ -z "$bad" ]] || fail "checks not green: $(tr '\n' ' ' <<<"$bad")"

# toelke's verdicts, oldest first: {at, lgtm, sha}. A review pins the commit it was made on.
verdicts="$(jq --arg r "$reviewer" --arg lgtm '^\s*lgtm[.!]*\s*$' '
  [ (.reviews[] | select(.author.login == $r)
      | select(.state == "APPROVED" or .state == "CHANGES_REQUESTED" or (.body | test($lgtm; "i")))
      | {at: .submittedAt, lgtm: (.state != "CHANGES_REQUESTED"), sha: .commit.oid}),
    (.comments[] | select(.author.login == $r) | select(.body | test($lgtm; "i"))
      | {at: .createdAt, lgtm: true, sha: null})
  ] | sort_by(.at)' <<<"$info")"
last="$(jq '.[-1] // empty' <<<"$verdicts")"
[[ -n "$last" ]] || fail "no LGTM from $reviewer"
[[ "$(jq -r .lgtm <<<"$last")" == true ]] || fail "$reviewer's latest review requests changes"
at="$(jq -r .at <<<"$last")"
reviewed="$(jq -r '.sha // empty' <<<"$last")"

if [[ -z "$reviewed" ]]; then
  # A plain comment: the reviewed head is the one replaced by the first force-push after it,
  # or else the newest commit that existed when the comment was written.
  reviewed="$(gh api --paginate "repos/$repo/issues/$pr/timeline?per_page=100" \
    --jq ".[] | select(.event == \"head_ref_force_pushed\" and .created_at > \"$at\") | .commit_id" | head -1)"
  if [[ -z "$reviewed" ]]; then
    reviewed="$(jq -r --arg at "$at" '[.commits[] | select(.committedDate <= $at)] | .[-1].oid // empty' <<<"$info")"
  fi
fi
[[ -n "$reviewed" ]] || fail "cannot tell which commit the LGTM refers to"

base_branch="$(jq -r .baseRefName <<<"$info")"
base="origin/$base_branch"
head="$(jq -r .headRefOid <<<"$info")"
git fetch -q origin "$base_branch" "$reviewed" "$head"
pid() { git diff "$(git merge-base "$base" "$1")" "$1" | git patch-id --stable | cut -d' ' -f1; }
[[ "$(pid "$reviewed")" == "$(pid "$head")" ]] \
  || fail "changed since $reviewer's LGTM on ${reviewed:0:7} ($at); ask for a new LGTM"

echo "mergeable: LGTM by $reviewer on ${reviewed:0:7}, no real change since, checks green"
