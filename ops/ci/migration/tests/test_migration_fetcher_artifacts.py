"""Migration-only comparisons using permanent execution fixture mechanics."""
import importlib.util
from pathlib import Path
import sys
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tests'))
import test_fetcher_artifacts as T
F = T.F
SCRIPTS = T.SCRIPTS
json = T.json
os = T.os
shutil = T.shutil
signal = T.signal
subprocess = T.subprocess
tempfile = T.tempfile
time = T.time
_SPEC=importlib.util.spec_from_file_location('compare_fetcher_artifacts', Path(__file__).resolve().parents[1] / 'compare-fetcher-artifacts.py')
C=importlib.util.module_from_spec(_SPEC);_SPEC.loader.exec_module(C)


@unittest.skipUnless(os.environ.get('RWX_LIVE_FETCHER_FIXTURE') == '1', 'Opt-in actual Forge, Just and diff workloads')
class LiveTests(T._LiveTestsFixtures, unittest.TestCase):

    def test_complete_actual_build_preserves_future_artifacts_and_original_parity(self):
        paired = self.root / '.ci/paired'
        for provider, path in [('circleci', 'circle'), ('rwx', 'rwx')]:
            result = self.execute(provider)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.retain('complete-' + path)
            shutil.copytree(self.report, paired / path)
        result = C.compare(paired, self.sha)
        self.assertTrue(result['verified_parity'])
        self.assertEqual(result['coverage']['artifact_files'], 2)
        self.assertIn('FutureContract.json', result['selection']['committed_sha256'])
        for name in ('settings.json', 'selection.json', 'coverage.json', 'build.stage.json'):
            path = paired / 'rwx' / name
            original = path.read_bytes()
            value = F.G.read(path)
            if name == 'settings.json':
                value['source_sha'] = '0' * 40
            if name == 'selection.json':
                value['committed_sha256'].pop('FutureContract.json')
            if name == 'coverage.json':
                value['artifact_files'] = 1
            if name == 'build.stage.json':
                value['argv'] = ['true']
            F.S.write(path, value)
            final_path = paired / 'rwx/final.json'
            final = F.G.read(final_path)
            original_final = final_path.read_bytes()
            final['original_sha256'][name] = F.S.digest(path)
            F.S.write(final_path, final)
            with self.subTest(name=name), self.assertRaises(ValueError):
                C.compare(paired, self.sha)
            path.write_bytes(original)
            final_path.write_bytes(original_final)
