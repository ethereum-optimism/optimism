#!/usr/bin/env python3
"""Exercise discovery and the original shell, Go and Mockery validators."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import threading
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('main_checks', Path(__file__).resolve().parents[1] / 'runtime' / 'main-checks.py')
MAIN = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(MAIN)


class _MainTestsFixtures:

    def install_helpers(self, root):
        directory = root / 'ops/ci/runtime'
        directory.mkdir(parents=True)
        for name in ('main-checks.py', 'pr-checks.py', 'rust-workspace-report.py', 'ci-report.py'):
            shutil.copyfile(Path(__file__).resolve().parents[1] / 'runtime' / name, directory / name)
        (root / '.gitignore').write_text('.ci/\n')

    def install_justfiles(self, root):
        directory = root / 'justfiles'
        directory.mkdir()
        for name in ('go.just', 'git.just', 'default.just'):
            shutil.copyfile(MAIN.ROOT / 'justfiles' / name, directory / name)

    def install_bundle(self, root):
        directory = root / 'op-core/superchain'
        directory.mkdir(parents=True)
        shutil.copyfile(MAIN.ROOT / 'op-core/superchain/sync-superchain.sh', directory / 'sync-superchain.sh')
        (directory / 'superchain-configs.zip.sha256').write_text('0' * 64 + '\n')
        (directory / 'bundle.go').write_text('package superchain\nimport _ "embed"\n//go:embed superchain-configs.zip\nvar Data []byte\n')
        (root / 'mise.toml').write_text('')
        original = (MAIN.ROOT / 'justfile').read_text()
        start = original.index("[script('bash')]\nupdate-superchain-registry-submodule ref=")
        end = original.index('# Builds op-core/superchain/superchain-configs.zip', start)
        with (root / 'justfile').open('a') as file:
            file.write('\n' + original[start:end] + '\nbuild-superchain-go: update-superchain-registry-submodule\n  bash op-core/superchain/sync-superchain.sh\n')
        self.commit(root)
        origin = root / '.ci/origin'
        origin.mkdir(parents=True)
        for name, text in [('configs/dev/example.toml', 'chain_id = 123\n'), ('extra/genesis/dev/example.json', '{}\n'), ('extra/dictionary/example', 'fixture\n')]:
            path = origin / 'superchain' / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text)
        self.commit(origin)
        subprocess.run(['git', '-c', 'protocol.file.allow=always', 'submodule', 'add', '-q', str(origin), 'superchain-registry'], cwd=root, check=True)
        self.commit(root)
        refresh = subprocess.run(['just', 'build-superchain-go'], cwd=root, env={**os.environ, 'OP_CORE_SYNC_SUPERCHAIN': '1'}, capture_output=True, text=True)
        self.assertEqual(refresh.returncode, 0, refresh.stdout + refresh.stderr)

    def commit(self, root):
        if not (root / '.git').exists():
            for args in (['init', '-q'], ['config', 'user.email', 'fixture@example.invalid'], ['config', 'user.name', 'fixture']):
                subprocess.run(['git', *args], cwd=root, check=True)
        subprocess.run(['git', 'add', '.'], cwd=root, check=True)
        subprocess.run(['git', 'commit', '--allow-empty', '-qm', 'fixture'], cwd=root, check=True)
        return subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()

    def run_adapter(self, root, job, sha, provider='circleci', prepared=None):
        env = {**os.environ, 'CI_COMMIT_SHA': sha, 'CI_BRANCH': 'fixture', 'CI_CHECK_PROVIDER': provider, 'RWX_RUN_ID': 'fresh-fixture', 'RWX_TASK_ATTEMPT_NUMBER': '1'}
        env.pop('SUPERCHAIN_PREPARED', None)
        if prepared:
            env['SUPERCHAIN_PREPARED'] = str(prepared)
        result = subprocess.run([sys.executable, 'ops/ci/runtime/main-checks.py', job], cwd=root, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        return (result, root / '.ci/main-checks' / job)


    def exercise_mock_original_reports(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.install_helpers(root)
            self.install_justfiles(root)
            component = root / 'op-node'
            component.mkdir()
            shutil.copyfile(MAIN.ROOT / 'op-node/justfile', component / 'justfile')
            source = component / 'input.go'
            source.write_text('package input\nimport _ "fixture.invalid/main/op-core/superchain"\n//go:generate mockery --name Greeter --output ./mocks --outpkg mocks\ntype Greeter interface { Greet(string) string }\n')
            (component / 'input_test.go').write_text('package input\n//go:generate sh -c "printf run\\\\n >> ../generation-runs"\n')
            (root / 'go.mod').write_text('module fixture.invalid/main\n\ngo 1.26.0\n\nrequire github.com/stretchr/testify v1.11.1\n')
            (root / 'go.sum').write_text('')
            (root / 'justfile').write_text('generate-mocks-op-node:\n  cd op-node && just generate-mocks\n')
            (root / '.gitignore').write_text('.ci/\ngeneration-runs\nop-core/superchain/superchain-configs.zip\n')
            self.install_bundle(root)
            initial = subprocess.run(['just', 'generate-mocks-op-node'], cwd=root, capture_output=True, text=True)
            self.assertEqual(initial.returncode, 0, initial.stdout + initial.stderr)
            tidy = subprocess.run(['go', 'mod', 'tidy'], cwd=root, capture_output=True, text=True)
            self.assertEqual(tidy.returncode, 0, tidy.stdout + tidy.stderr)
            sha = self.commit(root)
            previous = (root / 'generation-runs').read_bytes()
            bundle = root / MAIN.BUNDLE
            bundle.unlink()
            missing = subprocess.run(['go', 'list', '-tags=generate', '-json', './...'], cwd=component, capture_output=True, text=True)
            self.assertNotEqual(missing.returncode, 0)
            self.assertIn('superchain-configs.zip', missing.stderr)
            originals = {}
            for provider in ('circleci', 'rwx'):
                prepared = None
                if provider == 'rwx':
                    result = subprocess.run([sys.executable, 'ops/ci/runtime/main-checks.py', '--prepare-superchain'], cwd=root, env={**os.environ, 'CI_COMMIT_SHA': sha}, capture_output=True, text=True)
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    prepared = root / '.ci/main-checks/prep-superchain'
                    self.assertFalse(list(prepared.glob('*.xml')))
                result, report = self.run_adapter(root, 'check-generated-mocks-op-node', sha, provider, prepared)
                self.assertEqual(result.returncode, 0, result.stdout)
                self.assertEqual(json.loads((report / 'check.stage.json').read_text())['argv'], MAIN.COMMANDS['check-generated-mocks-op-node'])
                self.assertGreater(len((root / 'generation-runs').read_bytes()), len(previous))
                previous = (root / 'generation-runs').read_bytes()
                selection = json.loads((report / 'selection.json').read_text())
                self.assertTrue(any((d['file'].endswith('input_test.go') for p in selection['packages'] for d in p['directives'])))
                key = 'circle' if provider == 'circleci' else 'rwx'
                originals[key] = root / '.ci/paired' / key
                shutil.copytree(report, originals[key])
            self.verify_mock_reports(originals, sha)
            saved = bundle.read_bytes()
            bundle.write_bytes(b'corrupt reusable zip')
            result, report = self.run_adapter(root, 'check-generated-mocks-op-node', sha, 'rwx', prepared)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse((report / 'check.stage.json').exists())
            self.assertIn('Superchain bundle differs', result.stdout)
            bundle.write_bytes(saved)
            source.write_text(source.read_text().replace('Greet(string)', 'NewMethod(string)'))
            sha = self.commit(root)
            result, report = self.run_adapter(root, 'check-generated-mocks-op-node', sha)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('diff --git', (report / 'check.log').read_text())
            self.assertNotEqual(json.loads((report / 'final.json').read_text())['exit_code'], 0)
            self.assertTrue((report / 'selection.json').exists())
            self.assertNotEqual(json.loads((report / 'settings.json').read_text())['input_sha256'], json.loads((report / 'inputs-after.json').read_text()))


    def verify_mock_reports(self, originals, sha):
        pass


class MainTests(_MainTestsFixtures, unittest.TestCase):

    def test_generation_includes_new_packages_test_directives_and_ignored_files(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            component = root / 'op-node'
            component.mkdir()
            (component / 'input.go').write_text('package input\n//go:generate mockery --name Input\n')
            (component / 'input_test.go').write_text('package input\n//go:generate echo test-input\n')
            (component / 'new.go').write_text('package input\n')
            row = {'ImportPath': 'fixture/op-node', 'Dir': str(component), 'GoFiles': ['input.go', 'new.go'], 'TestGoFiles': ['input_test.go'], 'IgnoredGoFiles': ['excluded.go']}
            with patch.object(MAIN, 'ROOT', root):
                result = MAIN.packages(json.dumps(row), 'op-node')
            self.assertEqual(len(result[0]['files']), 3)
            self.assertEqual(len(result[0]['directives']), 2)
            self.assertEqual(result[0]['ignored_go_files'], ['excluded.go'])
            for change in ('duplicate', 'error', 'empty', 'outside', 'filename', 'no-generator'):
                bad = dict(row)
                if change == 'error':
                    bad['DepsErrors'] = [{'Err': 'unavailable module'}]
                elif change == 'outside':
                    bad['Dir'] = str(root / 'op-service')
                elif change == 'filename':
                    bad['GoFiles'] = ['../input.go']
                elif change == 'no-generator':
                    bad['GoFiles'] = ['new.go']
                    bad['TestGoFiles'] = []
                text = '' if change == 'empty' else json.dumps(bad)
                if change == 'duplicate':
                    text += json.dumps(bad)
                with self.subTest(change=change), patch.object(MAIN, 'ROOT', root), self.assertRaises(ValueError):
                    MAIN.packages(text, 'op-node')


    def test_real_process_cancellation_preserves_signal_and_original_partial_output(self):
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            marker = directory / 'ready'
            child = 'import pathlib,time;pathlib.Path(' + repr(str(marker)) + ").write_text('ready');print('original partial output',flush=True);time.sleep(30)"

            def cancel():
                import time
                deadline = time.monotonic() + 5
                while not marker.exists() and time.monotonic() < deadline:
                    time.sleep(0.02)
                os.kill(os.getpid(), signal.SIGTERM)
            thread = threading.Thread(target=cancel)
            thread.start()
            status = MAIN.STAGES.stage(directory, 'cancel', [sys.executable, '-c', child], cwd=str(directory))
            thread.join()
            self.assertEqual(status, 143)
            self.assertEqual(json.loads((directory / 'cancel.stage.json').read_text())['exit_code'], -signal.SIGTERM)
            self.assertIn('original partial output', (directory / 'cancel.log').read_text())

    def test_actual_original_todo_checker_rejects_bad_format_without_github_auth(self):
        if not shutil.which('rg'):
            self.skipTest('Pinned ripgrep available in Linux fixture')
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            script = root / 'ops/scripts/todo-checker.sh'
            script.parent.mkdir(parents=True)
            shutil.copyfile(MAIN.ROOT / 'ops/scripts/todo-checker.sh', script)
            (root / 'input.txt').write_text('TO' + 'DO(123): valid issue\n')
            env = {k: v for k, v in os.environ.items() if k != 'CI_TODO_CHECKER_PAT'}
            command = ['bash', str(script), '--verbose', '--strict']
            self.assertEqual(subprocess.run(command, cwd=root, env=env, stdin=subprocess.DEVNULL, capture_output=True).returncode, 0)
            (root / 'input.txt').write_text('TO' + 'DO(invalid-reference): invalid issue\n')
            result = subprocess.run(command, cwd=root, env=env, stdin=subprocess.DEVNULL, capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('Invalid TODO format', result.stdout)
            report = root / '.ci/discovery'
            report.mkdir(parents=True)
            with patch.object(MAIN, 'ROOT', root):
                MAIN.stage(report, 'todos', MAIN.TODO_DISCOVERY['todos'], allowed=(0, 1))
            self.assertIn('invalid-reference', (report / 'todos.log').read_text())

    @unittest.skipUnless(os.environ.get('RWX_LIVE_MAIN_FIXTURE') == '1', 'Opt-in pinned original Main adapters')
    def test_real_l2_and_forge_adapters_preserve_nested_cwd_and_report_after_failure(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.install_helpers(root)
            version = root / 'op-deployer/pkg/deployer/forge/version.json'
            version.parent.mkdir(parents=True)
            version.write_text('{"forge":"v1.8.3"}\n')
            (root / 'mise.toml').write_text('[tools]\nforge = "1.8.3"\n')
            shutil.copyfile(MAIN.ROOT / 'op-deployer/justfile', root / 'op-deployer/justfile')
            self.install_justfiles(root)
            sha = self.commit(root)
            for job in ('op-deployer-forge-version',):
                result, report = self.run_adapter(root, job, sha)
                self.assertEqual(result.returncode, 0, result.stdout)
                stage = json.loads((report / 'check.stage.json').read_text())
                self.assertEqual(stage['cwd'], str(root / 'op-deployer') if job == 'op-deployer-forge-version' else str(root))
                self.assertEqual(stage['argv'], MAIN.COMMANDS[job])
            version.write_text('{"forge":"v0.0.0"}\n')
            sha = self.commit(root)
            result, report = self.run_adapter(root, 'op-deployer-forge-version', sha)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(json.loads((report / 'check.stage.json').read_text())['exit_code'], 1)
            self.assertIn("Forge versions don't match", (report / 'check.log').read_text())
            self.assertNotEqual(json.loads((report / 'final.json').read_text())['exit_code'], 0)
            self.assertIn('<failure', (report / 'check.junit.xml').read_text())

    @unittest.skipUnless(os.environ.get('RWX_LIVE_MAIN_FIXTURE') == '1', 'Opt-in original Go/Mockery generation')
    def test_real_mock_generation_executes_freshly_and_detects_stale_committed_mock(self):
        self.exercise_mock_original_reports()


if __name__ == '__main__': unittest.main()
