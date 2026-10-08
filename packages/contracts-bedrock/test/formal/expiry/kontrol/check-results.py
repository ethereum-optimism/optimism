"""Checks Kontrol results against witnesses.tsv; exits non-zero on any mismatch.

Usage: python3 check-results.py <kontrol list output file> (run from packages/contracts-bedrock).
- every prove_* in the .k.sol files is listed in witnesses.tsv (as a proof or a witness);
- every proof PASSED (latest version), with no pending/failing/stuck nodes;
- every witness FAILED with a failing node and no stuck nodes (a genuine counterexample).
"""
import pathlib
import re
import sys

HERE = pathlib.Path(__file__).resolve().parent
pairs, witnesses, proofs = [], set(), set()
for line in (HERE / "witnesses.tsv").read_text().splitlines():
    if not line.strip() or line.startswith("#"):
        continue
    proof, witness = line.split("\t")
    proofs.add(proof)
    if witness != "-":
        witnesses.add(witness)
    pairs.append((proof, witness))

declared = set()
for sol in HERE.glob("solc*/*.k.sol"):
    contract = re.search(r"^contract (\w+)", sol.read_text(), re.M).group(1)
    for fn in re.findall(r"function (prove_\w+)\(", sol.read_text()):
        declared.add(f"{contract}.{fn}")

errors = []
for name in sorted(declared - proofs - witnesses):
    errors.append(f"{name}: not listed in witnesses.tsv")
for name in sorted((proofs | witnesses) - declared):
    errors.append(f"{name}: listed in witnesses.tsv but not declared")
for name in sorted(proofs & witnesses):
    errors.append(f"{name}: listed both as a proof and as a witness")

# Parse `kontrol list`: keep the highest version of each test.
latest = {}
blocks = re.split(r"\n(?=APRProof: )", pathlib.Path(sys.argv[1]).read_text())
for block in blocks:
    m = re.match(r"APRProof: \S*%(\w+)\.(\w+)\([^)]*\):(\d+)", block)
    if not m:
        continue
    name, version = f"{m.group(1)}.{m.group(2)}", int(m.group(3))
    stats = dict(re.findall(r"^\s+(\w+): (\S+)", block, re.M))
    if name not in latest or version > latest[name][0]:
        latest[name] = (version, stats)

if not latest:
    errors.append("no APRProof entries in the `kontrol list` output (did `kontrol list` fail?)")

for name in sorted(proofs):
    stats = latest.get(name, (0, None))[1]
    if stats is None:
        errors.append(f"{name}: no result")
    elif stats.get("status") != "ProofStatus.PASSED" or stats.get("stuck") != "0":
        errors.append(f"{name}: expected PASSED, got {stats.get('status')} (stuck {stats.get('stuck')})")
for name in sorted(witnesses):
    stats = latest.get(name, (0, None))[1]
    if stats is None:
        errors.append(f"{name}: no result")
    elif stats.get("status") != "ProofStatus.FAILED" or stats.get("failing") == "0" or stats.get("stuck") != "0":
        errors.append(
            f"{name}: expected a genuine counterexample, got {stats.get('status')} "
            f"(failing {stats.get('failing')}, stuck {stats.get('stuck')})"
        )

for e in errors:
    print("FAIL", e)
print(f"{len(proofs)} proofs, {len(witnesses)} witnesses, {len(errors)} problems")
sys.exit(1 if errors else 0)
