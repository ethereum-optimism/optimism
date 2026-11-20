#!/usr/bin/env python3
"""Discover the Go test packages and place them on shards.

The script writes `.ci/go-tests/manifest.json` with these keys:
  commit_sha  The commit that this task tests (CI_COMMIT_SHA).
  shards      One sorted package list per shard. Each package is in exactly one list.
  no_tests    The packages that have no test files with `-tags=ci`.
"""

import argparse
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[3]
SPEC = importlib.util.spec_from_file_location("shards", Path(__file__).with_name("go-package-shards.py"))
SHARDS = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SHARDS)


def discover(total, timings):
    # The selection and the build tag of the `_go-tests-ci-internal` recipe.
    selected = subprocess.check_output(["just", "list-test-packages"], cwd=ROOT, text=True).split()
    if not selected:
        raise ValueError("list-test-packages produced no packages")
    listing = subprocess.check_output(["go", "list", "-e", "-tags=ci", "-json", *selected], cwd=ROOT, text=True)
    packages = SHARDS.read_go_packages(listing)
    no_tests = sorted(package["ImportPath"] for package in packages
                      if not (package.get("TestGoFiles") or package.get("XTestGoFiles")))
    durations = json.loads(timings.read_text())["packages"]
    manifest = {"commit_sha": os.environ["CI_COMMIT_SHA"], "no_tests": no_tests,
                "shards": SHARDS.assign([package["ImportPath"] for package in packages], total, durations)}
    output = ROOT / ".ci/go-tests"
    output.mkdir(parents=True, exist_ok=True)
    (output / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")


def main():
    # A selected Go toolchain can be newer than mise's exported GOROOT.
    # Let the executable infer its matching SDK.
    os.environ.pop('GOROOT', None)
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--total", type=int, required=True, help="the number of shards")
    parser.add_argument("--timings", type=Path, required=True, help="a JSON file with package durations")
    args = parser.parse_args()
    try:
        discover(args.total, args.timings)
    except (ValueError, KeyError, OSError, subprocess.CalledProcessError) as error:
        print(f"ERROR: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
