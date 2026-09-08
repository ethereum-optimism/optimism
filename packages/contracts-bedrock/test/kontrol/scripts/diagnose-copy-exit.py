"""Inspect an exit state; --query asks the backend in CI without changing the proof."""
import argparse
import hashlib
import json
from pathlib import Path
import shlex

from pyk.kast.inner import KApply, KLabel, KSequence, KVariable
from pyk.kast.prelude.k import GENERATED_TOP_CELL, K_ITEM
from pyk.proof.reachability import APRProof

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("proof_dir", type=Path)
parser.add_argument("definition", type=Path)
parser.add_argument("output", type=Path)
parser.add_argument("--query", action="store_true")
parser.add_argument("--rpc-command", default="kore-rpc-booster")
args = parser.parse_args()
proof = APRProof.read_proof_data(args.proof_dir, "WITHDRAWAL-COPY-LOOP.word-copy")
target = proof.kcfg.node(proof.target).cterm
exits = []
for node in proof.kcfg.nodes:
    state = node.cterm
    k = state.cell("K_CELL")
    if (node.id != proof.target and state.cell("PC_CELL") == target.cell("PC_CELL")
            and state.cell("PROGRAM_CELL") == target.cell("PROGRAM_CELL")
            and isinstance(k, KSequence) and k.items
            and isinstance(k.items[0], KApply) and k.items[0].label.name == "execute"):
        exits.append(node)
if not exits:
    raise SystemExit("No saved plain-execute exit state; no implication queried")
source = min(exits, key=lambda node: node.id)


def digest(state):
    return hashlib.sha256(json.dumps(state.to_dict(), sort_keys=True).encode()).hexdigest()


report = {"diagnosticOnly": True, "proof": proof.id, "source": source.id, "target": proof.target,
          "sourceSha256": digest(source.cterm), "targetSha256": digest(target), "queried": args.query,
          "nativeStatus": proof.status.name, "admitted": proof.admitted,
          "pending": [node.id for node in proof.pending], "failing": [node.id for node in proof.failing],
          "subproofIds": proof.subproof_ids, "queryCompleted": False}
args.output.write_text(json.dumps(report, indent=2) + "\n")
if args.query:
    from kevm_pyk.kevm import KEVM, KEVMSemantics
    from kevm_pyk.utils import legacy_explore

    kevm = KEVM(args.definition)
    command = tuple(shlex.split(args.rpc_command)) + ("--equation-max-local-steps", "20")
    with legacy_explore(kevm, kcfg_semantics=KEVMSemantics(), id=proof.id,
                        kore_rpc_command=command,
                        llvm_definition_dir=args.definition / "llvm-library",
                        smt_timeout=16000, smt_retry_limit=0,
                        log_succ_rewrites=False, log_fail_rewrites=False) as explore:
        symbolic = explore.cterm_symbolic
        result = symbolic.implies(source.cterm, target, failure_reason=True, assume_defined=False)
        report["implicationValid"] = result.csubst is not None
        report["queryCompleted"] = True
        report["substitution"] = str(result.csubst) if result.csubst is not None else None
        report["failingCells"] = [{"cell": name, "difference": term.to_dict()} for name, term in result.failing_cells]
        report["remainingImplication"] = (result.remaining_implication.to_dict()
                                          if result.remaining_implication is not None else None)
        args.output.write_text(json.dumps(report, indent=2) + "\n")
        # Match CTermSymbolic.implies' existential binding; retain the response it discards.
        consequent = target.kast
        for name in target.free_vars:
            if name not in source.cterm.free_vars:
                consequent = KApply(KLabel("#Exists", [K_ITEM, GENERATED_TOP_CELL]), [KVariable(name), consequent])
        raw = symbolic._kore_client.implies(
            symbolic.kast_to_kore(source.cterm.kast), symbolic.kast_to_kore(consequent),
            assume_defined=False, haskell_logging=("DebugUnifyBottom",),
        )
        report["rawResponse"] = {"valid": raw.valid,
            "implication": symbolic.kore_to_kast(raw.implication).to_dict(),
            "predicate": symbolic.kore_to_kast(raw.predicate).to_dict() if raw.predicate is not None else None,
            "substitution": symbolic.kore_to_kast(raw.substitution).to_dict() if raw.substitution is not None else None,
            "haskellLogEntries": raw.haskell_log_entries}
args.output.write_text(json.dumps(report, indent=2) + "\n")
print(json.dumps({key: report[key] for key in
                  ("source", "target", "nativeStatus", "queried", "queryCompleted", "implicationValid") if key in report}))
