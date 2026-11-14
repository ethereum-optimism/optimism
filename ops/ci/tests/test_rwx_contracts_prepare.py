#!/usr/bin/env python3
"""Exercise RWX contract tool and source preparation, downloads and failures."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPTS = Path(__file__).resolve().parents[1] / 'runtime'
CONFIG = {
    "test": "test", "ffi": True, "optimizer": False, "optimizer_runs": 0,
    "fuzz": {"runs": 128}, "invariant": {"runs": 64, "depth": 32},
    "skip": [], "eth_rpc_url": None, "fork_block_number": None,
}
JUNIT = '<testsuites><testsuite><testcase name="test_a_succeeds"/><testcase name="test_fork_succeeds"><skipped/></testcase></testsuite></testsuites>'

# Commands are stubbed at the process boundary: the real helpers decide which
# commands, settings and reports to use, without installing tools or doing IO.
STUB = r'''#!/usr/bin/env python3
import json
import os
from pathlib import Path
import sys

command = Path(sys.argv[0]).name
args = sys.argv[1:]
env = os.environ
record = {"command": command, "args": args, "env": {
    key: value for key, value in env.items()
    if key.startswith(("FOUNDRY_", "DAPP_", "DEV_FEATURE__", "SYS_FEATURE__"))
    or key in ("FORK_TEST", "L2_FORK_TEST", "L2CM_ACTIVATION_TEST", "ETH_RPC_URL", "FORK_RPC_URL")
}}
with Path(env["CALL_LOG"]).open("a") as log:
    log.write(json.dumps(record) + "\n")

if command == "forge":
    if args == ["build"]:
        Path('forge-artifacts').mkdir(exist_ok=True)
        Path('forge-artifacts/A.t.sol').mkdir(exist_ok=True)
        Path('forge-artifacts/A.t.sol/A.json').write_text('{}')
        Path('cache').mkdir(exist_ok=True)
        Path('cache/solidity-files-cache.json').write_text('{}')
        sys.exit(int(env.get('FORGE_BUILD_EXIT', '0')))
    if args == ["test", "--junit"]:
        if env.get("OMIT_JUNIT") != "true":
            print(env["JUNIT_XML"])
        Path("cache/fuzz").mkdir(parents=True, exist_ok=True)
        Path("cache/fuzz/counterexample").write_text("fresh counterexample\n")
        sys.exit(int(env.get("TEST_EXIT", "0")))
    if args == ["test", "--rerun", "-vvv"]:
        print("Failing test trace")
        sys.exit(int(env.get("RERUN_EXIT", "0")))
    if args == ["config", "--json"]:
        config = json.loads(env["CONFIG_JSON"])
        if env.get("FOUNDRY_PROFILE") == "ci":
            config.update(optimizer=True, optimizer_runs=999999)
        print(json.dumps(config))
        sys.exit(int(env.get("CONFIG_EXIT", "0")))
    if args == ["--version"]:
        print("forge Version: 1.2.3")
        sys.exit(0)
elif command == "just":
    if args == ["test"]:
        print("Compiler output", file=sys.stderr)
        if env.get("OMIT_JUNIT") != "true":
            Path(env["JUNIT_TEST_PATH"]).write_text(env["JUNIT_XML"])
        Path(".resource-metering.csv").write_text("generated metering\n")
        Path(".testdata").mkdir(exist_ok=True)
        Path(".testdata/trace.txt").write_text("generated test data\n")
        Path("cache/fuzz").mkdir(parents=True, exist_ok=True)
        Path("cache/fuzz/counterexample").write_text("generated counterexample\n")
        sys.exit(int(env.get("TEST_EXIT", "0")))
    if args == ["test-rerun"]:
        print("Failing test trace")
        sys.exit(int(env.get("RERUN_EXIT", "0")))
    if args == ["lint-forge-tests-check-no-build"]:
        print("Go test convention validation")
        sys.exit(int(env.get("LINT_EXIT", "0")))
    if args == ["build-go-ffi"]:
        print("Go FFI build")
        Path('scripts/go-ffi').mkdir(parents=True, exist_ok=True)
        Path('scripts/go-ffi/go-ffi').write_text('compiled FFI')
        sys.exit(int(env.get("FFI_EXIT", "0")))
elif command == "git":
    if args == ["rev-parse", "HEAD"]:
        print("a" * 40)
        sys.exit(0)
    if args[:2] == ["submodule", "sync"]:
        sys.exit(0)
    if args[:2] == ["submodule", "status"]:
        print(" public pinned submodule")
        sys.exit(0)
    if args[:4] == ["-c", "protocol.file.allow=never", "submodule", "update"]:
        label = "git"
    else:
        sys.exit("Unexpected Git command")
elif command == "go" and args == ["mod", "download"]:
    label = "go"
elif command == "go" and args[:2] == ["build", "-buildvcs=false"]:
    target = Path(args[args.index("-o") + 1])
    target.write_text('#!/bin/sh\necho "Go test convention validation"\nexit "${LINT_EXIT:-0}"\n')
    target.chmod(0o755)
    sys.exit(int(env.get("CHECKER_BUILD_EXIT", "0")))
elif command == "mise":
    if args == ["bin-paths"]:
        print(env["STUB_BIN"])
        sys.exit(0)
    if args == ["install", "forge", "cast", "svm-rs"]:
        label = "mise"
    else:
        sys.exit("Unexpected mise command")
elif command == "svm":
    if len(args) == 2 and args[0] == "which":
        sys.exit(0 if args[1] in env.get("INSTALLED_SOLC", "").split(",") else 1)
    if len(args) == 2 and args[0] == "install":
        label = "svm-" + args[1]
    else:
        sys.exit("Unexpected svm command")
elif command == "sleep":
    sys.exit(0)
else:
    sys.exit("Unexpected command: " + command + " " + repr(args))

counter = Path(env["COUNTERS"]) / label
count = int(counter.read_text()) + 1 if counter.exists() else 1
counter.write_text(str(count))
if count <= int(env.get("FAIL_" + label.upper().replace("-", "_").replace(".", "_"), "0")):
    sys.exit(17)
print("Prepared " + label)
'''


class RwxContractsPrepareTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        scripts = self.root / "ops/ci/runtime"
        scripts.mkdir(parents=True)
        for name in ("rwx-contracts-prepare.sh", "rwx-source-archive.sh"):
            shutil.copyfile(SCRIPTS / name, scripts / name)
        self.contracts = self.root / "packages/contracts-bedrock"
        for name in ("test/A.t.sol", "test/nested/B.t.sol", "test/invariants/C.t.sol", "test/Support.sol"):
            path = self.contracts / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text("// fixture\n", encoding="utf-8")
        self.bin = self.root / "bin"
        self.bin.mkdir()
        for name in ("forge", "just", "git", "go", "mise", "svm", "sleep"):
            path = self.bin / name
            path.write_text(STUB, encoding="utf-8")
            path.chmod(0o755)
        self.log = self.root / "calls.jsonl"
        self.counters = self.root / "counters"
        self.counters.mkdir()

    def run_helper(self, name, args=(), **overrides):
        env = dict(os.environ)
        env.update(
            PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
            CI_BRANCH="feature/example", CONTRACT_FEATURE="main", CONFIG_JSON=json.dumps(CONFIG),
            JUNIT_XML=JUNIT, CALL_LOG=str(self.log), COUNTERS=str(self.counters), STUB_BIN=str(self.bin),
            RWX_ENV=str(self.root / "rwx-env"),
        )
        env.update(overrides)
        # Empty branch is intentionally passed through for missing metadata tests.
        return subprocess.run(
            ["bash", str(self.root / "ops/ci/runtime" / name), *args],
            cwd=self.root, env=env, text=True, capture_output=True,
        )

    def calls(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()] if self.log.exists() else []

    def assert_success(self, result):
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_source_retries_downloads_then_builds_go_ffi_with_full_inputs(self):
        result = self.run_helper("rwx-contracts-prepare.sh", ("source",), FAIL_GIT="1", FAIL_GO="2")
        self.assert_success(result)
        calls = self.calls()
        updates = [call for call in calls if call["command"] == "git" and "update" in call["args"]]
        self.assertEqual(len(updates), 2)
        self.assertEqual(updates[0]["args"], ["-c", "protocol.file.allow=never", "submodule", "update", "--init", "--recursive", "--jobs", "8"])
        self.assertEqual(len([call for call in calls if call["command"] == "go" and call["args"] == ["mod", "download"]]), 3)
        self.assertTrue(any(call["args"] == ["build-go-ffi"] for call in calls))
        self.assertTrue((self.root / ".ci/contracts-prepare/test-validation").is_file())
        self.assertTrue((self.root / ".ci/contracts-prepare/source.tar.gz").is_file())
        self.assertEqual((self.contracts / ".gitcommit").read_text().strip(), "a" * 40)
        self.assertTrue((self.root / ".ci/contracts-prepare/go-ffi-build.log").is_file())

    def test_failed_download_or_ffi_build_is_not_hidden_by_tee(self):
        for overrides, status, forbidden in (
            ({"FAIL_GIT": "5"}, 17, "go"), ({"FAIL_GO": "5"}, 17, "just"), ({"FFI_EXIT": "19"}, 19, None),
        ):
            with self.subTest(overrides=overrides):
                for path in self.counters.iterdir():
                    path.unlink()
                self.log.unlink(missing_ok=True)
                result = self.run_helper("rwx-contracts-prepare.sh", ("source",), **overrides)
                self.assertEqual(result.returncode, status, result.stdout + result.stderr)
                if forbidden:
                    self.assertFalse(any(call["command"] == forbidden for call in self.calls()))

    def test_tools_install_only_pinned_contract_tools_and_exact_missing_compilers(self):
        result = self.run_helper("rwx-contracts-prepare.sh", ("tools",), FAIL_MISE="1", FAIL_SVM_0_8_25="1", INSTALLED_SOLC="0.8.15")
        self.assert_success(result)
        calls = self.calls()
        installs = [call["args"] for call in calls if call["command"] == "mise" and call["args"][0] == "install"]
        self.assertEqual(installs, [["install", "forge", "cast", "svm-rs"]] * 2)
        self.assertEqual([call["args"][1] for call in calls if call["command"] == "svm" and call["args"][0] == "which"],
                         ["0.8.15", "0.8.19", "0.8.25", "0.8.28"])
        self.assertEqual([call["args"][1] for call in calls if call["command"] == "svm" and call["args"][0] == "install"],
                         ["0.8.19", "0.8.25", "0.8.25", "0.8.28"])
        self.assertTrue((self.root / "rwx-env/PATH").is_file())

    def test_failed_tool_or_compiler_download_stops_bootstrap(self):
        for overrides, expected_command in (({"FAIL_MISE": "5"}, "mise"), ({"FAIL_SVM_0_8_15": "5"}, "svm")):
            with self.subTest(overrides=overrides):
                for path in self.counters.iterdir():
                    path.unlink()
                self.log.unlink(missing_ok=True)
                result = self.run_helper("rwx-contracts-prepare.sh", ("tools",), **overrides)
                self.assertEqual(result.returncode, 17, result.stdout + result.stderr)
                downloads = [call for call in self.calls() if call["command"] == expected_command and call["args"][0] == "install"]
                self.assertEqual(len(downloads), 5)
                self.assertFalse((self.root / "rwx-env/PATH").exists())

    def test_failed_checker_build_stops_source_preparation(self):
        result = self.run_helper("rwx-contracts-prepare.sh", ("source",), CHECKER_BUILD_EXIT="8")
        self.assertEqual(result.returncode, 8, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
