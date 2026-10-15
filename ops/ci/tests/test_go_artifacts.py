#!/usr/bin/env python3
"""Reject stale/corrupt dependency inputs before runtime tests execute."""
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import tarfile
import io
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('artifacts', Path(__file__).resolve().parents[1] / 'runtime' / 'go-artifacts.py')
ART = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(ART)

class GoArtifactsTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / 'mise.toml').write_text('pinned')
        (self.root / 'fixture').mkdir()
        (self.root / 'fixture/binary').write_text('compiled')
        self.addCleanup(patch.stopall)
        patch.object(ART, 'ROOT', self.root).start()
        patch.dict(os.environ, CI_COMMIT_SHA='a'*40).start()
        patch.object(ART.subprocess, 'check_output', side_effect=lambda args, **kw: 'a'*40 if args[0]=='git' else 'go1.26').start()
        ART.pack('go', ['fixture'])
        self.artifact = self.root / '.ci/go-tests/dependencies/go'

    def test_restores_verified_fixture_and_retains_provenance(self):
        (self.root / 'fixture/binary').unlink()
        ART.restore('go', self.artifact)
        self.assertEqual((self.root / 'fixture/binary').read_text(), 'compiled')
        self.assertTrue((self.root / 'tmp/testlogs/dependencies/go.json').exists())

    def test_stale_sha_settings_and_toolchain_are_rejected(self):
        metadata = self.artifact / 'metadata.json'
        original = json.loads(metadata.read_text())
        for key, value in [('commit_sha','b'*40), ('settings', {}), ('mise_sha256', 'wrong'), ('files', {})]:
            with self.subTest(key=key):
                metadata.write_text(json.dumps({**original, key:value}))
                with self.assertRaises(ValueError): ART.restore('go', self.artifact)
        metadata.write_text(json.dumps(original))
        with patch.dict(os.environ, CI_COMMIT_SHA='c'*40):
            with self.assertRaises(ValueError): ART.restore('go', self.artifact)

    def test_missing_and_corrupt_archives_and_files_are_rejected(self):
        archive = self.artifact / 'files.tar.gz'
        source = archive.read_bytes()
        archive.write_bytes(b'corrupt')
        with self.assertRaises(ValueError): ART.restore('go', self.artifact)
        archive.unlink()
        with self.assertRaises(OSError): ART.restore('go', self.artifact)
        archive.write_bytes(source)
        metadata = self.artifact / 'metadata.json'
        data = json.loads(metadata.read_text())
        data['files']['fixture/binary']='wrong'
        metadata.write_text(json.dumps(data))
        with self.assertRaises(ValueError): ART.restore('go', self.artifact)

    def test_missing_output_and_revision_fail_at_producer(self):
        with self.assertRaises(ValueError): ART.pack('go', ['absent'])
        with patch.dict(os.environ, CI_COMMIT_SHA='b'*40):
            with self.assertRaises(ValueError): ART.pack('go', ['fixture'])

    def test_readonly_materialized_cache_is_verified_without_overwriting(self):
        binary = self.root / 'fixture/binary'; directory = binary.parent
        binary.chmod(0o444); directory.chmod(0o555)
        self.addCleanup(directory.chmod, 0o755)
        inode, changed = binary.stat().st_ino, binary.stat().st_mtime_ns
        ART.restore('go', self.artifact); ART.restore('go', self.artifact)
        self.assertEqual(binary.stat().st_ino, inode)
        self.assertEqual(binary.stat().st_mtime_ns, changed)
        self.assertEqual(binary.stat().st_mode & 0o777, 0o444)
        self.assertEqual(directory.stat().st_mode & 0o777, 0o555)
        self.assertEqual(binary.read_text(), 'compiled')

    def test_matching_destination_cannot_hide_corrupt_archived_bytes(self):
        archive = self.artifact / 'files.tar.gz'; data = b'corrupt'
        with tarfile.open(archive, 'w:gz') as stream:
            member = tarfile.TarInfo('fixture/binary'); member.size = len(data)
            stream.addfile(member, io.BytesIO(data))
        path = self.artifact / 'metadata.json'; metadata = json.loads(path.read_text())
        metadata['archive_sha256'] = ART.digest(archive); path.write_text(json.dumps(metadata))
        with self.assertRaisesRegex(ValueError, 'archived dependency'): ART.restore('go', self.artifact)

if __name__ == '__main__': unittest.main()
