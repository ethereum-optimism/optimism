#!/usr/bin/env python3
"""Execute the shared Just entrypoint through Circle and RWX selection paths."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
PACKAGE = 'github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/base'


@unittest.skipUnless(shutil.which('just') and shutil.which('jq'), 'Just and jq required')
class AcceptanceRunnerTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for directory in ['op-acceptance-tests/tests', 'ops/ci', 'ops/scripts', 'bin']:
            (self.root / directory).mkdir(parents=True)
        justfile = (ROOT / 'op-acceptance-tests/justfile').read_text().splitlines()
        justfile[0] = 'REPO_ROOT := "' + str(self.root) + '"'
        (self.root / 'op-acceptance-tests/justfile').write_text('\n'.join(justfile) + '\n')
        for name in ['acceptance-manifest.py', 'go-package-shards.py']:
            shutil.copyfile(ROOT / 'ops/ci' / name, self.root / 'ops/ci' / name)
        for name in ['gotestsum-split.sh', 'split-test-logs.sh', 'shard-tests.sh']:
            shutil.copy2(ROOT / 'ops/scripts' / name, self.root / 'ops/scripts' / name)
        self.script('go', '''import json, os, sys
p = os.environ['FIXTURE_PACKAGE']
if sys.argv[1] == 'version': print('go version fixture linux/amd64')
elif sys.argv[1] == 'list': print(json.dumps({'ImportPath': p}))
else:
 if os.environ.get('DISCOVERY_FAIL'): sys.exit(2)
 for n in ['TestOne', 'TestTwo']: print(json.dumps({'Action':'output','Package':p,'Output':n+'\\n'}))
 print(json.dumps({'Action':'pass','Package':p}))
''')
        self.script('circleci', '''from pathlib import Path
import sys
print(Path(sys.argv[-1]).read_text().splitlines()[0])
''')
        self.script('gotestsum', '''import json, os, sys
from pathlib import Path
a=sys.argv[1:]
def value(key):
 for i,s in enumerate(a):
  if s.startswith(key+'='): return s.split('=',1)[1]
  if s == key: return a[i+1]
p=Path(value('--jsonfile')); p.parent.mkdir(parents=True,exist_ok=True)
p.write_text(json.dumps({'Action':'pass','Package':os.environ['FIXTURE_PACKAGE'],'Test':'TestOne'})+'\\n')
j=Path(value('--junitfile')); j.parent.mkdir(parents=True,exist_ok=True); j.write_text('<testsuite/>')
Path(os.environ['FIXTURE_ARGS']).write_text(json.dumps(a))
sys.exit(int(os.environ.get('VERDICT_STATUS','0')))
''')
        self.env = {**os.environ, 'PATH': str(self.root / 'bin') + ':' + os.environ['PATH'],
                    'CIRCLE_NODE_TOTAL': '2', 'CIRCLE_NODE_INDEX': '0', 'CIRCLE_SHA1': 'a'*40,
                    'FIXTURE_PACKAGE': PACKAGE, 'FIXTURE_ARGS': str(self.root / 'args.json')}

    def script(self, name, body):
        path = self.root / 'bin' / name
        path.write_text('#!' + shutil.which('python3') + '\n' + body)
        path.chmod(0o755)

    def run_entrypoint(self, **env):
        return subprocess.run(['just', 'acceptance-test'], cwd=self.root / 'op-acceptance-tests',
                              env={**self.env, **env}, text=True, capture_output=True)

    def test_static_workflow_rejects_an_unsupported_shard_total(self):
        shutil.copyfile(ROOT / 'ops/ci/acceptance-tests.sh', self.root / 'ops/ci/acceptance-tests.sh')
        result = subprocess.run(['bash', str(self.root / 'ops/ci/acceptance-tests.sh'), 'discover'],
                                env={**self.env, 'CI_SHARD_TOTAL': '9'}, text=True, capture_output=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('exactly eight shards', result.stderr)
        self.assertFalse((self.root / '.ci/acceptance/discovery').exists())

    def test_runtime_supplies_verified_rust_paths_and_pinned_geth(self):
        shutil.copyfile(ROOT / 'ops/ci/acceptance-tests.sh', self.root / 'ops/ci/acceptance-tests.sh')
        (self.root / 'ops/ci/acceptance-report.py').write_text('')
        for binary in ['kona-host', 'kona-client', 'kona-node', 'kona-sp1-proposer',
                       'op-reth', 'op-reth-sdm-fixture']:
            path = self.root / 'rust/target/release' / binary
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text('#!/bin/sh\nexit 0\n'); path.chmod(0o755)
        executor = self.root / '.circleci-cache/rust-binaries/kona-sp1-super-range-executor'
        executor.parent.mkdir(parents=True)
        executor.write_text('#!/bin/sh\nexit 0\n'); executor.chmod(0o755)
        self.script('go', "print('/fixture/go')")
        self.script('geth', "print('Geth fixture pinned version')")
        self.script('mise', "import os; print(os.environ['FIXTURE_GETH'])")
        self.script('just', "import json, os; from pathlib import Path; Path(os.environ['FIXTURE_ARGS']).write_text(json.dumps({k:v for k,v in os.environ.items() if k.startswith('RUST_BINARY_PATH_')}))")
        result = subprocess.run(['bash', str(self.root / 'ops/ci/acceptance-tests.sh'), 'run'],
                                env={**self.env, 'CI_SHARD_TOTAL':'8',
                                     'FIXTURE_GETH':str(self.root / 'bin/geth')},
                                text=True, capture_output=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        paths = json.loads((self.root / 'args.json').read_text())
        self.assertEqual(len(paths), 7)
        self.assertTrue(all(Path(value).is_file() for value in paths.values()))
        tools = json.loads((self.root / 'tmp/testlogs/dependencies/runtime-tools.json').read_text())
        self.assertEqual(len(tools['geth_sha256']), 64)
        self.assertIn('pinned version', tools['geth_version'])
        (self.root / 'args.json').unlink()
        (self.root / 'bin/geth').unlink()
        missing = subprocess.run(['bash', str(self.root / 'ops/ci/acceptance-tests.sh'), 'run'],
                                 env={**self.env, 'CI_SHARD_TOTAL':'8',
                                      'FIXTURE_GETH':str(self.root / 'bin/geth')},
                                 text=True, capture_output=True)
        self.assertNotEqual(missing.returncode, 0)
        self.assertFalse((self.root / 'args.json').exists())

    def test_circle_retains_actual_selection_and_fresh_flags(self):
        result = self.run_entrypoint()
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)
        logs = next((self.root / 'op-acceptance-tests/logs').glob('testrun-*'))
        selection = json.loads((logs / 'discovery/selection.json').read_text())
        self.assertEqual(selection['assigned_tests'], [{'package': PACKAGE, 'name': 'TestOne'}])
        args = json.loads((self.root / 'args.json').read_text())
        self.assertIn('-count=1', args)
        self.assertIn('-run=^(TestOne)$', args)

    def test_listing_failure_never_runs_verdict(self):
        result = self.run_entrypoint(DISCOVERY_FAIL='1')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((self.root / 'args.json').exists())

    def test_rwx_verified_selection_preserves_failing_status_and_original_reports(self):
        discovery = self.root / 'discovery'
        discovery.mkdir()
        (discovery / 'packages.json').write_text(json.dumps({'ImportPath': PACKAGE}))
        events = [{'Action':'output','Package':PACKAGE,'Output':n+'\n'} for n in ['TestOne','TestTwo']]
        events.append({'Action':'pass','Package':PACKAGE})
        (discovery / 'listing.json').write_text('\n'.join(json.dumps(e) for e in events))
        subprocess.run(['python3', str(self.root / 'ops/ci/acceptance-manifest.py'), 'create', '--directory', str(discovery), '--total', '2'], env=self.env, check=True)
        result = self.run_entrypoint(CI_ACCEPTANCE_MANIFEST=str(discovery), CI_SHARD_INDEX='0', CI_SHARD_TOTAL='2', VERDICT_STATUS='1', ACCEPTANCE_TEST_HIDE_FAILURE_OUTPUT='true')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('--hide-summary=output', json.loads((self.root / 'args.json').read_text()))
        self.assertTrue(list((self.root / 'op-acceptance-tests/results').glob('*.xml')))
        self.assertTrue(list((self.root / 'op-acceptance-tests/logs').glob('*/raw_go_events-0.log')))


if __name__ == '__main__': unittest.main()
