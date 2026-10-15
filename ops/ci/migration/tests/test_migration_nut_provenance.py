"""Migration-only comparisons using permanent execution fixture mechanics."""
import importlib.util
from pathlib import Path
import sys
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tests'))
import test_nut_provenance as T
N = T.N
base64 = T.base64
copy = T.copy
json = T.json
os = T.os
patch = T.patch
preserve_working_directory = T.preserve_working_directory
shutil = T.shutil
signal = T.signal
subprocess = T.subprocess
tempfile = T.tempfile
threading = T.threading
time = T.time
_SPEC=importlib.util.spec_from_file_location('compare_nut_provenance', Path(__file__).resolve().parents[1] / 'compare-nut-provenance.py')
COMPARE=importlib.util.module_from_spec(_SPEC);_SPEC.loader.exec_module(COMPARE)


class ProvenanceTests(T._ProvenanceTestsFixtures, unittest.TestCase):

    def test_complete_originals_compare_all_gitlinks_and_actual_generated_bytes(self):
        a = self.report(self.temp / 'circle', 'circleci')
        b = self.report(self.temp / 'native', 'rwx')
        out = self.temp / 'comparison.json'
        self.assertEqual(COMPARE.compare(a, b, out), 0)
        coverage = N.G.read(out)['coverage']['karst']
        self.assertEqual(coverage['submodules'], 3)
        self.assertEqual(coverage['outcome'], 'passed')
        self.assertEqual(coverage['tests'], 0)
        (a / 'forks/karst/originals/worktree-status.txt').unlink()
        self.assertEqual(COMPARE.compare(a, b, out), 0)
        (b / 'forks/karst/originals/worktree-status.txt').unlink()
        self.assertEqual(COMPARE.compare(a, b, out), 1)

    def test_full_selection_retains_unused_base_observations_and_rejects_incomplete_replays(self):
        a = self.report(self.temp / 'circle', 'circleci')
        self.git(self.root, 'update-ref', '-d', 'refs/remotes/origin/develop')
        b = self.report(self.temp / 'native', 'rwx')
        out = self.temp / 'comparison.json'
        self.assertEqual(COMPARE.compare(a, b, out), 0)
        original = N.G.read(b / 'selection.json')
        record = N.G.read(out)['base_discovery']
        self.assertTrue(record['circleci']['base_lock_available'])
        self.assertFalse(record['rwx']['base_lock_available'])
        self.assertFalse(record['circleci']['affects_full_selection'])
        self.assertFalse(record['rwx']['affects_full_selection'])
        self.assertEqual(record['rwx']['stderr_sha256'], N.S.digest(b / 'base-discovery.stderr'))
        for changed in (original | {'mode': 'changed'}, original | {'selected_forks': []}, original | {'selected_forks': ['karst', 'karst']}, original | {'excluded': [{'fork': 'karst'}]}, original | {'base_lock_available': True}, original | {'base_revision': 'foreign-ref'}, original | {'base_lock_exit_code': -15}, original | {'base_lock_exit_code': 256}, original | {'base_lock_available': 1}, original | {'base_lock_available': True, 'base_lock_exit_code': 0, 'base_revision': None}, original | {'entries': {'karst': original['entries']['karst'] | {'commit': 'c' * 40}}}):
            N.S.write(b / 'selection.json', changed)
            self.reseal(b)
            with self.subTest(changed=changed):
                self.assertEqual(COMPARE.compare(a, b, out), 1)
        N.S.write(b / 'selection.json', original)
        self.reseal(b)
        self.assertEqual(COMPARE.compare(a, b, out), 0)

    def test_resealed_generation_config_and_same_provider_drift_cannot_pass(self):
        a = self.report(self.temp / 'circle', 'circleci')
        b = self.report(self.temp / 'native', 'rwx')
        out = self.temp / 'comparison.json'
        for name in ('regenerated-bundle.json', 'contracts.justfile', 'foundry.toml', 'mise.toml'):
            path = b / 'forks/karst/originals' / name
            original = path.read_bytes()
            path.write_bytes(original + b' ')
            self.reseal(b)
            self.assertEqual(COMPARE.compare(a, b, out), 1)
            path.write_bytes(original)
            self.reseal(b)
        settings = N.G.read(b / 'settings.json')
        N.S.write(b / 'settings.json', settings | {'source_sha': 'c' * 40})
        self.reseal(b)
        self.assertEqual(COMPARE.compare(a, b, out), 1)
