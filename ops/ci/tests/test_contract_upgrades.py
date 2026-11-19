#!/usr/bin/env python3
"""Exercise the upgrade runner's job environment, recipes, reruns and credential redaction.

This module also holds the fixtures that the contract suite and L2 fork tests share.
"""
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import importlib.util
import json
import os
from pathlib import Path
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import threading
import time
import unittest
import urllib.request
from unittest.mock import patch
import xml.etree.ElementTree as ET

RUNTIME = Path(__file__).resolve().parents[1] / 'runtime'
SPEC = importlib.util.spec_from_file_location('upgrade', RUNTIME / 'contract-upgrades.py')
UP = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(UP)
# Prints 1, or 0 when CI_FIXTURE_FAIL=1, and appends one byte to $CI_FIXTURE_MARKER on each call.
FFI = ('package main\nimport("fmt";"os")\nfunc main(){'
       'f,e:=os.OpenFile(os.Getenv("CI_FIXTURE_MARKER"),os.O_APPEND|os.O_CREATE|os.O_WRONLY,0600);'
       'if e!=nil{panic(e)};defer f.Close();f.WriteString("x");'
       'v:=1;if os.Getenv("CI_FIXTURE_FAIL")=="1"{v=0};fmt.Printf("0x%064x",v)}\n')
VM = ('interface Vm {function skip(bool) external;function ffi(string[] calldata) external returns(bytes memory);'
      'function envString(string calldata) external view returns(string memory);'
      'function createSelectFork(string calldata,uint256) external returns(uint256);} '
      'abstract contract Cheats {Vm constant vm=Vm(address(uint160(uint256(keccak256("hevm cheat code")))));'
      'function ffi() internal returns(uint256) {string[] memory args=new string[](1);args[0]="./scripts/go-ffi/go-ffi";'
      'return abi.decode(vm.ffi(args),(uint256));}} ')


def drive(module, root, arguments, statuses=(), environment=None, effect=None):
    """Run a runner's main() with every stage stubbed and its reports under root.

    Return the exit status and, for each stage, its name, its argv and the environment it saw.
    """
    shared = getattr(module, 'UP', module); calls = []; results = iter(statuses)
    def stage(directory, name, argv, redactor=None):
        calls.append((name, argv, dict(os.environ)))
        if effect: effect(directory, name)
        return next(results, 0)
    with patch.object(shared, 'ROOT', root), patch.object(shared, 'stage', stage), \
         patch.object(sys, 'argv', ['runner', *arguments]), patch.dict(os.environ, environment or {}, clear=True):
        try: status = module.main()
        except SystemExit as stop: status = stop.code
    return status, calls


def project(root, toml, sources):
    """Create a Foundry project with the real justfile, the runners and the FFI helper.

    Return the contracts directory and invoke(script, *arguments, **environment), which runs a runner in the project.
    """
    contracts = root / 'packages/contracts-bedrock'
    for name in ('contract-upgrades.py', 'contract-suites.py', 'ci-report.py'):
        target = root / 'ops/ci/runtime' / name; target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(RUNTIME / name, target)
    files = {'foundry.toml': toml, 'scripts/go-ffi/main.go': FFI, **sources}
    for name in ('justfile', 'scripts/checks/check-junit-tests-ran.sh', 'scripts/ops/apply-match-path.sh'):
        files[name] = (UP.CONTRACTS / name).read_text()
    for name, text in files.items():
        target = contracts / name; target.parent.mkdir(parents=True, exist_ok=True); target.write_text(text)
        if name.endswith('.sh'): target.chmod(0o755)
    def invoke(script, *arguments, **environment):
        return subprocess.run([sys.executable, str(root / 'ops/ci/runtime' / script), *arguments], cwd=root, text=True,
                              capture_output=True, timeout=300,
                              env={**os.environ, 'CI_BRANCH': 'pilot', 'CI_FIXTURE_MARKER': str(root / 'ffi-calls'), **environment})
    return contracts, invoke


class UpgradeTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(); self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)

    def test_compile_builds_once_with_the_profile_of_the_branch_and_no_variant_setting(self):
        # One build serves every variant, so it has the profile and none of a variant's variables.
        expected = {'CI_BRANCH': 'pilot', 'CI': 'true', 'FOUNDRY_PROFILE': 'liteci'}
        status, calls = drive(UP, self.root, ['compile'], environment={'CI_BRANCH': 'pilot'})
        self.assertEqual(status, 0)
        self.assertEqual(calls, [('go-ffi', ['just', 'build-go-ffi'], expected), ('contracts-build', ['forge', 'build'], expected)])
        # develop uses the ci profile. A failed helper build stops the task before the contract build.
        expected = {'CI_BRANCH': 'develop', 'CI': 'true', 'FOUNDRY_PROFILE': 'ci'}
        status, calls = drive(UP, self.root, ['compile'], [3], {'CI_BRANCH': 'develop'})
        self.assertEqual((status, calls), (3, [('go-ffi', ['just', 'build-go-ffi'], expected)]))

    def test_every_feature_exports_only_its_own_flag(self):
        for feature, variable in UP.FEATURES.items():
            with self.subTest(feature=feature), patch.dict(os.environ, {}, clear=True):
                UP.configure(feature, FOUNDRY_PROFILE='ci')
                flags = {name: value for name, value in os.environ.items() if name.startswith(('DEV_FEATURE__', 'SYS_FEATURE__'))}
                self.assertEqual(flags, {variable: 'true'} if variable else {})

    def test_run_gives_the_recipe_the_archive_url_and_reruns_a_failure_without_a_junit_path(self):
        url = 'https://example.invalid/v2/a-test-secret-token'
        environment = {'CI_BRANCH': 'pilot', 'OP_CI_MAINNET_L1_ARCHIVE_RPC_URL': url}
        junit = str(self.root / '.ci/contract-upgrades/chain-op/run/results.xml')
        status, calls = drive(UP, self.root, ['run', '--variant', 'chain-op'], [1, 1], environment)
        self.assertEqual(status, 1)
        self.assertEqual([call[:2] for call in calls], [('tests', ['just', 'test-upgrade']), ('rerun', ['just', 'test-upgrade-rerun'])])
        # Like CircleCI's trace step, the rerun writes no JUnit, so forge prints its traces.
        self.assertEqual([(call[2]['ETH_RPC_URL'], call[2].get('JUNIT_TEST_PATH')) for call in calls], [(url, junit), (url, None)])
        # A passing run and a canceled run have no rerun.
        for statuses in ([0], [143]):
            status, calls = drive(UP, self.root, ['run', '--variant', 'chain-op'], statuses, environment)
            self.assertEqual((status, [call[0] for call in calls]), (statuses[0], ['tests']))

    def test_run_redacts_the_archive_url_in_the_junit_report(self):
        url = 'https://example.invalid/secret-token-of-32-characters?key=query-token-value&chain=1'
        def effect(directory, name):
            if name != 'tests': return
            suites = ET.Element('testsuites'); case = ET.SubElement(suites, 'testcase', name='failed()')
            ET.SubElement(case, 'failure', message='connection failed: ' + url).text = 'reverted: ' + url
            Path(os.environ['JUNIT_TEST_PATH']).write_bytes(ET.tostring(suites))
        status, _ = drive(UP, self.root, ['run', '--variant', 'chain-op'], [1, 1],
                          {'CI_BRANCH': 'pilot', 'OP_CI_MAINNET_L1_ARCHIVE_RPC_URL': url}, effect)
        self.assertEqual(status, 1)
        report = (self.root / '.ci/contract-upgrades/chain-op/run/results.xml').read_bytes()
        # The report stays valid XML and keeps the failure text around the masked URL.
        failure = ET.fromstring(report).find('testcase/failure')
        self.assertEqual((failure.get('message'), failure.text), ('connection failed: <test-only-rpc>', 'reverted: <test-only-rpc>'))
        for private in (b'secret-token-of-32-characters', b'query-token-value'): self.assertNotIn(private, report)

    def test_run_without_an_archive_url_stops_before_any_stage(self):
        for environment in ({'CI_BRANCH': 'pilot'}, {'CI_BRANCH': 'pilot', 'OP_CI_MAINNET_L1_ARCHIVE_RPC_URL': 'file:///secret'}):
            status, calls = drive(UP, self.root, ['run', '--variant', 'chain-op'], environment=environment)
            self.assertIn('OP_CI_MAINNET_L1_ARCHIVE_RPC_URL', status); self.assertNotIn('secret', status); self.assertEqual(calls, [])

    def test_short_reflected_credentials_are_redacted(self):
        url = 'https://user:pass@provider.invalid/v2/short-token?apikey=abc123'
        redactor = UP.Redactor(url)
        for value in (url, 'user', 'pass', 'short-token', 'abc123'):
            self.assertNotIn(value.encode(), redactor(value.encode()))

    def test_redaction_covers_url_xml_encoded_credentials_and_fragments(self):
        value = 'https://user:password-token@example.invalid/api/path-token-of-32-characters?key=query-token-value&chain=1'
        mask = UP.Redactor(value)
        for data in (value, UP.html.escape(value), UP.urllib.parse.quote(value, safe=''), 'password-token', 'query-token-value', 'path-token-of-32-characters'):
            self.assertNotIn(data.encode(), mask(data.encode()))
        with self.assertRaises(ValueError): UP.Redactor('file:///secret')

    def test_real_stage_redacts_both_streams_before_saving_and_keeps_the_exit_status(self):
        url = 'https://example.invalid/a-test-secret-token'
        child = 'import sys;print(sys.argv[1]);print(sys.argv[1],file=sys.stderr);sys.exit(17)'
        with patch.object(UP, 'CONTRACTS', self.root):
            self.assertEqual(UP.stage(self.root, 'intentional', [sys.executable, '-c', child, url], UP.Redactor(url)), 17)
        for name in ('intentional.log', 'intentional.stderr.log'):
            self.assertEqual((self.root / name).read_text(), '<test-only-rpc>\n')
        for file in self.root.iterdir(): self.assertNotIn(b'a-test-secret-token', file.read_bytes())

    def test_real_process_group_cancellation_returns_the_signal_and_keeps_the_partial_log(self):
        # run_tests skips the rerun after a cancellation: stage reports 128 plus the signal.
        source = ('import importlib.util,os,signal,sys,threading;from pathlib import Path;'
                  's=importlib.util.spec_from_file_location("u",sys.argv[1]);u=importlib.util.module_from_spec(s);s.loader.exec_module(u);'
                  'u.CONTRACTS=Path(sys.argv[2]);'
                  'threading.Timer(0.4,lambda:os.kill(os.getpid(),signal.SIGTERM)).start();'
                  'sys.exit(u.stage(Path(sys.argv[2]),"canceled",[sys.executable,"-u","-c","import time;print(123,flush=True);time.sleep(20)"]))')
        result = subprocess.run([sys.executable, '-c', source, UP.__file__, str(self.root)], capture_output=True, timeout=10)
        self.assertEqual(result.returncode, 143, result.stderr.decode())
        self.assertEqual((self.root / 'canceled.log').read_text().strip(), '123')
        self.assertEqual(json.loads((self.root / 'canceled.stage.json').read_text())['exit_code'], -signal.SIGTERM)


@unittest.skipUnless(os.environ.get('RWX_LIVE_FORGE_FIXTURE') == '1', 'Opt-in fixture: needs the pinned just, go, forge and anvil')
class LiveForgeTests(unittest.TestCase):
    def test_real_recipes_fork_through_the_archive_url_and_redact_it_from_a_failure(self):
        temporary = tempfile.TemporaryDirectory(); self.addCleanup(temporary.cleanup); root = Path(temporary.name)
        _, invoke = project(root, '[profile.default]\nsrc="src"\ntest="test"\nout="forge-artifacts"\nsolc="0.8.15"\n'
            'ffi=true\nno_storage_caching=true\n[profile.liteci]\noptimizer=true\n', {'test/L1/Fixture.t.sol':
            'pragma solidity 0.8.15; ' + VM + 'contract Fixture is Cheats {'
            # A failing helper makes the failure reason quote the archive URL.
            'function test_ffi() public {require(ffi()==1,vm.envString("FORK_RPC_URL"));} '
            'function test_fork() public {vm.createSelectFork(vm.envString("FORK_RPC_URL"),66);assert(block.number==66);} '
            'function testFuzz_fresh(uint256 a) public pure {assert(a==a);} '
            'function test_skip() public {vm.skip(true);}}'})
        (root / 'go.mod').write_text('module example.invalid/upgrade-fixture\n\ngo 1.22\n')
        # The local chain has no block at 00:00 UTC, so the pinned-block lookup of the recipe gets a fixed answer.
        tools = root / 'bin'; tools.mkdir(); (tools / 'cast').write_text('#!/bin/sh\necho 66\n'); (tools / 'cast').chmod(0o755)
        with socket.socket() as listener:
            listener.bind(('127.0.0.1', 0)); port = listener.getsockname()[1]
        anvil = subprocess.Popen(['anvil', '--silent', '--port', str(port), '--chain-id', '1'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        self.addCleanup(anvil.wait, 10); self.addCleanup(anvil.terminate)
        def call(body):
            request = urllib.request.Request('http://127.0.0.1:' + str(port), data=body, headers={'Content-Type': 'application/json'})
            with urllib.request.urlopen(request, timeout=5) as response: return response.read()
        for _ in range(100):
            try: call(b'{"jsonrpc":"2.0","id":1,"method":"anvil_mine","params":["0x42"]}'); break
            except OSError: time.sleep(0.05)
        else: self.fail('Anvil did not start')
        # This relay gives the chain a URL with a key in its path and its query, like a hosted archive RPC.
        class Rpc(BaseHTTPRequestHandler):
            def do_POST(self):
                response = call(self.rfile.read(int(self.headers['Content-Length'])))
                self.send_response(200); self.send_header('Content-Type', 'application/json')
                self.send_header('Content-Length', str(len(response))); self.end_headers(); self.wfile.write(response)
            def log_message(self, *_): pass
        server = ThreadingHTTPServer(('127.0.0.1', 0), Rpc); thread = threading.Thread(target=server.serve_forever, daemon=True); thread.start()
        self.addCleanup(server.server_close); self.addCleanup(thread.join, 5); self.addCleanup(server.shutdown)
        environment = {'PATH': str(tools) + os.pathsep + os.environ['PATH'],
                       'OP_CI_MAINNET_L1_ARCHIVE_RPC_URL': 'http://127.0.0.1:' + str(server.server_port) + '/short-token?apikey=abc123'}
        for arguments in (['compile'], ['run', '--variant', 'feature-main']):
            result = invoke('contract-upgrades.py', *arguments, **environment)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        report = root / '.ci/contract-upgrades/feature-main/run'
        cases = {case.get('name'): case for case in ET.parse(report / 'results.xml').iter('testcase')}
        self.assertEqual(set(cases), {'test_ffi()', 'test_fork()', 'testFuzz_fresh(uint256)', 'test_skip()'})
        self.assertEqual([name for name, case in cases.items() if case.find('skipped') is not None], ['test_skip()'])
        self.assertFalse(any(case.find('failure') is not None for case in cases.values()))
        self.assertTrue((root / 'ffi-calls').read_text())
        result = invoke('contract-upgrades.py', 'run', '--variant', 'feature-main', CI_FIXTURE_FAIL='1', **environment)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('<failure', (report / 'results.xml').read_text())
        self.assertEqual(json.loads((report / 'rerun.stage.json').read_text())['argv'], ['just', 'test-upgrade-rerun'])
        self.assertIn('[FAIL', (report / 'rerun.log').read_text())
        saved = result.stdout + result.stderr + ''.join(path.read_text() for path in report.iterdir())
        self.assertIn('test-only-rpc', (report / 'results.xml').read_text()); self.assertIn('<test-only-rpc>', (report / 'rerun.log').read_text())
        for private in ('short-token', 'abc123'): self.assertNotIn(private, saved)


if __name__ == '__main__': unittest.main()
