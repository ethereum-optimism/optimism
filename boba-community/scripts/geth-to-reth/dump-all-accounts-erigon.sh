#!/usr/bin/env bash
set -euo pipefail

# Dump all accounts from erigon using debug_accountRange (handles pagination)
# Erigon differences from geth:
#   - 5 params instead of 6 (no 'incompletes' flag)
#   - start key is base64, not hex
#   - state root is 0x-prefixed
# Usage: dump-all-accounts-erigon.sh <block-hex> <output.json>

BLOCK="${1:?Usage: $0 <block-hex> <output.json>}"
OUTPUT="${2:?Usage: $0 <block-hex> <output.json>}"
RPC="${RPC_URL:-http://localhost:9545}"

PAGE=0
TOTAL=0
# 32 zero bytes in base64
START="AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

echo "Dumping all accounts at block $BLOCK from $RPC (erigon mode)"

while true; do
    PAGE=$((PAGE + 1))
    OUTFILE="$TMPDIR/page-${PAGE}.json"

    curl -s -X POST "$RPC" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"method\":\"debug_accountRange\",\"params\":[\"$BLOCK\",\"${START}\",256,false,false],\"id\":${PAGE}}" \
        > "$OUTFILE"

    # Check for RPC errors
    ERROR=$(jq -r '.error.message // empty' "$OUTFILE")
    if [ -n "$ERROR" ]; then
        echo "ERROR on page $PAGE: $ERROR"
        exit 1
    fi

    COUNT=$(jq '.result.accounts | length' "$OUTFILE")
    NEXT_B64=$(jq -r '.result.next // empty' "$OUTFILE")
    ROOT=$(jq -r '.result.root' "$OUTFILE")
    TOTAL=$((TOTAL + COUNT))

    echo "Page $PAGE: $COUNT accounts (total: $TOTAL)"

    if [ "$COUNT" -eq 0 ] || [ -z "$NEXT_B64" ]; then
        echo "Done. Total accounts: $TOTAL"
        break
    fi

    # Erigon returns base64 next cursor, pass directly as next start
    START="$NEXT_B64"

    if [ "$PAGE" -gt 10000 ]; then
        echo "ERROR: Too many pages (>10000), aborting"
        exit 1
    fi
done

# Merge all pages into single dump
# Strip 0x from root to match geth format (convert-to-reth.sh expects no prefix)
echo "Merging $PAGE pages..."
jq -s '
    {
        result: {
            root: (.[0].result.root | if startswith("0x") then .[2:] else . end),
            accounts: (reduce .[].result.accounts as $accts ({}; . + $accts))
        }
    }
' "$TMPDIR"/page-*.json > "$OUTPUT"

FINAL_COUNT=$(jq '.result.accounts | length' "$OUTPUT")
echo "Wrote $OUTPUT with $FINAL_COUNT accounts"
