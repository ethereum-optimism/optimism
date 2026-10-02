#!/usr/bin/env python3
"""Run freshly executed Go tests from cached, source-bound test binaries."""

from concurrent.futures import ThreadPoolExecutor
import argparse
from datetime import datetime, timezone
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import threading
import tarfile


ROOT = Path(__file__).resolve().parents[2]
BUILD = ROOT / ".ci/go-rollup/build"
PREFIX = "github.com/ethereum-optimism/optimism/op-node/rollup"
SUITE = "go-rollup"
SPEC = importlib.util.spec_from_file_location("shards", Path(__file__).with_name("go-package-shards.py"))
SHARDS = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SHARDS)
PRINT_LOCK = threading.Lock()


def digest(path):
    with path.open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def build():
    manifest = json.loads((ROOT / ".ci" / SUITE / "manifest.json").read_text())
    SHARDS.select_packages(manifest, PREFIX, 0, manifest["total"])
    packages = manifest["packages"]
    if SUITE == "go-tests":
        packages = SHARDS.select_packages(manifest, PREFIX, os.environ["CI_SHARD_INDEX"], manifest["total"])
    BUILD.mkdir(parents=True, exist_ok=True)
    subprocess.run(["go", "build", "-o", str(BUILD / "test2json"), "cmd/test2json"], cwd=ROOT, check=True)
    # go test -c compiles only. TestMain and init functions never run here.
    source = (ROOT / ".ci" / SUITE / "go-list.json").read_text()
    decoder, offset, objects = json.JSONDecoder(), 0, {}
    while offset < len(source):
        if source[offset].isspace():
            offset += 1
            continue
        item, offset = decoder.raw_decode(source, offset)
        objects[item["ImportPath"]] = item

    def compile_package(item):
        index, package = item
        output = BUILD / f"{index}.test"
        output.unlink(missing_ok=True)
        subprocess.run(["go", "test", "-c", "-p=4", "-tags=ci", "-o", str(output), package], cwd=ROOT, check=True)
        has_tests = bool(objects[package].get("TestGoFiles") or objects[package].get("XTestGoFiles"))
        if has_tests and not output.is_file():
            raise ValueError(f"Compilation did not produce a test binary: {package}")
        return package, {"file": output.name if has_tests else None,
                         "sha256": digest(output) if has_tests else None}

    with ThreadPoolExecutor(max_workers=4) as pool:
        binaries = dict(pool.map(compile_package, enumerate(packages)))
    metadata = {"commit_sha": os.environ["CI_COMMIT_SHA"], "tags": ["ci"],
                "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
                "test2json_sha256": digest(BUILD / "test2json"), "packages": binaries}
    if SUITE == "go-tests":
        metadata.update(suite=SUITE, settings=manifest["settings"],
                        shard_index=int(os.environ["CI_SHARD_INDEX"]), shard_total=manifest["total"],
                        manifest_sha256=digest(ROOT / ".ci" / SUITE / "manifest.json"),
                        compile_root=str(ROOT))
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    if head != metadata["commit_sha"]:
        raise ValueError("Go compilation belongs to a different commit")
    (BUILD / "metadata.json").write_text(json.dumps(metadata, indent=2) + "\n")
    (BUILD / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    if SUITE == "go-tests":
        # Runtime modules come from the shared preparation artifact. Do not
        # duplicate the entire module set into every shard's binary artifact.
        return
    # The production-import guards load the root package with go/packages at
    # runtime. Retain its real module sources and all downloaded graph metadata,
    # while excluding unrelated modules and all compiler object caches.
    source = subprocess.check_output(["go", "list", "-deps", "-json", "./op-node/rollup"], cwd=ROOT, text=True)
    decoder, offset, modules = json.JSONDecoder(), 0, set()
    module_cache = Path(os.environ["GOMODCACHE"])
    while offset < len(source):
        if source[offset].isspace():
            offset += 1
            continue
        item, offset = decoder.raw_decode(source, offset)
        module = item.get("Module", {})
        module = module.get("Replace", module)
        if module.get("Dir") and not module.get("Main"):
            directory = Path(module["Dir"])
            if directory.is_relative_to(module_cache):
                modules.add(directory.relative_to(module_cache))
    with tarfile.open(BUILD / "runtime-modules.tar.gz", "w:gz") as archive:
        for path in [Path("cache/download"), *sorted(modules)]:
            archive.add(module_cache / path, arcname=str(path), filter=lambda info: None if info.name.endswith((".zip", ".tmp", ".lock")) else info)


def verify():
    metadata = json.loads((BUILD / "metadata.json").read_text())
    manifest_path = ROOT / ".ci" / SUITE / "manifest.json"
    manifest = json.loads(manifest_path.read_text())
    selected = SHARDS.select_packages(manifest, PREFIX, os.environ["CI_SHARD_INDEX"], os.environ["CI_SHARD_TOTAL"])
    if metadata.get("commit_sha") != os.environ["CI_COMMIT_SHA"] or metadata.get("tags") != ["ci"]:
        raise ValueError("Go compilation revision or tags differ from this verdict")
    expected_packages = selected if SUITE == "go-tests" else manifest["packages"]
    if sorted(metadata["packages"]) != sorted(expected_packages):
        raise ValueError("Compiled Go package coverage differs from the authoritative manifest")
    if SUITE == "go-tests":
        expected_settings = {"tags": ["ci"], "short": False, "count": 1,
                             "package_parallelism": 4, "parallel": int(os.environ["PARALLEL"]),
                             "timeout": os.environ.get("TEST_TIMEOUT", "40m"),
                             "rerun_fails": 3, "rerun_fails_max_failures": 50}
        if (metadata.get("suite") != SUITE or metadata.get("settings") != expected_settings
                or manifest.get("settings") != expected_settings
                or manifest.get("commit_sha") != os.environ["CI_COMMIT_SHA"]
                or metadata.get("shard_index") != int(os.environ["CI_SHARD_INDEX"])
                or metadata.get("shard_total") != int(os.environ["CI_SHARD_TOTAL"])
                or metadata.get("manifest_sha256") != digest(manifest_path)
                or metadata.get("compile_root") != str(ROOT)):
            raise ValueError("Go suite settings, shard, source paths or manifest differ from this verdict")
    if digest(BUILD / "test2json") != metadata["test2json_sha256"]:
        raise ValueError("Go JSON reporter changed")
    for package in selected:
        binary = metadata["packages"][package]
        if binary["file"] is not None and digest(BUILD / binary["file"]) != binary["sha256"]:
            raise ValueError(f"Go test binary changed: {package}")
    return metadata, selected


def emit(line):
    with PRINT_LOCK:
        sys.stdout.write(line)
        sys.stdout.flush()


def run(args):
    metadata, selected = verify()
    # gotestsum's raw-command reruns append -test.run=<regexp> and a package.
    # Only a rerun may narrow the selection; the initial attempt runs it all.
    run_filter = None
    if args:
        if len(args) != 2 or not args[0].startswith("-test.run=") or args[1] not in selected:
            raise ValueError("Unexpected gotestsum rerun arguments")
        run_filter, selected = args[0][len("-test.run="):], [args[1]]

    def run_package(package):
        binary = metadata["packages"][package]["file"]
        if binary is None:
            now = datetime.now(timezone.utc).isoformat()
            for action in ("start", "skip"):
                emit(json.dumps({"Time": now, "Action": action, "Package": package}) + "\n")
            return 0
        # go test runs each binary in its package directory. Preserve fixtures,
        # -count=1, -parallel=nproc, -timeout=40m, and at most four packages.
        directory = ROOT / package.removeprefix("github.com/ethereum-optimism/optimism/")
        command = [str(BUILD / "test2json"), "-t", "-p", package, str(BUILD / binary),
                   "-test.v=test2json", "-test.count=1", "-test.parallel=" + os.environ["PARALLEL"],
                   "-test.timeout=" + os.environ.get("TEST_TIMEOUT", "40m"), "-test.paniconexit0"]
        if run_filter is not None:
            command.append("-test.run=" + run_filter)
        process = subprocess.Popen(command, cwd=directory, stdout=subprocess.PIPE, text=True)
        for line in process.stdout:
            emit(line)
        return process.wait()

    with ThreadPoolExecutor(max_workers=4) as pool:
        statuses = list(pool.map(run_package, selected))
    return 1 if any(status != 0 for status in statuses) else 0


if __name__ == "__main__":
    try:
        parser = argparse.ArgumentParser(description=__doc__)
        parser.add_argument("command", choices=["build", "verify", "run"])
        parser.add_argument("--suite", choices=["go-rollup", "go-tests"], default="go-rollup")
        options, arguments = parser.parse_known_args()
        command, SUITE = options.command, options.suite
        BUILD = ROOT / ".ci" / SUITE / "build"
        if SUITE == "go-tests":
            PREFIX = "github.com/ethereum-optimism/optimism"
        if command == "build":
            build()
        elif command == "verify":
            verify()
        elif command == "run":
            sys.exit(run(arguments))
        else:
            raise ValueError("Expected build, verify, or run")
    except (ValueError, KeyError, OSError, subprocess.CalledProcessError) as error:
        print(f"ERROR: {error}", file=sys.stderr)
        sys.exit(1)
