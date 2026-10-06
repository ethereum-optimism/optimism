#!/usr/bin/env python3
"""Require every original shard, dependency and fresh verdict in the E2E gate."""
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('e2e_gate', Path(__file__).resolve().parents[1] / 'runtime' / 'rust-e2e-gate.py')
GATE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(GATE)
E2E = GATE.E2E


class GateTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name); self.sha = 'a' * 40
        self.release = self.root / 'release'; self.release.mkdir()
        E2E.write(self.release / 'settings.json', {'source_sha': self.sha, 'suite': 'rust-e2e',
            'job': 'release', 'profile': 'release', 'features': ['default']})
        E2E.write(self.release / 'coverage.json', {'binding': {'source_sha': self.sha}, 'packages': ['fixture'], 'targets': ['target']})
        self.seal(self.release)
        self.reports = []
        for job, config in E2E.JOBS.items():
            names = ['TestNew', 'TestOld']; shards = E2E.partition(names, config['shards'])
            for index, assigned in enumerate(shards):
                report = self.root / f'{job}-{index}'; report.mkdir(); self.reports.append(report)
                E2E.write(report / 'selection.json', {'source_sha': self.sha, 'settings': E2E.settings(job),
                    'provider': 'rwx', 'shard_index': index, 'assigned_tests': assigned, 'tests': names,
                    'excluded_listing': [], 'rwx_run_id': 'native-run', 'rwx_task_attempt': '1',
                    'workspace_root': '/workspace', 'parallel_flag': 2 if job == 'proof' else None,
                    'binaries_sha256': {'suite.test': 'b' * 64, 'test2json': 'c' * 64}})
                (report / 'invocations').mkdir()
                for invocation in ([[name] for name in assigned] if config['per_test'] else [assigned] if assigned else []):
                    pattern = '^(' + '|'.join(invocation) + ')$'
                    E2E.write(report / 'invocations' / ((invocation[0] if config['per_test'] else 'package') + '.json'),
                        {'source_sha': self.sha, 'assigned_tests': invocation,
                         'argv': E2E.GO.binary_command('/compiled/test2json', '/compiled/suite.test', E2E.PREFIX + config['package'],
                                                     2 if job == 'proof' else None, config['timeout'], pattern),
                         'cwd': '/workspace/' + config['package'], 'exit_code': 0,
                         'binary_sha256': 'b' * 64, 'reporter_sha256': 'c' * 64})
                E2E.write(report / 'coverage.json', {'assigned': assigned, 'missing': [], 'extra': [], 'duplicates': [],
                    'cases': {name: {'outcome': 'pass', 'elapsed': None} for name in assigned}})
                (report / 'original.json').write_text(''.join(json.dumps({'Action': 'pass', 'Test': name,
                    'Package': E2E.PREFIX + config['package']}) + '\n' for name in assigned))
                (report / 'dependencies').mkdir()
                kinds = ['go', 'contracts-e2e', 'rust-e2e-release']
                if job in ('restart', 'simple-kona', 'simple-kona-sequencer'): kinds.append('prestate')
                for kind in kinds:
                    E2E.write(report / 'dependencies' / (kind + '.json'), {'kind': kind, 'commit_sha': self.sha, 'files': {'artifact': 'hash'}})
                self.seal(report)

    def seal(self, directory, status=0):
        E2E.write(directory / 'final.json', {'exit_code': status, 'report_errors': [],
            'original_sha256': {str(p.relative_to(directory)): E2E.GO.digest(p) for p in directory.rglob('*')
                               if p.is_file() and p.name != 'final.json'}})

    def validate(self, reports=None): return GATE.validate(self.release, self.reports if reports is None else reports, self.sha)

    def test_complete_discovery_with_empty_shards_passes(self):
        self.assertTrue(self.validate()['passed'])

    def test_missing_or_duplicate_shard_cannot_pass(self):
        for rows in (self.reports[:-1], self.reports + [self.reports[0]]):
            with self.assertRaisesRegex(ValueError, 'shard'): self.validate(rows)

    def test_failed_or_corrupt_originals_cannot_pass(self):
        self.seal(self.reports[0], 17)
        with self.assertRaisesRegex(ValueError, 'Unsuccessful'): self.validate()
        self.seal(self.reports[0])
        (self.reports[0] / 'coverage.json').write_text('corrupt')
        with self.assertRaisesRegex(ValueError, 'corrupt'): self.validate()

    def test_invalid_attempt_stale_source_and_unassigned_tests_rejected(self):
        report = self.reports[0]; original = E2E.read(report / 'selection.json')
        for key, value in (('rwx_task_attempt', '0'), ('rwx_task_attempt', '01'), ('rwx_task_attempt', 2),
                           ('rwx_run_id', None), ('source_sha', 'stale'), ('assigned_tests', [])):
            E2E.write(report / 'selection.json', original | {key: value})
            coverage = E2E.read(report / 'coverage.json'); coverage['assigned'] = value if key == 'assigned_tests' else original['assigned_tests']
            E2E.write(report / 'coverage.json', coverage); self.seal(report)
            with self.subTest(key=key), self.assertRaises(ValueError): self.validate()

    def test_successful_retried_shard_retains_attempt_and_failed_retry_cannot_pass(self):
        report = self.reports[0]
        E2E.write(report / 'selection.json', E2E.read(report / 'selection.json') | {'rwx_task_attempt': '2'})
        self.seal(report)
        summary = self.validate()
        self.assertTrue(summary['passed'])
        self.assertEqual(summary['jobs']['proof']['task_attempts']['0'], '2')
        self.assertEqual(summary['jobs']['proof']['task_attempts']['1'], '1')
        self.seal(report, 17)
        with self.assertRaisesRegex(ValueError, 'Unsuccessful'): self.validate()

    def test_missing_or_stale_runtime_dependency_cannot_pass(self):
        path = self.reports[-1] / 'dependencies/rust-e2e-release.json'
        E2E.write(path, {'kind': 'rust-e2e-release', 'commit_sha': 'stale', 'files': {'artifact': 'hash'}})
        self.seal(self.reports[-1])
        with self.assertRaisesRegex(ValueError, 'provenance'): self.validate()
        path.unlink(); self.seal(self.reports[-1])
        with self.assertRaisesRegex(ValueError, 'dependency'): self.validate()

    def test_missing_original_execution_is_not_a_successful_coverage_stub(self):
        report = self.reports[0]
        (report / 'original.json').write_text('')
        self.seal(report)
        with self.assertRaisesRegex(ValueError, 'original verdicts'): self.validate()

    def test_missing_invocation_changed_flags_and_wrong_binary_cannot_pass(self):
        path = self.reports[-1] / 'invocations/package.json'; original = E2E.read(path)
        for data in (original | {'binary_sha256': 'stale'}, original | {'argv': original['argv'] + ['-test.count=2']}):
            E2E.write(path, data); self.seal(self.reports[-1])
            with self.assertRaises(ValueError): self.validate()
        path.unlink(); self.seal(self.reports[-1])
        with self.assertRaisesRegex(ValueError, 'invocation'): self.validate()


if __name__ == '__main__': unittest.main()
