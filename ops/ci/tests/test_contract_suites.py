#!/usr/bin/env python3
"""Exercise complete file selection and real Forge/Go preparation and verdicts."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('suites', Path(__file__).resolve().parents[1] / 'runtime' / 'contract-suites.py')
CS = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(CS)


class SuiteTests(unittest.TestCase):
    def test_input_mutation_retains_exact_before_and_after_evidence(self):
        before = {'test/Fixture.t.sol': 'a'*64, 'go.sum': 'b'*64}
        after = dict(before, **{'go.sum': 'c'*64})
        with tempfile.TemporaryDirectory() as tmp, patch.object(CS, 'inputs', return_value=after):
            directory = Path(tmp)
            with self.assertRaisesRegex(ValueError, 'verdict changed source inputs: go.sum'):
                CS.verify_inputs(directory, before, 'verdict')
            self.assertEqual(json.loads((directory / 'source-after-verdict.json').read_text()), after)
            self.assertEqual(json.loads((directory / 'source-changes-verdict.json').read_text()),
                             {'go.sum': {'before': 'b'*64, 'after': 'c'*64}})

    def test_only_selected_writers_declare_outputs_and_changed_fixtures_are_retained(self):
        self.assertEqual(CS.runtime_outputs(['test/unrelated.t.sol']), [])
        self.assertEqual(CS.runtime_outputs(list(CS.TRACKED_OUTPUTS)), sorted(CS.TRACKED_OUTPUTS.values()))
        name = next(iter(CS.TRACKED_OUTPUTS.values()))
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'repo'; directory = Path(tmp) / 'report'; directory.mkdir()
            original = directory / 'tracked-fixtures/before' / name; original.parent.mkdir(parents=True); original.write_text('original')
            source = root / name; source.parent.mkdir(parents=True); source.write_text('generated')
            before = {name: CS.UP.digest(original), 'source.sol': 'a'*64}
            after = {name: CS.UP.digest(source), 'source.sol': 'a'*64}
            with patch.object(CS, 'ROOT', root), patch.object(CS, 'inputs', return_value=after):
                CS.verify_inputs(directory, before, 'verdict', [name])
                self.assertEqual((directory / 'tracked-fixtures/after' / name).read_text(), 'generated')
                self.assertEqual(json.loads((directory / 'runtime-fixtures.json').read_text()),
                                 {name: {'before_sha256': before[name], 'after_sha256': after[name]}})
                after['source.sol'] = 'b'*64
                with self.assertRaisesRegex(ValueError, 'changed source inputs: source.sol'):
                    CS.verify_inputs(directory, before, 'verdict', [name])
                with self.assertRaisesRegex(ValueError, 'Compilation cannot change'):
                    CS.verify_inputs(directory, before, 'preparation', [name])

    def test_files_include_new_nested_tests_and_reject_duplicates_or_unsafe_filters(self):
        self.assertEqual(CS.file_selection('test/New.t.sol\ntest/nested/Old.t.sol\n'), ['test/New.t.sol', 'test/nested/Old.t.sol'])
        self.assertEqual(CS.match_path(['test/New.t.sol', 'test/nested/Old.t.sol']), './test/{New.t.sol,nested/Old.t.sol}')
        for value in ('test/A.t.sol\ntest/A.t.sol\n', '../test/A.t.sol', '/test/A.t.sol', 'test/{A,B}.t.sol', 'test/a.sol'):
            with self.subTest(value=value), self.assertRaises(ValueError): CS.file_selection(value)
        with self.assertRaises(ValueError): CS.match_path([])

    def test_profiles_and_features_keep_all_fuzzing_and_clear_ambient_filters(self):
        with patch.dict(os.environ, {'CI_BRANCH': 'pilot', 'FOUNDRY_MATCH_TEST': 'partial', 'FOUNDRY_FUZZ_RUNS': '1',
                                    'DEV_FEATURE__UNKNOWN': 'true', 'ETH_RPC_URL': 'ambient'}, clear=True):
            self.assertEqual(CS.configure('modified', 'CUSTOM_GAS_TOKEN'), ('pilot', 'ciheavy'))
            self.assertEqual(os.environ['SYS_FEATURE__CUSTOM_GAS_TOKEN'], 'true')
            for name in ('FOUNDRY_MATCH_TEST', 'FOUNDRY_FUZZ_RUNS', 'DEV_FEATURE__UNKNOWN', 'ETH_RPC_URL'):
                self.assertNotIn(name, os.environ)
            os.environ['CI_BRANCH'] = 'develop'; self.assertEqual(CS.configure('standard', 'main'), ('develop', 'ci'))
            self.assertNotIn('SYS_FEATURE__CUSTOM_GAS_TOKEN', os.environ)

    def test_pinned_discovery_requires_an_available_full_commit_and_rwx_provider(self):
        sha = 'a' * 40
        self.assertIn(sha + '...HEAD', CS.file_command('modified', sha)[-1])
        self.assertEqual(CS.file_command('modified'), CS.FILE_COMMANDS['modified'])
        for suite, target in [('standard', sha), ('modified', 'develop'), ('modified', 'a' * 39), ('modified', 'a;false')]:
            with self.subTest(suite=suite, target=target), self.assertRaises(ValueError): CS.file_command(suite, target)
        with patch.dict(os.environ, {'CI_CONTRACT_TARGET_SHA': sha, 'CI_CONTRACT_PROVIDER': 'circleci'}, clear=True):
            with self.assertRaisesRegex(ValueError, 'requires RWX'): CS.pinned_target('modified')
        with patch.dict(os.environ, {'CI_CONTRACT_TARGET_SHA': sha, 'CI_CONTRACT_PROVIDER': 'rwx'}, clear=True), \
             patch.object(CS.UP, 'command', return_value='b' * 40):
            with self.assertRaisesRegex(ValueError, 'available commit'): CS.pinned_target('modified')

    def test_effective_heavy_settings_reject_reduced_runs_depth_or_timeout(self):
        config = {'fuzz': {'runs': 20000, 'timeout': 300}, 'invariant': {'runs': 128, 'depth': 512, 'timeout': 300}}
        CS.validate_config(config, 'modified')
        for key, field in [('fuzz', 'runs'), ('fuzz', 'timeout'), ('invariant', 'runs'), ('invariant', 'depth'), ('invariant', 'timeout')]:
            value = json.loads(json.dumps(config)); value[key][field] = 1
            with self.subTest(key=key, field=field), self.assertRaises(ValueError): CS.validate_config(value, 'modified')
        config['match_test'] = 'partial'
        with self.assertRaises(ValueError): CS.validate_config(config, 'modified')


class _LiveSuiteTestsFixtures:

    def verify_provider_reports(self, root, directories, sha, suite, tmp):
        pass

    def exercise_original_reports(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'repo'
            root.mkdir()
            contracts = root / 'packages/contracts-bedrock'
            (contracts / 'test/unit').mkdir(parents=True)
            (contracts / 'src/unit').mkdir(parents=True)
            (contracts / 'scripts/go-ffi').mkdir(parents=True)
            (root / 'ops/ci/runtime').mkdir(parents=True)
            for source in Path(CS.__file__).parent.glob('*.py'):
                if not source.name.startswith('test_'):
                    shutil.copy2(source, root / 'ops/ci/runtime' / source.name)
            text = subprocess.check_output(['go', 'list', '-deps', '-json', './packages/contracts-bedrock/scripts/checks/test-validation'], cwd=CS.ROOT, text=True)
            decoder, offset = (json.JSONDecoder(), 0)
            while offset < len(text):
                if text[offset].isspace():
                    offset += 1
                    continue
                row, offset = decoder.raw_decode(text, offset)
                if not row.get('Module', {}).get('Main'):
                    continue
                src = Path(row['Dir'])
                dst = root / src.relative_to(CS.ROOT)
                dst.mkdir(parents=True, exist_ok=True)
                for f in src.glob('*.go'):
                    shutil.copy2(f, dst / f.name)
                for name in row.get('EmbedFiles', []):
                    target = dst / name
                    target.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(src / name, target)
            for name in ('go.mod', 'go.sum'):
                shutil.copy2(CS.ROOT / name, root / name)
            shutil.copy2(CS.CONTRACTS / 'scripts/checks/test-validation/exclusions.toml', contracts / 'scripts/checks/test-validation/exclusions.toml')
            guard = contracts / 'scripts/checks/check-junit-tests-ran.sh'
            shutil.copy2(CS.CONTRACTS / 'scripts/checks/check-junit-tests-ran.sh', guard)
            original = (CS.CONTRACTS / 'justfile').read_text()
            recipes = []
            for name in ('build-go-ffi', 'test-rerun', 'lint-forge-tests-check-no-build'):
                start = original.index('\n' + name + ':')
                end = original.index('\n\n', start + 1)
                recipes.append(original[start + 1:end])
            (contracts / 'justfile').write_text('\n\n'.join(recipes) + '\n')
            (contracts / 'scripts/go-ffi/main.go').write_text('package main\nimport("fmt";"os")\nfunc main(){f,e:=os.OpenFile(os.Getenv("CI_SUITE_FIXTURE_MARKER"),os.O_APPEND|os.O_CREATE|os.O_WRONLY,0600);if e!=nil{panic(e)};defer f.Close();f.WriteString("x");v:=1;if os.Getenv("CI_SUITE_FIXTURE_FAIL")=="1"{v=0};fmt.Printf("0x%064x",v)}\n')
            (contracts / 'foundry.toml').write_text('[profile.default]\nsrc="src"\ntest="test"\nout="forge-artifacts"\nsolc="0.8.15"\nffi=true\nast=true\n[profile.liteci]\noptimizer=false\n[profile.liteci.fuzz]\nruns=128\n[profile.liteci.invariant]\nruns=64\ndepth=32\n[profile.ciheavy]\noptimizer=false\n[profile.ciheavy.fuzz]\nruns=20000\ntimeout=300\n[profile.ciheavy.invariant]\nruns=128\ndepth=512\ntimeout=300\n')
            (contracts / 'src/unit/Fixture.sol').write_text('// SPDX-License-Identifier: MIT\npragma solidity 0.8.15; interface Vm {function skip(bool,string calldata) external;function ffi(string[] calldata) external returns(bytes memory);}contract Fixture {uint256 public value;uint256 public copied;function runtime(uint256 a) public returns(uint256){value=a;copied=a;return a;}}')
            (contracts / 'test/unit/Fixture.t.sol').write_text('// SPDX-License-Identifier: MIT\npragma solidity 0.8.15; import {Fixture,Vm} from "../../src/unit/Fixture.sol";contract Fixture_Runtime_Test {Vm constant vm=Vm(address(uint160(uint256(keccak256("hevm cheat code")))));function test_runtime_succeeds() public {string[] memory args=new string[](1);args[0]="./scripts/go-ffi/go-ffi";assert(abi.decode(vm.ffi(args),(uint256))==1);}function testFuzz_runtime_succeeds(uint256 a) public {assert(new Fixture().runtime(a)==a);}function test_runtime_disabled_succeeds() public {vm.skip(true,"fixture disabled");}}')
            (contracts / 'src/unit/InvariantFixture.sol').write_text('// SPDX-License-Identifier: MIT\npragma solidity 0.8.15; import {Fixture} from "./Fixture.sol";contract InvariantFixture is Fixture {}')
            (contracts / 'test/unit/InvariantFixture.t.sol').write_text('// SPDX-License-Identifier: MIT\npragma solidity 0.8.15; import {InvariantFixture} from "../../src/unit/InvariantFixture.sol";contract InvariantFixture_Runtime_Test {InvariantFixture fixture;function setUp() public {fixture=new InvariantFixture();}function targetContracts() public view returns(address[] memory targets){targets=new address[](1);targets[0]=address(fixture);}function invariant_runtime_succeeds() public view {assert(fixture.value()==fixture.copied());}}')

            def git(*args, directory=root):
                return subprocess.check_output(['git', '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', *args], cwd=directory, text=True, stderr=subprocess.DEVNULL).strip()
            module = Path(tmp) / 'module'
            module.mkdir()
            git('init', '-q', directory=module)
            (module / 'original.txt').write_text('original')
            git('add', '.', directory=module)
            git('commit', '-qm', 'module', directory=module)
            git('init', '-q')
            git('-c', 'protocol.file.allow=always', 'submodule', 'add', '-q', str(module), 'modules/original')
            git('add', '.')
            git('commit', '-qm', 'baseline')
            base = git('rev-parse', 'HEAD')
            origin = Path(tmp) / 'origin.git'
            git('init', '--bare', '-q', str(origin))
            git('remote', 'add', 'origin', str(origin))
            git('push', '-q', 'origin', base + ':refs/heads/develop')
            path = contracts / 'test/unit/Fixture.t.sol'
            path.write_text(path.read_text() + '\n// Changed fixture retains the original full file selection.\n')
            git('add', '.')
            git('commit', '-qm', 'changed test')
            sha = git('rev-parse', 'HEAD')
            bin_dir = Path(tmp) / 'bin'
            bin_dir.mkdir()
            split = bin_dir / 'circleci'
            split.write_text('#!/bin/sh\n[ "$*" = "tests split --split-by=timings" ] || exit 17\ncat\n')
            split.chmod(493)
            marker = Path(tmp) / 'fresh-marker'
            env = dict(os.environ, CI_BRANCH='pilot', CI_COMMIT_SHA=sha, CI_SUITE_FIXTURE_MARKER=str(marker), GOFLAGS='-mod=readonly', PATH=str(bin_dir) + os.pathsep + os.environ['PATH'])
            for name in ('RWX_VALUES', 'CI_SUITE_FIXTURE_FAIL'):
                env.pop(name, None)

            def invoke(mode, suite, provider, prepared=None):
                argv = ['python3', str(root / 'ops/ci/runtime/contract-suites.py'), mode, '--suite', suite, '--feature', 'main']
                if prepared:
                    argv += ['--prepared', str(prepared)]
                settings = dict(env, CI_CONTRACT_PROVIDER=provider, RWX_RUN_ID='f' * 32, RWX_TASK_ATTEMPT_NUMBER='1')
                if suite == 'modified' and provider == 'rwx' and not env.get('RWX_VALUES'):
                    settings['CI_CONTRACT_TARGET_SHA'] = env.get('CI_SUITE_FIXTURE_TARGET', base)
                child = subprocess.Popen(argv, cwd=root, env=settings, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                try:
                    stdout, stderr = child.communicate(timeout=180)
                except subprocess.TimeoutExpired:
                    child.terminate()
                    try:
                        child.communicate(timeout=30)
                    except subprocess.TimeoutExpired:
                        child.kill()
                        child.communicate(timeout=30)
                    raise
                return subprocess.CompletedProcess(argv, child.returncode, stdout, stderr)
            for suite in CS.SUITES:
                directories = {}
                for provider in ('circleci', 'rwx'):
                    if suite == 'standard' and provider == 'circleci':
                        git('submodule', 'deinit', '-f', 'modules/original')
                    if provider == 'rwx':
                        result = invoke('prepare', suite, provider)
                        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                        prepared = root / '.ci/contract-suites' / (suite + '-main') / 'prepare'
                        self.assertFalse(list(prepared.glob('*.xml')))
                        self.assertEqual(json.loads((prepared / 'compile-only.json').read_text())['tests'], 0)
                        if suite == 'modified':
                            # The branch advances after compilation, but the run's
                            # immutable baseline and selected test remain valid.
                            git('push', '-q', 'origin', sha + ':refs/heads/develop')
                            git('fetch', '-q', 'origin', '+refs/heads/develop:refs/remotes/origin/develop')
                            env['CI_SUITE_FIXTURE_TARGET'] = sha
                            mismatch = invoke('run', suite, provider, prepared)
                            self.assertNotEqual(mismatch.returncode, 0)
                            self.assertIn('Pinned contract target differs from compilation', mismatch.stderr)
                            self.assertFalse((root / '.ci/contract-suites/modified-main/run/tests.stage.json').exists())
                            env.pop('CI_SUITE_FIXTURE_TARGET')
                    else:
                        prepared = None
                    before = len(marker.read_text()) if marker.exists() else 0
                    unavailable = module.with_name('unavailable-module')
                    if provider == 'rwx':
                        module.rename(unavailable)
                    try:
                        result = invoke('run', suite, provider, prepared)
                        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    finally:
                        if provider == 'rwx':
                            unavailable.rename(module)
                    self.assertEqual(len(marker.read_text()), before + 1)
                    report = root / '.ci/contract-suites' / (suite + '-main') / 'run'
                    destination = Path(tmp) / (suite + '-' + provider)
                    shutil.copytree(report, destination)
                    directories[provider] = destination
                    coverage = json.loads((report / 'coverage.json').read_text())
                    self.assertEqual(coverage['outcomes'], {'pass': 3 if suite == 'standard' else 2, 'skip': 1})
                    self.assertEqual('invariant_runtime_succeeds()' in [row['name'] for row in coverage['cases']], suite == 'standard')
                    skipped = [row for row in coverage['cases'] if row['outcome'] == 'skip']
                    self.assertEqual(skipped[0]['skip_reason'], {'attributes': {'message': 'fixture disabled'}, 'text': ''})
                    self.assertFalse(any((line.startswith('-') for line in (report / 'submodules.txt').read_text().splitlines())))
                    self.assertIn('All contract test validations passed', (report / 'lint-test-names.log').read_text())
                self.verify_provider_reports(root, directories, sha, suite, tmp)
            prepared = root / '.ci/contract-suites/standard-main/prepare'
            settings_path, manifest_path = (prepared / 'settings.json', prepared / 'final.json')
            original_settings, original_manifest = (settings_path.read_bytes(), manifest_path.read_bytes())
            value = json.loads(original_settings)
            value['runtime_output_paths'] = ['go.sum']
            CS.UP.write(settings_path, value)
            value = json.loads(original_manifest)
            value['original_sha256']['settings.json'] = CS.UP.digest(settings_path)
            CS.UP.write(manifest_path, value)
            result = invoke('run', 'standard', 'rwx', prepared)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('Unexpected compiled tracked runtime output role', result.stderr)
            self.assertFalse((root / '.ci/contract-suites/standard-main/run/tests.stage.json').exists())
            settings_path.write_bytes(original_settings)
            manifest_path.write_bytes(original_manifest)
            binary = contracts / 'scripts/go-ffi/go-ffi'
            binary.write_bytes(binary.read_bytes() + b'corrupt')
            result = invoke('run', 'standard', 'rwx', prepared)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('corrupt', result.stderr)
            self.assertFalse((root / '.ci/contract-suites/standard-main/run/tests.stage.json').exists())
            result = invoke('prepare', 'standard', 'rwx')
            self.assertEqual(result.returncode, 0, result.stderr)
            env['CI_SUITE_FIXTURE_FAIL'] = '1'
            result = invoke('run', 'standard', 'rwx', prepared)
            self.assertNotEqual(result.returncode, 0)
            report = root / '.ci/contract-suites/standard-main/run'
            self.assertIn('<failure', (report / 'original.junit.xml').read_text())
            self.assertTrue((report / 'rerun.stage.json').exists())
            self.assertNotEqual(json.loads((report / 'final.json').read_text())['exit_code'], 0)
            env.pop('CI_SUITE_FIXTURE_FAIL')
            git('push', '-q', 'origin', sha + ':refs/heads/develop')
            values = Path(tmp) / 'values'
            values.mkdir()
            env['RWX_VALUES'] = str(values)
            result = invoke('prepare', 'modified', 'rwx')
            self.assertEqual(result.returncode, 0, result.stderr)
            empty = root / '.ci/contract-suites/modified-main/prepare'
            self.assertEqual(json.loads((empty / 'coverage.json').read_text())['tests'], 0)
            self.assertEqual((values / 'eligible').read_text(), 'false\n')
            self.assertFalse(list(empty.glob('*.xml')))


if __name__ == '__main__': unittest.main()


@unittest.skipUnless(os.environ.get('RWX_LIVE_CONTRACT_SUITE_FIXTURE') == '1', 'Opt-in original Forge/Go fixture')
class LiveSuiteTests(_LiveSuiteTestsFixtures, unittest.TestCase):
    def test_actual_find_changed_files_compilation_fresh_ffi_failures_and_parity(self):
        self.exercise_original_reports()
