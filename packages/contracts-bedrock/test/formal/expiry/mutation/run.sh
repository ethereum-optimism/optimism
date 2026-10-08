#!/usr/bin/env bash
# Cross-layer mutation campaign for per-message interop expiry (see README.md in this directory).
#
# For every mutant in mutants.tsv: restore src/, apply its sed edit, run the cheap verification layers on the
# mutated code, and record one verdict per layer in results/matrix.tsv. Before the mutants, the same layers run on
# the unmutated code (id K00) and must all pass, otherwise the script stops.
#
# Layers (LAYERS, comma-separated, default "unit,inv,halmos,hevm"):
#   unit    the PR's unit tests of the four contracts, FOUNDRY_PROFILE=liteci, fixed fuzz seed;
#   inv     ../invariants (FOUNDRY_PROFILE=liteci, its pinned runs x depth), seed SEED; if nothing fails, two more
#           seeds (SEED+1, SEED+2);
#   halmos  the PASS checks (from ../halmos/expected.tsv) of the touched contract's phase-1 Halmos contract; if none
#           fails, also its phase-2 reachability contract (REACH=always runs both every time);
#   hevm    the develop-vs-branch equivalence harness (../hevm, Halmos engine) with the mutated messenger as the new
#           code; only for L2ToL2CrossDomainMessenger mutants.
# Verdicts: CAUGHT (with the failing checks), SURVIVED, NA (layer does not apply), ERROR (the mutant does not compile
# or the layer did not run), INCONCLUSIVE (Halmos timeouts or stuck paths and no counterexample).
#
# Env:
#   FORGE    forge binary (default: forge; the repo pins 1.8.3)
#   HALMOS   halmos 0.3.3 with ../halmos/halmos-selfdestruct.patch applied (default: halmos)
#   WRAP     command prefix for every heavy command, e.g. a memory cap:
#              WRAP="systemd-run --user --scope -q -p MemoryMax=16G -p MemorySwapMax=0"
#   ONLY     regex over mutant ids (default: all); the baseline K00 always runs
#   SEED     first fuzz seed (default 1)
#   REACH    "always" to run the phase-2 Halmos contract even when phase 1 already caught the mutant, "never" to skip
#            phase 2 (the baseline runs it unless REACH=never or SKIP_BASE_REACH=1, for when the phase-2 baseline was
#            established by another run on the same commit)
#   HALMOS_TIMEOUT  seconds per Halmos process (default 2400)
#
# Runs IN PLACE in packages/contracts-bedrock: DeployUtils.getDeployedCode reads forge-artifacts/ on disk whatever
# FOUNDRY_OUT says, so a separate output directory would let a stale artifact through. Run one instance per
# checkout. src/ must be clean; it is restored (git checkout -- src) after every mutant and on exit, and
# forge-artifacts/, cache/ (including cache/invariant, where forge replays failing sequences) are cleaned before
# every mutant. As a canary, the touched contract's liteci runtime bytecode must differ from the baseline's.
set -uo pipefail
if [ "${BASH_VERSINFO[0]}" -lt 4 ] || { [ "${BASH_VERSINFO[0]}" -eq 4 ] && [ "${BASH_VERSINFO[1]}" -lt 4 ]; }; then
  echo "needs bash >= 4.4" >&2
  exit 1
fi

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)" || exit 1
cd "$here/../../../.." || exit 1 # packages/contracts-bedrock

FORGE="${FORGE:-forge}"
HALMOS="${HALMOS:-halmos}"
WRAP="${WRAP:-}"
ONLY="${ONLY:-.}"
SEED="${SEED:-1}"
REACH="${REACH:-}"
HALMOS_TIMEOUT="${HALMOS_TIMEOUT:-2400}"
LAYERS=",${LAYERS:-unit,inv,halmos,hevm},"
read -r -a wrap <<<"$WRAP"
for l in ${LAYERS//,/ }; do
  case "$l" in unit | inv | halmos | hevm) ;; *) echo "unknown layer: $l" >&2; exit 1 ;; esac
done
# Forge layers: the liteci profile, and no inherited FOUNDRY_* path overrides (DeployUtils.getDeployedCode reads
# forge-artifacts/ in any case, so the build must write there).
fenv=(env -u FOUNDRY_OUT -u FOUNDRY_CACHE_PATH -u FOUNDRY_SRC -u FOUNDRY_TEST -u FOUNDRY_SCRIPT FOUNDRY_PROFILE=liteci)
# The hevm harness is its own foundry project (../hevm/foundry.toml): no inherited overrides there either.
eenv=(env -u FOUNDRY_OUT -u FOUNDRY_CACHE_PATH -u FOUNDRY_SRC -u FOUNDRY_TEST -u FOUNDRY_SCRIPT -u FOUNDRY_PROFILE)

HDIR=test/formal/expiry/halmos
# The Halmos build: only the halmos directory and what it imports, default profile, its own out/ and cache/.
henv=(env -u FOUNDRY_PROFILE FOUNDRY_SRC="$HDIR" FOUNDRY_TEST="$HDIR" FOUNDRY_SCRIPT="$HDIR" FOUNDRY_OUT="$HDIR/out"
  FOUNDRY_CACHE_PATH="$HDIR/cache")
EDIR=test/formal/expiry/hevm
UNIT_PATHS="test/{L1/L1CrossDomainMessenger,L2/L2ToL2CrossDomainMessenger,L2/UndeliveredMessageExporter,L2/SuperchainETHBridge}.t.sol"
INV_PATHS="test/formal/expiry/invariants/*"
RES="$here/results"
MATRIX="$RES/matrix.tsv"
mkdir -p "$RES"
: >"$MATRIX"

if ! git diff --quiet HEAD -- src || [ -n "$(git status --porcelain --untracked-files=all -- src)" ]; then
  echo "src/ differs from HEAD (staged or unstaged); run on a clean checkout" >&2
  exit 1
fi
trap 'git checkout -q HEAD -- src' EXIT

record() { # record <id> <layer> <verdict> <seconds> <detail>
  printf '%s\t%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "$4" "$5" | tee -a "$MATRIX"
}

# Failing tests in a forge log, one "File.t.sol:Contract.test" per line.
# A failing test prints "[FAIL: reason] name(...)"; a failing invariant may print its name on a later line, after
# the call sequence, and the summary at the end prints "[FAIL: reason] name" without arguments.
forge_failures() {
  sed -e 's/\x1b\[[0-9;]*m//g' "$1" | awk '
    /^Ran [0-9]+ tests? for / || /^Encountered [0-9]+ failing tests? in / { suite = $NF; sub(/^.*\//, "", suite) }
    pending && /^[ \t]*(test|invariant|setUp)[A-Za-z0-9_]*\(/ {
      name = $0; sub(/^[ \t]*/, "", name); sub(/\(.*$/, "", name); print suite "." name; pending = 0; next }
    /^\[FAIL/ {
      rest = $0; sub(/^\[FAIL.*\] /, "", rest)
      if (rest ~ /^(test|invariant|setUp)[A-Za-z0-9_]*/) { sub(/[( ].*$/, "", rest); print suite "." rest } else pending = 1 }' |
    sort -u
}

# Short, comma-separated summary of a list of names (first 8, then a count).
summarize() {
  awk 'NR <= 8 { s = s (NR > 1 ? ", " : "") $0 } END { if (NR > 8) s = s ", ... (" NR " in all)"; print s }'
}

# Runtime bytecode hash of a contract's artifact in forge-artifacts/.
artifact_hash() {
  python3 -I - "forge-artifacts/$1.sol/$1.json" <<'PY'
import hashlib, json, sys
try:
    print(hashlib.sha256(json.load(open(sys.argv[1]))["deployedBytecode"]["object"].encode()).hexdigest()[:16])
except Exception:
    print("missing")
PY
}

# One forge test layer: run_forge <id> <layer> <paths> <seed>; prints the failures to stdout, or COMPILE-ERROR, or
# RUN-ERROR when forge did not finish normally (an exit status other than 0 or 1, e.g. killed by the memory cap, or
# no final "Ran N test suites" summary), so that a truncated log is never read as "nothing failed".
run_forge() {
  local log="$RES/$1.$2.seed$4.log" code
  "${wrap[@]}" "${fenv[@]}" "$FORGE" test --fuzz-seed "$4" --match-path "$3" >"$log" 2>&1
  code=$?
  if grep -q "Compiler run failed\|Error: Compilation failed\|^Error (" "$log"; then
    echo "COMPILE-ERROR"
    return
  fi
  if { [ "$code" -ne 0 ] && [ "$code" -ne 1 ]; } || ! grep -Eq "Ran [0-9]+ test suites? " "$log"; then
    echo "RUN-ERROR"
    return
  fi
  forge_failures "$log"
}

layer_unit() { # layer_unit <id> <contract name>
  local t0 fails
  t0=$(date +%s)
  fails="$(run_forge "$1" unit "$UNIT_PATHS" "$SEED")"
  local secs=$(($(date +%s) - t0)) canary="" h
  if [ -n "$2" ] && [ "$1" != K00 ]; then
    h="$(artifact_hash "$2")"
    if [ "$h" = "$(cat "$RES/base-$2.sha" 2>/dev/null)" ] || [ "$h" = missing ]; then
      canary=" [canary: $2 liteci bytecode $h unchanged vs baseline]"
    fi
  fi
  if [ "$fails" = COMPILE-ERROR ] || [ "$fails" = RUN-ERROR ]; then
    record "$1" unit ERROR "$secs" "$fails"
  elif [ -n "$canary" ]; then
    record "$1" unit ERROR "$secs" "stale or unchanged artifact:$canary"
  elif [ -n "$fails" ]; then
    record "$1" unit CAUGHT "$secs" "$(echo "$fails" | summarize)$canary"
  else
    record "$1" unit SURVIVED "$secs" "${canary# }"
  fi
}

layer_inv() { # layer_inv <id>
  local t0 fails s
  t0=$(date +%s)
  for s in "$SEED" $((SEED + 1)) $((SEED + 2)); do
    rm -rf cache/invariant
    fails="$(run_forge "$1" inv "$INV_PATHS" "$s")"
    [ -n "$fails" ] && break
    [ "$1" = K00 ] && break # the baseline runs one seed
  done
  local secs=$(($(date +%s) - t0))
  if [ "$fails" = COMPILE-ERROR ] || [ "$fails" = RUN-ERROR ]; then
    record "$1" inv ERROR "$secs" "seed $s: $fails"
  elif [ -n "$fails" ]; then
    record "$1" inv CAUGHT "$secs" "seed $s: $(echo "$fails" | summarize)"
  else
    record "$1" inv SURVIVED "$secs" "seeds $SEED..$s"
  fi
}

# Verdict of the PASS checks of one Halmos JSON result: "CAUGHT names", "INCONCLUSIVE names", "SURVIVED" or "ERROR".
halmos_verdict() { # halmos_verdict <json> <expected check names...>
  python3 -I - "$@" <<'PY'
import json, sys
path, checks = sys.argv[1], set(sys.argv[2:])
if not checks:
    print("ERROR no expected checks"); sys.exit()
try:
    rows = [r for v in json.load(open(path))["test_results"].values() for r in (v or [])]
except Exception as e:
    print(f"ERROR no JSON ({e})"); sys.exit()
by = {r["name"].split("(")[0]: r for r in rows}
missing = sorted(checks - set(by))
caught, odd = [], []
for c in sorted(checks & set(by)):
    r = by[c]
    models = r.get("models") or []
    valid = any(m.get("is_valid") or (m.get("model") or {}).get("is_valid") for m in models)
    # A kill needs a counterexample Halmos validated; anything else is inconclusive.
    if r["exitcode"] == 1 and valid:
        caught.append(c)
    elif r["exitcode"] == 1 and models:
        odd.append(f"{c}(model not validated)")
    elif r["exitcode"] != 0 or (r.get("num_bounded_loops") or 0):
        odd.append(f"{c}(exit {r['exitcode']})")
if caught:
    print("CAUGHT " + ", ".join(caught))
elif odd or missing:
    print("INCONCLUSIVE " + ", ".join(odd + [m + "(missing)" for m in missing]))
else:
    print("SURVIVED")
PY
}

run_halmos_contract() { # run_halmos_contract <id> <contract>; prints the verdict line
  local -a checks cfg
  mapfile -t checks < <(awk -F'\t' -v c="$2" '$1 == c && $3 == "PASS" { print $2 }' "$HDIR/expected.tsv")
  if [ "${#checks[@]}" -eq 0 ]; then
    echo "ERROR no expected-PASS checks for $2 in expected.tsv"
    return
  fi
  local re
  re="$(printf '%s\n' "${checks[@]}" | sed 's/^check_//' | paste -sd '|' -)"
  if [ -f "$HDIR/halmos.toml" ]; then
    cfg=(--config "$HDIR/halmos.toml")
  else
    cfg=(--default-bytes-lengths "0,1,32,33,100,132,260" --solver-timeout-assertion 60s)
  fi
  local json="$RES/$1.halmos.$2.json"
  rm -f "$json"
  "${wrap[@]}" "${henv[@]}" timeout "$HALMOS_TIMEOUT" "$HALMOS" --forge-build-out "$HDIR/out" --no-status "${cfg[@]}" \
    --match-contract "^$2\$" --match-test "^check_($re)\\(" --json-output "$json" \
    >"$RES/$1.halmos.$2.log" 2>&1 </dev/null
  halmos_verdict "$json" "${checks[@]}"
}

layer_halmos() { # layer_halmos <id> <phase-1 contract> <phase-2 contract>
  local t0 v1 v2="" verdict detail
  t0=$(date +%s)
  if ! "${wrap[@]}" "${henv[@]}" "$FORGE" build --force >"$RES/$1.halmos.build.log" 2>&1; then
    record "$1" halmos ERROR "$(($(date +%s) - t0))" "Halmos build failed"
    return
  fi
  v1="$(run_halmos_contract "$1" "$2")"
  if [ "$REACH" = never ] || { [ "$1" = K00 ] && [ -n "${SKIP_BASE_REACH:-}" ]; }; then
    :
  elif [ "${v1%% *}" != CAUGHT ] || [ "$REACH" = always ] || [ "$1" = K00 ]; then
    v2="$(run_halmos_contract "$1" "$3")"
  fi
  verdict="${v1%% *}"
  detail="$2: $v1"
  [ -n "$v2" ] && detail="$detail; $3: $v2"
  if [ "$verdict" != CAUGHT ] && [ "${v2%% *}" = CAUGHT ]; then
    verdict=CAUGHT
  elif [ "$verdict" = SURVIVED ] && [ -n "$v2" ] && [ "${v2%% *}" != SURVIVED ]; then
    verdict="${v2%% *}"
  fi
  record "$1" halmos "$verdict" "$(($(date +%s) - t0))" "$detail"
}

layer_hevm() { # layer_hevm <id>: the mutated messenger replaces NEW in the equivalence harness
  local t0
  t0=$(date +%s)
  local m="X$1"
  rm -rf "$EDIR/mutants"
  mkdir -p "$EDIR/mutants"
  sed "s/^contract L2ToL2CrossDomainMessenger is/contract L2ToL2CrossDomainMessenger$m is/" \
    src/L2/L2ToL2CrossDomainMessenger.sol >"$EDIR/mutants/$m.sol"
  cat >"$EDIR/mutants/$m.t.sol" <<EOF
// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { L2ToL2CrossDomainMessenger_EquivalenceHalmos } from "../L2ToL2Equivalence.t.sol";
import { L2ToL2CrossDomainMessenger$m } from "./$m.sol";

contract ${m}_EquivalenceHalmos is L2ToL2CrossDomainMessenger_EquivalenceHalmos {
    function _newCode() internal pure override returns (bytes memory code_) {
        code_ = type(L2ToL2CrossDomainMessenger$m).runtimeCode;
    }
}
EOF
  if ! (cd "$EDIR" && "${wrap[@]}" "${eenv[@]}" "$FORGE" build >"$RES/$1.hevm.build.log" 2>&1); then
    record "$1" hevm ERROR "$(($(date +%s) - t0))" "harness build failed"
    rm -rf "$EDIR/mutants"
    return
  fi
  local -a solid generic
  mapfile -t solid < <(grep -oE 'function check_(sendMessage|relayMessage)[A-Za-z0-9_]*' "$EDIR/L2ToL2Equivalence.t.sol" |
    cut -d' ' -f2)
  mapfile -t generic < <(grep -oE 'function check_allSlots_[A-Za-z0-9_]*' "$EDIR/L2ToL2Equivalence.t.sol" | cut -d' ' -f2)
  local v=() layout fns re
  for layout in solidity generic; do
    if [ "$layout" = solidity ]; then fns=("${solid[@]}"); else fns=("${generic[@]}"); fi
    re="$(printf '%s\n' "${fns[@]}" | paste -sd '|' -)"
    rm -f "$RES/$1.hevm.$layout.json"
    (cd "$EDIR" && "${wrap[@]}" "${eenv[@]}" timeout "$HALMOS_TIMEOUT" "$HALMOS" --root . --contract "${m}_EquivalenceHalmos" \
      --function "^($re)\\(" --storage-layout "$layout" --solver-timeout-assertion 0 \
      --json-output "$RES/$1.hevm.$layout.json" >"$RES/$1.hevm.$layout.log" 2>&1 </dev/null)
    v+=("$(halmos_verdict "$RES/$1.hevm.$layout.json" "${fns[@]}")")
  done
  rm -rf "$EDIR/mutants"
  local verdict=SURVIVED
  case "${v[0]%% *} ${v[1]%% *}" in
    *CAUGHT*) verdict=CAUGHT ;;
    *ERROR*) verdict=ERROR ;;
    *INCONCLUSIVE*) verdict=INCONCLUSIVE ;;
  esac
  record "$1" hevm "$verdict" "$(($(date +%s) - t0))" "S-map: ${v[0]}; S-all: ${v[1]}"
}

run_mutant() { # run_mutant <id> <file> <sed expression>
  local id=$1 file=$2 expr=$3 name="" p1="" p2=""
  if ! git checkout -q HEAD -- src; then
    echo "git checkout of src/ failed; stopping" >&2
    exit 1
  fi
  if ! "${wrap[@]}" "${fenv[@]}" "$FORGE" clean >/dev/null 2>&1; then
    record "$id" all ERROR 0 "forge clean failed"
    return
  fi
  rm -rf cache/invariant "$EDIR/out" "$EDIR/cache"
  if [ -e forge-artifacts ] || [ -e cache/invariant ]; then
    record "$id" all ERROR 0 "forge clean did not remove forge-artifacts/ or cache/invariant"
    return
  fi
  if [ "$id" != K00 ]; then
    sed -i.mut -e "$expr" "$file" && rm -f "$file.mut"
    if git diff --quiet -- "$file"; then
      record "$id" all ERROR 0 "sed did not apply"
      return
    fi
  fi
  case "$file" in
    */L2ToL2CrossDomainMessenger.sol) name=L2ToL2CrossDomainMessenger p1=L2ToL2ExpiryHalmos p2=ReachL2ToL2Halmos ;;
    */UndeliveredMessageExporter.sol) name=UndeliveredMessageExporter p1=ExporterExpiryHalmos p2=ReachExporterHalmos ;;
    */L1CrossDomainMessenger.sol) name=L1CrossDomainMessenger p1=L1CDMExpiryHalmos p2=ReachL1CDMHalmos ;;
    */SuperchainETHBridge.sol) name=SuperchainETHBridge p1=RefundExpiryHalmos p2=ReachBridgeHalmos ;;
  esac
  if [[ $LAYERS == *,unit,* ]]; then
    layer_unit "$id" "$name"
    if [ "$(tail -n 1 "$MATRIX" | cut -f3,5 | grep -c 'stale or unchanged artifact')" -ne 0 ]; then
      git checkout -q HEAD -- src
      return # the other layers would read the same suspect artifacts
    fi
  fi
  [[ $LAYERS == *,inv,* ]] && layer_inv "$id"
  [[ $LAYERS == *,halmos,* ]] && [ -n "$p1" ] && layer_halmos "$id" "$p1" "$p2"
  if [[ $LAYERS == *,hevm,* ]]; then
    if [ "$name" = L2ToL2CrossDomainMessenger ] || [ "$id" = K00 ]; then
      layer_hevm "$id"
    else
      record "$id" hevm NA 0 "the equivalence harness covers only L2ToL2CrossDomainMessenger"
    fi
  fi
  git checkout -q HEAD -- src
}

# Baseline: every layer must pass on the unmutated code. The Halmos groups and the hevm layer run for the contracts
# that the selected mutants touch.
selected_files="$(awk -F'\t' -v only="$ONLY" '$1 !~ /^#/ && $1 ~ only { print $2 }' "$here/mutants.tsv" | sort -u)"
: >"$MATRIX.base"
MATRIX_SAVED="$MATRIX"
MATRIX="$MATRIX.base"
base_layers="$LAYERS"
[[ $selected_files == *L2ToL2CrossDomainMessenger.sol* ]] || LAYERS="${LAYERS//,hevm,/,}"
run_mutant K00 none ""
LAYERS="$base_layers"
for c in L2ToL2CrossDomainMessenger UndeliveredMessageExporter L1CrossDomainMessenger SuperchainETHBridge; do
  artifact_hash "$c" >"$RES/base-$c.sha"
done
if [[ $LAYERS == *,halmos,* ]]; then
  for triple in L2ToL2CrossDomainMessenger:L2ToL2ExpiryHalmos:ReachL2ToL2Halmos \
    UndeliveredMessageExporter:ExporterExpiryHalmos:ReachExporterHalmos \
    L1CrossDomainMessenger:L1CDMExpiryHalmos:ReachL1CDMHalmos SuperchainETHBridge:RefundExpiryHalmos:ReachBridgeHalmos; do
    IFS=: read -r c p1 p2 <<<"$triple"
    [[ $selected_files == *"/$c.sol"* ]] && layer_halmos K00 "$p1" "$p2"
  done
fi
MATRIX="$MATRIX_SAVED"
if awk -F'\t' '$3 != "SURVIVED" && $3 != "NA" { bad = 1 } END { exit !bad }' "$MATRIX.base"; then
  echo "baseline (unmutated) does not pass every layer; see $MATRIX.base" >&2
  exit 1
fi
echo "baseline passes every layer"

while IFS=$'\t' read -r id file _origin _what expr; do
  case "$id" in '#'* | '') continue ;; esac
  [[ $id =~ $ONLY ]] || continue
  run_mutant "$id" "$file" "$expr"
done <"$here/mutants.tsv"
"${wrap[@]}" "${fenv[@]}" "$FORGE" clean >/dev/null 2>&1
echo "done: $MATRIX"
