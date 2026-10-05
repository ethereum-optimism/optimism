#!/usr/bin/env python3
"""Reject incomplete coverage, altered instrumentation and mismatched archives."""
import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('compare', Path(__file__).with_name('compare-contract-coverage.py'))
C = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(C)
SHA = 'a'*40


class ComparisonTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup); self.root = Path(self.tmp.name); self.dirs = {}
        for provider in ('circle', 'rwx'):
            d = self.root / provider; d.mkdir(); self.dirs[provider] = d; root = '/' + provider
            source = 'test/L1/Fixture.t.sol'; identity = source + ':OPContractsManagerFixture_Upgrade_Test'
            artifact = 'packages/contracts-bedrock/forge-artifacts/Fixture.t.sol/OPContractsManagerFixture_Upgrade_Test.json'
            seed = '0x' + hashlib.sha256((SHA + ':main:contract-coverage').encode()).hexdigest()
            inputs = {'packages/contracts-bedrock/' + source: 'b'*64, 'packages/contracts-bedrock/src/A.sol': 'e'*64}
            settings = {'source_sha': SHA, 'feature': 'main', 'branch': 'pilot', 'profile': 'cicoverage', 'source_build_profile': 'default',
                        'benchmark_seed': seed, 'rpc_input_name': C.C.RPC_INPUT, 'authority': ['just', 'coverage-lcov-all'],
                        'ffi_replay_seed': seed, 'ffi_replay_policy': 'chacha8-arguments-v1',
                        'phases': list(C.C.PHASES), 'provider': 'circleci' if provider == 'circle' else 'rwx', 'workspace_root': root,
                        'forge': 'pinned', 'go': 'pinned', 'just': 'pinned', 'input_sha256': inputs, 'runtime_output_paths': [],
                        'rwx_run_id': 'c'*32 if provider == 'rwx' else None, 'rwx_task_attempt': '1'}
            self.write(d / 'settings.json', settings); self.write(d / 'source-after-preparation.json', inputs)
            (d / 'files.log').write_text(source + '\n'); self.write(d / 'file-selection.json', {'files': [source], 'partitions': [{'index': 0, 'files': [source]}]})
            (d / 'submodules.txt').write_text(' ' + 'd'*40 + ' modules/original\n')
            config = {'root': root, 'out': 'forge-artifacts', 'optimizer': False, 'threads': 16, 'compilation_restrictions': [],
                      'fuzz': {'runs': 1, 'seed': seed}, 'invariant': {'runs': 1, 'depth': 1}}
            self.write(d / 'foundry-config.json', config); self.write(d / 'build-config.json', {'root': root, 'optimizer': True})
            bindings = {identity: {'methods': {'test_a()': 'a', 'test_skip()': 'b'}, 'artifacts': {artifact: 'f'*64},
                         'creation_bytecode': {artifact: {'bytes': 1, 'sha256': hashlib.sha256(b'00').hexdigest()}}}}
            self.write(d / 'signature-bindings.json', bindings)
            self.write(d / 'compiled.json', {artifact: 'f'*64, 'packages/contracts-bedrock/scripts/go-ffi/go-ffi': 'a'*64})
            discovery = {source: {'OPContractsManagerFixture_Upgrade_Test': ['test_a', 'test_skip']}}
            selection = C.UP.selection(discovery, bindings)
            for phase in C.C.PHASES: self.write(d / (phase + '-discovery.json'), discovery)
            self.write(d / 'selection.json', {phase: selection for phase in C.C.PHASES})
            self.write(d / 'compile-only.json', {'tests': 0, 'selected_cases': {phase: 2 for phase in C.C.PHASES}})
            commands = {'files': C.CS.FILE_COMMANDS['standard'], 'submodules-sync': ['git', '-C', root, 'submodule', 'sync', '--recursive'],
                        'submodules-init': ['git', '-C', root, '-c', 'protocol.file.allow=never', 'submodule', 'update', '--init', '--recursive', '--jobs', '8'],
                        'build-config': ['forge', 'config', '--json'], 'go-ffi': ['just', 'build-go-ffi'], 'source-build': ['just', 'build-source'],
                        'coverage-build': ['forge', 'build'], 'foundry-config': ['forge', 'config', '--json'],
                        **{phase + '-discovery': argv for phase, argv in C.C.DISCOVERY.items()}}
            for name, argv in commands.items(): self.stage(d, name, argv, root)
            if provider == 'rwx':
                self.seal(d); (d / 'preparation-manifest.json').write_bytes((d / 'final.json').read_bytes())
                self.write(d / 'preparation-settings.json', settings); self.write(d / 'runtime-config.json', config)
                (d / 'runtime-files.log').write_text(source + '\n')
                self.stage(d, 'runtime-config', ['forge', 'config', '--json'], root); self.stage(d, 'runtime-files', C.CS.FILE_COMMANDS['standard'], root)
            block = {'source_sha': SHA, 'policy': 'Just current-day 00:00 UTC', 'chain_id': 1, 'number': 42, 'hash': '0x'+'a'*64, 'timestamp': 1234}
            self.write(d / 'block.json', block); preflight = d / 'archive-preflight'; preflight.mkdir()
            self.write(preflight / 'block.json', block); (preflight / 'pinned-block.log').write_text('42\n')
            self.stage(preflight, 'pinned-block', ['just', 'print-pinned-block-number'], root); self.seal(preflight)
            (preflight / 'final.json').rename(preflight / 'preflight-manifest.json')
            self.write(d / 'source-after-verdict.json', inputs); self.write(d / 'runtime-fixtures.json', {})
            for phase in C.C.PHASES:
                p = d / phase; p.mkdir()
                text = ('Running upgrade tests at block 42\n' if phase == 'upgrade' else '') + \
                       'Ran 2 tests for ' + identity + '\n[PASS] test_a() (gas: 1)\n[SKIP: original exclusion] test_skip() (gas: 0)\n' + \
                       'Ran 1 test suite in 1ms: 1 tests passed, 0 failed, 1 skipped (2 total tests)\n'
                (p / 'tests.log').write_text(text)
                attribution = {'version': 1, 'tests': [{'suite': identity, 'test': 'test_' + name + '()', 'status': status, 'kind': 'unit', 'covered': []}
                                for name, status in [('a', 'success'), ('skip', 'skipped')]]}
                self.write(p / 'original.attribution.json', attribution); cases = C.C.original_cases(text, attribution)
                self.write(p / 'original-events.json', C.C.original_evidence(text, attribution))
                self.write(p / 'original-cases.json', cases); C.C.derived_junit(p / 'derived.junit.xml', cases, phase)
                self.write(p / 'coverage.json', C.C.accounting(p / 'derived.junit.xml', phase, selection, bindings))
                (p / 'original.lcov.info').write_text('TN:\nSF:src/A.sol\nDA:6,4\nFN:6,A.set\nFNDA:4,A.set\n'
                    'FNF:1\nFNH:1\nLF:1\nLH:1\nBRF:0\nBRH:0\nend_of_record\n')
                self.write(p / 'lcov-records.json', C.C.lcov((p / 'original.lcov.info').read_text()))
                self.write(p / 'coverage-source-sha256.json', {'src/A.sol': 'e'*64})
                self.stage(p, 'tests', C.C.COMMANDS[phase], root)
            self.seal(d)

    def write(self, path, value): path.write_text(json.dumps(value))
    def stage(self, directory, name, argv, root):
        stdout = directory / (name + ('.json' if name.endswith(('config', 'discovery')) else '.log'))
        if not stdout.exists(): stdout.write_text('')
        stderr = directory / (name + '.stderr.log'); stderr.write_text('')
        self.write(directory / (name + '.stage.json'), {'argv': argv, 'cwd': root + '/packages/contracts-bedrock', 'exit_code': 0,
                   'stdout_sha256': C.UP.digest(stdout), 'stderr_sha256': C.UP.digest(stderr)})
    def seal(self, d): C.UP.finish(d, 0, [])
    def mutate(self, d, name, fn):
        path = d / name; value = json.loads(path.read_text()); fn(value); self.write(path, value); self.seal(d)
    def compare(self): return C.compare(self.dirs, 'main', SHA)

    def test_all_original_entries_and_both_passes_match(self): self.assertTrue(self.compare()['verified_parity'])

    def test_missing_pass_or_original_report_is_rejected(self):
        d = self.dirs['rwx']; (d / 'upgrade/original.lcov.info').unlink()
        with self.assertRaises(ValueError): self.compare()

    def test_matching_reduced_profile_or_unseeded_benchmark_is_rejected(self):
        for d in self.dirs.values(): self.mutate(d, 'foundry-config.json', lambda v: v['invariant'].update(depth=0))
        with self.assertRaisesRegex(ValueError, 'settings'): self.compare()
        self.setUp()
        for d in self.dirs.values(): self.mutate(d, 'settings.json', lambda v: v.update(benchmark_seed=None))
        with self.assertRaisesRegex(ValueError, 'seed'): self.compare()

    def test_matching_noop_command_is_rejected(self):
        for d in self.dirs.values(): self.mutate(d, 'ordinary/tests.stage.json', lambda v: v.update(argv=['true']))
        with self.assertRaisesRegex(ValueError, 'command'): self.compare()

    def test_unbound_ffi_replay_corpus_is_rejected(self):
        for d in self.dirs.values(): self.mutate(d, 'settings.json', lambda v:v.update(ffi_replay_seed='0x'+'b'*64))
        with self.assertRaisesRegex(ValueError,'FFI replay'):self.compare()

    def test_missing_new_file_and_duplicate_assignments_fail(self):
        for d in self.dirs.values():
            self.mutate(d, 'settings.json', lambda v: v['input_sha256'].update({'packages/contracts-bedrock/test/New.t.sol': 'f'*64}))
            self.mutate(d, 'source-after-preparation.json', lambda v: v.update({'packages/contracts-bedrock/test/New.t.sol': 'f'*64}))
        with self.assertRaisesRegex(ValueError, 'selection'): self.compare()
        self.setUp(); d = self.dirs['circle']
        self.mutate(d, 'file-selection.json', lambda v: v['partitions'].append(v['partitions'][0]))
        with self.assertRaisesRegex(ValueError, 'assignment'): self.compare()

    def test_changed_archive_hash_or_unavailable_original_pin_fails(self):
        d = self.dirs['rwx']; self.mutate(d, 'block.json', lambda v: v.update(hash='0x'+'b'*64))
        with self.assertRaisesRegex(ValueError, 'preflight'): self.compare()

    def test_raw_lcov_hit_difference_is_not_normalized_away(self):
        d = self.dirs['circle']; p = d / 'ordinary'; source = p / 'original.lcov.info'
        source.write_text(source.read_text().replace('DA:6,4', 'DA:6,5'))
        self.write(p / 'lcov-records.json', C.C.lcov(source.read_text())); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'differs at phases'): self.compare()

    def test_every_complete_attribution_item_field_and_hit_affects_parity(self):
        item={'source':'src/A.sol','contract':'A','kind':'branch','line_start':6,'line_end':7,
              'byte_start':10,'byte_end':30,'hits':4,'branch_id':1,'path_id':0}
        for d in self.dirs.values():
            self.mutate(d,'ordinary/original.attribution.json',lambda data:data['tests'][0].update(covered=[item]))
        self.assertTrue(self.compare()['verified_parity'])
        d=self.dirs['rwx'];original=(d/'ordinary/original.attribution.json').read_text()
        for field,value in [('hits',5),('byte_end',31),('branch_id',2),('path_id',1)]:
            data=json.loads(original);data['tests'][0]['covered'][0][field]=value
            self.write(d/'ordinary/original.attribution.json',data);self.seal(d)
            with self.subTest(field=field),self.assertRaisesRegex(ValueError,'differs at phases'):self.compare()

    def test_derived_junit_cannot_invent_or_change_a_skip_reason(self):
        d = self.dirs['circle']; p = d / 'ordinary/derived.junit.xml'; p.write_text(p.read_text().replace('original exclusion', 'invented exclusion'))
        self.seal(d)
        with self.assertRaisesRegex(ValueError, 'Derived coverage JUnit differs'): self.compare()

    def test_reused_verdict_unexplained_retry_source_mutation_and_corrupt_log_fail(self):
        d = self.dirs['rwx']; self.mutate(d, 'settings.json', lambda v: v.update(rwx_run_id=None))
        with self.assertRaisesRegex(ValueError, 'Reused'): self.compare()
        self.setUp(); d = self.dirs['circle']; self.stage(d, 'rerun', ['just', 'test-rerun'], '/circle'); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'retry'): self.compare()
        self.setUp(); d = self.dirs['circle']; self.mutate(d, 'source-after-verdict.json', lambda v: v.update({'go.sum': 'f'*64}))
        with self.assertRaisesRegex(ValueError, 'source mutation'): self.compare()
        self.setUp(); d = self.dirs['circle']; (d / 'ordinary/tests.log').write_text('corrupt'); self.seal(d)
        with self.assertRaisesRegex(ValueError, 'output hashes'): self.compare()


if __name__ == '__main__': unittest.main()
