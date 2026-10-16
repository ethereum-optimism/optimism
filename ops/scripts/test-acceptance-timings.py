#!/usr/bin/env python3
"""Regression checks for acceptance-test sharding and its JUnit timing history."""

import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
import xml.etree.ElementTree as ET

SCRIPT_DIR = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("junit_shard_timings", SCRIPT_DIR / "junit-shard-timings.py")
timings = importlib.util.module_from_spec(spec)
spec.loader.exec_module(timings)


class JUnitTimingsTest(unittest.TestCase):
    def normalize(self, xml):
        tree = ET.fromstring(xml)
        timings.aggregate_timings(tree)
        return tree

    def test_parallel_subtests_contribute_to_top_level_timing(self):
        # These are the zero parent and real subtest timings from canceled job
        # 5717816, whose partial history overloaded the next run's ZK shard.
        tree = self.normalize('''<testsuites><testsuite name="zk">
          <testcase name="TestValidProposal" time="0"/>
          <testcase name="TestValidProposal/op-node" time="39.79"/>
          <testcase name="TestValidProposal/supernode" time="73.67"/>
        </testsuite></testsuites>''')
        self.assertAlmostEqual(float(tree.find(".//testcase").get("time")), 113.46)

    def test_nested_subtests_are_not_counted_twice(self):
        tree = self.normalize('''<testsuites><testsuite>
          <testcase name="TestRoot" time="0"/>
          <testcase name="TestRoot/nested" time="0"/>
          <testcase name="TestRoot/nested/a" time="10"/>
          <testcase name="TestRoot/nested/b" time="20"/>
          <testcase name="TestRoot/other" time="5"/>
        </testsuite></testsuites>''')
        cases = {c.get("name"): float(c.get("time")) for c in tree.iter("testcase")}
        self.assertEqual(cases["TestRoot"], 35)
        self.assertEqual(cases["TestRoot/nested"], 30)
        before = ET.tostring(tree)
        timings.aggregate_timings(tree)
        self.assertEqual(ET.tostring(tree), before)

    def test_serial_parent_keeps_its_inclusive_elapsed_time(self):
        tree = self.normalize('''<testsuites><testsuite>
          <testcase name="TestRoot" time="100"/>
          <testcase name="TestRoot/child" time="30"/>
        </testsuite></testsuites>''')
        self.assertEqual(tree.find(".//testcase").get("time"), "100")

    def test_missing_intermediate_results_and_distinct_packages(self):
        tree = self.normalize('''<testsuites>
          <testsuite name="first">
            <testcase name="TestRoot" time="0"/>
            <testcase name="TestRoot/missing/leaf" time="30"/>
          </testsuite>
          <testsuite name="second">
            <testcase name="TestRoot" time="0"/>
            <testcase name="TestRoot/child" time="10"/>
          </testsuite>
        </testsuites>''')
        self.assertEqual([float(s.find("testcase").get("time")) for s in tree], [30, 10])

    def test_cli_preserves_outcomes_and_unknown_failure_duration(self):
        xml = '''<testsuites><testsuite name="zk" tests="3" failures="1" time="120">
          <testcase name="TestRoot" classname="zk" time="0"><failure message="failed">details</failure></testcase>
          <testcase name="TestRoot/child" classname="zk" time="30"><system-out>logs</system-out></testcase>
          <testcase name="TestAborted" classname="zk" time="-1"><skipped message="aborted"/></testcase>
        </testsuite></testsuites>'''
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "results.xml"
            path.write_text(xml)
            subprocess.run([sys.executable, str(SCRIPT_DIR / "junit-shard-timings.py"), str(path)], check=True)
            tree = ET.parse(path)
        suite = tree.find("testsuite")
        self.assertEqual(suite.attrib, {"name": "zk", "tests": "3", "failures": "1", "time": "120"})
        root, child, aborted = suite.findall("testcase")
        self.assertEqual(float(root.get("time")), 30)
        self.assertEqual(root.find("failure").text, "details")
        self.assertEqual(child.find("system-out").text, "logs")
        self.assertEqual(aborted.get("time"), "-1")
        self.assertEqual(aborted.find("skipped").get("message"), "aborted")


class ShardTestsTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        # A test double for CircleCI's weighted splitter. Unequal known shard
        # loads reproduce why tiny missing weights all fit on one light shard.
        # The real CircleCI agent is available only inside CircleCI jobs.
        circleci = self.directory / "circleci"
        circleci.write_text(f'''#!{sys.executable}
import json, os, sys
args = sys.argv[1:]
assert args[:2] == ["tests", "split"]
assert "--split-by=timings" in args and "--timings-type=testname" in args
weight = 0.0001
for arg in args:
    if arg.startswith("--time-default="):
        value = arg.split("=", 1)[1]
        weight = float(value[:-1]) * {{"s": 1, "m": 60}}[value[-1]]
history = json.loads(os.environ["TEST_TIMINGS"])
names = open(args[-1]).read().splitlines()
loads = [0] * int(os.environ["CIRCLE_NODE_TOTAL"])
shards = [[] for _ in loads]
for name in sorted(names, key=lambda name: (-history.get(name, weight), name)):
    node = min(range(len(loads)), key=loads.__getitem__)
    loads[node] += history.get(name, weight)
    shards[node].append(name)
print("\\n".join(shards[int(os.environ["CIRCLE_NODE_INDEX"])]))
''')
        circleci.chmod(0o755)

    def run_sharder(self, names, node=0, total=8, history=None):
        env = dict(os.environ, PATH=f"{self.directory}{os.pathsep}{os.environ['PATH']}",
                   CIRCLE_NODE_TOTAL=str(total), CIRCLE_NODE_INDEX=str(node),
                   TEST_TIMINGS=json.dumps(history or {}))
        # Include the Go package summary and other non-test lines emitted by -list.
        listing = "\n".join(names + ["ok example.com/tests 0.001s", "? example.com/empty [no test files]"])
        return subprocess.run(["bash", str(SCRIPT_DIR / "shard-tests.sh")], input=listing,
                              text=True, capture_output=True, env=env)

    def test_partial_canceled_run_history_does_not_concentrate_unknown_tests(self):
        known = {f"TestKnown{i}": 1000 + i * 10 for i in range(8)}
        unknown = [f"TestUnknown{i:02}" for i in range(64)]
        names = list(known) + unknown
        shards = []
        for node in range(8):
            result = self.run_sharder(names, node=node, history=known)
            self.assertEqual(result.returncode, 0, result.stderr)
            shards.append(result.stdout.splitlines())
        self.assertCountEqual([name for shard in shards for name in shard], names)
        self.assertLessEqual(max(len(set(shard) & set(unknown)) for shard in shards), 9)

    def test_empty_history_covers_every_test_once(self):
        names = [f"TestNew{i:02}" for i in range(64)]
        shards = [self.run_sharder(names, node=node).stdout.splitlines() for node in range(8)]
        self.assertCountEqual([name for shard in shards for name in shard], names)
        self.assertTrue(all(len(shard) == 8 for shard in shards))

    def test_single_node_passes_all_test_names_through(self):
        result = self.run_sharder(["TestOne", "TestTwo"], total=1)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.splitlines(), ["TestOne", "TestTwo"])

    def test_empty_discovery_fails_instead_of_skipping_the_suite(self):
        result = self.run_sharder([])
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("no tests to shard", result.stderr)


if __name__ == "__main__":
    unittest.main()
