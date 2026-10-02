#!/usr/bin/env python3
"""Retain authoritative Go discovery and effective CI settings without secrets."""

import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[2]
MODULE = "github.com/ethereum-optimism/optimism"
SUITES = {"go-rollup": MODULE + "/op-node/rollup", "go-tests": MODULE}
SPEC = importlib.util.spec_from_file_location("shards", Path(__file__).with_name("go-package-shards.py"))
SHARDS = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SHARDS)


def settings(*, fresh=True, short=False):
    return {"tags": ["ci"], "short": short, "count": 1 if fresh else None,
            "package_parallelism": 4, "parallel": int(os.environ.get("PARALLEL", "8")),
            "timeout": os.environ.get("TEST_TIMEOUT", "40m"),
            "rerun_fails": 3, "rerun_fails_max_failures": 50}


def validate_discovery(selected, source, prefix):
    SHARDS.validate_packages(selected, prefix)
    discovered = SHARDS.read_go_packages(source, prefix)
    if discovered != sorted(selected):
        raise ValueError("Tagged discovery differs from the authoritative package selection")
    return discovered


def discover(suite, total, timings):
    output = ROOT / ".ci" / suite
    output.mkdir(parents=True, exist_ok=True)
    if suite == "go-tests":
        selected = subprocess.check_output(["just", "list-test-packages"], cwd=ROOT, text=True).splitlines()
    else:
        selected = subprocess.check_output(["go", "list", "-e", "-tags=ci", "./op-node/rollup/..."], cwd=ROOT, text=True).splitlines()
    SHARDS.validate_packages(selected, SUITES[suite])
    source = subprocess.check_output(["go", "list", "-e", "-tags=ci", "-json", *selected], cwd=ROOT, text=True)
    packages = validate_discovery(selected, source, SUITES[suite])
    (output / "go-list.json").write_text(source)
    durations = json.loads(timings.read_text())["packages"] if timings else None
    manifest = SHARDS.create_manifest(packages, SUITES[suite], total, durations)
    manifest.update(suite=suite, commit_sha=os.environ["CI_COMMIT_SHA"], settings=settings())
    (output / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    (output / "all-packages.txt").write_text("\n".join(packages) + "\n")


def record_circle(flags):
    output = ROOT / "tmp/testlogs"
    selected = (output / "all-packages.txt").read_text().splitlines()
    packages = validate_discovery(selected, (output / "discovery.json").read_text(), MODULE)
    sha = os.environ.get("CIRCLE_SHA1") or os.environ.get("CI_COMMIT_SHA") or subprocess.check_output(
        ["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    branch = os.environ.get("CIRCLE_BRANCH") or os.environ.get("CI_BRANCH") or subprocess.check_output(
        ["git", "rev-parse", "--abbrev-ref", "HEAD"], cwd=ROOT, text=True).strip()
    metadata = {"version": 1, "suite": "go-tests", "commit_sha": sha,
                "branch": branch, "packages": packages,
                "shard_index": int(os.environ.get("CIRCLE_NODE_INDEX", "0")),
                "shard_total": int(os.environ.get("CIRCLE_NODE_TOTAL", "1")),
                "settings": settings(fresh=os.environ.get("CI_GO_FRESH_TESTS", "false") in ("1", "true"),
                                     short=flags == "-short")}
    (output / "selection.json").write_text(json.dumps(metadata, indent=2) + "\n")


def source_digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    discovery = commands.add_parser("discover")
    discovery.add_argument("--suite", choices=SUITES, required=True)
    discovery.add_argument("--total", required=True)
    discovery.add_argument("--timings", type=Path)
    circle = commands.add_parser("record-circle")
    circle.add_argument("--flags", default="")
    args = parser.parse_args()
    try:
        if args.command == "discover":
            discover(args.suite, args.total, args.timings)
        else:
            record_circle(args.flags)
    except (ValueError, KeyError, OSError, subprocess.CalledProcessError) as error:
        print(f"ERROR: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
