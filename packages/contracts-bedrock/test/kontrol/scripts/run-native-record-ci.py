"""CI-only single-receipt smoke diagnostic; never a full-suite readiness gate.

Temporary integration draft. The source manifest retains all eight receipts, but
the first native execution probe selects receipt49 to inspect frontend/domain and
CALL/return behavior before scheduling the remaining expensive proof searches.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tarfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source_bundle", type=Path)
    parser.add_argument("--definition", type=Path, default=Path("kout-proofs/kompiled"))
    parser.add_argument("--fixture", type=Path, default=Path("kout-proofs/WithdrawalAuthorization.k.sol/WithdrawalAuthorizationKontrol.json"))
    args = parser.parse_args()
    if os.environ.get("CI") != "true":
        raise SystemExit("Native proof execution is authorized in CI only")
    with tarfile.open(args.source_bundle, "r:xz") as archive:
        members = archive.getmembers()
        expected = {"manifest.json", *(f"receipt-{n}.k" for n in (49, 50, 51, 52, 57, 58, 59, 60))}
        if len(members) != len(expected) or {m.name for m in members} != expected:
            raise ValueError("Unexpected diagnostic bundle contents")
        if any(not m.isfile() or m.size > 8_000_000 for m in members):
            raise ValueError("Unexpected source member type or size")
        sources = {m.name: archive.extractfile(m).read() for m in members}
    manifest = json.loads(sources.pop("manifest.json"))
    if manifest["status"] != "EIGHT_SOURCE_SPECS_RENDERED_NOT_FRONTEND_CHECKED_OR_PROVED":
        raise ValueError("Require the explicit diagnostic source manifest")
    if hashlib.sha256((args.definition / "parsed.json").read_bytes()).hexdigest() != manifest["parsedDefinitionSha256"]:
        raise ValueError("Fresh CI definition differs from the source export definition")
    fixture = json.loads(args.fixture.read_text())
    runtime = bytes.fromhex(fixture["deployedBytecode"]["object"].removeprefix("0x"))
    if hashlib.sha256(runtime).hexdigest() != "cf893e7e999ac91964ac43e1081bc6a30ee047bc84940351064928510f130fea":
        raise ValueError("Fresh CI caller runtime differs from the saved receipt runtime")
    rows = manifest["specs"]
    if [row["node"] for row in rows] != [49, 50, 51, 52, 57, 58, 59, 60]:
        raise ValueError("Source bundle is missing an initializer receipt")
    output = Path("kout-proofs/native-record-diagnostic")
    output.mkdir(parents=True, exist_ok=True)
    for row in rows:
        relative = Path(row["file"])
        if relative.is_absolute() or ".." in relative.parts:
            raise ValueError("Unexpected source bundle path")
        source = sources[str(relative)]
        if len(source) != row["sourceBytes"] or hashlib.sha256(source).hexdigest() != row["sourceSha256"]:
            raise ValueError("Source bundle checksum mismatch")
        (output / f"receipt-{row['node']}.k").write_bytes(source)
    selected = rows[0]
    state = {"status": "STARTING_SINGLE_RECEIPT_DIAGNOSTIC", "selectedReceipt": selected["node"],
             "allEightSourceChecksumsVerified": True, "fullSecurityMilestone": False,
             "remainingReceiptsNotExecuted": [row["node"] for row in rows[1:]],
             "limits": "Caller gas/memory/raw ABI correspondence and all accepted security obligations remain open."}
    (output / "diagnostic-status.json").write_text(json.dumps(state, indent=2) + "\n")
    command = ["timeout", "--signal=INT", "--kill-after=30s", "60m", "kevm", "prove",
        str(output / "receipt-49.k"), "--definition", str(args.definition),
        "--spec-module", selected["module"], "--save-directory", str(output / "proofs"),
        "--reinit", "--workers", "1", "--max-depth", "1000", "--max-iterations", "10000",
        "--maintenance-rate", "1", "--smt-timeout", "16000", "--smt-retry-limit", "0",
        "--no-counterexample-information", "--kore-rpc-command",
        "kore-rpc-booster --equation-max-recursion 100 --equation-max-iterations 1000 --fallback-on Stuck,Aborted"]
    print("NATIVE CALL DIAGNOSTIC ONLY: receipt49; no full-suite or security completion", flush=True)
    result = subprocess.run(command, check=False)
    state.update(status="DIAGNOSTIC_PROCESS_FINISHED", exitCode=result.returncode)
    (output / "diagnostic-status.json").write_text(json.dumps(state, indent=2) + "\n")
    raise SystemExit(result.returncode)


if __name__ == "__main__":
    main()
