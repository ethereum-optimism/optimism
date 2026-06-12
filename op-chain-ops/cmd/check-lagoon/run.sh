#!/usr/bin/env bash
# Run the interop-smoke bridge test using a TOML config file.
#
# Usage:
#   ./run.sh <config.toml>
#
# Example:
#   ./run.sh ./bridge.toml

set -euo pipefail

CONFIG="${1:?Usage: $0 <config.toml>}"

if [[ ! -f "$CONFIG" ]]; then
  echo "Error: config file not found: $CONFIG" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

BUILD_DIR="$(mktemp -d)"
trap 'rm -rf "$BUILD_DIR"' EXIT

go build -o "$BUILD_DIR/interop-smoke" "$REPO_ROOT/op-chain-ops/cmd/interop-smoke"
"$BUILD_DIR/interop-smoke" bridge --config "$CONFIG"
