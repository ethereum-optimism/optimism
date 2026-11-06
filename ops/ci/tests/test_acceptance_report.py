#!/usr/bin/env python3
"""Original failure retention and complete acceptance verdict coverage."""
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('report', Path(__file__).resolve().parents[1] / 'runtime' / 'acceptance-report.py')
REPORT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(REPORT)
PACKAGE = REPORT.MANIFEST.PREFIX + '/base'


class AcceptanceReportTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.addCleanup(patch.stopall)
        patch.dict(os.environ, {'CI_COMMIT_SHA': 'a'*40}, clear=True).start()
        patch.object(REPORT.MANIFEST.subprocess, 'check_output', return_value='go version go1.26.6 linux/amd64\n').start()
        patch.object(REPORT, 'ROOT', self.root).start()
        # Discovery assigns TestOne to shard 0 and TestTwo to shard 1; shards 2 to 7 are empty.
        self.discovery = self.root / '.ci/acceptance/discovery'
        self.discovery.mkdir(parents=True)
        (self.discovery / 'packages.json').write_text(json.dumps({'ImportPath': PACKAGE}))
        events = [{'Action': 'output', 'Package': PACKAGE, 'Output': n + '\n'} for n in ['TestOne', 'TestTwo']]
        events.append({'Action': 'pass', 'Package': PACKAGE})
        (self.discovery / 'listing.json').write_text('\n'.join(json.dumps(e) for e in events))
        manifest = REPORT.MANIFEST.create(self.discovery / 'packages.json', self.discovery / 'listing.json', 8)
        (self.discovery / 'manifest.json').write_text(json.dumps(manifest))
        self.logs = self.root / 'op-acceptance-tests/logs/testrun-20261002-000000'
        self.logs.mkdir(parents=True)
        # The recipe runs one go test per test and names its reports "<shard>-<package>.<test>".
        name = '0-' + PACKAGE.replace('/', '.') + '.TestOne'
        self.junit = self.root / 'op-acceptance-tests/results' / ('results-' + name + '.xml')
        self.junit.parent.mkdir()
        self.junit.write_text('<testsuite failures="1"/>')
        self.original = self.logs / ('raw_go_events-' + name + '.log')
        self.original.write_text('\n'.join(json.dumps({'Action': a, 'Package': PACKAGE, 'Test': 'TestOne'}) for a in ['run', 'fail']) + '\n')
        self.output = self.root / 'reports'

    def collect(self, index='0'):
        REPORT.collect(self.output, self.discovery, index, '8')

    def coverage(self):
        return json.loads((self.output / 'coverage.json').read_text())

    def test_original_failures_and_native_projection_are_retained(self):
        self.collect()
        self.assertEqual((self.output / 'original.json').read_bytes(), self.original.read_bytes())
        self.assertEqual((self.output / 'junit' / self.junit.name).read_bytes(), self.junit.read_bytes())
        self.assertTrue((self.output / 'native.json').exists())
        self.assertEqual(self.coverage()['missing'], [])

    def test_partial_failure_retains_reports_before_rejecting_missing_cases(self):
        self.original.write_text(json.dumps({'Action': 'output', 'Package': PACKAGE, 'Output': 'compiler failed\n'}) + '\n')
        with self.assertRaises(ValueError): self.collect()
        self.assertTrue((self.output / 'original.json').exists())
        self.assertTrue((self.output / 'junit' / self.junit.name).exists())
        self.assertEqual(len(self.coverage()['missing']), 1)

    def test_duplicate_or_unassigned_verdicts_are_rejected(self):
        original = self.original.read_text()
        for name in ['TestOne', 'TestTwo']:
            self.original.write_text(original + json.dumps({'Action': 'pass', 'Package': PACKAGE, 'Test': name}) + '\n')
            with self.assertRaises(ValueError): self.collect()

    def test_a_run_of_an_unassigned_test_is_rejected(self):
        extra = self.logs / ('raw_go_events-0-' + PACKAGE.replace('/', '.') + '.TestTwo.log')
        extra.write_text('')
        with self.assertRaises(ValueError): self.collect()
        self.assertEqual(self.coverage()['extra_json'], [extra.name.removeprefix('raw_go_events-').removesuffix('.log')])

    def test_empty_assignment_is_a_failed_split(self):
        # Like the recipe, which fails an empty shard.
        self.original.unlink(); self.junit.unlink()
        with self.assertRaisesRegex(ValueError, 'Empty'): self.collect('7')

    def test_assigned_test_without_its_reports_fails_after_retaining_logs(self):
        (self.logs / 'all.log').write_text('runner output\n')
        self.junit.unlink()
        with self.assertRaises(ValueError): self.collect()
        self.assertEqual(len(self.coverage()['missing_junit']), 1)
        self.assertTrue((self.output / 'original.json').exists())
        self.original.unlink(); self.output = self.root / 'without-json'
        with self.assertRaises(ValueError): self.collect()
        self.assertEqual(len(self.coverage()['missing_json']), 1)
        self.assertTrue((self.output / 'logs/all.log').exists())


if __name__ == '__main__': unittest.main()
