import importlib.util
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest

SCRIPTS = Path(__file__).resolve().parents[1] / 'runtime'
SPEC = importlib.util.spec_from_file_location('fetcher', SCRIPTS / 'fetcher-artifacts.py')
F = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(F)


class SelectionTests(unittest.TestCase):
    def test_portability_changes_only_remapping_metadata_and_preserves_executable_fields(self):
        metadata = {'compiler': {'version': 'original'}, 'settings': {'remappings':
            ['/checkout/packages/contracts-bedrock/lib/example/:dep/=lib/dep/', 'lib/example/:dep/=lib/dep/']},
            'sources': {'original.sol': {'keccak256':'original source hash'}}}
        value = {'metadata': metadata, 'rawMetadata': json.dumps(metadata),
                 'abi': ['original'], 'bytecode': {'object':'original','sourceMap':'original'}, 'ast': {'id':13}}
        result = F.portable_artifact(value, Path('/checkout/packages/contracts-bedrock'))
        self.assertEqual(result['metadata']['settings']['remappings'], ['lib/example/:dep/=lib/dep/'])
        self.assertEqual(result['metadata']['sources'], metadata['sources'])
        for key in ('abi','bytecode','ast'): self.assertEqual(result[key], value[key])
        self.assertEqual(F.portable_artifact(result, Path('/different/root')), result)
        result['metadata']['settings']['remappings'] = ['/foreign/lib/:dep/=lib/dep/']
        result['rawMetadata'] = json.dumps(result['metadata'])
        with self.assertRaisesRegex(ValueError, 'Foreign absolute'): F.portable_artifact(result, Path('/checkout/packages/contracts-bedrock'))

    def test_portability_preserves_raw_solc_fields_omitted_by_foundrys_typed_metadata(self):
        metadata = {'compiler': {'version':'original'}, 'sources': {'input.sol': {'keccak256':'original'}},
                    'settings': {'remappings':['lib/=lib/']}, 'output': {'userdoc':{}}}
        raw = json.loads(json.dumps(metadata)); raw['output']['userdoc']['errors'] = {'OriginalError()':[{'notice':'original error doc'}]}
        value = {'metadata':metadata, 'rawMetadata':json.dumps(raw)}
        result = F.portable_artifact(value, Path('/checkout'))
        self.assertEqual(json.loads(result['rawMetadata'])['output'], raw['output'])
        self.assertEqual(result['metadata']['output'], metadata['output'])

    def test_failed_export_validation_preserves_all_existing_artifact_bytes(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp); source = root/'compiled'; target = root/'committed'; source.mkdir(); target.mkdir()
            (target/'original.json').write_text('original committed bytes')
            metadata = {'settings': {'compilationTarget': {'scripts/FetchChainInfo.s.sol':'NewContract'}, 'remappings':[]}}
            (source/'NewContract.json').write_text(json.dumps({'metadata':metadata, 'rawMetadata':'invalid solc JSON'}))
            with self.assertRaises(ValueError): F.export(source, target)
            self.assertEqual({p.name:p.read_text() for p in target.iterdir()}, {'original.json':'original committed bytes'})

    def test_inventory_rejects_empty_foreign_and_linked_inputs(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            with self.assertRaisesRegex(ValueError, 'Empty'): F.inventory(root)
            path = root / 'NewContract.json'
            path.write_text(json.dumps({'metadata': {'settings': {'compilationTarget': {'scripts/FetchChainInfo.s.sol': 'NewContract'}}}}))
            self.assertEqual(set(F.inventory(root)), {'NewContract.json'})
            value = json.loads(path.read_text()); value['metadata']['settings']['compilationTarget'] = {'src/Foreign.sol':'NewContract'}
            path.write_text(json.dumps(value))
            with self.assertRaisesRegex(ValueError, 'different source'): F.inventory(root)
            path.unlink(); path.symlink_to(root / 'missing.json')
            with self.assertRaisesRegex(ValueError, 'link'): F.inventory(root)


class _LiveTestsFixtures:

    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        self.contracts = self.root / 'packages/contracts-bedrock'
        target = self.root / 'ops/ci/runtime'
        target.mkdir(parents=True)
        for path in SCRIPTS.glob('*.py'):
            shutil.copy2(path, target / path.name)
        files = {'op-fetcher/justfile': (F.ROOT / 'op-fetcher/justfile').read_text(), 'packages/contracts-bedrock/justfile': 'clean:\n  rm -rf forge-artifacts cache\nbuild-dev *ARGS:\n  FOUNDRY_PROFILE=lite forge build {{ARGS}}\n', 'packages/contracts-bedrock/foundry.toml': '[profile.default]\nsrc="src"\nscript="scripts"\nout="forge-artifacts"\n[profile.lite]\noptimizer=false\n', 'packages/contracts-bedrock/scripts/FetchChainInfo.s.sol': '// SPDX-License-Identifier: MIT\n' + next((line for line in (F.ROOT / 'packages/contracts-bedrock/scripts/FetchChainInfo.s.sol').read_text().splitlines() if line.startswith('pragma solidity '))) + '\ncontract FetchChainInfo { function value() external pure returns(uint256) {return 42;} }\ncontract FutureContract {}\n', '.gitignore': '.ci/\npackages/contracts-bedrock/forge-artifacts/\npackages/contracts-bedrock/cache/\n'}
        for name, content in files.items():
            path = self.root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content)
        self.contracts.joinpath('src').mkdir()
        self.env = {**os.environ, 'CI': 'true', 'CI_BRANCH': 'codex/rwx-ci-pilot', 'RWX_RUN_ID': os.environ.get('RWX_RUN_ID', 'isolated-fetcher-fixture'), 'RWX_TASK_ATTEMPT_NUMBER': '1'}
        for name in list(self.env):
            if name.startswith(('FOUNDRY_', 'DAPP_', 'DEV_FEATURE__', 'SYS_FEATURE__')):
                del self.env[name]
        self.run_command(['git', 'init', '-q'], self.root)
        for key, value in [('user.name', 'CI fixture'), ('user.email', 'ci-fixture@example.invalid')]:
            self.run_command(['git', 'config', key, value], self.root)
        library = self.root / '.ci/library'
        library.mkdir(parents=True)
        self.run_command(['git', 'init', '-q'], library)
        for key, value in [('user.name', 'CI fixture'), ('user.email', 'ci-fixture@example.invalid')]:
            self.run_command(['git', 'config', key, value], library)
        (library / 'input.txt').write_text('original fixture submodule\n')
        self.run_command(['git', 'add', '.'], library)
        self.run_command(['git', 'commit', '-qm', 'Library input'], library)
        self.run_command(['git', '-c', 'protocol.file.allow=always', 'submodule', 'add', '-q', str(library), 'packages/contracts-bedrock/lib/fixture'], self.root)
        self.run_command(['just', 'build-contracts'], self.root / 'op-fetcher')
        self.commit()
        self.report = self.root / '.ci/fetcher-artifacts/run'

    def run_command(self, argv, cwd):
        result = subprocess.run(argv, cwd=cwd, env=self.env, capture_output=True, text=True, timeout=120)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return result.stdout.strip()

    def commit(self):
        self.run_command(['git', 'add', '.'], self.root)
        self.run_command(['git', 'commit', '-qm', 'Fetcher input'], self.root)
        self.sha = self.run_command(['git', 'rev-parse', 'HEAD'], self.root)
        self.env['CI_COMMIT_SHA'] = self.sha

    def execute(self, provider='circleci'):
        return subprocess.run([sys.executable, 'ops/ci/runtime/fetcher-artifacts.py', '--provider', provider], cwd=self.root, env=self.env, capture_output=True, text=True, timeout=120)

    def retain(self, name):
        target = F.ROOT / '.ci/fetcher-artifacts/helper-fixtures' / name
        shutil.rmtree(target, ignore_errors=True)
        shutil.copytree(self.report, target)


@unittest.skipUnless(os.environ.get('RWX_LIVE_FETCHER_FIXTURE') == '1', 'Opt-in actual Forge, Just and diff workloads')
class LiveTests(_LiveTestsFixtures, unittest.TestCase):

    def test_complete_actual_build_preserves_future_artifacts_and_original_parity(self):
        paired = self.root / '.ci/paired'
        for provider, path in [('circleci', 'circle'), ('rwx', 'rwx')]:
            result = self.execute(provider)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.retain('complete-' + path)
            shutil.copytree(self.report, paired / path)
        for name in ('settings.json', 'selection.json', 'coverage.json', 'build.stage.json'):
            path = paired / 'rwx' / name
            original = path.read_bytes()
            value = F.G.read(path)
            if name == 'settings.json':
                value['source_sha'] = '0' * 40
            if name == 'selection.json':
                value['committed_sha256'].pop('FutureContract.json')
            if name == 'coverage.json':
                value['artifact_files'] = 1
            if name == 'build.stage.json':
                value['argv'] = ['true']
            F.S.write(path, value)
            final_path = paired / 'rwx/final.json'
            final = F.G.read(final_path)
            original_final = final_path.read_bytes()
            final['original_sha256'][name] = F.S.digest(path)
            F.S.write(final_path, final)
            path.write_bytes(original)
            final_path.write_bytes(original_final)

    def test_checked_in_drift_is_not_overwritten_by_compilation(self):
        path = self.root / F.COMMITTED / 'FetchChainInfo.json'
        value = F.G.read(path)
        value['abi'] = []
        F.S.write(path, value)
        self.commit()
        original = path.read_bytes()
        result = self.execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(path.read_bytes(), original)
        self.assertIn('differ from committed', result.stderr)
        self.assertEqual(F.G.read(self.report / 'diff.stage.json')['exit_code'], 1)
        self.retain('committed-drift')

    def test_missing_checked_in_artifact_is_retained_as_a_failing_diff(self):
        path = self.root / F.COMMITTED / 'FutureContract.json'
        path.unlink()
        self.commit()
        result = self.execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(path.exists())
        self.assertIn('Only in', (self.report / 'diff.log').read_text())
        self.retain('missing-artifact')

    def test_actual_compilation_failure_keeps_original_diagnostics_and_untouched_inputs(self):
        path = self.contracts / 'scripts/FetchChainInfo.s.sol'
        path.write_text('invalid original solidity fixture\n')
        self.commit()
        original = F.inventory(self.root / F.COMMITTED)
        result = self.execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertNotEqual(F.G.read(self.report / 'build.stage.json')['exit_code'], 0)
        self.assertFalse((self.report / 'diff.stage.json').exists())
        self.assertEqual(F.inventory(self.root / F.COMMITTED), original)
        self.retain('compile-failure')

    def test_actual_just_cancellation_retains_partial_build_without_a_diff_verdict(self):
        path = self.root / 'op-fetcher/justfile'
        text = path.read_text().replace('  just build-dev --deny-warnings --skip test\n', '  just build-dev --deny-warnings --skip test\n  touch ../../.ci/cancel-ready\n  sleep 30\n')
        path.write_text(text)
        self.commit()
        process = subprocess.Popen([sys.executable, 'ops/ci/runtime/fetcher-artifacts.py', '--provider', 'circleci'], cwd=self.root, env=self.env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        try:
            deadline = time.monotonic() + 20
            while not (self.root / '.ci/cancel-ready').exists() and time.monotonic() < deadline:
                time.sleep(0.05)
            self.assertTrue((self.root / '.ci/cancel-ready').exists())
            process.send_signal(signal.SIGTERM)
            output = process.communicate(timeout=10)[0]
            self.assertEqual(process.returncode, 143, output)
            self.assertEqual(F.G.read(self.report / 'build.stage.json')['exit_code'], 143)
            self.assertFalse((self.report / 'diff.stage.json').exists())
            self.retain('cancelled-build')
        finally:
            if process.poll() is None:
                process.kill()
                process.communicate()


if __name__ == '__main__': unittest.main()
