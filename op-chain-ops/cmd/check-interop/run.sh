#!/usr/bin/env bash
# Run check-interop against a devnet with two L2 chains.
#
# Usage:
#   ./run.sh <devnet-dir> <private-key> [loops]
#
# Example:
#   ./run.sh ~/workspace/ethereum-optimism/devnets-private/dev/sdg-v1 abc123...def 10

set -euo pipefail

DEVNET_DIR="${1:?Usage: $0 <devnet-dir> <private-key> [loops]}"
PRIVATE_KEY="${2:?Usage: $0 <devnet-dir> <private-key> [loops]}"
LOOPS="${3:-10}"

MANIFEST="$DEVNET_DIR/manifest.yaml"
if [[ ! -f "$MANIFEST" ]]; then
  echo "Error: manifest.yaml not found at $MANIFEST" >&2
  exit 1
fi

# Parse the first two chain names from manifest.yaml.
mapfile -t CHAINS < <(grep '^\s*- name:' "$MANIFEST" | head -2 | awk '{print $3}')

if [[ ${#CHAINS[@]} -lt 2 ]]; then
  echo "Error: expected at least 2 chains in $MANIFEST, found ${#CHAINS[@]}" >&2
  exit 1
fi

# Construct RPC URLs from chain names.
RPC_BASE="us.networks.ent.dev.oplabs.cloud"
SOURCE_RPC="https://an-${CHAINS[0]}-opn-reth-a-rpc-0-op-reth.${RPC_BASE}"
DEST_RPC="https://an-${CHAINS[1]}-opn-reth-a-rpc-0-op-reth.${RPC_BASE}"

echo "Devnet:     $(basename "$DEVNET_DIR")"
echo "Chain A:    ${CHAINS[0]} -> $SOURCE_RPC"
echo "Chain B:    ${CHAINS[1]} -> $DEST_RPC"
echo "Loops:      $LOOPS"
echo ""

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

exec go run "$REPO_ROOT/op-chain-ops/cmd/check-interop" \
  --source-rpc "$SOURCE_RPC" \
  --dest-rpc "$DEST_RPC" \
  --relay \
  --loop "$LOOPS" \
  --private-key "$PRIVATE_KEY"
