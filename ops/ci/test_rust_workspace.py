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

    def test_compile_only_doctest_and_dependency_path(self):
        (self.root / 'settings.json').write_text('{"cargo_home":"/provider/cargo"}')
        (self.root / 'doctests-list.log').write_text('/provider/cargo/git/pin/src/lib.rs - Thing (line 4): test\n')
        (self.root / 'doctests.log').write_text('test /provider/cargo/git/pin/src/lib.rs - Thing (line 4) - compile ... ok\n')
        report = REPORT.libtest_report(self.root, 'doctests')
        self.assertEqual(report['missing'], [])
        self.assertEqual(report['cases'][0]['name'], '<cargo>/git/pin/src/lib.rs - Thing (line 4)')
        self.assertTrue(report['cases'][0]['compile_only'])

    def test_feature_plan_must_execute_exactly_once(self):
        (self.root / 'settings.json').write_text(json.dumps({'workspace_root': str(self.root),
            'feature_partition_index': 0, 'feature_partitions': 10}))
        (self.root / 'workspace.json').write_text(json.dumps({'packages': [
            {'name': 'crate', 'manifest_path': str(self.root / 'rust/crate/Cargo.toml')}]}))
        for phase in ('features', 'feature-tests'):
            (self.root / (phase + '-list.log')).write_text('cargo check --manifest-path crate/Cargo.toml --all-features\n')
            (self.root / (phase + '.log')).write_text('info: running `cargo check --all-features` on crate (1/1)\n')
        REPORT.feature_report(self.root)
        (self.root / 'features.log').write_text('')
        with self.assertRaisesRegex(ValueError, 'Incomplete'):
            REPORT.feature_report(self.root)

    def test_feature_partitions_validate_skips_and_empty_assignments(self):
        (self.root / 'workspace.json').write_text(json.dumps({'packages': [
            {'name': 'crate', 'manifest_path': str(self.root / 'rust/crate/Cargo.toml')}]}))
        (self.root / 'settings.json').write_text(json.dumps({'workspace_root': str(self.root),
            'feature_partition_index': 1, 'feature_partitions': 10}))
        for phase in ('features', 'feature-tests'):
            (self.root / (phase + '-list.log')).write_text('cargo check --manifest-path crate/Cargo.toml\n')
            (self.root / (phase + '.log')).write_text('info: skipping `cargo check` on crate (1/1)\n')
        REPORT.feature_report(self.root)
        (self.root / 'features.log').write_text('info: running `cargo check` on crate (1/1)\n')
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

    def interop_files(self):
        dump = ''
        for variant in ('false', 'true'):
            dump += f'activate={variant}\ngas=0x0000000000000001\n--- tx 0 ---\n'
            dump += ''.join(f'{field}=original\n' for field in
                ('source_hash', 'from', 'to', 'mint', 'value', 'gas_limit', 'is_system_tx', 'data'))
        for language in ('go', 'rust'):
            (self.root / f'interop-{language}.stdout').write_text(dump)
            (self.root / f'interop-{language}.stderr').write_text('original diagnostic\n')
            (self.root / f'interop-{language}.exit').write_text('0\n')

    def test_interop_requires_byte_equality_and_complete_cases(self):
        self.interop_files()
        self.assertEqual(len(REPORT.interop_report(self.root)), 4)
        (self.root / 'interop-rust.stdout').write_text('different\n')
        with self.assertRaisesRegex(ValueError, 'differ'):
            REPORT.interop_report(self.root)
        self.interop_files()
        for language in ('go', 'rust'):
            path = self.root / f'interop-{language}.stdout'
            path.write_text(path.read_text().replace('data=original\n', ''))
        with self.assertRaisesRegex(ValueError, 'fields'):
            REPORT.interop_report(self.root)

    def test_interop_dumper_failure_is_not_an_empty_match(self):
        self.interop_files()
        (self.root / 'interop-go.exit').write_text('7\n')
        with self.assertRaisesRegex(ValueError, 'did not succeed'):
            REPORT.interop_report(self.root)
        self.interop_files()
        for language in ('go', 'rust'):
            (self.root / f'interop-{language}.stdout').write_text('')
        with self.assertRaisesRegex(ValueError, 'empty'):
            REPORT.interop_report(self.root)


class InteropScriptTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / 'go.mod').touch()
        (self.root / 'rust/kona').mkdir(parents=True)
        (self.root / 'ops/scripts').mkdir(parents=True)
        self.script = self.root / 'ops/scripts/test-interop-deposits-diff.sh'
        shutil.copyfile(SCRIPTS.parent / 'scripts/test-interop-deposits-diff.sh', self.script)
        self.bin = self.root / 'bin'
        self.bin.mkdir()

    def run_diff(self, go_status=0, rust_status=0, different=False):
        for name, status in (('go', go_status), ('cargo', rust_status)):
            path = self.bin / name
            text = 'different' if name == 'cargo' and different else 'original stdout'
            path.write_text(f'#!/usr/bin/env bash\nprintf "%s\\n" "{text}"\nprintf "%s\\n" "{name} original stderr" >&2\nexit {status}\n')
            path.chmod(0o755)
        return subprocess.run(['bash', str(self.script)], env=os.environ | {
            'PATH': str(self.bin) + os.pathsep + os.environ['PATH'],
            'CI_INTEROP_REPORT_DIR': str(self.root / 'report')}, capture_output=True, text=True)

    def test_complete_original_streams_survive_success_and_diff_failure(self):
        self.assertEqual(self.run_diff().returncode, 0)
        self.assertEqual(self.run_diff(different=True).returncode, 1)
        for language in ('go', 'rust'):
            self.assertIn('original stderr', (self.root / f'report/interop-{language}.stderr').read_text())
            self.assertEqual((self.root / f'report/interop-{language}.exit').read_text(), '0\n')
        self.assertEqual((self.root / 'report/interop-rust.stdout').read_text(), 'different\n')

    def test_original_dumper_exit_is_retained(self):
        self.assertEqual(self.run_diff(go_status=7).returncode, 2)
        self.assertEqual((self.root / 'report/interop-go.exit').read_text(), '7\n')
        self.assertEqual(self.run_diff(rust_status=9).returncode, 2)
        self.assertEqual((self.root / 'report/interop-rust.exit').read_text(), '9\n')


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
    def test_circle_adapters_preserve_generic_fallbacks(self):
        definition = SCRIPTS.parent.parent / '.circleci/continue/rust-ci.yml'
        config = json.loads(subprocess.check_output(['yq', '-o=json', '.', str(definition)], text=True))
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root / 'rust').mkdir()
            (root / 'other').mkdir()
            (root / 'ops/ci').mkdir(parents=True)
            helper = root / 'ops/ci/rust-workspace.sh'
            helper.write_text('#!/usr/bin/env bash\nprintf "shared:%s\\n" "$1"\n')
            binaries = root / 'bin'
            binaries.mkdir()
            for name in ('cargo', 'rustup', 'typos', 'zepter'):
                path = binaries / name
                path.write_text(f'#!/usr/bin/env bash\nprintf "{name}:%s\\n" "$*"\n')
                path.chmod(0o755)
            env = os.environ | {'PATH': str(binaries) + os.pathsep + os.environ['PATH']}
            def run(job, params):
                step = next(s['run'] for s in config['jobs'][job]['steps'] if isinstance(s, dict) and 'run' in s)
                command = step['command']
                for key, value in params.items():
                    command = command.replace(f'<<parameters.{key}>>', value)
                result = subprocess.run(['bash', '-e', '-c', command], cwd=root / params['directory'],
                                        env=env, capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stderr)
                return result.stdout
            for job, params, expected in (
                ('rust-ci-typos', {'directory': 'rust'}, 'shared:typos'),
                ('rust-ci-typos', {'directory': 'other'}, 'typos:'),
                ('rust-ci-zepter', {'directory': 'rust', 'command': 'zepter run check'}, 'shared:zepter'),
                ('rust-ci-zepter', {'directory': 'rust', 'command': 'zepter run custom'}, 'zepter:run custom'),
            ):
                with self.subTest(job=job, params=params):
                    self.assertEqual(run(job, params).strip(), expected)
            invocations = config['workflows']['rust-ci']['jobs']
            for mode, name in (('wasm-unknown', 'rust-wasm-unknown'), ('wasm-wasi', 'rust-wasm-wasi')):
                params = next(i['rust-ci-cargo-hack-build'] for i in invocations if isinstance(i, dict)
                              and i.get('rust-ci-cargo-hack-build', {}).get('name') == name)
                params = {k: params[k] for k in ('directory', 'target', 'flags')}
                self.assertEqual(run('rust-ci-cargo-hack-build', params).strip(), 'shared:' + mode)
                params['flags'] = '--workspace'
                self.assertEqual(run('rust-ci-cargo-hack-build', params).splitlines(), [
                    'rustup:target add ' + params['target'],
                    'cargo:hack build --target ' + params['target'] + ' --workspace'])

    @unittest.skipUnless(shutil.which('yq'), 'requires the pinned yq tool')
    def test_fresh_verdicts_keep_compiler_caches_enabled(self):
        definition = SCRIPTS.parent.parent / '.rwx/rust.yml'
        tasks = json.loads(subprocess.check_output(['yq', '-o=json', '.tasks', str(definition)], text=True))
        selected = {'tests', 'doctest', 'build', 'docs', 'clippy', 'no-std', 'udeps'}
        selected.update({'wasm-unknown', 'wasm-wasi', 'zepter', 'typos', 'registry', 'interop'})
        selected.update(f'features-{i}' for i in range(10))
        actual = {t['key']: t for t in tasks if t['key'] in selected}
        self.assertEqual(set(actual), selected)
        for name, task in actual.items():
            with self.subTest(task=name):
                self.assertNotEqual(task.get('cache'), False, 'cache:false disables tool caches')
                self.assertTrue(task.get('tool-cache'))
                for nonce in ('RWX_RUN_ID', 'RWX_TASK_ATTEMPT_NUMBER'):
                    self.assertEqual(task['env'][nonce]['cache-key'], 'included')
                output = task['outputs']['filesystem']['filter']['workspace']
                self.assertIn('rust/target', output)
                self.assertNotIn('.ci/rust-workspace', output)
        self.assertIn('head-source', actual['tests']['use'])


@unittest.skipUnless(os.environ.get('RWX_LIVE_RUST_FIXTURE') == '1', 'opt-in pinned Linux Rust fixture')
class LiveRunnerTests(unittest.TestCase):
    def test_archived_tests_are_fresh_and_failure_reports_survive(self):
        """Exercise the real shell, nextest archive, doctests and cache preparation."""
        repo = SCRIPTS.parent.parent
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            helpers = root / 'ops/ci'
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
            circle = subprocess.run(['bash', '../ops/ci/rust-workspace.sh', 'doctest'],
                                    cwd=root / 'rust', env={**env, 'CI_RUST_PROVIDER': 'circleci'},
                                    capture_output=True, text=True, timeout=180)
            self.assertEqual(circle.returncode, 0, circle.stdout + circle.stderr)
            for index in ('0', '9'):
                feature = run('features', {'CI_RUST_PARTITION_INDEX': index})
                self.assertEqual(feature.returncode, 0, feature.stdout + feature.stderr)
            # Circle's reusable template defaults to a single unpartitioned node.
            single = run('features', {'CI_RUST_PARTITION_INDEX': '0', 'CI_RUST_PARTITION_TOTAL': '1'})
            self.assertEqual(single.returncode, 0, single.stdout + single.stderr)
            coverage = json.loads((root / '.ci/rust-workspace/features-0/feature-coverage.json').read_text())
            self.assertTrue(all(c['command_count'] == len(c['executed']) for c in coverage.values()))


if __name__ == '__main__':
    unittest.main()
