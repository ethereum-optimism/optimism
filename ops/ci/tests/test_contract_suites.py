#!/usr/bin/env python3
"""Exercise the suite runner's file selection, job environment, recipes and reruns."""
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import unittest
import xml.etree.ElementTree as ET

from test_contract_upgrades import RUNTIME, drive, project

SPEC = importlib.util.spec_from_file_location('suites', RUNTIME / 'contract-suites.py')
CS = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(CS)
FIND = ['find', 'test', '-name', '*.t.sol']


def listing(text):
    """Return a stage effect that makes the stubbed test_list command print text."""
    def effect(directory, name):
        if name == 'files': (directory / 'files.log').write_text(text)
    return effect


def git(root, *arguments):
    subprocess.check_call(['git', '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', *arguments],
                          cwd=root, stdout=subprocess.DEVNULL)


class SuiteTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(); self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)

    def test_run_gives_just_test_the_file_list_then_lints_the_test_names(self):
        status, calls = drive(CS, self.root, ['run', '--suite', 'standard', '--feature', 'CUSTOM_GAS_TOKEN'],
                              environment={'CI_BRANCH': 'pilot'}, effect=listing('test/L1/A.t.sol\ntest/L2/B.t.sol\n'))
        self.assertEqual(status, 0)
        self.assertEqual([call[:2] for call in calls], [('files', FIND), ('tests', ['just', 'test']),
                                                        ('lint-test-names', ['just', 'lint-forge-tests-check-no-build'])])
        # Like the CircleCI step, the recipe gets the files in TEST_FILES and a directory for its per-pass JUnit reports.
        expected = {'CI_BRANCH': 'pilot', 'CI': 'true', 'FOUNDRY_PROFILE': 'liteci', 'SYS_FEATURE__CUSTOM_GAS_TOKEN': 'true',
                    'TEST_FILES': 'test/L1/A.t.sol\ntest/L2/B.t.sol',
                    'JUNIT_TEST_DIR': str(self.root / '.ci/contract-suites/standard-CUSTOM_GAS_TOKEN/run')}
        self.assertEqual(calls[1][2], expected)

    def test_failed_tests_are_rerun_for_traces_without_junit_and_skip_the_lint(self):
        arguments = ['run', '--suite', 'standard', '--feature', 'main']
        status, calls = drive(CS, self.root, arguments, [0, 1, 1], {'CI_BRANCH': 'pilot'}, listing('test/L1/A.t.sol\n'))
        self.assertEqual(status, 1)
        self.assertEqual([call[:2] for call in calls], [('files', FIND), ('tests', ['just', 'test']), ('rerun', ['just', 'test-rerun'])])
        self.assertEqual(['JUNIT_TEST_DIR' in call[2] for call in calls], [False, True, False])
        # A failed lint fails the task after passing tests. A failed test_list command stops it at once.
        for statuses, expected in [([0, 0, 5], (5, ['files', 'tests', 'lint-test-names'])), ([2], (2, ['files']))]:
            status, calls = drive(CS, self.root, arguments, statuses, {'CI_BRANCH': 'pilot'}, listing('test/L1/A.t.sol\n'))
            self.assertEqual((status, [call[0] for call in calls]), expected)

    def test_prepare_builds_with_the_profile_of_the_circleci_job(self):
        for suite, branch, profile in [('standard', 'pilot', 'liteci'), ('standard', 'develop', 'ci'),
                                       ('modified', 'pilot', 'ciheavy'), ('modified', 'develop', 'ciheavy')]:
            with self.subTest(suite=suite, branch=branch):
                status, calls = drive(CS, self.root, ['prepare', '--suite', suite, '--feature', 'main'],
                                      environment={'CI_BRANCH': branch}, effect=listing('test/L1/A.t.sol\n'))
                self.assertEqual((status, [call[1] for call in calls]),
                                 (0, [CS.FILE_COMMANDS[suite], ['just', 'build-go-ffi'], ['forge', 'build']]))
                self.assertEqual({call[2]['FOUNDRY_PROFILE'] for call in calls}, {profile})

    def test_empty_file_list_stops_with_success_before_any_build_or_test(self):
        # With no TEST_FILES, `just test` would run every test with the heavy profile.
        for mode in ('prepare', 'run'):
            with self.subTest(mode=mode):
                status, calls = drive(CS, self.root, [mode, '--suite', 'modified', '--feature', 'main'],
                                      environment={'CI_BRANCH': 'pilot'}, effect=listing('\n'))
                self.assertEqual((status, [call[0] for call in calls]), (0, ['files']))

    def test_modified_file_list_is_circleci_heavy_fuzz_test_list(self):
        # contracts-bedrock-tests-heavy-fuzz-modified owns the changed-file selection.
        config = (CS.UP.ROOT / '.circleci/continue/main.yml').read_text()
        match = re.search(r"^( *)test_list: (git diff .*\n(?:\1  .*\n)*)", config, re.M)
        self.assertIsNotNone(match)
        self.assertEqual(' '.join(match[2].split()), CS.FILE_COMMANDS['modified'][-1])

    def test_real_diff_against_the_fetched_develop_selects_only_changed_test_files(self):
        # git is real. just and forge are stubs that record their arguments and TEST_FILES.
        contracts, invoke = project(self.root, '', {'test/unit/Fixture.t.sol': '// fixture\n', 'test/unit/Other.t.sol': '// other\n',
                                                    'test/unit/Support.sol': '// not a test\n'})
        tools = self.root / 'bin'; tools.mkdir(); log = self.root / 'calls.log'
        for name in ('just', 'forge'):
            (tools / name).write_text('#!/bin/sh\necho "' + name + ' $*|$TEST_FILES" >> "$CALL_LOG"\n'); (tools / name).chmod(0o755)
        git(self.root, 'init', '-q'); git(self.root, 'add', '.'); git(self.root, 'commit', '-qm', 'base')
        git(self.root, 'update-ref', 'refs/remotes/origin/develop', 'HEAD')
        def run(mode):
            result = invoke('contract-suites.py', mode, '--suite', 'modified', '--feature', 'main',
                            PATH=str(tools) + os.pathsep + os.environ['PATH'], CALL_LOG=str(log))
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            return result.stdout
        for mode in ('prepare', 'run'): self.assertIn('No test files to run.', run(mode))
        self.assertFalse(log.exists())
        for name in ('test/unit/Fixture.t.sol', 'test/unit/Support.sol'): (contracts / name).write_text('// changed\n')
        git(self.root, 'commit', '-qam', 'change')
        run('prepare'); run('run')
        self.assertEqual(log.read_text().splitlines(), ['just build-go-ffi|', 'forge build|', 'just test|test/unit/Fixture.t.sol',
                                                         'just lint-forge-tests-check-no-build|test/unit/Fixture.t.sol'])


@unittest.skipUnless(os.environ.get('RWX_LIVE_CONTRACT_SUITE_FIXTURE') == '1', 'Opt-in fixture: needs the pinned just, go and forge, and the Go modules')
class LiveSuiteTests(unittest.TestCase):
    def test_real_recipes_run_both_evm_passes_lint_and_rerun_a_failure(self):
        temporary = tempfile.TemporaryDirectory(); self.addCleanup(temporary.cleanup); root = Path(temporary.name)
        contracts, invoke = project(root, '[profile.default]\nsrc="src"\ntest="test"\nout="forge-artifacts"\nsolc="0.8.15"\nffi=true\nast=true\n'
            '[profile.liteci]\noptimizer=false\n[profile.ciheavy]\noptimizer=false\n', {
            'src/unit/Fixture.sol': '// SPDX-License-Identifier: MIT\npragma solidity 0.8.15; interface Vm {function skip(bool,string calldata) external;function ffi(string[] calldata) external returns(bytes memory);}contract Fixture {uint256 public value;uint256 public copied;function runtime(uint256 a) public returns(uint256){value=a;copied=a;return a;}}',
            'test/unit/Fixture.t.sol': '// SPDX-License-Identifier: MIT\npragma solidity 0.8.15; import {Fixture,Vm} from "../../src/unit/Fixture.sol";contract Fixture_Runtime_Test {Vm constant vm=Vm(address(uint160(uint256(keccak256("hevm cheat code")))));function test_runtime_succeeds() public {string[] memory args=new string[](1);args[0]="./scripts/go-ffi/go-ffi";assert(abi.decode(vm.ffi(args),(uint256))==1);}function testFuzz_runtime_succeeds(uint256 a) public {assert(new Fixture().runtime(a)==a);}function test_runtime_disabled_succeeds() public {vm.skip(true,"fixture disabled");}}',
            'src/L2/InvariantFixture.sol': '// SPDX-License-Identifier: MIT\npragma solidity 0.8.15; import {Fixture} from "../unit/Fixture.sol";contract InvariantFixture is Fixture {}',
            # A default-EVM test makes the standard suite run both EVM passes.
            'test/L2/InvariantFixture.t.sol': '// SPDX-License-Identifier: MIT\npragma solidity 0.8.15; import {InvariantFixture} from "../../src/L2/InvariantFixture.sol";contract InvariantFixture_Runtime_Test {InvariantFixture fixture;function setUp() public {fixture=new InvariantFixture();}function targetContracts() public view returns(address[] memory targets){targets=new address[](1);targets[0]=address(fixture);}function invariant_runtime_succeeds() public view {assert(fixture.value()==fixture.copied());}}'})
        # The lint recipe runs the real test-name checker, so the project needs its Go packages.
        real = CS.UP.ROOT
        text = subprocess.check_output(['go', 'list', '-deps', '-json', './packages/contracts-bedrock/scripts/checks/test-validation'], cwd=real, text=True)
        decoder, offset = json.JSONDecoder(), 0
        while offset < len(text):
            if text[offset].isspace(): offset += 1; continue
            row, offset = decoder.raw_decode(text, offset)
            if not row.get('Module', {}).get('Main'): continue
            source = Path(row['Dir'])
            for name in [f.name for f in source.glob('*.go')] + row.get('EmbedFiles', []):
                target = root / source.relative_to(real) / name; target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(source / name, target)
        for name in ('go.mod', 'go.sum', 'packages/contracts-bedrock/scripts/checks/test-validation/exclusions.toml'):
            shutil.copy2(real / name, root / name)
        git(root, 'init', '-q'); git(root, 'add', '.'); git(root, 'commit', '-qm', 'base')
        git(root, 'update-ref', 'refs/remotes/origin/develop', 'HEAD')
        path = contracts / 'test/unit/Fixture.t.sol'; path.write_text(path.read_text() + '\n// Changed test file.\n')
        git(root, 'commit', '-qam', 'changed test')
        amsterdam = {'test_runtime_succeeds()', 'testFuzz_runtime_succeeds(uint256)', 'test_runtime_disabled_succeeds()'}
        marker = root / 'ffi-calls'
        for suite, passes in [('standard', {'results-amsterdam.xml': amsterdam, 'results-l2.xml': {'invariant_runtime_succeeds()'}}),
                              ('modified', {'results-amsterdam.xml': amsterdam})]:
            with self.subTest(suite=suite):
                before = len(marker.read_text()) if marker.exists() else 0
                for mode in ('prepare', 'run'):
                    result = invoke('contract-suites.py', mode, '--suite', suite, '--feature', 'main', GOFLAGS='-mod=readonly')
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                report = root / '.ci/contract-suites' / (suite + '-main') / 'run'
                reports = {p.name: {case.get('name') for case in ET.parse(p).iter('testcase')} for p in report.glob('results-*.xml')}
                self.assertEqual(reports, passes)
                self.assertEqual(len(marker.read_text()), before + 1)
                self.assertIn('All contract test validations passed', (report / 'lint-test-names.log').read_text())
        result = invoke('contract-suites.py', 'run', '--suite', 'standard', '--feature', 'main', GOFLAGS='-mod=readonly', CI_FIXTURE_FAIL='1')
        self.assertNotEqual(result.returncode, 0)
        report = root / '.ci/contract-suites/standard-main/run'
        self.assertIn('<failure', (report / 'results-amsterdam.xml').read_text())
        self.assertEqual(json.loads((report / 'rerun.stage.json').read_text())['argv'], ['just', 'test-rerun'])
        self.assertFalse((report / 'lint-test-names.stage.json').exists())


if __name__ == '__main__': unittest.main()
