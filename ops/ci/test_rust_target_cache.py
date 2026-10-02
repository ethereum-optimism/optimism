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

SPEC = importlib.util.spec_from_file_location('rust_cache', Path(__file__).with_name('rust-target-cache.py'))
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
        self.source.write_text('pub trait Factory {}')
        os.utime(self.source, ns=(old, old))
        self.run_phase('prepare')
        self.assertGreater(self.source.stat().st_mtime_ns, old)
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
