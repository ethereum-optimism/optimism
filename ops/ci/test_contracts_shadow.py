#!/usr/bin/env python3
"""Exercise complete contract coverage, metadata, downloads and failed verdicts."""

import copy
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPTS = Path(__file__).resolve().parent
FEATURES = {
    "main": None,
    "CUSTOM_GAS_TOKEN": "SYS_FEATURE__CUSTOM_GAS_TOKEN",
    "OPTIMISM_PORTAL_INTEROP": "DEV_FEATURE__OPTIMISM_PORTAL_INTEROP",
    "ZK_DISPUTE_GAME": "DEV_FEATURE__ZK_DISPUTE_GAME",
}
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


class ContractsShadowTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        scripts = self.root / "ops/ci"
        scripts.mkdir(parents=True)
        for name in ("contracts-tests.sh", "contracts-test-env.sh", "contracts-test-report.py", "rwx-contracts-prepare.sh", "rwx-source-archive.sh", "rwx-contracts-build.py", "rwx-contracts-build.sh"):
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

    def run_helper(self, name="contracts-tests.sh", args=(), **overrides):
        env = dict(os.environ)
        env.update(
            PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
            CI_BRANCH="codex/contracts", CONTRACT_FEATURE="main", CONFIG_JSON=json.dumps(CONFIG),
            JUNIT_XML=JUNIT, CALL_LOG=str(self.log), COUNTERS=str(self.counters), STUB_BIN=str(self.bin),
            RWX_ENV=str(self.root / "rwx-env"),
        )
        env.update(overrides)
        # Empty branch is intentionally passed through for missing metadata tests.
        return subprocess.run(
            ["bash", str(self.root / "ops/ci" / name), *args],
            cwd=self.root, env=env, text=True, capture_output=True,
        )

    def calls(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()] if self.log.exists() else []

    def report(self, name):
        return self.contracts / "results/reports" / name

    def assert_success(self, result):
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_every_feature_uses_the_complete_unfiltered_inventory(self):
        for feature, variable in FEATURES.items():
            with self.subTest(feature=feature):
                self.assert_success(self.run_helper(CONTRACT_FEATURE=feature))
                meta = json.loads(self.report("metadata.json").read_text())
                self.assertEqual(meta["enabled_feature_environment"], variable)
                self.assertEqual(meta["test_files"], 3)
                self.assertEqual(self.report("test-files.txt").read_text().splitlines(),
                                 ["test/A.t.sol", "test/invariants/C.t.sol", "test/nested/B.t.sol"])
                test_call = [call for call in self.calls() if call["command"] == "just" and call["args"] == ["test"]][-1]
                enabled = {key: value for key, value in test_call["env"].items() if key.startswith(("DEV_FEATURE__", "SYS_FEATURE__"))}
                self.assertEqual(enabled, {variable: "true"} if variable else {})
                self.assertEqual(test_call["env"]["FORK_TEST"], "false")
                self.assertEqual(test_call["env"]["L2_FORK_TEST"], "false")
                self.assertEqual(test_call["env"]["L2CM_ACTIVATION_TEST"], "false")

    def test_develop_is_optimized_ci_and_all_other_branches_use_liteci(self):
        for branch, profile in (("develop", "ci"), ("codex/contracts", "liteci"),
                                ("gh-readonly-queue/develop/pr-1", "liteci"), ("develop-feature", "liteci")):
            with self.subTest(branch=branch):
                self.assert_success(self.run_helper(CI_BRANCH=branch))
                meta = json.loads(self.report("metadata.json").read_text())
                self.assertEqual(meta["profile"], profile)
                self.assertEqual((meta["fuzz_runs"], meta["invariant_runs"], meta["invariant_depth"]), (128, 64, 32))

    def test_inherited_filters_features_reduction_and_rpc_overrides_are_removed(self):
        self.assert_success(self.run_helper(
            FOUNDRY_PROFILE="lite", FOUNDRY_FUZZ_RUNS="1", FOUNDRY_MATCH_TEST="only_one", DAPP_TEST_NUMBER="1",
            DEV_FEATURE__OPTIMISM_PORTAL_INTEROP="true", SYS_FEATURE__CUSTOM_GAS_TOKEN="true",
            FORK_TEST="true", L2_FORK_TEST="true", L2CM_ACTIVATION_TEST="true", ETH_RPC_URL="fixture-rpc",
            FORK_RPC_URL="fixture-rpc",
        ))
        env = next(call["env"] for call in self.calls() if call["command"] == "just" and call["args"] == ["test"])
        self.assertEqual(env, {"FOUNDRY_PROFILE": "liteci", "FORK_TEST": "false", "L2_FORK_TEST": "false", "L2CM_ACTIVATION_TEST": "false"})

    def test_bad_metadata_fails_before_any_build_or_test(self):
        for overrides in ({"CI_BRANCH": ""}, {"CONTRACT_FEATURE": "unknown"}):
            with self.subTest(overrides=overrides):
                result = self.run_helper(**overrides)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.calls(), [])

    def test_effective_reduced_or_filtered_config_fails_before_tests(self):
        for field, value in (("fuzz", {"runs": 8}), ("invariant", {"runs": 64, "depth": 8}),
                             ("fuzz", {"runs": 128, "timeout": 1}), ("invariant", {"runs": 64, "depth": 32, "timeout": 1}),
                             ("test", "test/L1"), ("match_test", "test_one"), ("no_match_path", "invariants/*"),
                             ("skip", ["test"]), ("eth_rpc_url", "fixture-rpc"), ("ffi", False)):
            with self.subTest(field=field):
                self.log.unlink(missing_ok=True)
                config = copy.deepcopy(CONFIG)
                config[field] = value
                result = self.run_helper(CONFIG_JSON=json.dumps(config))
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(call["command"] == "just" for call in self.calls()))

    def test_empty_discovery_and_config_query_errors_fail_before_tests(self):
        result = self.run_helper(CONFIG_EXIT="23")
        self.assertEqual(result.returncode, 23)
        for path in (self.contracts / "test").rglob("*.t.sol"):
            path.unlink()
        result = self.run_helper()
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(call["command"] == "just" for call in self.calls()))

    def test_failed_test_status_survives_successful_or_failed_trace_rerun(self):
        xml = '<testsuite><testcase name="failed"><failure message="failure"/></testcase></testsuite>'
        for rerun_exit in ("0", "9"):
            with self.subTest(rerun_exit=rerun_exit):
                self.log.unlink(missing_ok=True)
                result = self.run_helper(TEST_EXIT="7", RERUN_EXIT=rerun_exit, JUNIT_XML=xml)
                self.assertEqual(result.returncode, 7, result.stdout + result.stderr)
                self.assertEqual((self.contracts / "results/results.xml").read_text(), xml)
                self.assertIn("Failing test trace", self.report("rerun-traces.log").read_text())
                self.assertIn("Compiler output", self.report("test.log").read_text())
                self.assertFalse(any(call["args"] == ["lint-forge-tests-check-no-build"] for call in self.calls()))
                self.assertTrue(self.report("generated/.resource-metering.csv").is_file())
                self.assertTrue(self.report("generated/.testdata/trace.txt").is_file())
                self.assertTrue(self.report("generated/cache/fuzz/counterexample").is_file())

    def test_missing_malformed_empty_or_false_green_junit_fails(self):
        for overrides in (
            {"OMIT_JUNIT": "true"}, {"JUNIT_XML": "not xml"}, {"JUNIT_XML": "<testsuite/>"},
            {"JUNIT_XML": '<testsuite><testcase name="skip"><skipped/></testcase></testsuite>'},
            {"JUNIT_XML": '<testsuite><testcase name="fail"><failure/></testcase></testsuite>'},
            {"JUNIT_XML": '<testsuite><testcase name="err"><error/></testcase></testsuite>'},
        ):
            with self.subTest(overrides=overrides):
                result = self.run_helper(**overrides)
                self.assertNotEqual(result.returncode, 0)
                self.assertTrue(self.report("report-validation.log").is_file())

    def test_name_validation_failure_is_preserved_after_passing_tests(self):
        result = self.run_helper(LINT_EXIT="11")
        self.assertEqual(result.returncode, 11)
        self.assertIn("Go test convention validation", self.report("test-validation.log").read_text())
        summary = json.loads(self.report("summary.json").read_text())
        self.assertEqual(summary, {"tests": 2, "skipped": 1, "failures": 0, "errors": 0})

    def test_a_prior_passing_report_cannot_hide_a_build_failure(self):
        self.assert_success(self.run_helper())
        result = self.run_helper(TEST_EXIT="31", OMIT_JUNIT="true")
        self.assertEqual(result.returncode, 31)
        self.assertFalse((self.contracts / "results/results.xml").exists())
        self.assertIn("No such file", self.report("report-validation.log").read_text())
        self.assertTrue(self.report("rerun-traces.log").is_file())

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

    def prepare_compilation(self, **overrides):
        self.assert_success(self.run_helper("rwx-contracts-prepare.sh", ("source",)))
        prerequisites = self.root / "mounted-prerequisites"
        shutil.copytree(self.root / ".ci/contracts-prepare", prerequisites)
        self.assert_success(self.run_helper("rwx-contracts-build.sh", (),
            CI_COMMIT_SHA="a" * 40, CONTRACT_PREREQUISITES=str(prerequisites), **overrides))

    def test_compiled_verdict_runs_the_full_suite_and_fresh_convention_check(self):
        self.prepare_compilation()
        self.log.unlink()
        self.assert_success(self.run_helper(RWX_COMPILED_CONTRACTS="true", CI_COMMIT_SHA="a" * 40))
        calls = self.calls()
        self.assertTrue(any(call["command"] == "forge" and call["args"] == ["test", "--junit"] for call in calls))
        self.assertFalse(any(call["command"] == "go" or call["command"] == "just" for call in calls))
        self.assertIn("Go test convention validation", self.report("test-validation.log").read_text())
        self.assertEqual(json.loads(self.report("summary.json").read_text())["tests"], 2)

    def test_compiled_profile_feature_sha_or_artifact_mismatch_cannot_run_tests(self):
        self.prepare_compilation()
        for overrides in ({"CI_BRANCH": "develop"}, {"CONTRACT_FEATURE": "CUSTOM_GAS_TOKEN"}, {"CI_COMMIT_SHA": "b" * 40}):
            self.log.unlink(missing_ok=True)
            result = self.run_helper(RWX_COMPILED_CONTRACTS="true", **{"CI_COMMIT_SHA": "a" * 40, **overrides})
            with self.subTest(overrides=overrides):
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(call["command"] == "forge" and call["args"][0] == "test" for call in self.calls()))
        with (self.root / ".ci/contracts-build/compilation.tar.gz").open("ab") as artifact:
            artifact.write(b"changed")
        self.assertNotEqual(self.run_helper(RWX_COMPILED_CONTRACTS="true", CI_COMMIT_SHA="a" * 40).returncode, 0)

    def test_compiled_verdict_keeps_first_failure_and_clears_stale_fuzz(self):
        self.prepare_compilation()
        stale = self.contracts / "cache/fuzz/stale"
        stale.parent.mkdir(parents=True, exist_ok=True)
        stale.write_text("old outcome")
        result = self.run_helper(RWX_COMPILED_CONTRACTS="true", CI_COMMIT_SHA="a" * 40, TEST_EXIT="29")
        self.assertEqual(result.returncode, 29, result.stdout + result.stderr)
        self.assertFalse(stale.exists())
        self.assertTrue(self.report("rerun-traces.log").is_file())
        self.assertTrue((self.contracts / "results/results.xml").is_file())

    def test_compiler_and_checker_build_failures_are_not_test_success(self):
        self.assert_success(self.run_helper("rwx-contracts-prepare.sh", ("source",)))
        prerequisites = self.root / "mounted-prerequisites"
        shutil.copytree(self.root / ".ci/contracts-prepare", prerequisites)
        result = self.run_helper("rwx-contracts-build.sh", (), CI_COMMIT_SHA="a" * 40,
                                 CONTRACT_PREREQUISITES=str(prerequisites), FORGE_BUILD_EXIT="7")
        self.assertEqual(result.returncode, 7, result.stdout + result.stderr)
        self.assertFalse((self.root / ".ci/contracts-build/compilation.tar.gz").exists())
        result = self.run_helper("rwx-contracts-prepare.sh", ("source",), CHECKER_BUILD_EXIT="8")
        self.assertEqual(result.returncode, 8)


if __name__ == "__main__":
    unittest.main()
