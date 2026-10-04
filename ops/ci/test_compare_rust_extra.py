#!/usr/bin/env python3
"""Reject damaged, stale, incomplete or mismatched provider evidence."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('compare_rust_extra', Path(__file__).with_name('compare-rust-extra.py'))
COMPARE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(COMPARE)
SHA = 'a' * 40


class ComparisonTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for provider in ('circle', 'rwx'):
            for job in COMPARE.JOBS:
                d = self.root / provider / job
                d.mkdir(parents=True)
                settings = {'job': job, 'source_sha': SHA, 'suite': 'rust-workspace',
                            'rwx_run_id': 'b' * 32, 'rwx_task_attempt': '1', 'input_sha256': {'Cargo.lock': 'c' * 64}}
                self.write(d / 'settings.json', settings)
                self.write(d / 'workspace.json', {'workspace_members': ['crate'], 'packages': [
                    {'id': 'crate', 'name': 'crate', 'version': '1', 'features': {}, 'targets': []}]})
                (d / 'checks.junit.xml').write_text('<testsuite><testcase name="check"/></testsuite>\n')
                stages = {}
                for name in ('workspace',) + COMPARE.STAGES[job]:
                    stages[name] = {'argv': ['real', name], 'cwd': 'rust', 'exit_code': 0}
                    self.write(d / (name + '.stage.json'), stages[name])
                    (d / (name + '.log')).write_text('original\n')
                if job.startswith('wasm-'):
                    self.write(d / 'wasm-coverage.json', {'selected': ['crate'], 'executed': ['crate']})
                    self.write(d / 'wasm-artifacts.json', {'source_sha': SHA, 'target': job, 'rustc': 'pinned',
                                                         'artifacts': {'libcrate.rlib': {'sha256': 'd' * 64}}})
                if job == 'registry':
                    self.write(d / 'registry-coverage.json', {'snapshots': ['chainList', 'configs', 'depsets']})
                    for phase in ('before', 'after'):
                        for name in ('chainList', 'configs', 'depsets'):
                            self.write(d / f'registry-{phase}-{name}.json', [1])
                if job == 'interop':
                    self.write(d / 'interop-coverage.json', {'cases': ['false', 'true']})
                    for language in ('go', 'rust'):
                        for suffix in ('stdout', 'stderr', 'exit'):
                            (d / f'interop-{language}.{suffix}').write_text('original\n')
                self.write(d / 'final.json', {'exit_code': 0, 'report_errors': [], 'stages': stages,
                    'original_sha256': {p.name: COMPARE.digest(p) for p in d.iterdir()}})

    def write(self, path, data):
        path.write_text(json.dumps(data) + '\n')

    def update(self, provider, job, filename, edit):
        d = self.root / provider / job
        data = COMPARE.read(d / filename)
        edit(data)
        self.write(d / filename, data)
        if filename != 'final.json':
            final = COMPARE.read(d / 'final.json')
            final['original_sha256'][filename] = COMPARE.digest(d / filename)
            self.write(d / 'final.json', final)

    def test_complete_reports_compare(self):
        self.assertTrue(COMPARE.compare(self.root, SHA)['verified_parity'])

    def test_corrupt_or_missing_nonempty_original_rejected(self):
        path = self.root / 'rwx/interop/interop-go.stderr'
        path.write_text('corrupt\n')
        with self.assertRaisesRegex(ValueError, 'corrupt original'):
            COMPARE.compare(self.root, SHA)
        path.unlink()
        with self.assertRaisesRegex(ValueError, 'missing original'):
            COMPARE.compare(self.root, SHA)

    def test_only_declared_empty_circle_streams_may_be_absent(self):
        d = self.root / 'circle/typos'
        path = d / 'typos.log'
        path.write_text('')
        self.update('circle', 'typos', 'final.json', lambda f: f['original_sha256'].update({'typos.log': COMPARE.digest(path)}))
        path.unlink()
        evidence = COMPARE.compare(self.root, SHA)
        self.assertEqual(evidence['manifest_declared_empty_logs'][0]['name'], 'typos.log')
        self.update('circle', 'typos', 'final.json', lambda f: f['original_sha256'].pop('typos.log'))
        with self.assertRaisesRegex(ValueError, 'incomplete original manifest'):
            COMPARE.compare(self.root, SHA)

    def test_stale_source_and_settings_drift_rejected(self):
        self.update('rwx', 'registry', 'settings.json', lambda s: s.update(source_sha='e' * 40))
        with self.assertRaisesRegex(ValueError, 'source or job mismatch'):
            COMPARE.compare(self.root, SHA)
        self.update('rwx', 'registry', 'settings.json', lambda s: s.update(source_sha=SHA, rustc='wrong'))
        with self.assertRaisesRegex(ValueError, 'settings differ'):
            COMPARE.compare(self.root, SHA)

    def test_missing_fresh_identity_or_unresolved_retry_rejected(self):
        self.update('rwx', 'typos', 'settings.json', lambda s: s.pop('rwx_run_id'))
        with self.assertRaisesRegex(ValueError, 'fresh identity'):
            COMPARE.compare(self.root, SHA)
        self.update('rwx', 'typos', 'settings.json', lambda s: s.update(rwx_run_id='b' * 32, rwx_task_attempt='2'))
        with self.assertRaisesRegex(ValueError, 'retries require investigation'):
            COMPARE.compare(self.root, SHA)

    def test_missing_manifest_stage_cannot_be_hidden(self):
        self.update('rwx', 'registry', 'final.json', lambda f: f['stages'].pop('registry-clean'))
        with self.assertRaisesRegex(ValueError, 'incomplete original manifest'):
            COMPARE.compare(self.root, SHA)

    def test_complete_package_manifest_and_coverage_must_agree(self):
        self.update('rwx', 'zepter', 'workspace.json', lambda m: m['packages'][0].update(features={'new': []}))
        with self.assertRaisesRegex(ValueError, 'workspace manifests differ'):
            COMPARE.compare(self.root, SHA)
        self.update('rwx', 'zepter', 'workspace.json', lambda m: m['packages'][0].update(features={}))
        self.update('rwx', 'interop', 'interop-coverage.json', lambda c: c.update(cases=['false']))
        with self.assertRaisesRegex(ValueError, 'full coverage differs'):
            COMPARE.compare(self.root, SHA)

    def test_failing_stage_or_changed_command_is_not_parity(self):
        self.update('rwx', 'zepter', 'final.json', lambda f: f['stages']['zepter'].update(exit_code=7))
        self.update('rwx', 'zepter', 'zepter.stage.json', lambda s: s.update(exit_code=7))
        with self.assertRaisesRegex(ValueError, 'outcomes differ'):
            COMPARE.compare(self.root, SHA)
        self.update('rwx', 'zepter', 'final.json', lambda f: f['stages']['zepter'].update(exit_code=0, argv=['no-op']))
        self.update('rwx', 'zepter', 'zepter.stage.json', lambda s: s.update(exit_code=0, argv=['no-op']))
        with self.assertRaisesRegex(ValueError, 'original commands'):
            COMPARE.compare(self.root, SHA)


if __name__ == '__main__':
    unittest.main()
