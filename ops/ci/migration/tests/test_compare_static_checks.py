#!/usr/bin/env python3
"""Reject stale sources, altered selections and fabricated static-check parity."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
import xml.etree.ElementTree as ET

SPEC=importlib.util.spec_from_file_location('compare_static',Path(__file__).resolve().parents[2] / 'migration' / 'compare-static-checks.py')
P=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(P)
C=P.C;SHA='a'*40;JOB='semgrep-scan-local'


class CompleteOriginals(unittest.TestCase):
    def setUp(self):
        tmp=tempfile.TemporaryDirectory();self.addCleanup(tmp.cleanup);self.directories={}
        for provider in ('circle','rwx'):
            d=Path(tmp.name)/provider;d.mkdir();self.directories[provider]=d;root='/'+provider
            source={'source.py':'1'*64,'.semgrep/rules/original.yaml':'2'*64,'.semgrep/tests/original.py':'3'*64,'.semgrepignore':'4'*64}
            settings={'source_sha':SHA,'branch':'pilot','job':JOB,'provider':'circleci' if provider=='circle' else 'rwx',
                'command':C.COMMANDS[JOB],'workspace_root':root,'tool_version':C.VERSIONS[JOB],
                'environment':C.semgrep_environment('pilot',SHA),'input_sha256':source,
                'baseline':{'ref':'develop','commit':'b'*40,'merge_base':'b'*40,'changed_files':{'source.py':'M'}},
                'rwx_run_id':'c'*32 if provider=='rwx' else None,'rwx_task_attempt':'1'}
            self.write(d/'settings.json',settings);self.write(d/'inputs-after.json',source)
            self.write(d/'selection.json',{'rules':{'.semgrep/rules/original.yaml':'2'*64},
                'fixtures':{'.semgrep/tests/original.py':'3'*64},'ignore':{'.semgrepignore':'4'*64}})
            self.write(d/'check.json',{'version':C.VERSIONS[JOB],'results':[],
                'errors':[{'code':3,'level':'warn','type':'PartialParsing','path':'source.py','message':'original warning'}],
                'paths':{'scanned':[root+'/source.py'],'skipped':[{'path':'ignored.py','reason':'semgrepignore_patterns_match'}]},
                'time':{'profiling_times':{provider:1}},'engine_requested':'OSS','skipped_rules':[],
                'profiling_results':{'provider':provider}})
            (d/'check.log').write_text('')
            self.write(d/'check.stage.json',{'argv':C.COMMANDS[JOB],'cwd':root,'exit_code':0,'stdin':'devnull',
                'log_sha256':C.digest(d/'check.log')})
            suite=ET.Element('testsuite',name='static-checks/'+JOB,tests='1',failures='0',skipped='0')
            ET.SubElement(suite,'testcase',name=JOB,classname=JOB);ET.ElementTree(suite).write(d/'derived.junit.xml')
            self.account(d);self.seal(d)

    def write(self,p,data):p.write_text(json.dumps(data))
    def account(self,d):
        settings=json.loads((d/'settings.json').read_text())
        coverage=C.machine_report(JOB,json.loads((d/'check.json').read_text()),settings['workspace_root'])
        self.write(d/'coverage.json',coverage);self.write(d/'scanned-source-sha256.json',C.scanned_inputs(coverage,settings['input_sha256']))
    def seal(self,d):
        self.write(d/'final.json',{'exit_code':0,'report_errors':[],'original_sha256':{
            p.name:C.digest(p) for p in d.iterdir() if p.name!='final.json'}})
    def edit(self,provider,name,action):
        d=self.directories[provider];p=d/name;data=json.loads(p.read_text());action(data);self.write(p,data)
        if name!='final.json':self.seal(d)
    def compare(self):return P.compare(self.directories,JOB,SHA)

    def test_complete_original_diagnostics_exclusions_and_bound_paths_survive(self):
        result=self.compare();self.assertTrue(result['verified_parity'])
        self.assertEqual(result['coverage']['errors'][0]['message'],'original warning')
        self.assertEqual(result['coverage']['paths']['scanned'],['<repo>/source.py'])
        self.assertEqual(result['coverage']['paths']['skipped'][0]['reason'],'semgrepignore_patterns_match')

    def test_missing_original_or_corruption_is_not_a_success(self):
        p=self.directories['rwx']/'check.json';p.write_text('{}')
        with self.assertRaisesRegex(ValueError,'corrupt'):self.compare()
        p.unlink()
        with self.assertRaises(OSError):self.compare()

    def test_only_original_manifest_declared_empty_circle_log_is_restored(self):
        (self.directories['circle']/'check.log').unlink();self.assertEqual(len(self.compare()['manifest_declared_empty_logs']),1)
        (self.directories['rwx']/'check.log').unlink()
        with self.assertRaises(OSError):self.compare()

    def test_same_stale_sha_on_both_providers_is_rejected(self):
        for provider in self.directories:self.edit(provider,'settings.json',lambda row:row.update(source_sha='d'*40))
        with self.assertRaisesRegex(ValueError,'source'):self.compare()

    def test_same_noop_command_on_both_providers_is_rejected(self):
        for provider in self.directories:self.edit(provider,'settings.json',lambda row:row.update(command=['true']))
        with self.assertRaisesRegex(ValueError,'command'):self.compare()

    def test_baseline_revision_drift_is_a_parity_error(self):
        self.edit('rwx','settings.json',lambda row:row['baseline'].update(commit='e'*40))
        with self.assertRaisesRegex(ValueError,'baseline'):self.compare()

    def test_missing_or_invalid_original_baseline_is_rejected(self):
        for value in (None,{'ref':'develop','commit':'b'*40,'merge_base':'b'*40,'changed_files':{'missing.py':'M'}}):
            for provider in self.directories:self.edit(provider,'settings.json',lambda row:row.update(baseline=value))
            with self.subTest(value=value),self.assertRaisesRegex(ValueError,'baseline'):self.compare()

    def test_disabling_incremental_scan_on_both_providers_is_rejected(self):
        for provider in self.directories:self.edit(provider,'settings.json',lambda row:row['environment'].pop('SEMGREP_BASELINE_REF'))
        with self.assertRaisesRegex(ValueError,'environment'):self.compare()

    def test_reused_native_verdict_or_unexplained_retry_is_rejected(self):
        self.edit('rwx','settings.json',lambda row:row.update(rwx_task_attempt='2'))
        with self.assertRaisesRegex(ValueError,'retry'):self.compare()

    def test_changed_exclusion_or_warning_requires_investigation(self):
        original=(self.directories['rwx']/'check.json').read_text()
        for field in ('paths','errors'):
            d=self.directories['rwx'];row=json.loads(original)
            if field=='paths':row['paths']['skipped'][0]['reason']='different exclusion'
            else:row['errors'][0]['message']='different warning'
            self.write(d/'check.json',row);self.account(d);self.seal(d)
            with self.subTest(field=field),self.assertRaisesRegex(ValueError,'coverage'):self.compare()

    def test_unbound_or_duplicate_scanned_source_cannot_be_hidden(self):
        d=self.directories['rwx'];row=json.loads((d/'check.json').read_text())
        row['paths']['scanned']=['source.py','source.py']
        with self.assertRaisesRegex(ValueError,'Duplicate'):C.machine_report(JOB,row,'/rwx')
        row['paths']['scanned']=['unbound.py'];self.write(d/'check.json',row);self.seal(d)
        with self.assertRaisesRegex(ValueError,'unbound'):self.compare()

    def test_original_failure_cannot_be_replaced_by_passing_junit(self):
        self.edit('rwx','final.json',lambda row:row.update(exit_code=2))
        with self.assertRaisesRegex(ValueError,'Failed'):self.compare()

    def test_empty_original_baseline_selection_remains_explicit_and_complete(self):
        for d in self.directories.values():
            row=json.loads((d/'check.json').read_text());row['paths']['scanned']=[];self.write(d/'check.json',row)
            self.account(d);self.seal(d)
        self.assertEqual(self.compare()['coverage']['paths']['scanned'],[])


if __name__=='__main__':unittest.main()
