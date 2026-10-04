#!/usr/bin/env python3
"""Reject runtime/source changes while identifying compiler-only metadata drift."""
from collections import Counter
import copy
import importlib.util
import io
import json
from pathlib import Path
import tarfile
import subprocess
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('contract_compare', Path(__file__).with_name('compare-contract-artifacts.py'))
COMPARE = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(COMPARE)


class ContractComparisonTests(unittest.TestCase):
    def setUp(self):
        self.a = {'abi': [{'type': 'function', 'name': 'correct'}], 'bytecode': {'object': '0x1234', 'sourceMap': '0:4:1'},
            'deployedBytecode': {'object': '0x4321'}, 'storageLayout': {'storage': []},
            'metadata': {'settings': {'optimizer': {'enabled': True}, 'remappings': ['/circle/lib/:x/=lib/x/']}},
            'ast': {'id': 11, 'nodeType': 'SourceUnit', 'src': '0:20:1',
                    'nodes': [{'id': 12, 'nodeType': 'ContractDefinition', 'name': 'Correct', 'scope': 11,
                               'typeDescriptions': {'typeIdentifier': 't_contract$_Correct_$12'},
                               'src': '3:12:1', 'literal': {'value': '42'}}]}, 'id': 1}
        self.b = copy.deepcopy(self.a); self.b['metadata']['settings']['remappings'][0] = '/native/lib/:x/=lib/x/'
        self.b['id'] = 3; self.b['ast']['id'] = 31; self.b['ast']['src'] = '0:20:3'
        self.b['ast']['nodes'][0].update(id=32, scope=31, src='3:12:3')
        self.b['ast']['nodes'][0]['typeDescriptions']['typeIdentifier'] = 't_contract$_Correct_$32'
        for row in (self.a, self.b): row['rawMetadata'] = json.dumps(row['metadata'])

    def compare(self):
        counts = Counter(); COMPARE.contract(self.a, self.b, ['/circle', '/native'], counts); return counts

    def test_all_runtime_fields_equal_with_explicit_identifier_differences(self):
        counts = self.compare()
        self.assertEqual(counts['workspace_path_metadata'], 2)
        self.assertGreater(counts['node_identifiers'], 0)
        self.assertEqual(counts['type_identifiers'], 1)

    def test_bytecode_abi_source_map_and_storage_changes_are_rejected(self):
        original = copy.deepcopy(self.b)
        for key, value in [('bytecode', {'object': '0xbad'}), ('abi', []), ('storageLayout', {'storage': ['bad']}),
                           ('bytecode', {'object': '0x1234', 'sourceMap': '1:4:1'})]:
            self.b = copy.deepcopy(original); self.b[key] = value
            with self.subTest(key=key), self.assertRaisesRegex(ValueError, 'runtime'): self.compare()

    def test_ast_literal_span_structure_and_type_changes_are_rejected(self):
        original = copy.deepcopy(self.b)
        for key, value in [('name', 'Wrong'), ('src', '4:12:3'), ('literal', {'value': '43'}),
                           ('typeDescriptions', {'typeIdentifier': 't_array$_Correct_$32_5_storage'})]:
            self.b = copy.deepcopy(original); self.b['ast']['nodes'][0][key] = value
            with self.subTest(key=key), self.assertRaises(ValueError): self.compare()
        self.b = copy.deepcopy(original); self.b['ast']['nodes'].append({'name': 'extra'})
        with self.assertRaisesRegex(ValueError, 'structure'): self.compare()

    def test_effective_optimizer_and_metadata_source_hash_changes_are_rejected(self):
        self.b['metadata']['settings']['optimizer']['enabled'] = False
        with self.assertRaisesRegex(ValueError, 'input metadata'): self.compare()

    def test_cache_content_is_checked_while_timestamps_and_build_ids_are_retained(self):
        a = {'files': {'A.sol': {'contentHash': 'correct', 'lastModificationDate': 10,
             'artifacts': {'A': {'build_id': 'old', 'path': 'A.json'}}}}, 'builds': ['old']}
        b = copy.deepcopy(a); b['files']['A.sol']['lastModificationDate'] = 20
        b['files']['A.sol']['artifacts']['A']['build_id'] = 'new'; b['builds'] = ['new']
        self.assertEqual(COMPARE.cache_inputs(a), COMPARE.cache_inputs(b))
        b['files']['A.sol']['contentHash'] = 'wrong'
        self.assertNotEqual(COMPARE.cache_inputs(a), COMPARE.cache_inputs(b))

    def test_archive_corruption_duplicate_missing_and_unsafe_members_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / 'files.tar.gz'; data = b'compiled'
            for mode in ('correct', 'duplicate', 'extra', 'unsafe', 'missing', 'corrupt'):
                with tarfile.open(path, 'w:gz') as stream:
                    for name in ([] if mode == 'missing' else ['A.json', 'A.json'] if mode == 'duplicate'
                                 else ['A.json', 'extra'] if mode == 'extra' else ['../outside'] if mode == 'unsafe' else ['A.json']):
                        member = tarfile.TarInfo(name); member.size = len(data); stream.addfile(member, io.BytesIO(data))
                metadata = {'archive_sha256': COMPARE.digest(path.read_bytes()), 'files': {'A.json': COMPARE.digest(data)}}
                if mode == 'corrupt': metadata['files']['A.json'] = 'wrong'
                with self.subTest(mode=mode):
                    if mode == 'correct': self.assertEqual(COMPARE.archive(path, metadata), {'A.json': data})
                    else:
                        with self.assertRaises(ValueError): COMPARE.archive(path, metadata)

    def test_actual_zstd_archive_works_with_the_pinned_python_reader(self):
        raw = io.BytesIO()
        with tarfile.open(fileobj=raw, mode='w') as stream:
            for name, data in [('src/A.sol', b'pragma solidity 0.8.15;'),
                               ('forge-artifacts/A.json', b'{"bytecode":"0x1234"}')]:
                member = tarfile.TarInfo(name); member.size = len(data)
                stream.addfile(member, io.BytesIO(data))
        compressed = subprocess.run(['zstd', '--compress', '--stdout'], input=raw.getvalue(),
                                    capture_output=True, check=True).stdout
        files, links = COMPARE.embedded(compressed)
        self.assertEqual(files, {'src/A.sol': COMPARE.digest(b'pragma solidity 0.8.15;'),
                                'forge-artifacts/A.json': COMPARE.digest(b'{"bytecode":"0x1234"}')})
        self.assertEqual(links, {})
        with self.assertRaises((ValueError, tarfile.ReadError)):
            COMPARE.embedded(compressed[:-8])

    def test_embedded_file_link_aliases_cannot_hide_duplicate_members(self):
        raw = io.BytesIO()
        with tarfile.open(fileobj=raw, mode='w:gz') as stream:
            file = tarfile.TarInfo('src/A.sol'); file.size = 1; stream.addfile(file, io.BytesIO(b'A'))
            link = tarfile.TarInfo('src/A.sol'); link.type = tarfile.SYMTYPE; link.linkname = 'elsewhere.sol'
            stream.addfile(link)
        with self.assertRaisesRegex(ValueError, 'Duplicate'): COMPARE.embedded(raw.getvalue())


if __name__ == '__main__': unittest.main()
