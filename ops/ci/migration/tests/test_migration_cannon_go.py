"""Migration-only comparisons using permanent execution fixture mechanics."""
import importlib.util
from pathlib import Path
import sys
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tests'))
import test_cannon_go as T
G = T.G
SCRIPTS = T.SCRIPTS
argparse = T.argparse
json = T.json
os = T.os
shutil = T.shutil
signal = T.signal
subprocess = T.subprocess
tempfile = T.tempfile
time = T.time
_SPEC=importlib.util.spec_from_file_location('compare_cannon_go', Path(__file__).resolve().parents[1] / 'compare-cannon-go.py')
COMPARE=importlib.util.module_from_spec(_SPEC);_SPEC.loader.exec_module(COMPARE)


@unittest.skipUnless(os.environ.get('RWX_LIVE_CANNON_GO_FIXTURE') == '1', 'Opt-in real Go, gotestsum and Forge fixtures')
class LiveTests(T._LiveTestsFixtures, unittest.TestCase):

    def test_real_fresh_execution_complete_selection_runtime_tools_and_parity(self):
        evidence = self.root / '.ci/comparison'
        evidence.mkdir()
        for provider, destination in [('circleci', 'circle'), ('rwx', 'rwx')]:
            result = self.run_fixture(provider)
            self.assertEqual(result.returncode, 0, result.stdout)
            shutil.copytree(self.report, evidence / destination)
            self.retain('fresh-' + destination)
        self.assertEqual((self.root / '.ci/fresh-marker').read_text(), 'fresh\nfresh\n')
        result = COMPARE.compare(evidence, self.sha)
        self.assertEqual(result['packages'], 4)
        self.assertEqual(result['initial_tests'], 5)
        self.assertEqual(result['outcomes'], {'pass': 6, 'skip': 1})
        self.assertEqual(result['settings']['rwx']['count'], 1)

    def test_real_failure_retains_original_case_logs_and_no_retries(self):
        result = self.run_fixture(CANNON_FIXTURE_FAIL='1')
        self.assertEqual(result.returncode, 1, result.stdout)
        final = G.read(self.report / 'final.json')
        coverage = G.read(self.report / 'coverage.json')
        self.assertEqual(final['exit_code'], 1)
        self.assertEqual(final['report_errors'], [])
        case = next((c for c in coverage['cases'] if c['name'] == 'TestFresh'))
        self.assertEqual(case['outcome'], 'fail')
        self.assertEqual(len(case['attempts']), 1)
        self.assertIn('original isolated Cannon failure', (self.report / 'original.json').read_text())
        self.retain('intentional-failure')
        with self.assertRaisesRegex(ValueError, 'Failed or incomplete'):
            COMPARE.originals(self.report, 'circleci')

    def test_real_cancellation_preserves_signal_and_partial_original_reports(self):
        log = self.root / '.ci/cancellation.log'
        with log.open('w') as stream:
            child = subprocess.Popen(self.args(), cwd=self.root, env={**self.environment, 'CANNON_FIXTURE_CANCEL': '1'}, stdout=stream, stderr=subprocess.STDOUT)
            try:
                deadline = time.monotonic() + 60
                while time.monotonic() < deadline and child.poll() is None:
                    path = self.report / 'original.json'
                    if path.exists() and 'fixture is waiting for cancellation' in path.read_text():
                        break
                    time.sleep(0.1)
                else:
                    self.fail('Actual Go fixture never reached cancellation boundary: ' + log.read_text())
                child.send_signal(signal.SIGTERM)
                self.assertNotEqual(child.wait(timeout=15), 0)
            finally:
                if child.poll() is None:
                    child.kill()
                    child.wait()
        final = G.read(self.report / 'final.json')
        self.assertEqual(G.read(self.report / 'tests.stage.json')['exit_code'], -signal.SIGTERM)
        self.assertEqual(final['exit_code'], 128 + signal.SIGTERM)
        self.assertTrue(final['report_errors'])
        self.assertIn('original.json', final['original_sha256'])
        self.retain('cancellation')
        with self.assertRaisesRegex(ValueError, 'Failed or incomplete'):
            COMPARE.originals(self.report, 'circleci')
