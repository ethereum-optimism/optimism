#!/usr/bin/env bash
# Determines the PR target branch and exports TARGET_BRANCH
# Can be sourced by other scripts: source scripts/ops/get-target-branch.sh
# Or called with bash: TARGET_BRANCH=$(bash scripts/ops/get-target-branch.sh)

set -euo pipefail

TARGET_BRANCH=""

# If this is a PR, get the target branch from GitHub API
if [ -n "${CIRCLE_PULL_REQUEST:-}" ]; then
  PR_NUMBER="${CIRCLE_PULL_REQUEST##*/}"
  TARGET_BRANCH=$(curl -sS --fail --connect-timeout 10 --max-time 30 \
    "https://api.github.com/repos/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}/pulls/${PR_NUMBER}" \
    2>/dev/null | jq -r .base.ref || echo "")
fi

# Fallbacks when not a PR or API did not return a branch
if [ -z "$TARGET_BRANCH" ] || [ "$TARGET_BRANCH" = "null" ]; then
  # In CircleCI, use the current branch
  # Locally or when CIRCLE_BRANCH isn't set, default to develop
  TARGET_BRANCH="${CIRCLE_BRANCH:-develop}"
fi

echo "Resolved TARGET_BRANCH=$TARGET_BRANCH" >&2

# When sourced, export the variable. When called, output to stdout.
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
  # Script is being sourced
  export TARGET_BRANCH
else
  # Script is being executed
  echo "$TARGET_BRANCH"
fi
