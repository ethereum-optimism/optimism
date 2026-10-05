"""Exercise complete Cannon discovery, fresh Go/Forge execution and cancellation."""
import importlib.util
import argparse
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

SCRIPTS = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location('cannon_go', SCRIPTS / 'cannon-go.py')
G = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(G)
SPEC = importlib.util.spec_from_file_location('compare_cannon', SCRIPTS / 'compare-cannon-go.py')
COMPARE = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(COMPARE)


class DiscoveryTests(unittest.TestCase):
    def test_new_packages_and_packages_without_tests_remain_included(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); directory = root / 'cannon/new'; directory.mkdir(parents=True)
            (directory / 'new.go').write_text('package new\n')
            row = {'ImportPath': G.PREFIX + '/new', 'Dir': str(directory), 'GoFiles': ['new.go']}
            found = G.packages(json.dumps(row), root)
            self.assertEqual(found[0]['go_files']['TestGoFiles'], [])
            self.assertEqual(found[0]['package'], G.PREFIX + '/new')
            package = row['ImportPath']
            listing = root / 'list.json'
            listing.write_text('\n'.join(json.dumps(e) for e in [
                {'Package': package, 'Action': 'start'}, {'Package': package, 'Action': 'output', 'Output': 'TestNew\n'},
                {'Package': package, 'Action': 'pass', 'Elapsed': 0}]) + '\n')
            self.assertEqual(G.listing(listing, found), {package: ['TestNew']})

    def test_failed_empty_duplicate_or_escaping_discovery_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); directory = root / 'cannon'; directory.mkdir()
            row = {'ImportPath': G.PREFIX, 'Dir': str(directory)}
            for text in ('', '{invalid', '[]', json.dumps(row) * 2,
                json.dumps(row | {'Error': {'Err': 'original compile error'}}),
                json.dumps(row | {'DepsErrors': [{'Err': 'missing embed'}]}),
                json.dumps(row | {'ImportPath': 'outside/suite'}),
                json.dumps(row | {'Dir': str(root)}), json.dumps(row | {'GoFiles': ['../outside.go']}),
                '{"ImportPath":"x","ImportPath":"y"}'):
                with self.subTest(text=text), self.assertRaises((ValueError, OSError)): G.packages(text, root)

    def test_original_settings_are_distinct_from_aggregate_go(self):
        settings = G.settings('rwx', True, True, 16)
        self.assertEqual(G.go_flags(settings), ['-timeout=10m', '-parallel=16', '-count=1', './...'])
        scheduled = G.settings('circleci', False, False, 8)
        self.assertEqual(G.go_flags(scheduled), ['-timeout=45m', '-parallel=8', './...'])
        self.assertEqual(settings['tags'], [])
        self.assertEqual(settings['rerun_fails'], 0)
        self.assertIsNone(settings['package_parallelism'])
        with self.assertRaises(ValueError): G.settings('rwx', True, False, 16)

    def test_circle_boolean_environment_and_cli_flags_preserve_freshness(self):
        for value, expected in [('true', True), ('1', True), ('false', False), ('0', False)]:
            parser = argparse.ArgumentParser(); parser.add_argument('--fresh-tests', type=G.boolean, default=value)
            self.assertIs(parser.parse_args([]).fresh_tests, expected)
            self.assertIs(parser.parse_args(['--fresh-tests', value]).fresh_tests, expected)
        with self.assertRaises(argparse.ArgumentTypeError): G.boolean('unrecognized')


@unittest.skipUnless(os.environ.get('RWX_LIVE_CANNON_GO_FIXTURE') == '1', 'Opt-in real Go, gotestsum and Forge fixtures')
class LiveTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(); self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name); scripts = self.root / 'ops/ci'; scripts.mkdir(parents=True)
        for name in ('cannon-go.py', 'compare-ci.py', 'rust-workspace-report.py', 'go-report.py', 'go-artifacts.py'):
            shutil.copy2(SCRIPTS / name, scripts / name)
        shell = self.root / 'ops/scripts'; shell.mkdir()
        for name in ('gotestsum-split.sh', 'split-test-logs.sh'): shutil.copy2(SCRIPTS.parent / 'scripts' / name, shell / name)
        files = {'go.mod': 'module github.com/ethereum-optimism/optimism\n\ngo 1.26.0\n',
            'mise.toml': '# Fixture tool versions come from the actual prepared worker.\n',
            '.gitignore': '.ci/\ntmp/\npackages/contracts-bedrock/cache/\npackages/contracts-bedrock/forge-artifacts/\n',
            'cannon/justfile': 'lint:\n    @echo "cannon lint is handled by the root lint-go target"\n',
            'cannon/no-tests/fixture.go': 'package fixture\n',
            'cannon/new/new_test.go': 'package fixture\nimport "testing"\nfunc TestNewPackage(t *testing.T) {}\n',
            'cannon/runtime/main.go': 'package main\nimport "fmt"\nfunc main() { fmt.Print("runtime fixture") }\n',
            'packages/contracts-bedrock/foundry.toml': '[profile.default]\nsrc="src"\nout="forge-artifacts"\nsolc="0.8.28"\n',
            'packages/contracts-bedrock/src/Fixture.sol': '// SPDX-License-Identifier: MIT\npragma solidity ^0.8.28;\ncontract Fixture { function value() external pure returns(uint256) { return 42; } }\n',
            'cannon/fixture/fixture_test.go': '''package fixture
import("os";"os/exec";"testing";"time";"strings")
func TestFresh(t *testing.T) {
 f,err:=os.OpenFile("../../.ci/fresh-marker",os.O_APPEND|os.O_CREATE|os.O_WRONLY,0600)
 if err!=nil {t.Fatal(err)}; defer f.Close(); if _,err=f.WriteString("fresh\\n");err!=nil {t.Fatal(err)}
 if os.Getenv("CANNON_FIXTURE_FAIL")=="1" {t.Fatal("original isolated Cannon failure")}
}
func TestSubtests(t *testing.T) {t.Run("pass",func(t *testing.T){});t.Run("skip",func(t *testing.T){t.Skip("original fixture skip")})}
func TestRuntime(t *testing.T) {
 if data,err:=os.ReadFile("../../packages/contracts-bedrock/forge-artifacts/Fixture.sol/Fixture.json");err!=nil||!strings.Contains(string(data),"bytecode") {t.Fatalf("contract fixture: %s %v",data,err)}
 if data,err:=exec.Command("go","build","-o","../../.ci/runtime-fixture","../runtime").CombinedOutput();err!=nil {t.Fatalf("runtime Go build: %s %v",data,err)}
 if data,err:=exec.Command("../../.ci/runtime-fixture").CombinedOutput();err!=nil||string(data)!="runtime fixture" {t.Fatalf("runtime Go fixture: %s %v",data,err)}
 if data,err:=exec.Command("forge","build","--root","../../packages/contracts-bedrock").CombinedOutput();err!=nil {t.Fatalf("runtime Forge build: %s %v",data,err)}
}
func TestCancel(t *testing.T) {if os.Getenv("CANNON_FIXTURE_CANCEL")=="1" {t.Log("fixture is waiting for cancellation");time.Sleep(60*time.Second)}}
'''}
        for name, text in files.items():
            path = self.root / name; path.parent.mkdir(parents=True, exist_ok=True); path.write_text(text)
        for argv in (['init', '-q', '-b', 'codex/rwx-ci-pilot'], ['config', 'user.email', 'fixture@example.invalid'], ['config', 'user.name', 'CI Fixture'],
                     ['add', '.'], ['commit', '-qm', 'complete Cannon fixture']):
            subprocess.run(['git', *argv], cwd=self.root, check=True)
        self.sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=self.root, text=True).strip()
        self.environment = {**os.environ, 'CI': 'true', 'CI_COMMIT_SHA': self.sha, 'CI_BRANCH': 'codex/rwx-ci-pilot',
            'CI_GO_FRESH_TESTS': '1',
            'GOCACHE': str(self.root / '.ci/compiler'), 'GOMODCACHE': str(self.root / '.ci/go-cache/pr-checks/modules')}
        self.report = self.root / '.ci/cannon-go/run'
        module = self.root / '.ci/go-cache/pr-checks/modules/fixture'; module.parent.mkdir(parents=True); module.write_text('original module fixture')
        module.chmod(0o444); module.parent.chmod(0o555)
        self.addCleanup(module.parent.chmod, 0o755)
        subprocess.run(['forge', 'build', '--root', 'packages/contracts-bedrock'], cwd=self.root, env=self.environment,
                       check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        for kind, paths in [('go-modules', ['.ci/go-cache/pr-checks/modules']),
                            ('contracts', ['packages/contracts-bedrock/forge-artifacts'])]:
            subprocess.run([sys.executable, 'ops/ci/go-artifacts.py', 'pack', kind, *paths], cwd=self.root,
                           env=self.environment, check=True)
        self.dependencies = self.root / '.ci/go-tests/dependencies'

    def args(self, provider='circleci'):
        argv = [sys.executable, str(self.root / 'ops/ci/cannon-go.py'), '--provider', provider]
        if provider == 'rwx': argv += ['--fresh-tests', 'true']
        if provider == 'rwx': argv += ['--module-artifact', str(self.dependencies / 'go-modules'),
                                       '--contract-artifact', str(self.dependencies / 'contracts')]
        return argv

    def run_fixture(self, provider='circleci', **extra):
        result = subprocess.run(self.args(provider), cwd=self.root, env={**self.environment, **extra,
            'RWX_RUN_ID': os.environ.get('RWX_RUN_ID', 'isolated-fixture'), 'RWX_TASK_ATTEMPT_NUMBER': '1'},
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, timeout=90)
        return result

    def retain(self, name):
        target = SCRIPTS.parent.parent / '.ci/cannon-go/helper-fixtures' / name
        shutil.rmtree(target, ignore_errors=True); shutil.copytree(self.report, target)

    def test_real_fresh_execution_complete_selection_runtime_tools_and_parity(self):
        evidence = self.root / '.ci/comparison'; evidence.mkdir()
        for provider, destination in [('circleci', 'circle'), ('rwx', 'rwx')]:
            result = self.run_fixture(provider); self.assertEqual(result.returncode, 0, result.stdout)
            shutil.copytree(self.report, evidence / destination); self.retain('fresh-' + destination)
        self.assertEqual((self.root / '.ci/fresh-marker').read_text(), 'fresh\nfresh\n')
        result = COMPARE.compare(evidence, self.sha)
        self.assertEqual(result['packages'], 4)
        self.assertEqual(result['initial_tests'], 5)
        self.assertEqual(result['outcomes'], {'pass': 6, 'skip': 1})
        self.assertEqual(result['settings']['rwx']['count'], 1)

    def test_real_failure_retains_original_case_logs_and_no_retries(self):
        result = self.run_fixture(CANNON_FIXTURE_FAIL='1'); self.assertEqual(result.returncode, 1, result.stdout)
        final = G.read(self.report / 'final.json'); coverage = G.read(self.report / 'coverage.json')
        self.assertEqual(final['exit_code'], 1); self.assertEqual(final['report_errors'], [])
        case = next(c for c in coverage['cases'] if c['name'] == 'TestFresh')
        self.assertEqual(case['outcome'], 'fail'); self.assertEqual(len(case['attempts']), 1)
        self.assertIn('original isolated Cannon failure', (self.report / 'original.json').read_text())
        self.retain('intentional-failure')
        with self.assertRaisesRegex(ValueError, 'Failed or incomplete'): COMPARE.originals(self.report, 'circleci')

    def test_real_cancellation_preserves_signal_and_partial_original_reports(self):
        log = self.root / '.ci/cancellation.log'
        with log.open('w') as stream:
            child = subprocess.Popen(self.args(), cwd=self.root,
                env={**self.environment, 'CANNON_FIXTURE_CANCEL': '1'}, stdout=stream, stderr=subprocess.STDOUT)
            try:
                deadline = time.monotonic() + 60
                while time.monotonic() < deadline and child.poll() is None:
                    path = self.report / 'original.json'
                    if path.exists() and 'fixture is waiting for cancellation' in path.read_text(): break
                    time.sleep(.1)
                else: self.fail('Actual Go fixture never reached cancellation boundary: ' + log.read_text())
                child.send_signal(signal.SIGTERM); self.assertNotEqual(child.wait(timeout=15), 0)
            finally:
                if child.poll() is None: child.kill(); child.wait()
        final = G.read(self.report / 'final.json')
        self.assertEqual(G.read(self.report / 'tests.stage.json')['exit_code'], -signal.SIGTERM)
        self.assertEqual(final['exit_code'], 128 + signal.SIGTERM)
        self.assertTrue(final['report_errors']); self.assertIn('original.json', final['original_sha256'])
        self.retain('cancellation')
        with self.assertRaisesRegex(ValueError, 'Failed or incomplete'): COMPARE.originals(self.report, 'circleci')

    def test_corrupt_and_stale_artifacts_fail_before_any_test_verdict(self):
        metadata = self.dependencies / 'contracts/metadata.json'; original = metadata.read_bytes()
        for name, value in [('stale-source', G.read(metadata) | {'commit_sha': 'f' * 40}),
                            ('stale-settings', G.read(metadata) | {'settings': {'profile': 'default'}}),
                            ('stale-toolchain', G.read(metadata) | {'tool_versions': {'go': 'wrong toolchain'}})]:
            G.S.write(metadata, value); result = self.run_fixture('rwx')
            self.assertEqual(result.returncode, 1, result.stdout)
            self.assertFalse((self.report / 'original.json').exists()); self.retain(name)
            metadata.write_bytes(original)
        archive = self.dependencies / 'contracts/files.tar.gz'; archive.write_bytes(archive.read_bytes() + b'corrupt')
        result = self.run_fixture('rwx'); self.assertEqual(result.returncode, 1, result.stdout)
        self.assertFalse((self.report / 'original.json').exists()); self.retain('corrupt-archive')


if __name__ == '__main__': unittest.main()
