#!/usr/bin/env bash
# Run the op-up interop smoke bridge test against a devnet with two L2 chains.
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
if ! [[ "$LOOPS" =~ ^[0-9]+$ ]]; then
  echo "Error: loops must be a non-negative integer, got $LOOPS" >&2
  exit 1
fi

TRIPS=$(( LOOPS == 0 ? 1 : LOOPS * 2 ))

echo "Loops:      $LOOPS ($TRIPS bridge trips)"
echo ""

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

BUILD_DIR="$(mktemp -d)"
trap 'rm -rf "$BUILD_DIR"' EXIT

go build -o "$BUILD_DIR/op-up" "$REPO_ROOT/op-up"

for ((trip = 1; trip <= TRIPS; trip++)); do
  if (( trip % 2 == 1 )); then
    L2A_RPC="$SOURCE_RPC"
    L2B_RPC="$DEST_RPC"
  else
    L2A_RPC="$DEST_RPC"
    L2B_RPC="$SOURCE_RPC"
  fi

  echo "Bridge trip $trip/$TRIPS"
  "$BUILD_DIR/op-up" smoke-interop bridge \
    --l2a-rpc "$L2A_RPC" \
    --l2b-rpc "$L2B_RPC" \
    --private-key "$PRIVATE_KEY"
done
