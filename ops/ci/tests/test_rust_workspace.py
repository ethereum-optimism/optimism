#!/usr/bin/env python3
"""Verify exhaustive Rust evidence, archive binding and failing subprocesses."""
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
from unittest.mock import patch

SCRIPTS = Path(__file__).resolve().parents[1] / 'runtime'
SPEC = importlib.util.spec_from_file_location('rust_workspace_report', Path(__file__).resolve().parents[1] / 'runtime' / 'rust-workspace-report.py')
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

    def test_no_std_requires_every_host_target_completion(self):
        source = 'no_std_packages=(\n  proof\n)\nhost_no_std_packages=(\n  reth-optimism-evm\n)\n'
        base = 'Checking no_std build for: proof\nSuccessfully checked no_std build for: proof\n'
        host = 'Checking host-target no_std build for: reth-optimism-evm\nSuccessfully checked host-target no_std build for: reth-optimism-evm\n'
        with patch.object(Path, 'read_text', autospec=True, side_effect=lambda p: source if p.name == 'justfile' else (self.root / 'source.log').read_bytes().decode()):
            for text in (base, base + host.splitlines()[0] + '\n', base + host + host):
                (self.root / 'source.log').write_text(text)
                with self.assertRaisesRegex(ValueError, 'no_std package coverage'): REPORT.no_std_report(self.root)
            (self.root / 'source.log').write_text(base + host)
            REPORT.no_std_report(self.root)
        report = json.loads((self.root / 'no-std-coverage.json').read_text())
        self.assertEqual(report['host_selected'], ['reth-optimism-evm'])
        self.assertEqual(report['host_completed'], report['host_selected'])

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

    def test_compile_only_doctest_and_dependency_path(self):
        (self.root / 'settings.json').write_text('{"cargo_home":"/provider/cargo"}')
        (self.root / 'doctests-list.log').write_text('/provider/cargo/git/pin/src/lib.rs - Thing (line 4): test\n')
        (self.root / 'doctests.log').write_text('test /provider/cargo/git/pin/src/lib.rs - Thing (line 4) - compile ... ok\n')
        report = REPORT.libtest_report(self.root, 'doctests')
        self.assertEqual(report['missing'], [])
        self.assertEqual(report['cases'][0]['name'], '<cargo>/git/pin/src/lib.rs - Thing (line 4)')
        self.assertTrue(report['cases'][0]['compile_only'])

    def test_libtest_modes_keep_list_identities(self):
        # Pinned rustc 1.99 decorates these verdicts, never their --list identities.
        for phase, name, mode, compile_only in (('beacon', 'tests::panics', 'should panic', False),
                                                ('doctests', 'src/lib.rs - value (line 17)', 'should panic', False),
                                                ('doctests', 'src/lib.rs - value (line 12)', 'compile fail', True),
                                                ('doctests', 'src/lib.rs - value (line 7)', 'compile', True)):
            with self.subTest(mode=mode, name=name):
                (self.root / (phase + '-list.log')).write_text(name + ': test\n\n1 test, 0 benchmarks\n')
                (self.root / (phase + '.log')).write_text(f'running 1 test\ntest {name} - {mode} ... ok\n')
                report = REPORT.libtest_report(self.root, phase)
                self.assertEqual((report['missing'], report['extra']), ([], []))
                self.assertEqual(report['cases'], [dict(suite=phase, name=name, outcome='pass',
                                                        retries=0, compile_only=compile_only)])

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

    def test_archive_rejects_incremental_mode_and_rustflags_changes(self):
        (self.root / 'tests.tar.zst').write_bytes(b'compiled tests')
        with patch.object(REPORT, 'command', return_value='pinned'), patch.object(REPORT, 'inputs', return_value={'source': 'same'}):
            with patch.dict(os.environ, {'CARGO_INCREMENTAL': '1', 'RUSTFLAGS': ''}):
                REPORT.artifact(self.root)
                REPORT.artifact(self.root, True)
                for setting, value in (('CARGO_INCREMENTAL', '0'), ('CARGO_PROFILE_DEV_INCREMENTAL', 'false'),
                                       ('RUSTFLAGS', '-C opt-level=2')):
                    with self.subTest(setting=setting), patch.dict(os.environ, {setting: value}):
                        with self.assertRaisesRegex(ValueError, 'mismatch'):
                            REPORT.artifact(self.root, True)

    def test_native_compiler_handoff_rejects_stale_partial_and_missing_state(self):
        target = self.root / 'target'; target.mkdir()
        state = target / '.rwx-source-fingerprint.json'; state.write_text('complete producer')
        with patch.dict(os.environ, {'CARGO_TARGET_DIR': str(target), 'CARGO_INCREMENTAL': '0'}), \
                patch.object(REPORT, 'command', return_value='pinned'), \
                patch.object(REPORT, 'inputs', return_value={'configuration': 'same'}):
            REPORT.compiler_artifact(self.root)
            manifest = self.root / 'compiler.json'
            REPORT.compiler_artifact(manifest, True)
            pending = target / '.rwx-source-pending.json'; pending.write_text('partial build')
            with self.assertRaisesRegex(ValueError, 'Uncommitted'):
                REPORT.compiler_artifact(manifest, True)
            pending.unlink()
            for env in ({'CARGO_INCREMENTAL': '1'}, {'RUSTFLAGS': '-C opt-level=2'}):
                with patch.dict(os.environ, env), self.assertRaisesRegex(ValueError, 'mismatch'):
                    REPORT.compiler_artifact(manifest, True)
            with patch.object(REPORT, 'command', return_value='new source'), self.assertRaisesRegex(ValueError, 'mismatch'):
                REPORT.compiler_artifact(manifest, True)
            state.write_text('different compiler state')
            with self.assertRaisesRegex(ValueError, 'mismatch'):
                REPORT.compiler_artifact(manifest, True)
            state.unlink()
            with self.assertRaises(FileNotFoundError):
                REPORT.compiler_artifact(manifest, True)

    def test_original_exit_is_not_hidden_by_report_errors(self):
        (self.root / 'settings.json').write_text('{"job":"tests"}')
        (self.root / 'junit.xml').unlink()
        self.assertEqual(REPORT.finish(self.root, 7), 7)
        self.assertTrue(json.loads((self.root / 'final.json').read_text())['report_errors'])

    def test_success_requires_complete_reports(self):
        (self.root / 'settings.json').write_text('{"job":"doctest"}')
        self.assertTrue(REPORT.finish(self.root, 0))


class ExtraReportTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)

    def wasm_files(self, job='wasm-unknown'):
        target, packages, no_defaults = REPORT.WASM[job]
        (self.root / 'settings.json').write_text(json.dumps({'workspace_root': str(self.root),
            'source_sha': 'a' * 40, 'rustc': 'pinned'}))
        (self.root / 'workspace.json').write_text(json.dumps({'packages': [
            {'name': p, 'manifest_path': str(self.root / f'rust/{p}/Cargo.toml')} for p in packages],
            'target_directory': str(self.root / 'target')}))
        plan, actual = [], []
        for i, p in enumerate(packages, 1):
            argv = f'cargo build {"--no-default-features " if no_defaults else ""}--target {target}'
            plan.append(f'{argv} --manifest-path {p}/Cargo.toml')
            actual.append(f'info: running `{argv}` on {p} ({i}/{len(packages)})')
            output = self.root / 'target' / target / 'debug' / f'lib{p.replace("-", "_")}.rlib'
            output.parent.mkdir(parents=True, exist_ok=True)
            output.write_bytes(b'!<arch>\ncompiled')
        (self.root / 'wasm-list.log').write_text('\n'.join(plan) + '\n')
        (self.root / 'wasm.log').write_text('\n'.join(actual) + '\n')

    def test_wasm_both_complete_plans_and_outputs(self):
        for job in REPORT.WASM:
            self.wasm_files(job)
            self.assertEqual(REPORT.wasm_report(self.root, job), REPORT.WASM[job][1])
            artifacts = json.loads((self.root / 'wasm-artifacts.json').read_text())
            self.assertEqual(len(artifacts['artifacts']), len(REPORT.WASM[job][1]))

    def test_wasm_missing_duplicate_or_wrong_target_rejected(self):
        for change in ('missing', 'duplicate', 'target', 'defaults'):
            self.wasm_files()
            path = self.root / 'wasm-list.log'
            lines = path.read_text().splitlines()
            if change == 'missing': lines.pop()
            if change == 'duplicate': lines.append(lines[0])
            if change == 'target': lines[0] = lines[0].replace('wasm32-unknown-unknown', 'wasm32-wasip1')
            if change == 'defaults': lines[0] = lines[0].replace('--no-default-features ', '')
            path.write_text('\n'.join(lines))
            with self.subTest(change=change), self.assertRaises(ValueError):
                REPORT.wasm_report(self.root, 'wasm-unknown')

    def test_wasm_original_failure_and_corrupt_outputs_rejected(self):
        self.wasm_files()
        (self.root / 'wasm.log').write_text('info: running `cargo build` on op-alloy-consensus (1/4)\nerror: original\n')
        with self.assertRaisesRegex(ValueError, 'Incomplete'):
            REPORT.wasm_report(self.root, 'wasm-unknown')
        self.wasm_files()
        next((self.root / 'target').rglob('*.rlib')).write_bytes(b'corrupt')
        with self.assertRaisesRegex(ValueError, 'Corrupt'):
            REPORT.wasm_report(self.root, 'wasm-unknown')
        self.wasm_files()
        next((self.root / 'target/wasm32-unknown-unknown').rglob('*.rlib')).unlink()
        with self.assertRaises(FileNotFoundError):
            REPORT.wasm_report(self.root, 'wasm-unknown')

    def test_registry_all_three_snapshots_are_compared(self):
        for name in ('chainList.json', 'configs.json', 'depsets.json'):
            for phase in ('before', 'after'):
                (self.root / f'registry-{phase}-{name}').write_text('[1]\n')
        REPORT.registry_report(self.root)
        (self.root / 'registry-after-depsets.json').write_text('[2]\n')
        with self.assertRaisesRegex(ValueError, 'differ'):
            REPORT.registry_report(self.root)
        (self.root / 'registry-after-chainList.json').unlink()
        with self.assertRaises(FileNotFoundError):
            REPORT.registry_report(self.root)

    def test_interop_verdict_is_the_original_script_exit(self):
        # The unchanged script compares its dumps itself and retains neither of them.
        REPORT.write(self.root / 'settings.json', {'job': 'interop'})
        for name in ('workspace', 'superchain-go', 'interop'):
            REPORT.write(self.root / (name + '.stage.json'), {'exit_code': 0})
        self.assertEqual(REPORT.finish(self.root, 0), 0)
        self.assertNotIn('failure', (self.root / 'checks.junit.xml').read_text())
        REPORT.write(self.root / 'interop.stage.json', {'exit_code': 1})
        self.assertEqual(REPORT.finish(self.root, 1), 1)
        self.assertIn('failure', (self.root / 'checks.junit.xml').read_text())
        (self.root / 'interop.stage.json').unlink()
        self.assertTrue(REPORT.finish(self.root, 0))


class FeatureCommandTests(unittest.TestCase):
    @unittest.skipUnless(shutil.which('just'), 'requires the pinned just tool')
    def test_partitions_pass_circle_recipe_arguments_with_one_lockfile_seed(self):
        """Run the shared shell; Just dry-runs the unchanged recipes, so arity errors fail."""
        repo = Path(__file__).resolve().parents[3]
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            for name in ('ops/ci/runtime/rust-workspace.sh', 'ops/ci/runtime/rust-workspace-report.py',
                         'ops/ci/runtime/ci-report.py', 'mise.toml', 'rust/justfile',
                         'rust/.config/nextest.toml', 'rust/.cargo/config.toml'):
                (root / name).parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(repo / name, root / name)
            (root / 'rust/Cargo.lock').write_text('version = 4\n')
            (root / 'superchain-registry').mkdir()
            (root / 'superchain-registry/README').write_text('fixture submodule identity')
            for args in (['init', '-q'], ['add', '.'], ['-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', 'commit', '-qm', 'fixture']):
                subprocess.run(['git', *args], cwd=root, check=True, capture_output=True)
            shims = root / 'bin'
            shims.mkdir()
            for name, body in (('just', 'exec ' + shlex.quote(shutil.which('just')) + ' --dry-run "$@"'),
                               ('cargo', 'true'), ('rustc', 'true'), ('sccache', 'true')):
                (shims / name).write_text('#!/usr/bin/env bash\n' + body + '\n')
                (shims / name).chmod(0o755)
            env = os.environ | {'PATH': str(shims) + os.pathsep + os.environ['PATH'], 'CI_RUST_PROVIDER': 'circleci',
                                'CI_COMMIT_SHA': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()}
            seed = REPORT.digest(root / 'rust/Cargo.lock')
            for index, total, partition in (('2', '10', '3/10'), ('0', '1', '')):
                with self.subTest(total=total):
                    result = subprocess.run(['bash', 'ops/ci/runtime/rust-workspace.sh', 'features'], cwd=root, env=env | {
                        'CI_RUST_PARTITION_INDEX': index, 'CI_RUST_PARTITION_TOTAL': total}, capture_output=True, text=True)
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    report = root / '.ci/rust-workspace' / ('features-' + index)
                    for stage, recipe in (('features', 'hack'), ('feature-tests', 'hack-tests-default')):
                        record = json.loads((report / (stage + '.stage.json')).read_text())
                        self.assertEqual((record['argv'], record['cwd']), (['just', recipe, partition, 'true', seed], 'rust'))
                        self.assertEqual('--partition ' + partition in (report / (stage + '.log')).read_text(), bool(partition))


class HostLinkerReportTests(unittest.TestCase):
    def test_rwx_wasm_requires_successful_preparation_and_seals_its_original_log(self):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            REPORT.write(directory / 'settings.json', {'job': 'wasm-unknown', 'provider': 'rwx'})
            for name in ('workspace', 'wasm-target', 'wasm-list', 'wasm'):
                REPORT.write(directory / (name + '.stage.json'), {'exit_code': 0})
            with patch.object(REPORT, 'wasm_report', return_value=['fixture']), \
                    patch.object(REPORT, 'check_junit'):
                self.assertNotEqual(REPORT.finish(directory, 0), 0)
                REPORT.write(directory / 'host-linker.stage.json', {'exit_code': 7})
                self.assertNotEqual(REPORT.finish(directory, 0), 0)
                (directory / 'host-linker.log').write_text('original materialization evidence\n')
                REPORT.write(directory / 'host-linker.stage.json', {'exit_code': 0})
                self.assertEqual(REPORT.finish(directory, 0), 0)
                final = json.loads((directory / 'final.json').read_text())
                self.assertEqual(final['original_sha256']['host-linker.log'],
                                 REPORT.digest(directory / 'host-linker.log'))
                # Circle's command contract does not acquire an RWX-only stage.
                REPORT.write(directory / 'settings.json', {'job': 'wasm-unknown', 'provider': 'circleci'})
                (directory / 'host-linker.stage.json').unlink()
                self.assertEqual(REPORT.finish(directory, 0), 0)


class StageTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / 'rust').mkdir()
        self.report = self.root / 'reports'
        self.report.mkdir()

    def run_stage(self, code):
        return subprocess.run([sys.executable, str(Path(__file__).resolve().parents[1] / 'runtime' / 'rust-workspace-report.py'),
                               'stage', str(self.report), 'failure', sys.executable, '-c', code],
                              cwd=self.root, capture_output=True)

    def test_real_failing_process_retains_original_log(self):
        result = self.run_stage('print("original failure", flush=True); raise SystemExit(7)')
        self.assertEqual(result.returncode, 7)
        self.assertIn('original failure', (self.report / 'failure.log').read_text())
        self.assertEqual(json.loads((self.report / 'failure.stage.json').read_text())['exit_code'], 7)

    def test_explicit_working_directory_is_used_and_recorded(self):
        result = subprocess.run([sys.executable, str(Path(__file__).resolve().parents[1] / 'runtime' / 'rust-workspace-report.py'),
            'stage-at', str(self.report), '.', 'root', sys.executable, '-c',
            'import os; print(os.getcwd())'], cwd=self.root, capture_output=True)
        self.assertEqual(result.returncode, 0)
        self.assertEqual((self.report / 'root.log').read_text().strip(), str(self.root.resolve()))
        self.assertEqual(json.loads((self.report / 'root.stage.json').read_text())['cwd'], '.')

    def test_cancellation_reaches_child_group(self):
        process = subprocess.Popen([sys.executable, str(Path(__file__).resolve().parents[1] / 'runtime' / 'rust-workspace-report.py'),
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
    def test_incremental_trial_is_limited_to_feature_checks_and_tests(self):
        repo = Path(__file__).resolve().parents[3]
        config = json.loads(subprocess.check_output(['yq', '-o=json', '.', str(repo / '.rwx/rust.yml')], text=True))
        selected = {'tests-cache', 'tests-build', 'tests-unit', 'tests-compiler', 'tests'} | {f'features-{i}' for i in range(10)} | {f'features-build-{i}' for i in range(10)}
        actual = {t['key'] for t in config['tasks'] if 'CI_RUST_INCREMENTAL' in t.get('env', {})}
        self.assertEqual(actual, selected)
        tasks = {t['key']: t for t in config['tasks']}
        for key in ('tests-cache', 'tests-build', 'tests-unit', 'tests-compiler'):
            self.assertEqual(tasks[key]['runner'], {'cpus': 16, 'memory': '32gb', 'disk': '150gb'})
        for key in ('tests-build', 'tests-unit', 'tests-compiler'):
            self.assertFalse(tasks[key]['cache'])
            self.assertNotIn('tool-cache', tasks[key])
        self.assertEqual(tasks['tests-build']['use'], ['tests-cache', 'head-source'])
        self.assertEqual(tasks['tests-compiler']['use'], ['tests-build', 'head-source'])
        self.assertEqual(tasks['tests-compiler']['env']['COMPILED_STATE'],
                         '${{ tasks.tests-build.artifacts.compiler-state }}')
        self.assertFalse(tasks['tests-compiler']['outputs']['filesystem'])
        self.assertFalse(tasks['tests-unit']['outputs']['filesystem'])
        self.assertEqual(tasks['tests-cache']['use'], 'rust-dependencies')
        self.assertEqual(tasks['tests-cache']['env']['CI_COMMIT_SHA'], '${{ tasks.compiler-base.values.sha }}')
        self.assertEqual(tasks['compiler-code']['with']['ref'], '${{ tasks.compiler-base.values.sha }}')
        self.assertEqual(tasks['compiler-tools']['use'], ['mise', 'compiler-code'])
        self.assertEqual(tasks['compiler-rust-tools']['use'], ['compiler-code', 'compiler-tools'])
        self.assertEqual(tasks['compiler-source']['use'], ['compiler-rust-tools', 'compiler-code'])
        self.assertEqual(tasks['rust-dependencies']['use'], 'compiler-source')
        self.assertNotIn('tool-cache', tasks['rust-dependencies'])
        self.assertEqual(tasks['rust-dependencies']['env']['COMPILER_CACHE_EPOCH'], '${{ init.cache-epoch }}')
        self.assertEqual(tasks['head-source']['use'], ['rust-dependencies', 'code'])
        self.assertIn('git clean -fdx -- rust', tasks['head-source']['run'])
        for trigger in ('cli', 'cache-rebuild'):
            self.assertEqual(config['on'][trigger]['init']['cargo-incremental'], '0')
        coordinator = json.loads(subprocess.check_output(['yq', '-o=json', '.tasks', str(repo / '.rwx/pr-gates.yml')], text=True))
        self.assertEqual(next(t for t in coordinator if t['key'] == 'native-workspace')['init']['cargo-incremental'], '0')

    @unittest.skipUnless(shutil.which('yq'), 'requires the pinned yq tool')
    def test_fresh_verdicts_keep_compiler_caches_enabled(self):
        definition = Path(__file__).resolve().parents[3] / '.rwx/rust.yml'
        tasks = json.loads(subprocess.check_output(['yq', '-o=json', '.tasks', str(definition)], text=True))
        selected = {'doctest', 'build', 'docs', 'clippy', 'no-std', 'udeps'}
        selected.update({'wasm-unknown', 'wasm-wasi', 'zepter', 'typos', 'registry', 'interop'})

        actual = {t['key']: t for t in tasks if t['key'] in selected}
        self.assertEqual(set(actual), selected)
        for name, task in actual.items():
            with self.subTest(task=name):
                self.assertNotEqual(task.get('cache'), False, 'cache:false disables tool caches')
                self.assertTrue(task.get('tool-cache'))
                for nonce in ('RWX_RUN_ID', 'RWX_TASK_ATTEMPT_NUMBER'):
                    if name.startswith('features-build-'):
                        self.assertNotIn(nonce, task['env'])
                    else:
                        self.assertEqual(task['env'][nonce]['cache-key'], 'included')
                output = task['outputs']['filesystem']['filter']['workspace']
                self.assertIn('rust/target', output)
                self.assertNotIn('.ci/rust-workspace', output)


    @unittest.skipUnless(shutil.which('yq'), 'requires the pinned yq tool')
    def test_features_reuse_compilers_but_keep_current_pr_gate_verdicts(self):
        repo = Path(__file__).resolve().parents[3]
        config = json.loads(subprocess.check_output(['yq', '-o=json', '.', str(repo / '.rwx/rust.yml')], text=True))
        tasks = {t['key']: t for t in config['tasks']}
        policy = json.loads((repo/'ops/ci/runtime/pr-gates.json').read_text())
        gate = next(d for d in policy['gates']['required-rust-ci']['dependencies'] if d['name'] == 'rust-cargo-hack')
        self.assertEqual(gate['tasks'], [f'features-{i}' for i in range(10)])
        for i in range(10):
            producer, verdict = tasks[f'features-build-{i}'], tasks[f'features-{i}']
            self.assertEqual(verdict['use'], [producer['key'], 'head-source'])
            self.assertEqual(producer['use'], 'rust-dependencies')
            self.assertEqual(producer['run'], 'python3 ops/ci/runtime/rust-compiler-baseline.py features')
            self.assertEqual(verdict['run'], 'bash ops/ci/runtime/rust-workspace.sh features')
            self.assertEqual(producer['runner'], verdict['runner'])
            self.assertNotEqual(producer.get('cache'), False)
            self.assertIn('rust/target', producer['outputs']['filesystem']['filter']['workspace'])
            self.assertFalse(verdict['cache'])
            self.assertNotIn('tool-cache', verdict)
            self.assertFalse(verdict['outputs']['filesystem'])
            self.assertIn(producer['key'], config['on']['cache-rebuild']['target'])
            self.assertNotIn(verdict['key'], config['on']['cache-rebuild']['target'])
            for name in ('CI_RUST_INCREMENTAL', 'CI_RUST_PARTITION_INDEX', 'CI_RUST_PARTITION_TOTAL'):
                self.assertEqual(producer['env'][name], verdict['env'][name])
            for nonce in ('RWX_RUN_ID', 'RWX_TASK_ATTEMPT_NUMBER'):
                self.assertNotIn(nonce, producer['env'])
                self.assertEqual(verdict['env'][nonce]['cache-key'], 'included')

    @unittest.skipUnless(shutil.which('yq'), 'requires the pinned yq tool')
    def test_split_workers_and_gate_bindings_preserve_the_full_workload(self):
        repo = Path(__file__).resolve().parents[3]
        config = json.loads(subprocess.check_output(['yq', '-o=json', '.', str(repo / '.rwx/rust.yml')], text=True))
        tasks = {t['key']: t for t in config['tasks']}
        self.assertNotIn('COMPILED_TARGET', tasks['tests-unit']['env'])
        self.assertNotIn('tests-compiler', tasks['tests-unit']['use'])
        self.assertNotIn('tests-unit', tasks['tests-compiler']['use'])
        self.assertEqual(tasks['tests']['after'], '${{ tests-unit.finished && tests-compiler.finished }}')
        self.assertEqual(tasks['tests']['cache'], False)
        self.assertEqual(tasks['tests']['outputs']['filesystem'], False)
        self.assertNotIn('tool-cache', tasks['tests'])
        for key in ('tests-unit', 'tests-compiler', 'tests'):
            self.assertEqual(tasks[key]['if'], "${{ tasks.route.values.run-rust-ci == 'true' }}")
            self.assertNotIn(key, config['on']['cache-rebuild']['target'])
        manifest = json.loads((repo / 'ops/ci/runtime/pr-gates.json').read_text())
        row = next(r for r in manifest['gates']['required-rust-ci']['dependencies'] if r['name'] == 'rust-tests')
        self.assertEqual(set(row['tasks']), {'tests-unit', 'tests-compiler', 'tests'})


class SplitReportTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.settings = {'suite': 'rust-workspace', 'source_sha': 'a' * 40,
                         'provider': 'rwx', 'rwx_run_id': 'b' * 32, 'rwx_task_attempt': '1',
                         'input_sha256': {'lock': 'pinned'}, 'rustc': 'rustc pinned',
                         'cargo': 'cargo pinned', 'nextest': 'nextest pinned',
                         'profile_incremental': {'DEV': 'true'}, 'test_filter': '!test(test_online)'}
        for job in ('tests', 'tests-unit', 'tests-compiler'):
            directory = self.root / job
            directory.mkdir()
            REPORT.write(directory / 'settings.json', self.settings | {'job': job})
            REPORT.write(directory / 'workspace.json', {'packages': ['fixture']})
            REPORT.write(directory / 'workspace.stage.json', {'exit_code': 0})
        unit = self.root / 'tests-unit'
        REPORT.write(unit / 'unit-list.json', {'rust-suites': {'fixture': {'testcases': {
            'fresh': {'filter-match': {'status': 'matches'}, 'ignored': False}}}}})
        (unit / 'unit.log').write_text('original nextest output\n')
        (unit / 'junit.xml').write_text('<testsuite><testcase classname="fixture" name="fresh"/></testsuite>')
        for phase in ('unit', 'unit-list'):
            REPORT.write(unit / (phase + '.stage.json'), {'exit_code': 0})
        compiler = self.root / 'tests-compiler'
        for phase in ('beacon', 'doctests'):
            (compiler / (phase + '-list.log')).write_text('fresh: test\n')
            (compiler / (phase + '.log')).write_text('test fresh ... ok\n')
            for stage in (phase, phase + '-list'):
                REPORT.write(compiler / (stage + '.stage.json'), {'exit_code': 0})
        self.assertEqual(REPORT.finish(unit, 0), 0)
        self.assertEqual(REPORT.finish(compiler, 0), 0)

    def merge(self):
        return REPORT.merge_tests(self.root / 'tests', self.root / 'tests-unit', self.root / 'tests-compiler')

    def seal_unit(self):
        directory = self.root / 'tests-unit'
        final = json.loads((directory / 'final.json').read_text())
        final['original_sha256'] = {p.name: REPORT.digest(p) for p in directory.iterdir()
                                    if p.is_file() and p.name != 'final.json'}
        REPORT.write(directory / 'final.json', final)

    def test_aggregate_preserves_originals_and_all_three_verdicts(self):
        self.assertEqual(self.merge(), 0)
        self.assertEqual(REPORT.finish(self.root / 'tests', 0, merged=True), 0)
        final = json.loads((self.root / 'tests/final.json').read_text())
        self.assertEqual(len(final['reports']), 3)
        for name in ('junit.xml', 'unit.log', 'unit-list.json'):
            self.assertEqual((self.root / 'tests' / name).read_bytes(), (self.root / 'tests-unit' / name).read_bytes())
        parts = json.loads((self.root / 'tests/parts.json').read_text())
        for label, job in (('unit', 'tests-unit'), ('compiler', 'tests-compiler')):
            self.assertEqual(REPORT.digest(self.root / 'tests' / (label + '.final.json')), parts[label]['final_sha256'])
            for original, target in parts[label]['original_files'].items():
                self.assertEqual((self.root / 'tests' / target).read_bytes(), (self.root / job / original).read_bytes())

    def test_aggregate_consumes_validated_child_reports_without_reselecting_tests(self):
        self.assertEqual(self.merge(), 0)
        with patch.object(REPORT, 'unit_report', side_effect=AssertionError('unit worker owns selection')), \
                patch.object(REPORT, 'libtest_report', side_effect=AssertionError('compiler worker owns selection')):
            self.assertEqual(REPORT.finish(self.root / 'tests', 0, merged=True), 0)

    def test_stale_source_settings_toolchain_and_run_are_rejected(self):
        path = self.root / 'tests-unit/settings.json'
        for key in ('source_sha', 'rustc', 'cargo', 'nextest', 'input_sha256',
                    'profile_incremental', 'rwx_run_id', 'test_filter', 'job'):
            with self.subTest(key=key):
                REPORT.write(path, self.settings | {'job': 'tests-unit', key: 'different'})
                self.seal_unit()
                with self.assertRaisesRegex(ValueError, 'mismatch'):
                    self.merge()
        REPORT.write(path, self.settings | {'job': 'tests-unit'})
        self.seal_unit()

    def test_missing_corrupt_and_linked_originals_are_rejected(self):
        path = self.root / 'tests-unit/unit.log'
        original = path.read_bytes()
        for kind in ('missing', 'corrupt', 'linked'):
            with self.subTest(kind=kind):
                path.unlink()
                if kind == 'corrupt':
                    path.write_bytes(b'changed failure evidence')
                elif kind == 'linked':
                    target = self.root / 'linked-log'; target.write_bytes(original)
                    path.symlink_to(target)
                with self.assertRaises((ValueError, OSError)):
                    self.merge()
                path.unlink(missing_ok=True)
                path.write_bytes(original)

    def test_original_failure_and_independent_retry_attempt_are_retained(self):
        unit = self.root / 'tests-unit'
        settings = json.loads((unit / 'settings.json').read_text())
        settings['rwx_task_attempt'] = '2'
        REPORT.write(unit / 'settings.json', settings)
        (unit / 'junit.xml').write_text('<testsuite><testcase classname="fixture" name="fresh"><failure message="original failure"/></testcase></testsuite>')
        REPORT.write(unit / 'unit.stage.json', {'exit_code': 7})
        self.assertEqual(REPORT.finish(unit, 7), 7)
        self.assertEqual(self.merge(), 7)
        self.assertEqual(REPORT.finish(self.root / 'tests', 7, merged=True), 7)
        final = json.loads((self.root / 'tests/final.json').read_text())
        self.assertEqual(final['exit_code'], 7)
        self.assertEqual(final['reports'][0]['outcomes'], {'fail': 1})
        self.assertEqual(json.loads((self.root / 'tests/parts.json').read_text())['unit']['task_attempt'], '2')

    def test_missing_phase_cannot_become_a_success(self):
        compiler = self.root / 'tests-compiler'
        (compiler / 'doctests.log').unlink()
        self.assertNotEqual(REPORT.finish(compiler, 0), 0)
        self.assertNotEqual(self.merge(), 0)
        self.assertNotEqual(REPORT.finish(self.root / 'tests', 0, merged=True), 0)


@unittest.skipUnless(os.environ.get('RWX_LIVE_RUST_FIXTURE') == '1', 'opt-in pinned Linux Rust fixture')
class LiveRunnerTests(unittest.TestCase):
    def test_archived_tests_are_fresh_and_failure_reports_survive(self):
        """Exercise the real shell, nextest archive, doctests and cache preparation."""
        repo = Path(__file__).resolve().parents[3]
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            helpers = root / 'ops/ci/runtime'
            helpers.mkdir(parents=True)
            for name in ('rust-workspace.sh', 'rust-workspace-report.py', 'ci-report.py', 'rust-target-cache.py', 'op-reth-report.py'):
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
/// ```no_run
/// assert_eq!(2 + 2, 4);
/// ```
pub fn compile_only() {}
#[test] fn fresh() { assert!(std::env::var("RWX_FIXTURE_FAIL").is_err(), "intentional original failure"); }
#[test] fn test_filtered_beacon_blobs_deserializes_on_small_stack() {}
#[test] #[ignore] fn ignored() {}
#[test] fn test_online_rpc() {}
''')
            # The production nextest config validates its named E2E binary.
            # Include that target in the fixture rather than weakening the config.
            (root / 'rust/providers/tests').mkdir()
            (root / 'rust/providers/tests/e2e_testsuite.rs').write_text('#[test] fn fixture() {}\n')
            # Exercise the real CLI tests against an archived non-test binary.
            # The unit worker restores it at its compile-time target path.
            shutil.copyfile(repo / 'rust/lokahi/tests/cli.rs', root / 'rust/providers/tests/cli.rs')
            (root / 'rust/providers/src/bin').mkdir()
            (root / 'rust/providers/src/bin/lokahi.rs').write_text('''fn main() {
    match std::env::args().nth(1).as_deref() {
        Some("-V") => println!("lokahi 0.0.0"),
        Some("--version") => println!("Version: fixture\\nCommit SHA: fixture\\nBuild Timestamp: fixture\\nBuild Profile: dev"),
        _ => println!("Hello Lokahi"),
    }
}
''')
            (root / '.gitignore').write_text('.ci/\nrust/target/\n**/superchain-configs.tar\n')
            (root / 'superchain-registry').mkdir()
            (root / 'superchain-registry/README').write_text('fixture submodule identity')
            checksums = root / 'rust/op-reth/crates/chainspec/res'
            checksums.mkdir(parents=True)
            bundle = checksums / 'superchain-configs.tar'
            bundle.write_bytes(b'fixture generated bundle')
            (checksums / 'superchain-configs.tar.sha256').write_text(REPORT.digest(bundle) + '\n')
            bundle.unlink()
            (root / 'rust/providers/build.rs').write_text('''fn main() {
    let path = "../op-reth/crates/chainspec/res/superchain-configs.tar";
    println!("cargo:rerun-if-changed={path}");
    if !std::path::Path::new(path).exists() {
        std::fs::write(path, b"fixture generated bundle").unwrap();
    }
}
''')
            subprocess.run(['cargo', 'generate-lockfile', '--offline'], cwd=root / 'rust', check=True, capture_output=True)
            for args in (['init', '-q'], ['add', '.'], ['-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', 'commit', '-qm', 'fixture']):
                subprocess.run(['git', *args], cwd=root, check=True, capture_output=True)
            sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()
            env = {**os.environ, 'CI_RUST_PROVIDER': 'rwx', 'CI_COMMIT_SHA': sha, 'CI_RUST_INCREMENTAL': '1',
                   'RWX_RUN_ID': 'a' * 32, 'RWX_TASK_ATTEMPT_NUMBER': '1'}
            def run(job, extra=None):
                result = subprocess.run(['bash', str(helpers / 'rust-workspace.sh'), job], cwd=root,
                                        env={**env, **(extra or {})}, capture_output=True, text=True, timeout=180)
                return result
            build = run('tests-build', {'CI_RUST_BASELINE': 'true'})
            self.assertEqual(build.returncode, 0, build.stdout + build.stderr)
            settled = json.loads((root / '.ci/rust-workspace/tests-build/final.json').read_text())
            self.assertEqual(settled['stages']['compiler-settle']['exit_code'], 0)
            settings = json.loads((root / '.ci/rust-workspace/tests-build/settings.json').read_text())
            self.assertIsNone(settings['incremental'])
            self.assertEqual(set(settings['profile_incremental'].values()), {'true'})
            self.assertTrue(any((root / 'rust/target/debug/incremental').iterdir()))
            self.assertTrue(any((root / 'rust/target/fast-build/incremental').iterdir()))
            packed = root / '.ci/rust-cache/target-cache.tar.zst'
            self.assertFalse(packed.exists())
            self.assertTrue((root / '.ci/rust-workspace/tests-build/compiler.json').is_file())
            before = json.loads((root / '.ci/rust-workspace/tests-build/cache-size.json').read_text())
            self.assertGreater(before['target']['bytes'], 0)
            # RWX restores individual files, with the archive excluded from its
            # tool-cache output. An unchanged compiler run uses those files.
            unchanged = run('tests-build')
            self.assertEqual(unchanged.returncode, 0, unchanged.stdout + unchanged.stderr)
            self.assertNotIn('Compiling kona-providers-alloy',
                             (root / '.ci/rust-workspace/tests-build/archive.log').read_text())
            self.assertFalse((root / '.ci/rust-workspace/tests-build/cache-restore.log').exists())
            bundle_stamp = bundle.stat().st_mtime_ns
            os.utime(bundle, ns=(time.time_ns(), time.time_ns()))
            # Recompile from the restored snapshot after a real
            # source edit. The next verdict still uses the complete new archive.
            source = root / 'rust/providers/src/lib.rs'
            source.write_text(source.read_text() + '\npub fn changed_crate() {}\n')
            subprocess.run(['git', '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid',
                            'commit', '-am', 'changed local crate', '-q'], cwd=root, check=True, capture_output=True)
            env['CI_COMMIT_SHA'] = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()
            warm = run('tests-build')
            self.assertEqual(warm.returncode, 0, warm.stdout + warm.stderr)
            freshness = json.loads((root / '.ci/rust-workspace/tests-build/cache-source.json').read_text())
            self.assertTrue(freshness['source_changed'])
            self.assertEqual(freshness['restored_generated_files'], 1)
            self.assertEqual(bundle.stat().st_mtime_ns, bundle_stamp // 10**9 * 10**9)
            self.assertIn('Compiling kona-providers-alloy',
                          (root / '.ci/rust-workspace/tests-build/archive.log').read_text())
            # Simulate the immutable native compiler filesystem independently
            # of the binary-only worker and fresh compiler verdicts.
            snapshot = root / 'producer-target'
            shutil.copytree(root / 'rust/target', snapshot)
            env['COMPILED_STATE'] = str(root / '.ci/rust-workspace/tests-build/compiler.json')
            env['TEST_ARCHIVE'] = str(root / '.ci/rust-workspace/tests-build')
            corrupt = root / 'corrupt-compiler.json'; corrupt.write_text('{}')
            rejected = run('tests', {'COMPILED_STATE': str(corrupt)})
            self.assertNotEqual(rejected.returncode, 0)
            final = json.loads((root / '.ci/rust-workspace/tests/final.json').read_text())
            self.assertEqual(final['exit_code'], rejected.returncode)
            self.assertIn('mismatch', rejected.stderr)
            self.assertFalse((root / '.ci/rust-workspace/tests/unit-list.json').exists())
            mismatch = run('tests', {'CI_RUST_INCREMENTAL': '0'})
            self.assertNotEqual(mismatch.returncode, 0)
            self.assertIn('mismatch', mismatch.stderr)
            self.assertEqual(json.loads((root / '.ci/rust-workspace/tests/final.json').read_text())['exit_code'], mismatch.returncode)
            def restore_target():
                shutil.rmtree(root / 'rust/target')
                shutil.copytree(snapshot, root / 'rust/target')
            # Every native consumer starts from the producer, rather than from
            # a previous verdict's mutated compiler fingerprint.
            restore_target()
            first = run('tests')
            self.assertEqual(first.returncode, 0, first.stdout + first.stderr)
            self.assertFalse(packed.exists(), 'verdict must not repack the compiler cache')
            self.assertFalse((root / '.ci/rust-workspace/tests/cache-pack.log').exists())
            restore_target()
            native = run('tests')
            self.assertEqual(native.returncode, 0, native.stdout + native.stderr)
            self.assertFalse((root / '.ci/rust-workspace/tests/cache-restore.log').exists())
            self.assertNotIn('Compiling kona-providers-alloy',
                             (root / '.ci/rust-workspace/tests/beacon-list.log').read_text())
            restore_target()
            failed = run('tests', {'RWX_FIXTURE_FAIL': '1'})
            self.assertNotEqual(failed.returncode, 0)
            final = json.loads((root / '.ci/rust-workspace/tests/final.json').read_text())
            self.assertEqual(final['exit_code'], failed.returncode)
            self.assertIn('intentional original failure', (root / '.ci/rust-workspace/tests/junit.xml').read_text())
            self.assertEqual(json.loads((root / '.ci/rust-workspace/tests/unit-coverage.json').read_text())['outcomes']['fail'], 1)
            restore_target()
            fresh = run('tests')
            self.assertEqual(fresh.returncode, 0, fresh.stdout + fresh.stderr)
            # A binary-only worker starts without producer targets and must
            # never inspect the compiler archive, even on its first execution.
            # It extracts only the test archive and compiles nothing.
            shutil.rmtree(root / 'rust/target')
            unit = run('tests-unit', {'COMPILED_TARGET': str(corrupt)})
            self.assertEqual(unit.returncode, 0, unit.stdout + unit.stderr)
            self.assertFalse((root / '.ci/rust-workspace/tests-unit/cache-restore.log').exists())
            self.assertTrue((root / 'rust/target/debug/lokahi').is_file())
            self.assertFalse((root / 'rust/target/debug/.fingerprint').exists())
            self.assertFalse((root / 'rust/target/debug/incremental').exists())
            self.assertFalse([p for p in (root / '.ci/rust-workspace/tests-unit').rglob('*') if p.name == 'lokahi'])
            self.assertFalse((root / '.ci/rust-workspace/tests-unit/beacon.log').exists())
            shutil.rmtree(root / 'rust/target')
            shutil.copytree(snapshot, root / 'rust/target')
            compiler = run('tests-compiler')
            self.assertEqual(compiler.returncode, 0, compiler.stdout + compiler.stderr)
            self.assertFalse((root / '.ci/rust-workspace/tests-compiler/unit.log').exists())
            aggregate_env = {'UNIT_REPORT': str(root / '.ci/rust-workspace/tests-unit'),
                             'COMPILER_REPORT': str(root / '.ci/rust-workspace/tests-compiler')}
            merged = run('tests-merge', aggregate_env)
            self.assertEqual(merged.returncode, 0, merged.stdout + merged.stderr)
            self.assertEqual(len(json.loads((root / '.ci/rust-workspace/tests/final.json').read_text())['reports']), 3)
            failed_unit = run('tests-unit', {'RWX_FIXTURE_FAIL': '1'})
            self.assertNotEqual(failed_unit.returncode, 0)
            merged_failure = run('tests-merge', aggregate_env)
            self.assertEqual(merged_failure.returncode, failed_unit.returncode)
            self.assertIn('intentional original failure', (root / '.ci/rust-workspace/tests/junit.xml').read_text())
            circle_cli = subprocess.run(['cargo', 'test', '--locked', '--test', 'cli'],
                                        cwd=root / 'rust', env=env, capture_output=True, text=True, timeout=180)
            self.assertEqual(circle_cli.returncode, 0, circle_cli.stdout + circle_cli.stderr)
            doc = run('doctest')
            self.assertEqual(doc.returncode, 0, doc.stdout + doc.stderr)
            self.assertEqual(json.loads((root / '.ci/rust-workspace/doctest/settings.json').read_text())['incremental'], '0')
            circle = subprocess.run(['bash', '../ops/ci/runtime/rust-workspace.sh', 'doctest'],
                                    cwd=root / 'rust', env={**env, 'CI_RUST_PROVIDER': 'circleci'},
                                    capture_output=True, text=True, timeout=180)
            self.assertEqual(circle.returncode, 0, circle.stdout + circle.stderr)
            self.assertEqual(json.loads((root / '.ci/rust-workspace/doctest/settings.json').read_text())['incremental'], '0')
            subprocess.run(['sccache', '--stop-server'], check=True, capture_output=True)
            for index in ('0', '9'):
                feature = run('features', {'CI_RUST_PARTITION_INDEX': index})
                self.assertEqual(feature.returncode, 0, feature.stdout + feature.stderr)
                settings = json.loads((root / f'.ci/rust-workspace/features-{index}/settings.json').read_text())
                self.assertIsNone(settings['incremental'])
                self.assertEqual(set(settings['profile_incremental'].values()), {'true'})
            feature_env = {'CI_RUST_PARTITION_INDEX': '0', 'CI_RUST_PARTITION_TOTAL': '1'}
            producer = run('features', feature_env)
            self.assertEqual(producer.returncode, 0, producer.stdout + producer.stderr)
            consumer = run('features', feature_env | {'RWX_RUN_ID': 'b'*32})
            self.assertEqual(consumer.returncode, 0, consumer.stdout + consumer.stderr)
            feature_report = root/'.ci/rust-workspace/features-0'
            self.assertEqual(json.loads((feature_report/'settings.json').read_text())['rwx_run_id'], 'b'*32)
            for phase in ('features', 'feature-tests'):
                log = (feature_report/(phase+'.log')).read_text()
                self.assertNotRegex(log, r'(?m)^\s*(?:Checking|Compiling) ')
            source.write_text(source.read_text()+'\nthis is an intentional syntax error\n')
            subprocess.run(['git', '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid',
                            'commit', '-am', 'feature source error', '-q'], cwd=root, check=True, capture_output=True)
            env['CI_COMMIT_SHA'] = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()
            failure = run('features', feature_env | {'RWX_RUN_ID': 'c'*32})
            self.assertNotEqual(failure.returncode, 0)
            self.assertEqual(json.loads((feature_report/'final.json').read_text())['exit_code'], failure.returncode)
            self.assertIn('error', (feature_report/'features.log').read_text())
            # Restore valid source and commit its new identity before Circle replay.
            source.write_text(source.read_text().replace('\nthis is an intentional syntax error\n', ''))
            subprocess.run(['git', '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid',
                            'commit', '-am', 'restore feature source', '-q'], cwd=root, check=True, capture_output=True)
            env['CI_COMMIT_SHA'] = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()
            # Circle's reusable template defaults to a single unpartitioned node.
            single = run('features', {'CI_RUST_PARTITION_INDEX': '0', 'CI_RUST_PARTITION_TOTAL': '1'})
            self.assertEqual(single.returncode, 0, single.stdout + single.stderr)
            for phase in ('features', 'feature-tests'):
                self.assertIn('without partition', (feature_report/(phase+'.log')).read_text())


if __name__ == '__main__':
    unittest.main()
