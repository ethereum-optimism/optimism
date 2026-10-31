"""Migration-only comparisons using permanent execution fixture mechanics."""
import importlib.util
from pathlib import Path
import sys
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tests'))
import test_nut_prefork as T
M = T.M
N = T.N
SCRIPTS = T.SCRIPTS
json = T.json
module = T.module
os = T.os
shutil = T.shutil
signal = T.signal
subprocess = T.subprocess
tempfile = T.tempfile
time = T.time
_SPEC=importlib.util.spec_from_file_location('compare_nut_prefork', Path(__file__).resolve().parents[1] / 'compare-nut-prefork.py')
COMPARE=importlib.util.module_from_spec(_SPEC);_SPEC.loader.exec_module(COMPARE)


@unittest.skipUnless(os.environ.get('RWX_LIVE_NUT_PREFORK_FIXTURE') == '1', 'Opt-in real Go, Forge and original Just recipes')
class LiveTests(T._LiveTestsFixtures, unittest.TestCase):

    def test_real_complete_fresh_execution_original_selection_and_strict_parity(self):
        paired = self.root / '.ci/paired'
        for provider, key in [('circleci', 'circle'), ('rwx', 'rwx')]:
            result = self.run_fixture(provider)
            self.assertEqual(result.returncode, 0, result.stdout)
            shutil.copytree(self.report, paired / key)
            self.retain('fresh-' + key)
        self.assertEqual((self.root / '.ci/fresh-marker').read_text().splitlines(), ['karst', 'lagoon', 'new-fork'] * 2)
        result = COMPARE.compare(paired, self.sha)
        self.assertTrue(result['verified_parity'])
        self.assertEqual(result['outcomes'], {'pass': 9})
        self.assertEqual(result['selection']['initial_tests'][N.PACKAGE], ['TestGenerateForkState', 'TestGenerateForkStateNew'])
        for change in ('source', 'command', 'omission', 'coverage', 'bundle', 'dependency'):
            original = {}
            for provider in ('circle', 'rwx'):
                directory = paired / provider
                name = {'source': 'settings.json', 'command': 'check.stage.json', 'omission': 'selection.json', 'coverage': 'coverage.json', 'bundle': 'dependencies/superchain-configs.zip', 'dependency': 'dependencies/contracts.json'}.get(change)
                if change == 'dependency' and provider == 'circle':
                    continue
                path = directory / name
                original[provider] = (path, path.read_bytes(), (directory / 'final.json').read_bytes())
                if change == 'bundle':
                    path.write_bytes(b'corrupt original bundle')
                else:
                    value = N.G.read(path)
                    if change == 'source':
                        value['source_sha'] = 'f' * 40
                    elif change == 'command':
                        value['argv'] = ['true']
                    elif change == 'omission':
                        value['forks'] = value['forks'][:-1]
                    elif change == 'coverage':
                        value['attempts_per_fork'] = 2
                    elif change == 'dependency':
                        value['settings'] = {'profile': 'default'}
                    N.S.write(path, value)
                final = N.G.read(directory / 'final.json')
                final['original_sha256'][name] = N.S.digest(path)
                N.S.write(directory / 'final.json', final)
            with self.subTest(change=change), self.assertRaises(ValueError):
                COMPARE.compare(paired, self.sha)
            for provider, (path, data, final) in original.items():
                path.write_bytes(data)
                (paired / provider / 'final.json').write_bytes(final)
