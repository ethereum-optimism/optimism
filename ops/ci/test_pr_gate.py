"""Exercise real Git/YQ authority and sealed native terminal receipt aggregation."""
import copy
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
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
        names|={row['native_config'] for row in manifest['gates'].values()}
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
        route=self.root/'.ci/routing.json';route.parent.mkdir(exist_ok=True)
        route.write_text(json.dumps({'c-run_rust_ci':True}));self.env['GATE_ROUTING']=str(route)

    def reports(self,selected=True):
        env={**self.env}
        for group in self.selection['groups']:
            original={**self.env,'GROUP_SELECTED':'true' if selected else 'false'}
            for task in G.group_tasks(self.manifest,group):
                for attribute in ('succeeded','failed','skipped'):
                    value=(attribute=='succeeded' and selected) or (attribute=='skipped' and not selected)
                    original['TASK_'+task.upper().replace('-','_')+'_'+attribute.upper()]=str(value).lower()
            with patch.dict(os.environ,original):self.assertEqual(G.receipt(group),0)
            env['GROUP_'+group.upper().replace('-','_')+'_REPORT']=str(self.root/'.ci/pr-gates/groups'/group)
            for attribute in ('succeeded','failed','skipped'):
                env['GROUP_'+group.upper().replace('-','_')+'_'+attribute.upper()]='true' if attribute=='succeeded' else 'false'
        return env

    def run_aggregate(self,env,failed=False):
        with patch.dict(os.environ,env):result=G.aggregate('required-rust-ci',failed)
        output=self.root/'.ci/pr-gates/aggregate/required-rust-ci'
        return result,output,G.read(output/'verdict.json')

    def reseal(self,artifact):
        final=G.read(artifact/'final.json')
        final['original_sha256']={k:v for k,v in G.seal(artifact).items() if k!='final.json'}
        G.S.write(artifact/'final.json',final)

    def test_exact_circle_authority_and_every_feature_partition(self):
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

    def test_native_coordinator_rejects_duplicate_execution_stale_revision_and_noop_gates(self):
        path=self.root/'.rwx/rust-gate.yml';source=path.read_text()
        for old,new in [('      commit-sha: ${{ init.commit-sha }}','      commit-sha: foreign-sha'),
                        ('python3 ops/ci/pr-gate.py aggregate required-rust-ci','true'),
                        ('.rust-gate-receipt.artifacts.receipt','.rust-gate-receipt.artifacts.report'),
                        ('tasks.native-fmt.tasks.rust-gate-receipt.succeeded','tasks.native-fmt.succeeded')]:
            # Match the embedded call rather than the route's environment.
            path.write_text(source.replace(old,new,1))
            with self.assertRaises(ValueError):G.configuration('required-rust-ci')
        path.write_text(source)
        child=self.root/'.rwx/rust.yml'
        child.write_text(child.read_text().replace('on:\n','on:\n  github: {}\n',1))
        with self.assertRaisesRegex(ValueError,'only once'):G.configuration('required-rust-ci')

    def test_missing_renamed_or_wrong_receipt_artifact_cannot_pass_configuration(self):
        child=self.root/'.rwx/pilot.yml';source=child.read_text()
        for old,new in [('key: receipt','key: report'),('path: .ci/pr-gates/groups/rust-fmt','path: unrelated/report')]:
            child.write_text(source.replace(old,new))
            with self.subTest(new=new),self.assertRaisesRegex(ValueError,'artifact'):G.configuration('required-rust-ci')

    def test_real_receipts_reject_failed_selected_skips_and_invalid_safe_skips(self):
        group='rust-fmt';output=self.root/'.ci/pr-gates/groups'/group
        for selected,state,expected in [('true','succeeded',0),('true','failed',1),('true','skipped',1),('false','skipped',0),('false','succeeded',1)]:
            env={**self.env,'GROUP_SELECTED':selected}
            for attribute in ('succeeded','failed','skipped'):env['TASK_RUST_FMT_'+attribute.upper()]='true' if state==attribute else 'false'
            with patch.dict(os.environ,env):self.assertEqual(G.receipt(group),expected)
            final=G.read(output/'final.json');self.assertEqual(final['exit_code'],expected)
            self.assertEqual(final['tests'],0);self.assertTrue(final['original_sha256'])
            self.assertEqual(G.read(output/'states.json')['rust-fmt'][state],True)

    def test_complete_real_receipts_cover_every_original_gate_dependency_once(self):
        env=self.reports();result,output,record=self.run_aggregate(env)
        self.assertEqual(result,0);self.assertEqual(record['state'],'passed')
        self.assertEqual([row['name'] for row in record['dependencies']],self.selection['requires'])
        self.assertTrue(all(row['outcome']=='passed' for row in record['dependencies']))
        self.assertEqual(G.originals(output)['tests'],0)
        for group in self.selection['groups']:
            self.assertEqual(G.read(output/'groups'/group/'states.json'),G.read(Path(env['GROUP_'+group.upper().replace('-','_')+'_REPORT'])/'states.json'))

    def test_resealed_stale_sha_branch_run_inputs_and_selection_cannot_pass(self):
        env=self.reports();artifact=Path(env['GROUP_RUST_FMT_REPORT']);original=G.read(artifact/'settings.json')
        for field,value in [('source_sha','b'*40),('branch','foreign-branch'),('native_run_id','b'*32),('input_sha256',{}),('selected',False),('selected',1),('definition',{}),('task_attempt','2')]:
            changed=copy.deepcopy(original);changed[field]=value;G.S.write(artifact/'settings.json',changed);self.reseal(artifact)
            result,output,_=self.run_aggregate(env);self.assertEqual(result,1)
            self.assertTrue(G.read(output/'final.json')['report_errors'])
        G.S.write(artifact/'settings.json',original);self.reseal(artifact)
        self.assertEqual(self.run_aggregate(env)[0],0)

    def test_corrupt_missing_duplicate_and_extra_originals_cannot_pass(self):
        env=self.reports();artifact=Path(env['GROUP_RUST_FMT_REPORT']);states=(artifact/'states.json').read_bytes()
        (artifact/'states.json').write_bytes(states+b' ');self.assertEqual(self.run_aggregate(env)[0],1)
        (artifact/'states.json').unlink();self.assertEqual(self.run_aggregate(env)[0],1)
        (artifact/'states.json').write_bytes(b'{"rust-fmt":{},"rust-fmt":{}}');self.reseal(artifact)
        self.assertEqual(self.run_aggregate(env)[0],1)
        G.S.write(artifact/'states.json',{'rust-fmt':{'succeeded':1,'failed':0,'skipped':0}});self.reseal(artifact)
        self.assertEqual(self.run_aggregate(env)[0],1)
        (artifact/'states.json').write_bytes(states);(artifact/'extra.json').write_text('{}');self.reseal(artifact)
        self.assertEqual(self.run_aggregate(env)[0],1)

    def test_failed_canceled_and_never_started_native_receipts_retain_failure_evidence(self):
        env=self.reports()
        for state in ('failed','skipped'):
            changed=dict(env)
            for attribute in ('succeeded','failed','skipped'):
                changed['GROUP_RUST_FMT_'+attribute.upper()]='true' if attribute==state else 'false'
            # Failure collection does not need an artifact from a receipt that
            # never ran; engine states remain available after terminal runs.
            changed.pop('GROUP_RUST_FMT_REPORT')
            result,output,record=self.run_aggregate(changed,failed=True)
            self.assertEqual(result,1);self.assertEqual(record['state'],'failed')
            self.assertTrue(G.read(output/'states.json')['rust-fmt'][state])
            self.assertEqual(G.read(output/'final.json')['tests'],0)
            self.assertTrue(G.read(output/'final.json')['original_sha256'])

    def test_unselected_workloads_require_real_safe_skip_receipts_and_zero_tests(self):
        env=self.reports(False);Path(env['GATE_ROUTING']).write_text(json.dumps({'c-run_rust_ci':False}))
        result,output,record=self.run_aggregate(env)
        self.assertEqual(result,0);self.assertEqual(record['state'],'safe_skip')
        self.assertTrue(all(row['outcome']=='safe_skip' for row in record['dependencies']))
        self.assertEqual(G.originals(output)['tests'],0)
        Path(env['GATE_ROUTING']).write_text(json.dumps({'c-run_rust_ci':True}))
        self.assertEqual(self.run_aggregate(env)[0],1)

    def test_single_executed_status_preserves_actual_terminal_observer_results(self):
        output=self.root/'.ci/pr-gates/status/required-rust-ci'
        for aggregate in ('succeeded','failed','skipped'):
            for failure in ('succeeded','failed','skipped'):
                env={**self.env}
                for task,state in [('aggregate',aggregate),('gate-failure',failure)]:
                    for attribute in ('succeeded','failed','skipped'):
                        env['OBSERVER_'+task.upper().replace('-','_')+'_'+attribute.upper()]=str(state==attribute).lower()
                expected=0 if (aggregate,failure)==('succeeded','skipped') else 1
                with self.subTest(aggregate=aggregate,failure=failure),patch.dict(os.environ,env):
                    self.assertEqual(G.gate_status('required-rust-ci'),expected)
                final=G.read(output/'final.json')
                self.assertEqual(final['exit_code'],expected);self.assertEqual(final['tests'],0)
                self.assertTrue(final['original_sha256']);self.assertTrue(G.read(output/'states.json')['aggregate'][aggregate])
        for state in ('TRUE','', 'pending'):
            env['OBSERVER_AGGREGATE_SUCCEEDED']=state
            with patch.dict(os.environ,env):self.assertEqual(G.gate_status('required-rust-ci'),1)

    def test_status_configuration_rejects_skipped_custom_checks_and_unbound_verdicts(self):
        path=self.root/'.rwx/rust-gate.yml';source=path.read_text()
        for old,new in [('tasks: gate-status','tasks: [aggregate, gate-failure]'),
                        ('python3 ops/ci/pr-gate.py status required-rust-ci','true'),
                        ('OBSERVER_AGGREGATE_SUCCEEDED: ${{ tasks.aggregate.succeeded }}',
                         'OBSERVER_AGGREGATE_SUCCEEDED: true'),
                        ('path: .ci/pr-gates/status/required-rust-ci','path: unrelated/status')]:
            path.write_text(source.replace(old,new,1))
            with self.subTest(new=new),self.assertRaises(ValueError):G.configuration('required-rust-ci')


if __name__=='__main__':unittest.main()
