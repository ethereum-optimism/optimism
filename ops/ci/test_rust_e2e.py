#!/usr/bin/env python3
"""Exercise complete E2E discovery, partitions, fresh binaries and failure reports."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import tempfile
import time
import unittest
from unittest.mock import patch

SCRIPTS = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location('rust_e2e', SCRIPTS / 'rust-e2e.py')
E2E = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(E2E)


class E2ETests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)

    def listing(self, names=('TestNew', 'TestOld'), error=False):
        E2E.write(self.root / 'packages.json', {'ImportPath': E2E.PREFIX + E2E.JOBS['proof']['package'],
            'TestGoFiles': ['fixture_test.go'], **({'DepsErrors': [{'Err': 'missing fixture'}]} if error else {})})
        (self.root / 'listing.log').write_text('\n'.join(names) + '\n')
        for name in ('packages', 'listing'): E2E.write(self.root / (name + '.stage.json'), {'exit_code': 0})

    def test_new_packages_tests_and_missing_timings_remain_selected(self):
        self.listing()
        names, excluded = E2E.discovery('proof', self.root)
        self.assertEqual(names, ['TestNew', 'TestOld']); self.assertEqual(excluded, [])
        shards = E2E.partition(names, 8, {'TestOld': 20})
        self.assertEqual(sorted(n for shard in shards for n in shard), names)
        self.assertEqual(sum(not shard for shard in shards), 6)

    def test_invalid_listing_or_discovery_never_becomes_empty_success(self):
        for mode in ('empty', 'duplicate', 'dependency', 'exit'):
            self.listing(names=() if mode == 'empty' else ('TestNew', 'TestNew') if mode == 'duplicate' else ('TestNew',), error=mode == 'dependency')
            if mode == 'exit': E2E.write(self.root / 'listing.stage.json', {'exit_code': 3})
            with self.subTest(mode=mode), self.assertRaises(ValueError): E2E.discovery('proof', self.root)

    def test_examples_and_fuzz_are_explicit_in_original_sharding_scope(self):
        self.listing(('TestNew', 'ExampleDemo', 'FuzzInputs'))
        names, excluded = E2E.discovery('proof', self.root)
        self.assertEqual(names, ['TestNew']); self.assertEqual(excluded, ['ExampleDemo', 'FuzzInputs'])
        data = E2E.read(self.root / 'packages.json')
        data['ImportPath'] = E2E.PREFIX + E2E.JOBS['simple-kona']['package']
        E2E.write(self.root / 'packages.json', data)
        self.assertEqual(E2E.discovery('simple-kona', self.root)[0], ['ExampleDemo', 'FuzzInputs', 'TestNew'])

    def test_duplicate_assignments_stale_inputs_corrupt_binaries_rejected(self):
        current = {'source_sha': 'pinned', 'settings': E2E.settings('proof')}
        self.listing()
        manifest = {'version': 1, **current, 'tests': ['TestNew', 'TestOld'], 'excluded_listing': [],
                    'durations': None, 'shards': E2E.partition(['TestNew', 'TestOld'], 8)}
        names = ['packages.json', 'packages.stage.json', 'packages.log', 'compile.log', 'compile.stage.json',
                 'reporter.stage.json', 'reporter.log', 'listing.stage.json', 'listing.log', 'manifest.json', 'suite.test', 'test2json']
        for name in names:
            path = self.root / name
            if not path.exists(): path.write_text('{"exit_code":0}' if name.endswith('.stage.json') else 'artifact')
        for mode in ('okay', 'duplicate', 'stale', 'corrupt', 'missing'):
            E2E.write(self.root / 'manifest.json', manifest)
            (self.root / 'suite.test').write_text('compiled')
            if mode == 'duplicate':
                changed = json.loads(json.dumps(manifest)); changed['shards'][0] += changed['tests']
                E2E.write(self.root / 'manifest.json', changed)
            E2E.write(self.root / 'metadata.json', {'version': 1, **current,
                'files': {n: E2E.GO.digest(self.root / n) for n in names}})
            if mode == 'stale':
                data = E2E.read(self.root / 'metadata.json'); data['source_sha'] = 'old'; E2E.write(self.root / 'metadata.json', data)
            if mode == 'corrupt': (self.root / 'suite.test').write_text('corrupt')
            if mode == 'missing': (self.root / 'suite.test').unlink()
            with self.subTest(mode=mode), patch.object(E2E, 'binding', return_value=current):
                if mode == 'okay': E2E.verify('proof', self.root)
                else:
                    with self.assertRaises((ValueError, OSError)): E2E.verify('proof', self.root)

    def test_original_failure_and_missing_verdicts_are_retained(self):
        (self.root / 'events').mkdir(); (self.root / 'junit').mkdir()
        E2E.write(self.root / 'selection.json', {'assigned_tests': ['TestOriginal', 'TestMissing']})
        events = [{'Action': 'run', 'Package': E2E.PREFIX + E2E.JOBS['proof']['package'], 'Test': 'TestOriginal'},
                  {'Action': 'fail', 'Package': E2E.PREFIX + E2E.JOBS['proof']['package'], 'Test': 'TestOriginal'}]
        (self.root / 'events/failure.json').write_text('\n'.join(json.dumps(e) for e in events) + '\n')
        self.assertEqual(E2E.report('proof', self.root, 17), 17)
        final = E2E.read(self.root / 'final.json')
        self.assertEqual(final['exit_code'], 17)
        self.assertTrue(final['report_errors'])
        self.assertEqual(E2E.read(self.root / 'coverage.json')['missing'], ['TestMissing'])
        self.assertEqual(final['original_sha256']['events/failure.json'], E2E.GO.digest(self.root / 'events/failure.json'))

    def test_configurations_preserve_validator_and_sequencer_roles(self):
        a, b = E2E.environment('simple-kona'), E2E.environment('simple-kona-sequencer')
        self.assertEqual(a['KONA_SEQUENCER_WITH_RETH'], '0'); self.assertEqual(b['KONA_SEQUENCER_WITH_RETH'], '1')
        self.assertEqual(a['OP_VALIDATOR_WITH_RETH'], '1'); self.assertEqual(b['OP_VALIDATOR_WITH_RETH'], '0')
        self.assertEqual(E2E.environment('op-reth')['OP_DEVSTACK_PROOF_VALIDATOR_EL'], 'op-reth-proof-v1')
        self.assertEqual(E2E.settings('proof')['timeout'], '60m')


@unittest.skipUnless(os.environ.get('RWX_LIVE_GO_FIXTURE') == '1', 'Opt-in real Go/gotestsum fixture')
class LiveE2ETests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        for name in ('rust-e2e.py', 'rust-e2e.sh', 'go-compiled-tests.py', 'go-package-shards.py', 'go-suite.py',
                     'rust-workspace-report.py', 'go-report.py'):
            path = self.root / 'ops/ci' / name; path.parent.mkdir(parents=True, exist_ok=True); shutil.copyfile(SCRIPTS / name, path)
        path = self.root / 'ops/scripts/split-test-logs.sh'; path.parent.mkdir(); shutil.copyfile(SCRIPTS.parent / 'scripts/split-test-logs.sh', path)
        for name in ('go.sum', 'mise.toml', 'rust/kona/tests/justfile', 'rust/op-reth/tests/justfile', 'ops/ci/rust-e2e-timings.json'):
            path = self.root / name; path.parent.mkdir(parents=True, exist_ok=True); path.write_text('')
        (self.root / 'ops/ci/rust-e2e-timings.json').write_text('{}')
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
        self.sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=self.root, text=True).strip()
        self.env = {**os.environ, 'CI_COMMIT_SHA': self.sha, 'CI_E2E_PROVIDER': 'rwx', 'CI_SHARD_INDEX': '0', 'PARALLEL': '2'}
        result = self.run_mode('compile')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def run_mode(self, mode, **env):
        return subprocess.run(['bash', 'ops/ci/rust-e2e.sh', mode, 'simple-kona'], cwd=self.root,
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

    def test_cancellation_preserves_partial_reports_and_cannot_pass(self):
        process = subprocess.Popen(['bash', 'ops/ci/rust-e2e.sh', 'run', 'simple-kona'], cwd=self.root,
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
