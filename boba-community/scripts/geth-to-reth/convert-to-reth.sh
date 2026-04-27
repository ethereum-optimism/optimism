#!/usr/bin/env bash
set -euo pipefail

# Convert geth debug_dumpBlock JSON to reth init-state JSONL format
#
# Key format differences:
# - storage values: geth has variable-length hex (no 0x), reth needs 0x + zero-padded to 64 hex chars

INPUT="${1:?Usage: $0 <geth-dump.json> <output.jsonl>}"
OUTPUT="${2:?Usage: $0 <geth-dump.json> <output.jsonl>}"

echo "Converting $INPUT -> $OUTPUT"

# Line 1: state root with 0x prefix
jq -c '{root: ("0x" + .result.root)}' "$INPUT" > "$OUTPUT"

# Remaining lines: one account per line
# - code: already 0x-prefixed from geth, pass through
# - storage values: zero-pad to 64 hex chars, add 0x prefix
# - balance/nonce: pass through as decimal (reth accepts HexOrDecimal)
jq -c '
  # Helper: left-pad a hex string (without 0x) to 64 chars
  def pad64: ("0" * (64 - length)) + .;

  .result.accounts | to_entries[] |
  {
    address: (if .value.address then .value.address else .key end),
    balance: .value.balance,
    nonce: .value.nonce,
    code: (if .value.code then .value.code else "0x" end),
    storage: (
      if .value.storage then
        .value.storage | with_entries(.value = "0x" + (.value | pad64))
      else {} end
    )
  }
' "$INPUT" >> "$OUTPUT"

LINES=$(wc -l < "$OUTPUT")
echo "Wrote $LINES lines (1 root + $((LINES - 1)) accounts)"
