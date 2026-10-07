#!/usr/bin/env bash
# Runs the Halmos expiry checks and verifies the results against the explicit inventory in expected.tsv
# (contract, check, expected outcome):
#   - every listed check must be present in halmos's results (missing, extra or empty contract results fail);
#   - PASS checks must pass with no loop cut short by --loop;
#   - FAIL checks (check_FALSE_*, check_INFO_*, *_PENDING) must fail with a counterexample halmos validated;
#   - each halmos process must exit 0 or 1 (1 = some check failed, expected here) and write its JSON.
# Exit status is nonzero on any deviation.
#
#   HALMOS         halmos 0.3.3 WITH halmos-selfdestruct.patch applied (default: halmos). With stock halmos the
#                  refund checks ERROR (SafeSend uses SELFDESTRUCT), so the run fails loudly, never silently. Build:
#                    uv venv /tmp/halmos-sd --python 3.12
#                    uv pip install --link-mode copy --python /tmp/halmos-sd/bin/python halmos==0.3.3  # copy: never patch uv's cache
#                    patch -d /tmp/halmos-sd/lib/python3.12/site-packages -p1 < test/formal/expiry/halmos/halmos-selfdestruct.patch
#                    HALMOS=/tmp/halmos-sd/bin/halmos test/formal/expiry/halmos/run.sh
#   BYTES_LENGTHS  lengths tried for every `bytes` parameter (default 0,1,32,33,100,132,260).
#
# No foundry.toml profile is needed: the FOUNDRY_* variables restrict compilation to this directory and what it
# imports, and keep the build output (halmos-out, halmos-cache) separate from the main build.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$here/../../../.." # packages/contracts-bedrock

export FOUNDRY_SRC=test/formal/expiry/halmos
export FOUNDRY_TEST=test/formal/expiry/halmos
export FOUNDRY_SCRIPT=test/formal/expiry/halmos
export FOUNDRY_OUT=halmos-out
export FOUNDRY_CACHE_PATH=halmos-cache

HALMOS="${HALMOS:-halmos}"
out="$(mktemp -d)"
trap 'rm -rf "$out"' EXIT

forge clean
forge build >/dev/null

status=0
for contract in $(cut -f1 "$here/expected.tsv" | sort -u); do
  set +e
  "$HALMOS" --forge-build-out halmos-out --no-status \
    --default-bytes-lengths "${BYTES_LENGTHS:-0,1,32,33,100,132,260}" \
    --loop 2 --solver-timeout-assertion 60s \
    --match-contract "^${contract}\$" --json-output "$out/$contract.json" >"$out/$contract.log" 2>&1
  code=$?
  set -e
  if [ "$code" -ne 0 ] && [ "$code" -ne 1 ]; then
    echo "BAD $contract: halmos exited with $code (see log below)"
    tail -20 "$out/$contract.log"
    status=1
  fi
done

python3 - "$here/expected.tsv" "$out" <<'EOF' || status=1
import json, os, sys
expected_path, out = sys.argv[1], sys.argv[2]
expected = {}
for line in open(expected_path):
    if line.strip():
        c, n, e = line.rstrip("\n").split("\t")
        expected[(c, n)] = e
bad = 0
seen = set()
for c in sorted({c for c, _ in expected}):
    path = os.path.join(out, c + ".json")
    if not os.path.exists(path):
        print(f"BAD {c}: no JSON output"); bad += 1; continue
    results = json.load(open(path)).get("test_results", {})
    rows = [r for k, v in results.items() if k.split(":")[-1] == c for r in (v or [])]
    if not rows:
        print(f"BAD {c}: no results (setUp failure or no checks ran)"); bad += 1; continue
    for r in rows:
        name = r["name"].split("(")[0]
        key = (c, name)
        if key not in expected:
            print(f"BAD {c}.{name}: not in expected.tsv (exitcode={r['exitcode']})"); bad += 1; continue
        seen.add(key)
        want = expected[key]
        code, models, loops = r["exitcode"], r["num_models"] or 0, r["num_bounded_loops"] or 0
        if want == "PASS":
            ok = code == 0 and loops == 0
            got = "PASS" if ok else f"exitcode={code} bounded_loops={loops}"
        else:
            valid = any(m.get("is_valid") for m in (r["models"] or []))
            ok = code == 1 and models > 0 and valid
            got = "FAIL (expected; valid counterexample)" if ok else f"exitcode={code} models={models} valid={valid}"
        bad += not ok
        print(f"{'ok ' if ok else 'BAD'} {c}.{name}: {got} ({r['time'][0]:.2f}s, paths={r['num_paths'][0]})")
for key in sorted(set(expected) - seen):
    print(f"BAD {key[0]}.{key[1]}: expected {expected[key]} but it did not run"); bad += 1
print("all results as expected" if bad == 0 else f"{bad} unexpected result(s)")
sys.exit(1 if bad else 0)
EOF
exit $status
