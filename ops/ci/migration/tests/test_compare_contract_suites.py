#!/usr/bin/env python3
"""Reject omitted workload, wrong commands and stale or corrupt producer reports."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
import sys
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tests'))
from ci_test_fixtures import ReportFixtures
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('compare', Path(__file__).resolve().parents[2] / 'migration' / 'compare-contract-suites.py')
C = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(C)
SHA = 'a' * 40


class ComparisonTests(ReportFixtures, unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup); self.root = Path(self.tmp.name); self.dirs = {}
        for provider in ('circle', 'rwx'):
            d = self.root / provider; d.mkdir(); self.dirs[provider] = d; root = '/' + provider
            files = ['test/Fixture.t.sol']; match = C.CS.match_path(files)
            settings = {'source_sha': SHA, 'suite': 'standard', 'feature': 'main', 'branch': 'pilot', 'profile': 'liteci',
                        'test_list': 'find test -name "*.t.sol"',
                        'provider': 'circleci' if provider == 'circle' else 'rwx', 'workspace_root': root,
                        'forge': 'pinned', 'go': 'pinned', 'just': 'pinned',
                        'input_sha256': {'packages/contracts-bedrock/test/Fixture.t.sol': 'b' * 64},
                        'runtime_output_paths': [],
                        'rwx_run_id': 'c' * 32 if provider == 'rwx' else None, 'rwx_task_attempt': '1'}
            self.write(d / 'settings.json', settings)
            self.write(d / 'source-after-preparation.json', settings['input_sha256'])
            config = {'out': 'out', 'root': root, 'fuzz': {'runs': 128}, 'invariant': {'runs': 64, 'depth': 32}}
            self.write(d / 'foundry-config.json', config); (d / 'files.log').write_text('test/Fixture.t.sol\n')
            self.write(d / 'file-selection.json', {'files': files, 'partitions': [{'index': 0, 'files': files}], 'match_path': match})
            discovery = {'test/Fixture.t.sol': {'Fixture': ['test_a', 'test_skip']}}; self.write(d / 'discovery.json', discovery)
            artifact = 'packages/contracts-bedrock/out/Fixture.t.sol/Fixture.json'
            methods = {'test/Fixture.t.sol:Fixture': {'methods': {'test_a()': 'a', 'test_skip()': 'b'}, 'artifacts': {artifact: 'bound'},
                       'creation_bytecode': {artifact: {'bytes': 1, 'sha256': C.UP.hashlib.sha256(b'00').hexdigest()}}}}
            self.write(d / 'signature-bindings.json', methods); self.write(d / 'compiled.json', {artifact: 'bound', 'packages/contracts-bedrock/scripts/go-ffi/go-ffi': 'ffi'})
            selected = C.UP.selection(discovery, methods); self.write(d / 'selection.json', selected)
            self.write(d / 'compile-only.json', {'tests': 0, 'selected_cases': 2, 'suite': 'standard', 'feature': 'main'})
            (d / 'submodules.txt').write_text(' ' + 'd' * 40 + ' original-module\n')
            commands = {'files': C.CS.FILE_COMMANDS['standard'], 'foundry-config': ['forge', 'config', '--json'],
                        'submodules-sync': ['git', '-C', root, 'submodule', 'sync', '--recursive'],
                        'submodules-init': ['git', '-C', root, '-c', 'protocol.file.allow=never', 'submodule',
                                            'update', '--init', '--recursive', '--jobs', '8'],
                        'go-ffi': ['just', 'build-go-ffi'], 'contracts-build': ['forge', 'build'],
                        'discovery': ['forge', 'test', '--list', '--json', '--match-path', match]}
            for name, argv in commands.items(): self.stage(d, name, argv, root)
            if provider == 'rwx':
                self.seal(d); (d / 'preparation-manifest.json').write_bytes((d / 'final.json').read_bytes())
                self.write(d / 'preparation-settings.json', settings)
                self.write(d / 'runtime-config.json', config); (d / 'runtime-files.log').write_text('test/Fixture.t.sol\n')
                self.stage(d, 'runtime-config', ['forge', 'config', '--json'], root)
                self.stage(d, 'runtime-files', C.CS.FILE_COMMANDS['standard'], root)
            else:
                (d / 'split.log').write_text('test/Fixture.t.sol\n'); self.stage(d, 'split', C.CS.SPLIT_COMMAND, root)
            (d / 'original.junit.xml').write_text('<testsuites><testsuite name="test/Fixture.t.sol:Fixture">'
                '<testcase name="test_a()"/><testcase name="test_skip()"><skipped>original guard</skipped></testcase></testsuite></testsuites>')
            self.write(d / 'source-after-verdict.json', settings['input_sha256'])
            self.write(d / 'runtime-fixtures.json', {})
            self.write(d / 'coverage.json', C.UP.junit(d / 'original.junit.xml', selected, methods))
            self.stage(d, 'tests', ['forge', 'test', '--match-path', match, '--junit'], root)
            self.stage(d, 'junit-nonempty', ['./scripts/checks/check-junit-tests-ran.sh', root + '/.ci/contract-suites/standard-main/run/original.junit.xml'], root)
            self.stage(d, 'lint-test-names', ['just', 'lint-forge-tests-check-no-build'], root); self.seal(d)

    def stage(self, d, name, argv, root): self.write(d / (name + '.stage.json'), {'argv': argv, 'cwd': root + '/packages/contracts-bedrock', 'exit_code': 0})
    def compare(self): return C.compare(self.dirs, 'standard', 'main', SHA)

    def test_complete_original_settings_assignments_and_verdicts_match(self):
        self.assertTrue(self.compare()['verified_parity'])

    def test_only_empty_non_test_interfaces_are_distinct_from_executable_workload(self):
        d = self.dirs['rwx']; name = 'test/Fixture.t.sol:VmContractHelper123'
        artifact = 'packages/contracts-bedrock/out/Fixture.t.sol/VmContractHelper123.json'
        row = {'methods': {'getCode(string)': 'a'}, 'artifacts': {artifact: 'helper'},
               'creation_bytecode': {artifact: {'bytes': 0, 'sha256': C.UP.hashlib.sha256(b'').hexdigest()}}}
        self.mutate(d, 'signature-bindings.json', lambda v: v.update({name: row}))
        self.mutate(d, 'compiled.json', lambda v: v.update({artifact: 'helper'}))
        def seal_preparation():
            manifest = json.loads((d / 'preparation-manifest.json').read_text())
            for filename in ('signature-bindings.json', 'compiled.json'):
                manifest['original_sha256'][filename] = C.UP.digest(d / filename)
            self.write(d / 'preparation-manifest.json', manifest); self.seal(d)
        seal_preparation()
        result = self.compare()
        self.assertEqual(result['non_test_interface_differences'][name]['rwx'],
                         {'methods': {'getCode(string)': 'a'}, 'deployable': False})
        for bytecode, method in [(1, 'getCode(string)'), (0, 'test_removed()'), (0, 'invariant_removed()')]:
            row['methods'] = {method: 'a'}
            row['creation_bytecode'][artifact] = {'bytes': bytecode, 'sha256': C.UP.hashlib.sha256(b'00' if bytecode else b'').hexdigest()}
            self.mutate(d, 'signature-bindings.json', lambda v: v.update({name: row})); seal_preparation()
            with self.subTest(bytecode=bytecode, method=method), self.assertRaisesRegex(ValueError, 'differs at methods'):
                self.compare()

    def test_omitted_new_file_is_rejected_even_when_both_providers_omit_it(self):
        for d in self.dirs.values():
            self.mutate(d, 'settings.json', lambda v: v['input_sha256'].update({'packages/contracts-bedrock/test/New.t.sol': 'f' * 64}))
            for phase in ('preparation', 'verdict'):
                self.mutate(d, 'source-after-' + phase + '.json', lambda v: v.update({'packages/contracts-bedrock/test/New.t.sol': 'f' * 64}))
        with self.assertRaisesRegex(ValueError, 'discovery'): self.compare()

    def test_matching_source_mutations_are_rejected(self):
        for d in self.dirs.values():
            self.mutate(d, 'source-after-verdict.json', lambda v: v.update({'go.sum': 'f' * 64}))
        with self.assertRaisesRegex(ValueError, 'source inputs changed during verdict'): self.compare()

    def tracked_fixture(self, d, name):
        before = d / 'tracked-fixtures/before' / name; before.parent.mkdir(parents=True); before.write_text('original')
        after = d / 'tracked-fixtures/after' / name; after.parent.mkdir(parents=True); after.write_text('generated')
        settings = json.loads((d / 'settings.json').read_text())
        settings['input_sha256'][name] = C.UP.digest(before); settings['runtime_output_paths'] = [name]
        self.write(d / 'settings.json', settings); self.write(d / 'source-after-preparation.json', settings['input_sha256'])
        final_inputs = dict(settings['input_sha256'], **{name: C.UP.digest(after)})
        self.write(d / 'source-after-verdict.json', final_inputs)
        self.write(d / 'runtime-fixtures.json', {name: {'before_sha256': C.UP.digest(before), 'after_sha256': C.UP.digest(after)}})
        self.write(d / 'source-changes-verdict.json', {name: {'before': C.UP.digest(before), 'after': C.UP.digest(after)}})
        if d == self.dirs['rwx']:
            self.write(d / 'preparation-settings.json', settings)
            manifest = json.loads((d / 'preparation-manifest.json').read_text())
            for file in ['settings.json', 'source-after-preparation.json', 'tracked-fixtures/before/' + name]:
                manifest['original_sha256'][file] = C.UP.digest(d / file)
            self.write(d / 'preparation-manifest.json', manifest)
        self.seal(d)

    def test_selected_tracked_runtime_fixture_requires_original_payload_hashes(self):
        name = 'packages/contracts-bedrock/snapshots/upgrades/current-upgrade-bundle.json'
        with patch.dict(C.CS.TRACKED_OUTPUTS, {'test/Fixture.t.sol': name}, clear=True):
            for d in self.dirs.values(): self.tracked_fixture(d, name)
            result = self.compare(); self.assertIn(name, result['runtime_fixtures'])
            d = self.dirs['rwx']; (d / 'tracked-fixtures/after' / name).write_text('corrupt'); self.seal(d)
            with self.assertRaisesRegex(ValueError, 'Corrupt original tracked runtime fixture'): self.compare()

    def test_matching_undeclared_runtime_output_role_is_rejected(self):
        for d in self.dirs.values(): self.mutate(d, 'settings.json', lambda v: v.update(runtime_output_paths=['go.sum']))
        with self.assertRaisesRegex(ValueError, 'Unexpected tracked runtime output role'): self.compare()

    def test_duplicate_partition_or_missing_original_is_rejected(self):
        d = self.dirs['rwx']; self.mutate(d, 'file-selection.json', lambda v: v['partitions'].append(v['partitions'][0]))
        with self.assertRaisesRegex(ValueError, 'assignment'): self.compare()
        (d / 'original.junit.xml').unlink()
        with self.assertRaises(ValueError): self.compare()

    def test_matching_wrong_command_is_not_proof_of_parity(self):
        for d in self.dirs.values(): self.mutate(d, 'tests.stage.json', lambda v: v.update(argv=['true']))
        with self.assertRaisesRegex(ValueError, 'command'): self.compare()

    def test_reduced_fuzzing_is_rejected(self):
        for d in self.dirs.values(): self.mutate(d, 'foundry-config.json', lambda v: v['fuzz'].update(runs=1))
        with self.assertRaisesRegex(ValueError, 'settings'): self.compare()

    def test_stale_sha_and_uninvestigated_retry_fail(self):
        d = self.dirs['rwx']; self.mutate(d, 'settings.json', lambda v: v.update(source_sha='f' * 40))
        with self.assertRaisesRegex(ValueError, 'source'): self.compare()
        self.setUp(); d = self.dirs['rwx']; self.stage(d, 'rerun', ['just', 'test-rerun'], '/rwx'); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'retry'): self.compare()

    def test_corrupt_original_or_unsealed_extra_is_rejected(self):
        d = self.dirs['rwx']; (d / 'original.junit.xml').write_text('corrupt')
        with self.assertRaises(ValueError): self.compare()
        self.setUp(); d = self.dirs['rwx']; (d / 'extra.txt').write_text('unsealed')
        with self.assertRaisesRegex(ValueError, 'Unsealed'): self.compare()

    def test_changed_runtime_config_or_corrupt_preparation_is_rejected(self):
        d = self.dirs['rwx']; self.mutate(d, 'runtime-config.json', lambda v: v.update(match_test='partial'))
        with self.assertRaisesRegex(ValueError, 'Runtime'): self.compare()
        self.setUp(); d = self.dirs['rwx']; self.mutate(d, 'preparation-manifest.json', lambda v: v.update(exit_code=17))
        with self.assertRaisesRegex(ValueError, 'producer'): self.compare()

    def test_native_reused_verdict_and_nonzero_tests_in_producer_are_rejected(self):
        d = self.dirs['rwx']; self.mutate(d, 'settings.json', lambda v: v.update(rwx_run_id=None))
        with self.assertRaisesRegex(ValueError, 'Reused'): self.compare()
        self.setUp(); d = self.dirs['circle']; self.mutate(d, 'compile-only.json', lambda v: v.update(tests=1))
        with self.assertRaisesRegex(ValueError, 'compile-only'): self.compare()


if __name__ == '__main__': unittest.main()
