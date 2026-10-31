#!/usr/bin/env python3
"""Run freshly executed Go tests from cached, source-bound test binaries."""

from concurrent.futures import ThreadPoolExecutor
import argparse
from datetime import datetime, timezone
import hashlib
import importlib.util
import json
import os
import re
import signal
from pathlib import Path
import subprocess
import sys
import threading
import tarfile


ROOT = Path(__file__).resolve().parents[3]
BUILD = ROOT / ".ci/go-rollup/build"
PREFIX = "github.com/ethereum-optimism/optimism/op-node/rollup"
SUITE = "go-rollup"
DISCOVERY_FILES = ("manifest.json", "go-list.json", "all-packages.txt")
SPEC = importlib.util.spec_from_file_location("shards", Path(__file__).with_name("go-package-shards.py"))
SHARDS = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SHARDS)
SUITE_SPEC = importlib.util.spec_from_file_location("suite_helper", Path(__file__).with_name("go-suite.py"))
SUITE_HELPER = importlib.util.module_from_spec(SUITE_SPEC)
SUITE_SPEC.loader.exec_module(SUITE_HELPER)
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


def digest(path):
    with path.open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def go_objects():
    source = (ROOT / ".ci" / SUITE / "go-list.json").read_text()
    decoder, offset, objects = json.JSONDecoder(), 0, {}
    while offset < len(source):
        if source[offset].isspace():
            offset += 1
            continue
        item, offset = decoder.raw_decode(source, offset)
        if item.get("Error") or item.get("DepsErrors") or item["ImportPath"] in objects:
            raise ValueError("Invalid tagged package discovery")
        objects[item["ImportPath"]] = item
    return objects


def full_settings():
    return SUITE_HELPER.settings()


def check_environment(manifest):
    head = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    if head != manifest.get('commit_sha') or head != os.environ['CI_COMMIT_SHA']:
        raise ValueError('Go checkout differs from requested compilation/verdict source')
    if subprocess.run(['git', 'diff', '--quiet', 'HEAD', '--'], cwd=ROOT).returncode:
        raise ValueError('Go tracked source changed before compilation/verdict')
    if manifest.get('environment') != SUITE_HELPER.environment() or manifest.get('go_environment') != SUITE_HELPER.go_environment():
        raise ValueError('Go compiler flags or test environment differ from discovery')


def build():
    manifest = json.loads((ROOT / ".ci" / SUITE / "manifest.json").read_text())
    SHARDS.select_packages(manifest, PREFIX, 0, manifest["total"])
    packages = manifest["packages"]
    if SUITE == "go-tests":
        packages = SHARDS.select_packages(manifest, PREFIX, os.environ["CI_SHARD_INDEX"], manifest["total"])
    if SUITE == "go-tests" and (manifest.get("suite") != SUITE or manifest.get("settings") != full_settings()
                                or manifest.get("commit_sha") != os.environ["CI_COMMIT_SHA"]):
        raise ValueError("Invalid compilation suite settings or revision")
    if SUITE == "go-tests":
        check_environment(manifest)
    BUILD.mkdir(parents=True, exist_ok=True)
    subprocess.run(["go", "build", "-o", str(BUILD / "test2json"), "cmd/test2json"], cwd=ROOT, check=True)
    # go test -c compiles only. TestMain and init functions never run here.
    objects = go_objects()
    cache_proofs = {}
    if sorted(objects) != manifest["packages"]:
        raise ValueError("Tagged discovery differs from compilation selection")

    def compile_package(item):
        index, package = item
        output = BUILD / f"{index}.test"
        output.unlink(missing_ok=True)
        command = ["go", "test", "-c", "-p=4", "-tags=ci", "-o", str(output), package]
        if SUITE == 'go-tests':
            # -x records actual compiler/linker subprocesses without changing
            # compilation or freshness. Keep original diagnostics on failure.
            result = subprocess.run(command[:3] + ['-x'] + command[3:], cwd=ROOT, text=True, capture_output=True)
            (BUILD / f'{index}.compile.log').write_text(result.stderr)
            cache_proofs[package] = {
                'compile_commands': len(re.findall(r'^.*?/compile -o ', result.stderr, re.MULTILINE)),
                'link_commands': len(re.findall(r'^.*?/link -o ', result.stderr, re.MULTILINE)),
                'log': f'{index}.compile.log',
            }
            if result.returncode:
                sys.stderr.write(result.stdout + result.stderr)
                raise subprocess.CalledProcessError(result.returncode, command)
        else:
            subprocess.run(command, cwd=ROOT, check=True)
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
        metadata.update(cache_proofs=cache_proofs, suite=SUITE, settings=manifest["settings"],
                        environment=manifest["environment"], go_environment=manifest["go_environment"],
                        shard_index=int(os.environ["CI_SHARD_INDEX"]), shard_total=manifest["total"],
                        manifest_sha256=digest(ROOT / ".ci" / SUITE / "manifest.json"),
                        compile_root=str(ROOT))
    if SUITE == 'go-tests' and PREFIX + '/op-deployer/pkg/deployer/integration_test/cli' in packages:
        # CLI TestMain otherwise recompiles the same helper in every execution
        # group with an empty runtime compiler cache. Handoff only the binary.
        subprocess.run(['go', 'build', '-o', str(BUILD / 'op-deployer'), './op-deployer/cmd/op-deployer'],
                       cwd=ROOT, check=True)
        metadata['cli_helper_sha256'] = digest(BUILD / 'op-deployer')
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    if head != metadata["commit_sha"]:
        raise ValueError("Go compilation belongs to a different commit")
    if SUITE == 'go-tests':
        metadata['discovery_sha256'] = {}
        for name in DISCOVERY_FILES:
            source = ROOT / '.ci' / SUITE / name
            (BUILD / name).write_bytes(source.read_bytes())
            metadata['discovery_sha256'][name] = digest(source)
    else:
        (BUILD / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    (BUILD / "metadata.json").write_text(json.dumps(metadata, indent=2) + "\n")
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


def restore_discovery():
    """Rebind discovery after inherited compiler-seed layers have been assembled."""
    if SUITE != 'go-tests':
        raise ValueError('Discovery restoration requires the full Go suite')
    metadata = json.loads((BUILD / 'metadata.json').read_text())
    hashes = metadata.get('discovery_sha256', {})
    if metadata.get('commit_sha') != os.environ['CI_COMMIT_SHA'] or set(hashes) != set(DISCOVERY_FILES):
        raise ValueError('Compiled discovery source or file coverage differs')
    for name in DISCOVERY_FILES:
        if digest(BUILD / name) != hashes[name]:
            raise ValueError('Compiled discovery file changed: ' + name)
    check_environment(json.loads((BUILD / 'manifest.json').read_text()))
    for name in DISCOVERY_FILES:
        (ROOT / '.ci' / SUITE / name).write_bytes((BUILD / name).read_bytes())


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
        check_environment(manifest)
        hashes = metadata.get('discovery_sha256', {})
        if set(hashes) != set(DISCOVERY_FILES) or any(digest(ROOT / '.ci' / SUITE / name) != hashes[name] for name in DISCOVERY_FILES):
            raise ValueError('Go runtime discovery differs from the sealed compilation inputs')
        expected_settings = full_settings()
        if (metadata.get("suite") != SUITE or metadata.get("settings") != expected_settings
                or metadata.get("environment") != manifest["environment"]
                or metadata.get("go_environment") != manifest["go_environment"]
                or manifest.get("settings") != expected_settings
                or manifest.get("commit_sha") != os.environ["CI_COMMIT_SHA"]
                or metadata.get("shard_index") != int(os.environ["CI_SHARD_INDEX"])
                or metadata.get("shard_total") != int(os.environ["CI_SHARD_TOTAL"])
                or metadata.get("manifest_sha256") != digest(manifest_path)
                or metadata.get("compile_root") != str(ROOT)):
            raise ValueError("Go suite settings, shard, source paths or manifest differ from this verdict")
    if SUITE == "go-tests" and metadata.get("go_version") != subprocess.check_output(["go", "version"], text=True).strip():
        raise ValueError("Go verdict toolchain differs from compilation")
    if digest(BUILD / "test2json") != metadata["test2json_sha256"]:
        raise ValueError("Go JSON reporter changed")
    objects = go_objects() if SUITE == "go-tests" else {}
    for package in selected:
        binary = metadata["packages"][package]
        if binary["file"] is not None and not re.fullmatch(r"[0-9]+\.test", binary["file"]):
            raise ValueError("Unsafe compiled binary filename")
        if SUITE == "go-tests":
            item = objects[package]
            if bool(item.get("TestGoFiles") or item.get("XTestGoFiles")) != (binary["file"] is not None):
                raise ValueError("Missing test binary or invalid no-test marker")
        if binary["file"] is not None and digest(BUILD / binary["file"]) != binary["sha256"]:
            raise ValueError(f"Go test binary changed: {package}")
    if SUITE == 'go-tests' and PREFIX + '/op-deployer/pkg/deployer/integration_test/cli' in selected:
        if not metadata.get('cli_helper_sha256') or digest(BUILD / 'op-deployer') != metadata['cli_helper_sha256']:
            raise ValueError('CLI runtime helper changed or is missing')
    return metadata, selected


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


def binary_command(reporter, binary, package, parallel, timeout, run_filter=None, extra_args=()):
    """Use one argument builder for execution and retained command provenance."""
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
    command.extend(extra_args)
    return command


def execute_binary(reporter, binary, package, directory, parallel, timeout, run_filter=None, extra_args=()):
    """Run a fresh binary with ordinary Go signal handling and cleanup bounds."""
    delay = backup_timeout(timeout)
    command = binary_command(reporter, binary, package, parallel, timeout, run_filter, extra_args)
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
    metadata, selected = verify()
    if metadata.get('cli_helper_sha256'):
        os.environ['OP_DEPLOYER_TEST_BINARY'] = str(BUILD / 'op-deployer')
    # gotestsum's raw-command reruns append -test.run=<regexp> and a package.
    # Only a rerun may narrow the selection; the initial attempt runs it all.
    run_filter = None
    if args:
        if len(args) != 2 or not args[0].startswith("-test.run=") or args[1] not in selected:
            raise ValueError("Unexpected gotestsum rerun arguments")
        run_filter, selected = args[0][len("-test.run="):], [args[1]]

    manifest = json.loads((ROOT / '.ci' / SUITE / 'manifest.json').read_text())
    units = manifest.get('work_shards', [[]])[int(os.environ['CI_SHARD_INDEX'])] if manifest.get('work_shards') else []
    assigned_tests = {unit['package']: unit['tests'] for unit in units if unit['tests'] is not None}

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
        package_filter = run_filter
        names = assigned_tests.get(package)
        if names is not None:
            if run_filter is None:
                package_filter = '^(' + '|'.join(re.escape(name) for name in names) + ')$'
            else:
                root_filter = run_filter.split('/')[0]
                all_names = manifest['split_tests'][package]
                if (not any(re.search(root_filter, name) for name in names)
                        or any(re.search(root_filter, name) for name in set(all_names) - set(names))):
                    raise ValueError('Retry would escape the assigned test group')
        return execute_binary(BUILD / "test2json", BUILD / binary, package, directory,
                              os.environ["PARALLEL"], os.environ.get("TEST_TIMEOUT", "40m"), package_filter)

    with ThreadPoolExecutor(max_workers=4) as pool:
        statuses = list(pool.map(run_package, selected))
    return 1 if any(status != 0 for status in statuses) else 0


if __name__ == "__main__":
    # Preserve the selected executable's SDK, rather than a parent's mise root.
    os.environ.pop('GOROOT', None)
    try:
        parser = argparse.ArgumentParser(description=__doc__)
        parser.add_argument("command", choices=["build", "restore", "verify", "run"])
        parser.add_argument("--suite", choices=["go-rollup", "go-tests"], default="go-rollup")
        options, arguments = parser.parse_known_args()
        command, SUITE = options.command, options.suite
        BUILD = ROOT / ".ci" / SUITE / "build"
        if SUITE == "go-tests":
            PREFIX = "github.com/ethereum-optimism/optimism"
        if command != "run" and arguments:
            raise ValueError("Unexpected compilation or verification arguments")
        signal.signal(signal.SIGTERM, cancel)
        signal.signal(signal.SIGINT, cancel)
        if command == "build":
            build()
        elif command == "restore":
            restore_discovery()
            verify()
        elif command == "verify":
            verify()
        elif command == "run":
            sys.exit(run(arguments))
        else:
            raise ValueError("Expected build, restore, verify, or run")
    except (ValueError, KeyError, OSError, subprocess.CalledProcessError) as error:
        print(f"ERROR: {error}", file=sys.stderr)
        sys.exit(1)
