"""Compile and execute original Go binaries without Circle or migration tooling."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

RUNTIME=Path(__file__).resolve().parents[1]/'runtime'
SPEC=importlib.util.spec_from_file_location('go_report',RUNTIME/'go-report.py')
REPORT=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(REPORT)


@unittest.skipUnless(all(shutil.which(t) for t in ('go','just','gotestsum')), 'requires actual Go, Just and gotestsum')
class NativeIsolationTests(unittest.TestCase):
    def test_compilation_fresh_verdicts_fixture_paths_and_original_failure_reporting(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);helpers=root/'ops/ci/runtime';helpers.mkdir(parents=True)
            for name in ('go-compiled-tests.py','go-suite.py','go-package-shards.py'):
                shutil.copy2(RUNTIME/name,helpers/name)
            files={'go.mod':'module github.com/ethereum-optimism/optimism\n\ngo 1.26.0\n',
                'justfile':'list-test-packages:\n  go list -tags=ci ./fixture/...\n',
                '.gitignore':'.ci/\n',
                'fixture/empty/empty.go':'package empty\n',
                'fixture/tests/fixture.txt':'original fixture',
                'fixture/tests/fixture_test.go':'''package fixture
import("os";"testing")
func TestFresh(t *testing.T) {
 data,e:=os.ReadFile("fixture.txt");if e!=nil || string(data)!="original fixture" {t.Fatalf("working directory: %s %v",data,e)}
 f,e:=os.OpenFile("../../.ci/calls",os.O_CREATE|os.O_WRONLY|os.O_APPEND,0600);if e!=nil {t.Fatal(e)};defer f.Close();f.WriteString("fresh\\n")
 if os.Getenv("GO_ISOLATED_FAILURE")=="1" {t.Fatal("original intentional Go failure")}
}
'''}
            for name,data in files.items():
                path=root/name;path.parent.mkdir(parents=True,exist_ok=True);path.write_text(data)
            for args in (['init','-q'],['add','.'],['-c','user.name=CI fixture','-c','user.email=fixture@example.invalid','commit','-qm','native Go fixture']):
                subprocess.run(['git',*args],cwd=root,check=True)
            sha=subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip()
            env=dict(os.environ,CI_COMMIT_SHA=sha,CI_BRANCH='codex/rwx-ci-pilot',CI_SHARD_INDEX='0',CI_SHARD_TOTAL='1',PARALLEL='8',TEST_TIMEOUT='40m')
            def execute(args,**extra):
                result=subprocess.run(args,cwd=root,env=env|extra,capture_output=True,text=True,timeout=120)
                return result
            discovered=execute([sys.executable,str(helpers/'go-suite.py'),'discover','--suite','go-tests','--total','1'])
            self.assertEqual(discovered.returncode,0,discovered.stderr)
            built=execute([sys.executable,str(helpers/'go-compiled-tests.py'),'build','--suite','go-tests'])
            self.assertEqual(built.returncode,0,built.stderr)
            self.assertFalse((root/'.ci/calls').exists(),'Compilation must execute zero tests')
            metadata=json.loads((root/'.ci/go-tests/build/metadata.json').read_text())
            self.assertIsNone(metadata['packages']['github.com/ethereum-optimism/optimism/fixture/empty']['file'])
            for label,failing in [('fresh-one',False),('fresh-two',False),('intentional-failure',True)]:
                output=root/'.ci'/label;output.mkdir()
                result=execute(['gotestsum','--format=standard-quiet','--jsonfile='+str(output/'original.json'),
                    '--junitfile='+str(output/'junit.xml'),'--raw-command','--',sys.executable,
                    str(helpers/'go-compiled-tests.py'),'run','--suite','go-tests'],GO_ISOLATED_FAILURE='1' if failing else '0')
                self.assertEqual(result.returncode,1 if failing else 0,result.stdout+result.stderr)
                REPORT.project(output/'original.json',output/'native.json')
                self.assertIn('TestFresh',(output/'original.json').read_text())
                if failing:
                    self.assertIn('original intentional Go failure',(output/'original.json').read_text())
                    self.assertIn('<failure',(output/'junit.xml').read_text())
            self.assertEqual((root/'.ci/calls').read_text(),'fresh\nfresh\nfresh\n')
            self.assertFalse((root/'.circleci').exists())
            self.assertFalse((root/'ops/ci/migration').exists())
