"""Migration-only comparisons using permanent execution fixture mechanics."""
import importlib.util
from pathlib import Path
import sys
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tests'))
import test_contract_coverage as T
C = T.C
http = T.http
json = T.json
os = T.os
patch = T.patch
shutil = T.shutil
subprocess = T.subprocess
tempfile = T.tempfile
threading = T.threading


@unittest.skipUnless(os.environ.get('RWX_LIVE_CONTRACT_COVERAGE_FIXTURE') == '1', 'Opt-in pinned Forge/Go original coverage fixture')
class LiveCoverageTests(T._LiveCoverageTestsFixtures, unittest.TestCase):
    def test_actual_just_passes_setup_skips_abstracts_fresh_execution_and_initial_failure(self):
        self.exercise_original_reports()

    def verify_provider_reports(self, root, circle, first, sha, tmp):
        result=subprocess.run(['python3',str(Path(__file__).resolve().parents[1]/'compare-contract-coverage.py'),'--circle', str(circle), '--rwx', str(first), '--sha', sha, '--feature', 'main','--output',str(Path(tmp)/'parity.json')],text=True,capture_output=True,timeout=30)
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertTrue(json.loads((Path(tmp)/'parity.json').read_text())['verified_parity'])
