#!/usr/bin/env bash
# Switch the evm-lean proof from `native_decide` to kernel checking. Idempotent.
#   usage: evm-lean-kernel/apply.sh [path/to/evm-lean]   (default: ../evm-lean next to this script)
# Then: cd evm-lean && lake build   (ExpiryEvm/Axioms.lean asserts the axiom footprint).
set -euo pipefail
K=$(cd "$(dirname "$0")" && pwd)
E=$(cd "${1:-$K/../evm-lean}" && pwd)
F="$K/files"

# 1. New modules: proved list reformulations of decode / D_J + tactics; kernel `evm_run`.
cp "$F/ExpiryEvm/KernelDecide.lean" "$F/ExpiryEvm/KernelRun.lean" "$E/ExpiryEvm/"
# 2. Axiom check: fail the build if a headline theorem depends on a non-standard axiom.
cp "$F/ExpiryEvm/Axioms.lean" "$E/ExpiryEvm/Axioms.lean"
# 3. Bytecode generator: one flat literal, JUMPDEST table proved by the kernel.
cp "$F/scripts/gen_bytecode.py" "$E/scripts/gen_bytecode.py"
HEX=""
for d in bytecode artifacts; do
  [ -f "$E/$d/L2ToL2CrossDomainMessenger.runtime.hex" ] && { HEX="$d/L2ToL2CrossDomainMessenger.runtime.hex"; break; }
done
[ -n "$HEX" ] || { echo "runtime hex not found under $E/{bytecode,artifacts}"; exit 1; }
( cd "$E" && python3 scripts/gen_bytecode.py "$HEX" ExpiryEvm/Bytecode.lean )
# 4. regen.sh: post-process EquiVM's generated block summaries (once).
POST='perl -pi -e "s/native_decide/evm_kdecide/g" ExpiryEvm/Blocks/RuntimeBlocks_*.lean  # kernel-checked facts (evm-lean-kernel)'
if ! grep -q 'evm_kdecide' "$E/scripts/regen.sh"; then
  perl -0pi -e 's|(--output ExpiryEvm/Blocks/RuntimeBlocks.lean[^\n]*\n)|$1'"$(printf '%s' "$POST" | sed 's/[|&]/\\&/g')"'\n|' "$E/scripts/regen.sh"
fi
grep -q 'evm_kdecide' "$E/scripts/regen.sh" || { echo "could not patch regen.sh"; exit 1; }
# 5. The block summaries already on disk.
perl -pi -e 's/native_decide/evm_kdecide/g' "$E"/ExpiryEvm/Blocks/RuntimeBlocks_*.lean
# 6. Hand-written trace segments.
for f in TraceEntry TraceCall1 TraceCall2 TraceStore; do
  perl -pi -e 's/\bjump_dest\b/kjump_dest/g; s/\bnative_decide\b/evm_kdecide/g; s/\bevm_run\b/kevm_run/g' "$E/ExpiryEvm/$f.lean"
done
grep -q '^import ExpiryEvm.KernelRun' "$E/ExpiryEvm/TraceStore.lean" || \
  perl -0pi -e 's/^/import ExpiryEvm.KernelRun\n/' "$E/ExpiryEvm/TraceStore.lean"
echo "remaining native_decide outside Concrete.lean:"
grep -rn 'native_decide' "$E/ExpiryEvm" --include='*.lean' | grep -v 'Concrete.lean\|Axioms.lean\|KernelDecide.lean\|KernelRun.lean' || echo "  none"
