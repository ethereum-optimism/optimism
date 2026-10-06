#!/usr/bin/env python3
"""Validate exhaustive release selection against real Cargo output and failures."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SCRIPTS = Path(__file__).resolve().parents[1] / 'runtime'
SPEC = importlib.util.spec_from_file_location('release_report', Path(__file__).resolve().parents[1] / 'runtime' / 'rust-e2e-release-report.py')
REPORT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(REPORT)


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.report = self.root / 'report'
        self.report.mkdir()
        self.binding = {'source_sha': 'a' * 40, 'profile': 'release'}
        self.packages, self.messages = [], []
        for name in ('kona-host', 'kona-node', 'op-reth', 'utility'):
            target = {'name': name, 'kind': ['bin'], 'required-features': []}
            self.packages.append({'id': name, 'name': name, 'features': {}, 'targets': [target]})
            path = self.root / 'rust/target/release' / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(b'compiled ' + name.encode())
            self.messages.append({'reason': 'compiler-artifact', 'package_id': name, 'target': target,
                'features': [], 'filenames': [str(path)], 'executable': str(path), 'fresh': False,
                'profile': {'test': False, 'opt_level': '3'}})
        self.files()

    def files(self):
        REPORT.write(self.report / 'settings.json', self.binding | {'workspace_root': str(self.root)})
        REPORT.write(self.report / 'workspace.json', {'packages': self.packages,
            'workspace_members': [p['id'] for p in self.packages]})
        (self.report / 'build.json').write_text('\n'.join(json.dumps(m) for m in
            self.messages + [{'reason': 'build-finished', 'success': True}]) + '\n')

    def validate(self):
        with patch.object(REPORT, 'binding', return_value=self.binding):
            return REPORT.artifacts(self.report)

    def test_complete_workspace_includes_utility_binary(self):
        self.assertEqual(set(self.validate()), {'kona-host', 'kona-node', 'op-reth', 'utility'})
        self.assertEqual(len(REPORT.read(self.report / 'coverage.json')['targets']), 4)

    def test_build_dependency_unit_is_retained_without_replacing_workspace_target(self):
        target = {'name': 'shared', 'kind': ['lib']}
        self.packages.append({'id': 'shared', 'name': 'shared', 'features': {}, 'targets': [target]})
        primary = self.root / 'rust/target/release/libshared.rlib'; primary.write_bytes(b'workspace')
        build = self.root / 'rust/target/release/deps/libshared-build.rlib'
        build.parent.mkdir(); build.write_bytes(b'build dependency')
        base = {'reason': 'compiler-artifact', 'package_id': 'shared', 'target': target,
                'features': [], 'executable': None, 'fresh': False}
        primary_unit = base | {'filenames': [str(primary)], 'profile': {'test': False, 'opt_level': '3'}}
        build_unit = base | {'filenames': [str(build)], 'profile': {'test': False, 'opt_level': '0'}}
        self.messages += [build_unit, primary_unit]; self.files(); self.validate()
        coverage = REPORT.read(self.report / 'coverage.json')
        self.assertEqual({t['role'] for t in coverage['targets'] if t['target'] == 'shared'}, {'workspace', 'build-dependency'})
        self.messages.remove(primary_unit); self.files()
        with self.assertRaisesRegex(ValueError, 'coverage'): self.validate()
        self.messages += [primary_unit, build_unit]; self.files()
        with self.assertRaisesRegex(ValueError, 'Duplicate'): self.validate()

    def test_missing_extra_duplicate_target_or_completion_rejected(self):
        original = list(self.messages)
        for mode in ('missing', 'extra', 'duplicate', 'completion'):
            self.messages = list(original)
            if mode == 'missing': self.messages.pop()
            if mode == 'extra': self.messages[0] = self.messages[0] | {'target': {'name': 'other', 'kind': ['bin']}}
            if mode == 'duplicate': self.messages.append(self.messages[0])
            self.files()
            if mode == 'completion':
                path = self.report / 'build.json'
                path.write_text(path.read_text().rsplit('\n', 2)[0] + '\n')
            with self.subTest(mode=mode), self.assertRaises(ValueError): self.validate()

    def test_feature_gated_target_exclusion_is_explicit(self):
        self.packages[0]['features'] = {'default': ['alias'], 'alias': ['enabled']}
        self.packages[0]['targets'].append({'name': 'disabled', 'kind': ['bin'], 'required-features': ['optional']})
        self.files()
        self.validate()
        self.assertEqual(REPORT.default_features(self.packages[0]), {'default', 'alias', 'enabled'})
        self.assertEqual(REPORT.read(self.report / 'coverage.json')['excluded_feature_gated_targets'][0]['target'], 'disabled')

    def test_missing_unsafe_artifact_and_wrong_profile_rejected(self):
        original = list(self.messages)
        for mode in ('missing', 'unsafe', 'profile'):
            self.messages = list(original)
            if mode in ('missing', 'unsafe'):
                path = self.root / ('rust/target/release/missing' if mode == 'missing' else 'other')
                if mode == 'unsafe': path.write_bytes(b'outside target')
                self.messages[0] = self.messages[0] | {'filenames': [str(path)]}
            else: self.messages[0] = self.messages[0] | {'profile': {'test': True, 'opt_level': '3'}}
            self.files()
            with self.subTest(mode=mode), self.assertRaises(ValueError): self.validate()

    def test_build_failure_preserves_original_status_and_hashes(self):
        (self.report / 'build.log').write_text('original compiler failure')
        self.assertEqual(REPORT.finish(self.report, 7), 7)
        final = REPORT.read(self.report / 'final.json')
        self.assertEqual(final['exit_code'], 7)
        self.assertEqual(final['original_sha256']['build.log'], REPORT.digest(self.report / 'build.log'))
        self.assertIn('<failure', (self.report / 'checks.junit.xml').read_text())

    def test_success_requires_same_binding_and_successful_stages(self):
        for mode in ('source', 'stage', 'dependency'):
            self.files()
            for name in ('workspace', 'build'): REPORT.write(self.report / (name + '.stage.json'), {'exit_code': 0})
            REPORT.write(self.report / 'dependency.json', {'commit_sha': self.binding['source_sha'], 'kind': 'rust-e2e-release'})
            if mode == 'source': REPORT.write(self.report / 'settings.json', {'source_sha': 'stale'})
            if mode == 'stage': REPORT.write(self.report / 'build.stage.json', {'exit_code': 1})
            if mode == 'dependency': REPORT.write(self.report / 'dependency.json', {'commit_sha': 'stale', 'kind': 'rust-e2e-release'})
            with self.subTest(mode=mode), patch.object(REPORT, 'binding', return_value=self.binding):
                self.assertNotEqual(REPORT.finish(self.report, 0), 0)
                self.assertTrue(REPORT.read(self.report / 'final.json')['report_errors'])


@unittest.skipUnless(os.environ.get('RWX_LIVE_RUST_FIXTURE') == '1', 'Opt-in real Cargo release fixture')
class LiveReleaseTests(unittest.TestCase):
    def test_full_default_workspace_build_and_fresh_target_reuse(self):
        if not shutil.which('cargo'): self.fail('Cargo is required for the live fixture')
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); rust = root / 'rust'; rust.mkdir()
            names = ['kona-host', 'kona-node', 'op-reth', 'utility']
            (rust / 'Cargo.toml').write_text('[workspace]\nresolver="2"\nmembers=' + json.dumps(names + ['shared']) +
                '\n[profile.release]\nopt-level=3\n')
            for name in names:
                crate = rust / name; (crate / 'src').mkdir(parents=True)
                (crate / 'Cargo.toml').write_text(f'[package]\nname="{name}"\nversion="0.1.0"\nedition="2021"\n[features]\ndefault=[]\n')
                (crate / 'src/main.rs').write_text('fn main() { println!("fresh executable"); }\n')
            shared = rust / 'shared'; (shared / 'src').mkdir(parents=True)
            (shared / 'Cargo.toml').write_text('[package]\nname="shared"\nversion="0.1.0"\nedition="2021"\n'
                '[features]\ndefault=["normal"]\nnormal=[]\n')
            (shared / 'src/lib.rs').write_text('pub fn build() {}\n')
            with (rust / 'utility/Cargo.toml').open('a') as manifest:
                manifest.write('[dependencies]\nshared={path="../shared"}\n'
                    '[build-dependencies]\nshared={path="../shared",default-features=false}\n')
            (rust / 'utility/build.rs').write_text('fn main() { shared::build(); }\n')
            (rust / 'utility/src/main.rs').write_text('fn main() { shared::build(); println!("fresh executable"); }\n')
            report = root / 'report'; report.mkdir()
            metadata = subprocess.check_output(['cargo', 'metadata', '--no-deps', '--format-version', '1'], cwd=rust)
            (report / 'workspace.json').write_bytes(metadata)
            REPORT.write(report / 'settings.json', {'workspace_root': str(root)})
            for attempt in range(2):
                result = subprocess.run(['cargo', 'build', '--profile', 'release', '--workspace', '--features', 'default',
                    '--message-format=json'], cwd=rust, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                self.assertEqual(result.returncode, 0, result.stderr.decode())
                (report / 'build.json').write_bytes(result.stdout)
                with patch.object(REPORT, 'binding', return_value={'source_sha': 'fixture'}):
                    binaries = REPORT.artifacts(report)
                self.assertEqual(set(binaries), set(names))
                targets = REPORT.read(report / 'coverage.json')['targets']
                self.assertEqual({t['role'] for t in targets if t['package'] == 'shared'}, {'workspace', 'build-dependency'})
                if attempt == 1: self.assertTrue(all(t['fresh'] for t in REPORT.read(report / 'coverage.json')['targets']))
                for info in binaries.values():
                    self.assertEqual(subprocess.check_output([root / info['path']], text=True).strip(), 'fresh executable')


if __name__ == '__main__': unittest.main()
