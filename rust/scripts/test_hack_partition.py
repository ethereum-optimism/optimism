#!/usr/bin/env python3
"""Contract tests for the rust-cargo-hack node split."""

from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path

MODULE_PATH = Path(__file__).with_name("hack-partition.py")
SPEC = importlib.util.spec_from_file_location("hack_partition", MODULE_PATH)
assert SPEC and SPEC.loader
partition = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = partition
SPEC.loader.exec_module(partition)

PACKAGES = [f"crate-{n}" for n in range(30)]
WEIGHTS = {package: 1 + (n * 7) % 11 for n, package in enumerate(PACKAGES)}


class SplitTests(unittest.TestCase):
    def test_every_package_lands_on_exactly_one_node_in_workspace_order(self) -> None:
        for nodes in range(1, len(PACKAGES) + 1):
            ranges = partition.split(PACKAGES, WEIGHTS, nodes)
            self.assertEqual(len(ranges), nodes)
            self.assertTrue(all(ranges))
            self.assertEqual([p for r in ranges for p in r], PACKAGES)

    def test_a_package_missing_from_the_table_is_still_assigned(self) -> None:
        for nodes in (1, 3, 10):
            ranges = partition.split(PACKAGES, {}, nodes)
            self.assertEqual([p for r in ranges for p in r], PACKAGES)

    def test_a_table_row_for_a_removed_package_changes_nothing(self) -> None:
        stale = {**WEIGHTS, "removed-crate": 1000}
        self.assertEqual(partition.split(PACKAGES, stale, 7), partition.split(PACKAGES, WEIGHTS, 7))

    def test_a_heavy_package_gets_a_node_to_itself(self) -> None:
        ranges = partition.split(["a", "b", "heavy", "c"], {"heavy": 100, "a": 1, "b": 1, "c": 1}, 3)
        self.assertIn(["heavy"], ranges)

    def test_more_nodes_than_packages_fails(self) -> None:
        with self.assertRaises(ValueError):
            partition.split(["a"], {}, 2)
        with self.assertRaises(ValueError):
            partition.split([], {}, 1)


class ParseTests(unittest.TestCase):
    def test_partition(self) -> None:
        self.assertEqual(partition.parse_partition("3/10"), (3, 10))
        for bad in ("", "3", "0/10", "11/10", "a/b", "3/10/1", "-1/10"):
            with self.assertRaises(ValueError, msg=bad):
                partition.parse_partition(bad)

    def test_weights_ignore_comments_and_blank_lines(self) -> None:
        self.assertEqual(partition.read_weights("# note\n\na 3  # why\nb 4\n"), {"a": 3, "b": 4})

    def test_the_checked_in_table_parses(self) -> None:
        self.assertTrue(partition.read_weights(partition.WEIGHTS_FILE.read_text()))


if __name__ == "__main__":
    unittest.main()
