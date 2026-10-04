#!/usr/bin/env python3
"""Reject incomplete module/check coverage and exercise the original Go runner."""
import importlib.util
import json
import os
from pathlib import Path
import shlex
import shutil
import subprocess
import sys
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('checks', Path(__file__).with_name('pr-checks.py'))
CHECKS = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(CHECKS)


class CheckTests(unittest.TestCase):
    def setUp(self):
        self.config = {'phases': [{'name': 'full', 'build': 'build source', 'checks': [
            {'name': 'first', 'command': 'original first'},
            {'name': 'new-check', 'command': 'original second', 'depends': ['first'], 'retry-clean': True}]}]}

    def test_new_checks_are_included_and_missing_dependency_or_duplicate_is_rejected(self):
        self.assertEqual(CHECKS.checks(self.config), ['first', 'new-check'])
        for change in ('duplicate', 'dependency', 'empty'):
            config = json.loads(json.dumps(self.config))
            if change == 'duplicate': config['phases'][0]['checks'][1]['name'] = 'first'
            elif change == 'dependency': config['phases'][0]['checks'][1]['depends'] = ['missing']
            else: config['phases'] = []
            with self.subTest(change=change), self.assertRaises(ValueError): CHECKS.checks(config)

    def test_spinner_repeats_do_not_erase_original_failure_or_retry(self):
        log = '\x1b[32m✓\x1b[0m first 1.0s\n✓ first 1.0s\n↻ new-check 2.0s (will retry with clean build)\n'
        log += '✓ new-check (retry) 1.1s\n✓ All checks passed (2/2)\n'
        result = CHECKS.check_verdicts(log, self.config)
        self.assertEqual(len(result['checks'][0]['attempts']), 1)
        self.assertEqual([r['outcome'] for r in result['checks'][1]['attempts']], ['fail', 'pass'])
        self.assertEqual(result['retries'], ['new-check'])

    def test_missing_extra_failed_or_skipped_check_cannot_be_success(self):
        for log in ('✓ first 1.0s\n✓ All checks passed (1/2)\n',
                    '✓ first 1.0s\n✗ new-check 2.0s\n✓ All checks passed (2/2)\n',
                    '✓ first 1.0s\n✓ new-check 2.0s\n✓ unknown 2.0s\n✓ All checks passed (2/2)\n',
                    '✓ first 1.0s\n✓ new-check 2.0s\n'):
            with self.subTest(log=log), self.assertRaises(ValueError): CHECKS.check_verdicts(log, self.config)

    def test_dependency_skip_and_unblocked_execution_are_both_retained(self):
        log = '↻ first 1.0s (will retry with clean build)\n✗   new-check (skipped)\n'
        log += '✓ first (retry) 0.5s\n✓ new-check (unblocked) 0.4s\n✓ All checks passed (2/2)\n'
        report = CHECKS.check_verdicts(log, self.config)
        self.assertEqual([r['outcome'] for r in report['checks'][1]['attempts']], ['skip', 'pass'])
        self.assertEqual(report['checks'][1]['attempts'][0]['reason'], 'dependency failed')

    def test_unlabelled_duplicate_or_unexplained_retry_is_rejected(self):
        for first in ('✓ first 1.0s\n✓ first 2.0s', '✓ first (retry) 1.0s'):
            with self.subTest(first=first), self.assertRaises(ValueError):
                CHECKS.check_verdicts(first + '\n✓ new-check 1.0s\n✓ All checks passed (2/2)\n', self.config)

    def test_complete_modules_preserve_versions_replacements_and_content_hashes(self):
        rows = [{'Path': 'main', 'Main': True, 'Dir': '/native/repo'},
                {'Path': 'dependency', 'Version': 'v1.0.0', 'Sum': 'h1:original', 'Replace':
                 {'Path': 'replacement', 'Version': 'v1.2.3', 'Sum': 'h1:replacement', 'Dir': '/native/modules'}}]
        manifest = CHECKS.modules('\n'.join(json.dumps(r) for r in rows))
        self.assertEqual(manifest[0]['Sum'], 'h1:original')
        self.assertEqual(manifest[0]['Replace']['Version'], 'v1.2.3')
        self.assertNotIn('Dir', manifest[0]['Replace'])

    def test_module_discovery_errors_duplicates_or_missing_main_are_rejected(self):
        for text in ('', '{"Path":"dep"}', '{"Path":"main","Main":true}\n{"Path":"main"}',
                     '{"Path":"main","Main":true,"Error":"unavailable"}', '{}', '{invalid'):
            with self.subTest(text=text), self.assertRaises(ValueError): CHECKS.modules(text)

    @unittest.skipUnless(os.environ.get('RWX_LIVE_GO_FIXTURE') == '1', 'Opt-in actual Go check-runner fixture')
    def test_real_check_runner_executes_dependencies_and_retains_clean_retry(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); marker = shlex.quote(str(root / 'marker')); retry = shlex.quote(str(root / 'retry'))
            config = {'phases': [{'name': 'proof', 'parallel': False,
                'build': f'[ ! -f {marker} ] || touch {retry}', 'checks': [
                    {'name': 'first', 'command': f'printf original > {marker}'},
                    {'name': 'new-check', 'command': f'grep -q original {marker} && test -f {retry}',
                     'depends': ['first'], 'retry-clean': True}]}]}
            path = root / 'checks.json'; path.write_text(json.dumps(config))
            # JSON is a YAML subset; the actual in-repo runner reads it unchanged.
            result = subprocess.run(['go', 'run', str(CHECKS.CONTRACTS / 'scripts/check-runner/main.go'),
                '-config', str(path), '-verbose', '-no-cache'], cwd=CHECKS.ROOT, text=True,
                stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            self.assertEqual(result.returncode, 0, result.stdout)
            report = CHECKS.check_verdicts(result.stdout, config)
            self.assertEqual(report['retries'], ['new-check'])
            self.assertEqual((root / 'marker').read_text(), 'original')

    @unittest.skipUnless(os.environ.get('RWX_LIVE_GO_FIXTURE') == '1', 'Opt-in actual Circle module adapter')
    def test_relative_circle_adapter_downloads_verifies_and_discovers_actual_modules(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); scripts = root / 'ops/ci'; scripts.mkdir(parents=True)
            for name in ('pr-checks.py', 'rust-workspace-report.py'): shutil.copyfile(Path(__file__).with_name(name), scripts / name)
            for name, text in [('go.mod', 'module fixture.invalid/ci\n\ngo 1.26.0\n'), ('go.sum', ''), ('mise.toml', ''), ('.gitignore', '.ci/\n')]:
                (root / name).write_text(text)
            for args in (['init', '-q'], ['config', 'user.email', 'fixture@example.invalid'], ['config', 'user.name', 'CI Fixture'], ['add', '.'], ['commit', '-qm', 'module fixture']):
                subprocess.run(['git', *args], cwd=root, check=True)
            sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()
            nested = root / 'packages/contracts-bedrock'; nested.mkdir(parents=True)
            result = subprocess.run([sys.executable, '../../ops/ci/pr-checks.py', 'go-modules'], cwd=nested,
                env={**os.environ, 'CI_COMMIT_SHA': sha, 'CI_CHECK_PROVIDER': 'circleci'}, text=True,
                stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            self.assertEqual(result.returncode, 0, result.stdout)
            report = root / '.ci/pr-checks/go-modules'
            self.assertEqual(json.loads((report / 'coverage.json').read_text())['selected'], 1)
            self.assertIn('all modules verified', (report / 'verify.log').read_text())
            self.assertEqual(json.loads((report / 'settings.json').read_text())['source_sha'], sha)
            self.assertEqual(json.loads((report / 'discovery.stage.json').read_text())['cwd'], str(root))


if __name__ == '__main__': unittest.main()
