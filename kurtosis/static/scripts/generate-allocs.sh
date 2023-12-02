#!/bin/sh

set -euo pipefail

mkdir -p /allocs
curl -X POST -H "Content-Type: application/json" -d \
  '{"jsonrpc":"2.0","method":"anvil_dumpState","params":[],"id":1}' $1 | \
  jq -r '.result' | xxd -r -p | gzip -d | \
  jq -r '.accounts | to_entries | map({key: (.key | sub("^0x"; "")), value: .value}) | from_entries' \
  > /allocs/allocs.json