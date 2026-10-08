#!/usr/bin/env bash
# Recompile L1CrossDomainMessenger with the repository's DEFAULT foundry profile, check the
# creation code against snapshots/semver-lock.json, obtain the deployed runtime (immutables filled
# by the real constructor, via anvil), and regenerate the Lean bytecode (L1cdmEvm/Bytecode.lean)
# and the EquiVM block summaries (L1cdmEvm/Blocks/). Needs forge, anvil, cast, jq, python3, and
# .lake/packages/EquiVM (`lake build` or `lake update` once). Aborts before touching any artifact
# if the compiled init code does not match the semver lock.
set -euo pipefail
HERE=$(cd "$(dirname "$0")/.." && pwd)
CB=$(cd "$HERE/../../../.." && pwd)            # packages/contracts-bedrock
NAME=L1CrossDomainMessenger
export FOUNDRY_PROFILE=default
(cd "$CB" && forge build "src/L1/$NAME.sol")
A="$CB/forge-artifacts/$NAME.sol/$NAME.json"
INIT=$(jq -r .bytecode.object "$A")
LOCK=$(jq -r ".\"src/L1/$NAME.sol:$NAME\".initCodeHash" "$CB/snapshots/semver-lock.json")
if [ "$(cast keccak "$INIT")" != "$LOCK" ]; then
  echo "init code hash $(cast keccak "$INIT") != semver-lock $LOCK; not regenerating" >&2
  exit 1
fi
# Deploy the init code on a throwaway anvil to get the runtime with immutables filled in.
PORT=18547
anvil --port $PORT --silent & APID=$!
trap 'kill $APID 2>/dev/null || true' EXIT
sleep 1.5
PK=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
ADDR=$(cast send --rpc-url http://127.0.0.1:$PORT --private-key $PK --create "$INIT" --json | jq -r .contractAddress)
RUNTIME=$(cast code --rpc-url http://127.0.0.1:$PORT "$ADDR")
mkdir -p "$HERE/bytecode"
echo "$RUNTIME" > "$HERE/bytecode/$NAME.runtime.hex"
echo "$INIT" > "$HERE/bytecode/$NAME.creation.hex"
jq -r .deployedBytecode.object "$A" > "$HERE/bytecode/$NAME.runtime-template.hex"
echo "solc:            $(jq -r .metadata.compiler.version "$A")"
echo "settings:        $(jq -c '.metadata.settings | {optimizer, evmVersion, metadata}' "$A")"
echo "immutables:      $(jq -c .deployedBytecode.immutableReferences "$A")"
echo "keccak(runtime): $(cast keccak "$RUNTIME")  (deployed, immutables filled)"
echo "keccak(initcode):$(cast keccak "$INIT")  == semver-lock initCodeHash"
cd "$HERE"
python3 scripts/gen_bytecode.py "bytecode/$NAME.runtime.hex" L1cdmEvm/Bytecode.lean
rm -rf L1cdmEvm/Blocks && mkdir -p L1cdmEvm/Blocks
python3 .lake/packages/EquiVM/scripts/generate_rd_blocks.py "bytecode/$NAME.runtime.hex" \
  --name l1cdm --code-term L1cdmEvm.l1cdmRuntime --bytecode-import L1cdmEvm.Bytecode \
  --output L1cdmEvm/Blocks/RuntimeBlocks.lean --shard-size 20 > /dev/null
# Kernel-checked closed facts instead of `native_decide` (see L1cdmEvm/KernelDecide.lean).
perl -pi -e 's/native_decide/evm_kdecide/g' L1cdmEvm/Blocks/RuntimeBlocks_*.lean
# Values the proofs read from the compiled code (see L1cdmEvm/Spec.lean):
#   the exporter address (Predeploys.UNDELIVERED_MESSAGE_EXPORTER, a PUSH20 in relayUndeliveredMessage)
#   and EXPIRE_MESSAGE_GAS_LIMIT (PUSH3 0x0186a0).
echo "PUSH20 0x4200…: $(cast disassemble "$RUNTIME" | grep -io 'PUSH20 0x42000000000000000000000000000000000000[0-9a-f]*' | sort | uniq -c | tr '\n' ' ')"
