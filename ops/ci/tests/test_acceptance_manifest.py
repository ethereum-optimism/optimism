#!/usr/bin/env python3
"""Acceptance discovery, exhaustive assignment and source/settings checks."""
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('acceptance', Path(__file__).resolve().parents[1] / 'runtime' / 'acceptance-manifest.py')
ACCEPTANCE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(ACCEPTANCE)


class AcceptanceManifestTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.packages = [ACCEPTANCE.PREFIX + '/base', ACCEPTANCE.PREFIX + '/interop', ACCEPTANCE.PREFIX + '/empty']
        self.package_file = self.root / 'packages.json'
        self.package_file.write_text('\n'.join(json.dumps({'ImportPath': p}) for p in self.packages))
        events = []
        for package, names in zip(self.packages, [['TestShared', 'TestBase'], ['TestShared', 'TestInterop'], []]):
            events += [{'Action': 'output', 'Package': package, 'Output': name + '\n'} for name in names]
            events.append({'Action': 'pass', 'Package': package})
        self.listing = self.root / 'listing.json'
        self.listing.write_text('\n'.join(json.dumps(e) for e in events))
        self.addCleanup(patch.stopall)
        patch.dict(os.environ, {'CI_COMMIT_SHA': 'a'*40}, clear=True).start()
        patch.object(ACCEPTANCE.subprocess, 'check_output', return_value='go version go1.26.6 linux/amd64\n').start()

    def test_complete_identity_assignment_with_duplicate_names_and_empty_packages(self):
        manifest = ACCEPTANCE.create(self.package_file, self.listing, 8)
        ACCEPTANCE.validate(manifest, 8)
        self.assertEqual(manifest['packages'], sorted(self.packages))
        self.assertEqual(len(manifest['tests']), 4)
        self.assertEqual(sum('TestShared' in s for s in manifest['shards']), 1)
        self.assertEqual(len([s for s in manifest['shards'] if not s]), 5)

    def test_failed_missing_unknown_and_duplicate_discovery_are_rejected(self):
        original = self.listing.read_text()
        for extra in [{'Action': 'fail', 'Package': self.packages[0]},
                      {'Action': 'pass', 'Package': 'outside'},
                      {'Action': 'output', 'Package': self.packages[0], 'Output': 'TestBase\n'}]:
            self.listing.write_text(original + '\n' + json.dumps(extra))
            with self.assertRaises(ValueError): ACCEPTANCE.create(self.package_file, self.listing, 8)
        self.listing.write_text('\n'.join(original.splitlines()[:-1]))
        with self.assertRaises(ValueError): ACCEPTANCE.create(self.package_file, self.listing, 8)

    def test_package_errors_and_empty_listing_are_rejected(self):
        self.package_file.write_text(json.dumps({'ImportPath': self.packages[0], 'DepsErrors': [{'Err': 'missing embed'}]}))
        with self.assertRaises(ValueError): ACCEPTANCE.create(self.package_file, self.listing, 8)

    def test_assignment_corruption_is_rejected(self):
        original = ACCEPTANCE.create(self.package_file, self.listing, 8)
        for change in ('missing', 'duplicate', 'unknown'):
            manifest = json.loads(json.dumps(original))
            if change == 'missing': manifest['shards'][0] = []
            if change == 'duplicate': manifest['shards'][1] += manifest['shards'][0]
            if change == 'unknown': manifest['shards'][0].append('TestUnknown')
            with self.assertRaises(ValueError): ACCEPTANCE.validate(manifest, 8)

    def test_revision_settings_and_discovery_corruption_fail_selection(self):
        path = self.root / 'manifest.json'
        manifest = ACCEPTANCE.create(self.package_file, self.listing, 8)
        path.write_text(json.dumps(manifest))
        self.assertTrue(ACCEPTANCE.select(path, 0, 8))
        for key, value in [('commit_sha', 'b'*40), ('settings', {}), ('go_version', 'stale')]:
            path.write_text(json.dumps({**manifest, key: value}))
            with self.assertRaises(ValueError): ACCEPTANCE.select(path, 0, 8)
        path.write_text(json.dumps(manifest))
        self.listing.write_text(self.listing.read_text() + '\n')
        with self.assertRaises(ValueError): ACCEPTANCE.select(path, 0, 8)

    def test_empty_shard_and_new_test_remain_valid(self):
        manifest = ACCEPTANCE.create(self.package_file, self.listing, 8)
        path = self.root / 'manifest.json'
        path.write_text(json.dumps(manifest))
        self.assertEqual(ACCEPTANCE.select(path, 7, 8), [])
        events = self.listing.read_text() + '\n' + json.dumps({'Action': 'output', 'Package': self.packages[2], 'Output': 'TestNew\n'})
        self.listing.write_text(events)
        manifest = ACCEPTANCE.create(self.package_file, self.listing, 8)
        ACCEPTANCE.validate(manifest, 8)
        self.assertIn('TestNew', [name for shard in manifest['shards'] for name in shard])


if __name__ == '__main__': unittest.main()
