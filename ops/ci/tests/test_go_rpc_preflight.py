#!/usr/bin/env python3
"""Missing/unreachable archive inputs fail before expensive builds, without leaks."""
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('rpc', Path(__file__).resolve().parents[1] / 'runtime' / 'go-rpc-preflight.py')
RPC = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(RPC)

class ArchivePreflightTest(unittest.TestCase):
    def test_preflight_runs_from_its_declared_bootstrap_artifact(self):
        root = Path(__file__).resolve().parents[3]
        config = json.loads(subprocess.check_output(['yq', '-o=json', '.', str(root / '.rwx/go-tests.yml')], text=True))
        bootstrap = next(task for task in config['tasks'] if task['key'] == 'bootstrap-inputs')
        artifact = next(item for item in bootstrap['outputs']['artifacts'] if item['key'] == 'ci-scripts')
        preflight = next(task for task in config['tasks'] if task['key'] == 'rpc-preflight')
        with tempfile.TemporaryDirectory() as tmp:
            mount = Path(tmp) / 'ci-scripts'
            shutil.copytree(root / artifact['path'], mount, ignore=shutil.ignore_patterns('__pycache__'))
            env = dict(os.environ, CI_SCRIPTS=str(mount))
            env.update({name: '' for name in RPC.NAMES})
            result = subprocess.run(['bash', '-c', preflight['run']], cwd=tmp, env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode, 1, result.stderr)
            self.assertIn('An archive RPC input is unavailable or invalid', result.stderr)
            self.assertNotIn("can't open file", result.stderr)

    def test_missing_input_and_provider_errors_do_not_leak_credentials(self):
        for inputs in [{}, {name:'https://secret-fixture.example/rpc' for name in RPC.NAMES}]:
            with patch.dict(os.environ, inputs, clear=True), patch.object(RPC.urllib.request, 'urlopen', side_effect=OSError('secret-fixture')):
                with self.assertRaises(ValueError) as error: RPC.check()
                self.assertNotIn('secret-fixture', str(error.exception))

    def test_both_probes_require_historical_state_results(self):
        inputs={name:'https://secret-fixture.example/rpc' for name in RPC.NAMES}
        for payload, succeeds in [({'id':1,'result':'0x0'},True), ({'id':1,'error':{'message':'secret-fixture'}},False), ({'id':2,'result':'0x0'},False)]:
            def answer(request, timeout):
                body=json.loads(request.data)
                self.assertEqual(body['params'][1], '0x1')
                return io.BytesIO(json.dumps(payload).encode())
            with patch.dict(os.environ,inputs,clear=True), patch.object(RPC.urllib.request,'urlopen',side_effect=answer) as call:
                if succeeds:
                    RPC.check()
                    self.assertEqual(call.call_count,2)
                else:
                    with self.assertRaises(ValueError) as error: RPC.check()
                    self.assertNotIn('secret-fixture',str(error.exception))

if __name__=='__main__': unittest.main()
