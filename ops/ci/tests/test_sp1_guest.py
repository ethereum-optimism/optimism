import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import tempfile
import time
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('sp1_guest', Path(__file__).resolve().parents[1] / 'runtime' / 'sp1-guest.py')
G = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(G)


class SelectionTests(unittest.TestCase):
    def test_occupied_canonical_workspace_fails_without_reading_or_changing_its_files(self):
        native=G.helper('sp1-guest-native-build')
        with tempfile.TemporaryDirectory() as temp:
            base=Path(temp);root=base/'source';root.mkdir()
            scripts=root/'ops/ci/runtime';scripts.mkdir(parents=True)
            for name in ('main-checks.py','pr-checks.py','rust-workspace-report.py','ci-report.py'):
                shutil.copyfile(Path(G.__file__).with_name(name),scripts/name)
            for argv in (['git','init','-q'],['git','add','.'],
                         ['git','-c','user.name=CI fixture','-c','user.email=ci-fixture@example.invalid','commit','-qm','occupied-directory fixture']):
                subprocess.run(argv,cwd=root,check=True,stdout=subprocess.DEVNULL)
            sha=subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip()
            occupied=base/'occupied';occupied.mkdir();sentinel=occupied/'original.txt';sentinel.write_bytes(b'original occupied data\n')
            with patch.object(native,'ROOT',root),patch.object(native.M,'ROOT',root),patch.object(native,'WORKSPACE',occupied),patch.object(native,'CARGO',base/'cargo'),patch.dict(os.environ,{'CI_COMMIT_SHA':sha}):
                self.assertEqual(native.execute(),1)
            self.assertEqual(sentinel.read_bytes(),b'original occupied data\n')
            report=G.read(root/'.ci/sp1-guest/dependency/final.json')
            self.assertEqual(report['exit_code'],1)
            self.assertEqual(report['tests'],0)
            self.assertFalse((root/'.ci/sp1-guest/dependency/files').exists())

    def test_original_program_loop_keeps_future_programs_and_rejects_invalid_selection(self):
        source = '\nbuild-elfs-native:\n    for name in super-range super-aggregation future-program; do\n'
        self.assertEqual(G.programs(source), ['super-range', 'super-aggregation', 'future-program'])
        for names in ('super-range', 'super-range super-aggregation super-range', ''):
            with self.assertRaises(ValueError): G.programs('\nbuild-elfs-native:\n    for name in '+names+'; do\n')

    def test_parameterized_authoritative_program_loop(self):
        source = '\nbuild-elfs-native features="":\n    for name in super-range super-aggregation future-program; do\n'
        self.assertEqual(G.programs(source), ['super-range', 'super-aggregation', 'future-program'])
        with self.assertRaisesRegex(ValueError, 'recipe'): G.programs('other-recipe:\n')

    def test_runtime_does_not_recover_manifested_empty_logs(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp)
            final = {'original_sha256':{'empty.log':G.hashlib.sha256(b'').hexdigest()}}
            G.S.write(path/'final.json', final)
            with self.assertRaisesRegex(ValueError, 'Missing'): G.verify_seals(path, final)
            self.assertFalse((path/'empty.log').exists())

    def test_guest_feature_markers_and_source_suffix_bind_actual_elf_bytes(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp)
            settings = {'programs':['super-range','super-aggregation'], 'source_sha':'a'*40,
                        'tools':{'sp1-version':'6.0.0'}, 'features':'test-config-fallback',
                        'elf_source_sha':'a'*40+'-test'}
            def write(features, source):
                (path/'vkeys.toml').write_text('git_sha="'+source+'"\nsp1_toolchain_tag="v6.0.0"\n'+
                    ''.join(n+'="0x'+'b'*64+'"\n' for n in settings['programs']))
                for n in settings['programs']:
                    data = bytearray(64); data[:6] = b'\x7fELF\x02\x01'; G.struct.pack_into('<HH',data,16,2,243)
                    data += ('KONA_SP1_BUILD{git_sha='+source+'}').encode()
                    if features and n=='super-range': data += G.TEST_CONFIG_MARKER
                    (path/(n+'-elf')).write_bytes(data)
            write(True, settings['elf_source_sha'])
            self.assertEqual(set(G.elf_record(settings,path)),set(settings['programs']))
            write(False, settings['elf_source_sha'])
            with self.assertRaisesRegex(ValueError, 'test-config'): G.elf_record(settings,path)
            write(True, settings['source_sha'])
            with self.assertRaisesRegex(ValueError, 'source'): G.elf_record(settings,path)
            settings.update(features='', elf_source_sha=settings['source_sha'])
            with self.assertRaisesRegex(ValueError, 'test-config'): G.elf_record(settings,path)
            write(False, settings['source_sha'])
            self.assertEqual(set(G.elf_record(settings,path)),set(settings['programs']))

    def test_report_seals_reject_missing_extra_corrupt_and_linked_originals(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp); original = path/'original.log'; original.write_bytes(b'original evidence\n')
            final = {'original_sha256':G.seal(path)}; G.S.write(path/'final.json', final)
            G.verify_seals(path, final)
            original.write_bytes(b'changed evidence\n')
            with self.assertRaisesRegex(ValueError, 'corrupt'): G.verify_seals(path, final)
            original.unlink()
            with self.assertRaisesRegex(ValueError, 'corrupt'): G.verify_seals(path, final)
            original.write_bytes(b'original evidence\n'); (path/'extra.log').write_bytes(b'extra')
            with self.assertRaisesRegex(ValueError, 'unsealed'): G.verify_seals(path, final)
            (path/'extra.log').unlink(); original.unlink(); original.symlink_to(path/'final.json')
            with self.assertRaisesRegex(ValueError, 'link'): G.verify_seals(path, final)


class _LiveTestsFixtures:

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.reports = G.REPORTS / 'helper-fixtures' / self._testMethodName
        if self.reports.exists():
            self.reports.rename(self.reports.with_name(self.reports.name + '-prior-' + str(time.time_ns())))
        self.reports.mkdir(parents=True, exist_ok=True)
        self.env = {**os.environ, 'CARGO_TARGET_DIR': str(self.root / 'target'), 'CARGO_INCREMENTAL': '0', 'RUSTFLAGS': '-Dwarnings'}
        for name in ('CARGO_ENCODED_RUSTFLAGS', 'RUSTC_WRAPPER', 'RUSTC_WORKSPACE_WRAPPER'):
            self.env.pop(name, None)
        (self.root / 'Cargo.toml').write_text('[workspace]\nresolver="2"\nmembers=["one","two","empty","future"]\n')
        for name in ('one', 'two', 'empty', 'future'):
            path = self.root / name
            (path / 'src').mkdir(parents=True)
            (path / 'Cargo.toml').write_text('[package]\nname="' + name + '"\nversion="0.1.0"\nedition="2021"\n')
            body = '' if name == 'empty' else '#[test] fn same_name() {assert_eq!(2+2,4);}\n'
            if name == 'one':
                body += '#[test] #[ignore="fixture original ignored case"] fn ignored_case() {}\n'
            (path / 'src/lib.rs').write_text(body)
        self.command('metadata', ['cargo', 'metadata', '--format-version', '1'], json_output=True)
        self.metadata = G.read(self.reports / 'metadata.json')
        self.settings = {'workspace_root': str(self.root), 'cargo_home': self.env.get('CARGO_HOME', str(Path.home() / '.cargo'))}

    def command(self, name, argv, json_output=False, expected=0):
        with patch.dict(os.environ, self.env, clear=True):
            status = G.S.stage(self.reports, name, argv, stdout_json=json_output, cwd=str(self.root), stdin=subprocess.DEVNULL)
        self.assertEqual(status, expected)
        return status

    @staticmethod
    def reseal(path):
        final = G.read(path / 'final.json')
        (path / 'final.json').unlink()
        final['original_sha256'] = G.seal(path)
        G.S.write(path / 'final.json', final)


@unittest.skipUnless(os.environ.get('RWX_LIVE_SP1_FIXTURE') == '1', 'Actual Cargo discovery, execution and production ELF fixtures')
class LiveTests(_LiveTestsFixtures, unittest.TestCase):

    def test_complete_actual_discovery_zero_case_future_package_and_scoped_duplicate_names(self):
        manifest = self.root/'future/Cargo.toml'
        manifest.write_text(manifest.read_text()+'[features]\nselected=[]\n')
        source = self.root/'future/src/lib.rs'
        source.write_text(source.read_text()+'#[cfg(feature="selected")] #[test] fn feature_selected() {}\n')
        self.command('guest-list', ['cargo', 'test', '--workspace', '--locked', '--all-features', '--', '--list'])
        self.command('guest', ['cargo', 'test', '--workspace', '--locked', '--all-features'])
        record = G.coverage(self.reports, 'guest', self.metadata, self.settings)
        self.assertEqual(record['packages'], ['empty', 'future', 'one', 'two'])
        self.assertEqual(record['outcomes'], {'pass': 4, 'skip': 1})
        self.assertEqual(len(record['targets']), 8)
        self.assertEqual(record['targets']['empty:empty:unit']['count'], 0)
        self.assertEqual(len({c['suite'] for c in record['cases'] if c['name'] == 'same_name'}), 3)
        text = (self.reports / 'guest-list.log').read_text()
        with self.assertRaises(ValueError):
            G.libtest(text.replace('same_name: test', '', 1), self.metadata, True, **{'root': str(self.root), 'cargo_home': self.settings['cargo_home']})

    def test_actual_failure_remains_failing_and_every_original_case_is_retained(self):
        (self.root / 'one/src/lib.rs').write_text('#[test] fn fails_for_real() {assert_eq!(2+2,5);}\n')
        self.command('guest-list', ['cargo', 'test', '--workspace', '--locked', '--', '--list'])
        self.command('guest', ['cargo', 'test', '--workspace', '--locked', '--no-fail-fast'], expected=101)
        record = G.coverage(self.reports, 'guest', self.metadata, self.settings)
        self.assertEqual(record['outcomes'], {'pass': 2, 'fail': 1})
        self.assertIn('assertion `left == right` failed', (self.reports / 'guest.log').read_text())
        self.assertEqual(G.read(self.reports / 'guest.stage.json')['exit_code'], 101)
        self.command('second-fresh', ['cargo', 'test', '--workspace', '--locked', '--no-fail-fast'], expected=101)
        self.assertIn('fails_for_real ... FAILED', (self.reports / 'second-fresh.log').read_text())

    def test_cancellation_retains_actual_signal_and_partial_original_log(self):
        script = self.root / 'cancel.py'
        script.write_text('import os,signal,time\nprint("original cancellation output",flush=True)\ntime.sleep(60)\n')
        runner = self.root / 'runner.py'
        runner.write_text('import importlib.util,subprocess,sys\nfrom pathlib import Path\ns=importlib.util.spec_from_file_location("g",' + repr(str(Path(G.__file__))) + ');g=importlib.util.module_from_spec(s);s.loader.exec_module(g)\nsys.exit(g.S.stage(Path(' + repr(str(self.reports)) + '),"cancel",[sys.executable,' + repr(str(script)) + '],cwd=' + repr(str(self.root)) + ',stdin=subprocess.DEVNULL))\n')
        child = subprocess.Popen(['python3', str(runner)], env=self.env, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
        try:
            deadline = time.monotonic() + 15
            while time.monotonic() < deadline:
                log = self.reports / 'cancel.log'
                if log.exists() and 'original cancellation output' in log.read_text():
                    break
                if child.poll() is not None:
                    self.fail(child.stderr.read().decode())
                time.sleep(0.05)
            else:
                self.fail('Actual cancellation child did not start')
            child.send_signal(signal.SIGTERM)
            self.assertEqual(child.wait(timeout=10), 143)
            self.assertEqual(G.read(self.reports / 'cancel.stage.json')['exit_code'], -signal.SIGTERM)
            self.assertIn('original cancellation output', log.read_text())
        finally:
            if child.poll() is None:
                child.kill()
                child.wait()
            child.stderr.close()

    def test_real_production_guest_artifact_rejects_stale_missing_corrupt_and_foreign_inputs(self):
        original = Path(os.environ['SP1_GUEST_ARTIFACT'])
        settings = G.read(original / 'settings.json')
        if settings['provider'] == 'rwx':
            settings = {**settings, 'workspace_root': G.read(original / 'native-workspace/settings.json')['native_workspace_root']}
        artifact = self.root / 'producer'
        shutil.copytree(original, artifact)
        with patch.object(G, 'SP1', self.root):
            G.restore_artifact(artifact, settings)
            for name, value in [('source_sha', '0' * 40), ('branch', 'foreign-branch'), ('features', 'foreign-feature'), ('elf_source_sha', '0'*40), ('check_rustflags', 'changed'), ('tools', {})]:
                with self.subTest(name=name), self.assertRaisesRegex(ValueError, 'Stale'):
                    G.restore_artifact(artifact, {**settings, name: value})
            path = artifact / 'files' / (settings['programs'][0] + '-elf')
            data = path.read_bytes()
            path.unlink()
            with self.assertRaisesRegex(ValueError, 'corrupt'):
                G.restore_artifact(artifact, settings)
            path.write_bytes(data[:-1] + bytes([data[-1] ^ 1]))
            with self.assertRaisesRegex(ValueError, 'corrupt'):
                G.restore_artifact(artifact, settings)
            path.write_bytes(data)
            (artifact / 'files/extra-elf').write_bytes(data)
            with self.assertRaisesRegex(ValueError, 'unsealed'):
                G.restore_artifact(artifact, settings)
            (artifact / 'files/extra-elf').unlink()
            path.write_bytes(b'not an ELF')
            final = G.read(artifact / 'final.json')
            final['original_sha256'][str(path.relative_to(artifact))] = G.S.digest(path)
            G.S.write(artifact / 'final.json', final)
            with self.assertRaisesRegex(ValueError, 'executable'):
                G.restore_artifact(artifact, settings)
        G.S.write(self.reports / 'production-artifact.json', {'source_sha': settings['source_sha'], 'elfs': G.read(original / 'elfs.json'), 'original_sha256': G.read(original / 'final.json')['original_sha256']})

    def test_original_pin_checks_reject_actual_manifest_and_lock_drift_before_installation(self):
        source = Path(os.environ['SP1_GUEST_ARTIFACT']) / 'toolchain/source'
        for path in source.rglob('*'):
            if path.is_file():
                target = self.root / path.relative_to(source)
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(path, target)
        manifest = self.root / 'rust/Cargo.toml'
        original = manifest.read_text()
        changed, count = re.subn('(sp1-sdk\\s*=\\s*\\{[^}\\n]*version\\s*=\\s*")[^"]+', '\\g<1>0.0.0', original)
        self.assertEqual(count, 1)
        manifest.write_text(changed)
        self.command('manifest-pin-failure', ['bash', 'ops/ci/runtime/sp1-guest-toolchain.sh'], expected=1)
        log = (self.reports / 'manifest-pin-failure.log').read_text()
        self.assertIn('ERROR: SP1 version drift', log)
        self.assertNotIn('Restored SP1 toolchain', log)
        manifest.write_text(original)
        (self.root / 'rust/kona/sp1/programs/Cargo.lock').write_text('invalid original TOML = [\n')
        self.command('lock-pin-failure', ['bash', 'ops/ci/runtime/sp1-guest-toolchain.sh'], expected=1)
        self.assertIn('ERROR: SP1 version drift', (self.reports / 'lock-pin-failure.log').read_text())


if __name__ == '__main__': unittest.main()
