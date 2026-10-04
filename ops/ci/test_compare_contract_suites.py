#!/usr/bin/env python3
"""Reject omitted workload, wrong commands and stale or corrupt producer reports."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('compare', Path(__file__).with_name('compare-contract-suites.py'))
C = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(C)
SHA = 'a' * 40


class ComparisonTests(unittest.TestCase):
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
                        'rwx_run_id': 'c' * 32 if provider == 'rwx' else None, 'rwx_task_attempt': '1'}
            self.write(d / 'settings.json', settings)
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
            self.write(d / 'coverage.json', C.UP.junit(d / 'original.junit.xml', selected, methods))
            self.stage(d, 'tests', ['forge', 'test', '--match-path', match, '--junit'], root)
            self.stage(d, 'junit-nonempty', ['./scripts/checks/check-junit-tests-ran.sh', root + '/.ci/contract-suites/standard-main/run/original.junit.xml'], root)
            self.stage(d, 'lint-test-names', ['just', 'lint-forge-tests-check-no-build'], root); self.seal(d)

    def write(self, path, value): path.write_text(json.dumps(value))
    def stage(self, d, name, argv, root): self.write(d / (name + '.stage.json'), {'argv': argv, 'cwd': root + '/packages/contracts-bedrock', 'exit_code': 0})
    def seal(self, d): self.write(d / 'final.json', {'exit_code': 0, 'report_errors': [], 'original_sha256': {
        str(p.relative_to(d)): C.UP.digest(p) for p in d.rglob('*') if p.is_file() and p.name != 'final.json'}})
    def mutate(self, d, filename, fn):
        path = d / filename; value = json.loads(path.read_text()); fn(value); self.write(path, value); self.seal(d)
    def compare(self): return C.compare(self.dirs, 'standard', 'main', SHA)

    def test_complete_original_settings_assignments_and_verdicts_match(self):
        self.assertTrue(self.compare()['verified_parity'])

    def test_omitted_new_file_is_rejected_even_when_both_providers_omit_it(self):
        for d in self.dirs.values(): self.mutate(d, 'settings.json', lambda v: v['input_sha256'].update({'packages/contracts-bedrock/test/New.t.sol': 'f' * 64}))
        with self.assertRaisesRegex(ValueError, 'discovery'): self.compare()

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
