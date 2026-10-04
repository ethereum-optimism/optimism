#!/usr/bin/env python3
"""Verify exhaustive Rust evidence, archive binding and failing subprocesses."""
import importlib.util
import json
import os
from pathlib import Path
import signal
import shutil
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

SCRIPTS = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location('rust_workspace_report', SCRIPTS / 'rust-workspace-report.py')
REPORT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(REPORT)


class ReportTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.discovery = {'rust-suites': {'crate::lib': {'testcases': {
            'runs': {'ignored': False, 'filter-match': {'status': 'matches'}},
            'ignored': {'ignored': True, 'filter-match': {'status': 'mismatch', 'reason': 'ignored'}},
            'test_online_rpc': {'ignored': False, 'filter-match': {'status': 'mismatch', 'reason': 'expression'}}}}}}
        self.junit = '<testsuites><testsuite><testcase classname="crate::lib" name="runs"/></testsuite></testsuites>'
        self.unit_files()

    def unit_files(self):
        (self.root / 'unit-list.json').write_text(json.dumps(self.discovery))
        (self.root / 'junit.xml').write_text(self.junit)

    def test_unit_exclusions_are_explicit(self):
        report = REPORT.unit_report(self.root)
        self.assertEqual(report['selected'], 1)
        self.assertEqual(len(report['excluded']), 2)
        self.assertEqual(report['missing'], [])

    def test_missing_verdict_is_not_a_skip(self):
        self.junit = '<testsuites/>'
        self.unit_files()
        with self.assertRaisesRegex(ValueError, 'Incomplete'):
            REPORT.unit_report(self.root)
        self.assertEqual(json.loads((self.root / 'unit-coverage.json').read_text())['missing'], [['crate::lib', 'runs']])

    def test_duplicate_verdict_rejected(self):
        self.junit = self.junit.replace('</testsuite>', '<testcase classname="crate::lib" name="runs"/></testsuite>')
        self.unit_files()
        with self.assertRaisesRegex(ValueError, 'Duplicate'):
            REPORT.unit_report(self.root)

    def test_unrecognized_exclusion_rejected(self):
        self.discovery['rust-suites']['crate::lib']['testcases']['runs']['filter-match'] = {'status': 'mismatch', 'reason': 'expression'}
        self.unit_files()
        with self.assertRaisesRegex(ValueError, 'Unrecognized'):
            REPORT.unit_report(self.root)

    def test_retry_and_failure_evidence_preserved(self):
        self.junit = self.junit.replace('/>', '><flakyFailure message="first failure"/><flakyError message="second attempt"/></testcase>', 1)
        self.unit_files()
        self.assertEqual(REPORT.unit_report(self.root)['cases'][0]['retries'], 2)
        self.junit = '<testsuite><testcase classname="crate::lib" name="runs"><failure message="original"/></testcase></testsuite>'
        self.unit_files()
        self.assertEqual(REPORT.unit_report(self.root)['outcomes'], {'fail': 1})

    def test_doctest_discovery_and_outcomes(self):
        (self.root / 'doctests-list.log').write_text('Doc-tests crate\nsrc/lib.rs - Thing (line 4): test\nsrc/lib.rs - skipped (line 8): test\n2 tests, 0 benchmarks\n')
        (self.root / 'doctests.log').write_text('test src/lib.rs - Thing (line 4) ... ok\ntest src/lib.rs - skipped (line 8) ... ignored\n')
        self.assertEqual(REPORT.libtest_report(self.root, 'doctests')['outcomes'], {'pass': 1, 'skip': 1})
        (self.root / 'doctests.log').write_text('test src/lib.rs - Thing (line 4) ... ok\n')
        with self.assertRaisesRegex(ValueError, 'Incomplete'):
            REPORT.libtest_report(self.root, 'doctests')

    def test_empty_doctest_listing_rejected(self):
        (self.root / 'doctests-list.log').write_text('compiler error')
        (self.root / 'doctests.log').write_text('')
        with self.assertRaisesRegex(ValueError, 'Empty'):
            REPORT.libtest_report(self.root, 'doctests')

    def test_feature_plan_must_execute_exactly_once(self):
        for phase in ('features', 'feature-tests'):
            (self.root / (phase + '-list.log')).write_text('cargo check --manifest-path crate/Cargo.toml --all-features\n')
            (self.root / (phase + '.log')).write_text('info: running `cargo check --manifest-path crate/Cargo.toml --all-features` on crate (1/1)\n')
        REPORT.feature_report(self.root)
        (self.root / 'features.log').write_text('')
        with self.assertRaisesRegex(ValueError, 'Incomplete'):
            REPORT.feature_report(self.root)

    def test_corrupt_and_stale_archive_rejected(self):
        archive = self.root / 'tests.tar.zst'
        archive.write_bytes(b'compiled tests')
        with patch.object(REPORT, 'command', return_value='pinned'), patch.object(REPORT, 'inputs', return_value={'source': 'new'}):
            REPORT.artifact(self.root)
            REPORT.artifact(self.root, True)
            data = json.loads((self.root / 'archive.json').read_text())
            data['source_sha'] = 'stale'
            (self.root / 'archive.json').write_text(json.dumps(data))
            with self.assertRaisesRegex(ValueError, 'mismatch'):
                REPORT.artifact(self.root, True)
            REPORT.artifact(self.root)
            archive.write_bytes(b'corrupt tests')
            with self.assertRaisesRegex(ValueError, 'mismatch'):
                REPORT.artifact(self.root, True)

    def test_original_exit_is_not_hidden_by_report_errors(self):
        (self.root / 'settings.json').write_text('{"job":"tests"}')
        (self.root / 'junit.xml').unlink()
        self.assertEqual(REPORT.finish(self.root, 7), 7)
        self.assertTrue(json.loads((self.root / 'final.json').read_text())['report_errors'])

    def test_success_requires_complete_reports(self):
        (self.root / 'settings.json').write_text('{"job":"doctest"}')
        self.assertTrue(REPORT.finish(self.root, 0))


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


@unittest.skipUnless(os.environ.get('RWX_LIVE_RUST_FIXTURE') == '1', 'opt-in pinned Linux Rust fixture')
class LiveRunnerTests(unittest.TestCase):
    def test_archived_tests_are_fresh_and_failure_reports_survive(self):
        """Exercise the real shell, nextest archive, doctests and cache preparation."""
        repo = SCRIPTS.parent.parent
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            helpers = root / 'ops/ci'
            helpers.mkdir(parents=True)
            for name in ('rust-workspace.sh', 'rust-workspace-report.py', 'rust-target-cache.py', 'op-reth-report.py'):
                shutil.copyfile(SCRIPTS / name, helpers / name)
            for name in ('mise.toml', 'rust/justfile', 'rust/.config/nextest.toml', 'rust/.cargo/config.toml'):
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(repo / name, path)
            (root / 'rust/Cargo.toml').write_text('[workspace]\nmembers=["providers"]\nresolver="2"\n[profile.fast-build]\ninherits="dev"\n')
            (root / 'rust/providers/src').mkdir(parents=True)
            (root / 'rust/providers/Cargo.toml').write_text('[package]\nname="kona-providers-alloy"\nversion="0.0.0"\nedition="2021"\n[features]\ndefault=[]\n')
            (root / 'rust/providers/src/lib.rs').write_text('''/// ```
/// assert_eq!(2 + 2, 4);
/// ```
pub fn example() {}
#[test] fn fresh() { assert!(std::env::var("RWX_FIXTURE_FAIL").is_err(), "intentional original failure"); }
#[test] fn test_filtered_beacon_blobs_deserializes_on_small_stack() {}
#[test] #[ignore] fn ignored() {}
#[test] fn test_online_rpc() {}
''')
            # The production nextest config validates its named E2E binary.
            # Include that target in the fixture rather than weakening the config.
            (root / 'rust/providers/tests').mkdir()
            (root / 'rust/providers/tests/e2e_testsuite.rs').write_text('#[test] fn fixture() {}\n')
            (root / '.gitignore').write_text('.ci/\nrust/target/\n')
            (root / 'superchain-registry').mkdir()
            (root / 'superchain-registry/README').write_text('fixture submodule identity')
            checksums = root / 'rust/op-reth/crates/chainspec/res'
            checksums.mkdir(parents=True)
            (checksums / 'superchain-configs.tar.sha256').write_text('0' * 64 + '\n')
            subprocess.run(['cargo', 'generate-lockfile', '--offline'], cwd=root / 'rust', check=True, capture_output=True)
            for args in (['init', '-q'], ['add', '.'], ['-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', 'commit', '-qm', 'fixture']):
                subprocess.run(['git', *args], cwd=root, check=True, capture_output=True)
            sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()
            env = {**os.environ, 'CI_RUST_PROVIDER': 'rwx', 'CI_COMMIT_SHA': sha}
            def run(job, extra=None):
                result = subprocess.run(['bash', str(helpers / 'rust-workspace.sh'), job], cwd=root,
                                        env={**env, **(extra or {})}, capture_output=True, text=True, timeout=180)
                return result
            build = run('tests-build')
            self.assertEqual(build.returncode, 0, build.stdout + build.stderr)
            env['TEST_ARCHIVE'] = str(root / '.ci/rust-workspace/tests-build')
            first = run('tests')
            self.assertEqual(first.returncode, 0, first.stdout + first.stderr)
            failed = run('tests', {'RWX_FIXTURE_FAIL': '1'})
            self.assertNotEqual(failed.returncode, 0)
            final = json.loads((root / '.ci/rust-workspace/tests/final.json').read_text())
            self.assertEqual(final['exit_code'], failed.returncode)
            self.assertIn('intentional original failure', (root / '.ci/rust-workspace/tests/junit.xml').read_text())
            self.assertEqual(json.loads((root / '.ci/rust-workspace/tests/unit-coverage.json').read_text())['outcomes']['fail'], 1)
            fresh = run('tests')
            self.assertEqual(fresh.returncode, 0, fresh.stdout + fresh.stderr)
            doc = run('doctest')
            self.assertEqual(doc.returncode, 0, doc.stdout + doc.stderr)


if __name__ == '__main__':
    unittest.main()
