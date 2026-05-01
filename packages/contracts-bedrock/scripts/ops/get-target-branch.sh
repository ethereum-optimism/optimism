#!/usr/bin/env bash
# Determines the PR target branch and exports TARGET_BRANCH

resolve_pr_target_branch() {
  local pr_number="${CIRCLE_PULL_REQUEST##*/}"
  local response
  local url
  local curl_args=(-fsSL)

  if [ -n "${GH_TOKEN:-}" ]; then
    curl_args+=(-H "Authorization: Bearer ${GH_TOKEN}")
  elif [ -n "${GITHUB_TOKEN:-}" ]; then
    curl_args+=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
  elif [ -n "${GHTOKEN:-}" ]; then
    curl_args+=(-H "Authorization: Bearer ${GHTOKEN}")
  fi

  if [ -z "${CIRCLE_PROJECT_USERNAME:-}" ] || [ -z "${CIRCLE_PROJECT_REPONAME:-}" ]; then
    echo "Error: CIRCLE_PROJECT_USERNAME and CIRCLE_PROJECT_REPONAME are required to resolve PR target branch" >&2
    return 1
  fi

  url="https://api.github.com/repos/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}/pulls/${pr_number}"
  response="$(curl "${curl_args[@]}" "$url")" || {
    echo "Error: failed to resolve target branch for PR #${pr_number}" >&2
    return 1
  }

  TARGET_BRANCH="$(jq -er '.base.ref | select(type == "string" and length > 0)' <<< "$response")" || {
    echo "Error: GitHub API response did not include base.ref for PR #${pr_number}" >&2
    return 1
  }
}

TARGET_BRANCH=""

# In merge queues, CIRCLE_BRANCH is gh-readonly-queue/<base>/pr-<n>-<sha>.
# Extract the base branch via regex; BASH_REMATCH[1] captures the first
# parenthesised group, i.e. the <base> segment between the slashes.
if [[ "${CIRCLE_BRANCH:-}" =~ ^gh-readonly-queue/([^/]+)/ ]]; then
  TARGET_BRANCH="${BASH_REMATCH[1]}"
elif [ -n "${CIRCLE_PULL_REQUEST:-}" ]; then
  # PR jobs must fail closed if GitHub cannot return the true base branch.
  # Falling back to CIRCLE_BRANCH can compare a branch against itself.
  resolve_pr_target_branch || exit 1
else
  # Non-PR/manual branch jobs do not have a reliable source for the true base.
  # Use the repository default branch instead of CIRCLE_BRANCH so diff-based
  # checks cannot accidentally compare a branch against itself.
  TARGET_BRANCH="develop"
fi

echo "Resolved TARGET_BRANCH=$TARGET_BRANCH" >&2
export TARGET_BRANCH
