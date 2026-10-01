#!/usr/bin/env python3
"""Exercise cache invalidation, artifact provenance and fresh failing verdicts."""

import copy
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPTS = Path(__file__).resolve().parent
SHA = "1" * 40
DISCOVERY = {"rust-suites": {"reth-optimism-node::e2e": {"testcases": {
    "passes": {"ignored": False, "filter-match": {"status": "matches"}},
    "ignored": {"ignored": True, "filter-match": {"status": "matches"}},
}}}}
JUNIT = '<testsuites><testsuite><testcase classname="reth-optimism-node::e2e" name="passes"/><testcase classname="reth-optimism-node::e2e" name="ignored"><skipped/></testcase></testsuite></testsuites>'

STUB = r'''#!/usr/bin/env python3
import json
import os
from pathlib import Path
import sys
command = Path(sys.argv[0]).name
args = sys.argv[1:]
with open(os.environ["CALL_LOG"], "a") as log:
    log.write(json.dumps({"command": command, "args": args,
                          "sync": os.environ.get("OP_RETH_SYNC_SUPERCHAIN"),
                          "wrapper": os.environ.get("RUSTC_WRAPPER")}) + "\n")
if command == "mold":
    os.execvp(args[1], args[1:])
elif command == "git":
    if args == ["rev-parse", "HEAD"]:
        print(os.environ["SOURCE_SHA"])
    elif args[:2] == ["submodule", "status"]:
        print("pinned public submodule")
    elif args[:2] == ["diff", "--exit-code"]:
        sys.exit(int(os.environ.get("DIFF_EXIT", "0")))
    else:
        sys.exit("Unexpected git command")
elif command == "just":
    if args != ["update-superchain-registry-submodule"]:
        sys.exit("Unexpected just command")
    sys.exit(int(os.environ.get("SOURCE_EXIT", "0")))
elif command == "sccache":
    if args[:1] == ["--show-stats"]:
        print('{"stats": {"cache_hits": {"Rust": 1}}}')
        sys.exit(int(os.environ.get("STATS_EXIT", "0")))
    elif args == ["--version"]:
        print("sccache 0.18.0")
elif command == "rustc":
    print("rustc 1.95.0")
elif command == "cargo":
    if args == ["--version"] or args == ["nextest", "--version"]:
        print("pinned tool")
    elif args[:1] == ["build"]:
        target = Path(os.environ["CARGO_TARGET_DIR"])
        profile = "release" if "release" in args else "debug"
        (target / profile).mkdir(parents=True, exist_ok=True)
        for binary in ("op-reth", "op-reth-sdm-fixture"):
            output = target / profile / binary
            output.write_text("#!/bin/sh\necho compiled-version\n")
            output.chmod(0o755)
        sys.exit(int(os.environ.get("BUILD_EXIT", "0")))
    elif args[:2] == ["nextest", "archive"]:
        Path(args[args.index("--archive-file") + 1]).write_text("compiled test archive")
        sys.exit(int(os.environ.get("BUILD_EXIT", "0")))
    elif args[:2] == ["nextest", "list"]:
        print(os.environ["DISCOVERY_JSON"])
    elif args[:2] == ["nextest", "run"]:
        target = Path(args[args.index("--target-dir-remap") + 1])
        junit = target / "nextest/default/junit.xml"
        junit.parent.mkdir(parents=True, exist_ok=True)
        if os.environ.get("OMIT_JUNIT") != "true":
            junit.write_text(os.environ["JUNIT_XML"])
        print("Fresh test execution")
        sys.exit(int(os.environ.get("TEST_EXIT", "0")))
    else:
        sys.exit("Unexpected cargo command: " + repr(args))
elif command == "tar":
    target = Path(args[args.index("-C") + 1]) / "target/nextest"
    target.mkdir(parents=True, exist_ok=True)
    (target / "binaries-metadata.json").write_text("{}")
    (target / "cargo-metadata.json").write_text("{}")
else:
    sys.exit("Unexpected command: " + command)
'''


class ShadowTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / ".git").mkdir()
        (self.root / "rust/op-reth/crates/chainspec/res").mkdir(parents=True)
        (self.root / "rust/Cargo.lock").write_text("pinned dependencies")
        self.scripts = self.root / "ops/ci"
        self.scripts.mkdir(parents=True)
        for name in ("op-reth-shadow.sh", "op-reth-report.py"):
            shutil.copyfile(SCRIPTS / name, self.scripts / name)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        for name in ("git", "just", "mold", "sccache", "rustc", "cargo", "tar"):
            path = self.bin / name
            path.write_text(STUB)
            path.chmod(0o755)
        self.log = self.root / "calls.jsonl"
        self.env = dict(os.environ, PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
                        CALL_LOG=str(self.log), SOURCE_SHA=SHA, CI_COMMIT_SHA=SHA,
                        CODEC_BASE_SHA=SHA, DISCOVERY_JSON=json.dumps(DISCOVERY), JUNIT_XML=JUNIT)

    def run_job(self, job, **overrides):
        return subprocess.run(["bash", str(self.scripts / "op-reth-shadow.sh"), job],
                              cwd=self.root, env=dict(self.env, **overrides), text=True,
                              stdout=subprocess.PIPE, stderr=subprocess.STDOUT)

    def calls(self, command):
        return [c for c in map(json.loads, self.log.read_text().splitlines()) if c["command"] == command]

    def report(self, job, name):
        return json.loads((self.root / f".ci/op-reth/{job}/{name}.json").read_text())

    def build_integration(self):
        result = self.run_job("integration-build")
        self.assertEqual(result.returncode, 0, result.stdout)

    def test_release_exports_both_verified_binaries(self):
        result = self.run_job("release-build")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertEqual(set(self.report("release-build", "binaries")["files"]), {"op-reth", "op-reth-sdm-fixture"})
        self.assertEqual(self.run_job("release").returncode, 0)

    def test_changed_binary_fails_verification(self):
        self.assertEqual(self.run_job("release-build").returncode, 0)
        (self.root / ".ci/op-reth/release-build/op-reth").write_text("corrupted")
        self.assertNotEqual(self.run_job("release").returncode, 0)

    def test_wrong_source_revision_fails_before_artifact_consumption(self):
        self.build_integration()
        result = self.run_job("integration", CI_COMMIT_SHA="2" * 40)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(c["args"][:2] == ["nextest", "run"] for c in self.calls("cargo")))

    def test_cold_probe_removes_only_targets_and_keeps_sccache(self):
        target = self.root / "rust/target/old-object"
        target.parent.mkdir(parents=True)
        target.write_text("old target")
        cache = self.root / ".ci/rust-cache/sccache/entry"
        cache.parent.mkdir(parents=True)
        cache.write_text("cached object")
        result = self.run_job("release-build", TARGET_CACHE_MODE="sccache-only")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertFalse(target.exists())
        self.assertEqual(cache.read_text(), "cached object")

    def test_warm_target_is_kept_but_stale_superchain_tar_removed(self):
        target = self.root / "rust/target/old-object"
        target.parent.mkdir(parents=True)
        target.write_text("old target")
        tar = self.root / "rust/op-reth/crates/chainspec/res/superchain-configs.tar"
        tar.write_text("stale source artifact")
        self.assertEqual(self.run_job("release-build").returncode, 0)
        self.assertTrue(target.exists())
        self.assertFalse(tar.exists())

    def test_invalid_cache_mode_fails(self):
        result = self.run_job("release-build", TARGET_CACHE_MODE="unknown")
        self.assertNotEqual(result.returncode, 0)

    def test_failed_compilation_retains_original_failure(self):
        result = self.run_job("release-build", BUILD_EXIT="23", STATS_EXIT="42")
        self.assertEqual(result.returncode, 23, result.stdout)
        self.assertEqual(self.report("release-build", "metadata")["exit_code"], 23)

    def test_successful_compile_with_failed_stats_fails(self):
        self.assertEqual(self.run_job("release-build", STATS_EXIT="42").returncode, 42)

    def test_integration_executes_fresh_without_compilation(self):
        self.build_integration()
        self.log.write_text("")
        result = self.run_job("integration")
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertEqual(self.report("integration", "coverage")["outcomes"], {"pass": 1, "skip": 1, "fail": 0})
        self.assertFalse(any(c["args"][:1] == ["build"] for c in self.calls("cargo")))
        self.assertTrue(any(c["args"][:2] == ["nextest", "run"] for c in self.calls("cargo")))

    def test_missing_junit_does_not_mask_test_failure(self):
        self.build_integration()
        self.assertEqual(self.run_job("integration", OMIT_JUNIT="true", TEST_EXIT="100").returncode, 100)

    def test_missing_junit_fails_successful_verdict(self):
        self.build_integration()
        self.assertNotEqual(self.run_job("integration", OMIT_JUNIT="true").returncode, 0)

    def test_discovery_omissions_and_duplicate_verdicts_fail(self):
        self.build_integration()
        missing = JUNIT.replace('<testcase classname="reth-optimism-node::e2e" name="passes"/>', "")
        duplicate = JUNIT.replace("</testsuite>", '<testcase classname="reth-optimism-node::e2e" name="passes"/></testsuite>')
        for junit in (missing, duplicate):
            with self.subTest(junit=junit):
                self.assertNotEqual(self.run_job("integration", JUNIT_XML=junit).returncode, 0)

    def test_filtered_or_unexpectedly_skipped_test_fails(self):
        self.build_integration()
        discovery = copy.deepcopy(DISCOVERY)
        discovery["rust-suites"]["reth-optimism-node::e2e"]["testcases"]["passes"]["filter-match"]["status"] = "mismatch"
        self.assertNotEqual(self.run_job("integration", DISCOVERY_JSON=json.dumps(discovery)).returncode, 0)
        junit = JUNIT.replace('name="passes"/>', 'name="passes"><skipped/></testcase>')
        self.assertNotEqual(self.run_job("integration", JUNIT_XML=junit).returncode, 0)

    def test_retried_failure_is_visible(self):
        self.build_integration()
        junit = JUNIT.replace('name="passes"/>', 'name="passes"><flakyFailure>first attempt failed</flakyFailure></testcase>')
        result = self.run_job("integration", JUNIT_XML=junit)
        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertEqual(self.report("integration", "coverage")["retried_cases"], [["reth-optimism-node::e2e", "passes"]])

    def test_snapshot_changes_sync_flag_and_checks_regenerated_files(self):
        self.assertEqual(self.run_job("snapshot-build").returncode, 0)
        self.assertEqual(self.run_job("snapshot", DIFF_EXIT="1").returncode, 1)
        builds = [c for c in self.calls("cargo") if c["args"][:1] == ["build"]]
        self.assertEqual([c["sync"] for c in builds], ["0", "1"])
        self.assertIsNone(builds[-1]["wrapper"])

    def test_failed_submodule_setup_propagates(self):
        self.assertEqual(self.run_job("source", SOURCE_EXIT="17").returncode, 17)


class VectorReportTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.vectors = self.root / "testdata/micro/compact"
        self.vectors.mkdir(parents=True)
        self.report = self.root / "reports"
        self.report.mkdir()
        self.path = self.vectors / "TxDeposit.json"
        self.path.write_text(json.dumps(["00"] * 100))

    def run_report(self, operation, sha=SHA):
        return subprocess.run(["python3", str(SCRIPTS / "op-reth-report.py"), operation,
                               str(self.report), sha], cwd=self.root, capture_output=True)

    def test_baseline_checksum_and_revision_are_enforced(self):
        self.assertEqual(self.run_report("vectors").returncode, 0)
        self.assertEqual(self.run_report("verify-vectors").returncode, 0)
        self.assertNotEqual(self.run_report("verify-vectors", "2" * 40).returncode, 0)
        self.path.write_text(json.dumps(["01"] * 100))
        self.assertNotEqual(self.run_report("verify-vectors").returncode, 0)

    def test_empty_or_partial_vector_sets_fail(self):
        self.path.write_text(json.dumps(["00"] * 99))
        self.assertNotEqual(self.run_report("vectors").returncode, 0)
        self.path.unlink()
        self.assertNotEqual(self.run_report("vectors").returncode, 0)


if __name__ == "__main__":
    unittest.main()
