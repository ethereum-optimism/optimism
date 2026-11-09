#!/usr/bin/env bash
set -euo pipefail

# Runs `apply --deployment-target genesis` with the mock SP1 verifier in an op-deployer image, on a
# fresh custom intent that uses the image's embedded contract artifacts.
#
# Usage: genesis-smoke-test.sh <image>

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <image>" >&2
  exit 1
fi
image="$1"

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT
intent="$workdir/intent.toml"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

op_deployer() {
  docker run --rm --user "$(id -u):$(id -g)" -v "$workdir:/data" \
    -e DEPLOYER_DEPLOY_MOCK_SP1_VERIFIER=true \
    --entrypoint op-deployer "$image" --cache-dir /data/.cache "$@"
}

op_deployer init --intent-type custom --l1-chain-id 900 --l2-chain-ids 901 --workdir /data

# init leaves roles zeroed and the EIP-1559 parameters unset, which apply rejects. Its custom gas
# token table is dropped so the chain uses ETH.
awk '
  /^[[:space:]]*\[chains\.customGasToken\][[:space:]]*$/ { skip = 1; next }
  skip && /^[[:space:]]*(\[|$)/ { skip = 0 }
  !skip
' "$intent" > "$intent.new"
mv "$intent.new" "$intent"
sed -i.orig \
  -e 's/"0x0000000000000000000000000000000000000000"/"0x1111111111111111111111111111111111111111"/' \
  -e 's/eip1559DenominatorCanyon = 0$/eip1559DenominatorCanyon = 250/' \
  -e 's/eip1559Denominator = 0$/eip1559Denominator = 50/' \
  -e 's/eip1559Elasticity = 0$/eip1559Elasticity = 6/' \
  "$intent"

if grep -q 'customGasToken' "$intent"; then
  fail "could not remove the custom gas token table from the init intent"
fi
if grep -Eq '0x0{40}([^0-9a-fA-F]|$)' "$intent"; then
  fail "the init intent has a zero address this script doesn't replace"
fi
if grep -Eq 'eip1559(DenominatorCanyon|Denominator|Elasticity) = 0$' "$intent"; then
  fail "could not set the EIP-1559 parameters in the init intent"
fi

op_deployer apply --deployment-target genesis --workdir /data

verifier="$(jq -r '.sp1Verifier // empty' "$workdir/state.json")"
[[ -n "$verifier" ]] || fail "the genesis apply recorded no SP1 verifier"
code="$(jq -r '.l1StateDump' "$workdir/state.json" | base64 -d | gunzip |
  jq -r --arg addr "${verifier,,}" '.[$addr].code // empty')"
[[ -n "$code" && "$code" != "0x" ]] || fail "the L1 genesis has no code at SP1 verifier $verifier"
echo "PASS: genesis apply deployed the mock SP1 verifier at $verifier"
