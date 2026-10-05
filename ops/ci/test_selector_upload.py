#!/usr/bin/env python3
"""Validate selector evidence against retained real Forge/compiler originals."""
import copy
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('upload', Path(__file__).with_name('selector-upload.py'))
UP = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(UP)
GOLD = Path(__file__).parent / 'fixtures/selector-upload'


def oracle(name='abi-oracle'):
    root = GOLD / name / 'project'
    cache = UP.read(root / 'cache/solidity-files-cache.json')
    units = {p.stem: UP.read(p) for p in (root / 'out/build-info').glob('*.json')}
    artifacts = {str(p.relative_to(root / 'out')): UP.read(p)
                 for p in (root / 'out').rglob('*.json') if p.parent.name != 'build-info'}
    return cache, units, artifacts


def expectation():
    hashes = UP.read(GOLD / 'live-import/fixture.json')['signatures']
    return {'hash32': hashes, 'signatures': {
        'function': {name: value[:10] for name, value in hashes.items() if name != 'Pong(uint256)'},
        'event': {'Pong(uint256)': hashes['Pong(uint256)']}}}


class SelectorTests(unittest.TestCase):
    def test_real_complete_oracle_keeps_empty_declarations_and_tuple_abi(self):
        result = UP.catalog(*oracle())
        self.assertEqual(result['selected'], ['src/Fixture.sol:Fixture:0.8.28:default'])
        self.assertEqual(result['compiled_sources'], ['src/Fixture.sol'])
        empty = result['declarations']['src/Fixture.sol:Empty:0.8.28:default']
        self.assertEqual(empty['selection'], 'empty-selector-abi')
        self.assertEqual(empty['abi'], [])
        selected = result['declarations'][result['selected'][0]]
        self.assertEqual({r['signature'] for r in selected['signatures']},
                         {'ping(uint256)', 'transform((uint256,address))', 'Problem(uint256)', 'Pong(uint256)'})

    def test_missing_extra_corrupt_or_wrong_compiler_inputs_fail(self):
        cache, units, artifacts = oracle()
        def missing_artifact(c, u, a): a.pop('Fixture.sol/Empty.json')
        def extra_artifact(c, u, a): a['foreign.json'] = copy.deepcopy(a['Fixture.sol/Empty.json'])
        def missing_unit(c, u, a): u.clear()
        def unknown_profile(c, u, a): c['profiles'].clear()
        def stale_version(c, u, a): next(iter(u.values()))['solcVersion'] = '0.8.27'
        def corrupt_abi(c, u, a): a['Fixture.sol/Fixture.json']['abi'][0]['name'] = 'changed'
        def foreign_source(c, u, a): a['Fixture.sol/Fixture.json']['id'] = 99
        def not_compiled(c, u, a): c['files']['src/Fixture.sol']['seenByCompiler'] = False
        def duplicate_unit(c, u, a): c['builds'].append(c['builds'][0])
        for mutation in (missing_artifact, extra_artifact, missing_unit, unknown_profile, stale_version,
                         corrupt_abi, foreign_source, not_compiled, duplicate_unit):
            inputs = copy.deepcopy((cache, units, artifacts))
            mutation(*inputs)
            with self.subTest(mutation=mutation.__name__), self.assertRaises((ValueError, KeyError)):
                UP.catalog(*inputs)

    def test_real_multiple_jobs_preserve_artifact_ids_and_bind_each_exact_unit(self):
        cache, units, artifacts = oracle('multi-profile-oracle')
        result = UP.catalog(cache, units, artifacts)
        shared = result['declarations']['src/Shared.sol:Shared:0.8.28:default']
        self.assertEqual((shared['artifact_source_id'], shared['unit_source_id']), (0, 1))
        self.assertEqual(len(result['declarations']), 4)
        self.assertEqual(len(result['selected']), 2)
        unit = units[shared['build_id']]
        unit['source_id_to_path']['1'] = 'foreign.sol'
        with self.assertRaisesRegex(ValueError, 'compiler unit source map'):
            UP.catalog(cache, units, artifacts)

    def test_source_and_test_exclusions_preserve_compiled_declarations(self):
        cache, units, artifacts = oracle()
        def move(old, new):
            row = cache['files'].pop(old)
            row['sourceName'] = new
            cache['files'][new] = row
            for unit in units.values():
                unit['input']['sources'][new] = unit['input']['sources'].pop(old)
                unit['output']['contracts'][new] = unit['output']['contracts'].pop(old)
                unit['output']['sources'][new] = unit['output']['sources'].pop(old)
                unit['source_id_to_path'] = {k: new if v == old else v for k, v in unit['source_id_to_path'].items()}
        for path, reason in [('lib/Fixture.sol', 'outside-source-path'), ('src/Fixture.t.sol', 'test-source')]:
            old = next(iter(cache['files']))
            move(old, path)
            # An entirely excluded project cannot prove a publisher workload.
            with self.subTest(path=path), self.assertRaisesRegex(ValueError, 'empty selector workload'):
                UP.catalog(cache, units, artifacts)

    def test_nested_tuple_arrays_and_constructor_do_not_change_selector_scope(self):
        parameter = {'type': 'tuple[][2]', 'components': [{'type': 'uint256'},
                     {'type': 'tuple', 'components': [{'type': 'bytes32[]'}, {'type': 'address'}]}]}
        self.assertEqual(UP.canonical_type(parameter), '(uint256,(bytes32[],address))[][2]')
        self.assertEqual(UP.signatures([{'type': 'constructor'}, {'type': 'receive'}, {'type': 'fallback'}]), [])
        for value in ({'type': 'uint'}, {'type': 'tuple'}, {'type': 'tuple[bad]', 'components': []}):
            with self.subTest(value=value), self.assertRaises(ValueError): UP.canonical_type(value)

    def test_original_seal_rejects_typed_counter_errors_and_nested_mutation(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / 'prepared').mkdir()
            UP.R.write(root / 'prepared/final.json', {'state': 'compiled', 'tests': 0})
            UP.seal(root, 'passed', 0, [], 1)
            UP.original(root, 'passed')
            for key, value in [('tests', False), ('uploads', True), ('uploads', 0), ('exit_code', False),
                               ('errors', ['original failure'])]:
                UP.seal(root, 'passed', 0, [], 1)
                final = UP.read(root / 'final.json'); final[key] = value
                UP.R.write(root / 'final.json', final)
                with self.subTest(key=key, value=value), self.assertRaises(ValueError): UP.original(root, 'passed')
            UP.seal(root, 'passed', 0, [], 1)
            (root / 'prepared/final.json').write_text('{}')
            with self.assertRaises(ValueError): UP.original(root, 'passed')
            UP.seal(root, 'compiled', 0, [])
            (root / 'extra.json').write_text('{}')
            with self.assertRaises(ValueError): UP.original(root, 'compiled')
            (root / 'link').symlink_to(root / 'extra.json')
            with self.assertRaises(ValueError): UP.files(root)

    def test_nonfinite_and_duplicate_original_json_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / 'input.json'
            for text in ('{"key":1,"key":2}', '{"key":NaN}', '{"key":Infinity}'):
                path.write_text(text)
                with self.subTest(text=text), self.assertRaises(ValueError): UP.read(path)

    def test_real_first_and_duplicate_imports_are_fully_accounted_and_denial_fails(self):
        wanted = expectation()
        for name, state in [('first-upload', 'imported'), ('duplicate-upload', 'duplicated')]:
            result = UP.imports(GOLD / 'live-import' / name, wanted)
            self.assertEqual(result['requests'], 1)
            for kind in ('function', 'event'):
                self.assertEqual(result['outcomes'][kind][state], wanted['signatures'][kind])
        with self.assertRaisesRegex(ValueError, 'Failed, foreign or incomplete'):
            UP.imports(GOLD / 'live-import/denied-write', wanted)

    def test_corrupt_missing_extra_repeated_or_foreign_original_posts_fail(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'original'
            def restore():
                if root.exists(): shutil.rmtree(root)
                shutil.copytree(GOLD / 'live-import/first-upload', root)
            restore()
            (root / 'http-0-request.json').write_text('{}')
            with self.assertRaisesRegex(ValueError, 'Corrupt original'): UP.imports(root, expectation())
            restore()
            (root / 'http-0.json').unlink()
            with self.assertRaisesRegex(ValueError, 'No actual'): UP.imports(root, expectation())
            restore()
            for suffix in ('.json', '-request.json', '-response.json'):
                shutil.copy2(root / ('http-0' + suffix), root / ('http-1' + suffix))
            with self.assertRaisesRegex(ValueError, 'repeated'): UP.imports(root, expectation())
            restore()
            row = UP.read(root / 'http-0.json'); row['host'] = 'unexpected.invalid'
            UP.R.write(root / 'http-0.json', row)
            with self.assertRaisesRegex(ValueError, 'foreign'): UP.imports(root, expectation())
            restore()
            wanted = expectation(); wanted['signatures']['function']['omitted()'] = '0x00000000'
            with self.assertRaisesRegex(ValueError, 'complete compiler'): UP.imports(root, wanted)

    def test_real_git_bytes_modes_and_symlinks_ignore_restored_mtimes(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                return subprocess.check_output(['git', '-C', str(root), '-c', 'user.name=fixture',
                    '-c', 'user.email=fixture@example.invalid', '-c', 'commit.gpgsign=false', *args], stderr=subprocess.DEVNULL).decode().strip()
            git('init', '-q')
            script = root / 'input.sh'; script.write_text('original'); script.chmod(0o755)
            (root / 'link').symlink_to('input.sh')
            git('add', '.'); git('commit', '-qm', 'original fixture'); sha = git('rev-parse', 'HEAD')
            inputs = UP.R.source_inputs(root, sha)
            self.assertEqual(inputs['input.sh']['mode'], '100755')
            self.assertEqual(inputs['link']['sha256'], UP.R.hashlib.sha256(b'input.sh').hexdigest())
            before = script.stat(); script.write_text('modified'); os.utime(script, ns=(before.st_atime_ns, before.st_mtime_ns))
            with self.assertRaisesRegex(ValueError, 'Changed committed'): UP.R.source_inputs(root, sha)
            script.write_text('original'); script.chmod(0o644)
            with self.assertRaisesRegex(ValueError, 'executable mode'): UP.R.source_inputs(root, sha)
            script.chmod(0o755); (root / 'link').unlink(); (root / 'link').symlink_to('foreign.sh')
            with self.assertRaisesRegex(ValueError, 'Changed committed'): UP.R.source_inputs(root, sha)


if __name__ == '__main__':
    unittest.main()
