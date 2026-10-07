#!/usr/bin/env python3
"""Verify reporting and failure handling against a complete real API capture."""
import copy
import importlib.util
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import tempfile
import time
import unittest

SPEC = importlib.util.spec_from_file_location('flaky', Path(__file__).resolve().parents[2] / 'migration' / 'flaky-report.py')
F = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(F)
GOLD = Path(__file__).parent / 'fixtures/flaky-report/public-insights.json'


class FlakyTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='original flaky ')
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.bin = self.root / 'bin'; self.bin.mkdir()
        self.capture = self.root / 'capture.json'; shutil.copyfile(GOLD, self.capture)
        curl = self.bin / 'curl'
        curl.write_text('''#!/usr/bin/env python3
import json,os,sys
from pathlib import Path
args=sys.argv[1:]
if args==['--version']:
    print('curl original fixture version'); sys.exit(0)
counter=Path(os.environ['RWX_FLAKY_REQUEST']+'.count')
number=int(counter.read_text()) if counter.exists() else 0
counter.write_text(str(number+1))
sequence=os.environ.get('RWX_FLAKY_STATUS_SEQUENCE',os.environ.get('RWX_FLAKY_STATUS','200')).split(',')
status=sequence[min(number,len(sequence)-1)]
body=Path(os.environ['RWX_FLAKY_FIXTURE']).read_bytes() if status=='200' else b'{"message":"original HTTP failure"}'
Path(args[args.index('--output')+1]).write_bytes(body)
Path(os.environ['RWX_FLAKY_REQUEST']).write_text(json.dumps({'authenticated':'-H' in args,
    'branch':args[args.index('--data-urlencode')+1], 'url':args[-1]}))
print(status)
sys.exit(22 if status != '200' else 0)
''')
        curl.chmod(0o755)
        self.env = os.environ | {'PATH': str(self.bin) + os.pathsep + os.environ['PATH'],
            'RWX_FLAKY_FIXTURE': str(self.capture), 'RWX_FLAKY_REQUEST': str(self.root / 'request.json')}

    def generate(self, branch='codex/rwx-ci-pilot', token=None):
        report = self.root / 'report with spaces'
        args = ['bash', str(F.ROOT / F.SCRIPT), '--branch', branch, '--output-dir', str(report)]
        if token: args += ['--token', token]
        result = subprocess.run(args, env=self.env, capture_output=True, text=True)
        return result, report

    def test_complete_original_capture_and_all_real_derived_rows(self):
        self.assertEqual(len(F.validate_api(GOLD)['flaky_tests']), 79)
        result, report = self.generate()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((report / 'flaky_tests.original.json').read_bytes(), GOLD.read_bytes())
        value = F.validate_report(report, 'codex/rwx-ci-pilot')
        self.assertEqual((value['source_rows'], value['acceptance_rows']), (79, 12))
        request = F.read(self.root / 'request.json')
        self.assertEqual(request, {'authenticated': False, 'branch': 'branch=codex/rwx-ci-pilot',
            'url': 'https://circleci.com/api/v2/insights/gh/ethereum-optimism/optimism/flaky-tests'})
        self.assertIn('Top 10 Flaky Tests for branch codex/rwx-ci-pilot', result.stdout)

    def test_authenticated_mode_preserves_the_same_reporting_workload(self):
        result, report = self.generate(token='fixture-token')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(F.read(self.root / 'request.json')['authenticated'])
        self.assertNotIn('fixture-token', result.stdout + result.stderr)
        self.assertEqual(F.validate_report(report, 'codex/rwx-ci-pilot')['acceptance_rows'], 12)

    def test_http_failure_retains_the_original_body_and_status(self):
        self.capture.write_text('{"message":"original HTTP failure"}')
        self.env['RWX_FLAKY_STATUS'] = '403'
        result, report = self.generate()
        self.assertEqual(result.returncode, 22)
        self.assertEqual((report / 'http-status.txt').read_text(), '403\n')
        self.assertEqual((report / 'flaky_tests.original.json').read_bytes(), self.capture.read_bytes())
        self.assertFalse((report / 'flaky_tests.html').exists())

    def test_transient_retry_preserves_the_first_failure_and_final_original(self):
        self.env['RWX_FLAKY_STATUS_SEQUENCE'] = '503,200'
        result, report = self.generate()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((report / 'api-attempt-1.http-status.txt').read_text(), '503\n')
        self.assertEqual(F.read(report / 'api-attempt-1.json'), {'message': 'original HTTP failure'})
        self.assertEqual((report / 'api-attempt-2.json').read_bytes(), GOLD.read_bytes())
        self.assertEqual(F.validate_report(report, 'codex/rwx-ci-pilot')['acceptance_rows'], 12)
        (report / 'api-attempt-1.http-status.txt').unlink()
        with self.assertRaisesRegex(ValueError, 'API attempts'): F.validate_report(report, 'codex/rwx-ci-pilot')

    def test_incomplete_corrupt_duplicate_or_untyped_originals_fail(self):
        original = F.read(GOLD)
        mutations = []
        value = copy.deepcopy(original); value['total_flaky_tests'] += 1; mutations.append(value)
        value = copy.deepcopy(original); value['total_flaky_tests'] = True; mutations.append(value)
        value = copy.deepcopy(original); value['flaky_tests'][0]['times_flaked'] = False; mutations.append(value)
        value = copy.deepcopy(original); value['flaky_tests'][0].pop('classname'); mutations.append(value)
        value = copy.deepcopy(original); value['flaky_tests'].append(value['flaky_tests'][0]); value['total_flaky_tests'] += 1; mutations.append(value)
        for i, value in enumerate(mutations):
            self.capture.write_text(json.dumps(value))
            with self.subTest(mutation=i), self.assertRaises((ValueError, KeyError)):
                F.validate_api(self.capture)
        for text in ('{"flaky_tests":[],"flaky_tests":[],"total_flaky_tests":0}',
                     '{"flaky_tests":[],"total_flaky_tests":NaN}', '{}', 'original non-JSON failure'):
            self.capture.write_text(text)
            with self.subTest(text=text), self.assertRaises(ValueError): F.validate_api(self.capture)
        self.capture.write_text('{}')
        result, report = self.generate()
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((report / 'flaky_tests.csv').exists())

    def test_csv_html_and_filtered_json_omissions_are_rejected(self):
        result, report = self.generate(); self.assertEqual(result.returncode, 0)
        path = report / 'flaky_tests.csv'; original = path.read_text()
        path.write_text('\n'.join(original.splitlines()[:-1]) + '\n')
        with self.assertRaisesRegex(ValueError, 'CSV'): F.validate_report(report, 'codex/rwx-ci-pilot')
        path.write_text(original)
        path = report / 'flaky_tests.html'; original = path.read_text(); path.write_text(original.replace('View Job', 'wrong link text', 1))
        with self.assertRaisesRegex(ValueError, 'HTML'): F.validate_report(report, 'codex/rwx-ci-pilot')
        path.write_text(original)
        path = report / 'flaky_tests.filtered.json'; value = F.read(path); value['flaky_tests'].pop(); F.write(path, value)
        with self.assertRaisesRegex(ValueError, 'report rows'): F.validate_report(report, 'codex/rwx-ci-pilot')

    def test_html_escapes_original_text_without_changing_csv_or_source(self):
        value = F.read(GOLD)
        row = next(row for row in value['flaky_tests'] if row['classname'].startswith(F.PREFIX))
        row['test_name'] = 'Test/<script> & "argument"'
        self.capture.write_text(json.dumps(value))
        branch = 'fixture/<&\"\''
        result, report = self.generate(branch=branch)
        self.assertEqual(result.returncode, 0, result.stderr)
        html = (report / 'flaky_tests.html').read_text()
        self.assertNotIn('<script>', html)
        self.assertIn('&lt;script&gt;', html)
        self.assertEqual(F.validate_report(report, branch)['source_rows'], 79)

    def test_legitimate_empty_api_still_generates_complete_reports(self):
        self.capture.write_text('{"flaky_tests":[],"total_flaky_tests":0}')
        result, report = self.generate()
        self.assertEqual(result.returncode, 0, result.stderr)
        value = F.validate_report(report, 'codex/rwx-ci-pilot')
        self.assertEqual((value['source_rows'], value['acceptance_rows']), (0, 0))

    def checkout(self):
        root = self.root / 'checkout'; root.mkdir()
        for name in F.INPUTS:
            path = root / name; path.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(F.ROOT / name, path)
        subprocess.run(['git', 'init', '-q', str(root)], check=True)
        subprocess.run(['git', '-C', str(root), 'add', '.'], check=True)
        subprocess.run(['git', '-C', str(root), '-c', 'user.name=Report fixture', '-c', 'user.email=report-fixture@example.invalid',
                        '-c', 'core.hooksPath=/dev/null', 'commit', '-qm', 'Fixture source'], check=True)
        sha = subprocess.check_output(['git', '-C', str(root), 'rev-parse', 'HEAD'], text=True).strip()
        self.env |= {'CI_BRANCH': 'codex/rwx-ci-pilot', 'CI_COMMIT_SHA': sha, 'CI_CHECK_PROVIDER': 'rwx'}
        return root

    def execute(self, root):
        return subprocess.run(['python3', str(root / 'ops/ci/migration/flaky-report.py'), 'run', '.ci/flaky-report/run'],
                              cwd=root, env=self.env, capture_output=True, text=True)

    def test_real_wrapper_success_and_http_failure_keep_complete_originals(self):
        root = self.checkout(); result = self.execute(root)
        self.assertEqual(result.returncode, 0, result.stderr)
        directory = root / '.ci/flaky-report/run'
        self.assertEqual(F.read(directory / 'final.json')['sha256'], F.files(directory))
        self.assertEqual(F.read(directory / 'final.json')['reports'], 1)
        self.assertEqual(F.read(directory / 'settings.json')['provider'], 'rwx')
        shutil.rmtree(directory); self.env['RWX_FLAKY_STATUS'] = '403'
        result = self.execute(root); self.assertNotEqual(result.returncode, 0)
        final = F.read(directory / 'final.json')
        self.assertEqual((final['state'], final['exit_code'], final['reports']), ('failed', 22, 0))
        self.assertEqual(final['sha256'], F.files(directory))
        self.assertEqual((directory / 'reports/http-status.txt').read_text(), '403\n')
        self.assertTrue((directory / 'reports/flaky_tests.original.json').is_file())

    def test_cancellation_keeps_original_process_failure_and_does_not_report_success(self):
        root = self.checkout()
        path = self.bin / 'curl'
        path.write_text('''#!/usr/bin/env python3
import os,sys,time
from pathlib import Path
if sys.argv[1:]==['--version']:
    print('curl original fixture version'); sys.exit(0)
Path(os.environ['RWX_FLAKY_REQUEST']).write_text('entered original request')
while True: time.sleep(1)
'''); path.chmod(0o755)
        with subprocess.Popen(['python3', str(root / 'ops/ci/migration/flaky-report.py'), 'run', '.ci/flaky-report/run'],
             cwd=root, env=self.env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True) as process:
            deadline = time.monotonic() + 10
            while not (self.root / 'request.json').exists() and time.monotonic() < deadline:
                self.assertIsNone(process.poll()); time.sleep(0.02)
            self.assertTrue((self.root / 'request.json').exists(), 'Original child request never started')
            process.send_signal(signal.SIGTERM)
            stdout, stderr = process.communicate(timeout=10)
            self.assertNotEqual(process.returncode, 0, stdout + stderr)
        directory = root / '.ci/flaky-report/run'
        execution, final = F.read(directory / 'execution.json'), F.read(directory / 'final.json')
        self.assertEqual(execution['signals'], [signal.SIGTERM])
        self.assertNotEqual(execution['exit_code'], 0)
        self.assertEqual((final['state'], final['reports'], final['tests']), ('failed', 0, 0))
        self.assertEqual(final['sha256'], F.files(directory))
        self.assertFalse((directory / 'reports/flaky_tests.html').exists())


if __name__ == '__main__':
    unittest.main()
