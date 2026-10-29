#!/usr/bin/env python3
"""Execute the unchanged Just entrypoint through the RWX wrapper and its circleci shim."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[3]
PACKAGE = 'github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/base'


@unittest.skipUnless(shutil.which('just'), 'Just required')
class AcceptanceRunnerTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.reports = self.root / '.ci/acceptance/reports'
        for directory in ['op-acceptance-tests/tests', 'ops/ci/runtime', 'ops/scripts', 'bin']:
            (self.root / directory).mkdir(parents=True)
        justfile = (ROOT / 'op-acceptance-tests/justfile').read_text().splitlines()
        justfile[0] = 'REPO_ROOT := "' + str(self.root) + '"'
        (self.root / 'op-acceptance-tests/justfile').write_text('\n'.join(justfile) + '\n')
        for name in ['acceptance-tests.sh', 'acceptance-manifest.py', 'acceptance-report.py', 'go-package-shards.py', 'go-report.py']:
            shutil.copyfile(ROOT / 'ops/ci/runtime' / name, self.root / 'ops/ci/runtime' / name)
        shutil.copytree(ROOT / 'ops/ci/runtime/acceptance-circleci', self.root / 'ops/ci/runtime/acceptance-circleci')
        for name in ['gotestsum-split.sh', 'split-test-logs.sh', 'shard-tests.sh']:
            shutil.copy2(ROOT / 'ops/scripts' / name, self.root / 'ops/scripts' / name)
        for binary in ['rust/target/release/' + name for name in ['kona-host', 'kona-client', 'kona-node', 'op-zk-proposer', 'op-reth']] \
                + ['.circleci-cache/rust-binaries/kona-sp1-super-range-executor']:
            path = self.root / binary
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text('#!/bin/sh\nexit 0\n'); path.chmod(0o755)
        # Discovery lists JSON events; the recipe lists plain names.
        self.script('go', '''import json, os, sys
p, names = os.environ['FIXTURE_PACKAGE'], os.environ.get('FIXTURE_TESTS', 'TestOne TestTwo').split()
if sys.argv[1] == 'version': print('go version fixture linux/amd64')
elif sys.argv[1] == 'env': print('/fixture/go')
elif sys.argv[1] == 'list': print(json.dumps({'ImportPath': p}))
elif os.environ.get('DISCOVERY_FAIL'): sys.exit(2)
elif '-json' in sys.argv:
 for n in names: print(json.dumps({'Action':'output','Package':p,'Output':n+'\\n'}))
 print(json.dumps({'Action':'pass','Package':p}))
else: print('\\n'.join(names) + '\\nok  ' + p + ' 0.01s')
''')
        self.script('gotestsum', '''import json, os, re, sys
from pathlib import Path
a=sys.argv[1:]
def value(key):
 for i,s in enumerate(a):
  if s.startswith(key+'='): return s.split('=',1)[1]
  if s == key: return a[i+1]
p, status = os.environ['FIXTURE_PACKAGE'], int(os.environ.get('VERDICT_STATUS','0'))
pattern = next((s.split('=',1)[1] for s in a if s.startswith('-run=')), '')
events = []
for n in os.environ.get('FIXTURE_TESTS', 'TestOne TestTwo').split():
 if re.search(pattern, n): events += [{'Action':'run','Package':p,'Test':n}, {'Action':'fail' if status else 'pass','Package':p,'Test':n}]
j=Path(value('--jsonfile')); j.parent.mkdir(parents=True,exist_ok=True); j.write_text(''.join(json.dumps(e)+'\\n' for e in events))
x=Path(value('--junitfile')); x.parent.mkdir(parents=True,exist_ok=True); x.write_text('<testsuite/>')
Path(os.environ['FIXTURE_ARGS']).write_text(json.dumps(a))
sys.exit(status)
''')
        self.script('geth', "print('Geth fixture pinned version')")
        self.script('mise', "import os; print(os.environ['FIXTURE_GETH'])")
        self.env = {**os.environ, 'PATH': str(self.root / 'bin') + ':' + os.environ['PATH'],
                    'CI_COMMIT_SHA': 'a'*40, 'CI_SHARD_INDEX': '0', 'CI_SHARD_TOTAL': '8',
                    'CI_ACCEPTANCE_DISCOVERY': str(self.root / '.ci/acceptance/discovery'),
                    'FIXTURE_PACKAGE': PACKAGE, 'FIXTURE_ARGS': str(self.root / 'args.json'),
                    'FIXTURE_GETH': str(self.root / 'bin/geth')}

    def script(self, name, body):
        path = self.root / 'bin' / name
        path.write_text('#!' + shutil.which('python3') + '\n' + body)
        path.chmod(0o755)

    def wrapper(self, phase, **env):
        return subprocess.run(['bash', str(self.root / 'ops/ci/runtime/acceptance-tests.sh'), phase],
                              env={**self.env, **env}, text=True, capture_output=True)

    def verdict(self, index, **env):
        discovery = self.wrapper('discover')
        self.assertEqual(discovery.returncode, 0, discovery.stderr)
        return self.wrapper('run', CI_SHARD_INDEX=str(index), **env)

    def test_static_workflow_rejects_an_unsupported_shard_total(self):
        result = self.wrapper('discover', CI_SHARD_TOTAL='9')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('exactly eight shards', result.stderr)
        self.assertFalse((self.root / '.ci/acceptance/discovery').exists())

    def test_runtime_supplies_verified_rust_paths_and_pinned_geth(self):
        (self.root / 'ops/ci/runtime/acceptance-report.py').write_text('')
        self.script('just', "import json, os; from pathlib import Path; Path(os.environ['FIXTURE_ARGS']).write_text(json.dumps({k:v for k,v in os.environ.items() if k.startswith('RUST_BINARY_PATH_')}))")
        result = self.wrapper('run')
        self.assertEqual(result.returncode, 0, result.stderr)
        paths = json.loads((self.root / 'args.json').read_text())
        self.assertEqual(len(paths), 6)
        self.assertTrue(all(Path(value).is_file() for value in paths.values()))
        tools = json.loads((self.root / 'tmp/testlogs/dependencies/runtime-tools.json').read_text())
        self.assertEqual(len(tools['geth_sha256']), 64)
        self.assertIn('pinned version', tools['geth_version'])
        (self.root / 'args.json').unlink()
        (self.root / 'bin/geth').unlink()
        missing = self.wrapper('run')
        self.assertNotEqual(missing.returncode, 0)
        self.assertFalse((self.root / 'args.json').exists())

    def test_verdict_runs_only_its_assignment_with_fresh_flags(self):
        result = self.verdict(0)
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)
        args = json.loads((self.root / 'args.json').read_text())
        self.assertIn('-count=1', args)
        self.assertIn('-run=^(TestOne)$', args)
        coverage = json.loads((self.reports / 'coverage.json').read_text())
        self.assertEqual((coverage['expected'], coverage['observed']), (1, 1))
        for name in ['original.json', 'native.json', 'junit/results-0.xml']:
            self.assertTrue((self.reports / name).is_file(), name)

    def test_failing_verdict_preserves_status_and_original_reports(self):
        result = self.verdict(0, VERDICT_STATUS='1')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(json.loads((self.reports / 'coverage.json').read_text())['missing'], [])
        self.assertIn('"fail"', (self.reports / 'original.json').read_text())
        self.assertTrue((self.reports / 'junit/results-0.xml').is_file())

    def test_valid_empty_shard_runs_nothing_and_passes(self):
        result = self.verdict(7)
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)
        self.assertFalse((self.root / 'args.json').exists())
        self.assertEqual(json.loads((self.reports / 'coverage.json').read_text())['expected'], 0)

    def test_listing_that_differs_from_discovery_never_runs_verdict(self):
        result = self.verdict(0, FIXTURE_TESTS='TestOne TestTwo TestNew')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('differs from acceptance discovery', result.stderr)
        self.assertFalse((self.root / 'args.json').exists())

    def test_listing_failure_never_runs_verdict(self):
        result = self.verdict(0, DISCOVERY_FAIL='1')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((self.root / 'args.json').exists())

    def test_shim_rejects_every_other_circleci_invocation(self):
        shim, listed = self.root / 'ops/ci/runtime/acceptance-circleci/circleci', self.root / 'listed.txt'
        listed.write_text('TestOne\nTestTwo\n')
        for argv in (['tests', 'glob', 'op-acceptance-tests/**'], ['tests', 'split'], ['tests', 'split', '--split-by=timings'],
                     ['tests', 'split', '--index=1', str(listed)], ['tests', 'split', str(listed), str(listed)], ['step', 'halt']):
            with self.subTest(argv=argv):
                result = subprocess.run([str(shim), *argv], env=self.env, text=True, capture_output=True)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('does not support', result.stderr)


if __name__ == '__main__': unittest.main()
