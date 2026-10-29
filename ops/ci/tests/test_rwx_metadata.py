"""Exercise RWX routing through CircleCI's routing scripts with real Git histories."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[3]
# The adapter reads CircleCI's routing data and policy without modifying them.
ROUTING_FILES = (
    "ops/ci/runtime/rwx-metadata.sh",
    ".circleci/routing.yml",
    ".circleci/scripts/collect-params.sh",
    ".circleci/scripts/compute-workflow-conditions.sh",
    ".circleci/scripts/workflow-helpers.sh",
)


CONSUMED = ("c-run_main", "c-run_rust_ci", "c-run_rust_e2e_ci", "c-run_contracts_feature_tests")


class RwxMetadataTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.upstream = self.root / "upstream"
        self.checkout = self.root / "checkout"
        self.values = self.root / "values"
        self.upstream.mkdir()
        self.git("init", "--initial-branch=develop", cwd=self.upstream)
        self.configure_git(self.upstream)
        for name in ROUTING_FILES:
            destination = self.upstream / name
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(ROOT / name, destination)
        self.write("docs/public-docs/example.md", "original\n", self.upstream)
        self.git("add", ".", cwd=self.upstream)
        self.git("commit", "-m", "initial", cwd=self.upstream)
        self.git("clone", str(self.upstream), str(self.checkout), cwd=self.root)
        self.configure_git(self.checkout)
        self.git("switch", "-c", "rwx/example")

    def git(self, *args, cwd=None):
        return subprocess.run(
            ["git", *args],
            cwd=cwd or self.checkout,
            check=True,
            text=True,
            capture_output=True,
        ).stdout.strip()

    def configure_git(self, directory):
        self.git("config", "user.name", "CI fixture", cwd=directory)
        self.git("config", "user.email", "ci-fixture@example.invalid", cwd=directory)

    def write(self, name, contents="changed\n", directory=None):
        path = (directory or self.checkout) / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(contents, encoding="utf-8")

    def commit(self, *names):
        self.git("add", "--", *names)
        self.git("commit", "-m", "fixture change")

    def run_metadata(self, **overrides):
        env = dict(os.environ)
        for name in ("BRANCH", "TAG", "TRIGGER_SOURCE", "SCHEDULE_NAME", "BASE_REVISION", "OUTPUT"):
            env.pop(name, None)
        env.update(
            CI_EVENT="push",
            CI_BRANCH="rwx/example",
            CI_TAG="",
            CI_COMMIT_SHA=self.git("rev-parse", "HEAD"),
            CI_BASE_REVISION="develop",
            RWX_VALUES=str(self.values),
        )
        env.update(overrides)
        return subprocess.run(
            ["bash", "ops/ci/runtime/rwx-metadata.sh"],
            cwd=self.checkout,
            env=env,
            text=True,
            capture_output=True,
        )

    def assert_routes(self, main, rust, **overrides):
        result = self.run_metadata(**overrides)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual((self.values / "run-main").read_text().strip(), str(main).lower())
        self.assertEqual((self.values / "run-rust-ci").read_text().strip(), str(rust).lower())
        params = json.loads((self.checkout / ".ci/pipeline-parameters.json").read_text())
        # Route tasks and gate aggregates read these as booleans, never as absent keys.
        for name in CONSUMED:
            self.assertIs(type(params[name]), bool, name)
        self.assertEqual((self.values / "run-rust-e2e-ci").read_text().strip(), str(params['c-run_rust_e2e_ci']).lower())
        return params

    def test_docs_only(self):
        self.write("docs/public-docs/example.md")
        self.commit("docs/public-docs/example.md")
        params = self.assert_routes(False, False)
        self.assertTrue(params["c-run_ci_gate_skip"])

    def test_cache_warming_has_a_successful_route_but_selects_no_verdicts(self):
        self.write("op-node/rollup/example.go")
        self.commit("op-node/rollup/example.go")
        params = self.assert_routes(False, False, CI_CACHE_WARM="true", CI_BRANCH="develop")
        self.assertTrue(params["cache-rebuild"])
        self.assertFalse(params["c-run_contracts_feature_tests"])
        self.assertFalse(params["c-run_rust_e2e_ci"])
        for branch in ("rwx/example", "external-fork/untrusted"):
            with self.subTest(branch=branch):
                self.assertNotEqual(self.run_metadata(CI_CACHE_WARM="true", CI_BRANCH=branch).returncode, 0)

    def test_cache_warming_still_rejects_a_different_checked_out_sha(self):
        result = self.run_metadata(CI_CACHE_WARM="true", CI_BRANCH="develop", CI_COMMIT_SHA="a" * 40)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((self.values / "run-main").exists())

    def test_docs_and_unclassified_code_run_main(self):
        self.write("docs/public-docs/example.md")
        self.write("new-component/main.go")
        self.commit("docs/public-docs/example.md", "new-component/main.go")
        self.assert_routes(True, False)

    def test_code_routes_like_circleci(self):
        self.write("op-node/example.go")
        self.commit("op-node/example.go")
        params = self.assert_routes(True, False)
        self.assertIs(params["c-run_contracts_feature_tests"], False)
        self.assertTrue(params["c-run_contracts_feature_tests_short"])
        self.write("rust/kona/example.rs")
        self.commit("rust/kona/example.rs")
        self.assert_routes(True, True)

    def test_rwx_and_ops_ci_changes_run_all_suites(self):
        for name in (".rwx/pr-gates.yml", "ops/ci/runtime/example.py"):
            with self.subTest(name=name):
                self.write(name)
                self.commit(name)
                params = self.assert_routes(True, True)
                self.assertTrue(params["c-run_contracts_feature_tests"])
                self.assertTrue(params["c-run_rust_e2e_ci"])

    def test_cli_patch_runs_all_suites(self):
        self.write("docs/public-docs/example.md")
        self.commit("docs/public-docs/example.md")
        # An uncommitted docs edit is still unseen by the committed diff, so it
        # selects every workload rather than the docs-only shortcut.
        self.write("docs/public-docs/example.md", "uncommitted\n")
        self.assert_routes(True, True)
        self.git("checkout", "--", "docs/public-docs/example.md")
        self.write("new-component/untracked.go")
        self.assert_routes(True, True)

    def test_generated_ci_directory_is_not_a_cli_patch(self):
        self.write("docs/public-docs/example.md")
        self.commit("docs/public-docs/example.md")
        self.write(".ci/previous/report.json", "{}\n")
        self.assert_routes(False, False)

    def test_develop_empty_diff_runs_full_postmerge_set(self):
        self.git("switch", "develop")
        params = self.assert_routes(True, True, CI_BRANCH="develop")
        self.assertTrue(params["c-run_kona_publish_prestates"])
        self.assertTrue(params["c-run_contracts_feature_tests"])

    def test_tag_routes_release(self):
        params = self.assert_routes(False, False, CI_BRANCH="", CI_TAG="v1.0.0")
        self.assertEqual(params, {"c-run_release": True, **dict.fromkeys(CONSUMED, False)})

    def test_non_push_events_are_rejected(self):
        result = self.run_metadata(CI_EVENT="schedule")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.values.exists())

    def test_wrong_sha_fails_before_reporting_values(self):
        result = self.run_metadata(CI_COMMIT_SHA="0" * 40)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("differs from CI_COMMIT_SHA", result.stderr)
        self.assertFalse(self.values.exists())

    def test_missing_base_fails_before_reporting_values(self):
        result = self.run_metadata(CI_BASE_REVISION="missing")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.values.exists())

    def test_shallow_checkout_fetches_merge_base(self):
        shutil.rmtree(self.checkout)
        self.git(
            "clone", "--depth=1", self.upstream.as_uri(), str(self.checkout), cwd=self.root
        )
        self.configure_git(self.checkout)
        self.git("switch", "-c", "rwx/example")
        self.write("docs/public-docs/example.md")
        self.commit("docs/public-docs/example.md")
        self.git("push", "origin", "HEAD:refs/heads/rwx/example")
        self.assert_routes(False, False)
        self.assertEqual(self.git("rev-parse", "--is-shallow-repository"), "false")


if __name__ == "__main__":
    unittest.main()
