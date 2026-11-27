"""Run the Cannon job shell against stub tools and check its exit status and report."""
import hashlib
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
import xml.etree.ElementTree as ET

REPO = Path(__file__).resolve().parents[3]
SPEC = importlib.util.spec_from_file_location('cannon_report', Path(__file__).resolve().parents[1] / 'runtime' / 'rust-cannon-report.py')
REPORT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(REPORT)
CONFIG = json.loads((REPO / REPORT.CONFIG).read_text())
KEYS = ('block_number', 'l2_claim', 'l2_output_root', 'l2_head', 'l1_head', 'l2_chain_id', 'filename', 'release_url')


class ReportTests(unittest.TestCase):
    def test_finish_keeps_the_exit_status_and_reports_one_guest_case(self):
        with tempfile.TemporaryDirectory() as temp:
            report = Path(temp)
            REPORT.write(report / 'settings.json', {'job': 'offline', 'witness': CONFIG})
            REPORT.write(report / 'offline.stage.json', {'exit_code': 7})
            for status in (7, 0):
                self.assertEqual(REPORT.finish(report, status), status)
                final = REPORT.read(report / 'final.json')
                self.assertEqual((final['exit_code'], final['stages']['offline']['exit_code']), (status, 7))
                cases = ET.parse(report / 'checks.junit.xml').findall('.//testcase')
                self.assertEqual([case.get('name') for case in cases],
                                 ['kona-client/' + CONFIG['l2_chain_id'] + '/' + CONFIG['block_number']])
                self.assertEqual(cases[0].find('failure') is not None, bool(status))


class RunnerTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory(); self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        # The two ignore files keep the witness archive and the recipe's final
        # Cannon state out of the Rust source check, as in the repository.
        for name in ('ops/ci/runtime/rust-cannon.sh', 'ops/ci/runtime/rust-cannon-report.py',
                     'ops/ci/runtime/rust-workspace-report.py', 'ops/ci/runtime/ci-report.py',
                     'ops/ci/runtime/rust-target-cache.py', str(REPORT.DOCKERFILE),
                     'rust/kona/bin/client/scripts/fetch-witness-tar.sh',
                     'rust/kona/bin/client/testdata/.gitignore', 'rust/kona/.gitignore'):
            target = self.root / name; target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(REPO / name, target)
        self.config = CONFIG | {'sha256': hashlib.sha256(b'fixture witness').hexdigest()}
        REPORT.write(self.root / REPORT.CONFIG, self.config)
        stubs = self.root / 'stubs'; stubs.mkdir()
        stub = stubs / 'stub'
        stub.write_text('''#!/usr/bin/env python3
import gzip,json,os,pathlib,sys,time
root=pathlib.Path(os.environ['FIXTURE_ROOT']); tool=pathlib.Path(sys.argv[0]).name; args=sys.argv[1:]
if tool in ('rustc','cargo','go'): print(tool+' pinned'); sys.exit(0)
if tool=='sccache':
    if '--show-stats' in args: print('{}')
    sys.exit(0)
if tool=='curl':
    path=pathlib.Path(args[args.index('-o')+1]); path.parent.mkdir(parents=True,exist_ok=True); path.write_bytes(b'fixture witness')
    sys.exit(int(os.environ.get('CURL_EXIT','0')))
if tool=='docker':
    (root/'docker.json').write_text(json.dumps(args)); sys.exit(0)
if tool=='just':
    (root/'invocations.json').write_text(json.dumps(args))
    if args[0]=='run-client-cannon-offline':
        if os.environ.get('WAIT_FOR_CANCEL'):
            (root/'running').touch(); print('guest running',flush=True); time.sleep(60)
        status=int(os.environ.get('OFFLINE_EXIT','0'))
        if status: print('original compile or runtime failure',flush=True); sys.exit(status)
        (root/'rust/kona/out.bin.gz').write_bytes(gzip.compress(b'final state'))
    sys.exit(0)
''')
        stub.chmod(0o755)
        for name in ('just', 'docker', 'rustc', 'cargo', 'go', 'curl', 'sccache'): (stubs / name).symlink_to(stub)
        subprocess.run(['git', 'init', '-q'], cwd=self.root, check=True)
        subprocess.run(['git', 'add', '.'], cwd=self.root, check=True)
        subprocess.run(['git', '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.test', 'commit', '-qm', 'fixture'], cwd=self.root, check=True)
        self.env = dict(os.environ, PATH=str(stubs) + ':' + os.environ['PATH'],
                        RWX_RUN_ID='a' * 32, RWX_TASK_ATTEMPT_NUMBER='1', FIXTURE_ROOT=str(self.root))

    def run_job(self, job, **env):
        return subprocess.run(['bash', 'ops/ci/runtime/rust-cannon.sh', job], cwd=self.root,
                              env=self.env | env, capture_output=True, text=True, timeout=15)

    def final(self, job):
        return REPORT.read(self.root / '.ci/rust-workspace' / ('cannon-' + job) / 'final.json')

    def test_offline_passes_the_pinned_witness_to_the_recipe(self):
        run = self.run_job('offline')
        self.assertEqual(run.returncode, 0, run.stdout + run.stderr)
        args = json.loads((self.root / 'invocations.json').read_text())
        self.assertEqual(args, ['run-client-cannon-offline'] + [CONFIG[key] for key in KEYS])
        self.assertEqual(self.final('offline')['exit_code'], 0)
        # Git ignores the recipe's final Cannon state, so the target cache commit accepts the run.
        self.assertTrue((self.root / 'rust/kona/out.bin.gz').exists())
        self.assertTrue((self.root / 'rust/target/.rwx-source-fingerprint.json').exists())

    def test_first_failure_not_retried_and_report_collected(self):
        run = self.run_job('offline', OFFLINE_EXIT='7')
        self.assertEqual(run.returncode, 7, run.stdout + run.stderr)
        final = self.final('offline')
        self.assertEqual(final['exit_code'], 7)
        self.assertEqual(final['stages']['offline']['exit_code'], 7)
        report = self.root / '.ci/rust-workspace/cannon-offline'
        self.assertIn('original compile or runtime failure', (report / 'offline.log').read_text())
        self.assertIsNotNone(ET.parse(report / 'checks.junit.xml').find('.//failure'))

    def test_witness_fetch_failure_and_pin_mismatch_fail_the_witness_job(self):
        run = self.run_job('witness')
        self.assertEqual(run.returncode, 0, run.stdout + run.stderr)
        self.assertEqual(self.final('witness')['stages']['witness']['argv'][-2:], [CONFIG['filename'], CONFIG['release_url']])
        run = self.run_job('witness', CURL_EXIT='22')
        self.assertEqual(run.returncode, 22)
        self.assertEqual({name: stage['exit_code'] for name, stage in self.final('witness')['stages'].items()}, {'witness': 22})
        REPORT.write(self.root / REPORT.CONFIG, self.config | {'sha256': '0' * 64})
        run = self.run_job('witness')
        self.assertNotEqual(run.returncode, 0)
        self.assertIn('checksum mismatch', run.stderr)
        self.assertEqual(self.final('witness')['stages']['witness']['exit_code'], 0)

    def test_env_pulls_the_base_image_before_the_recipe(self):
        run = self.run_job('env')
        self.assertEqual(run.returncode, 0, run.stdout + run.stderr)
        base = subprocess.check_output(['python3', 'ops/ci/runtime/rust-cannon-report.py', 'base'], cwd=self.root, text=True).strip()
        self.assertEqual(json.loads((self.root / 'docker.json').read_text()), ['pull', base])
        self.assertEqual(json.loads((self.root / 'invocations.json').read_text()), ['build-kona-env'])
        self.assertEqual(list(self.final('env')['stages']), ['base-pull', 'env'])

    @unittest.skipUnless(os.name == 'posix', 'POSIX cancellation')
    def test_cancellation_preserves_signal_and_original_stream(self):
        process = subprocess.Popen(['bash', 'ops/ci/runtime/rust-cannon.sh', 'offline'], cwd=self.root,
            env=self.env | {'WAIT_FOR_CANCEL': '1'}, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            text=True, start_new_session=True)
        try:
            deadline = time.monotonic() + 10
            while not (self.root / 'running').exists() and process.poll() is None and time.monotonic() < deadline:
                time.sleep(.05)
            self.assertTrue((self.root / 'running').exists())
            os.killpg(process.pid, signal.SIGTERM)
            process.communicate(timeout=10)
            final = self.final('offline')
            self.assertEqual(final['exit_code'], 143)
            self.assertEqual(final['stages']['offline']['exit_code'], -signal.SIGTERM)
            self.assertIn('guest running', (self.root / '.ci/rust-workspace/cannon-offline/offline.log').read_text())
        finally:
            if process.poll() is None:
                os.killpg(process.pid, signal.SIGKILL); process.communicate()


if __name__ == '__main__': unittest.main()
