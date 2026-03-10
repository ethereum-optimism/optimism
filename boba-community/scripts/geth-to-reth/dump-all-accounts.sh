#!/usr/bin/env bash
set -euo pipefail

# Dump all accounts from geth using debug_accountRange (handles pagination)
# Usage: dump-all-accounts.sh <block-hex> <output.json>

BLOCK="${1:?Usage: $0 <block-hex> <output.json>}"
OUTPUT="${2:?Usage: $0 <block-hex> <output.json>}"
RPC="${RPC_URL:-http://localhost:9545}"

PAGE=0
TOTAL=0
START="0x0000000000000000000000000000000000000000000000000000000000000000"
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

echo "Dumping all accounts at block $BLOCK from $RPC"

while true; do
    PAGE=$((PAGE + 1))
    OUTFILE="$TMPDIR/page-${PAGE}.json"

    curl -s -X POST "$RPC" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"method\":\"debug_accountRange\",\"params\":[\"$BLOCK\",\"${START}\",256,false,false,false],\"id\":${PAGE}}" \
        > "$OUTFILE"

    COUNT=$(jq '.result.accounts | length' "$OUTFILE")
    NEXT_B64=$(jq -r '.result.next // empty' "$OUTFILE")
    ROOT=$(jq -r '.result.root' "$OUTFILE")
    TOTAL=$((TOTAL + COUNT))

    echo "Page $PAGE: $COUNT accounts (total: $TOTAL)"

    if [ "$COUNT" -eq 0 ] || [ -z "$NEXT_B64" ]; then
        echo "Done. Total accounts: $TOTAL"
        break
    fi

    # Decode base64 next cursor to hex for start parameter
    NEXT_HEX=$(echo -n "$NEXT_B64" | base64 -d | od -A n -t x1 | tr -d ' \n')
    START="0x${NEXT_HEX}"

    if [ "$PAGE" -gt 10000 ]; then
        echo "ERROR: Too many pages (>10000), aborting"
        exit 1
    fi
done

# Merge all pages into single dump with same format as debug_dumpBlock
echo "Merging $PAGE pages..."
jq -s '
    {
        result: {
            root: .[0].result.root,
            accounts: (reduce .[].result.accounts as $accts ({}; . + $accts))
        }
    }
' "$TMPDIR"/page-*.json > "$OUTPUT"

FINAL_COUNT=$(jq '.result.accounts | length' "$OUTPUT")
echo "Wrote $OUTPUT with $FINAL_COUNT accounts"
