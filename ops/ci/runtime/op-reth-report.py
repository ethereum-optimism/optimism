#!/usr/bin/env python3
"""Prepare the superchain bundle and reject incomplete integration test evidence."""

import hashlib
import json
import re
from pathlib import Path
import sys
import xml.etree.ElementTree as ET


def digest(path):
    with Path(path).open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def write(directory, name, data):
    Path(directory, name).write_text(json.dumps(data, indent=2, sort_keys=True) + "\n")


def prepare_superchain():
    directory = Path("rust/op-reth/crates/chainspec/res")
    expected = (directory / "superchain-configs.tar.sha256").read_text().split()[0]
    if not re.fullmatch(r"[0-9a-f]{64}", expected):
        raise ValueError("Invalid committed superchain archive checksum")
    archive = directory / "superchain-configs.tar"
    if archive.exists() and digest(archive) != expected:
        archive.unlink()


def integration_report(directory):
    directory = Path(directory)
    discovery = json.loads((directory / "discovery.json").read_text())
    expected = {}
    ignored = set()
    for suite_id, suite in discovery["rust-suites"].items():
        for name, case in suite["testcases"].items():
            key = (suite_id, name)
            match = case.get("filter-match", {})
            if case["ignored"] is True and match == {"status": "mismatch", "reason": "ignored"}:
                ignored.add(key)
            elif case["ignored"] is not False or match != {"status": "matches"}:
                raise ValueError(f"Unexpected test filter: {key}")
            expected[key] = key in ignored
    if not expected:
        raise ValueError("Empty integration test discovery")
    if discovery.get("test-count", len(expected)) != len(expected):
        raise ValueError("Integration discovery count differs from its case inventory")
    actual = {}
    flaky = []
    for case in ET.parse(directory / "junit.xml").iter("testcase"):
        key = (case.attrib.get("classname", ""), case.attrib["name"])
        if key in actual:
            raise ValueError(f"Duplicate test verdict: {key}")
        outcome = "pass"
        if case.find("skipped") is not None:
            outcome = "skip"
        elif case.find("failure") is not None or case.find("error") is not None:
            outcome = "fail"
        actual[key] = outcome
        if case.find("flakyFailure") is not None or case.find("flakyError") is not None:
            flaky.append(key)
    # Pinned nextest omits ignored tests from JUnit. Their authoritative list
    # entry must explicitly say "ignored"; every runnable test still needs one
    # verdict. Never infer a skip merely because a verdict is missing.
    missing = set(expected) - ignored - set(actual)
    extra = set(actual) - set(expected)
    summary = {"discovered": len(expected), "reported": len(actual),
               "missing": sorted(missing), "extra": sorted(extra),
               "outcomes": {s: list(actual.values()).count(s) for s in ("pass", "skip", "fail")},
               "ignored_cases": [{"suite": s, "name": n, "reason": "ignored"}
                                 for s, n in sorted(ignored)],
               "retried_cases": sorted(flaky)}
    summary["outcomes"]["skip"] += len(ignored - set(actual))
    write(directory, "coverage.json", summary)
    if missing or extra:
        raise ValueError(f"Incomplete integration evidence: {len(missing)} missing, {len(extra)} extra")
    for key, outcome in actual.items():
        if (outcome == "skip") != expected[key]:
            raise ValueError(f"Unexpected integration skip status: {key}")
    print(json.dumps(summary, sort_keys=True))


def main(args):
    operation, *rest = args
    if operation == "integration":
        integration_report(*rest)
    elif operation == "prepare-superchain":
        prepare_superchain()
    else:
        raise ValueError(f"Unknown report operation: {operation}")


if __name__ == "__main__":
    main(sys.argv[1:])
