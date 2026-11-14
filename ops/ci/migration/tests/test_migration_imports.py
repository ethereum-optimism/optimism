"""Filtered helper commands must not rely on another test's sys.path changes."""
from pathlib import Path
import subprocess
import unittest


class MigrationImportTests(unittest.TestCase):
    def test_each_migration_test_imports_in_a_fresh_interpreter(self):
        directory = Path(__file__).resolve().parent
        code = ('import importlib.util,sys;from pathlib import Path;'
                'p=Path(sys.argv[1]);sys.path.insert(0,str(p.parent));'
                's=importlib.util.spec_from_file_location("isolated_test",p);'
                'm=importlib.util.module_from_spec(s);s.loader.exec_module(m)')
        for path in sorted(directory.glob('test_*.py')):
            with self.subTest(module=path.name):
                result = subprocess.run(['python3', '-c', code, str(path)], cwd=directory,
                                        capture_output=True, text=True, timeout=20)
                self.assertEqual(result.returncode, 0, result.stderr)
