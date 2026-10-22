"""Committed source integrity does not depend on publication rehearsal tooling."""
import importlib.util
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
SPEC=importlib.util.spec_from_file_location('source',Path(__file__).resolve().parents[1]/'runtime/ci-source.py')
SOURCE=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(SOURCE)

class SourceTests(unittest.TestCase):
    def test_real_git_bytes_modes_and_symlinks_ignore_restored_mtimes(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                return subprocess.check_output(['git', '-C', str(root), '-c', 'user.name=fixture',
                    '-c', 'user.email=fixture@example.invalid', '-c', 'commit.gpgsign=false', *args], stderr=subprocess.DEVNULL).decode().strip()
            git('init', '-q')
            script = root / 'input.sh'; script.write_text('original'); script.chmod(0o755)
            (root / 'link').symlink_to('input.sh')
            git('add', '.'); git('commit', '-qm', 'original fixture'); sha = git('rev-parse', 'HEAD')
            inputs = SOURCE.source_inputs(root, sha)
            self.assertEqual(inputs['input.sh']['mode'], '100755')
            self.assertEqual(inputs['link']['sha256'], SOURCE.hashlib.sha256(b'input.sh').hexdigest())
            before = script.stat(); script.write_text('modified'); os.utime(script, ns=(before.st_atime_ns, before.st_mtime_ns))
            with self.assertRaisesRegex(ValueError, 'Changed committed'): SOURCE.source_inputs(root, sha)
            script.write_text('original'); script.chmod(0o644)
            with self.assertRaisesRegex(ValueError, 'executable mode'): SOURCE.source_inputs(root, sha)
            script.chmod(0o755); (root / 'link').unlink(); (root / 'link').symlink_to('foreign.sh')
            with self.assertRaisesRegex(ValueError, 'Changed committed'): SOURCE.source_inputs(root, sha)
