"""Exercise actual gate discovery, HTTP pagination, terminal failures and originals."""
import copy
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
from unittest.mock import patch

SPEC=importlib.util.spec_from_file_location('pr_gate',Path(__file__).with_name('pr-gate.py'))
G=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(G)


class GateTests(unittest.TestCase):
    def setUp(self):
        temp=tempfile.TemporaryDirectory();self.addCleanup(temp.cleanup);self.root=Path(temp.name)
        manifest=G.read(G.MANIFEST)
        names={'ops/ci/pr-gates.json','ops/ci/pr-gate.py','ops/ci/main-checks.py','ops/ci/pr-checks.py','ops/ci/rust-workspace-report.py'}
        names|={row['config'] for row in manifest['groups'].values()}
        names|={row['circle_config'] for row in manifest['gates'].values()}
        for name in names:
            path=self.root/name;path.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(G.ROOT/name,path)
        (self.root/'.gitignore').write_text('.ci/\n')
        for argv in (['git','init','-q'],['git','add','.'],['git','-c','user.name=CI fixture','-c','user.email=ci-fixture@example.invalid','commit','-qm','Actual gate authority fixture']):
            subprocess.run(argv,cwd=self.root,check=True,stdout=subprocess.DEVNULL)
        self.sha=subprocess.check_output(['git','rev-parse','HEAD'],cwd=self.root,text=True).strip()
        for target,name,value in ((G,'ROOT',self.root),(G,'MANIFEST',self.root/'ops/ci/pr-gates.json'),(G.M,'ROOT',self.root)):
            handle=patch.object(target,name,value);handle.start();self.addCleanup(handle.stop)
        self.env={'CI_COMMIT_SHA':self.sha,'CI_BRANCH':'codex/rwx-ci-pilot','RWX_RUN_ID':'a'*32,'RWX_TASK_ATTEMPT_NUMBER':'1'}
        self.manifest,self.selection=G.configuration('required-rust-ci');self.selection=self.selection['required-rust-ci']
        self.contexts=[self.manifest['groups'][name]['status'] for name in self.selection['groups']]

    def status(self,name,identity,state='success'):
        return {'id':identity,'context':name,'state':state,'creator':self.manifest['trusted_actor'],
                'target_url':'https://cloud.rwx.com/optimism/runs/'+'b'*32}

    def server(self,rows,commit=None,shift=False):
        owner=self;requests=[]
        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                requests.append(self.path)
                if '/statuses?' in self.path:
                    if shift and len(requests)==2:rows.insert(0,owner.status('new-during-pagination',10000))
                    page=int(self.path.rsplit('page=',1)[1]);value=rows[(page-1)*100:page*100]
                else:value={'sha':owner.sha if commit is None else commit}
                data=json.dumps(value).encode();self.send_response(200);self.send_header('Content-Length',str(len(data)))
                self.end_headers();self.wfile.write(data)
            def log_message(self,*args):pass
        server=ThreadingHTTPServer(('127.0.0.1',0),Handler)
        thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
        self.addCleanup(server.server_close);self.addCleanup(server.shutdown)
        return 'http://127.0.0.1:'+str(server.server_port),requests

    def test_complete_real_authority_and_exact_terminal_dependency_names(self):
        self.assertEqual(len(self.selection['requires']),21)
        self.assertEqual(len(G.group_tasks(self.manifest,'rust-workspace')),26)
        path=self.root/'.circleci/continue/rust-ci.yml';original=path.read_text()
        for old,new in [('            - rust-fmt: terminal','            - future-required-job: terminal'),
                        ('            - rust-fmt: terminal','            - rust-fmt: success')]:
            path.write_text(original.replace(old,new,1))
            with self.assertRaisesRegex(ValueError,'Circle gate dependency'):G.configuration('required-rust-ci')
        path.write_text(original)
        native=self.root/'.rwx/rust.yml';source=native.read_text()
        native.write_text(source.replace('features-9.succeeded || features-9.failed || features-9.skipped','features-8.succeeded || features-8.failed || features-8.skipped'))
        with self.assertRaisesRegex(ValueError,'every terminal'):G.configuration('required-rust-ci')

    def test_terminal_receipts_fail_for_real_failed_and_not_run_inputs_and_preserve_safe_skips(self):
        group='rust-fmt';output=self.root/'.ci/pr-gates/groups'/group
        for selected,state,expected in [('true','succeeded',0),('true','failed',1),('true','skipped',1),('false','skipped',0),('false','succeeded',1)]:
            env={**self.env,'GROUP_SELECTED':selected}
            for attribute in ('succeeded','failed','skipped'):env['TASK_RUST_FMT_'+attribute.upper()]='true' if state==attribute else 'false'
            with patch.dict(os.environ,env):self.assertEqual(G.receipt(group),expected)
            final=G.read(output/'final.json');self.assertEqual(final['exit_code'],expected)
            self.assertEqual(final['tests'],0);self.assertTrue(final['original_sha256'])
            self.assertEqual(G.read(output/'states.json')['rust-fmt'][state],True)

    def test_latest_states_cannot_be_hidden_by_older_success_or_foreign_actors(self):
        rows=[self.status(name,i+1) for i,name in enumerate(self.contexts)]
        self.assertEqual(G.verdict(rows,self.manifest,self.selection)[0],'passed')
        for state,expected in [('pending','pending'),('failure','failed'),('error','failed')]:
            self.assertEqual(G.verdict(rows+[self.status(self.contexts[0],100,state)],self.manifest,self.selection)[0],expected)
        self.assertEqual(G.verdict(rows[:-1],self.manifest,self.selection)[0],'pending')
        foreign=copy.deepcopy(rows);foreign[0]['creator']['id']=1
        with self.assertRaisesRegex(ValueError,'Untrusted'):G.verdict(foreign,self.manifest,self.selection)
        invalid=copy.deepcopy(rows);invalid[0]['target_url']='https://cloud.rwx.com/another-org/runs/'+'b'*32
        with self.assertRaisesRegex(ValueError,'Untrusted'):G.verdict(invalid,self.manifest,self.selection)
        with self.assertRaisesRegex(ValueError,'Duplicate'):G.verdict(rows+[rows[0]],self.manifest,self.selection)

    def test_actual_http_pagination_retains_every_original_page_and_rechecks_snapshot(self):
        rows=[self.status('irrelevant-'+str(i),i+1) for i in range(205)]
        rows.extend(self.status(name,i+300) for i,name in enumerate(self.contexts))
        base,requests=self.server(rows);client=G.Client(self.root/'.ci/http','fixture-only',base)
        observed=client.statuses(self.manifest['repository'],self.sha)
        self.assertEqual(observed,rows);self.assertEqual(len(requests),4)
        self.assertEqual(G.verdict(observed,self.manifest,self.selection)[0],'passed')
        self.assertEqual(len(client.requests),4)
        for row in client.requests:
            self.assertEqual(row['http_status'],200)
            self.assertTrue((self.root/'.ci/http/responses'/row['response_sha256']).is_file())
            self.assertNotIn('Authorization',row)

    def test_actual_http_page_boundary_shift_is_retained_and_refetched_completely(self):
        rows=[self.status('irrelevant-'+str(i),i+1) for i in range(205)]
        rows.extend(self.status(name,i+300) for i,name in enumerate(self.contexts))
        base,requests=self.server(rows,shift=True);client=G.Client(self.root/'.ci/http','fixture-only',base)
        observed=client.statuses(self.manifest['repository'],self.sha)
        self.assertEqual(observed,rows);self.assertEqual(observed[0]['id'],10000)
        self.assertEqual(len(requests),7)
        self.assertEqual(G.verdict(observed,self.manifest,self.selection)[0],'passed')
        self.assertEqual(len(client.requests),7)

    def run_wait(self,rows,*,commit=None,final=True,previous=None):
        base,requests=self.server(rows,commit);real=G.Client
        class LocalClient(real):
            def __init__(self,directory,token):super().__init__(directory,token,base)
        route=self.root/'.ci/routing.json';route.parent.mkdir(exist_ok=True);route.write_text(json.dumps({'c-run_rust_ci':True}))
        env={**self.env,'GATE_ROUTING':str(route),'GATE_GITHUB_TOKEN':'fixture-only'}
        with patch.dict(os.environ,env),patch.object(G,'Client',LocalClient):
            result=G.wait('required-rust-ci',0,previous,final,poll=.01)
        return result,requests

    def test_actual_http_gate_and_sealed_continuation_reject_stale_missing_and_corrupt_originals(self):
        rows=[self.status(name,i+1) for i,name in enumerate(self.contexts)]
        result,requests=self.run_wait(rows);self.assertEqual(result,0);self.assertTrue(requests)
        report=self.root/'.ci/pr-gates/wait';G.originals(report)
        self.assertEqual(G.read(report/'verdict.json')['state'],'passed')
        previous=self.root/'.ci/previous';shutil.copytree(report,previous)
        result,requests=self.run_wait(rows,previous=previous);self.assertEqual(result,0);self.assertFalse(requests)
        settings=previous/'settings.json';original=settings.read_bytes();value=G.read(settings)
        value['source_sha']='0'*40;G.S.write(settings,value)
        final_path=previous/'final.json';original_final=final_path.read_bytes();final=G.read(final_path)
        final['original_sha256']['settings.json']=G.S.digest(settings);G.S.write(final_path,final)
        result,_=self.run_wait(rows,previous=previous);self.assertEqual(result,1)
        self.assertIn('Stale',G.read(report/'final.json')['report_errors'][0])
        settings.write_bytes(original);final_path.write_bytes(original_final)
        settings=previous/'settings.json';settings.write_text(settings.read_text()+'\n')
        result,_=self.run_wait(rows,previous=previous);self.assertEqual(result,1)
        self.assertIn('corrupt',G.read(report/'final.json')['report_errors'][0])

    def test_pending_wait_continuation_fetches_fresh_terminal_inputs(self):
        rows=[self.status(name,i+1) for i,name in enumerate(self.contexts)]
        result,_=self.run_wait(rows[:-1],final=False);self.assertEqual(result,0)
        report=self.root/'.ci/pr-gates/wait';G.originals(report)
        self.assertEqual(G.read(report/'verdict.json')['state'],'pending')
        previous=self.root/'.ci/previous';shutil.copytree(report,previous)
        result,requests=self.run_wait(rows,previous=previous);self.assertEqual(result,0);self.assertTrue(requests)
        self.assertEqual(G.read(report/'verdict.json')['state'],'passed')
        self.assertGreater(len(G.read(report/'requests.json')),3)

    def test_original_failures_wrong_sha_and_never_started_dependencies_remain_failures(self):
        for rows,commit in [([],None),([self.status(self.contexts[0],1,'failure')],None),([], '0'*40)]:
            result,_=self.run_wait(rows,commit=commit);self.assertEqual(result,1)
            final=G.read(self.root/'.ci/pr-gates/wait/final.json')
            self.assertTrue(final['report_errors']);self.assertEqual(final['tests'],0)
            self.assertTrue(final['original_sha256'])


if __name__=='__main__':unittest.main()
