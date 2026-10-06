import contextlib
import io
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('rust_cache', Path(__file__).resolve().parents[1] / 'runtime' / 'rust-target-cache.py')
CACHE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CACHE)


class RustTargetCacheTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        subprocess.run(['git', 'init', '-q', str(self.root)], check=True)
        self.source = self.root / 'rust/src/lib.rs'
        self.source.parent.mkdir(parents=True)
        self.source.write_text('pub fn old() {}')
        (self.root / 'mise.toml').write_text('[tools]\nrust="1.95.0"\n')
        subprocess.run(['git', 'add', '.'], cwd=self.root, check=True)
        self.target = self.root / 'rust/target'
        self.target.mkdir()
        (self.root / '.git/info/exclude').write_text('rust/target/\n')

    def run_phase(self, phase):
        with contextlib.redirect_stdout(io.StringIO()):
            CACHE.manage(phase, self.root, self.target)

    def test_changed_content_refreshes_normalized_mtimes_and_keeps_targets(self):
        self.run_phase('prepare'); self.run_phase('commit')
        binary = self.target / 'compiled-dependency'
        binary.write_text('retained object')
        old = self.source.stat().st_mtime_ns
        toolchain = self.root / 'mise.toml'
        toolchain_stamp = toolchain.stat().st_mtime_ns
        self.source.write_text('pub trait Factory {}')
        os.utime(self.source, ns=(old, old))
        self.run_phase('prepare')
        self.assertGreater(self.source.stat().st_mtime_ns, old)
        self.assertEqual(toolchain.stat().st_mtime_ns, toolchain_stamp)
        self.assertTrue(binary.exists())
        self.assertFalse((self.target / '.rwx-source-fingerprint.json').exists())

    def test_unchanged_source_restores_stable_mtimes_for_target_reuse(self):
        self.run_phase('prepare'); self.run_phase('commit')
        stamp = self.source.stat().st_mtime_ns
        os.utime(self.source, ns=(1000000000, 1000000000))
        self.run_phase('prepare')
        self.assertEqual(self.source.stat().st_mtime_ns, stamp)

    def test_failed_build_retries_refresh_source_again(self):
        self.run_phase('prepare')
        os.utime(self.source, ns=(1000000000, 1000000000))
        self.run_phase('prepare')
        self.assertGreater(self.source.stat().st_mtime_ns, 1000000000)

    def test_failed_build_then_rollback_does_not_restore_successful_timestamp(self):
        self.run_phase('prepare'); self.run_phase('commit')
        old = self.source.stat().st_mtime_ns
        self.source.write_text('pub fn changed() {}')
        self.run_phase('prepare')
        self.source.write_text('pub fn old() {}')
        self.run_phase('prepare')
        self.assertGreater(self.source.stat().st_mtime_ns, old)

    def test_legacy_cache_refreshes_once_then_preserves_per_file_timestamps(self):
        (self.target / '.rwx-source-fingerprint.json').write_text(json.dumps({
            'source_sha256': 'legacy', 'source_mtime_ns': 1000000000}))
        self.run_phase('prepare'); self.run_phase('commit')
        stamp = self.source.stat().st_mtime_ns
        state = json.loads((self.target / '.rwx-source-fingerprint.json').read_text())
        self.assertEqual(state['version'], 3)
        self.run_phase('prepare')
        self.assertEqual(self.source.stat().st_mtime_ns, stamp)

    def test_added_deleted_and_reintroduced_files_preserve_other_timestamps(self):
        self.run_phase('prepare'); self.run_phase('commit')
        original = self.source.stat().st_mtime_ns
        new = self.source.with_name('new.rs')
        new.write_text('pub struct New;')
        self.run_phase('prepare'); self.run_phase('commit')
        added = new.stat().st_mtime_ns
        new.unlink()
        self.run_phase('prepare'); self.run_phase('commit')
        state = json.loads((self.target / '.rwx-source-fingerprint.json').read_text())
        self.assertNotIn('rust/src/new.rs', state['files'])
        new.write_text('pub struct New;')
        os.utime(new, ns=(added, added))
        self.run_phase('prepare')
        self.assertGreater(new.stat().st_mtime_ns, added)
        self.assertEqual(self.source.stat().st_mtime_ns, original)

    def test_commit_rejects_source_changed_during_build(self):
        self.run_phase('prepare')
        self.source.write_text('changed during compilation')
        with self.assertRaises(ValueError): self.run_phase('commit')

    def test_new_untracked_source_and_toolchain_changes_invalidate(self):
        self.run_phase('prepare'); self.run_phase('commit')
        before = json.loads((self.target / '.rwx-source-fingerprint.json').read_text())
        (self.source.parent / 'new.rs').write_text('pub struct New;')
        (self.root / 'mise.toml').write_text('[tools]\nrust="1.96.0"\n')
        self.run_phase('prepare'); self.run_phase('commit')
        self.assertNotEqual(before, json.loads((self.target / '.rwx-source-fingerprint.json').read_text()))

    def test_embedded_nut_bundle_refreshes_sources_and_rejects_mid_build_change(self):
        bundle = self.root / 'op-core/nuts/bundles/fixture.json'
        bundle.parent.mkdir(parents=True); bundle.write_text('old embedded payload')
        self.run_phase('prepare'); self.run_phase('commit')
        old = bundle.stat().st_mtime_ns
        source_stamp = self.source.stat().st_mtime_ns
        before = json.loads((self.target / '.rwx-source-fingerprint.json').read_text())['source_sha256']
        bundle.write_text('new embedded payload'); os.utime(bundle, ns=(old, old))
        self.run_phase('prepare')
        self.assertGreater(bundle.stat().st_mtime_ns, old)
        self.assertEqual(self.source.stat().st_mtime_ns, source_stamp)
        self.assertNotEqual(before, json.loads((self.target / '.rwx-source-pending.json').read_text())['source_sha256'])
        bundle.write_text('changed while building')
        with self.assertRaises(ValueError): self.run_phase('commit')

    @unittest.skipUnless(shutil.which('cargo'), 'Cargo required for embedded bundle freshness reproduction')
    def test_cargo_rebuilds_external_embedded_bundle_and_reuses_unchanged_target(self):
        bundle = self.root / 'op-core/nuts/bundles/fixture.txt'
        bundle.parent.mkdir(parents=True); bundle.write_text('old bundle')
        (self.root / 'rust/Cargo.toml').write_text('[package]\nname="external-bundle"\nversion="0.1.0"\nedition="2021"\n')
        self.source.write_text('')
        (self.source.parent / 'main.rs').write_text('fn main() { print!("{}", include_str!("../../op-core/nuts/bundles/fixture.txt")); }')
        env = {**os.environ, 'CARGO_TARGET_DIR': str(self.target)}
        def build(): return subprocess.run(['cargo', 'build', '--offline'], cwd=self.root / 'rust', env=env, text=True, capture_output=True)
        subprocess.run(['cargo', 'generate-lockfile', '--offline'], cwd=self.root / 'rust', env=env, check=True, capture_output=True)
        self.run_phase('prepare')
        first = build(); self.assertEqual(first.returncode, 0, first.stderr); self.run_phase('commit')
        binary = self.target / 'debug/external-bundle'
        self.assertEqual(subprocess.check_output([str(binary)], text=True), 'old bundle')
        old = bundle.stat().st_mtime_ns
        bundle.write_text('new bundle'); os.utime(bundle, ns=(old, old))
        stale = build(); self.assertEqual(stale.returncode, 0, stale.stderr)
        self.assertEqual(subprocess.check_output([str(binary)], text=True), 'old bundle')
        self.run_phase('prepare')
        fresh = build(); self.assertEqual(fresh.returncode, 0, fresh.stderr); self.run_phase('commit')
        self.assertEqual(subprocess.check_output([str(binary)], text=True), 'new bundle')
        self.run_phase('prepare')
        reused = build(); self.assertEqual(reused.returncode, 0, reused.stderr)
        self.assertNotIn('Compiling external-bundle', reused.stderr)

    @unittest.skipUnless(shutil.which('cargo'), 'Cargo required for cache freshness reproduction')
    def test_cargo_rebuilds_changed_crate_and_keeps_unrelated_crate_fresh(self):
        for incremental in ('0', '1'):
            with self.subTest(incremental=incremental):
                target = self.target / incremental
                for name in ('provider', 'unrelated', 'consumer'):
                    folder = self.root / 'rust' / name
                    (folder / 'src').mkdir(parents=True, exist_ok=True)
                    dependencies = '[dependencies]\nprovider={path="../provider"}\nunrelated={path="../unrelated"}\n' if name == 'consumer' else ''
                    (folder / 'Cargo.toml').write_text(f'[package]\nname="{name}"\nversion="0.1.0"\nedition="2021"\n{dependencies}')
                (self.root / 'rust/Cargo.toml').write_text('[workspace]\nmembers=["provider","unrelated","consumer"]\nresolver="2"\n')
                provider = self.root / 'rust/provider/src/lib.rs'
                unrelated = self.root / 'rust/unrelated/src/lib.rs'
                provider.write_text('pub fn value() -> u8 { 1 }')
                unrelated.write_text('pub fn value() -> u8 { 10 }')
                (self.root / 'rust/consumer/src/main.rs').write_text('fn main() { print!("{}", provider::value() + unrelated::value()); }')
                env = {**os.environ, 'CARGO_TARGET_DIR': str(target)}
                if incremental == '1':
                    env.pop('CARGO_INCREMENTAL', None)
                    env.pop('CARGO_BUILD_INCREMENTAL', None)
                    env['CARGO_PROFILE_DEV_INCREMENTAL'] = 'true'
                else:
                    env['CARGO_INCREMENTAL'] = '0'
                def build():
                    result = subprocess.run(['cargo', 'build', '--offline', '--message-format=json'],
                        cwd=self.root / 'rust', env=env, text=True, capture_output=True)
                    self.assertEqual(result.returncode, 0, result.stderr)
                    return {row['target']['name']: row['fresh'] for row in
                            (json.loads(line) for line in result.stdout.splitlines())
                            if row['reason'] == 'compiler-artifact'}
                subprocess.run(['cargo', 'generate-lockfile', '--offline'], cwd=self.root / 'rust',
                               env=env, check=True, capture_output=True)
                with contextlib.redirect_stdout(io.StringIO()):
                    CACHE.manage('prepare', self.root, target)
                    build()
                    CACHE.manage('commit', self.root, target)
                stamp = unrelated.stat().st_mtime_ns
                old = provider.stat().st_mtime_ns
                # Reproduce normalized checkout mtimes, including a changed file
                # whose timestamp would otherwise falsely match the cached source.
                provider.write_text('pub fn value() -> u8 { 2 }')
                for path in (provider, unrelated):
                    os.utime(path, ns=(old, old))
                with contextlib.redirect_stdout(io.StringIO()):
                    CACHE.manage('prepare', self.root, target)
                    artifacts = build()
                    CACHE.manage('commit', self.root, target)
                self.assertFalse(artifacts['provider'])
                self.assertFalse(artifacts['consumer'])
                self.assertTrue(artifacts['unrelated'])
                self.assertEqual(unrelated.stat().st_mtime_ns, stamp)
                self.assertEqual(subprocess.check_output([str(target / 'debug/consumer')], text=True), '12')

    @unittest.skipUnless(shutil.which('cargo'), 'Cargo required for cache freshness reproduction')
    def test_cargo_rebuilds_changed_dependency_with_older_checkout_mtime(self):
        for name in ['provider','consumer']:
            folder=self.root/'rust'/name; (folder/'src').mkdir(parents=True)
            (folder/'Cargo.toml').write_text('[package]\nname="'+name+'"\nversion="0.1.0"\nedition="2021"\n'+('[dependencies]\nprovider={path="../provider"}\n' if name=='consumer' else ''))
        (self.root/'rust/Cargo.toml').write_text('[workspace]\nmembers=["provider","consumer"]\nresolver="2"\n')
        provider=self.root/'rust/provider/src/lib.rs'; consumer=self.root/'rust/consumer/src/lib.rs'
        provider.write_text('pub fn old() {}'); consumer.write_text('pub fn caller() {}')
        env={**os.environ,'CARGO_TARGET_DIR':str(self.target)}
        def check(): return subprocess.run(['cargo','check','--offline','-p','consumer'],cwd=self.root/'rust',env=env,text=True,capture_output=True)
        subprocess.run(['cargo','generate-lockfile','--offline'],cwd=self.root/'rust',env=env,check=True,capture_output=True)
        self.run_phase('prepare'); self.assertEqual(check().returncode,0); self.run_phase('commit')
        old=provider.stat().st_mtime_ns
        provider.write_text('pub trait Factory {}'); os.utime(provider,ns=(old,old))
        consumer.write_text('pub fn caller<T: provider::Factory>() {}')
        stale=check()
        self.assertNotEqual(stale.returncode,0,stale.stderr)
        self.assertIn('Factory',stale.stderr)
        self.run_phase('prepare')
        fresh=check(); self.assertEqual(fresh.returncode,0,fresh.stderr)
        self.run_phase('commit')
        os.utime(provider,ns=(1000000000,1000000000))
        self.run_phase('prepare')
        reused=check(); self.assertEqual(reused.returncode,0,reused.stderr)
        self.assertNotIn('Checking provider',reused.stderr)


if __name__ == '__main__': unittest.main()
