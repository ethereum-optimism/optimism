#!/usr/bin/env bash
set -euo pipefail

# Checks a `forge test --gas-report --json` stream against the EIP-7825 per-transaction gas
# cap (2^24 = 16,777,216). Fails if any src/ contract deployment, or any single function
# call, used GAS_CAP_BUFFER_PCT% of the cap or more. The buffer reserves headroom for
# intrinsic transaction gas and for an outer caller such as a Gnosis Safe (execTransaction
# overhead plus EIP-150 63/64 gas forwarding), which consume part of the capped budget
# before the operation itself runs.
#
# The gas report measures call frames, so numbers exclude intrinsic tx gas; the buffer
# absorbs that too. A call that reverts records max 0 — only operations that succeed in the
# tests are measured.
#
# Usage: GAS_CAP_BUFFER_PCT=80 check-gas-cap.sh <gas-report.json>

GAS_CAP="${GAS_CAP:-16777216}"
GAS_CAP_BUFFER_PCT="${GAS_CAP_BUFFER_PCT:-80}"
REPORT="${1:?usage: check-gas-cap.sh <gas-report.json>}"

TARGET=$((GAS_CAP * GAS_CAP_BUFFER_PCT / 100))

# The file is JSON-lines; the gas report is the array whose entries carry a "contract" key.
if ! jq -es 'map(select(type == "array" and length > 0 and (.[0] | type == "object" and has("contract")))) | length > 0' "$REPORT" > /dev/null; then
  echo "error: no gas report found in $REPORT (did the forge run fail before reporting?)" >&2
  exit 1
fi

flags=$(jq -r --argjson target "$TARGET" '
  select(type == "array" and length > 0 and (.[0] | type == "object" and has("contract")))
  | .[]
  | select(.contract | startswith("src/"))
  | .contract as $name
  | ({op: "(deployment)", gas: .deployment.gas},
     (.functions | to_entries[] | {op: .key, gas: .value.max}))
  | select(.gas >= $target)
  | "\($name)\t\(.op)\t\(.gas)"
' "$REPORT" | sort -t "$(printf '\t')" -k3 -rn)

if [ -n "$flags" ]; then
  count=$(echo "$flags" | wc -l | tr -d ' ')
  echo "error: $count operation(s) reached ${GAS_CAP_BUFFER_PCT}% of the EIP-7825 tx gas cap (target ${TARGET}, cap ${GAS_CAP}):" >&2
  printf 'CONTRACT\tOPERATION\tMAX GAS\n%s\n' "$flags" | column -t -s "$(printf '\t')" >&2
  exit 1
fi

echo "check-gas-cap: all measured src/ operations fit within ${TARGET} gas (${GAS_CAP_BUFFER_PCT}% of ${GAS_CAP})"
