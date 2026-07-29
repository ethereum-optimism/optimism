#!/usr/bin/env bash
#
# check-game-block-hashes.sh
#
# Compare the L2 block hash a node returns against a reference node for a set of
# L2 blocks. A mismatch means the node is on a different chain than the reference
# for that block — i.e. a challenger using it would derive output roots from a
# diverged history. Chain-agnostic: pass any two EL RPCs and any block numbers.
#
# Usage:
#   check-game-block-hashes.sh <node-rpc> <reference-rpc> [block ...]
#
#   <node-rpc>       EL JSON-RPC of the node under test (e.g. our op-geth).
#   <reference-rpc>  EL JSON-RPC trusted as the reference (e.g. a public RPC).
#   [block ...]      L2 block numbers (decimal). If omitted, read one per line
#                    from stdin (e.g. piped from the proposal blocks of a game set).
#
# Env: RETRIES (per-request attempts, default 5).
# Exit code: 0 if every block matches, 1 if any mismatch / RPC failure.

set -uo pipefail

NODE_RPC="${1:-}"; REF_RPC="${2:-}"
if [[ -z "$NODE_RPC" || -z "$REF_RPC" ]]; then
  echo "usage: $0 <node-rpc> <reference-rpc> [block ...]" >&2
  exit 2
fi
shift 2

RETRIES="${RETRIES:-5}"

if [[ $# -gt 0 ]]; then BLOCKS=("$@"); else mapfile -t BLOCKS; fi
if [[ ${#BLOCKS[@]} -eq 0 ]]; then echo "no blocks given (args or stdin)" >&2; exit 2; fi

# fetch_hash <rpc> <decimal-block> -> block hash, only if the node returns the
# exact block requested (guards against a lagging/pruned/load-balanced backend).
fetch_hash() {
  local rpc="$1" dec="$2" hexbn payload resp out attempt
  hexbn=$(printf '0x%x' "$dec")
  payload="{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_getBlockByNumber\",\"params\":[\"$hexbn\",false]}"
  for ((attempt = 1; attempt <= RETRIES; attempt++)); do
    resp=$(curl -s -m 20 -X POST "$rpc" -H 'content-type: application/json' --data "$payload")
    out=$(printf '%s' "$resp" | python3 -c '
import sys, json
try: d = json.load(sys.stdin)
except Exception: print("RETRY"); sys.exit()
r = d.get("result")
if not r or r.get("hash") is None or r.get("number") is None: print("RETRY"); sys.exit()
if int(r["number"], 16) != int(sys.argv[1]): print("RETRY"); sys.exit()  # backend returned a different block
print(r["hash"].lower())
' "$dec" 2>/dev/null)
    if [[ "$out" != "RETRY" && -n "$out" ]]; then printf '%s' "$out"; return 0; fi
    sleep 1
  done
  printf ''; return 1
}

printf '%-10s  %-66s  %-66s  %s\n' "Block" "Node" "Reference" "Result"
fail=0
for bn in "${BLOCKS[@]}"; do
  node=$(fetch_hash "$NODE_RPC" "$bn")
  ref=$(fetch_hash "$REF_RPC" "$bn")
  if [[ -z "$node" || -z "$ref" ]]; then
    result="⚠️  RPC-FAIL (node=${node:-none} ref=${ref:-none})"; fail=1
  elif [[ "$node" == "$ref" ]]; then
    result="✅ match"
  else
    result="❌ MISMATCH"; fail=1
  fi
  printf '%-10s  %-66s  %-66s  %s\n' "$bn" "${node:-<none>}" "${ref:-<none>}" "$result"
done

echo
if [[ "$fail" -eq 0 ]]; then echo "All ${#BLOCKS[@]} blocks match between the node and ${REF_RPC}."
else echo "One or more blocks did NOT match (or an RPC failed)."; fi
exit "$fail"
