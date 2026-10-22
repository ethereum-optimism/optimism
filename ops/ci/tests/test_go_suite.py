#!/usr/bin/env python3
"""Authoritative selection, tagged discovery and effective Circle evidence."""

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

    def test_circle_retains_complete_selection_settings_and_no_secret_values(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            logs = root / "tmp/testlogs"
            logs.mkdir(parents=True)
            (logs / "all-packages.txt").write_text("\n".join(self.packages) + "\n")
            (logs / "discovery.json").write_text("\n".join(json.dumps({"ImportPath": p}) for p in self.packages))
            for fresh, expected in (("false", None), ("0", None), ("true", 1), ("1", 1)):
                with patch.object(SUITE, "ROOT", root), patch.object(SUITE, "go_environment", return_value={"GOOS":"linux"}), patch.object(SUITE.subprocess, "check_output", return_value="go version fixture"), patch.dict(os.environ, {
                    "CIRCLE_SHA1": "a" * 40, "CIRCLE_BRANCH": "codex/pilot",
                    "CIRCLE_NODE_TOTAL": "12", "CIRCLE_NODE_INDEX": "4",
                    "CI_GO_FRESH_TESTS": fresh, "PARALLEL": "8", "TEST_TIMEOUT": "40m",
                    "OP_CI_MAINNET_L1_ARCHIVE_RPC_URL": "secret-fixture-never-retained"}):
                    SUITE.record_circle("")
                evidence = json.loads((logs / "selection.json").read_text())
                self.assertEqual(evidence["packages"], self.packages)
                self.assertEqual(evidence["settings"]["count"], expected)
                self.assertEqual(evidence["settings"]["parallel"], 8)
                self.assertFalse(evidence["settings"]["short"])
                self.assertEqual(evidence["shard_index"], 4)
                self.assertNotIn("secret-fixture", (logs / "selection.json").read_text())


if __name__ == "__main__":
    unittest.main()
