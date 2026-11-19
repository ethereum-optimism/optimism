#!/usr/bin/env bash
set -euo pipefail

# Verifies provenance for all forks whose bundle hash changed vs the PR's target
# branch (TARGET_BRANCH, default develop, or a commit SHA).
# For each fork whose hash changed, checks out the recorded commit,
# regenerates the bundle, and verifies it matches byte-for-byte.
# Unchanged forks are skipped to avoid expensive forge rebuilds.

# TARGET_BRANCH is a branch name, or a commit SHA on a backport push (the
# branch before the push).
base_branch="${TARGET_BRANCH:-develop}"
base_ref="origin/${base_branch}"
if [[ "$base_branch" =~ ^[0-9a-f]{40}$ ]]; then
  base_ref="$base_branch"
fi
git fetch -q origin "$base_branch" || true
git show "${base_ref}:op-core/nuts/fork_lock.toml" > /tmp/base_lock.toml 2> /dev/null || true
for fork in $(yq -p toml -o json op-core/nuts/fork_lock.toml | jq -r 'keys[]'); do
  base_hash=$(yq -p toml ".${fork}.hash" /tmp/base_lock.toml 2> /dev/null || echo "")
  curr_hash=$(yq -p toml ".${fork}.hash" op-core/nuts/fork_lock.toml)
  if [ "$base_hash" != "$curr_hash" ]; then
    echo "Verifying $fork (hash changed)..."
    go run ./ops/scripts/nut-provenance-verify "$fork"
  else
    echo "Skipping $fork (unchanged)"
  fi
done
