#!/usr/bin/env python3
"""Reject omissions and changed evidence using the complete original API fixture."""
import copy
import hashlib
import importlib.util
import json
from pathlib import Path
import shutil
import unittest
from unittest import mock


def helper(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).resolve().parents[2] / ('migration' if name.startswith('compare-') else 'migration/tests' if name=='test_flaky_report' else 'tests' if name.startswith('test_') else 'migration' if name=='flaky-report' else 'runtime') / (name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


C = helper('compare-flaky-report')
T = helper('test_flaky_report')


class ReportComparisonTests(unittest.TestCase):
    def setUp(self):
        self.generator = T.FlakyTests(); self.generator.setUp()
        self.addCleanup(self.generator.doCleanups)
        result, report = self.generator.generate()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.directory = self.generator.root / 'originals'; self.directory.mkdir()
        shutil.copytree(report, self.directory / 'reports')
        self.cwd = '/tmp/provider workspace'
        self.inputs = {name: b'committed fixture ' + name.encode() for name in C.F.INPUTS}
        C.F.write(self.directory / 'settings.json', {
            'source_sha': 'a' * 40, 'branch': 'codex/rwx-ci-pilot', 'provider': 'rwx',
            'scope': 'project-wide branch-agnostic Circle Insights', 'authenticated': False,
            'api_url': 'https://circleci.com/api/v2/insights/gh/ethereum-optimism/optimism/flaky-tests?branch=codex%2Frwx-ci-pilot',
            'input_sha256': {name: hashlib.sha256(raw).hexdigest() for name, raw in self.inputs.items()},
            'jq': 'jq-1.7.1', 'curl': 'curl fixture version', 'native_run_id': 'b' * 32, 'native_task_attempt': '1'})
        argv = ['bash', self.cwd + '/' + C.F.SCRIPT, '--branch', 'codex/rwx-ci-pilot', '--org', 'ethereum-optimism',
                '--repo', 'optimism', '--output-dir', self.cwd + '/.ci/flaky-report/run/reports']
        (self.directory / 'stdout.log').write_text(result.stdout.replace(str(report), argv[-1]))
        (self.directory / 'stderr.log').write_text(result.stderr)
        C.F.write(self.directory / 'execution.json', {'argv': argv, 'cwd': self.cwd, 'exit_code': 0,
            'signals': [], 'timed_out': False, 'started_at': 1700000000.0, 'elapsed_seconds': 1.0,
            'stdout_sha256': C.F.digest(self.directory / 'stdout.log'), 'stderr_sha256': C.F.digest(self.directory / 'stderr.log')})
        C.F.write(self.directory / 'coverage.json', {'source_rows': 79, 'acceptance_rows': 12,
            'scope': 'project-wide branch-agnostic Circle Insights'})
        self.seal()

    def seal(self):
        C.F.write(self.directory / 'final.json', {'state': 'passed', 'exit_code': 0, 'tests': 0, 'reports': 1,
            'errors': [], 'sha256': C.F.files(self.directory)})

    def read_report(self):
        with mock.patch.object(C.G, 'source', side_effect=lambda name, _: self.inputs[name]):
            return C.report(self.directory)

    def test_complete_nonempty_report_and_all_source_rows(self):
        value = self.read_report()
        self.assertEqual(value['rows'], {'source_rows': 79, 'acceptance_rows': 12})
        self.assertEqual(len(value['original']['flaky_tests']), 79)
        self.assertEqual(C.equal(value, copy.deepcopy(value))['attempts']['rwx'][0]['status'], '200')

    def test_corrupt_or_missing_originals_fail_even_with_a_new_seal(self):
        path = self.directory / 'reports/flaky_tests.csv'; raw = path.read_bytes()
        path.write_bytes(raw + b'changed')
        with self.assertRaisesRegex(ValueError, 'corrupt original'): self.read_report()
        self.seal()
        with self.assertRaises((ValueError, IndexError, json.JSONDecodeError)): self.read_report()
        path.write_bytes(raw); self.seal()
        path = self.directory / 'reports/api-attempt-1.stderr.log'; path.unlink(); self.seal()
        with self.assertRaises((ValueError, FileNotFoundError)): self.read_report()

    def test_changed_source_tool_or_process_cannot_be_resealed(self):
        path = self.directory / 'settings.json'; original = C.F.read(path)
        for key, changed in (('input_sha256', {}), ('jq', 'jq-0'), ('api_url', 'https://example.org'), ('authenticated', True)):
            C.F.write(path, original | {key: changed}); self.seal()
            with self.subTest(field=key), self.assertRaises(ValueError): self.read_report()
        C.F.write(path, original)
        path = self.directory / 'execution.json'; original = C.F.read(path)
        for key, changed in (('exit_code', 1), ('timed_out', True), ('signals', [15]), ('argv', ['true']), ('elapsed_seconds', 0)):
            C.F.write(path, original | {key: changed}); self.seal()
            with self.subTest(field=key), self.assertRaises(ValueError): self.read_report()

    def test_full_unfiltered_api_rows_and_dates_must_agree(self):
        original = self.read_report()
        for key in ('original', 'filtered', 'csv', 'html', 'stdout', 'rows'):
            changed = copy.deepcopy(original)
            if key in ('original', 'filtered'): changed[key]['flaky_tests'][0]['workflow_created_at'] = '2020-01-01T00:00:00Z'
            elif key == 'rows': changed[key]['source_rows'] -= 1
            elif key == 'stdout': changed[key] += 'extra omitted row\n'
            else: changed[key] += b'changed'
            with self.subTest(component=key), self.assertRaisesRegex(ValueError, 'parity differs'): C.equal(original, changed)

    def test_retry_history_is_retained_as_a_provider_difference(self):
        value = self.read_report(); changed = copy.deepcopy(value)
        changed['attempts'].insert(0, {'number': 1, 'status': '503', 'exit_code': 22,
            'body_sha256': 'c' * 64, 'stderr_sha256': 'd' * 64})
        changed['attempts'][1]['number'] = 2
        compared = C.equal(value, changed)
        self.assertEqual([row['status'] for row in compared['attempts']['rwx']], ['503', '200'])
        changed['attempts'][-1]['status'] = '403'
        with self.assertRaisesRegex(ValueError, 'final original'): C.equal(value, changed)

    def test_orphan_attempts_and_nonfinite_api_values_fail(self):
        path = self.directory / 'reports/api-attempt-9.json'; path.write_text('{}'); self.seal()
        with self.assertRaisesRegex(ValueError, 'attempt files'): self.read_report()
        path.unlink()
        path = self.directory / 'reports/flaky_tests.original.json'
        value = C.F.read(path); value['flaky_tests'][0]['time_wasted'] = 987654.321
        path.write_text(json.dumps(value).replace('987654.321', '1e309', 1))
        with self.assertRaisesRegex(ValueError, 'provenance'): C.F.validate_api(path)


if __name__ == '__main__': unittest.main()
