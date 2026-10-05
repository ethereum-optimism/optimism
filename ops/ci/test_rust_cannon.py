"""Exercise Cannon provenance, real subprocess exits and fresh VM verdicts."""
import contextlib
import gzip
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch
import xml.etree.ElementTree as ET

SCRIPTS = Path(__file__).resolve().parent
REPO = SCRIPTS.parent.parent
SPEC = importlib.util.spec_from_file_location('cannon_report', SCRIPTS / 'rust-cannon-report.py')
REPORT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(REPORT)
VARIANTS = [{'binary': 'kona-client', 'prestate_directory': 'prestate-artifacts-cannon'},
            {'binary': 'kona-client-int', 'prestate_directory': 'prestate-artifacts-cannon-interop'}]
ELF = b'\x7fELF\x02\x02\x01' + b'\0' * 11 + b'\0\x08' + b'compiled fixture'
CONFIG = json.loads((REPO / REPORT.CONFIG).read_text())


class ReportTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        previous = Path.cwd()
        os.chdir(self.root)
        self.addCleanup(os.chdir, previous)
        self.report = self.root / 'report'
        self.report.mkdir()
        self.bound = {'source_sha': 'a' * 40, 'input_sha256': {'input': 'b' * 64}, 'witness': CONFIG}
        bound = patch.object(REPORT, 'binding', return_value=self.bound)
        config = patch.object(REPORT, 'witness_config', return_value=CONFIG)
        bound.start(); config.start()
        self.addCleanup(bound.stop); self.addCleanup(config.stop)
        self.settings('offline')

    def settings(self, job, directory=None):
        REPORT.write((directory or self.report) / 'settings.json',
                     self.bound | {'suite': 'rust-cannon', 'job': job, 'variants': VARIANTS})

    def guest(self, **changes):
        state = {'exited': True, 'exitCode': 0, 'step': 12345, 'stateVersion': 8,
                 'witnessHash': '0x' + 'd' * 64, 'witness': '0x00010203'} | changes
        REPORT.write(self.report / 'guest.json', state)
        (self.report / 'out.bin.gz').write_bytes(gzip.compress(b'fresh state'))
        (self.report / 'offline.log').write_text('Successfully validated L2 block ' + CONFIG['l2_claim'])

    def dependency(self, kind='go'):
        producer = self.root / 'producer'
        producer.mkdir(exist_ok=True)
        self.settings(kind, producer)
        for name in ('variants.log', 'variants.stage.json', 'checks.junit.xml'):
            (producer / name).write_text('original')
        files = {}
        for name in REPORT.GO_FILES:
            path = self.root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(b'compiled')
            files[name] = {'sha256': REPORT.digest(path), 'size': path.stat().st_size}
        REPORT.write(producer / 'go-binaries.json', {'binding': self.bound, 'files': files})
        self.rehash(producer)
        return producer

    def rehash(self, directory):
        REPORT.write(directory / 'final.json', {'exit_code': 0, 'report_errors': [],
            'original_sha256': {str(p.relative_to(directory)): REPORT.digest(p)
                               for p in directory.rglob('*') if p.is_file() and p.name != 'final.json'}})

    def test_discovery_preserves_every_variant_and_rejects_errors(self):
        (self.report / 'variants.log').write_text('kona-client prestate-artifacts-cannon\nkona-client-int prestate-artifacts-cannon-interop\n')
        REPORT.inventory(self.report)
        self.assertEqual(REPORT.read(self.report / 'settings.json')['variants'], VARIANTS)
        for text in ('', 'compiler error', 'x prestate-artifacts-x\nx prestate-artifacts-y\n',
                     'x prestate-artifacts-x\ny prestate-artifacts-x\n', '../x prestate-artifacts-x'):
            (self.report / 'variants.log').write_text(text)
            with self.assertRaises(ValueError): REPORT.inventory(self.report)

    def test_all_mips_elf_variants_required(self):
        files = self.root / 'elfs'; files.mkdir()
        for v in VARIANTS: (files / v['binary']).write_bytes(ELF)
        REPORT.elf_manifest(self.report, files, 'elfs.json')
        self.assertEqual(set(REPORT.read(self.report / 'elfs.json')['files']), {'kona-client', 'kona-client-int'})
        (files / 'kona-client-int').unlink()
        with self.assertRaisesRegex(ValueError, 'Missing or extra'): REPORT.elf_manifest(self.report, files, 'elfs.json')
        (files / 'kona-client-int').write_bytes(b'not MIPS')
        with self.assertRaisesRegex(ValueError, 'Invalid ELF'): REPORT.elf_manifest(self.report, files, 'elfs.json')
        (files / 'extra').write_bytes(ELF)
        with self.assertRaisesRegex(ValueError, 'Missing or extra'): REPORT.elf_manifest(self.report, files, 'elfs.json')

    def test_guest_success_and_nonzero_exit_even_when_cli_succeeds(self):
        self.guest()
        REPORT.guest_report(self.report)
        evidence = REPORT.read(self.report / 'guest-coverage.json')
        self.assertTrue(evidence['fresh_execution'])
        self.assertEqual(evidence['boundary'], CONFIG)
        for state in ({'exitCode': 1}, {'exitCode': False}, {'exited': False}, {'exited': 1}):
            self.guest(**state)
            with self.assertRaisesRegex(ValueError, 'did not exit successfully'): REPORT.guest_report(self.report)

    def test_final_state_and_original_claim_are_required(self):
        for state in ({'step': 0}, {'step': True}, {'stateVersion': 7}, {'witness': '0x'}, {'witnessHash': 'bad'}):
            self.guest(**state)
            with self.assertRaisesRegex(ValueError, 'Invalid or empty'): REPORT.guest_report(self.report)
        self.guest(); (self.report / 'offline.log').write_text('Successfully validated L2 block 0xwrong')
        with self.assertRaisesRegex(ValueError, 'output-root'): REPORT.guest_report(self.report)
        self.guest(); (self.report / 'out.bin.gz').unlink()
        with self.assertRaises(FileNotFoundError): REPORT.guest_report(self.report)
        self.guest(); (self.report / 'out.bin.gz').write_bytes(b'stale JSON')
        with self.assertRaisesRegex(ValueError, 'compressed'): REPORT.guest_report(self.report)

    def test_producer_hashes_revision_selection_and_identity(self):
        p = self.dependency()
        REPORT.verify_dependency(self.report, p, 'go')
        path = Path(next(iter(REPORT.GO_FILES)))
        path.write_bytes(b'corrupt')
        with self.assertRaisesRegex(ValueError, 'binary checksum'): REPORT.verify_dependency(self.report, p, 'go')
        self.dependency()
        manifest = REPORT.read(p / 'go-binaries.json')
        manifest['binding'] = self.bound | {'source_sha': 'c' * 40}
        REPORT.write(p / 'go-binaries.json', manifest); self.rehash(p)
        with self.assertRaisesRegex(ValueError, 'toolchain mismatch'): REPORT.verify_dependency(self.report, p, 'go')
        self.dependency(); settings = REPORT.read(p / 'settings.json')
        settings['variants'] = VARIANTS[:1]
        REPORT.write(p / 'settings.json', settings); self.rehash(p)
        with self.assertRaisesRegex(ValueError, 'selection mismatch'): REPORT.verify_dependency(self.report, p, 'go')
        self.dependency(); self.settings('build', p); self.rehash(p)
        with self.assertRaisesRegex(ValueError, 'identity mismatch'): REPORT.verify_dependency(self.report, p, 'go')

    def test_incomplete_or_changed_originals_and_binary_inventory_rejected(self):
        p = self.dependency(); (p / 'variants.log').write_text('changed')
        with self.assertRaisesRegex(ValueError, 'original artifact checksum'): REPORT.verify_dependency(self.report, p, 'go')
        self.dependency(); final = REPORT.read(p / 'final.json'); final['original_sha256'] = {}
        REPORT.write(p / 'final.json', final)
        with self.assertRaisesRegex(ValueError, 'Incomplete'): REPORT.verify_dependency(self.report, p, 'go')
        self.dependency(); manifest = REPORT.read(p / 'go-binaries.json'); manifest['files'] = {}
        REPORT.write(p / 'go-binaries.json', manifest); self.rehash(p)
        with self.assertRaisesRegex(ValueError, 'inventory mismatch'): REPORT.verify_dependency(self.report, p, 'go')

    def test_build_owns_the_image_handoff_after_a_cold_rebuild(self):
        producer = self.root / 'build-producer'; producer.mkdir()
        self.settings('build', producer)
        for name in ('variants.log', 'variants.stage.json', 'checks.junit.xml'):
            (producer / name).write_text('original')
        target = self.root / 'rust/target/mips64-unknown-none/release-client-lto'
        target.mkdir(parents=True)
        for variant in VARIANTS: (target / variant['binary']).write_bytes(ELF)
        REPORT.elf_manifest(self.report, target, 'elfs.json')
        shutil.copyfile(self.report / 'elfs.json', producer / 'elfs.json')
        image = {'binding': self.bound, 'image_id': 'sha256:' + 'b' * 64}
        REPORT.write(producer / 'image.json', image); self.rehash(producer)
        with patch.object(REPORT, 'command', return_value=json.dumps([{'Id': image['image_id']}])):
            REPORT.verify_dependency(self.report, producer, 'build')
            bad = image | {'binding': self.bound | {'source_sha': 'c' * 40}}
            REPORT.write(producer / 'image.json', bad); self.rehash(producer)
            with self.assertRaisesRegex(ValueError, 'image revision'):
                REPORT.verify_dependency(self.report, producer, 'build')
        REPORT.write(producer / 'image.json', image); self.rehash(producer)
        with patch.object(REPORT, 'command', return_value=json.dumps([{'Id': 'sha256:' + 'a' * 64}])):
            with self.assertRaisesRegex(ValueError, 'Docker image mismatch'):
                REPORT.verify_dependency(self.report, producer, 'build')
        (producer / 'image.json').unlink(); self.rehash(producer)
        with self.assertRaisesRegex(ValueError, 'Incomplete'):
            REPORT.verify_dependency(self.report, producer, 'build')

    def test_failure_retains_original_exit_logs_nested_artifacts_and_one_guest_case(self):
        (self.report / 'offline.log').write_text('first failure, never retry')
        (self.report / 'elfs').mkdir(); (self.report / 'elfs/kona-client').write_bytes(ELF)
        self.assertEqual(REPORT.finish(self.report, 7), 7)
        final = REPORT.read(self.report / 'final.json')
        self.assertEqual(final['original_sha256']['offline.log'], REPORT.digest(self.report / 'offline.log'))
        self.assertIn('elfs/kona-client', final['original_sha256'])
        cases = ET.parse(self.report / 'checks.junit.xml').findall('.//testcase')
        self.assertEqual(len(cases), 1)
        self.assertEqual(cases[0].get('name'), 'kona-client/11155420/47600000')
        self.assertIsNotNone(cases[0].find('failure'))

    def test_missing_stages_fail_and_failed_discovery_reports_a_case(self):
        with contextlib.redirect_stderr(__import__('io').StringIO()): self.assertTrue(REPORT.finish(self.report, 0))
        self.settings('env')
        settings = REPORT.read(self.report / 'settings.json'); settings.pop('variants')
        REPORT.write(self.report / 'settings.json', settings)
        self.assertEqual(REPORT.finish(self.report, 9), 9)
        self.assertEqual(len(ET.parse(self.report / 'checks.junit.xml').findall('.//failure')), 1)


class RunnerTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory(); self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        for name in REPORT.INPUTS:
            target = self.root / name; target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(REPO / name, target)
        for tree in ('op-service', 'op-preimage', 'op-core/nuts/bundles'):
            path = self.root / tree / 'input'; path.parent.mkdir(parents=True, exist_ok=True); path.write_text('pinned')
        config = CONFIG | {'sha256': hashlib.sha256(b'fixture witness').hexdigest()}
        REPORT.write(self.root / REPORT.CONFIG, config)
        stubs = self.root / 'stubs'; stubs.mkdir()
        stub = stubs / 'stub'
        stub.write_text('''#!/usr/bin/env python3
import gzip,json,os,pathlib,sys,time
root=pathlib.Path(os.environ['FIXTURE_ROOT']); tool=pathlib.Path(sys.argv[0]).name; args=sys.argv[1:]
elf=b'\\x7fELF\\x02\\x02\\x01'+b'\\0'*11+b'\\0\\x08'+b'compiled fixture'
config=json.loads((root/'ops/ci/cannon-witness.json').read_text())
if tool in ('rustc','cargo','go'): print(tool+' pinned'); sys.exit(0)
if tool=='curl':
    path=pathlib.Path(args[args.index('-o')+1]); path.parent.mkdir(parents=True,exist_ok=True); path.write_bytes(b'fixture witness')
    sys.exit(int(os.environ.get('CURL_EXIT','0')))
if tool=='docker':
    if args[:2]==['image','inspect']:
        image=(root/'image-id').read_text() if (root/'image-id').exists() else 'sha256:'+'e'*64
        print(json.dumps([{'Id':image,'RepoDigests':['base@sha256:'+'f'*64]}]))
    elif args[0]=='create': print('fixture-container')
    elif args[0]=='run': print('nightly pinned; gcc pinned')
    elif args[0]=='cp':
        dest=pathlib.Path(args[-1]); dest.mkdir(exist_ok=True)
        for name in ('kona-client','kona-client-int'): (dest/name).write_bytes(elf)
    sys.exit(0)
if tool=='just':
    recipe=args[0]
    if recipe=='kona-prestate-variants': print('kona-client prestate-artifacts-cannon\\nkona-client-int prestate-artifacts-cannon-interop')
    elif recipe=='cannon':
        for name in ('cannon/bin/cannon','cannon/bin/cannon64-impl','cannon/multicannon/embeds/cannon-8'):
            p=root/name; p.parent.mkdir(parents=True,exist_ok=True); p.write_text((root/'stubs/stub').read_text()); p.chmod(0o755)
    elif recipe=='build-cannon-client':
        (root/'image-id').write_text('sha256:'+'b'*64)
        dest=root/'rust/target/mips64-unknown-none/release-client-lto'; dest.mkdir(parents=True,exist_ok=True)
        for name in ('kona-client','kona-client-int'): (dest/name).write_bytes(elf)
    elif recipe=='run-client-cannon-offline':
        (root/'invocations.json').write_text(json.dumps(args))
        if os.environ.get('WAIT_FOR_CANCEL'):
            (root/'running').touch(); print('guest running',flush=True); time.sleep(60)
        sys.exit_code=int(os.environ.get('OFFLINE_EXIT','0'))
        if sys.exit_code: print('original compile or runtime failure',flush=True); sys.exit(sys.exit_code)
        dest=root/'rust/target/mips64-unknown-none/release-client-lto'; dest.mkdir(parents=True,exist_ok=True)
        for name in ('kona-client','kona-client-int'): (dest/name).write_bytes(elf)
        (root/'rust/kona/out.bin.gz').write_bytes(gzip.compress(b'fresh final state'))
        print('Successfully validated L2 block '+config['l2_claim'])
    sys.exit(0)
if tool=='cannon':
    print(json.dumps({'exited':True,'exitCode':int(os.environ.get('GUEST_EXIT','0')),'step':12345,'stateVersion':8,'witnessHash':'0x'+'d'*64,'witness':'0x00010203'}))
''')
        stub.chmod(0o755)
        for name in ('just', 'docker', 'rustc', 'cargo', 'go', 'curl'): (stubs / name).symlink_to(stub)
        subprocess.run(['git', 'init', '-q'], cwd=self.root, check=True)
        subprocess.run(['git', 'add', '.'], cwd=self.root, check=True)
        subprocess.run(['git', '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.test', 'commit', '-qm', 'fixture'], cwd=self.root, check=True)
        sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=self.root, text=True).strip()
        self.env = dict(os.environ, PATH=str(stubs) + ':' + os.environ['PATH'], CI_RUST_PROVIDER='circleci',
                        CI_COMMIT_SHA=sha, CIRCLE_SHA1=sha, FIXTURE_ROOT=str(self.root))
        self.report = self.root / '.ci/rust-workspace/cannon-offline'

    def run_offline(self, **env):
        return subprocess.run(['bash', 'ops/ci/rust-cannon.sh', 'offline'], cwd=self.root,
                              env=self.env | env, capture_output=True, text=True, timeout=15)

    def test_fresh_state_original_parameters_and_nonzero_guest_failure(self):
        stale = self.root / 'rust/kona/out.bin.gz'; stale.write_bytes(b'stale state')
        run = self.run_offline()
        self.assertEqual(run.returncode, 0, run.stdout + run.stderr)
        self.assertNotEqual(stale.read_bytes(), b'stale state')
        args = json.loads((self.root / 'invocations.json').read_text())
        keys = ('block_number', 'l2_claim', 'l2_output_root', 'l2_head', 'l1_head', 'l2_chain_id', 'filename', 'release_url')
        self.assertEqual(args, ['run-client-cannon-offline'] + [CONFIG[k] for k in keys])
        run = self.run_offline(GUEST_EXIT='1')
        self.assertNotEqual(run.returncode, 0)
        final = REPORT.read(self.report / 'final.json')
        self.assertIn('Cannon guest did not exit successfully', final['report_errors'])
        self.assertEqual(final['stages']['offline']['exit_code'], 0)
        self.assertEqual(final['stages']['guest']['exit_code'], 0)
        self.assertIsNotNone(ET.parse(self.report / 'checks.junit.xml').find('.//failure'))

    def test_offline_consumes_the_build_image_instead_of_the_initial_environment(self):
        for job, extra in [('env', {}), ('build', {
            'CANNON_ENV_ARTIFACT': str(self.root / '.ci/rust-workspace/cannon-env')})]:
            run = subprocess.run(['bash', 'ops/ci/rust-cannon.sh', job], cwd=self.root,
                                 env=self.env | extra, capture_output=True, text=True, timeout=15)
            self.assertEqual(run.returncode, 0, run.stdout + run.stderr)
        producer = self.root / '.ci/rust-workspace/cannon-build'
        environment = self.root / '.ci/rust-workspace/cannon-env'
        self.assertNotEqual(REPORT.read(environment / 'image.json')['image_id'],
                            REPORT.read(producer / 'image.json')['image_id'])
        inputs = {'CANNON_ENV_ARTIFACT': str(environment), 'CANNON_BUILD_ARTIFACT': str(producer)}
        run = self.run_offline(**inputs)
        self.assertEqual(run.returncode, 0, run.stdout + run.stderr)
        self.assertTrue((self.report / 'verified-build.json').exists())
        self.assertFalse((self.report / 'verified-env.json').exists())
        (self.root / 'image-id').write_text('sha256:' + 'e' * 64)
        run = self.run_offline(**inputs)
        self.assertNotEqual(run.returncode, 0)
        self.assertIn('Inherited Cannon Docker image mismatch', run.stderr)
        self.assertNotIn('offline', REPORT.read(self.report / 'final.json')['stages'])

    def test_first_failure_not_retried_and_report_collected(self):
        run = self.run_offline(OFFLINE_EXIT='7')
        self.assertEqual(run.returncode, 7, run.stdout + run.stderr)
        final = REPORT.read(self.report / 'final.json')
        self.assertEqual(final['exit_code'], 7)
        self.assertEqual(final['stages']['offline']['exit_code'], 7)
        self.assertNotIn('guest', final['stages'])
        self.assertIn('original compile or runtime failure', (self.report / 'offline.log').read_text())

    def test_missing_public_witness_fails_before_builds(self):
        run = self.run_offline(CURL_EXIT='22')
        self.assertEqual(run.returncode, 22)
        self.assertEqual(set(REPORT.read(self.report / 'final.json')['stages']), {'variants', 'witness'})
        self.assertFalse((self.root / 'cannon/bin/cannon').exists())

    def test_missing_archive_discards_bodyless_etag(self):
        path = self.root / 'rust/kona/bin/client/testdata' / CONFIG['filename']
        path.parent.mkdir(parents=True, exist_ok=True)
        sidecar = Path(str(path) + '.etag'); sidecar.write_text('old etag')
        run = subprocess.run(['bash', 'rust/kona/bin/client/scripts/fetch-witness-tar.sh', CONFIG['filename'], CONFIG['release_url']],
                             cwd=self.root, env=self.env, capture_output=True, text=True)
        self.assertEqual(run.returncode, 0, run.stderr)
        self.assertFalse(sidecar.exists())
        self.assertEqual(path.read_bytes(), b'fixture witness')

    @unittest.skipUnless(os.name == 'posix', 'POSIX cancellation')
    def test_cancellation_preserves_signal_and_original_stream(self):
        process = subprocess.Popen(['bash', 'ops/ci/rust-cannon.sh', 'offline'], cwd=self.root,
            env=self.env | {'WAIT_FOR_CANCEL': '1'}, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            text=True, start_new_session=True)
        try:
            deadline = time.monotonic() + 10
            while not (self.root / 'running').exists() and process.poll() is None and time.monotonic() < deadline:
                time.sleep(.05)
            self.assertTrue((self.root / 'running').exists())
            os.killpg(process.pid, signal.SIGTERM)
            process.communicate(timeout=10)
            final = REPORT.read(self.report / 'final.json')
            self.assertEqual(final['exit_code'], 143)
            self.assertEqual(final['stages']['offline']['exit_code'], -signal.SIGTERM)
            self.assertIn('guest running', (self.report / 'offline.log').read_text())
        finally:
            if process.poll() is None:
                os.killpg(process.pid, signal.SIGKILL); process.communicate()


if __name__ == '__main__': unittest.main()
