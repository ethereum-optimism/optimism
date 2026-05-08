#!/usr/bin/env bash
# Merges the four continuation YAML configs into a single file using yq v4.
# The merged file is written to /tmp/merged-config.yml for the continuation step.
#
# Merge order: main → docs-ci → rust-ci → rust-e2e
# Later files win on key conflicts (same as path-filtering orb behaviour).
set -euo pipefail

YQ_VERSION="4.44.5"  # Keep in sync with mise.toml

# ---------------------------------------------------------------------------
# Ensure yq v4 is available
# ---------------------------------------------------------------------------
if ! command -v yq &>/dev/null || [[ "$(yq --version 2>&1)" != *"v${YQ_VERSION}"* ]]; then
  echo "Installing yq ${YQ_VERSION}..."
  wget -qO /tmp/yq \
    "https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_linux_amd64"
  chmod +x /tmp/yq
  YQ=/tmp/yq
else
  YQ=yq
fi

echo "yq version: $($YQ --version)"

# ---------------------------------------------------------------------------
# Deep-merge all continuation configs
# ---------------------------------------------------------------------------
$YQ eval-all '. as $item ireduce ({}; . * $item)' \
  .circleci/continue/main.yml \
  .circleci/continue/docs-ci.yml \
  .circleci/continue/rust-ci.yml \
  .circleci/continue/rust-e2e.yml \
  > /tmp/merged-config.yml

echo "Merged config written to /tmp/merged-config.yml ($(wc -l < /tmp/merged-config.yml) lines)"
