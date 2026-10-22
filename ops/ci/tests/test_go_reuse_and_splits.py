"""Prove dependency invalidation and exhaustive top-level test splitting."""
import copy
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch

RUNTIME = Path(__file__).resolve().parents[1] / 'runtime'


def load(name):
    spec = importlib.util.spec_from_file_location(name, RUNTIME / (name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


SHARDS = load('go-package-shards')
SOURCE = load('go-dependency-source')
PREFIX = 'github.com/ethereum-optimism/optimism'
HOT = PREFIX + '/op-deployer/pkg/deployer/integration_test/cli'


class SplitTests(unittest.TestCase):
    def test_all_tests_and_unsplit_packages_are_assigned_once_in_fixed_workers(self):
        names = ['TestA', 'TestB', 'TestC', 'TestNew', 'ExampleValue', 'FuzzInput']
        packages = [HOT, PREFIX + '/new', PREFIX + '/empty']
        manifest = SHARDS.create_manifest(packages, PREFIX, 12, {HOT: 600}, {HOT: names if names == sorted(names) else sorted(names)})
        seen = []
        for index in range(12):
            SHARDS.select_packages(manifest, PREFIX, index, 12)
            seen.extend(manifest['work_shards'][index])
        self.assertEqual(sorted(n for u in seen if u['package'] == HOT for n in u['tests']), sorted(names))
        self.assertEqual(sorted(u['package'] for u in seen if u['tests'] is None), sorted(packages[1:]))
        self.assertEqual(len([u for u in seen if u['package'] == HOT]), 4)
        for share in manifest['work_shards']:
            self.assertEqual(len(share), len({u['package'] for u in share}))
        for mutation in ('missing', 'duplicate', 'wrong_shard'):
            bad = copy.deepcopy(manifest)
            index = next(i for i, share in enumerate(bad['work_shards']) if any(u['tests'] for u in share))
            if mutation == 'missing': bad['work_shards'][index].clear()
            elif mutation == 'duplicate': bad['work_shards'][index].append(bad['work_shards'][index][0])
            else: bad['work_shards'][index], bad['work_shards'][(index+1)%12] = bad['work_shards'][(index+1)%12], bad['work_shards'][index]
            with self.assertRaises(ValueError): SHARDS.select_packages(bad, PREFIX, index, 12)

    @unittest.skipUnless(shutil.which('go'), 'requires pinned Go')
    def test_ast_inventory_never_executes_init_or_testmain_and_handles_examples(self):
        with tempfile.TemporaryDirectory() as tmp:
            source = Path(tmp) / 'names_test.go'
            source.write_text('''package fixture
import("os";"testing")
func init(){panic("must not execute init")}
func TestMain(m *testing.M){os.Exit(42)}
// func TestComment(t *testing.T) {}
func TestAlpha(t *testing.T){}
func TestBeta(t *testing.T){}
func Testlower(t *testing.T){}
func FuzzValue(f *testing.F){}
func BenchmarkValue(b *testing.B){}
func ExampleValue(){println("value") /* Output: value */}
func ExampleEmpty(){ /* Output: */ }
func ExampleNoOutput(){}
''')
            result = subprocess.run(['go', 'run', str(RUNTIME / 'go-test-names.go')], input=json.dumps([str(source)]), text=True, capture_output=True, timeout=120)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(json.loads(result.stdout), ['ExampleEmpty', 'ExampleValue', 'FuzzValue', 'TestAlpha', 'TestBeta'])


class SourceIdentityTests(unittest.TestCase):
    def test_go_only_edit_reuses_rust_helper_but_source_and_build_inputs_invalidate(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                return subprocess.check_output(['git', *args], cwd=root, text=True).strip()
            git('init', '-q')
            def commit(path, text):
                p=root/path;p.parent.mkdir(parents=True,exist_ok=True);p.write_text(text)
                git('add', '.');git('-c','user.name=Fixture','-c','user.email=fixture@example.invalid','commit','-qm','fixture')
                return git('rev-parse','HEAD')
            commit('mise.toml', '# toolchain')
            old=commit('rust/kona/src/lib.rs', 'old rust')
            with patch.object(SOURCE, 'ROOT', root):
                for kind in SOURCE.SCOPES:
                    self.assertEqual(len(SOURCE.fingerprint(kind)), 64)
                metadata={'commit_sha':old,'build_inputs_sha256':SOURCE.fingerprint('kona')}
                commit('op-service/errutil/errors.go', 'new go')
                SOURCE.verify('kona', metadata)
                self.assertEqual(SOURCE.ancestor('kona'), old)
                prestate = SOURCE.fingerprint('prestate')
                commit('cannon/mipsevm/state.go', 'changed cannon')
                self.assertNotEqual(prestate, SOURCE.fingerprint('prestate'))
                # Matching content alone cannot authorize an unrelated revision.
                with self.assertRaises((ValueError, subprocess.CalledProcessError)):
                    SOURCE.verify('kona', {**metadata,'commit_sha':'0'*40})
                commit('rust/kona/src/lib.rs', 'new rust')
                with self.assertRaises(ValueError): SOURCE.verify('kona', metadata)
                fresh={'commit_sha':git('rev-parse','HEAD'),'build_inputs_sha256':SOURCE.fingerprint('kona')}
                (root/'mise.toml').write_text('uncommitted tool change')
                with self.assertRaises(ValueError): SOURCE.verify('kona',fresh)
                git('checkout','--','mise.toml')
                commit('ops/ci/runtime/new-helper.py', 'changed build helper')
                with self.assertRaises(ValueError): SOURCE.verify('kona', fresh)


@unittest.skipUnless(all(shutil.which(t) for t in ('go', 'just', 'gotestsum')), 'requires pinned Go/Just/gotestsum')
class SplitExecutionTests(unittest.TestCase):
    def test_real_split_binaries_run_each_test_once_and_keep_failures_and_retries_scoped(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            helpers = root / 'ops/ci/runtime'; helpers.mkdir(parents=True)
            for name in ['go-compiled-tests.py', 'go-suite.py', 'go-package-shards.py', 'go-test-names.go']:
                shutil.copy2(RUNTIME / name, helpers / name)
            hot = root / HOT.removeprefix(PREFIX + '/')
            hot.mkdir(parents=True)
            files = {
                'go.mod': 'module ' + PREFIX + '\n\ngo 1.26.0\n',
                '.gitignore': '.ci/\n',
                'justfile': 'list-test-packages:\n  go list ./op-deployer/pkg/deployer/integration_test/cli\n',
                'op-deployer/cmd/op-deployer/main.go': 'package main\nfunc main() {}\n',
                str(hot.relative_to(root) / 'fixture_test.go'): '''package cli
import("os";"testing")
func TestMain(m *testing.M){
 if os.Getenv("OP_DEPLOYER_TEST_BINARY")=="" {os.Exit(44)}
 os.Exit(m.Run())
}
func TestA(t *testing.T){t.Run("nested",func(t *testing.T){})}
func TestB(t *testing.T){}
func TestC(t *testing.T){if os.Getenv("SPLIT_FAIL")=="1" {t.Fatal("original split failure")}}
func TestNew(t *testing.T){}
func Value() {}
func ExampleValue(){println("value") /* Output: value */}
'''.replace('println("value")', 'fmt.Println("value")').replace('"os";"testing"', '"os";"testing";"fmt"'),
            }
            for name, text in files.items():
                path=root/name;path.parent.mkdir(parents=True,exist_ok=True);path.write_text(text)
            for args in (['init','-q'],['add','.'],['-c','user.name=Fixture','-c','user.email=fixture@example.invalid','commit','-qm','split fixture']):
                subprocess.run(['git',*args],cwd=root,check=True)
            sha=subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip()
            env=dict(os.environ,CI_COMMIT_SHA=sha,CI_BRANCH='fixture',CI_SHARD_TOTAL='12',CI_SHARD_INDEX='0',PARALLEL='8',TEST_TIMEOUT='40m')
            def call(args, **extra):
                return subprocess.run(args,cwd=root,env=env|extra,capture_output=True,text=True,timeout=120)
            result=call(['python3',str(helpers/'go-suite.py'),'discover','--suite','go-tests','--total','12'])
            self.assertEqual(result.returncode,0,result.stderr)
            manifest=json.loads((root/'.ci/go-tests/manifest.json').read_text())
            # Static discovery must not run TestMain, which requires the sealed helper.
            self.assertEqual(manifest['split_tests'][HOT],['ExampleValue','TestA','TestB','TestC','TestNew'])
            events=[]
            for index,share in enumerate(manifest['work_shards']):
                if not share: continue
                result=call(['python3',str(helpers/'go-compiled-tests.py'),'build','--suite','go-tests'],CI_SHARD_INDEX=str(index))
                self.assertEqual(result.returncode,0,result.stderr)
                helper = root / '.ci/go-tests/build/op-deployer'
                original = helper.read_bytes()
                helper.write_bytes(b'corrupt prepared helper')
                tampered=call(['python3',str(helpers/'go-compiled-tests.py'),'verify','--suite','go-tests'],CI_SHARD_INDEX=str(index))
                self.assertNotEqual(tampered.returncode,0)
                helper.unlink()
                missing=call(['python3',str(helpers/'go-compiled-tests.py'),'verify','--suite','go-tests'],CI_SHARD_INDEX=str(index))
                self.assertNotEqual(missing.returncode,0)
                helper.write_bytes(original);helper.chmod(0o755)
                result=call(['python3',str(helpers/'go-compiled-tests.py'),'run','--suite','go-tests'],CI_SHARD_INDEX=str(index))
                self.assertEqual(result.returncode,0,result.stdout+result.stderr)
                events.extend(json.loads(line) for line in result.stdout.splitlines())
                names=share[0]['tests']
                if 'TestC' in names:
                    result=call(['python3',str(helpers/'go-compiled-tests.py'),'run','--suite','go-tests'],CI_SHARD_INDEX=str(index),SPLIT_FAIL='1')
                    self.assertEqual(result.returncode,1,result.stdout+result.stderr)
                    self.assertIn('original split failure',result.stdout)
                    result=call(['python3',str(helpers/'go-compiled-tests.py'),'run','--suite','go-tests','-test.run=^TestC$',HOT],CI_SHARD_INDEX=str(index))
                    self.assertEqual(result.returncode,0,result.stdout+result.stderr)
                    foreign=next(n for n in manifest['split_tests'][HOT] if n not in names)
                    result=call(['python3',str(helpers/'go-compiled-tests.py'),'run','--suite','go-tests','-test.run=^'+foreign+'$',HOT],CI_SHARD_INDEX=str(index))
                    self.assertNotEqual(result.returncode,0)
                    self.assertIn('escape the assigned test group',result.stderr)
            terminal=[e['Test'] for e in events if e.get('Action')=='pass' and e.get('Test') and '/' not in e['Test']]
            self.assertEqual(sorted(terminal), manifest['split_tests'][HOT])
