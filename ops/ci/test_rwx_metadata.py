"""Exercise RWX routing with real Git histories and local CLI patches."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPTS = Path(__file__).resolve().parent


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
        destination = self.upstream / "ops/ci"
        destination.mkdir(parents=True)
        for name in (
            "rwx-metadata.sh",
            "collect-params.sh",
            "compute-workflow-conditions.sh",
            "workflow-helpers.sh",
            "routing.yml",
        ):
            shutil.copyfile(SCRIPTS / name, destination / name)
        (self.upstream / ".gitignore").write_text(".ci/\n", encoding="utf-8")
        self.write("docs/public-docs/example.md", "original\n", self.upstream)
        self.git("add", ".", cwd=self.upstream)
        self.git("commit", "-m", "initial", cwd=self.upstream)
        self.git("clone", str(self.upstream), str(self.checkout), cwd=self.root)
        self.configure_git(self.checkout)
        self.git("switch", "-c", "codex/test")

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
        env.update(
            CI_EVENT="push",
            CI_BRANCH="codex/test",
            CI_TAG="",
            CI_COMMIT_SHA=self.git("rev-parse", "HEAD"),
            CI_BASE_REVISION="develop",
            RWX_VALUES=str(self.values),
        )
        env.update(overrides)
        return subprocess.run(
            ["bash", "ops/ci/rwx-metadata.sh"],
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
        return json.loads((self.checkout / ".ci/pipeline-parameters.json").read_text())

    def test_docs_only(self):
        self.write("docs/public-docs/example.md")
        self.commit("docs/public-docs/example.md")
        params = self.assert_routes(False, False)
        self.assertTrue(params["c-run_ci_gate_skip"])

    def test_cache_warming_has_a_successful_route_but_selects_no_verdicts(self):
        self.write("op-node/rollup/example.go")
        self.commit("op-node/rollup/example.go")
        for branch in ("develop", "codex/rwx-ci-pilot"):
            with self.subTest(branch=branch):
                params = self.assert_routes(False, False, CI_CACHE_WARM="true", CI_BRANCH=branch)
                self.assertTrue(params["cache-rebuild"])
                self.assertFalse(params["c-run_contracts_feature_tests"])
        self.assertNotEqual(self.run_metadata(CI_CACHE_WARM="true", CI_BRANCH="external-fork/untrusted").returncode, 0)

    def test_cache_warming_still_rejects_a_different_checked_out_sha(self):
        result = self.run_metadata(CI_CACHE_WARM="true", CI_BRANCH="develop", CI_COMMIT_SHA="a" * 40)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((self.values / "run-main").exists())

    def test_docs_and_unclassified_code_run_main(self):
        self.write("docs/public-docs/example.md")
        self.write("new-component/main.go")
        self.commit("docs/public-docs/example.md", "new-component/main.go")
        self.assert_routes(True, False)

    def test_rwx_config_runs_all_relevant_suites(self):
        self.write(".rwx/pilot.yml")
        self.commit(".rwx/pilot.yml")
        params = self.assert_routes(True, True)
        self.assertTrue(params["c-run_contracts_feature_tests"])

    def test_cli_patch_includes_untracked_configuration(self):
        self.write("docs/public-docs/example.md")
        self.commit("docs/public-docs/example.md")
        self.write(".rwx/pilot.yml")
        self.assert_routes(True, True)

    def test_cli_patch_includes_tracked_code(self):
        self.write("docs/public-docs/example.md")
        self.commit("docs/public-docs/example.md")
        self.write("ops/ci/routing.yml", (SCRIPTS / "routing.yml").read_text() + "\n")
        self.assert_routes(True, True)

    def test_develop_empty_diff_runs_full_postmerge_set(self):
        self.git("switch", "develop")
        params = self.assert_routes(True, True, CI_BRANCH="develop")
        self.assertTrue(params["c-run_kona_publish_prestates"])

    def test_merge_queue_forces_contracts_for_code_changes(self):
        self.write("op-node/example.go")
        self.commit("op-node/example.go")
        params = self.assert_routes(True, False, CI_BRANCH="gh-readonly-queue/develop/pr-1")
        self.assertTrue(params["c-run_contracts_feature_tests"])

    def test_tag_routes_release(self):
        params = self.assert_routes(False, False, CI_BRANCH="", CI_TAG="v1.0.0")
        self.assertEqual(params, {"c-run_release": True})

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
        self.git("switch", "-c", "codex/test")
        self.write("docs/public-docs/example.md")
        self.commit("docs/public-docs/example.md")
        self.git("push", "origin", "HEAD:refs/heads/codex/test")
        self.assert_routes(False, False)
        self.assertEqual(self.git("rev-parse", "--is-shallow-repository"), "false")


if __name__ == "__main__":
    unittest.main()
