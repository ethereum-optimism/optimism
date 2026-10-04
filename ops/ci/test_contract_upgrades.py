#!/usr/bin/env python3
"""Exercise authoritative discovery, original failures, credentials and forks."""
import importlib.util
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import io
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

SPEC = importlib.util.spec_from_file_location('upgrade', Path(__file__).with_name('contract-upgrades.py'))
UP = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(UP)


class UpgradeTests(unittest.TestCase):
    def test_selection_includes_every_new_test_and_rejects_duplicate_or_errors(self):
        data = {'test/L1/First.t.sol': {'First': ['test_a()', 'testFuzz_new(uint256)']},
                'test/cannon/New.t.sol': {'New': ['test_added()']}}
        self.assertEqual(len(UP.selection(data)), 3)
        for value in ({}, {'error': 'compile failed'}, {'test/A.t.sol': {'A': ['test_a()', 'test_a()']}}, {'test/A.t.sol': {'A': [None]}}):
            with self.subTest(value=value), self.assertRaises(ValueError): UP.selection(value)

    def test_complete_junit_preserves_skip_and_rejects_missing_duplicate_failure_or_all_skip(self):
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp) / 'junit.xml'; selected = [['test/A.t.sol:A', 'test_a()'], ['test/A.t.sol:A', 'test_skip()']]
            original = '<testsuite><testcase classname="test/A.t.sol:A" name="test_a()"/><testcase classname="test/A.t.sol:A" name="test_skip()"><skipped>feature disabled</skipped></testcase></testsuite>'
            p.write_text(original); result = UP.junit(p, selected)
            self.assertEqual(result['outcomes'], {'pass': 1, 'skip': 1}); self.assertEqual(result['cases'][1]['skip_reason']['text'], 'feature disabled')
            for value in (original.replace('test_a()', 'extra()'), original.replace('test_skip()', 'test_a()'),
                          original.replace('name="test_a()"/>', 'name="test_a()"><failure>original failure</failure></testcase>'),
                          original.replace('name="test_a()"/>', 'name="test_a()"><skipped/></testcase>')):
                p.write_text(value)
                with self.subTest(value=value), self.assertRaises(ValueError): UP.junit(p, selected)

    def test_setup_skip_expands_only_the_bound_contract_and_retains_original(self):
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp) / 'junit.xml'
            selected = [('test/A.t.sol:A', 'test_a()'), ('test/B.t.sol:B', 'test_one()'), ('test/B.t.sol:B', 'test_two()')]
            original = '<testsuites><testsuite name="test/A.t.sol:A"><testcase name="test_a()"/></testsuite>' \
                       '<testsuite name="test/B.t.sol:B"><testcase name="setUp()"><skipped message="fork disabled"/></testcase></testsuite></testsuites>'
            p.write_text(original); result = UP.junit(p, selected)
            self.assertEqual(result['outcomes'], {'pass': 1, 'skip': 2})
            self.assertEqual(len(result['original_cases']), 2)
            self.assertEqual([c['verdict_source'] for c in result['cases']], ['test_a()', 'setUp()', 'setUp()'])
            for value in (original.replace('test/B.t.sol:B', 'test/Extra.t.sol:Extra'),
                          original.replace('<skipped message="fork disabled"/>', '<failure>setup failed</failure>'),
                          original.replace('<testcase name="setUp()">', '<testcase name="test_one()"/><testcase name="setUp()">')):
                p.write_text(value)
                with self.subTest(value=value), self.assertRaises(ValueError): UP.junit(p, selected)

    def test_absent_verdict_requires_original_empty_creation_bytecode(self):
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp) / 'junit.xml'; p.write_text('<testsuite name="test/A.t.sol:A"><testcase name="test_a()"/></testsuite>')
            selected = [('test/A.t.sol:A', 'test_a()'), ('test/Parent.t.sol:Parent', 'test_inherited()')]
            bindings = {identity: {'creation_bytecode': {'bytes': 1, 'sha256': UP.hashlib.sha256(b'00').hexdigest()}}
                        for identity, _ in selected}
            with self.assertRaises(ValueError): UP.junit(p, selected, bindings)
            bindings['test/Parent.t.sol:Parent']['creation_bytecode'] = {'bytes': 0, 'sha256': UP.hashlib.sha256(b'').hexdigest()}
            result = UP.junit(p, selected, bindings)
            self.assertEqual(result['outcomes'], {'pass': 1}); self.assertEqual(len(result['non_executable']), 1)
            self.assertEqual(result['non_executable'][0]['name'], 'test_inherited()')
            bindings['test/Parent.t.sol:Parent']['creation_bytecode']['sha256'] = 'a' * 64
            with self.assertRaisesRegex(ValueError, 'bytecode hash'): UP.junit(p, selected, bindings)

    def test_all_seven_occurrences_have_distinct_identity_including_both_op_main(self):
        self.assertEqual(len(UP.VARIANTS), 7)
        self.assertEqual(UP.VARIANTS['chain-op'], UP.VARIANTS['feature-main'])
        with patch.dict(os.environ, {'CI_BRANCH': 'pilot', 'FOUNDRY_MATCH_TEST': 'partial',
                         'FOUNDRY_FUZZ_RUNS': '0', 'DEV_FEATURE__UNRELATED': 'true', 'ETH_RPC_URL': 'ambient'}, clear=True):
            self.assertEqual(UP.configure('feature-CUSTOM_GAS_TOKEN'), ('pilot', 'op', 'CUSTOM_GAS_TOKEN'))
            self.assertEqual(os.environ['FOUNDRY_FUZZ_RUNS'], '1'); self.assertEqual(os.environ['FOUNDRY_FUZZ_SEED'], '42424242')
            self.assertEqual(os.environ['FOUNDRY_PROFILE'], 'liteci'); self.assertEqual(os.environ['SYS_FEATURE__CUSTOM_GAS_TOKEN'], 'true')
            for name in ('ETH_RPC_URL', 'FOUNDRY_MATCH_TEST', 'DEV_FEATURE__UNRELATED'): self.assertNotIn(name, os.environ)
            os.environ['CI_BRANCH'] = 'develop'; UP.configure('chain-ink'); self.assertEqual(os.environ['FOUNDRY_PROFILE'], 'ci')
            self.assertNotIn('SYS_FEATURE__CUSTOM_GAS_TOKEN', os.environ)

    def test_rpc_identity_checks_actual_chain_height_hash_and_availability(self):
        correct = {'number': '0x42', 'hash': '0x' + 'a' * 64, 'timestamp': '0x64'}
        with patch.object(UP, 'rpc', side_effect=['0x1', correct]): self.assertEqual(UP.block('not exposed', 66)['number'], 66)
        for responses in (['0xa', correct], ['0x1', None], ['0x1', correct | {'number': '0x43'}], ['0x1', correct | {'hash': 'corrupt'}]):
            with patch.object(UP, 'rpc', side_effect=responses), self.assertRaises(ValueError): UP.block('not exposed', 66)
        with patch.object(UP.urllib.request, 'urlopen', side_effect=OSError('secret-url-in-exception')), patch.object(UP.time, 'sleep'):
            with self.assertRaisesRegex(ValueError, '^Test-only archive RPC unavailable: OSError$'): UP.rpc('https://example.invalid/private', 'test', [])

    def test_rpc_uses_working_archive_client_headers_and_safe_error_categories(self):
        url = 'https://example.invalid/private-secret'
        with patch.object(UP.urllib.request, 'urlopen', return_value=io.BytesIO(b'{"id":1,"result":"0x1"}')) as opened:
            self.assertEqual(UP.rpc(url, 'eth_chainId', []), '0x1')
            self.assertEqual(opened.call_args.args[0].get_header('User-agent'), 'Go-http-client/1.1')
        error = UP.urllib.error.HTTPError(url, 403, 'private-secret response', {}, None)
        with patch.object(UP.urllib.request, 'urlopen', side_effect=error), patch.object(UP.time, 'sleep'):
            with self.assertRaisesRegex(ValueError, '^Test-only archive RPC unavailable: HTTP 403$'): UP.rpc(url, 'eth_chainId', [])

    def test_rpc_redaction_covers_url_xml_encoded_credentials_and_fragments(self):
        value = 'https://user:password-token@example.invalid/api/path-token-of-32-characters?key=query-token-value&chain=1'
        mask = UP.Redactor(value)
        for data in (value, UP.html.escape(value), UP.urllib.parse.quote(value, safe=''), 'password-token', 'query-token-value', 'path-token-of-32-characters'):
            self.assertNotIn(data.encode(), mask(data.encode()))
        with self.assertRaises(ValueError): UP.Redactor('file:///secret')

    def test_real_stage_keeps_initial_failure_and_redacts_both_streams_before_hashing(self):
        with tempfile.TemporaryDirectory() as tmp, patch.object(UP, 'CONTRACTS', Path(tmp)):
            d = Path(tmp); url = 'https://example.invalid/a-test-secret-token'
            status = UP.stage(d, 'intentional', [sys.executable, '-c', 'import sys;print(sys.argv[1]);print(sys.argv[1],file=sys.stderr);sys.exit(17)', url], UP.Redactor(url), True)
            self.assertEqual(status, 17)
            for p in [d / 'intentional.json', d / 'intentional.stderr.log']:
                self.assertNotIn(url, p.read_text()); self.assertIn('<test-only-rpc>', p.read_text())
            self.assertEqual(json.loads((d / 'intentional.stage.json').read_text())['stdout_sha256'], UP.digest(d / 'intentional.json'))
            self.assertEqual(UP.finish(d, status, []), 17)
            # Stage argv can never include an RPC URL in the production adapter.
            self.assertEqual(json.loads((d / 'final.json').read_text())['exit_code'], 17)
            for file in d.iterdir(): self.assertNotIn(url.encode(), file.read_bytes())

    def test_real_process_group_cancellation_retains_signal_and_partial_original(self):
        with tempfile.TemporaryDirectory() as tmp:
            d = Path(tmp)
            source = ('import importlib.util,json,os,signal,sys,threading,time;from pathlib import Path;'
                's=importlib.util.spec_from_file_location("u",sys.argv[1]);u=importlib.util.module_from_spec(s);s.loader.exec_module(u);'
                'u.CONTRACTS=Path(sys.argv[2]);'
                'threading.Timer(0.4,lambda:os.kill(os.getpid(),signal.SIGTERM)).start();'
                'status=u.stage(Path(sys.argv[2]),"canceled",[sys.executable,"-u","-c","import time;print(123,flush=True);time.sleep(20)"]);'
                'sys.exit(u.finish(Path(sys.argv[2]),status,[]))')
            result = subprocess.run([sys.executable, '-c', source, str(Path(UP.__file__)), str(d)], capture_output=True, timeout=10)
            self.assertEqual(result.returncode, 143, result.stderr.decode())
            self.assertEqual((d / 'canceled.log').read_text().strip(), '123')
            self.assertEqual(json.loads((d / 'canceled.stage.json').read_text())['exit_code'], -signal.SIGTERM)


@unittest.skipUnless(os.environ.get('RWX_LIVE_FORGE_FIXTURE') == '1', 'Opt-in pinned Forge discovery/verdict fixture')
class LiveForgeTests(unittest.TestCase):
    def test_complete_prepare_and_fresh_verdict_use_real_just_go_ffi_and_forge(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); contracts = root / 'packages/contracts-bedrock'
            (contracts / 'test/L1').mkdir(parents=True); (contracts / 'scripts/go-ffi').mkdir(parents=True)
            (contracts / 'scripts/checks').mkdir(parents=True); (root / 'ops/ci').mkdir(parents=True)
            for name in ('contract-upgrades.py', 'git-submodule-report.py', 'compare-rust-e2e.py', 'compare-contract-artifacts.py'):
                shutil.copy2(Path(UP.__file__).with_name(name), root / 'ops/ci' / name)
            original_just = (UP.CONTRACTS / 'justfile').read_text()
            # Keep the production build/runtime recipes byte for byte. Only the
            # external daily-height lookup uses a fixed local RPC fixture.
            recipes = []
            for name in ('build-go-ffi', 'prepare-upgrade-env', 'test-upgrade', 'test-upgrade-rerun'):
                start = original_just.index('\n' + name + ':') if name == 'build-go-ffi' else original_just.index('\n' + name + ' ')
                end = original_just.index('\n\n', start + 1)
                recipes.append(original_just[start + 1:end])
            (contracts / 'justfile').write_text('\n\n'.join(recipes) + '\n\nprint-pinned-block-number:\n  @echo 66\n')
            check = contracts / 'scripts/checks/check-junit-tests-ran.sh'
            shutil.copy2(UP.CONTRACTS / 'scripts/checks/check-junit-tests-ran.sh', check)
            (root / 'go.mod').write_text('module example.invalid/upgrade-fixture\n\ngo 1.26.5\n')
            (contracts / 'scripts/go-ffi/main.go').write_text('package main\nimport("fmt";"os")\nfunc main(){'
                'f,e:=os.OpenFile(os.Getenv("CI_UPGRADE_FIXTURE_MARKER"),os.O_APPEND|os.O_CREATE|os.O_WRONLY,0600);'
                'if e!=nil{panic(e)};defer f.Close();f.WriteString("x");'
                'v:=1;if os.Getenv("CI_UPGRADE_FIXTURE_FAIL")=="1"{v=0};fmt.Printf("0x%064x",v)}\n')
            (contracts / 'foundry.toml').write_text('[profile.default]\nsrc="src"\ntest="test"\nout="forge-artifacts"\n'
                'solc="0.8.15"\nffi=true\n[profile.liteci]\noptimizer=true\n[profile.liteci.fuzz]\nruns=1\n')
            (contracts / 'test/L1/Fixture.t.sol').write_text('pragma solidity 0.8.15; interface Vm {'
                'function skip(bool) external; function ffi(string[] calldata) external returns(bytes memory); } '
                'contract Fixture { Vm constant vm=Vm(address(uint160(uint256(keccak256("hevm cheat code"))))); '
                'function test_runtimeFFI() public {string[] memory args=new string[](1);args[0]="./scripts/go-ffi/go-ffi";'
                'assert(abi.decode(vm.ffi(args),(uint256))==1);} '
                'function testFuzz_fresh(uint256 a) public pure { assert(a==a); } '
                'function test_skip() public { vm.skip(true); } }')
            def git(directory, *args):
                return subprocess.check_output(['git', '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', *args],
                                               cwd=directory, text=True, stderr=subprocess.DEVNULL).strip()
            module = root / 'fixture-module'; module.mkdir(); git(module, 'init', '-q')
            (module / 'original.txt').write_text('original fixture revision\n'); git(module, 'add', '.'); git(module, 'commit', '-qm', 'fixture')
            git(root, 'init', '-q'); git(root, '-c', 'protocol.file.allow=always', 'submodule', 'add', '-q', str(module), 'modules/original')
            shutil.rmtree(module); git(root, 'add', '.'); git(root, 'commit', '-qm', 'complete fixture'); sha = git(root, 'rev-parse', 'HEAD')
            class Rpc(BaseHTTPRequestHandler):
                def do_POST(self):
                    if self.headers.get('User-Agent') != 'Go-http-client/1.1':
                        self.send_error(403); return
                    data = json.loads(self.rfile.read(int(self.headers['Content-Length'])))
                    values = {'eth_chainId': '0x1', 'eth_getBlockByNumber': {'number': '0x42', 'hash': '0x' + 'a' * 64, 'timestamp': '0x64'}}
                    response = json.dumps({'jsonrpc':'2.0', 'id':data['id'], 'result':values[data['method']]}).encode()
                    self.send_response(200); self.send_header('Content-Type','application/json'); self.end_headers(); self.wfile.write(response)
                def log_message(self, *_): pass
            server = ThreadingHTTPServer(('127.0.0.1', 0), Rpc); thread = threading.Thread(target=server.serve_forever, daemon=True); thread.start()
            self.addCleanup(server.server_close); self.addCleanup(thread.join, 5); self.addCleanup(server.shutdown)
            marker = root / 'fresh-execution.txt'
            env = {'CI_BRANCH':'pilot', 'CI_COMMIT_SHA':sha, 'CI_CONTRACT_PROVIDER':'rwx', 'RWX_RUN_ID':'f' * 32,
                   'RWX_TASK_ATTEMPT_NUMBER':'1', 'CI_UPGRADE_FIXTURE_MARKER':str(marker),
                   'OP_CI_MAINNET_L1_ARCHIVE_RPC_URL':'http://127.0.0.1:' + str(server.server_port)}
            previous = Path.cwd(); self.addCleanup(os.chdir, previous)
            with patch.object(UP, 'ROOT', root), patch.object(UP, 'CONTRACTS', contracts), patch.dict(os.environ, env):
                def execute(args):
                    with patch.object(sys, 'argv', [str(UP.__file__), *args]): return UP.main()
                self.assertEqual(execute(['preflight']), 0)
                pinned = root / '.ci/contract-upgrades/preflight/block.json'
                self.assertEqual(execute(['prepare', '--variant', 'feature-main']), 0)
                prepared = root / '.ci/contract-upgrades/feature-main/prepare'
                args = ['run', '--variant', 'feature-main', '--prepared', str(prepared), '--block', str(pinned)]
                for size in (1, 2):
                    self.assertEqual(execute(args), 0)
                    self.assertEqual(marker.read_text(), 'x' * size)
                    result = json.loads((root / '.ci/contract-upgrades/feature-main/run/coverage.json').read_text())
                    self.assertEqual(result['outcomes'], {'pass':2, 'skip':1})
                ffi = contracts / 'scripts/go-ffi/go-ffi'; original = ffi.read_bytes(); ffi.write_bytes(original + b'corrupt')
                self.assertEqual(execute(args), 1); self.assertEqual(marker.read_text(), 'xx'); ffi.write_bytes(original)
                with patch.dict(os.environ, {'CI_UPGRADE_FIXTURE_FAIL':'1'}): self.assertNotEqual(execute(args), 0)
                failed = root / '.ci/contract-upgrades/feature-main/run'
                self.assertIn('<failure', (failed / 'original.junit.xml').read_text())
                self.assertIn('<failure', (failed / 'diagnostic.junit.xml').read_text())
                self.assertNotEqual(json.loads((failed / 'final.json').read_text())['exit_code'], 0)

    def test_actual_discovery_matches_actual_junit_including_fuzz_and_skip(self):
        self.assertTrue(shutil.which('forge'))
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); (root / 'test/L1').mkdir(parents=True)
            (root / 'foundry.toml').write_text('[profile.default]\nsrc="src"\ntest="test"\nsolc="0.8.15"\n[profile.default.fuzz]\nruns=1\n')
            (root / 'test/L1/Fixture.t.sol').write_text('pragma solidity 0.8.15; interface Vm { function skip(bool) external; } '
                'contract Fixture { function test_pass() public pure { assert(7==7); } '
                'function testFuzz_fresh(uint256 a) public pure { assert(a==a); } '
                'function testFuzz_fresh(address a) public pure { assert(a==a); } '
                'function test_skip() public { Vm(address(uint160(uint256(keccak256("hevm cheat code"))))).skip(true); } } '
                'abstract contract AbstractFixture { function test_inheritedA() public pure { assert(1==1); } '
                'function test_inheritedB() public pure { assert(2==2); } } contract InheritedFixture is AbstractFixture {} '
                'contract SetupSkippedFixture { function setUp() public { Vm(address(uint160(uint256(keccak256("hevm cheat code"))))).skip(true); } '
                'function test_one() public pure { assert(3==3); } function test_two() public pure { assert(4==4); } }')
            args = ['forge', 'test', '--match-path', UP.MATCH]
            found = subprocess.check_output(args + ['--list', '--json'], cwd=root)
            actual = subprocess.check_output(args + ['--junit'], cwd=root)
            junit = root / 'original.xml'; junit.write_bytes(actual)
            with patch.object(UP, 'ROOT', root): signatures = UP.compiler_signatures(root / 'out')
            discovered = UP.selection(json.loads(found), signatures)
            result = UP.junit(junit, discovered, signatures)
            self.assertEqual(len(discovered), 10)
            self.assertEqual(result['outcomes'], {'pass': 5, 'skip': 3}); self.assertEqual(len(result['cases']), 8)
            self.assertEqual(len(result['non_executable']), 2); self.assertEqual(len(result['original_cases']), 7)
            self.assertEqual(list(result['suite_setup_skips']), ['test/L1/Fixture.t.sol:SetupSkippedFixture'])


if __name__ == '__main__': unittest.main()
