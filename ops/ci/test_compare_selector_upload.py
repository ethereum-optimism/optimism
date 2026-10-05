#!/usr/bin/env python3
"""Exercise selector comparison boundaries using retained real tool originals."""
import importlib.util
import json
from pathlib import Path
import shutil
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('compare_selectors', Path(__file__).with_name('compare-selector-upload.py'))
C = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(C)
GOLD = Path(__file__).parent / 'fixtures/selector-upload'


class ComparisonTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)

    def original(self, name='multi-profile-oracle'):
        source = GOLD / name / 'project'
        root = self.root / name
        root.mkdir()
        shutil.copy2(source / 'cache/solidity-files-cache.json', root / 'compiler-cache.json')
        shutil.copytree(source / 'out/build-info', root / 'compiler-units')
        shutil.copytree(source / 'out', root / 'compiler-artifacts', ignore=shutil.ignore_patterns('build-info'))
        self.selection(root)
        return root

    def selection(self, root):
        cache = C.read(root / 'compiler-cache.json')
        units = {p.stem: C.read(p) for p in (root / 'compiler-units').glob('*.json')}
        artifacts = {str(p.relative_to(root / 'compiler-artifacts')): C.read(p)
                     for p in (root / 'compiler-artifacts').rglob('*.json')}
        C.R.write(root / 'selection.json', C.U.catalog(cache, units, artifacts))

    def test_actual_multiple_compilers_keep_every_source_abi_and_profile(self):
        root = self.original()
        value = C.compiler(root, '/var/mint-workspace')
        self.assertEqual(len(value['units']), 2)
        self.assertEqual(len(value['selection']['declarations']), 4)
        self.assertEqual(len(value['selection']['selected']), 2)
        self.assertEqual(value['selection']['compiled_sources'], ['src/AUnrestricted.sol', 'src/Shared.sol', 'src/ZRestricted.sol'])

    def test_original_mtimes_and_valid_profile_completion_order_are_not_outcomes(self):
        root = self.original()
        before = C.compiler(root, '/var/mint-workspace')
        cache = C.read(root / 'compiler-cache.json')
        for row in cache['files'].values(): row['lastModificationDate'] += 100000
        C.R.write(root / 'compiler-cache.json', cache)
        for path in (root / 'compiler-artifacts/Shared.sol').glob('*.json'):
            value = C.read(path); value['id'] = 0
            C.R.write(path, value)
        self.selection(root)
        self.assertEqual(before, C.compiler(root, '/var/mint-workspace'))
        path = root / 'compiler-artifacts/Shared.sol/Shared.json'
        value = C.read(path); value['id'] = 99; C.R.write(path, value)
        with self.assertRaisesRegex(ValueError, 'Corrupt compiler ABI binding'):
            C.compiler(root, '/var/mint-workspace')

    def test_complete_compiler_inputs_and_unselected_outputs_affect_comparison(self):
        root = self.original('abi-oracle')
        original = C.compiler(root, '/var/mint-workspace')
        path = next((root / 'compiler-units').glob('*.json'))
        value = C.read(path)
        value['input']['sources']['src/Fixture.sol']['content'] += '\n// changed input\n'
        C.R.write(path, value)
        self.selection(root)
        self.assertNotEqual(original, C.compiler(root, '/var/mint-workspace'))
        value['output']['contracts']['src/Fixture.sol']['Empty']['abi'] = [{'type': 'fallback', 'stateMutability': 'nonpayable'}]
        C.R.write(path, value)
        with self.assertRaisesRegex(ValueError, 'Corrupt compiler ABI binding'):
            C.compiler(root, '/var/mint-workspace')

    def test_missing_unit_or_resealed_derived_selection_is_rejected(self):
        root = self.original()
        path = root / 'selection.json'; value = C.read(path); value['selected'].pop()
        C.R.write(path, value)
        with self.assertRaisesRegex(ValueError, 'Changed original complete ABI selection'):
            C.compiler(root, '/var/mint-workspace')
        self.selection(root)
        next((root / 'compiler-units').glob('*.json')).unlink()
        with self.assertRaisesRegex(ValueError, 'Incomplete compiler ABI inventory'):
            C.compiler(root, '/var/mint-workspace')

    def test_only_actual_workspace_path_prefixes_are_normalized(self):
        value = {'path': '/provider/source.sol', 'root': '/provider', 'other': '/provider-extra/source.sol',
                 'message': 'unchanged /provider/source.sol', 'url': 'https://example.invalid/provider',
                 'nested': ['/provider/lib', {'name': 'provider'}]}
        self.assertEqual(C.normalized(value, '/provider'), value | {'path': '$WORKSPACE/source.sol',
            'root': '$WORKSPACE', 'nested': ['$WORKSPACE/lib', {'name': 'provider'}]})

    def test_circle_alias_is_bound_to_the_isolated_original_workflow(self):
        source = {'workflows': {'selector-upload-replay': {'jobs': [
            {'contracts-bedrock-upload': {'selector_shadow': True}}]}}}
        for alias in ('contracts-bedrock-upload', 'contracts-bedrock-upload-1'):
            publisher = {'steps': [{'run': {'command': 'original isolated publisher'}}]}
            compiled = {'workflows': {'selector-upload-replay': {'jobs': [{alias: {}}]}},
                        'jobs': {alias: publisher}}
            self.assertEqual(C.circle_replay_job(source, compiled, alias), publisher)
            with self.assertRaisesRegex(ValueError, 'actual Circle replay selection'):
                C.circle_replay_job(source, compiled, 'contracts-bedrock-upload-2')
            with self.assertRaisesRegex(ValueError, 'original Circle replay selection'):
                C.circle_replay_job({'workflows': {'selector-upload-replay': {'jobs': [
                    {'contracts-bedrock-upload': {'selector_shadow': False}}]}}}, compiled, alias)
        for alias in ('production-upload', 'contracts-bedrock-upload-0', 'contracts-bedrock-upload-extra'):
            compiled = {'workflows': {'selector-upload-replay': {'jobs': [{alias: {}}]}},
                        'jobs': {alias: {'steps': []}}}
            with self.subTest(alias=alias), self.assertRaisesRegex(ValueError, 'actual Circle replay selection'):
                C.circle_replay_job(source, compiled, alias)

    def test_real_provider_users_are_unprivileged_and_failure_evidence_is_required(self):
        users = []
        for provider in ('circleci', 'rwx'):
            original = GOLD / 'provider-isolation' / provider
            execution = C.read(original / 'execution.json')
            C.process(original, ['just', 'update-selectors'], execution['cwd'])
            users.append(C.read(original / 'isolation.json')['uid'])
            root = self.root / provider; shutil.copytree(original, root)
            isolation = C.read(root / 'isolation.json'); isolation['uid'] = 0
            C.R.write(root / 'isolation.json', isolation)
            with self.assertRaisesRegex(ValueError, 'destination isolation'):
                C.process(root, ['just', 'update-selectors'], execution['cwd'])
            shutil.copy2(original / 'isolation.json', root / 'isolation.json')
            for key, value in [('exit_code', 1), ('timed_out', True), ('signals', [15])]:
                changed = execution | {key: value}; C.R.write(root / 'execution.json', changed)
                with self.subTest(provider=provider, field=key), self.assertRaisesRegex(ValueError, 'real execution'):
                    C.process(root, ['just', 'update-selectors'], execution['cwd'])
            C.R.write(root / 'execution.json', execution)
            (root / 'stdout.log').write_text('altered original')
            with self.assertRaisesRegex(ValueError, 'real execution'):
                C.process(root, ['just', 'update-selectors'], execution['cwd'])
        self.assertEqual(users, [1001, 1000])

    def test_original_posts_are_compared_completely_despite_array_order(self):
        root = self.root / 'post'
        shutil.copytree(GOLD / 'live-import/first-upload', root)
        fixture = C.read(GOLD / 'live-import/fixture.json')['signatures']
        wanted = {'hash32': fixture, 'signatures': {
            'function': {key: value[:10] for key, value in fixture.items() if key != 'Pong(uint256)'},
            'event': {'Pong(uint256)': fixture['Pong(uint256)']}}}
        before = C.protocol(root, wanted)
        request = root / 'http-0-request.json'
        value = C.read(request)
        for rows in value.values(): rows.reverse()
        C.R.write(request, value)
        metadata = root / 'http-0.json'; value = C.read(metadata); value['request_sha256'] = C.R.digest(request)
        C.R.write(metadata, value)
        self.assertEqual(before, C.protocol(root, wanted))
        value = C.read(request); value['function'].pop(); C.R.write(request, value)
        metadata_value = C.read(metadata); metadata_value['request_sha256'] = C.R.digest(request)
        C.R.write(metadata, metadata_value)
        with self.assertRaisesRegex(ValueError, 'Omitted, repeated or rejected'):
            C.protocol(root, wanted)


if __name__ == '__main__':
    unittest.main()
