#!/usr/bin/env python3
"""Exercise the L2 fork runner's job environment, fork block, recipes and reruns, and the RPC pacer."""
from contextlib import contextmanager
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import importlib.util
from pathlib import Path
import tempfile
import threading
import unittest
from unittest import mock
import urllib.error
import urllib.request

from test_contract_upgrades import RUNTIME, drive

SPEC = importlib.util.spec_from_file_location('l2', RUNTIME / 'contract-l2-fork.py')
L = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(L)
TESTS = ['just', 'test-l2-fork-upgrade']


@contextmanager
def paced(upstream):
    yield 'paced:' + upstream


class L2Tests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(); self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)

    def run_fork(self, statuses=(), block='latest'):
        with mock.patch.object(L.UP.REPORT, 'command', return_value='66') as command, mock.patch.object(L.PACER, 'serve', paced):
            status, calls = drive(L, self.root, ['run', '--fork-block', block], statuses)
        return status, calls, command

    def test_latest_forks_the_current_head_then_checks_the_bundle_and_runs_the_tests(self):
        status, calls, command = self.run_fork()
        self.assertEqual(status, 0)
        command.assert_called_once_with(['cast', 'block-number', '--rpc-url', L.RPC])
        self.assertEqual([call[:2] for call in calls], [('nut-bundle-check', ['just', 'nut-bundle-check-no-build']), ('tests', TESTS)])
        expected = {'CI': 'true', 'FOUNDRY_PROFILE': 'ci', 'L2_FORK_BLOCK_NUMBER': '66',
                    'JUNIT_TEST_PATH': str(self.root / '.ci/contract-l2-fork/run/results.xml')}
        # Only the tests read the fork, and they read it through the pacer.
        self.assertEqual([call[2] for call in calls], [expected, {**expected, 'L2_FORK_RPC_URL': 'paced:https://mainnet.optimism.io'}])

    def test_given_fork_block_is_used_without_a_request_to_the_rpc(self):
        status, calls, command = self.run_fork(block='123')
        command.assert_not_called()
        self.assertEqual((status, calls[1][2]['L2_FORK_BLOCK_NUMBER']), (0, '123'))

    def test_nut_bundle_failure_prevents_the_tests(self):
        status, calls, _ = self.run_fork([17])
        self.assertEqual((status, [call[0] for call in calls]), (17, ['nut-bundle-check']))

    def test_test_failure_is_kept_and_the_rerun_writes_no_junit(self):
        status, calls, _ = self.run_fork([0, 23, 0])
        self.assertEqual(status, 23)
        self.assertEqual([call[1] for call in calls], [['just', 'nut-bundle-check-no-build'], TESTS,
                                                       ['just', 'test-l2-fork-upgrade-rerun']])
        self.assertEqual([call[2].get('L2_FORK_RPC_URL') for call in calls[1:]], ['paced:' + L.RPC] * 2)
        # Like CircleCI's trace step, the rerun writes no JUnit, so forge prints its traces.
        report = str(self.root / '.ci/contract-l2-fork/run/results.xml')
        self.assertEqual([call[2].get('JUNIT_TEST_PATH') for call in calls], [report, report, None])

    def test_prepare_builds_with_the_ci_profile_and_no_rpc(self):
        status, calls = drive(L, self.root, ['prepare'])
        self.assertEqual(status, 0)
        expected = {'CI': 'true', 'FOUNDRY_PROFILE': 'ci'}
        self.assertEqual(calls, [('go-ffi', ['just', 'build-go-ffi'], expected), ('contracts-build', ['forge', 'build'], expected)])


class PacerTests(unittest.TestCase):
    def setUp(self):
        # A stand-in upstream that records each request and answers with the next (status, body).
        self.seen = seen = []; self.answers = answers = []

        class Upstream(BaseHTTPRequestHandler):
            def do_POST(self):
                seen.append((self.headers['User-Agent'], self.headers['Content-Type'], self.rfile.read(int(self.headers['Content-Length']))))
                status, body = answers.pop(0)
                self.send_response(status); self.send_header('Content-Length', str(len(body))); self.end_headers(); self.wfile.write(body)

            def log_message(self, *_): pass

        server = ThreadingHTTPServer(('127.0.0.1', 0), Upstream)
        threading.Thread(target=server.serve_forever, daemon=True).start()
        self.addCleanup(server.server_close); self.addCleanup(server.shutdown)
        self.upstream = 'http://127.0.0.1:' + str(server.server_port)

    def post(self, endpoint, body):
        request = urllib.request.Request(endpoint, data=body, headers={'Content-Type': 'application/json', 'User-Agent': 'forge'})
        try:
            with urllib.request.urlopen(request, timeout=10) as response: return response.status, response.read()
        except urllib.error.HTTPError as failure:
            with failure: return failure.code, failure.read()

    def test_client_receives_each_upstream_answer_unchanged_and_the_pacer_does_not_retry(self):
        self.answers += [(200, b'{"result":"0x1"}'), (429, b'{"error":"slow down"}')]
        with L.PACER.serve(self.upstream, wait=lambda seconds: None) as endpoint:
            self.assertEqual(self.post(endpoint, b'{"id":1}'), (200, b'{"result":"0x1"}'))
            # Forge owns the retries: it must see the rate limit answer itself.
            self.assertEqual(self.post(endpoint, b'{"id":2}'), (429, b'{"error":"slow down"}'))
        self.assertEqual(self.seen, [('forge', 'application/json', b'{"id":1}'), ('forge', 'application/json', b'{"id":2}')])

    def test_requests_start_one_interval_apart(self):
        now = [100.0]; waits = []
        def wait(seconds): waits.append(seconds); now[0] += seconds
        self.answers += [(200, b'{}')] * 3
        with L.PACER.serve(self.upstream, rate=2, clock=lambda: now[0], wait=wait) as endpoint:
            for _ in range(3): self.post(endpoint, b'{}')
            # A request that arrives after its start time does not wait.
            now[0] += 5; self.answers.append((200, b'{}')); self.post(endpoint, b'{}')
        self.assertEqual(waits, [0, 0.5, 0.5, 0])

    def test_unreachable_upstream_is_an_http_503_that_forge_retries(self):
        with L.PACER.serve('http://127.0.0.1:1', wait=lambda seconds: None) as endpoint:
            self.assertEqual(self.post(endpoint, b'{}'), (503, b''))


if __name__ == '__main__': unittest.main()
