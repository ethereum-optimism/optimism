#!/usr/bin/env python3
"""Missing/unreachable archive inputs fail before expensive builds, without leaks."""
import importlib.util
import io
import json
import os
from pathlib import Path
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('rpc', Path(__file__).with_name('go-rpc-preflight.py'))
RPC = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(RPC)

class ArchivePreflightTest(unittest.TestCase):
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
