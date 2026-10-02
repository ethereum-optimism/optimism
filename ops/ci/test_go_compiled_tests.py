#!/usr/bin/env python3
"""Check fresh binary execution, rerun selection, fixtures, and cache integrity."""

import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPTS = Path(__file__).resolve().parent
PREFIX = "github.com/ethereum-optimism/optimism/op-node/rollup"
SPEC = importlib.util.spec_from_file_location("shards", SCRIPTS / "go-package-shards.py")
SHARDS = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SHARDS)


class CompiledGoTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        scripts = self.root / "ops/ci"
        scripts.mkdir(parents=True)
        for name in ("go-compiled-tests.py", "go-package-shards.py"):
            shutil.copy2(SCRIPTS / name, scripts / name)
        self.build = self.root / ".ci/go-rollup/build"
        self.build.mkdir(parents=True)
        self.packages = [PREFIX, PREFIX + "/derive", PREFIX + "/derive/mocks"]
        self.manifest = SHARDS.create_manifest(self.packages, PREFIX, 1)
        (self.build.parent / "manifest.json").write_text(json.dumps(self.manifest))
        self.executable(self.build / "test2json", '''#!/usr/bin/env python3
import os, sys
os.environ['REPORT_PACKAGE'] = sys.argv[3]
os.execv(sys.argv[4], sys.argv[4:])
''')
        binaries = {}
        for index, package in enumerate(self.packages):
            directory = self.root / package.removeprefix("github.com/ethereum-optimism/optimism/")
            directory.mkdir(parents=True, exist_ok=True)
            (directory / "fixture.txt").write_text(package)
            filename = f"{index}.test"
            if package.endswith("/mocks"):
                binaries[package] = {"file": None, "sha256": None}
                continue
            self.executable(self.build / filename, '''#!/usr/bin/env python3
import json, os, pathlib, sys
package = os.environ['REPORT_PACKAGE']
assert pathlib.Path('fixture.txt').read_text() == package
with open(os.environ['BINARY_CALLS'], 'a') as calls:
    calls.write(json.dumps({'package': package, 'args': sys.argv[1:], 'cwd': str(pathlib.Path.cwd())}) + '\\n')
print(json.dumps({'Action': 'run', 'Package': package, 'Test': 'TestFresh'}))
failed = os.environ.get('FAIL_PACKAGE') == package
print(json.dumps({'Action': 'fail' if failed else 'pass', 'Package': package, 'Test': 'TestFresh'}))
print(json.dumps({'Action': 'fail' if failed else 'pass', 'Package': package, 'Elapsed': 0.01}))
sys.exit(1 if failed else 0)
''')
            binaries[package] = {"file": filename, "sha256": self.digest(self.build / filename)}
        self.metadata = {"commit_sha": "a" * 40, "tags": ["ci"], "packages": binaries,
                         "test2json_sha256": self.digest(self.build / "test2json")}
        self.save_metadata()
        self.calls = self.root / "calls.jsonl"
        self.env = {**os.environ, "CI_COMMIT_SHA": "a" * 40, "CI_SHARD_TOTAL": "1", "CI_SHARD_INDEX": "0",
                    "PARALLEL": "8", "TEST_TIMEOUT": "40m", "BINARY_CALLS": str(self.calls)}

    def executable(self, path, source):
        path.write_text(source)
        path.chmod(0o755)

    def digest(self, path):
        return hashlib.sha256(path.read_bytes()).hexdigest()

    def save_metadata(self):
        (self.build / "metadata.json").write_text(json.dumps(self.metadata))

    def run_helper(self, *args, **env):
        return subprocess.run(["python3", str(self.root / "ops/ci/go-compiled-tests.py"), *args],
                              env={**self.env, **env}, capture_output=True, text=True)

    def test_every_attempt_executes_binaries_and_preserves_flags_cwd_and_no_test_packages(self):
        for _ in range(2):
            result = self.run_helper("run")
            self.assertEqual(result.returncode, 0, result.stderr)
            events = [json.loads(line) for line in result.stdout.splitlines()]
            self.assertEqual(sorted({event["Package"] for event in events}), self.packages)
        calls = [json.loads(line) for line in self.calls.read_text().splitlines()]
        self.assertEqual(len(calls), 4)
        for call in calls:
            self.assertIn("-test.count=1", call["args"])
            self.assertIn("-test.parallel=8", call["args"])
            self.assertIn("-test.timeout=40m", call["args"])
            self.assertIn("-test.paniconexit0", call["args"])
            self.assertFalse(any(arg.startswith("-test.run") for arg in call["args"]))

    def test_failures_are_retained_and_only_gotestsum_reruns_can_select_cases(self):
        failed = self.run_helper("run", FAIL_PACKAGE=self.packages[1])
        self.assertEqual(failed.returncode, 1)
        self.assertIn('"Action": "fail"', failed.stdout)
        self.calls.unlink()
        rerun = self.run_helper("run", "-test.run=^TestFresh$/^nested$", self.packages[1])
        self.assertEqual(rerun.returncode, 0, rerun.stderr)
        calls = [json.loads(line) for line in self.calls.read_text().splitlines()]
        self.assertEqual(len(calls), 1)
        self.assertEqual(calls[0]["package"], self.packages[1])
        self.assertIn("-test.run=^TestFresh$/^nested$", calls[0]["args"])

    def test_changed_sha_tags_binaries_or_package_coverage_fail_before_execution(self):
        for key, value in (("commit_sha", "b" * 40), ("tags", []), ("packages", {})):
            original = self.metadata[key]
            self.metadata[key] = value
            self.save_metadata()
            with self.subTest(key=key):
                self.assertNotEqual(self.run_helper("run").returncode, 0)
                self.assertFalse(self.calls.exists())
            self.metadata[key] = original
        self.save_metadata()
        with (self.build / "0.test").open("a") as binary:
            binary.write("# changed artifact\n")
        self.assertNotEqual(self.run_helper("run").returncode, 0)
        self.assertFalse(self.calls.exists())

    def test_invalid_rerun_scope_and_malformed_manifests_fail_closed(self):
        for args in (("-run=TestFresh",), ("-run=TestFresh", PREFIX + "/outside"), ("-short", self.packages[0])):
            with self.subTest(args=args):
                self.assertNotEqual(self.run_helper("run", *args).returncode, 0)
                self.assertFalse(self.calls.exists())
        self.manifest["shards"][0].pop()
        (self.build.parent / "manifest.json").write_text(json.dumps(self.manifest))
        self.assertNotEqual(self.run_helper("run").returncode, 0)
        self.assertFalse(self.calls.exists())

    @unittest.skipUnless(shutil.which("gotestsum"), "Pinned gotestsum is exercised in the hosted Go helper task")
    def test_actual_gotestsum_preserves_three_failed_reruns_and_reports(self):
        report = self.root / "attempts.json"
        junit = self.root / "junit.xml"
        result = subprocess.run(["gotestsum", "--rerun-fails=3", "--rerun-fails-max-failures=50",
                                 "--jsonfile=" + str(report), "--junitfile=" + str(junit), "--raw-command", "--",
                                 "python3", str(self.root / "ops/ci/go-compiled-tests.py"), "run"],
                                env={**self.env, "FAIL_PACKAGE": self.packages[1]}, text=True, capture_output=True)
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        calls = [json.loads(line) for line in self.calls.read_text().splitlines()]
        failed_calls = [call for call in calls if call["package"] == self.packages[1]]
        self.assertEqual(len(failed_calls), 4, result.stdout + result.stderr)
        for call in failed_calls[1:]:
            self.assertIn("-test.run=^TestFresh$", call["args"])
        events = [json.loads(line) for line in report.read_text().splitlines()]
        failures = [event for event in events if event.get("Test") == "TestFresh" and event["Action"] == "fail"]
        self.assertEqual(len(failures), 4)
        self.assertTrue(junit.is_file())


if __name__ == "__main__":
    unittest.main()
