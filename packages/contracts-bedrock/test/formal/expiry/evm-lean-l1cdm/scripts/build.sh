#!/usr/bin/env bash
# Build everything and check the axiom footprint. The 30 generated block shards are built in
# batches of 6: kernel-checking their bytecode facts takes ~1–2 GB per shard, so building all of
# them at once can exceed a 16 GB memory cap. Prefix heavy commands with a memory cap on shared
# machines, e.g. `systemd-run --user --scope -p MemoryMax=16G -p MemorySwapMax=0 scripts/build.sh`.
set -euo pipefail
cd "$(dirname "$0")/.."
lake exe cache get >/dev/null 2>&1 || true
lake build L1cdmEvm.Bytecode L1cdmEvm.KernelRun
for s in 1 7 13 19 25; do
  t=""
  for i in $(seq $s $((s + 5))); do t="$t L1cdmEvm.Blocks.RuntimeBlocks_$(printf %03d "$i")"; done
  # shellcheck disable=SC2086
  lake build $t
done
lake build L1cdmEvm L1cdmEvm.Axioms
