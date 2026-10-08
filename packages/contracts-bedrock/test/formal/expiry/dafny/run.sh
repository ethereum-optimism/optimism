#!/usr/bin/env bash
# Verify the expiry bridge lemmas against the op-supernode Dafny model.
#   DAFNY=/path/to/dafny ./run.sh            # bridge + expected failures
#   DAFNY=/path/to/dafny ./run.sh --model    # also re-verify the whole existing model (~minutes)
# Exits non-zero if ExpiryBridge.dfy has any error, or if ExpectFail.dfy does not fail with
# exactly two errors.
set -euo pipefail
cd "$(dirname "$0")"
DAFNY="${DAFNY:-dafny}"
CORES="${CORES:-8}"
# --allow-warnings: the included model has three `assume` statements without {:axiom}
# (Interop.dfy:1019, 1020, 1631); Dafny warns on them and would otherwise exit 2.
FLAGS=(--cores "$CORES" --allow-warnings)

echo "dafny: $("$DAFNY" --version)"

if [[ "${1:-}" == "--model" ]]; then
  echo "== existing model: op-supernode/dafny-models/Interop.dfy"
  # A generous per-VC time limit: under heavy host load the default limits made
  # ApplyPendingTransition's VC time out (internal prover error); with 1200 s all 7313 VCs pass.
  "$DAFNY" verify "${FLAGS[@]}" --verification-time-limit 1200 ../../../../../../op-supernode/dafny-models/Interop.dfy
fi

echo "== ExpiryBridge.dfy (must verify with 0 errors)"
"$DAFNY" verify "${FLAGS[@]}" ExpiryBridge.dfy

echo "== ExpectFail.dfy (must fail with exactly 2 errors)"
set +e
out="$("$DAFNY" verify "${FLAGS[@]}" ExpectFail.dfy 2>&1)"
set -e
echo "$out" | grep -E "^ExpectFail.dfy.*Error|verifier finished"
if ! echo "$out" | grep -q "finished with 2 verified, 2 errors"; then
  echo "FAIL: ExpectFail.dfy did not fail as expected" >&2
  exit 1
fi
echo "OK: both too-strong variants are rejected"
