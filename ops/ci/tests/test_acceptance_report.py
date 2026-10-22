#!/usr/bin/env python3
"""Original failure retention and complete acceptance verdict coverage."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('report', Path(__file__).resolve().parents[1] / 'runtime' / 'acceptance-report.py')
REPORT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(REPORT)


class AcceptanceReportTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.logs = self.root / 'op-acceptance-tests/logs/testrun-20261002'
        (self.logs / 'discovery').mkdir(parents=True)
        (self.logs / 'discovery/selection.json').write_text(json.dumps({'assigned_tests': [{'package': 'fixture', 'name': 'TestOne'}]}))
        results = self.root / 'op-acceptance-tests/results'
        results.mkdir()
        (results / 'results-0.xml').write_text('<testsuite failures="1"/>')
        self.original = self.logs / 'raw_go_events-0.log'
        self.original.write_text('\n'.join(json.dumps({'Action': a, 'Package': 'fixture', 'Test': 'TestOne'}) for a in ['run', 'fail']) + '\n')
        self.output = self.root / 'reports'
        patch.object(REPORT, 'ROOT', self.root).start()
        self.addCleanup(patch.stopall)

    def test_original_failures_and_native_projection_are_retained(self):
        REPORT.collect(self.output)
        self.assertEqual((self.output / 'original.json').read_bytes(), self.original.read_bytes())
        self.assertTrue((self.output / 'native.json').exists())
        self.assertEqual(json.loads((self.output / 'coverage.json').read_text())['missing'], [])

    def test_partial_failure_retains_reports_before_rejecting_missing_cases(self):
        self.original.write_text(json.dumps({'Action': 'output', 'Package': 'fixture', 'Output': 'compiler failed\n'}) + '\n')
        with self.assertRaises(ValueError): REPORT.collect(self.output)
        self.assertTrue((self.output / 'original.json').exists())
        self.assertTrue((self.output / 'junit/results-0.xml').exists())
        self.assertEqual(len(json.loads((self.output / 'coverage.json').read_text())['missing']), 1)

    def test_duplicate_or_unassigned_verdicts_are_rejected(self):
        original = self.original.read_text()
        for name in ['TestOne', 'TestExtra']:
            self.original.write_text(original + json.dumps({'Action': 'pass', 'Package': 'fixture', 'Test': name}) + '\n')
            with self.assertRaises(ValueError): REPORT.collect(self.output)


if __name__ == '__main__': unittest.main()
