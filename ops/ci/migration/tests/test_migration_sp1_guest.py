"""Migration-only comparisons using permanent execution fixture mechanics."""
import importlib.util
from pathlib import Path
import sys
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tests'))
import test_sp1_guest as T
G = T.G
json = T.json
os = T.os
patch = T.patch
re = T.re
shutil = T.shutil
signal = T.signal
subprocess = T.subprocess
tempfile = T.tempfile
time = T.time


class MigrationEvidenceTests(unittest.TestCase):
    def test_only_circle_empty_logs_are_recovered_and_audited(self):
        spec = importlib.util.spec_from_file_location('compare_sp1', Path(__file__).resolve().parents[1] / 'compare-sp1-guest.py')
        compare = importlib.util.module_from_spec(spec); spec.loader.exec_module(compare)
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp); empty = []
            final = {'original_sha256':{'empty.log':G.hashlib.sha256(b'').hexdigest()}}
            G.S.write(directory/'final.json',final)
            with self.assertRaises((ValueError, FileNotFoundError)): compare.verify_seals(directory,final,False,empty)
            self.assertFalse((directory/'empty.log').exists())
            compare.verify_seals(directory,final,True,empty)
            self.assertEqual((directory/'empty.log').read_bytes(),b'')
            self.assertEqual(empty,[{'path':'empty.log','sha256':G.hashlib.sha256(b'').hexdigest(),'report':str(directory)}])
            (directory/'empty.log').unlink()
            final['original_sha256']={'empty.json':G.hashlib.sha256(b'').hexdigest()}
            with self.assertRaises((ValueError, FileNotFoundError)): compare.verify_seals(directory,final,True,[])
            self.assertFalse((directory/'empty.json').exists())


@unittest.skipUnless(os.environ.get('RWX_LIVE_SP1_FIXTURE') == '1', 'Actual Cargo discovery, execution and production ELF fixtures')
class LiveTests(T._LiveTestsFixtures, unittest.TestCase):

    @unittest.skipUnless(os.environ.get('SP1_GUEST_REPORT'), 'Complete real SP1 original reports are required')
    def test_comparer_rejects_resealed_missing_cases_dependency_graphs_and_wrong_toolchains(self):
        spec = importlib.util.spec_from_file_location('compare_sp1', Path(__file__).resolve().parents[1] / 'compare-sp1-guest.py')
        compare = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(compare)
        source = Path(os.environ['SP1_GUEST_REPORT'])
        settings = G.read(source / 'settings.json')
        directory = self.root / 'comparison'
        for name, provider in [('circle', 'circleci'), ('rwx', 'rwx')]:
            target = directory / name
            shutil.copytree(source, target)
            for path in (target, target / 'producer'):
                value = G.read(path / 'settings.json')
                value['provider'] = provider
                value['rwx_run_id'] = 'isolated-comparer-fixture'
                value['rwx_task_attempt'] = '1'
                G.S.write(path / 'settings.json', value)
                if provider == 'circleci':
                    for stage in compare.CACHE:
                        for file in path.glob(stage + '.*'):
                            file.unlink()
            self.reseal(target / 'producer')
            self.reseal(target)
        self.assertTrue(compare.compare(directory, settings['source_sha'])['verified_parity'])
        target = directory / 'rwx'
        coverage = target / 'guest-coverage.json'
        original = coverage.read_bytes()
        value = G.read(coverage)
        value['cases'] = value['cases'][1:]
        G.S.write(coverage, value)
        self.reseal(target)
        with self.assertRaisesRegex(ValueError, 'case report'):
            compare.compare(directory, settings['source_sha'])
        coverage.write_bytes(original)
        self.reseal(target)
        metadata = target / 'guest-workspace.json'
        original = metadata.read_bytes()
        value = G.read(metadata)
        value['packages'][0]['version'] = '9.9.9'
        G.S.write(metadata, value)
        self.reseal(target)
        with self.assertRaisesRegex(ValueError, 'dependency graphs'):
            compare.compare(directory, settings['source_sha'])
        metadata.write_bytes(original)
        self.reseal(target)
        cache = target / 'cache-settings.json'
        original = cache.read_bytes()
        value = G.read(cache)
        value['compiler_input_sha256'] = '0' * 64
        G.S.write(cache, value)
        self.reseal(target)
        with self.assertRaisesRegex(ValueError, 'compiler cache identity'):
            compare.compare(directory, settings['source_sha'])
        cache.write_bytes(original)
        self.reseal(target)
        toolchain = target / 'toolchain/tools.json'
        value = G.read(toolchain)
        value['succinct-rustc-sha256'] = '0' * 64
        G.S.write(toolchain, value)
        self.reseal(target / 'toolchain')
        self.reseal(target)
        with self.assertRaisesRegex(ValueError, 'compiler identity'):
            compare.compare(directory, settings['source_sha'])
        G.S.write(self.reports / 'comparer-fixture.json', {'source_sha': settings['source_sha'], 'authority': 'Complete real workload originals; provider aliases are isolated verifier fixtures only', 'resealed_mutations_rejected': ['missing actual case', 'dependency version', 'compiler cache identity', 'actual compiler hash']})
