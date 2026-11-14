import errno
import importlib.util
import os
from pathlib import Path
import stat
import tempfile
import unittest
from unittest.mock import patch


SPEC = importlib.util.spec_from_file_location(
    'linker', Path(__file__).parents[1] / 'runtime/rwx-materialize-linker.py')
LINKER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(LINKER)


class MaterializeLinkerTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.path = Path(self.directory.name) / 'crt.o'
        self.path.write_bytes(b'original linker input')
        self.path.chmod(0o640)
        os.utime(self.path, ns=(1_700_000_000_000_000_000, 1_700_000_001_123_456_789))

    def test_detaches_hardlink_preserving_bytes_owner_mode_and_mtime(self):
        alias = self.path.with_name('alias')
        os.link(self.path, alias)
        before = self.path.stat()
        record = {}
        LINKER.materialize(self.path, record)
        after = self.path.stat()
        self.assertEqual(self.path.read_bytes(), alias.read_bytes())
        self.assertEqual(alias.stat().st_ino, before.st_ino)
        self.assertNotEqual(after.st_ino, before.st_ino)
        self.assertEqual(after.st_nlink, 1)
        self.assertEqual((after.st_uid, after.st_gid, stat.S_IMODE(after.st_mode), after.st_atime_ns, after.st_mtime_ns),
                         (before.st_uid, before.st_gid, stat.S_IMODE(before.st_mode), before.st_atime_ns, before.st_mtime_ns))
        self.assertTrue(record['detached'])
        self.assertEqual(record['read_attempts'], 1)
        self.assertEqual(list(self.path.parent.glob('.rwx-linker-*')), [])

    def test_retries_only_timed_out_reads_then_copies_original_bytes(self):
        original = Path.read_bytes
        calls = []

        def read(path):
            if path == self.path:
                calls.append(path)
                if len(calls) == 1:
                    raise OSError(errno.ETIMEDOUT, 'Connection timed out')
            return original(path)

        record = {}
        with patch.object(Path, 'read_bytes', read), patch.object(LINKER.time, 'sleep') as sleep:
            LINKER.materialize(self.path, record)
        self.assertEqual(record['read_attempts'], 2)
        self.assertEqual(record['read_timeouts'], 1)
        self.assertEqual(self.path.read_bytes(), b'original linker input')
        sleep.assert_called_once_with(1)

    def test_persistent_timeout_is_bounded_and_does_not_replace_input(self):
        before = self.path.stat()
        record = {}
        with patch.object(Path, 'read_bytes', side_effect=OSError(errno.ETIMEDOUT, 'timeout')), \
                patch.object(LINKER.time, 'sleep'):
            with self.assertRaises(OSError):
                LINKER.materialize(self.path, record)
        self.assertEqual(record['read_attempts'], 3)
        self.assertEqual(record['read_timeouts'], 3)
        self.assertEqual(self.path.stat().st_ino, before.st_ino)

    def test_other_read_errors_fail_immediately(self):
        for code in (errno.EACCES, errno.ENOENT, errno.EIO):
            with self.subTest(code=code):
                record = {}
                with patch.object(Path, 'read_bytes', side_effect=OSError(code, 'failure')), \
                        patch.object(LINKER.time, 'sleep') as sleep:
                    with self.assertRaises(OSError):
                        LINKER.materialize(self.path, record)
                self.assertEqual(record['read_attempts'], 1)
                sleep.assert_not_called()

    def test_corrupt_local_copy_never_replaces_input(self):
        before = self.path.stat()
        original = Path.read_bytes

        def read(path):
            return original(path) if path == self.path else b'corrupt copy'

        with patch.object(Path, 'read_bytes', read):
            with self.assertRaisesRegex(ValueError, 'digest mismatch'):
                LINKER.materialize(self.path, {})
        self.assertEqual(self.path.stat().st_ino, before.st_ino)
        self.assertEqual(self.path.read_bytes(), b'original linker input')
        self.assertEqual(list(self.path.parent.glob('.rwx-linker-*')), [])

    def test_failed_replace_keeps_original_and_cleans_temporary(self):
        before = self.path.stat()
        with patch.object(LINKER.os, 'replace', side_effect=OSError(errno.ENOSPC, 'disk full')):
            with self.assertRaises(OSError):
                LINKER.materialize(self.path, {})
        self.assertEqual(self.path.stat().st_ino, before.st_ino)
        self.assertEqual(list(self.path.parent.glob('.rwx-linker-*')), [])

    def test_rejects_missing_driver_input_and_outside_system_root(self):
        for value in ('crti.o', str(self.path)):
            with self.subTest(value=value), patch.object(LINKER.subprocess, 'check_output', return_value=value):
                with self.assertRaises(ValueError):
                    LINKER.linker_inputs()

    def test_reports_partial_failure_and_returns_failure_status(self):
        with patch.object(LINKER, 'linker_inputs', return_value=[self.path]), \
                patch.object(LINKER, 'materialize', side_effect=OSError(errno.EIO, 'failure')), \
                patch('builtins.print') as output:
            self.assertEqual(LINKER.main(), 1)
        import json
        report = json.loads(output.call_args.args[0])
        self.assertFalse(report['success'])
        self.assertIn('failure', report['error'])


if __name__ == '__main__':
    unittest.main()
