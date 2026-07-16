#!/usr/bin/env bash
# Merges the three continuation YAML configs into a single file using yq v4.
# yq is installed via mise (see mise.toml).
# The merged file is written to /tmp/merged-config.yml for the continuation step.
#
# Merge order: helpers → main → rust-ci → rust-e2e
# Later files win on key conflicts (same as path-filtering orb behaviour).
# helpers.yml holds shared command definitions (e.g. the Go cache helpers) so
# they live in one place instead of being duplicated across the configs.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTINUE_DIR="$(cd "${SCRIPT_DIR}/../continue" && pwd)"

# Deep-merge all continuation configs.
# explode(.) resolves YAML anchors/aliases before merging so that the output
# never contains undefined alias references (e.g. *rust-cache-version).
# $item is a yq expression variable, not a shell variable.
# Single quotes are intentional to prevent shell expansion.
# shellcheck disable=SC2016
yq eval-all 'explode(.) | . as $item ireduce ({}; . * $item)' \
  "${CONTINUE_DIR}/helpers.yml" \
  "${CONTINUE_DIR}/main.yml" \
  "${CONTINUE_DIR}/rust-ci.yml" \
  "${CONTINUE_DIR}/rust-e2e.yml" \
  > /tmp/merged-config.yml

echo "Merged config written to /tmp/merged-config.yml ($(wc -l < /tmp/merged-config.yml) lines)"
