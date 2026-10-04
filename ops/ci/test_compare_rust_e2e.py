#!/usr/bin/env python3
"""Reject missing originals, divergent skips, duplicate cases and incomplete invocations."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('compare_e2e', Path(__file__).with_name('compare-rust-e2e.py'))
COMPARE = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(COMPARE)
E2E = COMPARE.E2E


class CompareTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name); self.sha = 'a' * 40
        self.directory = self.root / 'simple-kona-0'; self.directory.mkdir()
        self.names = ['TestPass', 'TestSkip']; self.package = E2E.PREFIX + E2E.JOBS['simple-kona']['package']
        record = {'source_sha': self.sha, 'settings': E2E.settings('simple-kona'), 'provider': 'rwx',
            'workspace_root': '/workspace', 'assigned_tests': self.names, 'rwx_run_id': 'native', 'rwx_task_attempt': '1', 'parallel_flag': None,
            'binaries_sha256': {'suite.test': 'b' * 64, 'test2json': 'c' * 64}}
        E2E.write(self.directory / 'selection.json', record)
        E2E.write(self.directory / 'coverage.json', {'assigned': self.names, 'missing': [], 'extra': [], 'duplicates': []})
        (self.directory / 'assigned.txt').write_text('\n'.join(self.names) + '\n')
        (self.directory / 'environment.nul').write_bytes(b'key\0value\0')
        events = [{'Package': self.package, 'Action': 'start'}]
        for name, action in [('TestPass', 'pass'), ('TestSkip', 'skip')]:
            events.extend([{'Package': self.package, 'Action': 'run', 'Test': name},
                {'Package': self.package, 'Action': 'output', 'Test': name, 'Output': 'fixture_test.go:7: original reason\n'},
                {'Package': self.package, 'Action': action, 'Test': name, 'Elapsed': 0}])
        events.append({'Package': self.package, 'Action': 'pass', 'Elapsed': 0})
        (self.directory / 'original.json').write_text('\n'.join(json.dumps(e) for e in events) + '\n')
        E2E.PROJECT.project(self.directory / 'original.json', self.directory / 'native.json')
        (self.directory / 'junit').mkdir()
        (self.directory / 'junit/package.xml').write_text(f'<testsuite name="{self.package}"><testcase classname="{self.package}" name="TestPass"/><testcase classname="{self.package}" name="TestSkip"><skipped message="original reason"/></testcase></testsuite>')
        (self.directory / 'invocations').mkdir()
        argv = E2E.GO.binary_command('/compiled/test2json', '/compiled/suite.test', self.package, None, '40m', '^(TestPass|TestSkip)$')
        E2E.write(self.directory / 'invocations/package.json', {'source_sha': self.sha, 'assigned_tests': self.names,
            'argv': argv, 'cwd': '/workspace/' + E2E.JOBS['simple-kona']['package'], 'exit_code': 0,
            'binary_sha256': 'b' * 64, 'reporter_sha256': 'c' * 64})
        self.seal()

    def seal(self):
        E2E.write(self.directory / 'final.json', {'exit_code': 0, 'report_errors': [],
            'original_sha256': {str(p.relative_to(self.directory)): E2E.GO.digest(p) for p in self.directory.rglob('*')
                               if p.is_file() and p.name != 'final.json'}})

    def validate(self): return COMPARE.test_report(self.directory, 'simple-kona', self.sha, 'rwx', [], {})

    def test_complete_original_verdicts_skips_and_invocation(self):
        record, cases, history = self.validate()
        self.assertEqual(cases[(self.package, 'TestSkip')]['skip_reason'], 'original reason')
        self.assertEqual(record['assigned_tests'], self.names)
        self.assertEqual(history[0]['outcomes'], ['pass'])

    def test_corrupt_original_or_missing_junit_rejected(self):
        (self.directory / 'original.json').write_text('corrupt')
        with self.assertRaises((ValueError, OSError)): self.validate()

    def test_junit_different_outcome_cannot_pass(self):
        p = self.directory / 'junit/package.xml'; p.write_text(p.read_text().replace('<skipped message="original reason"/>', '<failure/>'))
        self.seal()
        with self.assertRaisesRegex(ValueError, 'JUnit'): self.validate()

    def test_missing_or_failed_invocation_cannot_pass(self):
        path = self.directory / 'invocations/package.json'; original = E2E.read(path)
        for data in (original | {'exit_code': 1}, original | {'assigned_tests': ['TestPass']}):
            E2E.write(path, data); self.seal()
            with self.assertRaises(ValueError): self.validate()
        path.unlink(); self.seal()
        with self.assertRaisesRegex(ValueError, 'invocation'): self.validate()

    def test_stale_source_and_task_retries_need_investigation(self):
        path = self.directory / 'selection.json'; original = E2E.read(path)
        for data in (original | {'source_sha': 'stale'}, original | {'rwx_task_attempt': '2'}):
            E2E.write(path, data); self.seal()
            with self.assertRaises(ValueError): self.validate()

    def test_original_package_failure_is_not_hidden_by_passing_cases(self):
        p = self.directory / 'original.json'; rows = [json.loads(line) for line in p.read_text().splitlines()]
        rows[-1]['Action'] = 'fail'; p.write_text('\n'.join(json.dumps(e) for e in rows) + '\n'); self.seal()
        with self.assertRaisesRegex(ValueError, 'package/case failure'): self.validate()

    def test_circle_empty_logs_are_explicit_not_missing_verdicts(self):
        path = self.directory / 'empty.log'; path.write_bytes(b''); self.seal(); path.unlink()
        empty = []
        COMPARE.originals(self.directory, {'empty.log'}, empty, 'circle/report')
        self.assertEqual(empty, [{'report': 'circle/report', 'path': 'empty.log', 'sha256': COMPARE.EMPTY}])
        (self.directory / 'original.json').unlink()
        with self.assertRaises(OSError): COMPARE.originals(self.directory, {'original.json'}, [], 'circle/report')

    def test_observed_finalized_skip_preserves_both_reasons_and_only_resolves_logger_time(self):
        source = 'func TestL2FinalizedSync(gt *testing.T) {\n t := devtest.ParallelT(gt)\n t.Skip("Skipping finalized sync test")\n}'
        circle = 'INFO [10-04|19:54:11.285] Running test in parallel scope=/TestL2FinalizedSync INFO [10-04|19:55:06.892] Skipping finalized sync test scope=/TestL2FinalizedSync'
        rwx = circle.replace('19:54:11.285', '19:45:45.854').replace('19:55:06.892', '19:47:03.335')
        key = (self.package, 'TestL2FinalizedSync')
        result = COMPARE.skip_comparison('simple-kona', key, circle, rwx, source)
        self.assertEqual(result['original_reasons'], {'circle': circle, 'rwx': rwx})
        self.assertEqual(result['comparison'], 'logger timestamps only')
        for invalid in (rwx.replace('INFO', 'WARN'), rwx.replace('Skipping finalized sync test', 'Different skip'),
                        rwx.replace('scope=/TestL2FinalizedSync', 'scope=/OtherTest'), rwx.replace('10-04', '99-99')):
            with self.subTest(invalid=invalid), self.assertRaises(ValueError):
                COMPARE.skip_comparison('simple-kona', key, circle, invalid, source)
        with self.assertRaisesRegex(ValueError, 'source'):
            COMPARE.skip_comparison('simple-kona', key, circle, rwx, source.replace('t.Skip(', 'if changed { t.Skip('))
        with self.assertRaises(ValueError): COMPARE.skip_comparison('proof', key, circle, rwx, source)
        with self.assertRaises(ValueError): COMPARE.skip_comparison('simple-kona', (self.package, 'OtherTest'), circle, rwx, source)


if __name__ == '__main__': unittest.main()
