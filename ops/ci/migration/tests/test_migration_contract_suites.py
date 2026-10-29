"""Migration-only comparisons using permanent execution fixture mechanics."""
import importlib.util
from pathlib import Path
import sys
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tests'))
import test_contract_suites as T
CS = T.CS
json = T.json
os = T.os
patch = T.patch
shutil = T.shutil
subprocess = T.subprocess
tempfile = T.tempfile


@unittest.skipUnless(os.environ.get('RWX_LIVE_CONTRACT_SUITE_FIXTURE') == '1', 'Opt-in pinned Forge and production Go-validator fixture')
class LiveSuiteTests(T._LiveSuiteTestsFixtures, unittest.TestCase):
    def test_actual_find_changed_files_compilation_fresh_ffi_failures_and_parity(self):
        self.exercise_original_reports()

    def verify_provider_reports(self, root, directories, sha, suite, tmp):
        result=subprocess.run(['python3',str(Path(__file__).resolve().parents[1]/'compare-contract-suites.py'),'--circle', str(directories['circleci']), '--rwx', str(directories['rwx']), '--sha', sha, '--suite', suite, '--feature', 'main','--output',str(Path(tmp)/'parity.json')],text=True,capture_output=True,timeout=30)
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertTrue(json.loads((Path(tmp)/'parity.json').read_text())['verified_parity'])
