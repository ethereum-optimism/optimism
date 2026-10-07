"""Offline recovery is explicit; permanent verification remains read-only."""
import importlib.util
from pathlib import Path
import tempfile
import unittest
_SPEC=importlib.util.spec_from_file_location('evidence',Path(__file__).resolve().parents[1]/'report-evidence.py')
E=importlib.util.module_from_spec(_SPEC);_SPEC.loader.exec_module(E)


class EvidenceTests(unittest.TestCase):
    def test_recovery_requires_original_empty_hash_and_explicit_authorization(self):
        for authorized,empty in [(False,True),(True,True),(True,False)]:
            with self.subTest(authorized=authorized,empty=empty),tempfile.TemporaryDirectory() as tmp:
                root=Path(tmp);sha=E.REPORT.EMPTY_SHA256 if empty else 'a'*64;recovered=[]
                if authorized and empty:
                    E.verify_files(root,{'nested/empty.log':sha},missing_empty=recovered,label='circle/fixture')
                    self.assertEqual(recovered,[{'report':'circle/fixture','path':'nested/empty.log','sha256':sha}])
                    self.assertEqual((root/'nested/empty.log').read_bytes(),b'')
                else:
                    with self.assertRaises(OSError):E.verify_files(root,{'nested/empty.log':sha},missing_empty=recovered if authorized else None)
                    self.assertEqual(list(root.iterdir()),[])

    def test_recovery_rejects_unsafe_paths_and_linked_destinations(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)/'reports';root.mkdir();outside=Path(tmp)/'outside';outside.mkdir();(root/'linked').symlink_to(outside,target_is_directory=True)
            for name in ('../outside/empty.log','/foreign/empty.log','linked/empty.log'):
                with self.subTest(name=name),self.assertRaises(ValueError):E.verify_files(root,{name:E.REPORT.EMPTY_SHA256},missing_empty=[])
            self.assertEqual(list(outside.iterdir()),[])

    def test_suite_recovery_allowlist_does_not_authorize_a_missing_verdict(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            with self.assertRaises(OSError):
                E.verify_files(root,{'verdict.json':E.REPORT.EMPTY_SHA256},missing_empty=[],recoverable={'diagnostic.log'})
            self.assertEqual(list(root.iterdir()),[])
