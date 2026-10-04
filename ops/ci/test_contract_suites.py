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

SPEC = importlib.util.spec_from_file_location('suites', Path(__file__).with_name('contract-suites.py'))
CS = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(CS)


class SuiteTests(unittest.TestCase):
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

    def test_effective_heavy_settings_reject_reduced_runs_depth_or_timeout(self):
        config = {'fuzz': {'runs': 20000, 'timeout': 300}, 'invariant': {'runs': 128, 'depth': 512, 'timeout': 300}}
        CS.validate_config(config, 'modified')
        for key, field in [('fuzz', 'runs'), ('fuzz', 'timeout'), ('invariant', 'runs'), ('invariant', 'depth'), ('invariant', 'timeout')]:
            value = json.loads(json.dumps(config)); value[key][field] = 1
            with self.subTest(key=key, field=field), self.assertRaises(ValueError): CS.validate_config(value, 'modified')
        config['match_test'] = 'partial'
        with self.assertRaises(ValueError): CS.validate_config(config, 'modified')


@unittest.skipUnless(os.environ.get('RWX_LIVE_CONTRACT_SUITE_FIXTURE') == '1', 'Opt-in pinned Forge and production Go-validator fixture')
class LiveSuiteTests(unittest.TestCase):
    def test_actual_find_changed_files_compilation_fresh_ffi_failures_and_parity(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'repo'; root.mkdir(); contracts = root / 'packages/contracts-bedrock'
            (contracts / 'test/unit').mkdir(parents=True); (contracts / 'src/unit').mkdir(parents=True); (contracts / 'scripts/go-ffi').mkdir(parents=True)
            (root / 'ops/ci').mkdir(parents=True)
            for source in Path(CS.__file__).parent.glob('*.py'):
                if not source.name.startswith('test_'): shutil.copy2(source, root / 'ops/ci' / source.name)
            # Copy the real validator's in-repository Go dependencies. External
            # module versions and checksums are the production lockfiles.
            text = subprocess.check_output(['go', 'list', '-deps', '-json', './packages/contracts-bedrock/scripts/checks/test-validation'], cwd=CS.ROOT, text=True)
            decoder, offset = json.JSONDecoder(), 0
            while offset < len(text):
                if text[offset].isspace(): offset += 1; continue
                row, offset = decoder.raw_decode(text, offset)
                if not row.get('Module', {}).get('Main'): continue
                src = Path(row['Dir']); dst = root / src.relative_to(CS.ROOT); dst.mkdir(parents=True, exist_ok=True)
                for f in src.glob('*.go'): shutil.copy2(f, dst / f.name)
                for name in row.get('EmbedFiles', []):
                    target = dst / name; target.parent.mkdir(parents=True, exist_ok=True); shutil.copy2(src / name, target)
            for name in ('go.mod', 'go.sum'): shutil.copy2(CS.ROOT / name, root / name)
            shutil.copy2(CS.CONTRACTS / 'scripts/checks/test-validation/exclusions.toml', contracts / 'scripts/checks/test-validation/exclusions.toml')
            guard = contracts / 'scripts/checks/check-junit-tests-ran.sh'
            shutil.copy2(CS.CONTRACTS / 'scripts/checks/check-junit-tests-ran.sh', guard)
            original = (CS.CONTRACTS / 'justfile').read_text(); recipes = []
            for name in ('build-go-ffi', 'test-rerun', 'lint-forge-tests-check-no-build'):
                start = original.index('\n' + name + ':'); end = original.index('\n\n', start + 1)
                recipes.append(original[start + 1:end])
            (contracts / 'justfile').write_text('\n\n'.join(recipes) + '\n')
            (contracts / 'scripts/go-ffi/main.go').write_text('package main\nimport("fmt";"os")\nfunc main(){'
                'f,e:=os.OpenFile(os.Getenv("CI_SUITE_FIXTURE_MARKER"),os.O_APPEND|os.O_CREATE|os.O_WRONLY,0600);'
                'if e!=nil{panic(e)};defer f.Close();f.WriteString("x");v:=1;'
                'if os.Getenv("CI_SUITE_FIXTURE_FAIL")=="1"{v=0};fmt.Printf("0x%064x",v)}\n')
            (contracts / 'foundry.toml').write_text('[profile.default]\nsrc="src"\ntest="test"\nout="forge-artifacts"\n'
                'solc="0.8.15"\nffi=true\nast=true\n[profile.liteci]\noptimizer=false\n[profile.liteci.fuzz]\nruns=128\n'
                '[profile.liteci.invariant]\nruns=64\ndepth=32\n[profile.ciheavy]\noptimizer=false\n'
                '[profile.ciheavy.fuzz]\nruns=20000\ntimeout=300\n[profile.ciheavy.invariant]\nruns=128\ndepth=512\ntimeout=300\n')
            (contracts / 'src/unit/Fixture.sol').write_text('// SPDX-License-Identifier: MIT\npragma solidity 0.8.15; '
                'interface Vm {function skip(bool) external;function ffi(string[] calldata) external returns(bytes memory);}'
                'contract Fixture {function runtime(uint256 a) public pure returns(uint256){return a;}}')
            (contracts / 'test/unit/Fixture.t.sol').write_text('// SPDX-License-Identifier: MIT\npragma solidity 0.8.15; '
                'import {Fixture,Vm} from "../../src/unit/Fixture.sol";contract Fixture_Runtime_Test {'
                'Vm constant vm=Vm(address(uint160(uint256(keccak256("hevm cheat code")))));'
                'function test_runtime_succeeds() public {string[] memory args=new string[](1);'
                'args[0]="./scripts/go-ffi/go-ffi";assert(abi.decode(vm.ffi(args),(uint256))==1);}'
                'function testFuzz_runtime_succeeds(uint256 a) public {assert(new Fixture().runtime(a)==a);}'
                'function test_runtime_disabled_succeeds() public {vm.skip(true);}}')
            def git(*args, directory=root):
                return subprocess.check_output(['git', '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', *args],
                                               cwd=directory, text=True, stderr=subprocess.DEVNULL).strip()
            module = Path(tmp) / 'module'; module.mkdir(); git('init', '-q', directory=module)
            (module / 'original.txt').write_text('original'); git('add', '.', directory=module); git('commit', '-qm', 'module', directory=module)
            git('init', '-q'); git('-c', 'protocol.file.allow=always', 'submodule', 'add', '-q', str(module), 'modules/original')
            git('add', '.'); git('commit', '-qm', 'baseline'); base = git('rev-parse', 'HEAD')
            origin = Path(tmp) / 'origin.git'; git('init', '--bare', '-q', str(origin)); git('remote', 'add', 'origin', str(origin))
            git('push', '-q', 'origin', base + ':refs/heads/develop')
            path = contracts / 'test/unit/Fixture.t.sol'; path.write_text(path.read_text() + '\n// Changed fixture retains the original full file selection.\n')
            git('add', '.'); git('commit', '-qm', 'changed test'); sha = git('rev-parse', 'HEAD')
            # An identity partition adapter tests stdin/CLI wiring outside the
            # Circle agent. Hosted evidence must use the actual Circle splitter.
            bin_dir = Path(tmp) / 'bin'; bin_dir.mkdir(); split = bin_dir / 'circleci'
            split.write_text('#!/bin/sh\n[ "$*" = "tests split --split-by=timings" ] || exit 17\ncat\n'); split.chmod(0o755)
            marker = Path(tmp) / 'fresh-marker'
            env = dict(os.environ, CI_BRANCH='pilot', CI_COMMIT_SHA=sha, CI_SUITE_FIXTURE_MARKER=str(marker),
                       GOFLAGS='-mod=readonly', PATH=str(bin_dir) + os.pathsep + os.environ['PATH'])
            for name in ('RWX_VALUES', 'CI_SUITE_FIXTURE_FAIL'): env.pop(name, None)
            def invoke(mode, suite, provider, prepared=None):
                argv = ['python3', str(root / 'ops/ci/contract-suites.py'), mode, '--suite', suite, '--feature', 'main']
                if prepared: argv += ['--prepared', str(prepared)]
                settings = dict(env, CI_CONTRACT_PROVIDER=provider, RWX_RUN_ID='f'*32, RWX_TASK_ATTEMPT_NUMBER='1')
                return subprocess.run(argv, cwd=root, env=settings, text=True, capture_output=True, timeout=180)
            for suite in CS.SUITES:
                directories = {}
                for provider in ('circleci', 'rwx'):
                    if provider == 'rwx':
                        result = invoke('prepare', suite, provider); self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                        prepared = root / '.ci/contract-suites' / (suite + '-main') / 'prepare'
                        self.assertFalse(list(prepared.glob('*.xml')))
                        self.assertEqual(json.loads((prepared / 'compile-only.json').read_text())['tests'], 0)
                    else: prepared = None
                    before = len(marker.read_text()) if marker.exists() else 0
                    result = invoke('run', suite, provider, prepared); self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    self.assertEqual(len(marker.read_text()), before + 1)
                    report = root / '.ci/contract-suites' / (suite + '-main') / 'run'
                    destination = Path(tmp) / (suite + '-' + provider); shutil.copytree(report, destination); directories[provider] = destination
                    coverage = json.loads((report / 'coverage.json').read_text()); self.assertEqual(coverage['outcomes'], {'pass': 2, 'skip': 1})
                    self.assertIn('All contract test validations passed', (report / 'lint-test-names.log').read_text())
                result = subprocess.run(['python3', str(root / 'ops/ci/compare-contract-suites.py'), '--circle', str(directories['circleci']),
                    '--rwx', str(directories['rwx']), '--sha', sha, '--suite', suite, '--feature', 'main', '--output', str(Path(tmp) / (suite + '.json'))],
                    text=True, capture_output=True)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            prepared = root / '.ci/contract-suites/standard-main/prepare'
            binary = contracts / 'scripts/go-ffi/go-ffi'; binary.write_bytes(binary.read_bytes() + b'corrupt')
            result = invoke('run', 'standard', 'rwx', prepared)
            self.assertNotEqual(result.returncode, 0); self.assertIn('corrupt', result.stderr)
            self.assertFalse((root / '.ci/contract-suites/standard-main/run/tests.stage.json').exists())
            result = invoke('prepare', 'standard', 'rwx'); self.assertEqual(result.returncode, 0, result.stderr)
            env['CI_SUITE_FIXTURE_FAIL'] = '1'
            result = invoke('run', 'standard', 'rwx', prepared); self.assertNotEqual(result.returncode, 0)
            report = root / '.ci/contract-suites/standard-main/run'
            self.assertIn('<failure', (report / 'original.junit.xml').read_text())
            self.assertTrue((report / 'rerun.stage.json').exists())
            self.assertNotEqual(json.loads((report / 'final.json').read_text())['exit_code'], 0)
            env.pop('CI_SUITE_FIXTURE_FAIL'); git('push', '-q', 'origin', sha + ':refs/heads/develop')
            values = Path(tmp) / 'values'; values.mkdir(); env['RWX_VALUES'] = str(values)
            result = invoke('prepare', 'modified', 'rwx'); self.assertEqual(result.returncode, 0, result.stderr)
            empty = root / '.ci/contract-suites/modified-main/prepare'
            self.assertEqual(json.loads((empty / 'coverage.json').read_text())['tests'], 0)
            self.assertEqual((values / 'eligible').read_text(), 'false\n')
            self.assertFalse(list(empty.glob('*.xml')))


if __name__ == '__main__': unittest.main()
