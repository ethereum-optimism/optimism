"""Migration-only comparisons using permanent execution fixture mechanics."""
import copy
import importlib.util
import json
from pathlib import Path
import sys
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tests'))
import test_kontrol_build as T
K = T.K
SCRIPTS = T.SCRIPTS
helper = T.helper
os = T.os
shutil = T.shutil
signal = T.signal
subprocess = T.subprocess
tempfile = T.tempfile
time = T.time
_SPEC=importlib.util.spec_from_file_location('compare_kontrol_build', Path(__file__).resolve().parents[1] / 'compare-kontrol-build.py')
C=importlib.util.module_from_spec(_SPEC);_SPEC.loader.exec_module(C)


@unittest.skipUnless(os.environ.get('RWX_LIVE_KONTROL_FIXTURE') == '1', 'Opt-in actual Forge, Just and pinned Kontrol Docker workloads')
class LiveTests(T._LiveTestsFixtures, unittest.TestCase):

    def test_actual_two_variants_fresh_build_future_proof_discovery_and_strict_parity(self):
        pair = self.exercise_complete_workloads()
        report = C.compare(pair, self.sha)
        self.assertTrue(report['verified_parity'])
        self.assertIn(K.PROOFS + '/FutureProof.sol', report['selection']['proof_sources'])
        self.assertEqual(report['coverage']['tests'], 0)
        original = C.compiler(pair / 'rwx/initial-compiler')
        summaries = C.compiler(pair / 'rwx/summaries-compiler')
        retained = report['compiler_phases']['summaries']['verified_retained_previous_artifacts']['rwx']
        self.assertTrue(retained)
        retained_path = retained[0]['path']
        self.assertEqual(original[retained_path], summaries[retained_path])
        histories = ([original], [original])
        changed = dict(summaries)
        value = json.loads(changed[retained_path])
        value['bytecode']['object'] = '0xdeadbeef'
        changed[retained_path] = json.dumps(value).encode()
        with self.assertRaisesRegex(ValueError, 'retained'):
            C.compare_compiler(summaries, changed, [str(self.root)] * 2, histories)
        added = dict(summaries)
        added[retained_path.removesuffix('.json') + '.extra.json'] = summaries[retained_path]
        with self.assertRaisesRegex(ValueError, 'retained'):
            C.compare_compiler(summaries, added, [str(self.root)] * 2, histories)
        missing = dict(summaries)
        missing.pop(retained_path)
        with self.assertRaisesRegex(ValueError, 'inventories'):
            C.compare_compiler(summaries, missing, [str(self.root)] * 2, histories)
        with self.assertRaisesRegex(ValueError, 'unbound'):
            C.compare_compiler(summaries, summaries, [str(self.root)] * 2)
        cache_name = 'packages/contracts-bedrock/cache/solidity-files-cache.json'
        cache = json.loads(original[cache_name])
        binding = cache['files']['scripts/Artifacts.s.sol']['artifacts']['Artifacts']['0.8.28']['default']
        old = 'packages/contracts-bedrock/forge-artifacts/' + binding['path']
        alternate = 'Artifacts.s.sol/Artifacts.0.8.28.default.json'
        if binding['path'] == alternate:
            alternate = 'Artifacts.s.sol/Artifacts.0.8.28.json'
        new = 'packages/contracts-bedrock/forge-artifacts/' + alternate
        renamed = dict(original)
        renamed[new] = renamed.pop(old)
        binding['path'] = alternate
        renamed[cache_name] = json.dumps(cache).encode()
        aliases = C.compare_compiler(original, renamed, [str(self.root)] * 2)['verified_artifact_path_aliases']
        self.assertEqual(len(aliases), 1)
        missing = dict(renamed)
        missing.pop(new)
        with self.assertRaises(ValueError):
            C.compare_compiler(original, missing, [str(self.root)] * 2)
        extra = dict(renamed)
        extra[old] = renamed[new]
        with self.assertRaisesRegex(ValueError, 'unbound'):
            C.compare_compiler(original, extra, [str(self.root)] * 2)
        corrupt = dict(renamed)
        value = json.loads(corrupt[new])
        value['bytecode']['object'] = '0xdeadbeef'
        corrupt[new] = json.dumps(value).encode()
        with self.assertRaisesRegex(ValueError, 'runtime'):
            C.compare_compiler(original, corrupt, [str(self.root)] * 2)
        wrong = copy.deepcopy(cache)
        artifact = wrong['files']['scripts/Artifacts.s.sol']['artifacts']['Artifacts']
        artifact['0.8.29'] = artifact.pop('0.8.28')
        mismatch = dict(renamed)
        mismatch[cache_name] = json.dumps(wrong).encode()
        with self.assertRaises(ValueError):
            C.compare_compiler(original, mismatch, [str(self.root)] * 2)
        duplicate = copy.deepcopy(cache)
        duplicate['files']['scripts/Artifacts.s.sol']['artifacts']['OtherContract'] = {'0.8.28': {'default': binding}}
        mismatch = dict(renamed)
        mismatch[cache_name] = json.dumps(duplicate).encode()
        with self.assertRaises(ValueError):
            C.compare_compiler(original, mismatch, [str(self.root)] * 2)
        old_cache = json.loads(original[cache_name])
        source = 'test/kontrol/proofs/utils/DeploymentSummary.sol'
        name = 'DeploymentSummary'
        versions = old_cache['files'][source]['artifacts'][name]
        version = next(iter(versions))
        old_binding = versions[version]['default']
        old_path = 'packages/contracts-bedrock/forge-artifacts/' + old_binding['path']
        qualified = 'DeploymentSummary.sol/DeploymentSummary.' + version + '.json'
        retained_path = 'packages/contracts-bedrock/forge-artifacts/' + qualified
        prior = dict(original)
        prior[retained_path] = prior.pop(old_path)
        old_binding['path'] = qualified
        prior[cache_name] = json.dumps(old_cache).encode()
        rebuilt = C.compiler(pair / 'rwx/proofs-compiler')
        self.assertNotEqual(rebuilt[old_path], prior[retained_path])
        rebuilt[retained_path] = prior[retained_path]
        historical = C.compare_compiler(rebuilt, rebuilt, [str(self.root)] * 2, ([prior], [prior]))
        self.assertEqual(historical['contract_artifacts'], len(C.compiler_bindings(rebuilt, [prior])['bindings']) + 1)
        self.assertEqual(historical['verified_retained_previous_artifacts']['rwx'][0]['bound_phase'], 0)
        continued = C.compare_compiler(rebuilt, rebuilt, [str(self.root)] * 2, ([prior, rebuilt], [prior, rebuilt]))
        self.assertEqual(continued['verified_retained_previous_artifacts']['rwx'][0]['bound_phase'], 0)
        bad = dict(rebuilt)
        bad[retained_path] = rebuilt[old_path]
        with self.assertRaisesRegex(ValueError, 'retained'):
            C.compare_compiler(rebuilt, bad, [str(self.root)] * 2, ([prior], [prior]))
        bad = dict(rebuilt)
        bad.pop(retained_path)
        with self.assertRaisesRegex(ValueError, 'retained compiler inventories'):
            C.compare_compiler(rebuilt, bad, [str(self.root)] * 2, ([prior], [prior]))
        bad = dict(rebuilt)
        bad[retained_path.removesuffix('.json') + '.extra.json'] = bad[retained_path]
        with self.assertRaisesRegex(ValueError, 'retained'):
            C.compare_compiler(rebuilt, bad, [str(self.root)] * 2, ([prior], [prior]))
        path = pair / 'rwx/runtime-image.json'
        original = path.read_bytes()
        final_path = pair / 'rwx/final.json'
        original_final = final_path.read_bytes()
        image = json.loads(original)
        selected = K.IMAGE.selection()
        for identity in (selected['config_digest'], selected['image'].split('@', 1)[1], 'sha256:' + 'f' * 64):
            image[0]['Id'] = identity
            K.S.write(path, image)
            final = K.G.read(final_path)
            final['original_sha256']['runtime-image.json'] = K.S.digest(path)
            K.S.write(final_path, final)
            if identity.endswith('f' * 64):
                with self.assertRaises(ValueError):
                    C.compare(pair, self.sha)
            else:
                self.assertTrue(C.compare(pair, self.sha)['verified_parity'])
        path.write_bytes(original)
        final_path.write_bytes(original_final)
        for name in ('settings.json', 'selection.json', 'coverage.json', 'proofs.stage.json', 'dependencies/contracts-kontrol/metadata.json'):
            path = pair / 'rwx' / name
            original = path.read_bytes()
            final_path = pair / 'rwx/final.json'
            original_final = final_path.read_bytes()
            value = K.G.read(path)
            if name == 'settings.json':
                value['source_sha'] = 'f' * 40
            elif name == 'selection.json':
                value['proof_sources'].pop(K.PROOFS + '/FutureProof.sol')
            elif name == 'coverage.json':
                value['variants'] = ['default']
            elif name == 'proofs.stage.json':
                value['argv'] = ['true']
            else:
                value['settings'] = {'profile': 'default'}
            K.S.write(path, value)
            final = K.G.read(final_path)
            final['original_sha256'][name] = K.S.digest(path)
            K.S.write(final_path, final)
            with self.subTest(name=name), self.assertRaises(ValueError):
                C.compare(pair, self.sha)
            path.write_bytes(original)
            final_path.write_bytes(original_final)
        path = pair / 'rwx/variants/fault-proofs/generated/files' / K.GENERATED['fault-proofs'][0]
        original = path.read_bytes()
        path.write_bytes(original + b'// actual changed generated file\n')
        with self.assertRaises(ValueError):
            C.compare(pair, self.sha)
