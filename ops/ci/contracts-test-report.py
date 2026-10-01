#!/usr/bin/env python3
"""Validate complete standard contract inputs and the resulting JUnit verdict."""

import argparse
import json
import os
from pathlib import Path
import sys
import xml.etree.ElementTree as ET


FEATURES = {
    "main": None,
    "CUSTOM_GAS_TOKEN": "SYS_FEATURE__CUSTOM_GAS_TOKEN",
    "OPTIMISM_PORTAL_INTEROP": "DEV_FEATURE__OPTIMISM_PORTAL_INTEROP",
    "ZK_DISPUTE_GAME": "DEV_FEATURE__ZK_DISPUTE_GAME",
}
FILTERS = ("match_test", "no_match_test", "match_contract", "no_match_contract", "match_path", "no_match_path", "skip")


def prepare(config_path, reports):
    branch = os.environ["CI_BRANCH"]
    feature = os.environ["CONTRACT_FEATURE"]
    profile = "ci" if branch == "develop" else "liteci"
    config = json.loads(config_path.read_text(encoding="utf-8"))
    expected = {
        "test": "test", "ffi": True, "optimizer": profile == "ci",
        "optimizer_runs": 999999 if profile == "ci" else 0,
    }
    for key, value in expected.items():
        if config.get(key) != value:
            raise ValueError(f"Unexpected effective Foundry {key}; expected {value!r}")
    if config.get("fuzz", {}).get("runs") != 128:
        raise ValueError("Standard contract fuzz runs must be 128")
    if config.get("invariant", {}).get("runs") != 64 or config.get("invariant", {}).get("depth") != 32:
        raise ValueError("Standard contract invariants must use 64 runs and depth 32")
    if config.get("fuzz", {}).get("timeout") is not None or config.get("invariant", {}).get("timeout") is not None:
        raise ValueError("Standard contract runs cannot be reduced by fuzz or invariant timeouts")
    for key in FILTERS:
        if config.get(key):
            raise ValueError(f"Standard contract suite cannot use the {key} filter")
    if config.get("eth_rpc_url") or config.get("fork_block_number") is not None:
        raise ValueError("Standard contract suite cannot enable a Foundry RPC fork")
    files = sorted(str(path) for path in Path("test").rglob("*.t.sol") if path.is_file())
    if not files:
        raise ValueError("No contract test files discovered")
    (reports / "test-files.txt").write_text("".join(f"{path}\n" for path in files), encoding="utf-8")
    metadata = {
        "branch": branch, "profile": profile, "feature": feature,
        "enabled_feature_environment": FEATURES[feature], "test_files": len(files),
        "fuzz_runs": 128, "invariant_runs": 64, "invariant_depth": 32,
        "fork_tests": False, "l2_fork_tests": False, "l2cm_activation_tests": False,
    }
    (reports / "metadata.json").write_text(json.dumps(metadata, indent=2) + "\n", encoding="utf-8")


def verdict(junit, reports):
    root = ET.parse(junit).getroot()
    if root.tag not in ("testsuite", "testsuites"):
        raise ValueError("Expected a JUnit testsuite or testsuites document")
    cases = list(root.iter("testcase"))
    skipped = sum(case.find("skipped") is not None for case in cases)
    failures = sum(case.find("failure") is not None for case in cases)
    errors = sum(case.find("error") is not None for case in cases)
    summary = {"tests": len(cases), "skipped": skipped, "failures": failures, "errors": errors}
    (reports / "summary.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    if not cases or len(cases) == skipped:
        raise ValueError("Standard contract JUnit report contains no executed tests")
    if failures or errors:
        raise ValueError(f"JUnit reports {failures} failures and {errors} errors")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("prepare", "verdict"))
    parser.add_argument("path", type=Path)
    parser.add_argument("--reports", type=Path, default=Path("results/reports"))
    args = parser.parse_args()
    try:
        if args.mode == "prepare":
            prepare(args.path, args.reports)
        else:
            verdict(args.path, args.reports)
    except (KeyError, OSError, ValueError, ET.ParseError) as error:
        print(f"Contract report validation failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
