"""Temporary CI diagnostic: replay frozen segments, without updating any proof graph."""

import argparse
import hashlib
import json
import logging
import time
from pathlib import Path

from kevm_pyk.kevm import KEVM
from kevm_pyk.utils import legacy_explore
from kontrol.foundry import KontrolSemantics
from kontrol.options import ProveOptions
from pyk.cterm import CTerm

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("root", type=Path)
parser.add_argument("node", type=int, choices=(23, 30))
parser.add_argument("--check-only", action="store_true")
args = parser.parse_args()
target, depth = {23: (25, 40), 30: (33, 381)}[args.node]
proof_dir, = args.root.glob("proofs/*WithdrawalAuthorizationKontrol.prove_checkWithdrawal_equivalence*")
graph = json.loads((proof_dir / "kcfg/kcfg.json").read_text())
assert any(e["source"] == args.node and e["target"] == target and e["depth"] == depth for e in graph["edges"])
node_file = proof_dir / f"kcfg/nodes/{args.node}.json"
source = CTerm.from_dict(json.loads(node_file.read_text())["cterm"])
expected = CTerm.from_dict(json.loads((proof_dir / f"kcfg/nodes/{target}.json").read_text())["cterm"])
output = args.root.parent / "profiles" / f"node-{args.node}"
output.mkdir(parents=True, exist_ok=True)
options = ProveOptions({})
cut_points = KontrolSemantics.cut_point_rules(
    options.break_on_jumpi, options.break_on_jump, options.break_on_calls,
    options.break_on_storage, options.break_on_basic_blocks, options.break_on_load_program,
)
manifest = dict(
    diagnostic_only=True,
    source_pipeline=133962,
    source_revision="fc60971443bac306796749a26d8ad37041b8bede",
    node=args.node,
    target=target,
    depth=depth,
    request_depth=10000,
    cut_point_rules=cut_points,
    input_sha256=hashlib.sha256(node_file.read_bytes()).hexdigest(),
)
(output / "input.json").write_text(json.dumps(manifest, indent=2))
print(json.dumps(manifest), flush=True)
if args.check_only:
    raise SystemExit(0)

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(name)s %(levelname)s %(message)s")
kevm = KEVM(args.root / "kompiled")
command = [
    "kore-rpc-booster", "--no-post-exec-simplify",
    "--equation-max-recursion", "100", "--equation-max-iterations", "1000",
    "--log-level", "Timing", "--log-timestamps", "--log-format", "json",
    "--solver-transcript", str(output / "smt.log"),
]
started = time.monotonic()
with legacy_explore(
    kevm,
    kore_rpc_command=command,
    llvm_definition_dir=args.root / "kompiled/llvm-library",
    smt_timeout=16000,
    smt_retry_limit=0,
    log_succ_rewrites=False,
    log_fail_rewrites=False,
) as explorer:
    # Use the saved constraints unchanged: no assume-defined, gas, or stack-setting override.
    # Match the original prover request through its symbolic branch, not just the saved edge.
    result = explorer.cterm_symbolic.execute(
        source, depth=10000, cut_point_rules=cut_points,
        terminal_rules=["EVM.halt"], haskell_logging=False,
    )
    manifest.update(
        replay_seconds=time.monotonic() - started,
        actual_depth=result.depth,
        vacuous=result.vacuous,
        next_states=len(result.next_states),
        exact_saved_target_match=result.state == expected,
    )
    (output / "result.json").write_text(json.dumps(manifest, indent=2))
    (output / "state.json").write_text(json.dumps(result.state.to_dict()))
    (output / "next-states.json").write_text(json.dumps([n.state.to_dict() for n in result.next_states]))
    print(json.dumps(manifest), flush=True)
    assert result.depth == depth and not result.vacuous and len(result.next_states) == 2
    assert result.state == expected, "Replay endpoint differs; inspect before drawing performance conclusions"
