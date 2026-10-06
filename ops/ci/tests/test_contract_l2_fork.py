#!/usr/bin/env python3
"""Exercise L2 RPC originals, complete settings, stale outputs and fresh failures."""
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import threading
import unittest
from unittest import mock

SPEC=importlib.util.spec_from_file_location('l2',Path(__file__).resolve().parents[1] / 'runtime' / 'contract-l2-fork.py')
L=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(L)


class L2Tests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory(prefix='L2 originals ');self.addCleanup(self.tmp.cleanup)
        self.directory=Path(self.tmp.name)

    def test_complete_ci_profile_clears_ambient_filters_features_and_rpc(self):
        with mock.patch.dict(os.environ, {'CI_BRANCH':'codex/rwx-ci-pilot','FOUNDRY_PROFILE':'liteci','FOUNDRY_FUZZ_RUNS':'1',
             'FOUNDRY_MATCH_TEST':'partial','FOUNDRY_NO_RPC_RATE_LIMIT':'true',
             'DEV_FEATURE__UNRELATED':'true','ETH_RPC_URL':'ambient','L2_FORK_CHAIN':'other'},clear=True):
            self.assertEqual(L.configure(),'codex/rwx-ci-pilot')
            self.assertEqual(os.environ['FOUNDRY_PROFILE'],'ci');self.assertEqual(os.environ['L2_FORK_TEST'],'true')
            for name in ('FOUNDRY_FUZZ_RUNS','FOUNDRY_MATCH_TEST','FOUNDRY_NO_RPC_RATE_LIMIT',
                         'DEV_FEATURE__UNRELATED','ETH_RPC_URL','L2_FORK_CHAIN'):
                self.assertNotIn(name,os.environ)
        config={'fuzz':{'runs':128},'invariant':{'runs':64,'depth':32}}
        L.effective(config)
        for changed in (config|{'match_test':'partial'},config|{'fuzz':{'runs':1}},config|{'skip':['required']},
                        config|{'invariant':{'runs':64,'depth':1}},config|{'no_rpc_rate_limit':True}):
            with self.subTest(config=changed),self.assertRaises(ValueError):L.effective(changed)

    def test_circle_nested_shell_retains_the_initialized_job_and_runtime_relay(self):
        upstream,_=self.server()
        initialization=self.directory/'bash-env'
        initialization.write_text('export L2_FORK_RPC_URL='+upstream+'\n')
        code='''import json,os,urllib.request
request=urllib.request.Request(os.environ['L2_FORK_RPC_URL'],data=b'{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}',headers={'Content-Type':'application/json'})
with urllib.request.urlopen(request,timeout=5) as response:chain=json.load(response)['result']
print(json.dumps({'rpc':os.environ['L2_FORK_RPC_URL'],'chain':chain,'modules':os.environ['GOMODCACHE']}))'''
        with mock.patch.dict(os.environ,{'CI_BRANCH':'codex/rwx-ci-pilot','BASH_ENV':str(initialization),
             'L2_FORK_RPC_URL':upstream,'GOMODCACHE':'initialized job modules'}), \
             mock.patch.object(L.UP,'CONTRACTS',self.directory):
            L.configure()
            with L.P.serve(self.directory/'runtime-rpc',upstream) as endpoint:
                os.environ['L2_FORK_RPC_URL']=endpoint
                argv=['bash','-c','exec "$1" -c "$2"','snapshot',sys.executable,code]
                self.assertEqual(L.UP.stage(self.directory,'nested-shell',argv,json_output=True),0)
            value=L.read(self.directory/'nested-shell.json')
            self.assertEqual(value,{'rpc':endpoint,'chain':'0xa','modules':'initialized job modules'})
            self.assertEqual(L.read(self.directory/'runtime-rpc/final.json')['requests'],1)

    def server(self, responses=None):
        observations=[]
        values={'eth_chainId':'0xa','eth_getBlockByNumber':{'number':'0x42','hash':'0x'+'a'*64,'timestamp':'0x64'},
                'eth_getCode':'0x123456','eth_getStorageAt':'0x'+'0'*63+'1','eth_call':'0x123456','eth_blockNumber':'0x42'}
        if responses:values.update(responses)
        class Rpc(BaseHTTPRequestHandler):
            def do_POST(self):
                body=json.loads(self.rfile.read(int(self.headers['Content-Length'])))
                observations.append({'request':body,'headers':dict(self.headers)})
                data=json.dumps({'jsonrpc':'2.0','id':body['id'],'result':values[body['method']]}).encode()
                self.send_response(200);self.send_header('Content-Type','application/json');self.end_headers();self.wfile.write(data)
            def log_message(self,*_):pass
        server=ThreadingHTTPServer(('127.0.0.1',0),Rpc);thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
        self.addCleanup(server.server_close);self.addCleanup(thread.join,5);self.addCleanup(server.shutdown)
        return 'http://127.0.0.1:'+str(server.server_port),observations

    def test_real_anonymous_rpc_preflight_retains_every_original_request_and_response(self):
        url,observations=self.server()
        with mock.patch.object(L,'RPC',url),mock.patch.object(L.UP,'revision',return_value='b'*40):
            block=L.preflight(self.directory,'66')
        self.assertEqual((block['chain_id'],block['number'],block['hash']),(10,66,'0x'+'a'*64))
        self.assertEqual(len(observations),5)
        self.assertEqual([row['request']['method'] for row in observations],
                         ['eth_chainId','eth_getBlockByNumber','eth_getCode','eth_getStorageAt','eth_call'])
        self.assertTrue(all(row['headers']['User-Agent']=='Go-http-client/1.1' for row in observations))
        self.assertTrue(all(not any('token' in key.lower() or key.lower()=='authorization' for key in row['headers']) for row in observations))
        for i,row in enumerate(observations):
            request=L.read(self.directory/('rpc-'+str(i)+'-request.json'))
            response=L.read(self.directory/('rpc-'+str(i)+'-attempt-1.json'))
            self.assertEqual(request,row['request']);self.assertEqual(response['id'],request['id'])
        L.seal(self.directory,0,[]);L.original(self.directory)
        (self.directory/'rpc-0-attempt-1.json').write_text('{}')
        with self.assertRaisesRegex(ValueError,'corrupt original'):L.original(self.directory)

    def test_shared_pilot_snapshot_avoids_independently_selected_heads_and_preserves_other_refs(self):
        url,observations=self.server()
        root=self.directory/'source';(root/'ops/ci/runtime').mkdir(parents=True)
        snapshot=root/'ops/ci/runtime/pilot-l2-fork-block.txt';snapshot.write_text('66\n')
        with mock.patch.object(L,'RPC',url),mock.patch.object(L,'ROOT',root),mock.patch.object(L.UP,'revision',return_value='a'*40):
            with mock.patch.dict(os.environ,{'CI_BRANCH':'codex/rwx-ci-pilot'},clear=True):
                chosen=L.preflight(self.directory,'latest')
            self.assertEqual(chosen['requested'],'66')
            self.assertEqual(len(observations),5)
            self.assertNotIn('eth_blockNumber',[row['request']['method'] for row in observations])
            other=self.directory/'develop';other.mkdir()
            with mock.patch.dict(os.environ,{'CI_BRANCH':'develop'},clear=True):
                chosen=L.preflight(other,'latest')
            self.assertEqual(chosen['requested'],'latest');self.assertEqual(observations[5]['request']['method'],'eth_blockNumber')
            snapshot.write_text('66; untrusted\n')
            invalid=self.directory/'invalid';invalid.mkdir()
            with mock.patch.dict(os.environ,{'CI_BRANCH':'codex/rwx-ci-pilot'},clear=True),self.assertRaisesRegex(ValueError,'snapshot'):
                L.preflight(invalid,'latest')

    def test_unsealed_block_only_mount_is_rejected_before_any_test_command(self):
        mounted=self.directory/'mounted';mounted.mkdir();pinned=mounted/'block.json';L.write(pinned,{'number':66})
        with mock.patch.object(L.UP,'stage') as stage,self.assertRaises(FileNotFoundError):
            L.verdict(self.directory,pinned)
        stage.assert_not_called()

    def test_wrong_chain_block_or_missing_state_is_rejected_with_originals_retained(self):
        for i,changed in enumerate(({'eth_chainId':'0x1'}, {'eth_getBlockByNumber':None}, {'eth_getCode':'0x'},
              {'eth_getStorageAt':'0x'+'0'*64},{'eth_call':'0x'})):
            directory=self.directory/str(i);directory.mkdir();url,_=self.server(changed)
            with self.subTest(response=changed),mock.patch.object(L,'RPC',url),self.assertRaises(ValueError):
                L.preflight(directory,'66')
            self.assertTrue(list(directory.glob('rpc-*-attempt-1.json')))
        for height in ('0','-1','0x42','66; command',''):
            with self.subTest(height=height),self.assertRaisesRegex(ValueError,'height'):L.preflight(self.directory,height)

    def test_http_failure_keeps_complete_body_and_never_retries_permanent_denial(self):
        error=L.urllib.error.HTTPError(L.RPC,403,'original denial',{},None)
        with mock.patch.object(error,'read',return_value=b'{"error":"original denial"}'), \
             mock.patch.object(L.urllib.request,'urlopen',side_effect=error) as opened:
            with self.assertRaisesRegex(ValueError,'unavailable: 403'):L.call(self.directory,'eth_chainId',[])
        self.assertEqual(opened.call_count,1)
        self.assertEqual((self.directory/'rpc-0-attempt-1.json').read_bytes(),b'{"error":"original denial"}')
        self.assertEqual(L.read(self.directory/'rpc-0-attempt-1.metadata.json')['http_status'],403)

    def test_transient_failure_keeps_both_original_attempts(self):
        error=L.urllib.error.HTTPError(L.RPC,503,'original failure',{},None)
        class Response:
            status=200
            def __enter__(self):return self
            def __exit__(self,*_):pass
            def read(self):return b'{"jsonrpc":"2.0","id":1,"result":"0xa"}'
        with mock.patch.object(error,'read',return_value=b'original 503'), \
             mock.patch.object(L.urllib.request,'urlopen',side_effect=[error,Response()]),mock.patch.object(L.time,'sleep'):
            self.assertEqual(L.call(self.directory,'eth_chainId',[]),'0xa')
        self.assertEqual((self.directory/'rpc-0-attempt-1.json').read_bytes(),b'original 503')
        self.assertEqual(L.read(self.directory/'rpc-0-attempt-2.metadata.json')['http_status'],200)

    def test_restore_rejects_stale_settings_or_corrupt_compiler_bytes(self):
        prepared=self.directory/'prepared';prepared.mkdir();runtime=self.directory/'runtime';runtime.mkdir()
        root=self.directory/'source';(root/'packages/contracts-bedrock').mkdir(parents=True)
        file=root/'packages/contracts-bedrock/original.bin';file.write_bytes(b'compiled original')
        settings={'source_sha':'a'*40,'profile':'ci','branch':'codex/rwx-ci-pilot','rwx_run_id':'b'*32,'rwx_task_attempt':'1'}
        modules=' '+('d'*40)+' packages/contracts-bedrock/lib/forge-std (fixture)\n'
        L.write(prepared/'settings.json',settings);(prepared/'submodules.txt').write_text(modules)
        L.write(prepared/'foundry-config.json',{'fuzz':{'runs':128},'invariant':{'runs':64,'depth':32}})
        L.write(prepared/'compiled.json',{'packages/contracts-bedrock/original.bin':L.digest(file)});L.seal(prepared,0,[])
        with mock.patch.object(L,'ROOT',root),mock.patch.object(L,'settings',return_value=settings|{'source_sha':'c'*40}):
            with self.assertRaisesRegex(ValueError,'Stale'):L.restored(runtime,prepared)
        file.write_bytes(b'corrupt original')
        with mock.patch.object(L,'ROOT',root),mock.patch.object(L,'settings',return_value=settings),mock.patch.object(L.UP,'command',return_value=modules.strip()):
            with self.assertRaisesRegex(ValueError,'compiled output'):L.restored(runtime,prepared)

    def test_nut_bundle_failure_prevents_tests_and_keeps_initial_command(self):
        preflight=self.directory/'preflight';preflight.mkdir();runtime=self.directory/'run';runtime.mkdir()
        value={'source_sha':'a'*40,'rpc_url':L.RPC,'chain_id':10,'number':66,'hash':'0x'+'b'*64,'timestamp':100,
               'policy':'pinned original OP Mainnet L2 fork block','requested':'66'}
        L.write(preflight/'block.json',value);L.seal(preflight,0,[])
        with mock.patch.object(L.UP,'revision',return_value='a'*40),mock.patch.object(L,'block',return_value={k:value[k] for k in ('chain_id','number','hash','timestamp')}), \
             mock.patch.object(L,'CONTRACTS',self.directory),mock.patch.object(L.UP,'stage',return_value=17) as stage, \
             mock.patch.dict(os.environ,{},clear=True):
            self.assertEqual(L.verdict(runtime,preflight/'block.json'),(17,0))
        self.assertEqual(stage.call_count,1);self.assertEqual(stage.call_args.args[2],['just','nut-bundle-check-no-build'])
        self.assertTrue((runtime/'preflight/final.json').is_file());self.assertFalse((runtime/'original.junit.xml').exists())

    def test_initial_test_failure_is_preserved_after_diagnostic_rerun(self):
        preflight=self.directory/'preflight';preflight.mkdir();runtime=self.directory/'run';runtime.mkdir()
        value={'source_sha':'a'*40,'rpc_url':L.RPC,'chain_id':10,'number':66,'hash':'0x'+'b'*64,'timestamp':100,
               'policy':'pinned original OP Mainnet L2 fork block','requested':'66'}
        L.write(preflight/'block.json',value);L.seal(preflight,0,[])
        with mock.patch.object(L.UP,'revision',return_value='a'*40),mock.patch.object(L,'block',return_value={k:value[k] for k in ('chain_id','number','hash','timestamp')}), \
             mock.patch.object(L,'CONTRACTS',self.directory),mock.patch.object(L.UP,'stage',side_effect=[0,23,0]) as stage, \
             mock.patch.dict(os.environ,{},clear=True):
            self.assertEqual(L.verdict(runtime,preflight/'block.json'),(23,0))
        self.assertEqual([call.args[2] for call in stage.call_args_list],
                         [['just','nut-bundle-check-no-build'],
                          ['just','test-l2-fork-upgrade','--threads','1','--compute-units-per-second','100'],
                          ['just','test-l2-fork-upgrade-rerun','--threads','1','--compute-units-per-second','100']])

    def test_real_child_failure_keeps_both_original_streams_and_a_failing_seal(self):
        with mock.patch.object(L.UP,'CONTRACTS',self.directory):
            status=L.UP.stage(self.directory,'intentional',[sys.executable,'-c','import sys;print("original output");print("original error",file=sys.stderr);sys.exit(17)'])
        self.assertEqual(status,17);L.seal(self.directory,status,[])
        self.assertEqual((self.directory/'intentional.log').read_text(),'original output\n')
        self.assertEqual((self.directory/'intentional.stderr.log').read_text(),'original error\n')
        with self.assertRaisesRegex(ValueError,'failed'):L.original(self.directory)


if __name__=='__main__':unittest.main()
