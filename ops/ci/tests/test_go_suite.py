#!/usr/bin/env python3
"""Check the manifest that Go discovery writes."""

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
MODULE = "github.com/ethereum-optimism/optimism"


class GoSuiteTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.timings = self.root / "timings.json"
        self.timings.write_text(json.dumps({"packages": {MODULE + "/op-e2e": 60, MODULE + "/op-batcher": 1}}))
        self.objects = [{"ImportPath": MODULE + "/op-batcher", "TestGoFiles": ["batcher_test.go"]},
                        {"ImportPath": MODULE + "/op-e2e", "XTestGoFiles": ["e2e_test.go"]},
                        {"ImportPath": MODULE + "/op-service/mocks", "GoFiles": ["mocks.go"]}]
        self.calls = []

    def discover(self, total, selected=None, objects=None):
        outputs = {"just": "\n".join(self.packages() if selected is None else selected) + "\n",
                   "go": "\n".join(json.dumps(item) for item in (self.objects if objects is None else objects))}
        def command(argv, **_):
            self.calls.append(argv)
            return outputs[argv[0]]
        with patch.object(SUITE, "ROOT", self.root), patch.object(SUITE.subprocess, "check_output", side_effect=command), \
                patch.dict(os.environ, {"CI_COMMIT_SHA": "a" * 40}):
            SUITE.discover(total, self.timings)
        return json.loads((self.root / ".ci/go-tests/manifest.json").read_text())

    def packages(self):
        return [item["ImportPath"] for item in self.objects]

    def test_manifest_holds_the_commit_every_package_once_and_the_packages_without_tests(self):
        manifest = self.discover(2)
        self.assertEqual(set(manifest), {"commit_sha", "shards", "no_tests"})
        self.assertEqual(manifest["commit_sha"], "a" * 40)
        # The 60-second package gets a shard of its own.
        self.assertEqual(manifest["shards"], [[MODULE + "/op-e2e"], [MODULE + "/op-batcher", MODULE + "/op-service/mocks"]])
        self.assertEqual(manifest["no_tests"], [MODULE + "/op-service/mocks"])

    def test_discovery_lists_the_recipe_selection_with_the_ci_tag(self):
        self.discover(1)
        self.assertEqual(self.calls, [["just", "list-test-packages"],
                                      ["go", "list", "-e", "-tags=ci", "-json", *self.packages()]])

    def test_empty_selection_and_go_list_errors_fail(self):
        with self.assertRaisesRegex(ValueError, "no packages"):
            self.discover(1, selected=[])
        broken = [{"ImportPath": MODULE + "/op-batcher", "DepsErrors": [{"Err": "missing bundle"}]}]
        with self.assertRaisesRegex(ValueError, "go list reported an error"):
            self.discover(1, objects=broken)
        self.assertFalse((self.root / ".ci/go-tests/manifest.json").exists())


if __name__ == "__main__":
    unittest.main()
