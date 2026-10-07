#!/usr/bin/env bash
# Transfer source/fixtures without Git history, tool caches, or old verdicts.
# Keep the whole source tree, including recursively populated submodules.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../../.."
: "${1:?Pass a workspace-relative archive path under .ci}"
case "$1" in .ci/*) ;; *) echo 'Archive must live under .ci.' >&2; exit 1 ;; esac
mkdir -p "$(dirname "$1")"
tar --exclude=.git --exclude=./.ci --exclude=./tmp --exclude=./node_modules \
  --exclude=./rust/target --exclude=./packages/contracts-bedrock/cache \
  --exclude=./packages/contracts-bedrock/forge-artifacts \
  --exclude=./packages/contracts-bedrock/results \
  -czf "$1" .
