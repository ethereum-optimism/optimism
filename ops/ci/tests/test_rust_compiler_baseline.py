import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('baseline', Path(__file__).resolve().parents[1] / 'runtime/rust-compiler-baseline.py')
BASELINE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(BASELINE)


class BaselineTests(unittest.TestCase):
    def test_optional_failed_ancestor_retains_failure_and_discards_partial_targets(self):
        for status in (0, 101):
            with self.subTest(status=status), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                target = root / 'rust/target'; target.mkdir(parents=True)
                (target / '.rwx-source-pending.json').write_text('partial')
                report = root / '.ci/rust-workspace/tests-build'; report.mkdir(parents=True)
                (report / 'final.json').write_text(json.dumps({'exit_code': status}))
                (report / 'archive.log').write_text('original compiler output')
                (report / 'tests.tar.zst').write_bytes(b'large binary archive')
                with patch.object(BASELINE.subprocess, 'run', return_value=subprocess.CompletedProcess([], status)) as build, \
                        patch.object(BASELINE.subprocess, 'check_output', return_value='a'*40+'\n'):
                    BASELINE.prepare('tests-build', root)
                self.assertEqual(build.call_args.kwargs['env']['CI_RUST_BASELINE'], 'true')
                self.assertEqual(target.exists(), status == 0)
                evidence = root / '.ci/rust-workspace/compiler-baseline/tests-build'
                self.assertEqual(json.loads((evidence / 'final.json').read_text())['exit_code'], status)
                self.assertEqual((evidence / 'archive.log').read_text(), 'original compiler output')
                self.assertFalse((evidence / 'tests.tar.zst').exists())
                binding = json.loads((evidence / 'compiler-baseline.json').read_text())
                self.assertEqual(binding['usable'], status == 0)
                self.assertEqual(binding['build_exit_code'], status)
                self.assertEqual(binding['source_sha'], 'a'*40)

    def test_failed_cleanup_cannot_claim_a_clean_fallback(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / 'rust/target').mkdir(parents=True)
            with patch.object(BASELINE.subprocess, 'run', return_value=subprocess.CompletedProcess([], 101)), \
                    patch.object(BASELINE.shutil, 'rmtree', side_effect=PermissionError('retained target')):
                with self.assertRaises(PermissionError):
                    BASELINE.prepare('tests-build', root)
            self.assertFalse((root / '.ci/rust-workspace/compiler-baseline/tests-build/compiler-baseline.json').exists())
