#!/usr/bin/env python3
"""Check `go list` reading and the placement of Go packages on shards."""

import importlib.util
import json
import unittest
from pathlib import Path


PREFIX = "github.com/ethereum-optimism/optimism/op-node/rollup"
SPEC = importlib.util.spec_from_file_location("go_package_shards", Path(__file__).resolve().parents[1] / 'runtime' / 'go-package-shards.py')
SHARDS = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SHARDS)


class PackageShardsTest(unittest.TestCase):
    def setUp(self):
        self.packages = [PREFIX, PREFIX + "/derive", PREFIX + "/derive/mocks", PREFIX + "/engine", PREFIX + "/sync"]

    def test_every_package_is_assigned_exactly_once(self):
        for total in (1, 2, 3, 8):
            with self.subTest(total=total):
                shards = SHARDS.assign(self.packages, total, {})
                self.assertEqual(len(shards), total)
                self.assertEqual(sorted(package for shard in shards for package in shard), sorted(self.packages))

    def test_assignment_does_not_depend_on_the_input_order(self):
        durations = {self.packages[0]: 30, self.packages[1]: 12}
        self.assertEqual(SHARDS.assign(self.packages, 2, durations),
                         SHARDS.assign(list(reversed(self.packages)), 2, durations))

    def test_durations_balance_long_packages_and_keep_packages_without_one(self):
        durations = {self.packages[0]: 30, self.packages[1]: 12, self.packages[2]: 6, self.packages[3]: 2}
        shards = SHARDS.assign(self.packages, 4, durations)
        self.assertEqual(sorted(package for shard in shards for package in shard), sorted(self.packages))
        self.assertEqual(shards[0], [self.packages[0]])
        # The package without a duration weighs the median, 9 seconds: more than the two shortest together.
        self.assertEqual(shards[2:], [[self.packages[4]], [self.packages[2], self.packages[3]]])

    def test_go_list_stream_is_read_in_full(self):
        objects = [{"ImportPath": package, "TestGoFiles": ["test.go"]} for package in self.packages]
        objects[2].pop("TestGoFiles")
        self.assertEqual(SHARDS.read_go_packages("\n".join(json.dumps(item, indent=1) for item in objects) + "\n"), objects)
        self.assertEqual(SHARDS.read_go_packages(""), [])

    def test_go_list_package_and_dependency_errors_fail(self):
        for error in ({"Error": {"Err": "compile error"}}, {"DepsErrors": [{"Err": "missing embed"}]}):
            with self.subTest(error=error):
                with self.assertRaisesRegex(ValueError, "go list reported an error"):
                    SHARDS.read_go_packages(json.dumps({"ImportPath": PREFIX, **error}))

    def test_truncated_go_list_stream_fails(self):
        with self.assertRaises(ValueError):
            SHARDS.read_go_packages(json.dumps({"ImportPath": PREFIX}) + "{")


if __name__ == "__main__":
    unittest.main()
