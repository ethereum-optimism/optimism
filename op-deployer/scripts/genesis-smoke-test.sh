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

op_deployer() {
  docker run --rm --user "$(id -u):$(id -g)" -v "$workdir:/data" \
    -e DEPLOYER_DEPLOY_MOCK_SP1_VERIFIER=true \
    --entrypoint op-deployer "$image" --cache-dir /data/.cache "$@"
}

op_deployer init --intent-type custom --l1-chain-id 900 --l2-chain-ids 901 --workdir /data

# init leaves roles zeroed and the EIP-1559 parameters unset, which apply rejects. The custom gas
# token section is dropped so the chain uses ETH.
sed -i.orig \
  -e '/^  \[chains.customGasToken\]/,$d' \
  -e 's/"0x0000000000000000000000000000000000000000"/"0x1111111111111111111111111111111111111111"/' \
  -e 's/eip1559DenominatorCanyon = 0/eip1559DenominatorCanyon = 250/' \
  -e 's/eip1559Denominator = 0/eip1559Denominator = 50/' \
  -e 's/eip1559Elasticity = 0/eip1559Elasticity = 6/' \
  "$workdir/intent.toml"

op_deployer apply --deployment-target genesis --workdir /data

verifier="$(jq -r '.sp1Verifier // empty' "$workdir/state.json")"
if [[ -z "$verifier" ]]; then
  echo "FAIL: the genesis apply recorded no SP1 verifier" >&2
  exit 1
fi
echo "PASS: genesis apply deployed the mock SP1 verifier at $verifier"
