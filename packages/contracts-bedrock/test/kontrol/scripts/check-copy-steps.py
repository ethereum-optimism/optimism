"""Reject incomplete copy proofs before native dependency reuse."""
import json
from pathlib import Path
import sys

from pyk.kast.prelude.ml import is_bottom, is_top
from pyk.proof.reachability import APRProof

root = Path(sys.argv[1])
prefix = "WITHDRAWAL-COPY-LOOP.word-copy-append"
dependencies = {prefix + "-step": set(), prefix + "-positive": {prefix + "-step"},
                prefix: {prefix + "-step", prefix + "-positive"}}
phase = int(sys.argv[2])
if phase not in (1, 2, 3):
    raise SystemExit("Expected proof phase 1, 2 or 3")
expected = set(list(dependencies)[:phase])
metadata = list(root.rglob("proof.json"))
if len(metadata) != phase or {path.parent.name for path in metadata} != expected:
    raise SystemExit(f"Unexpected copy graphs for phase {phase}")
for path in metadata:
    record = json.loads(path.read_text())
    proof = APRProof.read_proof_data(root, path.parent.name)
    if (
        record.get("type") != "APRProof" or proof.id != path.parent.name or not proof.passed or proof.admitted
        or proof.pending or proof.failing or set(proof.subproof_ids) != dependencies[proof.id]
        or record.get("circularity") is not (proof.id == prefix + "-positive") or record.get("bounded") != []
        or proof.bmc_depth is not None
        or record.get("node_refutations") != {}
        or proof.kcfg.vacuous or proof.kcfg.stuck
        or not proof.kcfg.edges() or not proof.kcfg.covers()
    ):
        raise SystemExit(f"Incomplete copy proof: {proof.id}")
    for node in (proof.init, proof.target):
        term = proof.kcfg.node(node).cterm.kast
        if is_bottom(term, weak=True) or is_top(term, weak=True):
            raise SystemExit(f"Degenerate copy endpoint: {proof.id}:{node}")
print(f"Verified all {phase} completed copy graphs and their dependencies")
