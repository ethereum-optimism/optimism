"""Exercise real fresh Go/Forge generation, original drift guards and parity."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest

SCRIPTS = Path(__file__).resolve().parents[1] / 'runtime'


def module(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).resolve().parents[1] / ('tests' if name.startswith('test_') else 'runtime') / (name + '.py'))
    result = importlib.util.module_from_spec(spec); spec.loader.exec_module(result)
    return result


N = module('nut-prefork'); M = module('test_main_checks')


class DiscoveryTests(unittest.TestCase):
    def test_complete_state_selection_includes_new_forks_and_preserves_jovian_exclusion(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); directory = root / 'op-core/nuts/state'; directory.mkdir(parents=True)
            for name in ('jovian','karst','lagoon','new-fork'): (directory / (name + '_state.json')).write_text('{"state":1}\n')
            selected = N.select_states(root)
            self.assertEqual(selected['forks'], ['karst','lagoon','new-fork'])
            self.assertEqual(len(selected['states']),4)
            self.assertEqual(selected['excluded'][0]['fork'],'jovian')
            for data in ('{}','[]','invalid'):
                (directory / 'karst_state.json').write_text(data)
                with self.subTest(data=data),self.assertRaises(ValueError): N.select_states(root)

    def test_package_discovery_rejects_errors_duplicates_and_foreign_directories(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); directory = root / N.PACKAGE_DIR; directory.mkdir(parents=True)
            (directory / 'state_test.go').write_text('package proofs\n')
            row = {'ImportPath':N.PACKAGE,'Dir':str(directory),'TestGoFiles':['state_test.go']}
            self.assertEqual(len(N.package_selection(json.dumps(row),root)),1)
            for value in ('',json.dumps(row)*2,json.dumps(row|{'Error':{'Err':'original error'}}),
                          json.dumps(row|{'DepsErrors':[{'Err':'missing embed'}]}),
                          json.dumps(row|{'Dir':str(root)}),json.dumps(row|{'TestGoFiles':['../escape.go']})):
                with self.subTest(value=value),self.assertRaises((ValueError,OSError)): N.package_selection(value,root)

    def test_native_grouping_preserves_actual_cases_and_partial_reports(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); directory = root / 'forks/karst'; directory.mkdir(parents=True)
            (directory / 'junit.xml').write_text('<testsuites><testsuite name="proofs" tests="1" failures="1"><testcase classname="actual/package" name="TestGenerateForkState"><failure message="original failure"/></testcase></testsuite></testsuites>')
            N.native_junit(root,['karst','lagoon'])
            cases = list(N.ET.parse(root / 'native.junit.xml').iter('testcase'))
            self.assertEqual(len(cases),1)
            self.assertEqual(cases[0].get('classname'),'karst/actual/package')
            self.assertEqual(cases[0].find('failure').get('message'),'original failure')
            self.assertEqual(set(N.G.read(root / 'native.metadata.json')['original_junit_sha256']),{'karst'})


class _LiveTestsFixtures:

    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        helper = M.MainTests()
        helper.install_helpers(self.root)
        for path in SCRIPTS.glob('*.py'):
            shutil.copy2(path, self.root / 'ops/ci/runtime' / path.name)
        shell = self.root / 'ops/scripts'
        shell.mkdir()
        for name in ('gotestsum-split.sh', 'split-test-logs.sh'):
            shutil.copy2(SCRIPTS.parents[1] / 'scripts' / name, shell / name)
        shutil.copy2(SCRIPTS / 'nut-prefork-test.sh', self.root / 'ops/ci/runtime/nut-prefork-test.sh')
        original = (N.ROOT / 'justfile').read_text()
        start = original.index("[script('bash')]\n_check-nut-prefork-states:")
        end = original.index('# Checks that committed NUT pre-fork states regenerate', start)
        child = original.index('_nut-prefork-state-for fork:')
        child_end = original.index('\n\n', child)
        (self.root / 'justfile').write_text(original[start:end] + '\n' + original[child:child_end] + '\n')
        files = {'go.mod': 'module github.com/ethereum-optimism/optimism\n\ngo 1.26.0\n', '.gitignore': '.ci/\ntmp/\nop-core/superchain/superchain-configs.zip\npackages/contracts-bedrock/cache/\npackages/contracts-bedrock/forge-artifacts/\n', 'packages/contracts-bedrock/foundry.toml': '[profile.default]\nsrc="src"\nout="forge-artifacts"\nsolc="0.8.28"\n', 'packages/contracts-bedrock/src/Fixture.sol': '// SPDX-License-Identifier: MIT\npragma solidity ^0.8.28;\ncontract Fixture { function value() external pure returns(uint256) { return 42; } }\n', N.PACKAGE_DIR + '/state_test.go': 'package proofs\nimport("os";"testing";"path/filepath";"strings";"time")\nfunc TestGenerateForkState(t *testing.T) {\n fork:=os.Getenv("OP_E2E_GEN_PREFORK_STATE"); if fork==""||fork=="jovian" {t.Fatalf("unexpected fork %q",fork)}\n root:="../../../.."\n marker:=filepath.Join(root,".ci/fresh-marker"); f,err:=os.OpenFile(marker,os.O_APPEND|os.O_CREATE|os.O_WRONLY,0600)\n if err!=nil {t.Fatal(err)}; if _,err=f.WriteString(fork+"\\n");err!=nil {t.Fatal(err)};f.Close()\n if os.Getenv("NUT_FIXTURE_CANCEL")=="1" {t.Log("actual fork waiting for cancellation");time.Sleep(60*time.Second)}\n if os.Getenv("NUT_FIXTURE_FAIL")=="1" {t.Fatal("original isolated pre-fork failure")}\n if data,err:=os.ReadFile(filepath.Join(root,"packages/contracts-bedrock/forge-artifacts/Fixture.sol/Fixture.json"));err!=nil||!strings.Contains(string(data),"bytecode") {t.Fatalf("source-relative contract artifact %s %v",data,err)}\n t.Run("write/"+fork,func(t *testing.T) {path:=filepath.Join(root,"op-core/nuts/state",fork+"_state.json");data,err:=os.ReadFile(path);if err!=nil {t.Fatal(err)}\n if os.Getenv("NUT_FIXTURE_DRIFT")=="1" {data=[]byte("{\\"drift\\":1}\\n")};if err=os.WriteFile(path,data,0640);err!=nil {t.Fatal(err)}})\n}\nfunc TestGenerateForkStateNew(t *testing.T) {}\nfunc TestOutsideSelection(t *testing.T) {t.Fatal("unselected test must never execute")}\n'}
        for name, text in files.items():
            path = self.root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text)
        for fork in ('jovian', 'karst', 'lagoon', 'new-fork'):
            path = self.root / 'op-core/nuts/state' / (fork + '_state.json')
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(json.dumps({'fork': fork}) + '\n')
        helper.install_bundle(self.root)
        self.sha = helper.commit(self.root)
        self.env = {**os.environ, 'CI': 'true', 'CI_COMMIT_SHA': self.sha, 'CI_BRANCH': 'codex/rwx-ci-pilot', 'GOCACHE': str(self.root / '.ci/compiler'), 'GOMODCACHE': str(self.root / '.ci/go-cache/pr-checks/modules'), 'RWX_RUN_ID': os.environ.get('RWX_RUN_ID', 'isolated-prefork-fixture'), 'RWX_TASK_ATTEMPT_NUMBER': '1'}
        self.env.pop('NUT_PREFORK_REPORT_ROOT', None)
        cache = self.root / '.ci/go-cache/pr-checks/modules/fixture'
        cache.parent.mkdir(parents=True)
        cache.write_text('verified original module fixture')
        cache.chmod(292)
        cache.parent.chmod(365)
        self.addCleanup(cache.parent.chmod, 493)
        subprocess.run(['forge', 'build', '--root', 'packages/contracts-bedrock'], cwd=self.root, env=self.env, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        for kind, paths in [('go-modules', ['.ci/go-cache/pr-checks/modules']), ('contracts', ['packages/contracts-bedrock/forge-artifacts'])]:
            subprocess.run([sys.executable, 'ops/ci/runtime/go-artifacts.py', 'pack', kind, *paths], cwd=self.root, env=self.env, check=True)
        result = subprocess.run([sys.executable, 'ops/ci/runtime/main-checks.py', '--prepare-superchain'], cwd=self.root, env=self.env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        self.assertEqual(result.returncode, 0, result.stdout)
        self.dependencies = self.root / '.ci/go-tests/dependencies'
        self.bundle = self.root / '.ci/main-checks/prep-superchain'
        self.report = self.root / '.ci/nut-prefork/run'

    def args(self, provider):
        argv = [sys.executable, 'ops/ci/runtime/nut-prefork.py', '--provider', provider]
        if provider == 'rwx':
            argv += ['--module-artifact', str(self.dependencies / 'go-modules'), '--contract-artifact', str(self.dependencies / 'contracts'), '--bundle-artifact', str(self.bundle)]
        return argv

    def run_fixture(self, provider='circleci', **extra):
        return subprocess.run(self.args(provider), cwd=self.root, env={**self.env, **extra}, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, timeout=120)

    def retain(self, name):
        target = N.ROOT / '.ci/nut-prefork/helper-fixtures' / name
        shutil.rmtree(target, ignore_errors=True)
        shutil.copytree(self.report, target)


@unittest.skipUnless(os.environ.get('RWX_LIVE_NUT_PREFORK_FIXTURE') == '1', 'Opt-in real Go, Forge and original Just recipes')
class LiveTests(_LiveTestsFixtures, unittest.TestCase):

    def test_real_complete_fresh_execution_original_selection_and_strict_parity(self):
        paired = self.root / '.ci/paired'
        for provider, key in [('circleci', 'circle'), ('rwx', 'rwx')]:
            result = self.run_fixture(provider)
            self.assertEqual(result.returncode, 0, result.stdout)
            shutil.copytree(self.report, paired / key)
            self.retain('fresh-' + key)
        self.assertEqual((self.root / '.ci/fresh-marker').read_text().splitlines(), ['karst', 'lagoon', 'new-fork'] * 2)
        for change in ('source', 'command', 'omission', 'coverage', 'bundle', 'dependency'):
            original = {}
            for provider in ('circle', 'rwx'):
                directory = paired / provider
                name = {'source': 'settings.json', 'command': 'check.stage.json', 'omission': 'selection.json', 'coverage': 'coverage.json', 'bundle': 'dependencies/superchain-configs.zip', 'dependency': 'dependencies/contracts.json'}.get(change)
                if change == 'dependency' and provider == 'circle':
                    continue
                path = directory / name
                original[provider] = (path, path.read_bytes(), (directory / 'final.json').read_bytes())
                if change == 'bundle':
                    path.write_bytes(b'corrupt original bundle')
                else:
                    value = N.G.read(path)
                    if change == 'source':
                        value['source_sha'] = 'f' * 40
                    elif change == 'command':
                        value['argv'] = ['true']
                    elif change == 'omission':
                        value['forks'] = value['forks'][:-1]
                    elif change == 'coverage':
                        value['attempts_per_fork'] = 2
                    elif change == 'dependency':
                        value['settings'] = {'profile': 'default'}
                    N.S.write(path, value)
                final = N.G.read(directory / 'final.json')
                final['original_sha256'][name] = N.S.digest(path)
                N.S.write(directory / 'final.json', final)
            for provider, (path, data, final) in original.items():
                path.write_bytes(data)
                (paired / provider / 'final.json').write_bytes(final)

    def test_original_git_diff_rejects_state_drift_even_when_go_cases_pass(self):
        result = self.run_fixture(NUT_FIXTURE_DRIFT='1')
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertNotEqual(N.G.read(self.report / 'check.stage.json')['exit_code'], 0)
        self.assertIn('diff --git', (self.report / 'check.log').read_text())
        self.assertEqual(N.G.read(self.report / 'coverage.json')['outcomes'], {'pass': 9})
        self.assertNotEqual(N.G.read(self.report / 'final.json')['exit_code'], 0)
        self.retain('original-state-drift')

    def test_actual_go_failure_retains_partial_originals_without_retries_or_fake_forks(self):
        result = self.run_fixture(NUT_FIXTURE_FAIL='1')
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertEqual(sorted((p.name for p in (self.report / 'forks').iterdir())), ['karst'])
        source = (self.report / 'forks/karst/original.json').read_text()
        self.assertIn('original isolated pre-fork failure', source)
        self.assertIn('"Action":"fail"', source)
        self.assertNotEqual(N.G.read(self.report / 'final.json')['exit_code'], 0)
        self.retain('intentional-failure')

    def test_actual_cancellation_keeps_signal_and_original_partial_fork_report(self):
        log = self.root / '.ci/cancel.log'
        with log.open('w') as output:
            child = subprocess.Popen(self.args('circleci'), cwd=self.root, env={**self.env, 'NUT_FIXTURE_CANCEL': '1'}, stdout=output, stderr=subprocess.STDOUT)
            try:
                deadline = time.monotonic() + 90
                while time.monotonic() < deadline and child.poll() is None:
                    source = self.report / 'forks/karst/original.json'
                    if source.exists() and 'actual fork waiting for cancellation' in source.read_text():
                        break
                    time.sleep(0.1)
                else:
                    self.fail('Real fork did not reach cancellation boundary: ' + log.read_text())
                child.send_signal(signal.SIGTERM)
                self.assertNotEqual(child.wait(timeout=20), 0)
            finally:
                if child.poll() is None:
                    child.kill()
                    child.wait()
        self.retain('cancellation')
        final = N.G.read(self.report / 'final.json')
        self.assertEqual(final['exit_code'], 143)
        self.assertEqual(N.G.read(self.report / 'check.stage.json')['exit_code'], 143)
        self.assertEqual(N.G.read(self.report / 'forks/karst/tests.stage.json')['exit_code'], -signal.SIGTERM)
        self.assertIn('forks/karst/original.json', final['original_sha256'])
        self.assertFalse((self.report / 'forks/lagoon').exists())

    def test_runtime_rejects_stale_settings_tools_and_corrupt_bundle_before_tests(self):
        metadata = self.dependencies / 'contracts/metadata.json'
        original = metadata.read_bytes()
        for name, value in [('stale-source', N.G.read(metadata) | {'commit_sha': 'f' * 40}), ('stale-settings', N.G.read(metadata) | {'settings': {'profile': 'default'}}), ('stale-tools', N.G.read(metadata) | {'tool_versions': {'go': 'wrong actual toolchain'}})]:
            N.S.write(metadata, value)
            result = self.run_fixture('rwx')
            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertFalse((self.report / 'packages.stage.json').exists())
            self.assertFalse((self.report / 'forks').exists())
            self.retain('rejected-' + name)
            metadata.write_bytes(original)
        bundle = self.root / N.MAIN.BUNDLE
        saved = bundle.read_bytes()
        bundle.write_bytes(b'corrupt reusable bundle')
        result = self.run_fixture('rwx')
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertFalse((self.report / 'packages.stage.json').exists())
        self.assertFalse((self.root / '.ci/fresh-marker').exists())
        self.retain('rejected-bundle')
        bundle.write_bytes(saved)


if __name__=='__main__':unittest.main()
