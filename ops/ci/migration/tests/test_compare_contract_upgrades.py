#!/usr/bin/env python3
"""Reject incomplete or mismatched original L1 upgrade evidence."""
import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
import sys
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'tests'))
from ci_test_fixtures import ReportFixtures

SPEC=importlib.util.spec_from_file_location('compare',Path(__file__).resolve().parents[2] / 'migration' / 'compare-contract-upgrades.py')
C=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(C)
SHA='a'*40


class ComparisonTests(ReportFixtures, unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup)
        self.root=Path(self.tmp.name);self.directories={}
        for provider in ['circle','rwx']:
            d=self.root/provider;d.mkdir();self.directories[provider]=d;workspace='/'+provider
            settings={'source_sha':SHA,'variant':'feature-main','chain':'op','feature':'main','branch':'pilot','profile':'liteci',
                      'fuzz_seed':'42424242','fuzz_runs':1,'provider':'circleci' if provider=='circle' else 'rwx',
                      'workspace_root':workspace,'forge':'pinned','go':'pinned','just':'pinned','input_sha256':{'source':'original'},
                      'rwx_run_id':'c'*32 if provider=='rwx' else None,'rwx_task_attempt':'1'}
            self.write(d/'settings.json',settings);self.write(d/'foundry-config.json',{'out':'out','root':workspace})
            discovery={'test/L1/Original.t.sol':{'Original':['test_a','test_skip']}};self.write(d/'discovery.json',discovery)
            methods={'test/L1/Original.t.sol:Original':{'methods':{'test_a()':'original','test_skip()':'original'},
                         'creation_bytecode':{'packages/contracts-bedrock/out/Original.t.sol/Original.json':
                             {'bytes':1,'sha256':hashlib.sha256(b'00').hexdigest()}},
                         'artifacts':{'packages/contracts-bedrock/out/Original.t.sol/Original.json':'artifact-hash'}}}
            self.write(d/'signature-bindings.json',methods);self.write(d/'compiled.json',methods['test/L1/Original.t.sol:Original']['artifacts'])
            selected=C.UP.selection(discovery,methods);self.write(d/'selection.json',selected)
            junit=d/'original.junit.xml';junit.write_text('<testsuites><testsuite name="test/L1/Original.t.sol:Original">'
                '<testcase name="test_a()"/><testcase name="test_skip()"><skipped message="original">feature disabled</skipped></testcase></testsuite></testsuites>')
            self.write(d/'coverage.json',C.UP.junit(junit,selected,methods));(d/'submodules.txt').write_text(' ' + 'd' * 40 + ' original-submodule\n')
            self.write(d/'block.json',{'source_sha':SHA,'chain_id':1,'number':66,'hash':'0x'+'b'*64,'timestamp':100,'policy':'Just current-day 00:00 UTC'})
            for name,argv in {'foundry-config':['forge','config','--json'],'go-ffi':['just','build-go-ffi'],
                              'contracts-build':['forge','build'],
                              'discovery':['forge','test','--list','--json','--match-path',C.UP.MATCH],'tests':['just','test-upgrade']}.items():
                self.write(d/(name+'.stage.json'),{'argv':argv,'cwd':workspace+'/packages/contracts-bedrock','exit_code':0})
            self.seal(d)

    def compare(self):return C.compare(self.directories,'feature-main',SHA)

    def test_complete_original_selection_signatures_skips_settings_and_block_match(self):
        self.assertTrue(self.compare()['verified_parity'])

    def test_stale_revision_failed_original_and_corrupt_file_fail_closed(self):
        d=self.directories['rwx']
        for mode in ['stale','failed','corrupt']:
            with self.subTest(mode=mode):
                self.setUp();d=self.directories['rwx']
                if mode=='stale':
                    p=d/'settings.json';row=json.loads(p.read_text());row['source_sha']='f'*40;self.write(p,row);self.seal(d)
                elif mode=='failed':
                    p=d/'final.json';row=json.loads(p.read_text());row['exit_code']=1;self.write(p,row)
                else:(d/'original.junit.xml').write_text('damaged original')
                with self.assertRaises(ValueError):self.compare()

    def test_fork_height_and_hash_drift_require_investigation(self):
        for key,value in [('number',67),('hash','0x'+'c'*64)]:
            with self.subTest(field=key):
                self.setUp();d=self.directories['rwx'];p=d/'block.json';v=json.loads(p.read_text());v[key]=value;self.write(p,v);self.seal(d)
                with self.assertRaisesRegex(ValueError,'block'):self.compare()

    def test_matching_wrong_command_is_not_a_valid_workload(self):
        for d in self.directories.values():
            p=d/'tests.stage.json';v=json.loads(p.read_text());v['argv']=['true'];self.write(p,v);self.seal(d)
        with self.assertRaisesRegex(ValueError,'command'):self.compare()

    def test_compiler_signature_binding_cannot_be_detached_from_artifacts(self):
        d=self.directories['rwx'];self.write(d/'compiled.json',{});self.seal(d)
        with self.assertRaisesRegex(ValueError,'artifact binding'):self.compare()

    def test_missing_test_extra_original_and_different_skip_are_rejected(self):
        for mode in ['missing','extra','skip']:
            with self.subTest(mode=mode):
                self.setUp();d=self.directories['rwx'];p=d/'original.junit.xml';v=p.read_text()
                if mode=='missing':v=v.replace('<testcase name="test_a()"/>','')
                elif mode=='extra':v=v.replace('test_a()','extra()')
                else:v=v.replace('feature disabled','different behavior')
                p.write_text(v)
                if mode=='skip':self.write(d/'coverage.json',C.UP.junit(p,json.loads((d/'selection.json').read_text()),json.loads((d/'signature-bindings.json').read_text())))
                self.seal(d)
                with self.assertRaises(ValueError):self.compare()

    def test_uninvestigated_retry_or_reused_verdict_remains_an_error(self):
        d=self.directories['rwx'];self.write(d/'rerun.stage.json',{'argv':['just','test-upgrade-rerun'],'exit_code':0});self.seal(d)
        with self.assertRaisesRegex(ValueError,'retry'):self.compare()
        (d/'rerun.stage.json').unlink();p=d/'settings.json';v=json.loads(p.read_text());v['rwx_run_id']=None;self.write(p,v);self.seal(d)
        with self.assertRaisesRegex(ValueError,'fresh'):self.compare()

    def test_profile_feature_and_submodule_drift_are_not_path_differences(self):
        d=self.directories['rwx'];p=d/'settings.json';v=json.loads(p.read_text());v['feature']='ZK_DISPUTE_GAME';self.write(p,v);self.seal(d)
        with self.assertRaisesRegex(ValueError,'workload'):self.compare()
        v['feature']='main';self.write(p,v);(d/'submodules.txt').write_text(' ' + 'e' * 40 + ' original-submodule\n');self.seal(d)
        with self.assertRaisesRegex(ValueError,'submodules'):self.compare()


if __name__=='__main__':unittest.main()
