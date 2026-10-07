"""Reject incomplete or mismatched Cannon provider evidence."""
import gzip
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('compare_cannon', Path(__file__).resolve().parents[2] / 'migration' / 'compare-rust-cannon.py')
COMPARE = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(COMPARE)
SHA = 'a' * 40
ELF = b'\x7fELF\x02\x02\x01' + b'\0' * 11 + b'\0\x08' + b'compiled fixture'


class ComparisonTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory(); self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        witness = {'filename': 'fixture.tar.zst', 'sha256': 'b' * 64, 'l2_claim': '0x' + 'c' * 64}
        bound = dict.fromkeys(COMPARE.BINDING, 'pinned') | {'source_sha': SHA, 'witness': witness}
        variants = [{'binary': 'kona-client', 'prestate_directory': 'prestate-artifacts-cannon'}]
        for provider in ('circle', 'rwx'):
            for job in COMPARE.JOBS:
                d = self.root / provider / ('cannon-' + job); d.mkdir(parents=True)
                workdir = '/' + provider
                settings = bound | {'job': job, 'suite': 'rust-cannon', 'provider': 'circleci' if provider == 'circle' else 'rwx',
                    'workspace_root': workdir, 'variants': variants, 'rustc': 'pinned', 'cargo': 'pinned',
                    'rwx_run_id': 'd' * 32, 'rwx_task_attempt': '1'}
                self.write(d / 'settings.json', settings)
                if provider == 'rwx' and job != 'offline': self.write(d / 'verified-env.json', {'producer_source_sha': SHA, 'manifest_sha256': '1' * 64})
                (d / 'checks.junit.xml').write_text('<testsuite><testcase name="cannon"/></testsuite>')
                stages = {}
                for name in ('variants', job) + (('guest',) if job == 'offline' else ()):
                    argv = [workdir + '/cannon/bin/cannon', 'witness'] if name == 'guest' else ['just', name]
                    stages[name] = {'argv': argv, 'cwd': 'rust', 'exit_code': 0}
                    self.write(d / (name + '.stage.json'), stages[name])
                    (d / (name + '.log')).write_text('kona-client prestate-artifacts-cannon\n' if name == 'variants' else 'original')
                self.write(d / 'image.json', {'binding': bound, 'image_id': provider, 'base': 'pinned', 'base_digests': ['digest'], 'tools': 'pinned'})
                for folder in ('image-elfs',) + (('elfs',) if job != 'lint' else ()):
                    (d / folder).mkdir(); (d / folder / 'kona-client').write_bytes(ELF)
                    self.write(d / (folder + '.json'), {'binding': bound, 'variants': variants,
                        'files': {'kona-client': {'sha256': COMPARE.digest(d / folder / 'kona-client'), 'size': len(ELF)}}})
                if job == 'offline':
                    state = {'exited': True, 'exitCode': 0, 'step': 100, 'stateVersion': 8,
                             'witnessHash': '0x' + 'f' * 64, 'witness': '0x0011'}
                    self.write(d / 'guest.json', state)
                    (d / 'offline.log').write_text('Successfully validated L2 block ' + witness['l2_claim'])
                    (d / 'out.bin.gz').write_bytes(gzip.compress(b'complete final state'))
                    self.write(d / 'guest-coverage.json', {'boundary': witness, 'final': state, 'fresh_execution': True,
                        'final_state_sha256': COMPARE.digest(d / 'out.bin.gz')})
                    self.write(d / 'go-binaries.json', {'binding': bound, 'go': 'pinned',
                        'files': {name: {'sha256': 'e' * 64} for name in COMPARE.GO_FILES}})
                    self.write(d / 'witness.json', {'binding': bound, 'filename': witness['filename'], 'sha256': witness['sha256'], 'size': 123})
                    if provider == 'rwx':
                        for kind in ('go', 'witness', 'build'):
                            verified = {'producer_source_sha': SHA, 'manifest_sha256': '1' * 64}
                            if kind == 'build': verified['image_manifest_sha256'] = COMPARE.digest(self.root / provider / 'cannon-build/image.json')
                            self.write(d / ('verified-' + kind + '.json'), verified)
                self.write(d / 'final.json', {'exit_code': 0, 'report_errors': [], 'stages': stages})
                self.rehash(d)

    def write(self, path, data): path.write_text(json.dumps(data) + '\n')

    def rehash(self, directory):
        final = COMPARE.read(directory / 'final.json')
        final['original_sha256'] = {str(p.relative_to(directory)): COMPARE.digest(p)
                                   for p in directory.rglob('*') if p.is_file() and p.name != 'final.json'}
        self.write(directory / 'final.json', final)

    def directory(self, provider='rwx', job='offline'): return self.root / provider / ('cannon-' + job)

    def update(self, filename, change, provider='rwx', job='offline'):
        d = self.directory(provider, job); data = COMPARE.read(d / filename); change(data)
        self.write(d / filename, data)
        if filename != 'final.json': self.rehash(d)

    def test_complete_reports_with_distinct_host_paths_and_image_ids(self):
        result = COMPARE.compare(self.root, SHA)
        self.assertTrue(result['verified_parity'])
        self.assertTrue(result['jobs']['offline']['guest']['complete_state_byte_identical'])

    def test_corrupt_missing_or_unmanifested_original_rejected(self):
        d = self.directory(); (d / 'offline.log').write_text('corrupt')
        with self.assertRaisesRegex(ValueError, 'Corrupt original'): COMPARE.compare(self.root, SHA)
        (d / 'offline.log').unlink()
        with self.assertRaisesRegex(ValueError, 'Missing nonempty'): COMPARE.compare(self.root, SHA)
        self.update('final.json', lambda f: f['original_sha256'].pop('offline.log'))
        with self.assertRaisesRegex(ValueError, 'Incomplete original'): COMPARE.compare(self.root, SHA)

    def test_only_manifest_declared_empty_circle_guest_stream_may_be_absent(self):
        d = self.directory('circle'); (d / 'guest.log').write_text(''); self.rehash(d); (d / 'guest.log').unlink()
        self.assertEqual(COMPARE.compare(self.root, SHA)['manifest_declared_empty_logs'][0]['name'], 'guest.log')
        (d / 'guest.json').unlink()
        with self.assertRaisesRegex(ValueError, 'Missing nonempty'): COMPARE.compare(self.root, SHA)

    def test_failed_guest_report_and_unresolved_retries_rejected(self):
        self.update('settings.json', lambda s: s.update(rwx_task_attempt='2'))
        with self.assertRaisesRegex(ValueError, 'retries require investigation'): COMPARE.compare(self.root, SHA)
        self.update('settings.json', lambda s: s.update(rwx_task_attempt='1'))
        self.update('final.json', lambda f: f.update(report_errors=['failed guest']))
        with self.assertRaisesRegex(ValueError, 'Failed or incomplete'): COMPARE.compare(self.root, SHA)

    def test_mismatched_source_settings_or_variant_selection_rejected(self):
        self.update('settings.json', lambda s: s.update(source_sha='9' * 40))
        with self.assertRaisesRegex(ValueError, 'source or workload'): COMPARE.compare(self.root, SHA)
        self.update('settings.json', lambda s: s.update(source_sha=SHA, variants=[]))
        with self.assertRaisesRegex(ValueError, 'settings differ'): COMPARE.compare(self.root, SHA)

    def test_original_command_and_summary_drift_rejected(self):
        self.update('guest.stage.json', lambda s: s.update(argv=['wrong', 'guest']))
        with self.assertRaisesRegex(ValueError, 'summary differs'): COMPARE.compare(self.root, SHA)
        self.update('final.json', lambda f: f['stages']['guest'].update(argv=['wrong', 'guest']))
        with self.assertRaisesRegex(ValueError, 'command, working directory'): COMPARE.compare(self.root, SHA)

    def test_altered_elf_binding_or_binary_inventory_rejected(self):
        self.update('elfs.json', lambda m: m['files']['kona-client'].update(sha256='9' * 64))
        with self.assertRaisesRegex(ValueError, 'ELF inventories'): COMPARE.compare(self.root, SHA)

    def test_stale_inherited_producer_and_missing_go_implementation_rejected(self):
        self.update('verified-go.json', lambda m: m.update(producer_source_sha='9' * 40))
        with self.assertRaisesRegex(ValueError, 'producer provenance'): COMPARE.compare(self.root, SHA)
        self.update('verified-go.json', lambda m: m.update(producer_source_sha=SHA))
        self.update('go-binaries.json', lambda m: m['files'].pop('cannon/bin/cannon64-impl'))
        with self.assertRaisesRegex(ValueError, 'Go build provenance'): COMPARE.compare(self.root, SHA)

    def test_offline_image_is_bound_to_the_counted_build_producer(self):
        self.update('verified-build.json', lambda m: m.update(image_manifest_sha256='9' * 64))
        with self.assertRaisesRegex(ValueError, 'image handoff'):
            COMPARE.compare(self.root, SHA)
        self.update('verified-build.json', lambda m: m.pop('image_manifest_sha256'))
        with self.assertRaisesRegex(ValueError, 'image handoff'):
            COMPARE.compare(self.root, SHA)

    def test_guest_state_drift_even_with_equal_coverage_summaries_rejected(self):
        self.update('guest.json', lambda s: s.update(exitCode=1))
        with self.assertRaisesRegex(ValueError, 'unsuccessful original'): COMPARE.compare(self.root, SHA)

    def test_full_state_drift_despite_identical_witness_summary_rejected(self):
        d = self.directory(); (d / 'out.bin.gz').write_bytes(gzip.compress(b'different full state'))
        self.update('guest-coverage.json', lambda g: g.update(final_state_sha256=COMPARE.digest(d / 'out.bin.gz')))
        with self.assertRaisesRegex(ValueError, 'Complete uncompressed'): COMPARE.compare(self.root, SHA)


if __name__ == '__main__': unittest.main()
