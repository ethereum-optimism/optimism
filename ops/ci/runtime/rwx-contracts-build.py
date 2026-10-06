#!/usr/bin/env python3
"""Package compilation separately from fresh, complete contract verdicts."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess


ROOT = Path(__file__).resolve().parents[3]
PREP = ROOT / ".ci/contracts-prepare"
BUILD = ROOT / ".ci/contracts-build"
CONTRACTS = ROOT / "packages/contracts-bedrock"


def digest(path):
    with path.open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def configuration():
    return json.loads(subprocess.check_output(["forge", "config", "--json"], cwd=CONTRACTS))


def identity():
    return {
        "commit_sha": os.environ["CI_COMMIT_SHA"],
        "feature": os.environ["CONTRACT_FEATURE"],
        "profile": os.environ["FOUNDRY_PROFILE"],
        "configuration": configuration(),
    }


def verify():
    recorded = json.loads((BUILD / "metadata.json").read_text())
    expected = identity()
    for key, value in expected.items():
        if recorded.get(key) != value:
            raise ValueError(f"Contract compilation {key} differs from this verdict")
    for filename in ("source.tar.gz", "compilation.tar.gz", "test-validation"):
        if recorded["sha256"].get(filename) != digest(BUILD / filename):
            raise ValueError(f"Contract compilation artifact changed: {filename}")
    if not (CONTRACTS / "scripts/go-ffi/go-ffi").is_file():
        raise ValueError("Compiled Go FFI is missing")
    if (CONTRACTS / ".gitcommit").read_text().strip() != os.environ["CI_COMMIT_SHA"]:
        raise ValueError("Contract deployment commit metadata differs from this verdict")


def package():
    if (PREP / "commit-sha.txt").read_text().strip() != os.environ["CI_COMMIT_SHA"]:
        raise ValueError("Contract prerequisites belong to a different commit")
    BUILD.mkdir(parents=True, exist_ok=True)
    shutil.copy2(PREP / "source.tar.gz", BUILD / "source.tar.gz")
    shutil.copy2(PREP / "test-validation", BUILD / "test-validation")
    # Include only compilation state, never inherited fuzz cases or verdicts.
    paths = ["forge-artifacts", "cache/solidity-files-cache.json"]
    if (CONTRACTS / "artifacts/build-info").exists():
        paths.append("artifacts/build-info")
    subprocess.run(["tar", "-czf", str(BUILD / "compilation.tar.gz"), *paths], cwd=CONTRACTS, check=True)
    metadata = identity()
    metadata["sha256"] = {name: digest(BUILD / name) for name in ("source.tar.gz", "compilation.tar.gz", "test-validation")}
    (BUILD / "metadata.json").write_text(json.dumps(metadata, indent=2) + "\n")


def restore(artifact):
    if BUILD.exists():
        shutil.rmtree(BUILD)
    shutil.copytree(artifact, BUILD)
    subprocess.run(["tar", "-xzf", str(BUILD / "source.tar.gz"), "-C", str(ROOT)], check=True)
    # Delete every old artifact before unpacking the selected feature's build.
    for path in ("forge-artifacts", "cache", "artifacts/build-info"):
        shutil.rmtree(CONTRACTS / path, ignore_errors=True)
    subprocess.run(["tar", "-xzf", str(BUILD / "compilation.tar.gz"), "-C", str(CONTRACTS)], check=True)
    PREP.mkdir(parents=True, exist_ok=True)
    shutil.copy2(BUILD / "test-validation", PREP / "test-validation")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("package", "restore", "verify"))
    parser.add_argument("artifact", nargs="?", type=Path)
    args = parser.parse_args()
    if args.command == "restore":
        if args.artifact is None:
            parser.error("restore needs the compilation artifact directory")
        restore(args.artifact)
    elif args.command == "package":
        package()
    else:
        verify()
