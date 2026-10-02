#!/usr/bin/env python3
"""Exercise the legacy per-job early halt for shared CI changes on stacked PRs."""

import importlib.util
import os
from pathlib import Path
import unittest
from unittest import mock


SCRIPT = Path(__file__).resolve().parents[1] / "check-changed/main.py"
SPEC = importlib.util.spec_from_file_location("check_changed", SCRIPT)
CHECK_CHANGED = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CHECK_CHANGED)


class CheckChangedTest(unittest.TestCase):
    def assert_build_decision(self, paths, build):
        base_sha = "a" * 40
        head_sha = "b" * 40
        pr = {"url": "https://api.github.com/repos/fixture/repo/pulls/1",
              "base": {"sha": base_sha}, "head": {"sha": head_sha}}
        env = {"CIRCLE_PULL_REQUESTS": "https://github.com/fixture/repo/pull/1",
               "GITHUB_ACCESS_TOKEN": "fixture-token"}
        with mock.patch.dict(os.environ, env, clear=True), \
                mock.patch.object(CHECK_CHANGED.sys, "argv", [str(SCRIPT), "contracts-bedrock"]), \
                mock.patch.object(CHECK_CHANGED, "git_cmd",
                                  side_effect=["codex/stacked-shadow", "\n".join(paths)]) as git, \
                mock.patch.object(CHECK_CHANGED, "fetch_pull_request", return_value=pr) as fetch, \
                mock.patch.object(CHECK_CHANGED.subprocess, "check_call") as halt, \
                mock.patch.object(CHECK_CHANGED, "log"):
            with self.assertRaises(SystemExit) as result:
                CHECK_CHANGED.main()
            self.assertEqual(result.exception.code, 0)
            fetch.assert_called_once_with(1, "fixture-token")
            self.assertEqual(git.call_args_list[1].args[0],
                             f"diff --name-only {base_sha}...{head_sha}")
            if build:
                halt.assert_not_called()
            else:
                halt.assert_called_once_with(["circleci", "step", "halt"])

    def test_shared_ci_paths_execute_contracts(self):
        for path in (".rwx/contracts.yml", ".rwx/pilot.yml", ".rwx/go-rollup.yml",
                     "ops/ci/contracts-tests.sh", "ops/ci/routing.yml",
                     "ops/ci/test_check_changed.py"):
            with self.subTest(path=path):
                self.assert_build_decision([path], build=True)

    def test_existing_ci_and_contract_paths_still_execute(self):
        for path in (".circleci/continue/main.yml", ".github/workflows/check.yml",
                     "packages/contracts-bedrock/test/L1/Example.t.sol", "go.mod"):
            with self.subTest(path=path):
                self.assert_build_decision([path], build=True)

    def test_docs_and_unrelated_components_still_halt(self):
        for path in ("docs/ai/rwx-migration.md", "docs/public-docs/example.md",
                     "op-node/rollup/example.go", "rust/kona/example.rs"):
            with self.subTest(path=path):
                self.assert_build_decision([path], build=False)

    def test_ci_patterns_match_only_root_directories(self):
        for path in ("nested/.rwx/contracts.yml", ".rwx-other/contracts.yml",
                     "component/ops/ci/script.sh", "ops/circleci/script.sh",
                     "ops/ci-script.sh"):
            with self.subTest(path=path):
                self.assert_build_decision([path], build=False)

    def test_shared_ci_changes_execute_alongside_docs(self):
        self.assert_build_decision(["docs/ai/rwx-migration.md", "ops/ci/contracts-tests.sh"],
                                   build=True)


if __name__ == "__main__":
    unittest.main()
