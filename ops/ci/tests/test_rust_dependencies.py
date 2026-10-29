import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('dependencies', Path(__file__).resolve().parents[1] / 'runtime/rust-dependencies.py')
DEPS = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(DEPS)


class DependencyTests(unittest.TestCase):
    def test_prefetch_preserves_attempts_and_failed_ancestor_remains_optional(self):
        for exits in ([101, 0], [101] * 5):
            with self.subTest(exits=exits), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp); (root / 'rust').mkdir()
                (root / 'rust/Cargo.lock').write_text('locked parent dependencies')
                values = iter(exits)
                def run(args, **kwargs):
                    status = next(values)
                    self.assertEqual(args, ['cargo', 'fetch', '--locked'])
                    self.assertEqual(kwargs['env']['CARGO_HOME'], str(root / '.ci/rust-cache/cargo'))
                    kwargs['stdout'].write(f'original fetch exit {status}\n'.encode())
                    return subprocess.CompletedProcess(args, status)
                with patch.object(DEPS.subprocess, 'run', side_effect=run), \
                        patch.object(DEPS.subprocess, 'check_output', return_value='a'*40), \
                        patch.object(DEPS.time, 'sleep'):
                    DEPS.fetch(root)
                report = root / '.ci/rust-dependencies'
                value = json.loads((report / 'result.json').read_text())
                self.assertEqual([a['exit_code'] for a in value['attempts']], exits)
                self.assertEqual(value['usable'], exits[-1] == 0)
                self.assertEqual(value['original_log_sha256'], hashlib.sha256((report / 'fetch.log').read_bytes()).hexdigest())
                self.assertIn(b'original fetch exit 101', (report / 'fetch.log').read_bytes())
