"""Exercise complete L2 original comparisons; synthetic fixtures add no coverage."""
import importlib.util
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import test_contract_l2_fork as fixtures

SPEC=importlib.util.spec_from_file_location('compare_l2',Path(__file__).with_name('compare-contract-l2-fork.py'))
C=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(C)
L=C.L


class L2ParityTests(unittest.TestCase):
    server=fixtures.L2Tests.server

    def setUp(self):
        temporary=tempfile.TemporaryDirectory(prefix='L2 parity originals ');self.addCleanup(temporary.cleanup)
        self.directory=Path(temporary.name)
        self.url,_=self.server()
        handle=mock.patch.object(L,'RPC',self.url);handle.start();self.addCleanup(handle.stop)
        self.circle=self.fixture('circleci');self.native=self.fixture('rwx')

    def fixture(self,provider):
        directory=self.directory/provider;directory.mkdir()
        sha='a'*40;workspace='/fixture';identity='test/L2/fork/Portal.t.sol:Portal'
        settings={'source_sha':sha,'branch':'codex/rwx-ci-pilot','profile':'ci','feature':'main','chain':'op-mainnet',
          'match_path':L.MATCH,'runtime':dict(L.RUNTIME),
          'provider':provider,'workspace_root':workspace,'inputs':{},'implementation':{},
          'tools':{name:{'sha256':'b'*64,'version':'fixture pinned tool'} for name in ('forge','cast','go','just')},
          'rwx_run_id':'c'*32 if provider=='rwx' else None,'rwx_task_attempt':'1' if provider=='rwx' else None}
        L.write(directory/'settings.json',settings)
        config={'fuzz':{'runs':128},'invariant':{'runs':64,'depth':32},'root':workspace+'/packages/contracts-bedrock'}
        L.write(directory/'foundry-config.json',config)
        L.write(directory/'discovery.json',{'test/L2/fork/Portal.t.sol':{'Portal':['test_prove','test_upgrade']}})
        artifact_name='packages/contracts-bedrock/forge-artifacts/Portal.t.sol/Portal.json'
        artifact=directory/'compiler-artifacts'/artifact_name;artifact.parent.mkdir(parents=True)
        methods={'test_prove()':'12345678','test_upgrade()':'87654321'}
        L.write(artifact,{'metadata':{'settings':{'compilationTarget':{'test/L2/fork/Portal.t.sol':'Portal'}}},
                         'methodIdentifiers':methods,'bytecode':{'object':'0x6000'}})
        previous=L.UP.ROOT
        try:
            L.UP.ROOT=directory/'compiler-artifacts'
            bindings=L.UP.compiler_signatures(directory/'compiler-artifacts/packages/contracts-bedrock/forge-artifacts')
        finally:L.UP.ROOT=previous
        selected=L.UP.selection(L.read(directory/'discovery.json'),bindings)
        L.write(directory/'signature-bindings.json',bindings);L.write(directory/'selection.json',selected)
        L.write(directory/'compiled.json',{artifact_name:L.digest(artifact),'packages/contracts-bedrock/scripts/go-ffi/go-ffi':'d'*64})
        (directory/'submodules.txt').write_text(' '+('e'*40)+' packages/contracts-bedrock/lib/forge-std (fixture)\n')
        xml='<testsuites><testsuite name="'+identity+'">'+''.join('<testcase name="'+name+'"/>' for _,name in selected)+'</testsuite></testsuites>'
        (directory/'original.junit.xml').write_text(xml)
        L.write(directory/'coverage.json',L.UP.junit(directory/'original.junit.xml',selected,bindings))
        commands={'foundry-config':['forge','config','--json'],'go-ffi':['just','build-go-ffi'],'contracts-build':['forge','build'],
          'discovery':['forge','test','--list','--json','--match-path',L.MATCH],
          'nut-bundle-check':['just','nut-bundle-check-no-build'],'tests':['just','test-l2-fork-upgrade'] + L.TEST_ARGS}
        for name,argv in commands.items():
            stdout=directory/(name+('.json' if name in ('foundry-config','discovery') else '.log'))
            if not stdout.exists():stdout.write_text('fixture original '+name+'\n')
            stderr=directory/(name+'.stderr.log');stderr.write_bytes(b'')
            L.write(directory/(name+'.stage.json'),{'argv':argv,'cwd':workspace+'/packages/contracts-bedrock','exit_code':0,
                'started_at':1,'elapsed_seconds':1,'stdout_sha256':L.digest(stdout),'stderr_sha256':L.digest(stderr)})
        preflight=directory/'preflight';preflight.mkdir()
        with mock.patch.object(L.UP,'revision',return_value=sha):
            chosen=L.preflight(preflight,'latest' if provider=='rwx' else '66')
        L.seal(preflight,0,[]);L.write(directory/'block.json',chosen);L.block(directory,chosen['number'])
        L.seal(directory,0,[],len(selected));return directory

    def equal(self):
        with mock.patch.object(C,'authority'):return C.equal(self.circle,self.native)

    def reseal(self,directory=None):
        directory=directory or self.native
        final=L.read(directory/'final.json');L.seal(directory,final['exit_code'],final['errors'],final['tests'])

    def test_complete_raw_selection_junit_compiler_and_common_rpc_parity(self):
        circle,native=self.equal()
        self.assertEqual(circle['selection'],native['selection'])
        self.assertEqual(circle['coverage']['outcomes'],{'pass':2})
        self.assertFalse(circle['preflight']['latest_discovery']);self.assertTrue(native['preflight']['latest_discovery'])
        self.assertEqual(len(native['preflight']['common_rpc']),5)
        self.assertEqual(len(list((self.native/'preflight').glob('rpc-*-request.json'))),6)

    def test_resealed_missing_original_junit_verdict_cannot_pass(self):
        path=self.native/'original.junit.xml';path.write_text('<testsuites/>');self.reseal()
        with self.assertRaisesRegex(ValueError,'verdict'):self.equal()

    def test_resealed_extra_discovery_cannot_hide_unexecuted_test(self):
        value=L.read(self.native/'discovery.json');value['test/L2/fork/Portal.t.sol']['Portal'].append('test_missing')
        L.write(self.native/'discovery.json',value);self.reseal()
        with self.assertRaisesRegex(ValueError,'signatures'):self.equal()

    def test_resealed_compiler_signature_binding_must_match_raw_artifact(self):
        value=L.read(self.native/'signature-bindings.json')
        value['test/L2/fork/Portal.t.sol:Portal']['methods']['test_prove()']='00000000'
        L.write(self.native/'signature-bindings.json',value);self.reseal()
        with self.assertRaisesRegex(ValueError,'compiler signatures'):self.equal()

    def test_resealed_skip_reason_and_outcome_difference_is_rejected(self):
        path=self.native/'original.junit.xml'
        path.write_text(path.read_text().replace('<testcase name="test_upgrade()"/>','<testcase name="test_upgrade()"><skipped message="changed original guard"/></testcase>'))
        L.write(self.native/'coverage.json',L.UP.junit(path,L.read(self.native/'selection.json'),L.read(self.native/'signature-bindings.json')))
        self.reseal()
        with self.assertRaisesRegex(ValueError,'parity differs at coverage'):self.equal()

    def test_resealed_missing_extra_wrong_id_and_nonfinite_rpc_frames_fail(self):
        for name in ('extra','wrong-id','nonfinite','missing'):
            directory=self.directory/name;directory.mkdir()
            with mock.patch.object(L.UP,'revision',return_value='a'*40):L.preflight(directory,'66')
            L.seal(directory,0,[])
            if name=='extra':(directory/'rpc-5-attempt-1.json').write_text('{}')
            elif name=='wrong-id':
                path=directory/'rpc-0-attempt-1.json';value=L.read(path);value['id']=2;L.write(path,value)
                metadata=directory/'rpc-0-attempt-1.metadata.json';value=L.read(metadata);value['response_sha256']=L.digest(path);L.write(metadata,value)
            elif name=='nonfinite':
                path=directory/'rpc-0-attempt-1.metadata.json';value=L.read(path);value['elapsed_seconds']=float('inf');L.write(path,value)
            else:(directory/'rpc-0-attempt-1.json').unlink()
            L.seal(directory,0,[])
            with self.subTest(change=name),self.assertRaises((ValueError,FileNotFoundError)):C.preflight(directory,'a'*40)

    def test_resealed_diagnostic_pass_cannot_replace_original_failing_execution(self):
        L.write(self.native/'rerun.stage.json',{'argv':['just','test-l2-fork-upgrade-rerun'],'exit_code':0});self.reseal()
        with self.assertRaisesRegex(ValueError,'diagnostic rerun'):self.equal()

    def test_resealed_unbounded_or_changed_rpc_concurrency_is_rejected(self):
        settings=L.read(self.native/'settings.json');settings['runtime']['threads']=6
        L.write(self.native/'settings.json',settings);self.reseal()
        with self.assertRaisesRegex(ValueError,'RPC concurrency'):self.equal()

    def test_resealed_original_command_cannot_omit_rpc_budget(self):
        path=self.native/'tests.stage.json';stage=L.read(path)
        stage['argv']=['just','test-l2-fork-upgrade','--threads','1']
        L.write(path,stage);self.reseal()
        with self.assertRaisesRegex(ValueError,'original L2 command'):self.equal()

    def test_runtime_original_block_and_retry_history_must_match_preflight(self):
        path=self.native/'rpc-1-attempt-1.json';value=L.read(path);value['result']['hash']='0x'+'f'*64;L.write(path,value)
        metadata=self.native/'rpc-1-attempt-1.metadata.json';value=L.read(metadata);value['response_sha256']=L.digest(path);L.write(metadata,value);self.reseal()
        with self.assertRaisesRegex(ValueError,'runtime block'):self.equal()


if __name__=='__main__':unittest.main()
