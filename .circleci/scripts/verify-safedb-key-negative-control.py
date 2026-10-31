import contextlib
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time

PACKAGE = "./op-acceptance-tests/tests/safedb_parity"
RESYNC = "TestSafeHeadRecordsMatchAfterKonaCLResync"
STEADY = "TestSafeHeadRecordsMatchWhileDeriving"


@contextlib.contextmanager
def wrong_record_key(source):
    original = source.read_bytes()
    declaration = b"func (d *SafeDB) SafeHeadUpdated(safeHead eth.L2BlockRef, l1Head eth.BlockID) error {\n"
    if original.count(declaration) != 1:
        raise RuntimeError("Expected exactly one SafeHeadUpdated declaration")
    try:
        source.write_bytes(original.replace(declaration, declaration + b"\tl1Head = safeHead.L1Origin\n"))
        yield
    finally:
        source.write_bytes(original)
        if source.read_bytes() != original:
            raise RuntimeError("Failed to restore original source")


def validate_events(events, exit_code, expected_names, expect_mismatch=False):
    terminal = {}
    mismatch = False
    for event in events:
        name = event.get("Test")
        if name in expected_names:
            if event.get("Action") in {"fail", "pass", "skip"}:
                terminal[name] = event["Action"]
            if event.get("Action") == "output" and "Mismatched safe head data" in event.get("Output", ""):
                mismatch = True
    expected = {name: "fail" if expect_mismatch else "pass" for name in expected_names}
    valid = terminal == expected and (exit_code != 0 and mismatch if expect_mismatch else exit_code == 0)
    return {"exit_code": exit_code, "expected_mismatch": expect_mismatch, "mismatch_observed": mismatch, "terminal_tests": terminal, "valid": valid}


def run_phase(name, names, output, expect_mismatch=False):
    events_path = output / f"{name}-events.json"
    junit_path = str(output / f"{name}.xml") if expect_mismatch else f"op-acceptance-tests/results/key-control-{name}.xml"
    command = ["gotestsum", "--format", "testname", "--jsonfile", str(events_path), "--junitfile", junit_path, "--", "-count=1", "-p=1", "-parallel=1", "-timeout=12m", "-run=^(" + "|".join(names) + ")$", PACKAGE]
    environment = dict(os.environ, BASH_ENV="/dev/null", DEVSTACK_L2CL_KIND="op-node", LOG_LEVEL="info")
    print(f"Starting {name}: {command}", flush=True)
    started = time.monotonic()
    with (output / f"{name}.log").open("w") as log:
        process = subprocess.run(command, env=environment, stdout=log, stderr=subprocess.STDOUT, check=False)
    events = [json.loads(line) for line in events_path.read_text().splitlines()]
    result = validate_events(events, process.returncode, names, expect_mismatch)
    result["elapsed_seconds"] = time.monotonic() - started
    print(f"Completed {name}: {result}", flush=True)
    return result


def main():
    output = Path("op-acceptance-tests/logs/safedb-key-negative-control")
    output.mkdir(parents=True, exist_ok=True)
    Path("op-acceptance-tests/results").mkdir(parents=True, exist_ok=True)
    source = Path("op-node/node/safedb/safedb.go")
    original_hash = hashlib.sha256(source.read_bytes()).hexdigest()
    summary = {"phases": {}, "source_commit": subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip(), "source_sha256_before": original_hash}
    try:
        summary["phases"]["baseline"] = run_phase("baseline", [STEADY, RESYNC], output)
        if summary["phases"]["baseline"]["terminal_tests"].get(STEADY) != "pass":
            raise RuntimeError("Positive steady-state baseline must pass without skips")
        with wrong_record_key(source):
            summary["phases"]["wrong-key"] = run_phase("wrong-key", [STEADY], output, expect_mismatch=True)
        summary["phases"]["restored"] = run_phase("restored", [STEADY, RESYNC], output)
    except Exception as error:
        summary["error"] = str(error)
    finally:
        summary["source_sha256_after"] = hashlib.sha256(source.read_bytes()).hexdigest()
        summary["source_restored"] = summary["source_sha256_after"] == original_hash
        summary["success"] = "error" not in summary and summary["source_restored"] and len(summary["phases"]) == 3 and all(phase["valid"] for phase in summary["phases"].values())
        (output / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
        print(json.dumps(summary, indent=2), flush=True)
    return 0 if summary["success"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
