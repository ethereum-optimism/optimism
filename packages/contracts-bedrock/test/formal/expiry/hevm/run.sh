#!/usr/bin/env bash
# Runs the equivalence checks of develop's vs the current L2ToL2CrossDomainMessenger (README.md).
#   1. bytecode level (`hevm equivalence`, abstract storage): the shared getters, plus witnesses;
#   2. the harness with hevm (`hevm test`, prove_*);
#   3. the harness with Halmos (check_*; check_allSlots_* with the generic storage layout);
#   4. mutants of the current source: each must be caught by the check named for it, and the
#      unmutated source compiled the same way (M0) must pass those same checks;
#   5. the concrete tests (forge test): the event fuzz and the witness replays.
# Verdicts: PASS = complete exploration, no counterexample, no timeout/unknown/bounded loop.
# WITNESS = an actual counterexample (hevm: a validated one; halmos: exit code COUNTEREXAMPLE with a
# model), required for every *_nonvacuity_* check and every mutant; each Halmos witness also has a
# concrete replay in L2ToL2EquivalenceConcrete.t.sol. Anything else is UNEXPECTED and fails the run.
# Needs: hevm (tested 0.58.0), z3 (4.13.3), halmos (0.3.3), forge, python3, solc 0.8.25 (hevm runs
# `solc` to ABI-encode --sig), and a systemd user manager (for the memory cap).
# Regenerate the bytecode first if the contract changed: gen-bytecodes.sh <develop dir> [--check].
# Env: SOLC (solc 0.8.25; default: the svm install), SOLVER (z3), SMT_TIMEOUT (s, 900),
#      MATCH (regex over check names, default all), MEM_CAP (16G per engine call, no swap;
#      `none` disables), NUM_SOLVERS (hevm solver processes, 4; they count against the cap),
#      FUZZ_RUNS (event fuzz runs per test, 5000), SKIP_MUTANTS=1.
# Results: results/summary.txt (one line per check) and one log per check in results/.
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)" || exit 1
cd "$HERE" || exit 1
SOLVER="${SOLVER:-z3}"
SMT_TIMEOUT="${SMT_TIMEOUT:-900}"
MATCH="${MATCH:-.}"
MEM_CAP="${MEM_CAP:-16G}"
NUM_SOLVERS="${NUM_SOLVERS:-4}"
FUZZ_RUNS="${FUZZ_RUNS:-5000}"
HALMOS_CONTRACT=L2ToL2CrossDomainMessenger_EquivalenceHalmos
HARNESS=L2ToL2Equivalence.t.sol

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

# shellcheck disable=SC2317 # capped is only ever called indirectly, through run_hevm/run_halmos
capped() { # run "$@" in a systemd scope with a hard memory cap and no swap
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
record() { # record <want> <got> <secs> <label>
  local verdict=ok
  [ "$1" = "$2" ] || { verdict=UNEXPECTED; status=1; }
  note "$(printf '%-11s %-12s %-22s %5ss  %s' "$verdict" "want=$1" "got=$2" "$3" "$4")"
}
# ANSI-stripped copy of a log, grepped as a file (piping into `grep -q` under pipefail would turn an
# early grep exit into a SIGPIPE failure on large logs).
strip() { sed -e 's/\x1b\[[0-9;]*m//g' "$1" >"$1.txt"; echo "$1.txt"; }

# Verdict of an hevm log (`hevm test` or `hevm equivalence`).
hevm_verdict() { # hevm_verdict <log> <rc>
  local txt rc="$2"
  txt=$(strip "$1")
  if grep -q "partially explore" "$txt"; then
    echo PARTIAL
  elif grep -qE "Counterexample: *\[validated\]|^Not equivalent" "$txt"; then
    echo WITNESS
  elif grep -qE "Counterexample" "$txt"; then
    echo UNVALIDATED-CEX
  elif [ "$rc" = 0 ] && grep -qE "^ *\[PASS\]|No discrepancies found" "$txt"; then
    echo PASS
  elif grep -qE "all branches reverted|\[FAIL\]" "$txt"; then
    echo FAIL-NO-CEX # e.g. every branch reverted: not a witness
  else
    echo "ERROR(rc=$rc)"
  fi
}

# Verdict of one Halmos test from its --json-output file.
halmos_verdict() { # halmos_verdict <json> <function name>
  python3 -I - "$1" "$2" <<'PY'
import json, sys
try:
    j = json.load(open(sys.argv[1]))
except Exception as e:  # noqa: BLE001
    print("ERROR(no-json)"); sys.exit()
res = [t for ts in j.get("test_results", {}).values() for t in ts if t["name"].split("(")[0] == sys.argv[2]]
if len(res) != 1:
    print(f"ERROR(found {len(res)})"); sys.exit()
t = res[0]
names = {0: "PASS", 1: "COUNTEREXAMPLE", 2: "TIMEOUT", 3: "STUCK", 4: "REVERT_ALL", 5: "EXCEPTION"}
code = names.get(t["exitcode"], f"EXIT{t['exitcode']}")
bounded = t.get("num_bounded_loops") or 0
if bounded:
    print(f"PARTIAL(bounded-loops={bounded})")
elif code == "PASS":
    print("PASS")
elif code == "COUNTEREXAMPLE":
    # A model that mentions an uninterpreted function (keccak) is "potentially invalid" to Halmos;
    # the witnesses are replayed concretely in L2ToL2EquivalenceConcrete.t.sol (step 5).
    models = t.get("models") or []
    if not models:
        print("NO-MODEL")
    elif any((m.get("model") or {}).get("is_valid") for m in models):
        print("WITNESS")
    else:
        print("WITNESS")  # keccak-dependent model
else:
    print(code)
PY
}

run_hevm() { # run_hevm <want> <label> <log> <cmd...>
  local want="$1" label="$2" log="$3"
  shift 3
  local t0 rc
  t0=$(date +%s)
  capped "$@" >"$log" 2>&1
  rc=$?
  record "$want" "$(hevm_verdict "$log" "$rc")" "$(($(date +%s) - t0))" "$label"
}

run_halmos() { # run_halmos <want> <contract> <function> <log-prefix>
  local want="$1" contract="$2" fn="$3" prefix="$4" layout=solidity t0
  case "$fn" in check_allSlots_*) layout=generic ;; esac
  t0=$(date +%s)
  capped halmos --root . --contract "$contract" --function "^${fn}\\(" --storage-layout "$layout" \
    --solver-timeout-assertion 0 --json-output "$prefix.json" >"$prefix.log" 2>&1
  record "$want" "$(halmos_verdict "$prefix.json" "$fn")" "$(($(date +%s) - t0))" \
    "halmos[$layout] $contract.$fn"
}

note "hevm $(hevm version 2>/dev/null) solver=$SOLVER smt-timeout=${SMT_TIMEOUT}s; $(halmos --version 2>/dev/null)"
note "develop runtime sha256 $(sha256sum develop.runtime.hex | cut -c1-16)  current runtime sha256 $(sha256sum current.runtime.hex | cut -c1-16)"

# 1. Bytecode level: same abstract storage for both codes, symbolic calldata restricted to the
#    selector and ABI-encoded symbolic arguments.
EQ=(hevm equivalence --code-a-file develop.runtime.hex --code-b-file current.runtime.hex --solver "$SOLVER" \
  --smt-timeout "$SMT_TIMEOUT" --num-solvers "$NUM_SOLVERS")
for sig in "successfulMessages(bytes32)" "sentMessages(uint256)" "messageNonce()" "messageVersion()" \
  "crossDomainMessageSender()" "crossDomainMessageSource()" "crossDomainMessageContext()"; do
  run_hevm PASS "hevm equivalence $sig" "$OUT/equiv-${sig%%(*}.log" "${EQ[@]}" --sig "$sig"
done
# Witness: version() changed (1.3.2 -> 2.0.0).
run_hevm WITNESS "hevm equivalence version() [witness: intentional change]" "$OUT/equiv-version.log" \
  "${EQ[@]}" --sig "version()"
# Witness: the whole of sendMessage differs (E1 reverts and E2's timestamp write both separate
# the codes). hevm explores it only partially, which is why the harness exists; the targets of the
# counterexamples it does report are recorded, not asserted.
t0=$(date +%s)
capped "${EQ[@]}" --sig "sendMessage(uint256,address,bytes memory)" >"$OUT/equiv-sendMessage.log" 2>&1
txt=$(strip "$OUT/equiv-sendMessage.log")
targets=$(grep -oE 'SymAddr "arg2": 0x[0-9a-fA-F]+' "$txt" | awk '{print $3}' | sort -u | tr '\n' ' ')
if grep -q "^Not equivalent" "$txt"; then got=WITNESS; else got=NO-CEX; fi
record WITNESS "$got" "$(($(date +%s) - t0))" "hevm equivalence sendMessage (partial; cex targets: ${targets:-none})"

# Log-blindness witness: two contracts that differ ONLY in emitted event data are "equivalent".
TMP="$(mktemp -d)"
cat >"$TMP/A.sol" <<'EOF'
// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;
contract A { event E(uint256 x); uint256 s; function f(uint256 x) external { s = x; emit E(x); } }
EOF
sed 's/emit E(x)/emit E(x ^ 1)/' "$TMP/A.sol" >"$TMP/B.sol"
solc --bin-runtime "$TMP/A.sol" 2>/dev/null | tail -1 >"$TMP/a.hex"
solc --bin-runtime "$TMP/B.sol" 2>/dev/null | tail -1 >"$TMP/b.hex"
run_hevm PASS "log-blindness: contracts differing only in event data are 'equivalent'" "$OUT/log-blindness.log" \
  hevm equivalence --code-a-file "$TMP/a.hex" --code-b-file "$TMP/b.hex" --solver "$SOLVER"
rm -rf "$TMP"

# Build (clean, so the engines only see current artifacts; no mutants yet).
rm -rf out cache mutants
forge build >"$OUT/forge-build.log" 2>&1 || { note "forge build FAILED"; exit 1; }

# 2. hevm.
while IFS= read -r fn; do
  want=PASS
  case "$fn" in prove_nonvacuity_*) want=WITNESS ;; esac
  run_hevm "$want" "hevm test $fn" "$OUT/$fn.log" \
    hevm test --root . --match "^${fn}\$" --solver "$SOLVER" --smt-timeout "$SMT_TIMEOUT" --num-solvers "$NUM_SOLVERS"
done < <(grep -oE 'function prove_[A-Za-z0-9_]+' "$HARNESS" | cut -d' ' -f2 | grep -E "$MATCH")

# 3. Halmos.
while IFS= read -r fn; do
  want=PASS
  case "$fn" in check_nonvacuity_*) want=WITNESS ;; esac
  run_halmos "$want" "$HALMOS_CONTRACT" "$fn" "$OUT/halmos-$fn"
done < <(grep -oE 'function check_[A-Za-z0-9_]+' "$HARNESS" | cut -d' ' -f2 | grep -E "$MATCH")

# 4. Mutants. Each patches the current source, is compiled here (optimizer off), and replaces NEW;
#    each check in the third field must then find a counterexample. M0 is the unpatched source
#    compiled the same way: it must pass every one of those checks. The optional fourth field lists
#    checks that are KNOWN BLIND to the mutant and must PASS: Halmos 0.3.3's generic storage keeps
#    keccak-derived slots apart from raw slots, so S-all (check_allSlots_*) cannot see a changed
#    mapping entry; M5/M6 record that limitation and that S-map catches those mutants. The generated files live in mutants/
#    only for the duration of this step (they must never reach the repo build).
if [ -z "${SKIP_MUTANTS:-}" ]; then
  trap 'rm -rf "$HERE/mutants"' EXIT
  mkdir -p mutants
  SRC="$HERE/../../../../src/L2/L2ToL2CrossDomainMessenger.sol"
  MUTANTS=(
    "M0|identity|check_allSlots_relayMessage_len37 check_allSlots_sendMessage_len37 check_relayMessage_len37 check_sendMessage_len37"
    "M1|relay writes a stray slot (sstore(5, 1))|check_allSlots_relayMessage_len37"
    "M2|send writes a stray slot (sstore(5, 1))|check_allSlots_sendMessage_len37"
    "M3|relay passes another hash to the inbox|check_relayMessage_len37"
    "M4|relay forwards no ETH to the target|check_relayMessage_len37"
    "M5|relay does not set successfulMessages[H]|check_relayMessage_len37|check_allSlots_relayMessage_len37"
    "M6|send does not set sentMessages[nonce]|check_sendMessage_len37|check_allSlots_sendMessage_len37"
  )
  python3 -I - "$SRC" mutants <<'PY' || { note "mutant generation FAILED"; exit 1; }
import sys
src, out = sys.argv[1], sys.argv[2]
s = open(src).read()
patches = {
    "M0": [],
    "M1": [("successfulMessages[messageHash] = true;",
            "successfulMessages[messageHash] = true;\n        assembly { sstore(5, 1) }")],
    "M2": [("sentMessageTimestamps[messageHash_] = block.timestamp;",
            "sentMessageTimestamps[messageHash_] = block.timestamp;\n        assembly { sstore(5, 1) }")],
    "M3": [("validateMessage(_id, keccak256(_sentMessage));",
            "validateMessage(_id, bytes32(uint256(keccak256(_sentMessage)) ^ 1));")],
    "M4": [("target.call{ value: msg.value }(message)", "target.call{ value: 0 }(message)")],
    "M5": [("successfulMessages[messageHash] = true;", "")],
    "M6": [("sentMessages[nonce] = messageHash_;", "")],
}
for name, ps in patches.items():
    m = s.replace("contract L2ToL2CrossDomainMessenger is", f"contract L2ToL2CrossDomainMessenger{name} is")
    for old, new in ps:
        assert m.count(old) == 1, (name, old)
        m = m.replace(old, new)
    open(f"{out}/{name}.sol", "w").write(m)
    open(f"{out}/{name}.t.sol", "w").write(f"""// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import {{ L2ToL2CrossDomainMessenger_EquivalenceHalmos }} from "../L2ToL2Equivalence.t.sol";
import {{ L2ToL2CrossDomainMessenger{name} }} from "./{name}.sol";

contract {name}_EquivalenceHalmos is L2ToL2CrossDomainMessenger_EquivalenceHalmos {{
    function _newCode() internal pure override returns (bytes memory code_) {{
        code_ = type(L2ToL2CrossDomainMessenger{name}).runtimeCode;
    }}
}}
""")
PY
  forge build >"$OUT/forge-build-mutants.log" 2>&1 || { note "mutant build FAILED"; exit 1; }
  for entry in "${MUTANTS[@]}"; do
    IFS='|' read -r name what checks blind <<<"$entry"
    read -r -a check_list <<<"$checks"
    read -r -a blind_list <<<"${blind:-}"
    want=WITNESS
    [ "$name" = M0 ] && want=PASS
    for fn in "${check_list[@]}"; do
      run_halmos "$want" "${name}_EquivalenceHalmos" "$fn" "$OUT/mutant-$name-$fn"
    done
    for fn in "${blind_list[@]}"; do
      run_halmos PASS "${name}_EquivalenceHalmos" "$fn" "$OUT/mutant-$name-$fn-blind"
      note "            (known blind spot: $fn does not see $name)"
    done
    note "            ($name: $what)"
  done
  rm -rf mutants
  trap - EXIT
fi

# 5. Concrete tests: the event fuzz and the witness replays (the mutants are gone by now).
t0=$(date +%s)
capped forge test --root . --match-path L2ToL2EquivalenceConcrete.t.sol --fuzz-runs "$FUZZ_RUNS" \
  >"$OUT/concrete.log" 2>&1
rc=$?
passed=$(grep -c "^\[PASS\]" "$OUT/concrete.log")
if [ $rc = 0 ] && grep -q "Suite result: ok" "$OUT/concrete.log" && ! grep -q "\[FAIL" "$OUT/concrete.log"; then
  got="PASS"
else
  got="FAIL(rc=$rc)"
fi
record PASS "$got" "$(($(date +%s) - t0))" "forge concrete tests: event fuzz ($FUZZ_RUNS runs) + witness replays ($passed passed)"

note "overall: $([ $status -eq 0 ] && echo 'all as expected' || echo 'UNEXPECTED results')"
exit $status
