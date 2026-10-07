"""Exercise shared mechanics at the report and subprocess trust boundaries."""
import ast
import importlib.util
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

SCRIPTS = Path(__file__).resolve().parents[1] / 'runtime'
SPEC = importlib.util.spec_from_file_location('ci_report', Path(__file__).resolve().parents[1] / 'runtime' / 'ci-report.py')
REPORT = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(REPORT)


class ReportTests(unittest.TestCase):
    def test_required_corrupt_and_unsafe_originals_are_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            path = root / 'original.json'; path.write_text('original evidence')
            hashes = REPORT.file_hashes(root)
            self.assertEqual(REPORT.verify_files(root, hashes, required={'original.json'}), hashes)
            path.write_text('corrupted')
            with self.assertRaisesRegex(ValueError, 'corrupt'):
                REPORT.verify_files(root, hashes)
            for manifest in ({}, {'../outside': 'a'*64}, {'/outside': 'a'*64}, {'original.json': 'invalid'}):
                with self.subTest(manifest=manifest), self.assertRaises(ValueError):
                    REPORT.verify_files(root, manifest, required={'original.json'})

    def test_missing_empty_original_is_rejected_without_modifying_the_directory(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            with self.assertRaises(OSError):
                REPORT.verify_files(root, {"nested/empty.log": REPORT.EMPTY_SHA256})
            self.assertEqual(list(root.iterdir()), [])

    def test_linked_artifact_bytes_are_not_accepted(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp); outside=root/'outside'; outside.mkdir()
            (outside/'same.log').write_text('original')
            report=root/'report'; report.mkdir()
            (report/'linked').symlink_to(outside, target_is_directory=True)
            with self.assertRaisesRegex(ValueError, 'Linked'):
                REPORT.verify_files(report, {'linked/same.log': REPORT.digest(outside/'same.log')})

    def test_split_drains_large_stderr_without_contaminating_original_json(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            # Exceeds a pipe buffer. A serial reader would deadlock before JSON.
            code = 'import sys;sys.stderr.write("x"*200000);print("{\\"complete\\":true}");sys.exit(17)'
            harness = ('import importlib.util,sys;from pathlib import Path;'
                       's=importlib.util.spec_from_file_location("r",sys.argv[1]);'
                       'r=importlib.util.module_from_spec(s);s.loader.exec_module(r);'
                       'sys.exit(r.stage(Path(sys.argv[2]),"failure",[sys.executable,"-c",sys.argv[3]],'
                       'cwd=sys.argv[2],layout="split",stdout_json=True))')
            result = subprocess.run([sys.executable, '-c', harness, str(Path(__file__).resolve().parents[1] / 'runtime' / 'ci-report.py'), tmp, code],
                                    capture_output=True, timeout=10)
            self.assertEqual(result.returncode, 17, result.stderr.decode())
            self.assertEqual(REPORT.read(root / 'failure.json'), {'complete': True})
            self.assertEqual((root / 'failure.stderr.log').stat().st_size, 200000)
            stage = REPORT.read(root / 'failure.stage.json')
            self.assertEqual(stage['exit_code'], 17)
            self.assertEqual(stage['stderr_sha256'], REPORT.digest(root / 'failure.stderr.log'))

    def test_combined_separate_stdout_preserves_devnull_and_stage_schema(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            status = REPORT.stage(root, 'original', [sys.executable, '-c',
                'import sys;assert sys.stdin.read()=="";print("bytes");print("diagnostic",file=sys.stderr)'],
                cwd=tmp, stdout_file='original.stdout', stdin=subprocess.DEVNULL)
            self.assertEqual(status, 0)
            self.assertEqual((root / 'original.stdout').read_text(), 'bytes\n')
            self.assertEqual((root / 'original.log').read_text(), 'diagnostic\n')
            row = REPORT.read(root / 'original.stage.json')
            self.assertEqual(row['stdin'], 'devnull')
            self.assertEqual(row['stdout_file'], 'original.stdout')
            self.assertEqual(row['stdout_sha256'], REPORT.digest(root / 'original.stdout'))
            self.assertEqual(row['log_sha256'], REPORT.digest(root / 'original.log'))

    def test_runtime_imports_work_without_offline_comparers(self):
        families = {
            'contract-upgrades.py': ('ci-report.py', 'git-submodule-report.py'),
            'cannon-go.py': ('ci-report.py', 'ci-test-results.py', 'rust-workspace-report.py',
                             'go-artifacts.py', 'go-report.py'),
        }
        for entry, dependencies in families.items():
            with self.subTest(entry=entry), tempfile.TemporaryDirectory() as tmp:
                directory = Path(tmp) / 'ops/ci/runtime'
                directory.mkdir(parents=True)
                for name in (entry, *dependencies):
                    shutil.copy2(SCRIPTS / name, directory / name)
                code = ('import importlib.util,sys;'
                        's=importlib.util.spec_from_file_location("runner",sys.argv[1]);'
                        'm=importlib.util.module_from_spec(s);s.loader.exec_module(m)')
                result = subprocess.run([sys.executable, '-c', code, str(directory / entry)],
                                        capture_output=True, timeout=10)
                self.assertEqual(result.returncode, 0, result.stderr.decode())

    def test_runtime_import_graph_has_no_comparer_dependency(self):
        for path in SCRIPTS.glob('*.py'):
            if path.name.startswith(('compare-', 'test_', 'ci_test_')):
                continue
            for node in ast.walk(ast.parse(path.read_text())):
                # Dynamic imports use helper(name) or Path.with_name(filename).
                if not isinstance(node, ast.Call):
                    continue
                fn = node.func
                if not ((isinstance(fn, ast.Name) and fn.id == 'helper') or
                        (isinstance(fn, ast.Attribute) and fn.attr == 'with_name')):
                    continue
                for arg in node.args:
                    if isinstance(arg, ast.Constant) and isinstance(arg.value, str):
                        self.assertFalse(arg.value.startswith('compare-'), path.name + ': ' + arg.value)
