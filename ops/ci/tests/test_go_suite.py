#!/usr/bin/env python3
"""Authoritative selection, tagged discovery and effective settings evidence."""

import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch


SPEC = importlib.util.spec_from_file_location("suite", Path(__file__).resolve().parents[1] / 'runtime' / 'go-suite.py')
SUITE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SUITE)


class GoSuiteTest(unittest.TestCase):
    def setUp(self):
        self.packages = [SUITE.MODULE + "/op-batcher", SUITE.MODULE + "/op-e2e"]

    def test_tagged_discovery_must_exactly_match_selection(self):
        source = "\n".join(json.dumps({"ImportPath": p}) for p in reversed(self.packages))
        self.assertEqual(SUITE.validate_discovery(self.packages, source, SUITE.MODULE), self.packages)
        for selected in (self.packages[:1], self.packages + [SUITE.MODULE + "/new"], self.packages * 2):
            with self.subTest(selected=selected), self.assertRaises(ValueError):
                SUITE.validate_discovery(selected, source, SUITE.MODULE)
        broken = json.dumps({"ImportPath": self.packages[0], "DepsErrors": [{"Err": "missing bundle"}]})
        with self.assertRaises(ValueError):
            SUITE.validate_discovery(self.packages[:1], broken, SUITE.MODULE)

    def test_independent_test_concurrency_is_retained(self):
        for parallel in (8, 16, 32):
            with patch.dict(os.environ, {"PARALLEL": str(parallel)}):
                self.assertEqual(SUITE.settings()["parallel"], parallel)

    def test_discovery_retains_fresh_selection_settings_and_no_secret_values(self):
        source = "\n".join(json.dumps({"ImportPath": p}) for p in self.packages)
        outputs = {"list-test-packages": "\n".join(self.packages) + "\n", "list": source, "version": "go version fixture"}
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(SUITE, "ROOT", root), patch.object(SUITE, "go_environment", return_value={"GOOS": "linux"}), \
                    patch.object(SUITE.subprocess, "check_output", side_effect=lambda argv, **_: outputs[argv[1]]), \
                    patch.dict(os.environ, {"CI_COMMIT_SHA": "a" * 40, "CI_BRANCH": "feature/example", "PARALLEL": "8",
                                            "TEST_TIMEOUT": "40m", "OP_CI_MAINNET_L1_ARCHIVE_RPC_URL": "secret-fixture-never-retained"}):
                SUITE.discover("go-tests", "1", None)
            text = (root / ".ci/go-tests/manifest.json").read_text()
            manifest = json.loads(text)
            self.assertEqual(manifest["packages"], self.packages)
            self.assertEqual((root / ".ci/go-tests/all-packages.txt").read_text().splitlines(), self.packages)
            self.assertEqual((manifest["branch"], manifest["settings"]["count"], manifest["settings"]["short"]), ("feature/example", 1, False))
            self.assertEqual(manifest["settings"]["parallel"], 8)
            self.assertNotIn("secret-fixture", text)


if __name__ == "__main__":
    unittest.main()
