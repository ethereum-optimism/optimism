#!/usr/bin/env bash
# Recompile L2ToL2CrossDomainMessenger with the repository's default foundry profile, record the
# artifact, and regenerate the Lean bytecode (ExpiryEvm/Bytecode.lean) and the EquiVM block
# summaries (ExpiryEvm/Blocks/). Run from anywhere; needs forge, jq, cast, python3, and a
# `lake build` (or `lake update`) done once so that .lake/packages/EquiVM exists.
set -euo pipefail
HERE=$(cd "$(dirname "$0")/.." && pwd)
CB=$(cd "$HERE/../../../.." && pwd)            # packages/contracts-bedrock
NAME=L2ToL2CrossDomainMessenger
(cd "$CB" && forge build "src/L2/$NAME.sol")
A="$CB/forge-artifacts/$NAME.sol/$NAME.json"
jq -r .deployedBytecode.object "$A" > "$HERE/artifacts/$NAME.runtime.hex"
jq -r .bytecode.object "$A" > "$HERE/artifacts/$NAME.creation.hex"
echo "solc:            $(jq -r .metadata.compiler.version "$A")"
echo "settings:        $(jq -c '.metadata.settings | {optimizer, evmVersion, metadata}' "$A")"
echo "keccak(runtime): $(cast keccak "$(cat "$HERE/artifacts/$NAME.runtime.hex")")"
echo "keccak(initcode):$(cast keccak "$(cat "$HERE/artifacts/$NAME.creation.hex")")"
echo "semver-lock:     $(jq -r ".\"src/L2/$NAME.sol:$NAME\".initCodeHash" "$CB/snapshots/semver-lock.json")"
cd "$HERE"
python3 scripts/gen_bytecode.py "artifacts/$NAME.runtime.hex" ExpiryEvm/Bytecode.lean
rm -rf ExpiryEvm/Blocks && mkdir -p ExpiryEvm/Blocks
python3 .lake/packages/EquiVM/scripts/generate_rd_blocks.py "artifacts/$NAME.runtime.hex" \
  --name l2tol2 --code-term ExpiryEvm.l2tol2Runtime --bytecode-import ExpiryEvm.Bytecode \
  --output ExpiryEvm/Blocks/RuntimeBlocks.lean --shard-size 20 > /dev/null
# The expiry window as compiled: the PUSH3 operand right after expireMessage's sentAt check.
echo "MESSAGE_EXPIRY_WINDOW operands (PUSH3 in the runtime; set P_contract in ExpiryEvm/Spec.lean):"
cast disassemble "$(cat "artifacts/$NAME.runtime.hex")" | grep -n "PUSH3" | head
