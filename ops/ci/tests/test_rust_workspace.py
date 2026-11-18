#!/usr/bin/env python3
"""Run the Rust workspace job shell and its stage recorder against stub tools."""
import importlib.util
import json
import os
from pathlib import Path
import shlex
import signal
import shutil
import subprocess
import sys
import tempfile
import time
import unittest

SCRIPTS = Path(__file__).resolve().parents[1] / 'runtime'
REPO = Path(__file__).resolve().parents[3]
SPEC = importlib.util.spec_from_file_location('rust_workspace_report', SCRIPTS / 'rust-workspace-report.py')
REPORT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(REPORT)


class ReportTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)

    def test_finish_keeps_the_original_exit_and_stage_records(self):
        REPORT.write(self.root / 'settings.json', {'job': 'tests-compiler'})
        REPORT.write(self.root / 'doctests.stage.json', {'exit_code': 7})
        self.assertEqual(REPORT.finish(self.root, 7), 7)
        final = json.loads((self.root / 'final.json').read_text())
        self.assertEqual((final['exit_code'], final['stages']['doctests']['exit_code']), (7, 7))
        # Only a job whose command writes no test report gets a JUnit case.
        self.assertFalse((self.root / 'checks.junit.xml').exists())

    def test_check_job_junit_follows_the_original_script_exit(self):
        REPORT.write(self.root / 'settings.json', {'job': 'interop'})
        self.assertEqual(REPORT.finish(self.root, 0), 0)
        self.assertNotIn('failure', (self.root / 'checks.junit.xml').read_text())
        self.assertEqual(REPORT.finish(self.root, 1), 1)
        self.assertIn('failure', (self.root / 'checks.junit.xml').read_text())


@unittest.skipUnless(shutil.which('just'), 'requires the pinned just tool')
class JobCommandTests(unittest.TestCase):
    """Run the shared shell; Just dry-runs the unchanged recipes, so arity errors fail."""

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        root = self.root = Path(self.temp.name)
        for name in ('ops/ci/runtime/rust-workspace.sh', 'ops/ci/runtime/rust-workspace-report.py',
                     'ops/ci/runtime/ci-report.py', 'ops/ci/runtime/op-reth-report.py',
                     'ops/ci/runtime/rust-target-cache.py', 'mise.toml', 'rust/justfile'):
            (root / name).parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(REPO / name, root / name)
        (root / 'rust/Cargo.lock').write_text('version = 4\n')
        self.snapshot = root / 'rust/kona/crates/protocol/registry/etc/chainList.json'
        self.snapshot.parent.mkdir(parents=True)
        self.snapshot.write_text('[]\n')
        checksum = root / 'rust/op-reth/crates/chainspec/res/superchain-configs.tar.sha256'
        checksum.parent.mkdir(parents=True)
        checksum.write_text('0' * 64 + '  superchain-configs.tar\n')
        for args in (['init', '-q'], ['add', '.'], ['-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', 'commit', '-qm', 'fixture']):
            subprocess.run(['git', *args], cwd=root, check=True, capture_output=True)
        shims = root / 'bin'
        shims.mkdir()
        # The Cargo stub stands for nextest only: it writes a JUnit report and exits as the test asks.
        nextest = ('if [[ "$1 $2" == "nextest run" ]]; then mkdir -p "$CARGO_TARGET_DIR/nextest/default"; '
                   'echo "<testsuites/>" > "$CARGO_TARGET_DIR/nextest/default/junit.xml"; exit "${NEXTEST_EXIT:-0}"; fi')
        for name, body in (('just', 'exec ' + shlex.quote(shutil.which('just')) + ' --dry-run "$@"'),
                           ('cargo', nextest), ('rustc', 'true'), ('sccache', 'true'), ('tar', 'true')):
            (shims / name).write_text('#!/usr/bin/env bash\n' + body + '\n')
            (shims / name).chmod(0o755)
        self.env = os.environ | {'PATH': str(shims) + os.pathsep + os.environ['PATH'],
                                 'RWX_RUN_ID': 'a' * 32, 'RWX_TASK_ATTEMPT_NUMBER': '1'}

    def run_job(self, job, **env):
        return subprocess.run(['bash', 'ops/ci/runtime/rust-workspace.sh', job], cwd=self.root,
                              env=self.env | env, capture_output=True, text=True)

    def report(self, name):
        return self.root / '.ci/rust-workspace' / name

    def test_partitions_pass_original_recipe_arguments_with_one_lockfile_seed(self):
        seed = REPORT.digest(self.root / 'rust/Cargo.lock')
        for index, total, partition in (('2', '10', '3/10'), ('0', '1', '')):
            with self.subTest(total=total):
                result = self.run_job('features', CI_RUST_PARTITION_INDEX=index, CI_RUST_PARTITION_TOTAL=total)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                report = self.report('features-' + index)
                for stage, recipe in (('features', 'hack'), ('feature-tests', 'hack-tests-default')):
                    record = json.loads((report / (stage + '.stage.json')).read_text())
                    self.assertEqual((record['argv'], record['cwd']), (['just', recipe, partition, 'true', seed], 'rust'))
                    self.assertEqual('--partition ' + partition in (report / (stage + '.log')).read_text(), bool(partition))

    def test_unit_job_exports_the_nextest_junit_and_keeps_the_nextest_exit(self):
        archive = self.root / 'archive'
        archive.mkdir()
        for status in (0, 7):
            with self.subTest(status=status):
                result = self.run_job('tests-unit', TEST_ARCHIVE=str(archive), NEXTEST_EXIT=str(status))
                self.assertEqual(result.returncode, status, result.stdout + result.stderr)
                report = self.report('tests-unit')
                self.assertEqual((report / 'junit.xml').read_text(), '<testsuites/>\n')
                record = json.loads((report / 'unit.stage.json').read_text())
                self.assertEqual((record['exit_code'], record['argv'][:3], record['argv'][-2:]),
                                 (status, ['cargo', 'nextest', 'run'], ['-E', '!test(test_online)']))
                self.assertEqual(json.loads((report / 'final.json').read_text())['exit_code'], status)

    def test_compiler_tests_run_the_original_recipes(self):
        result = self.run_job('tests-compiler')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        final = json.loads((self.report('tests-compiler') / 'final.json').read_text())
        self.assertEqual(final['exit_code'], 0)
        self.assertEqual({name: record['argv'] for name, record in final['stages'].items()},
                         {'beacon': ['just', 'test-beacon-blob-stack'], 'doctests': ['just', 'test-docs']})

    def test_registry_verdict_is_the_diff_of_the_regenerated_snapshots(self):
        result = self.run_job('registry')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        report = self.report('registry')
        final = json.loads((report / 'final.json').read_text())
        self.assertEqual(list(final['stages']), ['registry', 'registry-clean', 'registry-diff'])
        self.assertEqual(final['stages']['registry']['cwd'], 'rust/kona')
        self.assertNotIn('failure', (report / 'checks.junit.xml').read_text())
        # The stub Cargo regenerates nothing. A snapshot that differs from the commit stands for stale output.
        self.snapshot.write_text('[1]\n')
        result = self.run_job('registry')
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        final = json.loads((report / 'final.json').read_text())
        self.assertEqual((final['exit_code'], final['stages']['registry-diff']['exit_code']), (1, 1))
        self.assertIn('failure', (report / 'checks.junit.xml').read_text())


class StageTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / 'rust').mkdir()
        self.report = self.root / 'reports'
        self.report.mkdir()

    def run_stage(self, code):
        return subprocess.run([sys.executable, str(SCRIPTS / 'rust-workspace-report.py'),
                               'stage', str(self.report), 'failure', sys.executable, '-c', code],
                              cwd=self.root, capture_output=True)

    def test_real_failing_process_retains_original_log(self):
        result = self.run_stage('print("original failure", flush=True); raise SystemExit(7)')
        self.assertEqual(result.returncode, 7)
        self.assertIn('original failure', (self.report / 'failure.log').read_text())
        self.assertEqual(json.loads((self.report / 'failure.stage.json').read_text())['exit_code'], 7)

    def test_explicit_working_directory_is_used_and_recorded(self):
        result = subprocess.run([sys.executable, str(SCRIPTS / 'rust-workspace-report.py'),
            'stage-at', str(self.report), '.', 'root', sys.executable, '-c',
            'import os; print(os.getcwd())'], cwd=self.root, capture_output=True)
        self.assertEqual(result.returncode, 0)
        self.assertEqual((self.report / 'root.log').read_text().strip(), str(self.root.resolve()))
        self.assertEqual(json.loads((self.report / 'root.stage.json').read_text())['cwd'], '.')

    def test_cancellation_reaches_child_group(self):
        process = subprocess.Popen([sys.executable, str(SCRIPTS / 'rust-workspace-report.py'),
                                    'stage', str(self.report), 'cancel', sys.executable, '-c',
                                    'import time; print("running", flush=True); time.sleep(60)'],
                                   cwd=self.root, stdout=subprocess.DEVNULL)
        try:
            for _ in range(100):
                path = self.report / 'cancel.log'
                if path.exists() and 'running' in path.read_text():
                    break
                time.sleep(.02)
            else:
                self.fail('Child did not start')
            process.send_signal(signal.SIGTERM)
            self.assertEqual(process.wait(timeout=5), 143)
            self.assertEqual(json.loads((self.report / 'cancel.stage.json').read_text())['exit_code'], -signal.SIGTERM)
        finally:
            if process.poll() is None:
                process.kill()
                process.wait()


class ConfigurationTests(unittest.TestCase):
    @unittest.skipUnless(shutil.which('yq'), 'requires the pinned yq tool')
    def test_feature_tasks_cover_every_partition_once(self):
        """Each hand-written task pair must check its own cargo-hack partition; a copied index leaves one unchecked."""
        tasks = {task['key']: task for task in json.loads(subprocess.check_output(
            ['yq', '-o=json', '.tasks', str(REPO / '.rwx/rust.yml')], text=True))}
        for index in range(10):
            for key in (f'features-build-{index}', f'features-{index}'):
                env = tasks[key]['env']
                self.assertEqual((env['CI_RUST_PARTITION_INDEX'], env['CI_RUST_PARTITION_TOTAL']), (str(index), '10'), key)


if __name__ == '__main__':
    unittest.main()
