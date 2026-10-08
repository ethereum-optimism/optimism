#!/usr/bin/env bash
# Verify the expiry bridge lemmas against the op-supernode Dafny model.
#   DAFNY=/path/to/dafny ./run.sh            # bridge + expected failures
#   DAFNY=/path/to/dafny ./run.sh --model    # also re-verify the whole existing model,
#                                            # all six files (~10 min on a loaded 32-core box)
# Exits non-zero if ExpiryBridge.dfy has any error, or if ExpectFail.dfy does not fail in exactly
# the expected way (see below).
set -euo pipefail
cd "$(dirname "$0")"
DAFNY="${DAFNY:-dafny}"
CORES="${CORES:-8}"
# --allow-warnings: the included model has three `assume` statements without {:axiom}
# (Interop.dfy:1019, 1020, 1631); Dafny warns on them and would otherwise exit 2.
FLAGS=(--cores "$CORES" --allow-warnings)

echo "dafny: $("$DAFNY" --version)"

if [[ "${1:-}" == "--model" ]]; then
  echo "== existing model: op-supernode/dafny-models (Interop.dfy + every included file)"
  # --verify-included-files: Interop.dfy includes VerifiedDB, ChainContainer, LogsDB, Types, Utils;
  # without it Dafny verifies only Interop.dfy's own declarations.
  # A generous per-VC time limit: under heavy host load the default limit made
  # ApplyPendingTransition's VC time out (internal prover error).
  "$DAFNY" verify "${FLAGS[@]}" --verify-included-files --verification-time-limit 1200 \
    ../../../../../../op-supernode/dafny-models/Interop.dfy
fi

echo "== ExpiryBridge.dfy (must verify with 0 errors)"
"$DAFNY" verify "${FLAGS[@]}" ExpiryBridge.dfy

echo "== ExpectFail.dfy (must fail: exit 4, exactly the two EXPECT-FAIL postconditions)"
set +e
out="$("$DAFNY" verify "${FLAGS[@]}" ExpectFail.dfy 2>&1)"
status=$?
set -e
echo "$out" | grep -E "^ExpectFail.dfy.*(Error|Related location)|verifier finished" || true

fail() { echo "FAIL: ExpectFail.dfy: $*" >&2; exit 1; }
# Dafny exit code 4 = verification errors (not a parse/resolution error, not success).
[[ $status -eq 4 ]] || fail "expected exit status 4, got $status"
echo "$out" | grep -q "finished with 2 verified, 2 errors" || fail "expected '2 verified, 2 errors'"
# Every error must be a postcondition failure.
n_err=$(echo "$out" | grep -cE "^ExpectFail.dfy\([0-9]+,[0-9]+\): Error:" || true)
n_post=$(echo "$out" | grep -cE "^ExpectFail.dfy\([0-9]+,[0-9]+\): Error: a postcondition could not be proved" || true)
[[ $n_err -eq 2 && $n_post -eq 2 ]] || fail "expected 2 postcondition errors, got $n_post of $n_err"
# The failing postconditions must be exactly the lines marked EXPECT-FAIL (one per lemma).
expected=$(grep -n "// EXPECT-FAIL" ExpectFail.dfy | grep -v "^[0-9]*://" | cut -d: -f1 | sort -n | tr '\n' ' ')
actual=$(echo "$out" | sed -nE 's/^ExpectFail.dfy\(([0-9]+),[0-9]+\): Related location: this is the postcondition that could not be proved.*/\1/p' | sort -n | tr '\n' ' ')
[[ -n "$expected" && "$expected" == "$actual" ]] || fail "failing postcondition lines '$actual' != EXPECT-FAIL lines '$expected'"
echo "OK: NoPeriodAssumption and ExclusiveBoundary are rejected at their EXPECT-FAIL postconditions (lines $expected)"
