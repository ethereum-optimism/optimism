#!/usr/bin/env python3
"""Exercise the L2 fork runner's job environment, fork block, recipes and reruns."""
import importlib.util
from pathlib import Path
import tempfile
import unittest
from unittest import mock

from test_contract_upgrades import RUNTIME, drive

SPEC = importlib.util.spec_from_file_location('l2', RUNTIME / 'contract-l2-fork.py')
L = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(L)
TESTS = ['just', 'test-l2-fork-upgrade', '--threads', '1', '--compute-units-per-second', '100']


class L2Tests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(); self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)

    def run_fork(self, statuses=(), block='latest'):
        with mock.patch.object(L.UP.REPORT, 'command', return_value='66') as command:
            status, calls = drive(L, self.root, ['run', '--fork-block', block], statuses)
        return status, calls, command

    def test_latest_forks_the_current_head_then_checks_the_bundle_and_runs_the_tests(self):
        status, calls, command = self.run_fork()
        self.assertEqual(status, 0)
        command.assert_called_once_with(['cast', 'block-number', '--rpc-url', L.RPC])
        self.assertEqual([call[:2] for call in calls], [('nut-bundle-check', ['just', 'nut-bundle-check-no-build']), ('tests', TESTS)])
        expected = {'CI': 'true', 'FOUNDRY_PROFILE': 'ci', 'L2_FORK_RPC_URL': 'https://mainnet.optimism.io', 'L2_FORK_BLOCK_NUMBER': '66',
                    'JUNIT_TEST_PATH': str(self.root / '.ci/contract-l2-fork/run/results.xml')}
        self.assertEqual([call[2] for call in calls], [expected] * 2)

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
                                                       ['just', 'test-l2-fork-upgrade-rerun', *TESTS[2:]]])
        # Like CircleCI's trace step, the rerun writes no JUnit, so forge prints its traces.
        report = str(self.root / '.ci/contract-l2-fork/run/results.xml')
        self.assertEqual([call[2].get('JUNIT_TEST_PATH') for call in calls], [report, report, None])

    def test_prepare_builds_with_the_ci_profile_and_no_rpc(self):
        status, calls = drive(L, self.root, ['prepare'])
        self.assertEqual(status, 0)
        expected = {'CI': 'true', 'FOUNDRY_PROFILE': 'ci'}
        self.assertEqual(calls, [('go-ffi', ['just', 'build-go-ffi'], expected), ('contracts-build', ['forge', 'build'], expected)])


if __name__ == '__main__': unittest.main()
