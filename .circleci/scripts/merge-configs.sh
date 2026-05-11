#!/usr/bin/env bash
# Merges the four continuation YAML configs into a single file using yq v4.
# yq is installed via mise (see mise.toml).
# The merged file is written to /tmp/merged-config.yml for the continuation step.
#
# Merge order: main → docs-ci → rust-ci → rust-e2e
# Later files win on key conflicts (same as path-filtering orb behaviour).
set -euo pipefail

# Deep-merge all continuation configs.
# explode(.) resolves YAML anchors/aliases before merging so that the output
# never contains undefined alias references (e.g. *rust-cache-version).
# $item is a yq expression variable, not a shell variable.
# Single quotes are intentional to prevent shell expansion.
# shellcheck disable=SC2016
yq eval-all 'explode(.) | . as $item ireduce ({}; . * $item)' \
  .circleci/continue/main.yml \
  .circleci/continue/docs-ci.yml \
  .circleci/continue/rust-ci.yml \
  .circleci/continue/rust-e2e.yml \
  > /tmp/merged-config.yml

echo "Merged config written to /tmp/merged-config.yml ($(wc -l < /tmp/merged-config.yml) lines)"
