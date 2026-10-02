#!/usr/bin/env python3
"""Native reporting must preserve verdicts/retries and the immutable original."""
import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('report', Path(__file__).with_name('go-report.py'))
REPORT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(REPORT)


class GoReportTest(unittest.TestCase):
    def render(self, events):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        source, native = Path(temp.name) / 'original.json', Path(temp.name) / 'native.json'
        original = ''.join(json.dumps(e) + '\n' for e in events)
        source.write_text(original)
        REPORT.project(source, native)
        self.assertEqual(source.read_text(), original)
        metadata = json.loads(native.with_suffix('.metadata.json').read_text())
        self.assertEqual(metadata['original_sha256'], hashlib.sha256(original.encode()).hexdigest())
        return [json.loads(line) for line in native.read_text().splitlines()]

    def test_original_retries_and_skips_survive_without_passed_output_noise(self):
        events = [{'Action':'start', 'Package':'p'}]
        for outcome in ['fail', 'fail', 'fail', 'pass']:
            events += [{'Action':'run', 'Package':'p', 'Test':'TestRetry'},
                       {'Action':'output', 'Package':'p', 'Test':'TestRetry', 'Output':'original failure evidence'},
                       {'Action':outcome, 'Package':'p', 'Test':'TestRetry', 'Elapsed':1}]
        events += [{'Action':'output', 'Package':'p', 'Test':'TestPass', 'Output':'noisy passing output'},
                   {'Action':'pass', 'Package':'p', 'Test':'TestPass'},
                   {'Action':'output', 'Package':'p', 'Test':'TestSkip', 'Output':'specific skip reason'},
                   {'Action':'skip', 'Package':'p', 'Test':'TestSkip'}, {'Action':'pass', 'Package':'p'}]
        native = self.render(events)
        self.assertEqual([e for e in native if e['Action']!='output'], [e for e in events if e['Action']!='output'])
        self.assertEqual(len([e for e in native if e.get('Test')=='TestRetry' and e['Action']=='output']), 4)
        self.assertIn('specific skip reason', str(native))
        self.assertNotIn('noisy passing output', str(native))

    def test_failure_output_is_bounded_and_explicitly_marked(self):
        native = self.render([{'Action':'output', 'Package':'p', 'Test':'TestFailure', 'Output':'x' * (REPORT.LIMIT * 2)},
                              {'Action':'fail', 'Package':'p', 'Test':'TestFailure'}])
        self.assertIn('complete original events', native[0]['Output'])
        self.assertEqual(len(native[1]['Output'].encode()), REPORT.LIMIT)
        self.assertEqual(native[-1]['Action'], 'fail')

    def test_no_test_packages_and_canceled_partial_attempts_are_not_rewritten(self):
        events = [{'Action':'start', 'Package':'empty'}, {'Action':'skip', 'Package':'empty'},
                  {'Action':'run', 'Package':'p', 'Test':'TestCanceled'}]
        self.assertEqual(self.render(events), events)

    def test_malformed_original_report_is_rejected(self):
        with tempfile.TemporaryDirectory() as d:
            source = Path(d) / 'original.json'
            source.write_text('{broken}\n')
            with self.assertRaises(ValueError): REPORT.project(source, Path(d) / 'native.json')


if __name__ == '__main__': unittest.main()
