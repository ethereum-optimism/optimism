#!/usr/bin/env python3
"""Coverage and failure propagation checks for the bounded Go shadow."""

import copy
import importlib.util
import json
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[3]
PREFIX = "github.com/ethereum-optimism/optimism/op-node/rollup"
SPEC = importlib.util.spec_from_file_location("go_package_shards", Path(__file__).resolve().parents[1] / 'runtime' / 'go-package-shards.py')
SHARDS = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SHARDS)


class PackageShardsTest(unittest.TestCase):
    def setUp(self):
        self.packages = [PREFIX, PREFIX + "/derive", PREFIX + "/derive/mocks", PREFIX + "/engine", PREFIX + "/sync"]

    def test_whole_manifest_is_assigned_exactly_once(self):
        for total in (1, 2, 3, 8):
            with self.subTest(total=total):
                manifest = SHARDS.create_manifest(self.packages, PREFIX, total)
                shares = [SHARDS.select_packages(manifest, PREFIX, index, total) for index in range(total)]
                flattened = [package for share in shares for package in share]
                self.assertEqual(sorted(flattened), sorted(self.packages))
                self.assertEqual(len(flattened), len(set(flattened)))

    def test_order_is_stable_and_packages_without_tests_are_retained(self):
        objects = [{"ImportPath": package, "TestGoFiles": ["test.go"]} for package in reversed(self.packages)]
        objects[2].pop("TestGoFiles")
        packages = SHARDS.read_go_packages("\n".join(json.dumps(item) for item in objects), PREFIX)
        self.assertEqual(packages, sorted(self.packages))
        self.assertEqual(SHARDS.create_manifest(packages, PREFIX, 2), SHARDS.create_manifest(list(reversed(packages)), PREFIX, 2))

    def test_timings_balance_long_packages_and_retain_new_packages(self):
        durations = {self.packages[0]: 30, self.packages[1]: 12, self.packages[2]: 6, self.packages[3]: 2}
        manifest = SHARDS.create_manifest(self.packages, PREFIX, 4, durations)
        shares = [SHARDS.select_packages(manifest, PREFIX, index, 4) for index in range(4)]
        self.assertEqual(sorted(package for share in shares for package in share), sorted(self.packages))
        self.assertEqual(manifest["durations"][self.packages[-1]], 9)
        self.assertEqual(shares[0], [self.packages[0]])
        self.assertEqual(manifest, SHARDS.create_manifest(list(reversed(self.packages)), PREFIX, 4, durations))

    def test_invalid_or_changed_timings_cannot_hide_coverage(self):
        for value in (True, -1, float("inf"), float("nan"), "1"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                SHARDS.create_manifest(self.packages, PREFIX, 4, {self.packages[0]: value})
        manifest = SHARDS.create_manifest(self.packages, PREFIX, 2, {self.packages[0]: 30})
        manifest["shards"][0].clear()
        with self.assertRaises(ValueError):
            SHARDS.select_packages(manifest, PREFIX, 0, 2)

    def test_go_list_package_and_dependency_errors_fail_closed(self):
        for error in ({"Error": {"Err": "compile error"}}, {"DepsErrors": [{"Err": "missing embed"}]}):
            with self.subTest(error=error):
                with self.assertRaisesRegex(ValueError, "go list reported an error"):
                    SHARDS.read_go_packages(json.dumps({"ImportPath": PREFIX, **error}), PREFIX)

    def test_invalid_go_list_streams_fail_closed(self):
        streams = ("", "   ", "[]", "{}", "{", json.dumps({"ImportPath": PREFIX + "/../../elsewhere"}),
                   json.dumps({"ImportPath": PREFIX + "-other"}), json.dumps({"ImportPath": PREFIX + " unsafe"}),
                   json.dumps({"ImportPath": PREFIX}) * 2)
        for stream in streams:
            with self.subTest(stream=stream), self.assertRaises(ValueError):
                SHARDS.read_go_packages(stream, PREFIX)

    def test_invalid_totals_and_indexes_fail_closed(self):
        for total in ("0", "-1", "01", "1.0", "", "257", "true"):
            with self.subTest(total=total), self.assertRaises(ValueError):
                SHARDS.create_manifest(self.packages, PREFIX, total)
        manifest = SHARDS.create_manifest(self.packages, PREFIX, 2)
        for index in ("-1", "2", "01", "", "1.5"):
            with self.subTest(index=index), self.assertRaises(ValueError):
                SHARDS.select_packages(manifest, PREFIX, index, 2)

    def test_missing_duplicate_extra_or_reordered_assignments_fail(self):
        manifest = SHARDS.create_manifest(self.packages, PREFIX, 2)
        corrupt = []
        missing = copy.deepcopy(manifest)
        missing["shards"][0].pop()
        corrupt.append(missing)
        duplicate = copy.deepcopy(manifest)
        duplicate["shards"][1].append(duplicate["shards"][0][0])
        corrupt.append(duplicate)
        extra = copy.deepcopy(manifest)
        extra["shards"].append([])
        corrupt.append(extra)
        order = copy.deepcopy(manifest)
        order["packages"].reverse()
        corrupt.append(order)
        for broken in corrupt:
            with self.subTest(broken=broken), self.assertRaises(ValueError):
                SHARDS.select_packages(broken, PREFIX, 0, 2)

    def test_scope_total_version_and_duplicate_manifest_packages_fail(self):
        manifest = SHARDS.create_manifest(self.packages, PREFIX, 2)
        for field, value in (("prefix", PREFIX + "/derive"), ("total", 3), ("total", True), ("version", 2),
                             ("packages", self.packages + [self.packages[0]])):
            broken = {**manifest, field: value}
            with self.subTest(field=field, value=value), self.assertRaises(ValueError):
                SHARDS.select_packages(broken, PREFIX, 0, 2)


class RollupRunnerTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for relative in ("ops/ci/runtime/go-rollup-tests.sh", "ops/ci/runtime/go-package-shards.py", "ops/ci/runtime/go-rollup-timings.json", "ops/scripts/gotestsum-split.sh", "ops/scripts/split-test-logs.sh"):
            target = self.root / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(REPO_ROOT / relative, target)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.packages = [PREFIX, PREFIX + "/derive", PREFIX + "/derive/mocks", PREFIX + "/engine", PREFIX + "/sync"]
        self.go_json = self.root / "go-list-input.json"
        self.go_json.write_text("\n".join(json.dumps({"ImportPath": package}) for package in reversed(self.packages)))
        self.write_executable("go", "#!/bin/sh\nprintf '%s\\n' \"$@\" >\"$GO_ARGS\"\ncat \"$GO_LIST_INPUT\"\nexit \"${GO_EXIT:-0}\"\n")
        self.write_executable("gotestsum", """#!/usr/bin/env python3
import json, os, pathlib, sys
args = sys.argv[1:]
pathlib.Path(os.environ['GOTESTSUM_ARGS']).write_text(json.dumps(args))
for arg in args:
    if arg.startswith('--junitfile='):
        pathlib.Path(arg.split('=', 1)[1]).write_text('<testsuites/>')
    if arg.startswith('--jsonfile='):
        events = [{'Action': 'run', 'Package': 'fixture', 'Test': 'TestExample'}, {'Action': 'output', 'Package': 'fixture', 'Test': 'TestExample', 'Output': 'failure detail\\n'}]
        pathlib.Path(arg.split('=', 1)[1]).write_text('\\n'.join(json.dumps(event) for event in events) + '\\n')
sys.exit(int(os.environ.get('GOTESTSUM_EXIT', '0')))
""")
        self.env = {key: value for key, value in os.environ.items() if not key.startswith("CIRCLE") and "RPC_URL" not in key}
        self.env.update(PATH=str(self.bin) + os.pathsep + os.environ["PATH"], GO_ARGS=str(self.root / "go-args.txt"),
                        GO_LIST_INPUT=str(self.go_json), GOTESTSUM_ARGS=str(self.root / "gotestsum-args.json"),
                        CI_SHARD_TOTAL="2", CI_SHARD_INDEX="0")

    def write_executable(self, name, source):
        target = self.bin / name
        target.write_text(source)
        target.chmod(0o755)

    def run_helper(self, mode="run", **env):
        return subprocess.run(["bash", str(self.root / "ops/ci/runtime/go-rollup-tests.sh"), mode], env={**self.env, **env}, text=True, capture_output=True)

    def prepare(self):
        result = self.run_helper("prepare")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.root / "go-args.txt").read_text().splitlines(), ["list", "-e", "-tags=ci", "-json", "./op-node/rollup/..."])

    def test_runner_preserves_complete_coverage_flags_retries_and_reports(self):
        self.prepare()
        seen = []
        for index in (0, 1):
            result = self.run_helper(CI_SHARD_INDEX=str(index))
            self.assertEqual(result.returncode, 0, result.stderr)
            args = json.loads((self.root / "gotestsum-args.json").read_text())
            seen.extend(next(arg for arg in args if arg.startswith("--packages=")).split("=", 1)[1].split())
            for flag in ("--format=standard-verbose", "--rerun-fails=3", "--rerun-fails-max-failures=50", "-count=1", "-p=4", "-timeout=40m", "-tags=ci"):
                self.assertIn(flag, args)
            self.assertFalse(any(arg in ("-short", "-run") or arg.startswith("-run=") for arg in args))
            self.assertTrue((self.root / f"tmp/test-results/results-{index}.xml").is_file())
            self.assertTrue((self.root / "tmp/testlogs/log.json").is_file())
            self.assertEqual((self.root / "tmp/testlogs/per-test/fixture/TestExample.log").read_text(), "failure detail\n")
        self.assertEqual(sorted(seen), sorted(self.packages))
        self.assertEqual(len(seen), len(set(seen)))

    def test_failed_tests_keep_exit_status_and_failure_logs(self):
        self.prepare()
        result = self.run_helper(GOTESTSUM_EXIT="23")
        self.assertEqual(result.returncode, 23, result.stderr)
        self.assertTrue((self.root / "tmp/test-results/results-0.xml").is_file())
        self.assertIn("failure detail", (self.root / "tmp/testlogs/per-test/fixture/TestExample.log").read_text())

    def test_go_command_failure_and_hidden_dependency_errors_stop_preparation(self):
        result = self.run_helper("prepare", GO_EXIT="7")
        self.assertEqual(result.returncode, 7)
        self.assertFalse((self.root / ".ci/go-rollup/manifest.json").exists())
        self.go_json.write_text(json.dumps({"ImportPath": PREFIX, "DepsErrors": [{"Err": "missing embed"}]}))
        result = self.run_helper("prepare")
        self.assertEqual(result.returncode, 1)
        self.assertIn("go list reported an error", result.stderr)
        self.assertFalse((self.root / ".ci/go-rollup/manifest.json").exists())

    def test_missing_manifest_and_invalid_index_cannot_run_tests(self):
        self.assertNotEqual(self.run_helper().returncode, 0)
        self.prepare()
        self.assertNotEqual(self.run_helper(CI_SHARD_INDEX="2").returncode, 0)
        self.assertFalse((self.root / "gotestsum-args.json").exists())

    def test_valid_empty_shard_is_explicit_and_does_not_run_tests(self):
        self.go_json.write_text(json.dumps({"ImportPath": PREFIX}))
        self.prepare()
        result = self.run_helper(CI_SHARD_INDEX="1")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("No packages assigned", result.stdout)
        self.assertFalse((self.root / "gotestsum-args.json").exists())


if __name__ == "__main__":
    unittest.main()
