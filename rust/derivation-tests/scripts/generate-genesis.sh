#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MONOREPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
OP_DEPLOYER="$MONOREPO_ROOT/bin/op-deployer"

# Build op-deployer if it doesn't exist
if [ ! -f "$OP_DEPLOYER" ]; then
  echo "Building op-deployer..."
  go build -o "$OP_DEPLOYER" "$MONOREPO_ROOT/op-deployer/cmd/op-deployer"
fi

# Create a temp workdir
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# Initialize the workdir scaffold
"$OP_DEPLOYER" init \
  --l1-chain-id 900 \
  --l2-chain-ids 901 \
  --intent-type custom \
  --workdir "$WORKDIR"

# Overwrite the generated intent with our curated one
cp "$SCRIPT_DIR/../intent.toml" "$WORKDIR/intent.toml"

# Pin the CREATE2 salt for deterministic contract addresses across runs.
# Without this, op-deployer generates a random salt, causing different contract
# addresses and genesis hashes on every invocation.
jq '.create2Salt = "0x0000000000000000000000000000000000000000000000000000000000000001"' \
  "$WORKDIR/state.json" > "$WORKDIR/state.tmp" && mv "$WORKDIR/state.tmp" "$WORKDIR/state.json"

# Run apply to generate genesis state
"$OP_DEPLOYER" apply \
  --deployment-target genesis \
  --workdir "$WORKDIR"

# Create output directory
OUTPUT_DIR="$SCRIPT_DIR/../testdata/generated"
mkdir -p "$OUTPUT_DIR"

# Extract L2 genesis and rollup config.
# Flags must appear before the positional chain-id argument.
"$OP_DEPLOYER" inspect genesis --workdir "$WORKDIR" --outfile "$OUTPUT_DIR/genesis.json" 901
"$OP_DEPLOYER" inspect rollup --workdir "$WORKDIR" --outfile "$OUTPUT_DIR/rollup.json" 901

# Reconstruct the full L1 genesis from the state dump.
# op-deployer does not serialize L1DevGenesis to state.json (tagged json:"-"),
# so we reconstruct it using the same logic as SealL1DevGenesis.
echo "Reconstructing L1 genesis..."
go run "$SCRIPT_DIR/extract-l1-genesis" "$WORKDIR/state.json" "$OUTPUT_DIR/l1-genesis.json"

# Compute genesis block hashes and state roots using go-ethereum's Genesis.ToBlock().
# This ensures hash values match exactly what op-program/op-geth compute.
echo "Computing genesis hashes..."
go run "$SCRIPT_DIR/compute-genesis-hashes" \
  "$OUTPUT_DIR/l1-genesis.json" \
  "$OUTPUT_DIR/genesis.json" \
  "$OUTPUT_DIR/genesis-hashes.json"

echo "Genesis files generated in $OUTPUT_DIR"
