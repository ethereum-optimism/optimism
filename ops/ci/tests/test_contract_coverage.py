#!/usr/bin/env python3
"""Exercise original coverage reports, both Just passes and fresh failure evidence."""
import http.server
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import threading
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('coverage', Path(__file__).resolve().parents[1] / 'runtime' / 'contract-coverage.py')
C = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(C)


class ReportTests(unittest.TestCase):
    def test_replay_controls_ffi_entropy_and_unseeded_runs_clear_it(self):
        with patch.dict(os.environ, {'CI_BRANCH':'pilot', 'CI_CONTRACT_COVERAGE_REPLAY':'true',
                                    C.FFI_REPLAY_INPUT:'inherited'}, clear=True), patch.object(C.UP,'revision',return_value='a'*40):
            _, seed=C.configure('main')
            self.assertEqual(os.environ[C.FFI_REPLAY_INPUT],seed)
            os.environ['CI_CONTRACT_COVERAGE_REPLAY']='false'
            self.assertIsNone(C.configure('main')[1])
            self.assertNotIn(C.FFI_REPLAY_INPUT,os.environ)

    def test_merged_campaign_retains_every_predicate_and_original_anchor(self):
        log = 'Ran 1 test for test/A.t.sol:A\n[PASS]\nA invariants:\n[PASS] invariant_value\n[PASS] invariant_valuePositive\n' \
              ' A invariants (runs: 1, calls: 1, reverts: 0)\nRan 1 test suite in 1ms: 1 tests passed, 0 failed, 0 skipped (1 total tests)\n'
        row = {'suite': 'test/A.t.sol:A', 'test': 'invariant_value()', 'status': 'success', 'kind': 'invariant', 'covered': []}
        original = {'version': 1, 'tests': [row]}
        evidence=C.original_evidence(log,original)
        self.assertEqual([row['name'] for row in evidence['cases']],['invariant_value()','invariant_valuePositive()'])
        self.assertEqual(evidence['engine_summary']['total'],1)
        self.assertEqual(evidence['invariant_campaigns'][0]['anchor'],'invariant_value()')
        row['kind'] = 'unit'
        with self.assertRaisesRegex(ValueError, 'invariant attribution anchor'): C.original_cases(log, original)
        row.update(kind='invariant', test='invariant_value(uint256)')
        with self.assertRaisesRegex(ValueError, 'invariant attribution anchor'): C.original_cases(log, original)
        row['test']='invariant_value()'
        for changed in (log.replace(' A invariants (runs: 1, calls: 1, reverts: 0)\n',''),
                        log.replace('[PASS] invariant_valuePositive\n',''),
                        log.replace('[PASS] invariant_valuePositive\n','[PASS] invariant_valuePositive\n[PASS] invariant_valuePositive\n'),
                        log.replace('(1 total tests)','(2 total tests)')):
            with self.subTest(log=changed),self.assertRaises(ValueError):C.original_cases(changed,original)

    def test_failed_campaign_with_passing_predicates_keeps_aggregate_failure(self):
        log='Ran 1 test for test/A.t.sol:A\nA invariants: 0/2 invariants broken\n[PASS] invariant_a\n[PASS] invariant_b\n' \
            ' A invariants (runs: 1, calls: 1, reverts: 0)\nRan 1 test suite in 1ms: 0 tests passed, 1 failed, 0 skipped (1 total tests)\n'
        original={'version':1,'tests':[{'suite':'test/A.t.sol:A','test':'invariant_a()','status':'failure','kind':'invariant','covered':[]}]}
        evidence=C.original_evidence(log,original)
        with tempfile.TemporaryDirectory() as tmp:
            p=Path(tmp)/'derived.xml';C.derived_junit(p,evidence['cases'],'ordinary',evidence['invariant_campaigns'])
            self.assertEqual(len(list(C.ET.parse(p).iter('failure'))),1)

    def test_parallel_campaign_completion_order_preserves_equal_evidence(self):
        groups={name:'Ran 1 test for test/'+name+'.t.sol:'+name+'\n[PASS]\n'+name+' invariants:\n'
                '[PASS] invariant_a\n[PASS] invariant_b\n '+name+' invariants (runs: 1, calls: 1, reverts: 0)\n'
                for name in ('A','B')}
        original={'version':1,'tests':[{'suite':'test/'+name+'.t.sol:'+name,'test':'invariant_a()',
                  'status':'success','kind':'invariant','covered':[]} for name in groups]}
        summary='Ran 2 test suites in 1ms: 2 tests passed, 0 failed, 0 skipped (2 total tests)\n'
        self.assertEqual(C.original_evidence(groups['A']+groups['B']+summary,original),
                         C.original_evidence(groups['B']+groups['A']+summary,original))

    def test_verdicts_bind_to_attribution_without_failure_summary_duplicates(self):
        log = 'Ran 3 tests for test/A.t.sol:A\n[PASS] test_a() (gas: 12)\n[SKIP: original guard] test_b() (gas: 0)\n' \
              '[FAIL: original assertion] test_c() (gas: 34)\nRan 1 test suite in 1ms: 1 tests passed, 1 failed, 1 skipped (3 total tests)\n' \
              'Failing tests:\n[FAIL: original assertion] test_c() (gas: 34)\n'
        original = {'version': 1, 'tests': [{'suite': 'test/A.t.sol:A', 'test': 'test_' + name + '()', 'status': status, 'kind': 'unit', 'covered': []}
                    for name, status in [('a', 'success'), ('b', 'skipped'), ('c', 'failure')]]}
        cases = C.original_cases(log, original)
        self.assertEqual([r['name'] for r in cases], ['test_a()', 'test_b()', 'test_c()'])
        self.assertEqual([r['outcome'] for r in cases], ['pass', 'skip', 'fail'])
        self.assertEqual(cases[1]['reason'], 'original guard')
        for text in (log.split('Ran 1 test suite')[0], log.replace('[PASS] test_a() (gas: 12)', '[PASS] test_a() (gas: 12)\n[PASS] test_a() (gas: 12)')):
            with self.assertRaises(ValueError): C.original_cases(text, original)
        original['tests'][0]['status'] = 'failure'
        with self.assertRaisesRegex(ValueError, 'disagree'): C.original_cases(log, original)

    def test_complete_lcov_preserves_every_hit_and_rejects_truncated_or_outside_sources(self):
        text = 'TN:\nSF:src/A.sol\nDA:6,4\nFN:6,A.set\nFNDA:4,A.set\nFNF:1\nFNH:1\nLF:1\nLH:1\nBRF:0\nBRH:0\nend_of_record\n'
        self.assertIn(('DA', '6,4'), C.lcov(text)[0]['entries'])
        for bad in (text.replace('end_of_record\n', ''), text.replace('LH:1', 'LH:2'), text.replace('src/A.sol', '../A.sol'),
                    text.replace('BRF:0\n', ''), text.replace('FNF:1', 'FNF:invalid')):
            with self.subTest(bad=bad), self.assertRaises(ValueError): C.lcov(bad)

    def test_effective_profile_filters_and_seed_are_verified(self):
        value = {'fuzz': {'runs': 1, 'seed': '0x02'}, 'invariant': {'runs': 1, 'depth': 1},
                 'optimizer': False, 'threads': 16, 'compilation_restrictions': []}
        C.validate_config(value, '0x02')
        for key, changed in [('threads', 2), ('optimizer', True), ('match_test', 'partial')]:
            wrong = dict(value, **{key: changed})
            with self.subTest(key=key), self.assertRaises(ValueError): C.validate_config(wrong, '0x02')
        with self.assertRaises(ValueError): C.validate_config(value, None)


class _LiveCoverageTestsFixtures:

    def verify_provider_reports(self, root, circle, first, sha, tmp):
        pass

    def exercise_original_reports(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'repo'
            contracts = root / 'packages/contracts-bedrock'
            (contracts / 'test/L1').mkdir(parents=True)
            (contracts / 'src').mkdir()
            (contracts / 'scripts/go-ffi').mkdir(parents=True)
            (root / 'ops/ci/runtime').mkdir(parents=True)
            for source in Path(C.__file__).parent.glob('*.py'):
                if not source.name.startswith('test_'):
                    shutil.copy2(source, root / 'ops/ci/runtime' / source.name)
            (root / 'go.mod').write_text('module example.invalid/coverage-fixture\ngo 1.26.5\n')
            (contracts / 'scripts/go-ffi/main.go').write_text('package main\nimport("fmt";"os")\nfunc main(){f,e:=os.OpenFile(os.Getenv("CI_COVERAGE_FIXTURE_MARKER"),os.O_APPEND|os.O_CREATE|os.O_WRONLY,0600);if e!=nil{panic(e)};defer f.Close();f.WriteString("x");v:=1;if os.Getenv("CI_COVERAGE_FIXTURE_FAIL")=="1"{v=0};fmt.Printf("0x%064x",v)}\n')
            original = (C.CONTRACTS / 'justfile').read_text()
            recipes = []
            for name in ('build-go-ffi', 'build-source', 'coverage-lcov', 'coverage-upgrade', 'coverage-lcov-upgrade', 'coverage-lcov-all', 'prepare-upgrade-env', 'test-rerun'):
                match = C.re.search('^' + C.re.escape(name) + '(?:[ :*].*)?:.*$', original, C.re.M)
                if match is None:
                    raise AssertionError('Missing production Just recipe ' + name)
                end = original.index('\n\n', match.start())
                recipes.append(original[match.start():end])
            (contracts / 'justfile').write_text('\n\n'.join(recipes) + '\n\nprint-pinned-block-number:\n  @echo 42\n')
            (contracts / 'foundry.toml').write_text('[profile.default]\nsrc="src"\ntest="test"\nout="forge-artifacts"\nsolc="0.8.15"\nffi=true\nast=true\noptimizer=true\nextra_output=["devdoc","userdoc","metadata","storageLayout"]\n[profile.cicoverage]\noptimizer=false\nthreads=16\ncompilation_restrictions=[]\n[profile.cicoverage.fuzz]\nruns=1\n[profile.cicoverage.invariant]\nruns=1\ndepth=1\n')
            (contracts / 'src/Fixture.sol').write_text('// SPDX-License-Identifier: MIT\npragma solidity 0.8.15;contract Fixture {function value(uint256 a) external pure returns(uint256){return a==3?1:2;}}interface Vm {function skip(bool,string calldata) external;function ffi(string[] calldata) external returns(bytes memory);function envOr(string calldata,bool) external view returns(bool);function envUint(string calldata) external view returns(uint256);}')
            (contracts / 'test/L1/Fixture.t.sol').write_text('// SPDX-License-Identifier: MIT\npragma solidity 0.8.15;import {Fixture,Vm} from "../../src/Fixture.sol";contract FixtureTest {Vm constant vm=Vm(address(uint160(uint256(keccak256("hevm cheat code")))));function test_runtime() public {string[] memory a=new string[](1);a[0]="./scripts/go-ffi/go-ffi";assert(abi.decode(vm.ffi(a),(uint256))==1);}function testFuzz_value(uint256 a) public {assert(new Fixture().value(a)>0);}function test_skip() public {vm.skip(true,"original fixture exclusion");}}contract OPContractsManagerFixture_Upgrade_Test {Vm constant vm=Vm(address(uint160(uint256(keccak256("hevm cheat code")))));function setUp() public {vm.skip(!vm.envOr("FORK_TEST",false),"upgrade requires fork");}function test_upgrade() public {assert(vm.envUint("FORK_BLOCK_NUMBER")==42);assert(vm.envUint("FOUNDRY_FORK_RETRIES")==10);assert(vm.envUint("FOUNDRY_FORK_RETRY_BACKOFF")==1000);string[] memory a=new string[](1);a[0]="./scripts/go-ffi/go-ffi";assert(abi.decode(vm.ffi(a),(uint256))==1);}}abstract contract DeclaredTest {function test_abstract() public pure {assert(true);}}contract InvariantTarget {uint256 public value=1;function handler(uint256) external {value=1;}}contract InvariantFixtureTest {InvariantTarget target;function setUp() public {target=new InvariantTarget();}function targetContracts() external view returns(address[] memory result){result=new address[](1);result[0]=address(target);}Vm constant vm=Vm(address(uint160(uint256(keccak256("hevm cheat code")))));function invariant_value() public view {assert(target.value()==1&&!vm.envOr("CI_COVERAGE_FIXTURE_INVARIANT_FAIL",false));}function invariant_valuePositive() public view {assert(target.value()>0);}}')

            def git(*args):
                return subprocess.check_output(['git', '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', *args], cwd=root, text=True, stderr=subprocess.DEVNULL).strip()
            module = Path(tmp) / 'module'
            module.mkdir()
            for args in [('init', '-q'), ('add', '.'), ('commit', '-qm', 'original module')]:
                if args[0] == 'add':
                    (module / 'original.txt').write_text('original')
                subprocess.check_call(['git', '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', *args], cwd=module, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            git('init', '-q')
            git('-c', 'protocol.file.allow=always', 'submodule', 'add', '-q', str(module), 'modules/original')
            git('add', '.')
            git('commit', '-qm', 'coverage fixture')
            sha = git('rev-parse', 'HEAD')

            class Archive(http.server.BaseHTTPRequestHandler):

                def do_POST(self):
                    request = json.loads(self.rfile.read(int(self.headers['Content-Length'])))
                    value = '0x1' if request['method'] == 'eth_chainId' else {'number': '0x2a', 'hash': '0x' + 'a' * 64, 'timestamp': '0x1234'}
                    body = json.dumps({'jsonrpc': '2.0', 'id': request['id'], 'result': value}).encode()
                    self.send_response(200)
                    self.send_header('Content-Type', 'application/json')
                    self.end_headers()
                    self.wfile.write(body)

                def log_message(self, *_):
                    pass
            server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Archive)
            thread = threading.Thread(target=server.serve_forever, daemon=True)
            thread.start()
            self.addCleanup(server.server_close)
            self.addCleanup(server.shutdown)
            url = 'http://127.0.0.1:' + str(server.server_address[1]) + '/very-private-ci-test-secret'
            marker = Path(tmp) / 'marker'
            env = dict(os.environ, CI_BRANCH='pilot', CI_COMMIT_SHA=sha, CI_CONTRACT_COVERAGE_REPLAY='true', CI_COVERAGE_FIXTURE_MARKER=str(marker), OP_CI_MAINNET_L1_ARCHIVE_RPC_URL=url)
            retained_failures = 0

            def invoke(mode, provider='circleci', prepared=None, block=None, failing=False, available=True, invariant_failing=False):
                nonlocal retained_failures
                argv = ['python3', str(root / 'ops/ci/runtime/contract-coverage.py'), mode]
                if mode != 'preflight':
                    argv += ['--feature', 'main']
                if prepared:
                    argv += ['--prepared', str(prepared)]
                if block:
                    argv += ['--block', str(block)]
                current = dict(env, CI_CONTRACT_PROVIDER=provider, RWX_RUN_ID='b' * 32, RWX_TASK_ATTEMPT_NUMBER='1')
                if failing:
                    current['CI_COVERAGE_FIXTURE_FAIL'] = '1'
                if invariant_failing:
                    current['CI_COVERAGE_FIXTURE_INVARIANT_FAIL'] = '1'
                if not available:
                    current.pop('OP_CI_MAINNET_L1_ARCHIVE_RPC_URL')
                child = subprocess.Popen(argv, cwd=root, env=current, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
                try:
                    stdout, stderr = child.communicate(timeout=180)
                except subprocess.TimeoutExpired:
                    child.terminate()
                    stdout, stderr = child.communicate(timeout=20)
                    self.fail('Pinned coverage fixture timed out: ' + stderr[-1200:])
                self.assertNotIn(url, stdout + stderr)
                if child.returncode and os.environ.get('RWX_COVERAGE_FIXTURE_RETAIN_DIR'):
                    retained_failures += 1
                    label = 'unit-failure' if failing else 'invariant-failure' if invariant_failing else 'unavailable-rpc' if not available else 'validation-failure'
                    target = Path(os.environ['RWX_COVERAGE_FIXTURE_RETAIN_DIR']) / sha / f'{retained_failures:02d}-{mode}-{label}'
                    while target.exists():
                        target = target.with_name(target.name + '-next')
                    shutil.copytree(root, target)
                return (child.returncode, root / '.ci/contract-coverage' / ('preflight' if mode == 'preflight' else 'main/' + mode), stdout + stderr)
            code, _, log = invoke('run', available=False)
            self.assertNotEqual(code, 0, log)
            self.assertFalse(marker.exists())
            code, preflight, log = invoke('preflight')
            self.assertEqual(code, 0, log)
            code, prepared, log = invoke('prepare', 'rwx')
            self.assertEqual(code, 0, log)
            self.assertFalse(marker.exists())
            self.assertEqual(json.loads((prepared / 'compile-only.json').read_text())['tests'], 0)
            code, directory, log = invoke('run', 'rwx', prepared, preflight / 'block.json')
            self.assertEqual(code, 0, log)
            self.assertEqual(marker.read_text(), 'xx')
            ordinary = json.loads((directory / 'ordinary/coverage.json').read_text())
            self.assertEqual(ordinary['outcomes'], {'pass': 4, 'skip': 2})
            events = json.loads((directory / 'ordinary/original-events.json').read_text())
            self.assertEqual(events['engine_summary'], {'suites': 3, 'pass': 3, 'fail': 0, 'skip': 2, 'total': 5})
            self.assertEqual(len(events['invariant_campaigns'][0]['members']), 2)
            self.assertEqual(len(ordinary['non_executable']), 1)
            self.assertEqual(json.loads((directory / 'upgrade/coverage.json').read_text())['outcomes'], {'pass': 1})
            phase_ids = {}
            for phase in C.PHASES:
                phase_ids[phase] = {(r.attrib['classname'], r.attrib['name']) for r in C.ET.parse(directory / phase / 'derived.junit.xml').iter('testcase')}
                self.assertTrue(all((identity.startswith(phase + ':') for identity, _ in phase_ids[phase])))
            self.assertTrue(phase_ids['ordinary'].isdisjoint(phase_ids['upgrade']))
            self.assertTrue((contracts / 'lcov.info').is_file())
            self.assertTrue((contracts / 'lcov-upgrade.info').is_file())
            first = Path(tmp) / 'native'
            shutil.copytree(directory, first)
            code, directory, log = invoke('run', block=preflight / 'block.json')
            self.assertEqual(code, 0, log)
            self.assertEqual(marker.read_text(), 'xxxx')
            for phase in C.PHASES:
                self.assertEqual(json.loads((directory / phase / 'original-cases.json').read_text()), json.loads((first / phase / 'original-cases.json').read_text()))
                self.assertEqual(C.lcov((directory / phase / 'original.lcov.info').read_text()), C.lcov((first / phase / 'original.lcov.info').read_text()))
            circle = Path(tmp) / 'circle'
            shutil.copytree(directory, circle)
            self.verify_provider_reports(root, circle, first, sha, tmp)
            code, prepared, log = invoke('prepare', 'rwx')
            self.assertEqual(code, 0, log)
            ffi = contracts / 'scripts/go-ffi/go-ffi'
            original_binary = ffi.read_bytes()
            ffi.write_bytes(original_binary + b'corrupt')
            code, directory, log = invoke('run', 'rwx', prepared, preflight / 'block.json')
            self.assertNotEqual(code, 0, log)
            self.assertEqual(marker.read_text(), 'xxxx')
            self.assertFalse((directory / 'ordinary').exists())
            ffi.write_bytes(original_binary)
            settings = prepared / 'settings.json'
            original_settings = settings.read_bytes()
            stale = json.loads(original_settings)
            stale['source_sha'] = 'f' * 40
            settings.write_text(json.dumps(stale))
            C.UP.finish(prepared, 0, [])
            code, directory, log = invoke('run', 'rwx', prepared, preflight / 'block.json')
            self.assertNotEqual(code, 0, log)
            self.assertEqual(marker.read_text(), 'xxxx')
            self.assertFalse((directory / 'ordinary').exists())
            settings.write_bytes(original_settings)
            C.UP.finish(prepared, 0, [])
            code, directory, log = invoke('run', block=preflight / 'block.json', failing=True)
            self.assertNotEqual(code, 0, log)
            self.assertEqual(json.loads((directory / 'ordinary/tests.stage.json').read_text())['exit_code'], 1)
            self.assertTrue((directory / 'ordinary/original.lcov.info').is_file())
            self.assertTrue((directory / 'ordinary/original.attribution.json').is_file())
            self.assertTrue(any((r['outcome'] == 'fail' for r in json.loads((directory / 'ordinary/original-cases.json').read_text()))))
            self.assertFalse((directory / 'upgrade').exists())
            self.assertTrue((directory / 'rerun.stage.json').exists())
            self.assertNotEqual(json.loads((directory / 'final.json').read_text())['exit_code'], 0)
            self.assertFalse(json.loads((directory / 'final.json').read_text())['report_errors'])
            code, directory, log = invoke('run', block=preflight / 'block.json', invariant_failing=True)
            self.assertNotEqual(code, 0, log)
            events = json.loads((directory / 'ordinary/original-events.json').read_text())
            self.assertEqual(events['invariant_campaigns'][0]['outcome'], 'fail')
            self.assertEqual({r['name']: r['outcome'] for r in events['cases'] if r['class'].endswith(':InvariantFixtureTest')}, {'invariant_value()': 'fail', 'invariant_valuePositive()': 'pass'})
            self.assertEqual(json.loads((directory / 'ordinary/tests.stage.json').read_text())['exit_code'], 1)
            self.assertFalse(json.loads((directory / 'final.json').read_text())['report_errors'])


if __name__ == '__main__': unittest.main()


@unittest.skipUnless(os.environ.get('RWX_LIVE_CONTRACT_COVERAGE_FIXTURE') == '1', 'Opt-in original Forge/Go fixture')
class LiveCoverageTests(_LiveCoverageTestsFixtures, unittest.TestCase):
    def test_actual_just_passes_setup_skips_abstracts_fresh_execution_and_initial_failure(self):
        self.exercise_original_reports()
