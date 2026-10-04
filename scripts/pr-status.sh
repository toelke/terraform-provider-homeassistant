#!/usr/bin/env bash
# One line per open PR, oldest first, with what PR tending needs to know:
#   mergeable   GitHub's view (CONFLICTING means: rebase)
#   behind      commits on origin/main that the branch doesn't have
#   checks      ok / failing names / pending / none
#   threads     unresolved review threads
#   last        who wrote the newest comment: "human" (toelke without the Claude Code footer),
#               "agent", or "-"; a human comment newer than the agent's needs an answer
#   lgtm        result of scripts/lgtm-check.sh
set -euo pipefail

cd "$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
git fetch -q --prune origin
owner_repo="$(gh repo view --json nameWithOwner --jq .nameWithOwner)"
owner="${owner_repo%/*}" repo="${owner_repo#*/}"

printf '%-5s %-38s %-12s %-7s %-22s %-8s %-6s %s\n' PR BRANCH MERGEABLE BEHIND CHECKS THREADS LAST LGTM
for pr in $(gh pr list --json number --jq 'sort_by(.number) | .[].number'); do
  info="$(gh pr view "$pr" --json headRefName,mergeable,statusCheckRollup,comments)"
  branch="$(jq -r .headRefName <<<"$info")"
  behind="$(git rev-list --count "origin/$branch..origin/main" 2>/dev/null || echo '?')"
  checks="$(jq -r '
    .statusCheckRollup | if length == 0 then "none" else
      ([.[] | select((.conclusion // .state // "") | test("^(FAILURE|ERROR|CANCELLED|TIMED_OUT|ACTION_REQUIRED)$")) | .name // .context]) as $bad
      | ([.[] | select((.status // "") | test("^(QUEUED|IN_PROGRESS|PENDING|WAITING)$"))] | length) as $pending
      | if ($bad | length) > 0 then "failing: " + ($bad | join(","))
        elif $pending > 0 then "pending"
        else "ok" end
    end' <<<"$info")"
  threads="$(gh api graphql -f query="{repository(owner:\"$owner\",name:\"$repo\"){pullRequest(number:$pr){reviewThreads(first:100){nodes{isResolved}}}}}" \
    --jq '[.data.repository.pullRequest.reviewThreads.nodes[] | select(.isResolved | not)] | length')"
  last="$(jq -r '.comments[-1] // empty | if (.body | test("Generated with \\[Claude Code\\]")) then "agent" else "human" end' <<<"$info")"
  lgtm="$( (scripts/lgtm-check.sh "$pr" 2>&1 || true) | tail -1 | sed 's/^not mergeable: //; s/^mergeable: .*/ready to merge/')"
  printf '%-5s %-38s %-12s %-7s %-22s %-8s %-6s %s\n' "#$pr" "$branch" "$(jq -r .mergeable <<<"$info")" \
    "$behind" "$checks" "$threads" "${last:--}" "$lgtm"
done
