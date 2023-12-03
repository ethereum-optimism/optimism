#!/bin/sh

set -euo pipefail

echo 'def lpad(len; fill):
                  tostring | (len - length) as $l | (fill * $l)[:$l] + .;' \
      > $HOME/.jq

mkdir -p /allocs
curl -X POST -H "Content-Type: application/json" -d \
  '{"jsonrpc":"2.0","method":"anvil_dumpState","params":[],"id":1}' $1 | \
  jq -r '.result' | xxd -r -p | gzip -d | \
  jq -r '.accounts | to_entries | map({key: (.key | sub("^0x"; "")), value: .value}) | from_entries' |
  jq -r 'map_values(
    .storage |= (to_entries | map({
      key: (.key | ltrimstr("0x") | lpad(64; "0") | ("0x" + .)),
      value: (.value | ltrimstr("0x") | lpad(64; "0") | ("0x" + .))
    }) | from_entries)
  )' \
  > /allocs/allocs.json