"""Enforce removable migration tooling and native-only runtime inputs."""
import ast
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT=Path(__file__).resolve().parents[3]
CI=ROOT/'ops/ci'


class LayoutTests(unittest.TestCase):
    def test_all_helper_files_have_an_owner(self):
        self.assertFalse([p.name for p in CI.iterdir() if p.is_file()])
        owners={p.name for p in CI.iterdir() if p.is_dir() and p.name!='__pycache__'}
        self.assertTrue({'runtime','tests'} <= owners <= {'runtime','tests','migration'})

    def test_runtime_and_permanent_tests_do_not_import_migration(self):
        for directory in (CI/'runtime',CI/'tests'):
            for path in directory.glob('*.py'):
                tree=ast.parse(path.read_text())
                for node in ast.walk(tree):
                    if isinstance(node,(ast.Import,ast.ImportFrom)):
                        names=[n.name for n in node.names] if isinstance(node,ast.Import) else [node.module or '']
                        self.assertFalse(any('migration' in name or 'compare_' in name for name in names),str(path))
                    if isinstance(node,ast.Call):
                        function=node.func
                        is_path=isinstance(function,ast.Attribute) and function.attr in ('with_name','spec_from_file_location')
                        is_helper=isinstance(function,ast.Name) and function.id in ('helper','module')
                        if is_path or is_helper:
                            for value in node.args:
                                if isinstance(value,ast.Constant) and isinstance(value.value,str):
                                    self.assertFalse(value.value.startswith(('compare-','migration/')),str(path))

    def test_filtered_sp1_import_succeeds_after_migration_is_removed(self):
        import json
        config=json.loads(subprocess.check_output(['yq','-o=json','.',str(ROOT/'.rwx/sp1-guest.yml')],text=True))
        task=next(t for t in config['tasks'] if t['key']=='toolchain')
        with tempfile.TemporaryDirectory() as tmp:
            target=Path(tmp)
            for name in task['filter']:
                self.assertNotIn('migration', Path(name).parts, name)
                destination=target/name;destination.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(ROOT/name,destination)
            code='import importlib.util,sys;s=importlib.util.spec_from_file_location("sp1",sys.argv[1]);m=importlib.util.module_from_spec(s);s.loader.exec_module(m)'
            result=subprocess.run(['python3','-c',code,str(target/'ops/ci/runtime/sp1-guest.py')],capture_output=True,text=True)
            self.assertEqual(result.returncode,0,result.stderr)

    def test_every_runtime_module_imports_without_circle_or_migration(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp); target=root/'ops/ci/runtime'
            shutil.copytree(CI/'runtime', target, ignore=shutil.ignore_patterns('__pycache__'))
            self.assertFalse((root/'.circleci').exists())
            self.assertFalse((root/'ops/ci/migration').exists())
            code='import importlib.util,sys;s=importlib.util.spec_from_file_location("runtime",sys.argv[1]);m=importlib.util.module_from_spec(s);s.loader.exec_module(m)'
            for path in sorted(target.glob('*.py')):
                with self.subTest(module=path.name):
                    result=subprocess.run(['python3','-c',code,str(path)],cwd=root,capture_output=True,text=True,timeout=10)
                    self.assertEqual(result.returncode,0,result.stderr)
