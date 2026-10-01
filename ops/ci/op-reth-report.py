#!/usr/bin/env python3
"""Record source/cache provenance and reject incomplete shadow evidence."""

import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time
import xml.etree.ElementTree as ET


def digest(path):
    with Path(path).open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def command(*args):
    return subprocess.check_output(args, text=True).strip()


def write(directory, name, data):
    Path(directory, name).write_text(json.dumps(data, indent=2, sort_keys=True) + "\n")


def source_sha(expected):
    actual = command("git", "rev-parse", "HEAD")
    if actual != expected:
        raise ValueError(f"Source revision {actual} differs from expected {expected}")
    return actual


def binary_manifest(directory, expected, names, verify=False):
    directory = Path(directory)
    if not names or len(set(names)) != len(names):
        raise ValueError("Binary inventory must be nonempty and unique")
    if verify:
        stored = json.loads((directory / "binaries.json").read_text())
        if stored["source_sha"] != expected or set(stored["files"]) != set(names):
            raise ValueError("Binary source revision or inventory mismatch")
    else:
        stored = {"source_sha": source_sha(expected), "files": {}}
    for name in names:
        path = directory / name
        if not path.is_file() or path.stat().st_size == 0:
            raise ValueError(f"Missing or empty build artifact: {path}")
        actual = digest(path)
        if verify:
            if stored["files"][name] != actual:
                raise ValueError(f"Build artifact checksum mismatch: {name}")
        else:
            stored["files"][name] = actual
    if not verify:
        write(directory, "binaries.json", stored)


def vector_manifest(directory, expected, verify=False):
    files = sorted(Path("testdata/micro/compact").glob("*.json"))
    if not files:
        raise ValueError("Empty compact vector inventory")
    inventory = {}
    for path in files:
        values = json.loads(path.read_text())
        if not isinstance(values, list) or len(values) != 100 or not all(isinstance(v, str) for v in values):
            raise ValueError(f"Incomplete vector file: {path}")
        inventory[path.name] = digest(path)
    manifest = {"base_sha": expected, "vectors_per_type": 100, "files": inventory}
    if verify:
        if json.loads(Path(directory, "vectors.json").read_text()) != manifest:
            raise ValueError("Compact vector source, checksum or inventory mismatch")
    else:
        write(directory, "vectors.json", manifest)


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


def metadata(directory, job, started, status):
    data = {
        "job": job, "source_sha": command("git", "rev-parse", "HEAD"),
        "elapsed_seconds": int(time.time()) - int(started), "exit_code": int(status),
        "cargo_lock_sha256": digest("rust/Cargo.lock"),
        "rustc": command("rustc", "--version"), "cargo": command("cargo", "--version"),
        "sccache": command("sccache", "--version"),
        "nextest": command("cargo", "nextest", "--version"),
        "cpus": os.cpu_count(),
        "cache_epoch": os.environ.get("CACHE_EPOCH", "v1"),
        "build_probe": os.environ.get("BUILD_PROBE", ""),
        "target_cache_mode": os.environ.get("TARGET_CACHE_MODE", "keep"),
        "cargo_incremental": os.environ.get("CARGO_INCREMENTAL"),
    }
    for key, name in (("cpu_quota", "/sys/fs/cgroup/cpu.max"),
                      ("memory_limit", "/sys/fs/cgroup/memory.max")):
        if Path(name).is_file():
            data[key] = Path(name).read_text().strip()
    write(directory, "metadata.json", data)


def main(args):
    operation, directory, *rest = args
    if operation in ("binaries", "verify-binaries"):
        binary_manifest(directory, rest[0], rest[1:], operation == "verify-binaries")
    elif operation in ("vectors", "verify-vectors"):
        vector_manifest(directory, rest[0], operation == "verify-vectors")
    elif operation == "integration":
        integration_report(directory)
    elif operation == "metadata":
        metadata(directory, *rest)
    else:
        raise ValueError(f"Unknown report operation: {operation}")


if __name__ == "__main__":
    main(sys.argv[1:])
