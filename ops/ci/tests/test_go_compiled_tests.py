#!/usr/bin/env python3
"""Check compilation of a shard, binary execution, rerun selection and cancellation."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import signal
import time
import tempfile
import unittest


SCRIPTS = Path(__file__).resolve().parents[1] / 'runtime'
MODULE = "github.com/ethereum-optimism/optimism"
PREFIX = MODULE + "/op-node/rollup"
# Stands in for the Go command: `go build -o <file> cmd/test2json` and `go test -c ... -o <file> <package>`.
GO = '''#!/usr/bin/env python3
import json, os, pathlib, sys
args = sys.argv[1:]
with open(os.environ['GO_CALLS'], 'a') as calls:
    calls.write(json.dumps({'args': args, 'goroot': os.environ.get('GOROOT')}) + '\\n')
package = args[-1]
if package == os.environ.get('GO_FAIL_PACKAGE'):
    sys.exit(1)
if not package.endswith('/mocks') and package != os.environ.get('GO_NO_BINARY_PACKAGE'):
    pathlib.Path(args[args.index('-o') + 1]).write_text('compiled ' + package)
'''


class CompiledGoTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        scripts = self.root / "ops/ci/runtime"
        scripts.mkdir(parents=True)
        shutil.copy2(SCRIPTS / "go-compiled-tests.py", scripts / "go-compiled-tests.py")
        self.build = self.root / ".ci/go-tests/build"
        self.build.mkdir(parents=True)
        self.packages = [PREFIX, PREFIX + "/derive", PREFIX + "/derive/mocks"]
        self.manifest = {"commit_sha": "a" * 40, "shards": [self.packages], "no_tests": [self.packages[2]]}
        self.save_manifest()
        self.executable(self.build / "test2json", '''#!/usr/bin/env python3
import os, signal, sys
signal.signal(signal.SIGINT, signal.SIG_IGN)
signal.signal(signal.SIGQUIT, signal.SIG_IGN)
os.environ['REPORT_PACKAGE'] = sys.argv[3]
os.execv(sys.argv[4], sys.argv[4:])
''')
        for index, package in enumerate(self.packages):
            directory = self.root / package.removeprefix(MODULE + "/")
            directory.mkdir(parents=True, exist_ok=True)
            (directory / "fixture.txt").write_text(package)
            if package in self.manifest["no_tests"]:
                continue
            self.executable(self.build / f"{index}.test", '''#!/usr/bin/env python3
import json, os, pathlib, sys
package = os.environ['REPORT_PACKAGE']
assert pathlib.Path('fixture.txt').read_text() == package
with open(os.environ['BINARY_CALLS'], 'a') as calls:
    calls.write(json.dumps({'package': package, 'args': sys.argv[1:], 'goroot': os.environ.get('GOROOT')}) + '\\n')
print(json.dumps({'Action': 'run', 'Package': package, 'Test': 'TestFresh'}))
failed = os.environ.get('FAIL_PACKAGE') == package
print(json.dumps({'Action': 'fail' if failed else 'pass', 'Package': package, 'Test': 'TestFresh'}))
print(json.dumps({'Action': 'fail' if failed else 'pass', 'Package': package, 'Elapsed': 0.01}))
sys.exit(1 if failed else 0)
''')
        tools = self.root / "bin"
        tools.mkdir()
        self.executable(tools / "go", GO)
        self.calls = self.root / "calls.jsonl"
        self.go_calls = self.root / "go-calls.jsonl"
        self.env = {**os.environ, "CI_COMMIT_SHA": "a" * 40, "CI_SHARD_INDEX": "0", "PARALLEL": "8", "TEST_TIMEOUT": "40m",
                    "BINARY_CALLS": str(self.calls), "GO_CALLS": str(self.go_calls),
                    "PATH": str(tools) + os.pathsep + os.environ["PATH"]}

    def executable(self, path, source):
        path.write_text(source)
        path.chmod(0o755)

    def save_manifest(self):
        (self.build.parent / "manifest.json").write_text(json.dumps(self.manifest))

    def command(self, *args):
        return ["python3", str(self.root / "ops/ci/runtime/go-compiled-tests.py"), *args]

    def run_helper(self, *args, **env):
        return subprocess.run(self.command(*args), env={**self.env, **env}, capture_output=True, text=True)

    def recorded(self, path):
        return [json.loads(line) for line in path.read_text().splitlines()]

    def test_build_compiles_the_shard_and_keeps_its_manifest(self):
        for binary in self.build.glob("*.test"):
            binary.unlink()
        (self.build / "2.test").write_text("binary of an earlier shard layout")
        result = self.run_helper("build", GOROOT="/stale-parent-sdk")
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = self.recorded(self.go_calls)
        self.assertEqual(calls[0]["args"], ["build", "-o", str(self.build / "test2json"), "cmd/test2json"])
        self.assertEqual(sorted(call["args"] for call in calls[1:]),
                         [["test", "-c", "-p=4", "-tags=ci", "-o", str(self.build / f"{index}.test"), package]
                          for index, package in enumerate(self.packages)])
        self.assertEqual([call["goroot"] for call in calls], [None] * 4)
        self.assertEqual(sorted(path.name for path in self.build.glob("*.test")), ["0.test", "1.test"])
        self.assertEqual(json.loads((self.build / "manifest.json").read_text()), self.manifest)
        self.assertFalse(self.calls.exists(), "Compilation must run no test binary")

    def test_build_fails_for_another_commits_manifest_a_compile_error_or_a_missing_binary(self):
        result = self.run_helper("build", CI_COMMIT_SHA="b" * 40)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("different commit", result.stderr)
        self.assertFalse(self.go_calls.exists())
        for variable in ("GO_FAIL_PACKAGE", "GO_NO_BINARY_PACKAGE"):
            with self.subTest(variable=variable):
                self.assertNotEqual(self.run_helper("build", **{variable: self.packages[1]}).returncode, 0)
        self.assertFalse((self.build / "manifest.json").exists())

    def test_restore_replaces_an_older_manifest_with_the_compiled_one(self):
        shutil.copyfile(self.build.parent / "manifest.json", self.build / "manifest.json")
        self.manifest = {"commit_sha": "0" * 40, "shards": [[PREFIX + "/removed"]], "no_tests": []}
        self.save_manifest()
        self.assertNotEqual(self.run_helper("run").returncode, 0)
        self.assertFalse(self.calls.exists())
        result = self.run_helper("restore")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.build.parent / "manifest.json").read_bytes(), (self.build / "manifest.json").read_bytes())
        self.assertEqual(self.run_helper("run").returncode, 0)
        self.assertEqual(len(self.recorded(self.calls)), 2)

    def test_every_attempt_executes_binaries_and_preserves_flags_cwd_and_no_test_packages(self):
        for _ in range(2):
            result = self.run_helper("run", GOROOT="/stale-parent-sdk")
            self.assertEqual(result.returncode, 0, result.stderr)
            events = [json.loads(line) for line in result.stdout.splitlines()]
            self.assertEqual(sorted({event["Package"] for event in events}), self.packages)
            self.assertEqual([event["Action"] for event in events if event["Package"] == self.packages[2]], ["start", "skip"])
        calls = self.recorded(self.calls)
        self.assertEqual(len(calls), 4)
        for call in calls:
            self.assertIn("-test.count=1", call["args"])
            self.assertIn("-test.parallel=8", call["args"])
            self.assertIn("-test.timeout=40m", call["args"])
            self.assertIn("-test.paniconexit0", call["args"])
            self.assertFalse(any(arg.startswith("-test.run") for arg in call["args"]))
            # A test binary that runs `go` must find the SDK of the go on PATH.
            self.assertIsNone(call["goroot"])

    def test_run_executes_only_its_own_shard(self):
        self.manifest["shards"] = [[self.packages[2]], self.packages[:2]]
        self.save_manifest()
        result = self.run_helper("run", CI_SHARD_INDEX="1")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(sorted(call["package"] for call in self.recorded(self.calls)), self.packages[:2])
        self.calls.unlink()
        result = self.run_helper("run", CI_SHARD_INDEX="0")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual({json.loads(line)["Package"] for line in result.stdout.splitlines()}, {self.packages[2]})
        self.assertFalse(self.calls.exists())
        self.assertNotEqual(self.run_helper("run", CI_SHARD_INDEX="2").returncode, 0)

    def test_failures_are_retained_and_only_gotestsum_reruns_can_select_cases(self):
        failed = self.run_helper("run", FAIL_PACKAGE=self.packages[1])
        self.assertEqual(failed.returncode, 1)
        self.assertIn('"Action": "fail"', failed.stdout)
        self.calls.unlink()
        rerun = self.run_helper("run", "-test.run=^TestFresh$/^nested$", self.packages[1])
        self.assertEqual(rerun.returncode, 0, rerun.stderr)
        calls = self.recorded(self.calls)
        self.assertEqual(len(calls), 1)
        self.assertEqual(calls[0]["package"], self.packages[1])
        self.assertIn("-test.run=^TestFresh$/^nested$", calls[0]["args"])

    def test_other_rerun_arguments_fail_before_execution(self):
        for args in (("-run=TestFresh",), ("-run=TestFresh", PREFIX + "/outside"), ("-short", self.packages[0]),
                     ("-test.run=TestFresh", PREFIX + "/outside")):
            with self.subTest(args=args):
                self.assertNotEqual(self.run_helper("run", *args).returncode, 0)
                self.assertFalse(self.calls.exists())

    def test_invalid_package_timeout_fails_before_spawning_binaries(self):
        result = self.run_helper('run', TEST_TIMEOUT='invalid')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.calls.exists())

    def test_cancellation_terminates_active_binary_process_group(self):
        pidfile = self.root / "binary.pid"
        self.executable(self.build / "0.test", "#!/usr/bin/env python3\nimport os,time,pathlib\npathlib.Path(" + repr(str(pidfile)) + ").write_text(str(os.getpid()))\ntime.sleep(300)\n")
        process = subprocess.Popen(self.command('run'), env=self.env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        try:
            deadline = time.monotonic() + 10
            while not pidfile.exists() and process.poll() is None and time.monotonic() < deadline:
                time.sleep(0.02)
            self.assertTrue(pidfile.exists())
            child = int(pidfile.read_text())
            process.send_signal(signal.SIGTERM)
            _, stderr = process.communicate(timeout=10)
            self.assertEqual(process.returncode, 143, stderr)
            with self.assertRaises(ProcessLookupError):
                os.kill(child, 0)
        finally:
            if process.poll() is None:
                process.kill()
                process.communicate()
            if pidfile.exists():
                try: os.kill(int(pidfile.read_text()), signal.SIGKILL)
                except ProcessLookupError: pass

    def test_subprocess_interrupt_disposition_matches_go_test(self):
        self.executable(self.build / '0.test', '''#!/usr/bin/env python3
import json, signal, subprocess
p = subprocess.Popen(['/bin/sleep', '30'])
try:
    p.send_signal(signal.SIGINT)
    p.wait(timeout=2)
    assert p.returncode == -signal.SIGINT
finally:
    if p.poll() is None: p.kill(); p.wait()
''')
        result = self.run_helper('run')
        self.assertEqual(result.returncode, 0, result.stderr)

    @unittest.skipUnless(shutil.which("gotestsum"), "Pinned gotestsum is exercised on Linux")
    def test_actual_gotestsum_stops_retries_above_fifty_failures(self):
        self.executable(self.build / "0.test", "#!/usr/bin/env python3\nimport json,os\np=os.environ['REPORT_PACKAGE']\nwith open(os.environ['BINARY_CALLS'],'a') as f:f.write(p+'\\n')\nfor i in range(51):\n for action in ['run','fail']: print(json.dumps({'Action':action,'Package':p,'Test':'TestFailure'+str(i)}))\nprint(json.dumps({'Action':'fail','Package':p,'Elapsed':0.01}))\nraise SystemExit(1)\n")
        command = ['gotestsum', '--rerun-fails=3', '--rerun-fails-max-failures=50',
                   '--jsonfile='+str(self.root/'many-failures.json'), '--raw-command', '--', *self.command('run')]
        result = subprocess.run(command, env=self.env, capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.calls.read_text().splitlines().count(self.packages[0]), 1, result.stdout + result.stderr)

    @unittest.skipUnless(shutil.which("gotestsum"), "Pinned gotestsum is exercised in the hosted Go helper task")
    def test_actual_gotestsum_preserves_three_failed_reruns_and_reports(self):
        report = self.root / "attempts.json"
        junit = self.root / "junit.xml"
        result = subprocess.run(["gotestsum", "--rerun-fails=3", "--rerun-fails-max-failures=50",
                                 "--jsonfile=" + str(report), "--junitfile=" + str(junit), "--raw-command", "--", *self.command("run")],
                                env={**self.env, "FAIL_PACKAGE": self.packages[1]}, text=True, capture_output=True)
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        failed_calls = [call for call in self.recorded(self.calls) if call["package"] == self.packages[1]]
        self.assertEqual(len(failed_calls), 4, result.stdout + result.stderr)
        for call in failed_calls[1:]:
            self.assertIn("-test.run=^TestFresh$", call["args"])
        events = [json.loads(line) for line in report.read_text().splitlines()]
        failures = [event for event in events if event.get("Test") == "TestFresh" and event["Action"] == "fail"]
        self.assertEqual(len(failures), 4)
        self.assertTrue(junit.is_file())


if __name__ == "__main__":
    unittest.main()
