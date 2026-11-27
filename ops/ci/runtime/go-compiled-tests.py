#!/usr/bin/env python3
"""Compile the Go test binaries of one shard, and run them.

build    Compile the packages of shard CI_SHARD_INDEX into `.ci/go-tests/build`.
restore  Make `.ci/go-tests/manifest.json` the manifest that `build` used.
run      Run the binaries of shard CI_SHARD_INDEX and print their test2json
         events. gotestsum calls `run`, and calls it again with a test filter
         and a package for each rerun of a failed test.

`go-suite.py` writes the manifest. Binary `<n>.test` belongs to package `n` of
the shard's package list.
"""

from concurrent.futures import ThreadPoolExecutor
import argparse
from datetime import datetime, timezone
import json
import os
import re
import shutil
import signal
from pathlib import Path
import subprocess
import sys
import threading


ROOT = Path(__file__).resolve().parents[3]
MODULE = "github.com/ethereum-optimism/optimism"
MANIFEST = ROOT / ".ci/go-tests/manifest.json"
BUILD = ROOT / ".ci/go-tests/build"
PRINT_LOCK = threading.Lock()
PROCESSES = set()
PROCESS_LOCK = threading.Lock()


def backup_timeout(value):
    # go test grants at least one extra minute beyond the testing package's
    # alarm, bounding TestMain cleanup and inherited output pipes as well.
    units = {'ns': 1e-9, 'us': 1e-6, 'µs': 1e-6, 'ms': 1e-3, 's': 1, 'm': 60, 'h': 3600}
    if value == '0':
        return None
    pattern = r'(\d+(?:\.\d*)?|\.\d+)(ns|us|µs|ms|s|m|h)'
    pieces = re.findall(pattern, value)
    if not pieces or ''.join(number + unit for number, unit in pieces) != value:
        raise ValueError('Invalid Go package timeout')
    seconds = sum(float(number) * units[unit] for number, unit in pieces)
    return seconds + 60 if seconds else None


def build():
    manifest = json.loads(MANIFEST.read_text())
    # A compiler-seed layer carries the manifest of an older commit. This task
    # must compile the packages that the discovery task selected for its commit.
    if manifest["commit_sha"] != os.environ["CI_COMMIT_SHA"]:
        raise ValueError("Go discovery belongs to a different commit")
    packages = manifest["shards"][int(os.environ["CI_SHARD_INDEX"])]
    BUILD.mkdir(parents=True, exist_ok=True)
    subprocess.run(["go", "build", "-o", str(BUILD / "test2json"), "cmd/test2json"], cwd=ROOT, check=True)

    def compile_package(item):
        index, package = item
        output = BUILD / f"{index}.test"
        output.unlink(missing_ok=True)
        # go test -c compiles only. TestMain and init functions never run here.
        subprocess.run(["go", "test", "-c", "-p=4", "-tags=ci", "-o", str(output), package], cwd=ROOT, check=True)
        # For a package without test files, go test -c succeeds and writes no binary.
        if package not in manifest["no_tests"] and not output.is_file():
            raise ValueError(f"Compilation did not produce a test binary: {package}")

    with ThreadPoolExecutor(max_workers=4) as pool:
        list(pool.map(compile_package, enumerate(packages)))
    shutil.copyfile(MANIFEST, BUILD / "manifest.json")


def restore_discovery():
    """Replace the workspace manifest with the one that `build` stored with the binaries.

    The verdict task inherits compiler-seed layers through its compile task.
    Such a layer carries the manifest of an older commit, and that file can
    replace the one from the discovery task.
    """
    shutil.copyfile(BUILD / "manifest.json", MANIFEST)


def emit(line):
    with PRINT_LOCK:
        sys.stdout.write(line)
        sys.stdout.flush()


def cancel(signum, _frame):
    with PROCESS_LOCK:
        for process in PROCESSES:
            try:
                os.killpg(process.pid, signal.SIGTERM)
            except ProcessLookupError:
                pass
    raise SystemExit(128 + signum)


def execute_binary(reporter, binary, package, directory, parallel, timeout, run_filter=None):
    """Run a fresh binary with ordinary Go signal handling and cleanup bounds."""
    delay = backup_timeout(timeout)
    # test2json command mode ignores INT/QUIT before spawning its child.
    # Reset those dispositions before exec, as ordinary go test does;
    # otherwise subprocess fixtures inherit SIG_IGN and cannot stop.
    launch = 'import os,signal,sys; signal.signal(signal.SIGINT,signal.SIG_DFL); signal.signal(signal.SIGQUIT,signal.SIG_DFL); os.execv(sys.argv[1],sys.argv[1:])'
    command = [str(reporter), "-t", "-p", package, sys.executable, "-c", launch, str(binary),
               "-test.v=test2json", "-test.count=1",
               "-test.timeout=" + timeout, "-test.paniconexit0"]
    if parallel is not None:
        command.append("-test.parallel=" + str(parallel))
    if run_filter is not None:
        command.append("-test.run=" + run_filter)
    with PROCESS_LOCK:
        process = subprocess.Popen(command, cwd=directory, stdout=subprocess.PIPE, text=True, start_new_session=True)
        PROCESSES.add(process)
    def expire():
        emit(json.dumps({'Time': datetime.now(timezone.utc).isoformat(), 'Action': 'output', 'Package': package,
                         'Output': '*** Test process exceeded the package timeout plus one-minute cleanup grace.\n'}) + '\n')
        try:
            os.killpg(process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
    watchdog = threading.Timer(delay, expire) if delay is not None else None
    if watchdog:
        watchdog.daemon = True
        watchdog.start()
    try:
        for line in process.stdout:
            emit(line)
        return process.wait()
    finally:
        if watchdog:
            watchdog.cancel()
        process.stdout.close()
        with PROCESS_LOCK:
            PROCESSES.discard(process)

def run(args):
    manifest = json.loads(MANIFEST.read_text())
    packages = manifest["shards"][int(os.environ["CI_SHARD_INDEX"])]
    binaries = {package: None if package in manifest["no_tests"] else BUILD / f"{index}.test"
                for index, package in enumerate(packages)}
    # gotestsum's raw-command reruns append -test.run=<regexp> and a package.
    # Only a rerun may narrow the selection; the initial attempt runs it all.
    run_filter = None
    if args:
        if len(args) != 2 or not args[0].startswith("-test.run=") or args[1] not in binaries:
            raise ValueError("Unexpected gotestsum rerun arguments")
        run_filter, packages = args[0][len("-test.run="):], [args[1]]

    def run_package(package):
        if binaries[package] is None:
            now = datetime.now(timezone.utc).isoformat()
            for action in ("start", "skip"):
                emit(json.dumps({"Time": now, "Action": action, "Package": package}) + "\n")
            return 0
        # go test runs each binary in its package directory, where its fixtures are.
        directory = ROOT / package.removeprefix(MODULE).lstrip("/")
        return execute_binary(BUILD / "test2json", binaries[package], package, directory,
                              os.environ["PARALLEL"], os.environ["TEST_TIMEOUT"], run_filter)

    # At most four packages run at a time, like the recipe's `go test -p=4`.
    with ThreadPoolExecutor(max_workers=4) as pool:
        statuses = list(pool.map(run_package, packages))
    return 1 if any(status != 0 for status in statuses) else 0


if __name__ == "__main__":
    # Preserve the selected executable's SDK, rather than a parent's mise root.
    os.environ.pop('GOROOT', None)
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("command", choices=["build", "restore", "run"])
    options, arguments = parser.parse_known_args()
    try:
        signal.signal(signal.SIGTERM, cancel)
        signal.signal(signal.SIGINT, cancel)
        if options.command == "build":
            build()
        elif options.command == "restore":
            restore_discovery()
        else:
            sys.exit(run(arguments))
    except (ValueError, KeyError, IndexError, OSError, subprocess.CalledProcessError) as error:
        print(f"ERROR: {error}", file=sys.stderr)
        sys.exit(1)
