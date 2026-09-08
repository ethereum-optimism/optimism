"""Reject incomplete step proofs before native dependency reuse."""
import json
from pathlib import Path
import sys

from pyk.kast.prelude.ml import is_bottom, is_top
from pyk.proof.reachability import APRProof

root = Path(sys.argv[1])
expected = {"WITHDRAWAL-COPY-LOOP.word-copy-append-step"}
metadata = list(root.rglob("proof.json"))
if len(metadata) != 1 or {path.parent.name for path in metadata} != expected:
    raise SystemExit("Expected exactly one independent copy-step graph")
for path in metadata:
    record = json.loads(path.read_text())
    proof = APRProof.read_proof_data(root, path.parent.name)
    if (
        record.get("type") != "APRProof" or proof.id != path.parent.name or not proof.passed or proof.admitted
        or proof.pending or proof.failing or proof.subproof_ids
        or record.get("circularity") is not False or record.get("bounded") != []
        or proof.bmc_depth is not None
        or record.get("node_refutations") != {}
        or proof.kcfg.vacuous or proof.kcfg.stuck
        or not proof.kcfg.edges() or not proof.kcfg.covers()
    ):
        raise SystemExit(f"Incomplete independent step: {proof.id}")
    for node in (proof.init, proof.target):
        term = proof.kcfg.node(node).cterm.kast
        if is_bottom(term, weak=True) or is_top(term, weak=True):
            raise SystemExit(f"Degenerate step endpoint: {proof.id}:{node}")
print("Verified the completed independent copy-step graph")
