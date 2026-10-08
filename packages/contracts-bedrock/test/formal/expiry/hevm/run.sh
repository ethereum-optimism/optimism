#!/usr/bin/env bash
# Runs the equivalence checks of develop's vs the current L2ToL2CrossDomainMessenger.
#   1. bytecode level (`hevm equivalence`, abstract storage) for the shared getters, plus expected-FAIL checks;
#   2. the harness L2ToL2Equivalence.t.sol with hevm (`hevm test`, contract L2ToL2Equivalence_Hevm, prove_*);
#   3. the same harness with Halmos (contract L2ToL2Equivalence_Halmos, check_*).
#   Non-vacuity checks (*_nonvacuity_*) must FAIL; everything else must PASS.
# Needs: hevm (tested with 0.58.0), z3 (4.13.3), halmos (0.3.3), forge, and solc 0.8.25 (hevm shells out to `solc` to
# ABI-encode --sig). Regenerate the bytecode first if the contract changed: gen-bytecodes.sh <develop dir>.
# Env: SOLC (path to solc 0.8.25; default: the svm install), SOLVER (z3), SMT_TIMEOUT (seconds, 900),
#      MATCH (regex over function names, default all), MEM_CAP (default 16G: every hevm/halmos call runs in a systemd
#      user scope with that MemoryMax and no swap, so a blow-up kills only that call; hevm can otherwise use tens of
#      GB), NUM_SOLVERS (hevm solver processes, default 4; they count against the cap).
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$HERE"
SOLVER="${SOLVER:-z3}"
SMT_TIMEOUT="${SMT_TIMEOUT:-900}"
MATCH="${MATCH:-.}"
MEM_CAP="${MEM_CAP:-16G}"
NUM_SOLVERS="${NUM_SOLVERS:-4}"
# hevm shells out to `solc` (to ABI-encode --sig); pin it to 0.8.25 regardless of shims on PATH.
SOLC="${SOLC:-}"
for c in "$SOLC" "$HOME/.local/share/svm/0.8.25/solc-0.8.25" "$HOME/.svm/0.8.25/solc-0.8.25"; do
  [ -n "$c" ] && [ -x "$c" ] && { SOLC="$c"; break; }
done
[ -x "$SOLC" ] || { echo "set SOLC to a solc 0.8.25 binary" >&2; exit 1; }
SOLC_BIN="$(mktemp -d)"
ln -s "$SOLC" "$SOLC_BIN/solc"
export PATH="$SOLC_BIN:$PATH"
solc --version | grep -q "0.8.25" || { echo "solc is not 0.8.25" >&2; exit 1; }
OUT="$HERE/results"
mkdir -p "$OUT"
SUMMARY="$OUT/summary.txt"
: >"$SUMMARY"
status=0

capped() { # run "$@" in a systemd scope with a hard memory cap (no swap); MEM_CAP=none disables it
  if [ "$MEM_CAP" = none ]; then
    "$@"
  else
    systemd-run --user --scope --quiet -p MemoryMax="$MEM_CAP" -p MemorySwapMax=0 "$@"
  fi
}
if [ "$MEM_CAP" != none ] && ! systemd-run --user --scope --quiet true 2>/dev/null; then
  echo "no systemd user manager for the memory cap; set MEM_CAP=none to run uncapped" >&2
  exit 1
fi

note() { echo "$*" | tee -a "$SUMMARY"; }

# expect <PASS|FAIL> <label> <logfile> <cmd...>
expect() {
  local want="$1" label="$2" log="$3"
  shift 3
  local t0 t1 got
  t0=$(date +%s)
  "$@" >"$log" 2>&1
  local rc=$?
  t1=$(date +%s)
  # Classify from the output, not only the exit code (hevm test exits 0 when --match selects nothing).
  # A counterexample is a FAIL even if exploration was partial; a PASS with partial exploration, a solver timeout or
  # an unknown result is only PARTIAL (not a proof). hevm also reports [FAIL] for partial runs, hence the order.
  if grep -qE "Not equivalent|Counterexample:" "$log" && ! grep -q "Counterexample: unknown" "$log"; then
    got=FAIL
  elif grep -qE "partially explore|\[TIMEOUT\]|Counterexample: unknown|\[ERROR\]" "$log"; then
    got=PARTIAL
  elif grep -qF "[FAIL]" "$log"; then
    got=FAIL
  elif grep -qE "No discrepancies found|\[PASS\]" "$log"; then
    got=PASS
  else
    got="ERROR(rc=$rc)"
  fi
  local verdict=ok
  [ "$got" = "$want" ] || { verdict=UNEXPECTED; status=1; }
  note "$(printf '%-11s %-8s %-8s %5ss  %s' "$verdict" "want=$want" "got=$got" "$((t1 - t0))" "$label")"
}

note "hevm $(hevm version 2>/dev/null) solver=$SOLVER smt-timeout=${SMT_TIMEOUT}s"
note "develop runtime sha256 $(sha256sum develop.runtime.hex | cut -c1-16)  current runtime sha256 $(sha256sum current.runtime.hex | cut -c1-16)"

# 1. Bytecode-level equivalence of the shared getters (both codes, same abstract storage, symbolic calldata
#    restricted to the selector and ABI-encoded symbolic arguments).
for sig in "successfulMessages(bytes32)" "sentMessages(uint256)" "messageNonce()" "messageVersion()" \
  "crossDomainMessageSender()" "crossDomainMessageSource()" "crossDomainMessageContext()"; do
  expect PASS "equivalence $sig" "$OUT/equiv-${sig%%(*}.log" \
    capped hevm equivalence --code-a-file develop.runtime.hex --code-b-file current.runtime.hex --sig "$sig" \
    --solver "$SOLVER" --smt-timeout "$SMT_TIMEOUT"
done
# Expected FAIL: version() changed (1.3.2 -> 2.0.0). Shows the getter checks are not vacuous.
expect FAIL "equivalence version() [non-vacuity: intentional change]" "$OUT/equiv-version.log" \
  capped hevm equivalence --code-a-file develop.runtime.hex --code-b-file current.runtime.hex --sig "version()" \
  --solver "$SOLVER" --smt-timeout "$SMT_TIMEOUT"
# Expected FAIL (counterexamples only with target 0x..07): whole sendMessage at bytecode level. hevm only partially
# explores it (symbolic-length bytes), which is why the harness below fixes message lengths.
expect FAIL "equivalence sendMessage(uint256,address,bytes) [expected: cex only for unsafe targets 0x..07 / 0x..16]" \
  "$OUT/equiv-sendMessage.log" \
  capped hevm equivalence --code-a-file develop.runtime.hex --code-b-file current.runtime.hex \
  --sig "sendMessage(uint256,address,bytes memory)" --solver "$SOLVER" --smt-timeout "$SMT_TIMEOUT" \
  --num-solvers "$NUM_SOLVERS"

# Log-blindness witness: two contracts that differ ONLY in emitted event data are reported equivalent, so
# `hevm equivalence` (and the harness, which cannot read logs either) says nothing about events.
TMP="$(mktemp -d)"
cat >"$TMP/A.sol" <<'EOF'
// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;
contract A { event E(uint256 x); uint256 s; function f(uint256 x) external { s = x; emit E(x); } }
EOF
sed 's/emit E(x)/emit E(x ^ 1)/' "$TMP/A.sol" >"$TMP/B.sol"
solc --bin-runtime "$TMP/A.sol" 2>/dev/null | tail -1 >"$TMP/a.hex"
solc --bin-runtime "$TMP/B.sol" 2>/dev/null | tail -1 >"$TMP/b.hex"
expect PASS "log-blindness witness: contracts differing only in event data are 'equivalent'" "$OUT/log-blindness.log" \
  hevm equivalence --code-a-file "$TMP/a.hex" --code-b-file "$TMP/b.hex" --solver "$SOLVER"
rm -rf "$TMP"

# 2. Symbolic harness, hevm.
rm -rf out cache # clean build, so the engines only see current artifacts
forge build >"$OUT/forge-build.log" 2>&1 || { note "forge build FAILED"; exit 1; }
for fn in $(grep -oE 'function prove_[A-Za-z0-9_]+' L2ToL2Equivalence.t.sol | cut -d' ' -f2 | grep -E "$MATCH"); do
  want=PASS
  case "$fn" in prove_nonvacuity_*) want=FAIL ;; esac
  expect "$want" "hevm test $fn" "$OUT/$fn.log" \
    capped hevm test --root . --match "^${fn}\$" --solver "$SOLVER" --smt-timeout "$SMT_TIMEOUT" \
    --num-solvers "$NUM_SOLVERS"
done

# 3. Symbolic harness, Halmos (no assertion timeout: a TIMEOUT would show up as PARTIAL).
for fn in $(grep -oE 'function check_[A-Za-z0-9_]+' L2ToL2Equivalence.t.sol | cut -d' ' -f2 | grep -E "$MATCH"); do
  want=PASS
  case "$fn" in check_nonvacuity_*) want=FAIL ;; esac
  expect "$want" "halmos $fn" "$OUT/halmos-$fn.log" \
    capped halmos --root . --contract L2ToL2Equivalence_Halmos --function "^${fn}\\(" \
    --solver-timeout-assertion 0
done

note "overall: $([ $status -eq 0 ] && echo 'all as expected' || echo 'UNEXPECTED results')"
exit $status
