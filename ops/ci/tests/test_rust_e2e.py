#!/usr/bin/env python3
"""Exercise complete E2E discovery, partitions, fresh binaries and failure reports."""
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import tempfile
import time
import unittest
from unittest.mock import patch

SCRIPTS = Path(__file__).resolve().parents[1] / 'runtime'
SPEC = importlib.util.spec_from_file_location('rust_e2e', Path(__file__).resolve().parents[1] / 'runtime' / 'rust-e2e.py')
E2E = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(E2E)


class E2ETests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)

    def listing(self, names=('TestNew', 'TestOld')):
        (self.root / 'listing.log').write_text('\n'.join(names) + '\n')

    def test_every_discovered_test_is_assigned_to_exactly_one_shard(self):
        self.listing()
        names, excluded = E2E.discovery('proof', self.root)
        self.assertEqual(names, ['TestNew', 'TestOld']); self.assertEqual(excluded, [])
        shards = E2E.partition(names, 8)
        self.assertEqual(sorted(n for shard in shards for n in shard), names)
        self.assertEqual(sum(not shard for shard in shards), 6)
        many = ['Test%02d' % index for index in range(37)]
        shards = E2E.partition(many, 8)
        self.assertEqual(sorted(n for shard in shards for n in shard), many)
        self.assertEqual([len(shard) for shard in shards], [5, 5, 5, 5, 5, 4, 4, 4])

    def test_empty_or_duplicate_listing_never_becomes_empty_success(self):
        for mode in ('empty', 'duplicate'):
            self.listing(names=() if mode == 'empty' else ('TestNew', 'TestNew'))
            with self.subTest(mode=mode), self.assertRaises(ValueError): E2E.discovery('proof', self.root)

    def test_selection_writes_one_shard_and_rejects_an_unknown_shard(self):
        E2E.write(self.root / 'manifest.json', {'shards': [['TestA', 'TestC'], ['TestB'], []]})
        E2E.select(self.root, self.root, 0)
        self.assertEqual((self.root / 'assigned.txt').read_text(), 'TestA\nTestC\n')
        self.assertEqual(E2E.read(self.root / 'selection.json'), {'shard_index': 0, 'assigned_tests': ['TestA', 'TestC']})
        E2E.select(self.root, self.root, 2)
        self.assertEqual((self.root / 'assigned.txt').read_text(), '')
        for index in (-1, 3):
            with self.subTest(index=index), self.assertRaises(ValueError): E2E.select(self.root, self.root, index)

    def test_examples_and_fuzz_are_explicit_in_original_sharding_scope(self):
        self.listing(('TestNew', 'ExampleDemo', 'FuzzInputs'))
        names, excluded = E2E.discovery('proof', self.root)
        self.assertEqual(names, ['TestNew']); self.assertEqual(excluded, ['ExampleDemo', 'FuzzInputs'])
        self.assertEqual(E2E.discovery('simple-kona', self.root)[0], ['ExampleDemo', 'FuzzInputs', 'TestNew'])

    def test_original_failure_and_missing_verdicts_are_retained(self):
        (self.root / 'events').mkdir()
        E2E.write(self.root / 'selection.json', {'assigned_tests': ['TestOriginal', 'TestMissing']})
        events = [{'Action': 'run', 'Package': E2E.PREFIX + E2E.JOBS['proof']['package'], 'Test': 'TestOriginal'},
                  {'Action': 'fail', 'Package': E2E.PREFIX + E2E.JOBS['proof']['package'], 'Test': 'TestOriginal'}]
        (self.root / 'events/failure.json').write_text('\n'.join(json.dumps(e) for e in events) + '\n')
        self.assertEqual(E2E.report('proof', self.root, 17), 17)
        final = E2E.read(self.root / 'final.json')
        self.assertEqual(final['exit_code'], 17)
        self.assertTrue(final['report_errors'])
        self.assertEqual(E2E.read(self.root / 'coverage.json')['missing'], ['TestMissing'])
        self.assertEqual((self.root / 'original.json').read_bytes(), (self.root / 'events/failure.json').read_bytes())

    def test_complete_verdicts_report_success_and_a_native_report(self):
        (self.root / 'events').mkdir()
        E2E.write(self.root / 'selection.json', {'assigned_tests': ['TestOriginal']})
        package = E2E.PREFIX + E2E.JOBS['proof']['package']
        events = [{'Action': 'run', 'Package': package, 'Test': 'TestOriginal'},
                  {'Action': 'pass', 'Package': package, 'Test': 'TestOriginal', 'Elapsed': 1.5},
                  {'Action': 'pass', 'Package': package, 'Elapsed': 1.5}]
        (self.root / 'events/TestOriginal.json').write_text('\n'.join(json.dumps(e) for e in events) + '\n')
        self.assertEqual(E2E.report('proof', self.root, 0), 0)
        self.assertEqual(E2E.read(self.root / 'final.json'), {'exit_code': 0, 'report_errors': []})
        self.assertEqual(E2E.read(self.root / 'coverage.json')['cases'], {'TestOriginal': {'outcome': 'pass', 'elapsed': 1.5}})
        self.assertIn('TestOriginal', (self.root / 'native.json').read_text())

    def test_configurations_preserve_validator_and_sequencer_roles(self):
        a, b = E2E.environment('simple-kona'), E2E.environment('simple-kona-sequencer')
        self.assertEqual(a['KONA_SEQUENCER_WITH_RETH'], '0'); self.assertEqual(b['KONA_SEQUENCER_WITH_RETH'], '1')
        self.assertEqual(a['OP_VALIDATOR_WITH_RETH'], '1'); self.assertEqual(b['OP_VALIDATOR_WITH_RETH'], '0')
        self.assertEqual(E2E.environment('op-reth')['OP_RETH_ENABLE_PROOF_HISTORY'], 'true')
        self.assertEqual(E2E.JOBS['proof']['timeout'], '60m')

    def test_op_reth_environment_matches_owning_recipe(self):
        # RWX runs the compiled suite without the recipe, so it must export
        # exactly what test-e2e-sysgo exports, and nothing the recipe dropped.
        recipe = (SCRIPTS.parents[2] / 'rust/op-reth/tests/justfile').read_text()
        body = re.search(r'^test-e2e-sysgo:.*\n((?:[ \t]+.*\n|\n)*)', recipe, re.M)
        self.assertIsNotNone(body)
        source = str(SCRIPTS.parents[2] / 'rust/op-reth/tests')
        exported = {name: os.path.normpath(value.strip('"').replace('{{SOURCE_DIR}}', source))
                    if '{{SOURCE_DIR}}' in value else value.strip('"')
                    for name, value in re.findall(r'^\s*export (\w+)=(\S+)$', body[1], re.M)}
        self.assertEqual(E2E.environment('op-reth'), exported)

    def test_only_proofs_use_explicit_host_parallelism(self):
        with patch.dict(os.environ, PARALLEL='32'):
            self.assertEqual(E2E.parallel_flag('proof'), 32)
            for job in ('restart', 'simple-kona', 'simple-kona-sequencer', 'op-reth'):
                self.assertIsNone(E2E.parallel_flag(job))


@unittest.skipUnless(os.environ.get('RWX_LIVE_GO_FIXTURE') == '1', 'Opt-in real Go/gotestsum fixture')
class LiveE2ETests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        for name in ('rust-e2e.py', 'rust-e2e.sh', 'go-compiled-tests.py', 'rust-workspace-report.py', 'ci-report.py', 'go-report.py'):
            path = self.root / 'ops/ci/runtime' / name; path.parent.mkdir(parents=True, exist_ok=True); shutil.copyfile(SCRIPTS / name, path)
        path = self.root / 'ops/scripts/split-test-logs.sh'; path.parent.mkdir(); shutil.copyfile(SCRIPTS.parents[1] / 'scripts/split-test-logs.sh', path)
        (self.root / 'go.mod').write_text('module github.com/ethereum-optimism/optimism\n\ngo 1.24.0\n')
        (self.root / '.gitignore').write_text('.ci/\nexecuted\n')
        package = self.root / 'rust/kona/tests/node/common'; package.mkdir(parents=True)
        (package / 'fixture.txt').write_text('correct package directory')
        (package / 'fixture_test.go').write_text('''package common
import ("testing"; "os"; "time")
func TestFresh(t *testing.T) { b,e:=os.ReadFile("fixture.txt"); if e!=nil || string(b)!="correct package directory" { t.Fatal("fixture path") }; f,e:=os.OpenFile("../../../../../executed",os.O_CREATE|os.O_APPEND|os.O_WRONLY,0600); if e!=nil { t.Fatal(e) }; defer f.Close(); f.WriteString("fresh\\n") }
func TestSkip(t *testing.T) { t.Skip("original skip reason") }
func TestFailure(t *testing.T) { if os.Getenv("E2E_INTENTIONAL_FAILURE")=="1" { t.Fatal("original failure") } }
func TestSlow(t *testing.T) { if os.Getenv("E2E_WAIT")=="1" { time.Sleep(time.Minute) } }
''')
        for args in (['init', '-q'], ['config', 'user.email', 'fixture@example.invalid'], ['config', 'user.name', 'CI Fixture'], ['add', '.'], ['commit', '-qm', 'fixture']):
            subprocess.run(['git', *args], cwd=self.root, check=True)
        self.env = {**os.environ, 'CI_SHARD_INDEX': '0', 'PARALLEL': '2'}
        result = self.run_mode('compile')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def run_mode(self, mode, **env):
        return subprocess.run(['bash', 'ops/ci/runtime/rust-e2e.sh', mode, 'simple-kona'], cwd=self.root,
                              env={**self.env, **env}, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    def test_fresh_execution_twice_original_skip_and_fixture_paths(self):
        for _ in range(2):
            result = self.run_mode('run')
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual((self.root / 'executed').read_text(), 'fresh\nfresh\n')
        report = self.root / '.ci/rust-e2e/reports/simple-kona'
        self.assertEqual(E2E.read(report / 'coverage.json')['cases']['TestSkip']['outcome'], 'skip')
        self.assertIn('original skip reason', (report / 'original.json').read_text())

    def test_real_failure_retains_original_json_junit_and_failing_status(self):
        result = self.run_mode('run', E2E_INTENTIONAL_FAILURE='1')
        self.assertNotEqual(result.returncode, 0)
        report = self.root / '.ci/rust-e2e/reports/simple-kona'
        final = E2E.read(report / 'final.json')
        self.assertNotEqual(final['exit_code'], 0); self.assertEqual(final['report_errors'], [])
        self.assertEqual(E2E.read(report / 'coverage.json')['cases']['TestFailure']['outcome'], 'fail')
        self.assertIn('original failure', (report / 'original.json').read_text())
        self.assertIn('<failure', (report / 'junit/package.xml').read_text())

    @unittest.skipUnless(os.environ.get('RWX_LIVE_FORGE_FIXTURE') == '1', 'Opt-in real runtime Forge fixture')
    def test_op_reth_builds_the_nested_Forge_fixture(self):
        tests = self.root / 'rust/op-reth/tests'; package = tests / 'proofs/core'; package.mkdir(parents=True)
        contracts = tests / 'proofs/contracts'; (contracts / 'src').mkdir(parents=True)
        (self.root / 'justfile').write_text('build-contracts:\n  @echo Wrong root recipe >&2; exit 17\n')
        (tests / 'justfile').write_text('SOURCE_DIR := justfile_directory()\nbuild-contracts:\n'
            '  cd "{{SOURCE_DIR}}/proofs/contracts" && forge build\n')
        (contracts / 'foundry.toml').write_text('[profile.default]\nsrc="src"\nout="artifacts"\n')
        (contracts / 'src/Fixture.sol').write_text('// SPDX-License-Identifier: MIT\npragma solidity 0.8.15;\n'
            'contract Fixture { function value() public pure returns (uint256) { return 42; } }\n')
        (package / 'fixture_test.go').write_text('''package core
import ("testing"; "os"; "encoding/json")
func TestRuntimeForge(t *testing.T) {
 b,e:=os.ReadFile("../contracts/artifacts/Fixture.sol/Fixture.json"); if e!=nil { t.Fatal(e) }
 var artifact struct { Bytecode struct { Object string } }; if e=json.Unmarshal(b,&artifact); e!=nil { t.Fatal(e) }
 if len(artifact.Bytecode.Object)<4 { t.Fatal("Missing actual compiled fixture bytecode") }
}
''')
        for args in (['add', '.'], ['commit', '-qm', 'nested Forge fixture']):
            subprocess.run(['git', *args], cwd=self.root, check=True)
        for mode in ('compile', 'run'):
            result = subprocess.run(['bash', 'ops/ci/runtime/rust-e2e.sh', mode, 'op-reth'], cwd=self.root,
                                    env=self.env, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        report = self.root / '.ci/rust-e2e/reports/op-reth'
        stage = E2E.read(report / 'proof-contracts.stage.json')
        self.assertEqual(stage['cwd'], 'rust/op-reth/tests')
        self.assertEqual(stage['argv'], ['just', 'build-contracts'])
        self.assertEqual(E2E.read(report / 'coverage.json')['cases']['TestRuntimeForge']['outcome'], 'pass')

    def test_proof_runs_each_assigned_test_in_its_own_process_with_explicit_parallelism(self):
        package = self.root / 'rust/kona/tests/proofs'; package.mkdir(parents=True)
        (package / 'fixture_test.go').write_text('package proofs\nimport ("flag"; "fmt"; "os"; "testing")\n'
            'func record(t *testing.T) { f,e:=os.OpenFile("../../../../executed",os.O_CREATE|os.O_APPEND|os.O_WRONLY,0600); if e!=nil { t.Fatal(e) }; defer f.Close(); '
            'fmt.Fprintf(f, "%s %d %s\\n", t.Name(), os.Getpid(), flag.Lookup("test.parallel").Value) }\n'
            + ''.join('func Test%d(t *testing.T) { record(t) }\n' % index for index in range(9)))
        for mode in ('compile', 'run'):
            result = subprocess.run(['bash', 'ops/ci/runtime/rust-e2e.sh', mode, 'proof'], cwd=self.root,
                                    env=self.env, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        # Nine tests on eight shards: shard 0 runs the first and the ninth.
        rows = [line.split() for line in (self.root / 'executed').read_text().splitlines()]
        self.assertEqual([(name, parallel) for name, _, parallel in rows], [('Test0', '2'), ('Test8', '2')])
        self.assertNotEqual(rows[0][1], rows[1][1])
        report = self.root / '.ci/rust-e2e/reports/proof'
        self.assertEqual(sorted(p.name for p in (report / 'junit').iterdir()), ['Test0.xml', 'Test8.xml'])
        self.assertEqual(E2E.read(report / 'final.json'), {'exit_code': 0, 'report_errors': []})

    def test_cancellation_preserves_partial_reports_and_cannot_pass(self):
        process = subprocess.Popen(['bash', 'ops/ci/runtime/rust-e2e.sh', 'run', 'simple-kona'], cwd=self.root,
            env={**self.env, 'E2E_WAIT': '1'}, start_new_session=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        path = self.root / '.ci/rust-e2e/reports/simple-kona/events/package.json'
        deadline = time.monotonic() + 20
        while time.monotonic() < deadline:
            if path.exists() and '"Test":"TestSlow"' in path.read_text(): break
            if process.poll() is not None: break
            time.sleep(.1)
        os.killpg(process.pid, signal.SIGTERM)
        out, err = process.communicate(timeout=20)
        self.assertNotEqual(process.returncode, 0, out + err)
        report = self.root / '.ci/rust-e2e/reports/simple-kona'
        final = E2E.read(report / 'final.json')
        self.assertNotEqual(final['exit_code'], 0)
        self.assertIn('TestSlow', (report / 'original.json').read_text())


if __name__ == '__main__': unittest.main()
