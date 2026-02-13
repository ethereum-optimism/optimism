#!/usr/bin/env bash
# Determines the PR target branch and exports TARGET_BRANCH

TARGET_BRANCH=""
if [ -n "${CIRCLE_PULL_REQUEST:-}" ]; then
  TARGET_BRANCH=$(curl -s "https://api.github.com/repos/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}/pulls/${CIRCLE_PULL_REQUEST##*/}" | jq -r .base.ref)
fi

# Fallbacks when not a PR or API did not return a branch
if [ -z "$TARGET_BRANCH" ] || [ "$TARGET_BRANCH" = "null" ]; then
  # In merge queues, CIRCLE_BRANCH is gh-readonly-queue/<base>/pr-<n>-<sha>.
  # Extract the base branch via regex; BASH_REMATCH[1] captures the first
  # parenthesised group, i.e. the <base> segment between the slashes.
  if [[ "${CIRCLE_BRANCH:-}" =~ ^gh-readonly-queue/([^/]+)/ ]]; then
    TARGET_BRANCH="${BASH_REMATCH[1]}"
  else
    TARGET_BRANCH="${CIRCLE_BRANCH:-develop}"
  fi
fi

echo "Resolved TARGET_BRANCH=$TARGET_BRANCH" >&2
export TARGET_BRANCH
