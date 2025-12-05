#!/usr/bin/env bash
set -euo pipefail

# Pulls artifacts with conditional fallback based on branch and PR labels
# - PR branches: Use fallback by default (faster builds)
# - develop branch: Always build fresh (accuracy)
# - force-use-fresh-artifacts label: Override fallback (emergency escape hatch)

USE_FALLBACK=false

# Check if we're on a PR (not develop branch)
if [ "${CIRCLE_BRANCH:-}" != "develop" ]; then
  USE_FALLBACK=true

  # Check if PR has force-use-fresh-artifacts label (override fallback)
  if [ -n "${CIRCLE_PULL_REQUEST:-}" ]; then
    # Extract PR number from URL
    PR_NUMBER=$(echo "${CIRCLE_PULL_REQUEST}" | grep -o '[0-9]*$')

    # Query GitHub API for PR details (fail safe: proceed with fallback on error)
    if PR_DATA=$(curl -sS --fail --connect-timeout 10 --max-time 30 -H "Authorization: token ${MISE_GITHUB_TOKEN}" \
      "https://api.github.com/repos/ethereum-optimism/optimism/pulls/${PR_NUMBER}" 2>/dev/null); then

      if echo "$PR_DATA" | jq -e 'any(.labels[]; .name == "force-use-fresh-artifacts")' >/dev/null 2>&1; then
        echo "Force use fresh artifacts label detected, skipping fallback"
        USE_FALLBACK=false
      fi
    else
      echo "Warning: Failed to fetch PR labels from GitHub API, proceeding with fallback"
    fi
  fi
fi

# Pull artifacts with or without fallback
if [ "$USE_FALLBACK" = "true" ]; then
  bash scripts/ops/pull-artifacts.sh --fallback-to-latest
else
  bash scripts/ops/pull-artifacts.sh
fi
