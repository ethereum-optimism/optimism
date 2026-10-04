#!/usr/bin/env python3
"""Fail closed for stale, damaged, incomplete and mismatched original checks."""
import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
import xml.etree.ElementTree as ET

SPEC = importlib.util.spec_from_file_location('compare_checks', Path(__file__).with_name('compare-pr-checks.py'))
COMPARE = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(COMPARE)
SHA = 'a' * 40


class OriginalCheckTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name); self.directories = {}
        for provider in ('circle', 'rwx'):
            directory = self.root / provider; directory.mkdir(); self.directories[provider] = directory
            workspace = '/' + provider
            self.write(directory / 'settings.json', {'source_sha': SHA, 'job': 'go-modules',
                'provider': 'circleci' if provider == 'circle' else 'rwx', 'workspace_root': workspace,
                'branch': 'pilot', 'go': 'go1.26.6', 'just': 'pinned', 'input_sha256': {'go.mod': 'original'}})
            (directory / 'discovery.json').write_text('{"Path":"main","Main":true}\n{"Path":"dependency","Version":"v1.0.0","Sum":"h1:original"}\n')
            self.write(directory / 'coverage.json', {'modules': COMPARE.CHECKS.modules((directory / 'discovery.json').read_text()),
                'selected': 2, 'verified': True, 'download_attempts': 1})
            commands = {'download-0': ['go', 'mod', 'download'], 'verify': ['go', 'mod', 'verify'],
                        'discovery': ['go', 'list', '-m', '-json', 'all']}
            for name, argv in commands.items():
                self.write(directory / (name + '.stage.json'), {'argv': argv, 'cwd': workspace, 'exit_code': 0})
                (directory / (name + '.log')).write_text('')
            suite = ET.Element('testsuite'); ET.SubElement(suite, 'testcase', name='go-modules', classname='go-modules')
            ET.ElementTree(suite).write(directory / 'checks.junit.xml')
            self.seal(directory)

    def write(self, path, value): path.write_text(json.dumps(value))

    def seal(self, directory):
        self.write(directory / 'final.json', {'exit_code': 0, 'report_errors': [], 'original_sha256': {
            p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in directory.iterdir() if p.name != 'final.json'}})

    def compare(self): return COMPARE.compare(self.directories, 'go-modules', SHA)

    def fast(self):
        configuration = {'phases': [{'name': 'original', 'checks': [
            {'name': 'first', 'command': 'original first'}, {'name': 'second', 'command': 'original second'}]}]}
        for provider, d in self.directories.items():
            settings = json.loads((d / 'settings.json').read_text())
            for p in d.iterdir(): p.unlink()
            settings.update(job='contracts-fast', forge='pinned Forge', semgrep='1.137.0',
                            rwx_run_id='c' * 32 if provider == 'rwx' else None, rwx_task_attempt='1',
                            target_branch='develop', target_sha='d' * 40, merge_base_sha='e' * 40)
            self.write(d / 'settings.json', settings); self.write(d / 'selection.json', configuration)
            (d / 'checks.log').write_text('✓ first 1.0s\n✓ second 2.0s\n✓ All checks passed (2/2)\n')
            self.write(d / 'coverage.json', COMPARE.CHECKS.check_verdicts((d / 'checks.log').read_text(), configuration))
            (d / 'submodules.txt').write_text(' ' + 'd' * 40 + ' contract-library (original-description)\n')
            self.write(d / 'foundry-config.json', {'optimizer': True, 'root': settings['workspace_root']})
            for name, argv in [('checks', ['just', 'check-fast', '-verbose']), ('foundry-config', ['forge', 'config', '--json']),
                               ('fetch-target', ['git', 'fetch', '--no-tags', 'origin', '+refs/heads/develop:refs/remotes/origin/develop'])]:
                self.write(d / (name + '.stage.json'), {'argv': argv, 'cwd': settings['workspace_root'] + ('/packages/contracts-bedrock' if name != 'fetch-target' else ''), 'exit_code': 0})
            suite = ET.Element('testsuite')
            for name in ('first', 'second'): ET.SubElement(suite, 'testcase', name=name, classname='contracts-fast')
            ET.ElementTree(suite).write(d / 'checks.junit.xml'); self.seal(d)

    def compare_fast(self): return COMPARE.compare(self.directories, 'contracts-fast', SHA)

    def test_complete_modules_and_original_commands_match(self):
        self.assertTrue(self.compare()['verified_parity'])

    def test_only_manifest_declared_empty_circle_logs_can_be_restored(self):
        (self.directories['circle'] / 'verify.log').unlink()
        self.assertEqual(len(self.compare()['manifest_declared_empty_logs']), 1)
        (self.directories['rwx'] / 'verify.log').unlink()
        with self.assertRaises(OSError): self.compare()

    def test_corrupt_or_missing_nonempty_originals_fail_validation(self):
        (self.directories['rwx'] / 'discovery.json').write_text('{}')
        with self.assertRaisesRegex(ValueError, 'corrupt'): self.compare()

    def test_stale_sha_does_not_become_matching_success(self):
        p = self.directories['rwx'] / 'settings.json'; value = json.loads(p.read_text()); value['source_sha'] = 'b' * 40
        self.write(p, value); self.seal(p.parent)
        with self.assertRaisesRegex(ValueError, 'source'): self.compare()

    def test_different_module_version_remains_a_parity_error(self):
        d = self.directories['rwx']; p = d / 'discovery.json'; p.write_text(p.read_text().replace('v1.0.0', 'v1.0.1'))
        coverage = json.loads((d / 'coverage.json').read_text()); coverage['modules'] = COMPARE.CHECKS.modules(p.read_text())
        self.write(d / 'coverage.json', coverage); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'graph'): self.compare()

    def test_same_wrong_command_on_both_providers_is_rejected(self):
        for d in self.directories.values():
            p = d / 'verify.stage.json'; row = json.loads(p.read_text()); row['argv'] = ['true']; self.write(p, row); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'command'): self.compare()

    def test_failed_original_cannot_be_hidden_by_passing_junit(self):
        d = self.directories['rwx']; p = d / 'final.json'; row = json.loads(p.read_text()); row['exit_code'] = 17; self.write(p, row)
        with self.assertRaisesRegex(ValueError, 'Failed'): self.compare()

    def test_missing_verification_or_extra_stage_is_rejected(self):
        d = self.directories['rwx']; (d / 'verify.stage.json').unlink(); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'Incomplete'): self.compare()

    def test_complete_check_phases_commands_and_native_identity_match(self):
        self.fast(); self.assertTrue(self.compare_fast()['verified_parity'])

    def test_check_source_configuration_changes_are_not_path_normalization(self):
        self.fast(); d = self.directories['rwx']; self.write(d / 'foundry-config.json', {'optimizer': False, 'root': '/rwx'}); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'Foundry'): self.compare_fast()

    def test_check_retry_history_difference_requires_investigation(self):
        self.fast(); d = self.directories['rwx']; p = d / 'checks.log'
        p.write_text('↻ first 1.0s (will retry with clean build)\n✓ first (retry) 0.3s\n✓ second 2.0s\n✓ All checks passed (2/2)\n')
        self.write(d / 'coverage.json', COMPARE.CHECKS.check_verdicts(p.read_text(), json.loads((d / 'selection.json').read_text())))
        self.seal(d)
        with self.assertRaisesRegex(ValueError, 'retry'): self.compare_fast()

    def test_descriptive_git_refs_may_differ_only_with_identical_complete_pointers(self):
        self.fast(); d = self.directories['rwx']; p = d / 'submodules.txt'
        p.write_text(p.read_text().replace('original-description', 'other-tag-description')); self.seal(d)
        result = self.compare_fast(); self.assertTrue(result['verified_parity'])
        self.assertEqual(result['submodule_revisions'], [{'path': 'contract-library', 'sha': 'd' * 40}])
        for text in ['+' + 'd' * 40 + ' contract-library\n', ' ' + 'e' * 40 + ' contract-library\n',
                     ' ' + 'd' * 40 + ' different-path\n', ' ' + 'd' * 40 + ' contract-library\n ' + 'd' * 40 + ' contract-library\n']:
            p.write_text(text); self.seal(d)
            with self.subTest(original=text), self.assertRaises(ValueError): self.compare_fast()

    def test_target_revision_drift_remains_a_parity_error(self):
        self.fast(); d = self.directories['rwx']; p = d / 'settings.json'
        row = json.loads(p.read_text()); row['target_sha'] = 'f' * 40; self.write(p, row); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'target_sha'): self.compare_fast()

    def test_reused_native_verdict_or_different_submodule_is_rejected(self):
        self.fast(); d = self.directories['rwx']; p = d / 'settings.json'; settings = json.loads(p.read_text()); settings['rwx_run_id'] = None
        self.write(p, settings); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'fresh'): self.compare_fast()
        settings['rwx_run_id'] = 'c' * 32; self.write(p, settings)
        (d / 'submodules.txt').write_text(' stale-revision contract-library\n'); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'submodule'): self.compare_fast()


if __name__ == '__main__': unittest.main()
