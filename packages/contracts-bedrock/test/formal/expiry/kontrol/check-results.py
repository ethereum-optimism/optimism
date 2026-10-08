"""Checks Kontrol results against witnesses.tsv; exits non-zero on any mismatch.

Usage: python3 check-results.py <kontrol list output file> (run from packages/contracts-bedrock).
- every prove_* in the .k.sol files is listed in witnesses.tsv (as a proof or a witness);
- every proof PASSED (latest version), not admitted, with no pending/failing/stuck nodes;
- every witness FAILED with at least one failing node and no stuck nodes (a counterexample leaf);
  pending nodes are allowed there, because Kontrol stops a failed proof early (fail-fast).
It fails closed: an APRProof entry it cannot parse, or a missing or non-numeric node count, is an
error. It does not check which assertion a witness's failing leaf reached, nor that the results
come from the current sources (run with KONTROL_FRESH=1 for that).
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

errors = []
declared = set()
for sol in HERE.glob("solc*/*.k.sol"):
    text = sol.read_text()
    contracts = re.findall(r"^contract (\w+)", text, re.M)
    if len(contracts) != 1:
        errors.append(f"{sol.name}: expected exactly one contract, found {contracts}")
        continue
    for fn in re.findall(r"function\s+(prove_\w+)\s*\(", text):
        declared.add(f"{contracts[0]}.{fn}")

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
    if not block.startswith("APRProof: "):
        continue
    m = re.match(r"APRProof: \S*%(\w+)\.(\w+)\([^)]*\):(\d+)\s*$", block.splitlines()[0])
    if not m:
        errors.append(f"unparseable APRProof header: {block.splitlines()[0]!r}")
        continue
    name, version = f"{m.group(1)}.{m.group(2)}", int(m.group(3))
    stats = dict(re.findall(r"^\s+(\w+): (\S+)", block, re.M))
    if name not in latest or version > latest[name][0]:
        latest[name] = (version, stats)

if not latest:
    errors.append("no APRProof entries in the `kontrol list` output (did `kontrol list` fail?)")


def counts(stats):
    """The pending/failing/stuck node counts as ints, or None if any is missing or not a number."""
    try:
        c = {k: int(stats[k]) for k in ("pending", "failing", "stuck")}
    except (KeyError, ValueError):
        return None
    return c if all(v >= 0 for v in c.values()) else None


for name in sorted(proofs):
    stats = latest.get(name, (0, None))[1]
    if stats is None:
        errors.append(f"{name}: no result")
        continue
    c = counts(stats)
    if (
        stats.get("status") != "ProofStatus.PASSED"
        or stats.get("admitted") != "False"
        or c is None
        or c != {"pending": 0, "failing": 0, "stuck": 0}
    ):
        errors.append(f"{name}: expected PASSED, got {stats.get('status')} (admitted {stats.get('admitted')}, {c})")
for name in sorted(witnesses):
    stats = latest.get(name, (0, None))[1]
    if stats is None:
        errors.append(f"{name}: no result")
        continue
    c = counts(stats)
    if stats.get("status") != "ProofStatus.FAILED" or c is None or c["failing"] < 1 or c["stuck"] != 0:
        errors.append(f"{name}: expected a counterexample leaf, got {stats.get('status')} ({c})")

for e in errors:
    print("FAIL", e)
print(f"{len(proofs)} proofs, {len(witnesses)} witnesses, {len(errors)} problems")
sys.exit(1 if errors else 0)
