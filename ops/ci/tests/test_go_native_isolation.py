"""Discover, compile and run a Go fixture module with the actual Go, Just and gotestsum."""
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
MODULE='github.com/ethereum-optimism/optimism'


@unittest.skipUnless(all(shutil.which(t) for t in ('go','just','gotestsum')), 'requires actual Go, Just and gotestsum')
class NativeIsolationTests(unittest.TestCase):
    def test_compilation_fresh_verdicts_fixture_paths_and_original_failure_reporting(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);helpers=root/'ops/ci/runtime';helpers.mkdir(parents=True)
            for name in ('go-compiled-tests.py','go-suite.py','go-package-shards.py'):
                shutil.copy2(RUNTIME/name,helpers/name)
            files={'go.mod':'module '+MODULE+'\n\ngo 1.26.0\n',
                'justfile':'list-test-packages:\n  go list -tags=ci ./fixture/...\n',
                'timings.json':'{"packages": {}}\n',
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
            env=dict(os.environ,CI_COMMIT_SHA='a'*40,CI_SHARD_INDEX='0',PARALLEL='8',TEST_TIMEOUT='40m')
            def execute(args,**extra):
                result=subprocess.run(args,cwd=root,env=env|extra,capture_output=True,text=True,timeout=120)
                return result
            discovered=execute([sys.executable,str(helpers/'go-suite.py'),'--total','1','--timings','timings.json'])
            self.assertEqual(discovered.returncode,0,discovered.stderr)
            manifest=root/'.ci/go-tests/manifest.json'
            self.assertEqual(json.loads(manifest.read_text()),{'commit_sha':'a'*40,'no_tests':[MODULE+'/fixture/empty'],
                'shards':[[MODULE+'/fixture/empty',MODULE+'/fixture/tests']]})
            built=execute([sys.executable,str(helpers/'go-compiled-tests.py'),'build'])
            self.assertEqual(built.returncode,0,built.stderr)
            self.assertFalse((root/'.ci/calls').exists(),'Compilation must execute zero tests')
            # Package 0 has no test files, so only package 1 has a binary.
            self.assertEqual(sorted(p.name for p in (root/'.ci/go-tests/build').glob('*.test')),['1.test'])
            manifest.write_text('the manifest of an older commit')
            restored=execute([sys.executable,str(helpers/'go-compiled-tests.py'),'restore'])
            self.assertEqual(restored.returncode,0,restored.stderr)
            for label,failing in [('fresh-one',False),('fresh-two',False),('intentional-failure',True)]:
                output=root/'.ci'/label;output.mkdir()
                result=execute(['gotestsum','--format=standard-quiet','--jsonfile='+str(output/'original.json'),
                    '--junitfile='+str(output/'junit.xml'),'--raw-command','--',sys.executable,
                    str(helpers/'go-compiled-tests.py'),'run'],GO_ISOLATED_FAILURE='1' if failing else '0')
                self.assertEqual(result.returncode,1 if failing else 0,result.stdout+result.stderr)
                REPORT.project(output/'original.json',output/'native.json')
                self.assertIn('TestFresh',(output/'original.json').read_text())
                if failing:
                    self.assertIn('original intentional Go failure',(output/'original.json').read_text())
                    self.assertIn('<failure',(output/'junit.xml').read_text())
            self.assertEqual((root/'.ci/calls').read_text(),'fresh\nfresh\nfresh\n')
