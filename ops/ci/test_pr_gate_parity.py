"""Reject stale, bypassed and incomplete provider evidence using real Git/YQ."""
import copy
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch

import test_pr_gate as fixtures

SPEC = importlib.util.spec_from_file_location('compare_pr_gates', Path(__file__).with_name('compare-pr-gates.py'))
C = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(C)
G = fixtures.G


class GateParityTests(unittest.TestCase):
    reports = fixtures.GateTests.reports
    run_aggregate = fixtures.GateTests.run_aggregate
    reseal = fixtures.GateTests.reseal

    def setUp(self):
        fixtures.GateTests.setUp(self)
        handle = patch.object(C, 'ROOT', self.root)
        handle.start()
        self.addCleanup(handle.stop)
        env = self.reports()
        result, self.native, _ = self.run_aggregate(env)
        self.assertEqual(result, 0)
        for task, state in [('aggregate', 'succeeded'), ('gate-failure', 'skipped')]:
            for attribute in C.PASS:
                env['OBSERVER_' + task.upper().replace('-', '_') + '_' + attribute.upper()] = str(state == attribute).lower()
        with patch.dict(os.environ, env):
            self.assertEqual(G.gate_status('required-rust-ci'), 0)
        self.status = self.root / '.ci/pr-gates/status/required-rust-ci'
        self.circle = self.root / '.ci/provider-originals'
        self.circle.mkdir()
        self.row = self.manifest['gates']['required-rust-ci']
        self.names = self.selection['requires']
        self.write(self.circle / 'settings.json', {'provider': 'circleci', 'source_sha': self.sha,
                   'branch': self.env['CI_BRANCH'], 'gate': 'required-rust-ci', 'pipeline_id': 'pipeline',
                   'workflow_id': 'workflow', 'workflow_name': 'rust-ci', 'job_number': 100})
        self.write(self.circle / 'pipeline.json', {'id': 'pipeline', 'errors': [], 'vcs': {
                   'revision': self.sha, 'branch': self.env['CI_BRANCH']}})
        self.write(self.circle / 'workflow.json', {'id': 'workflow', 'pipeline_id': 'pipeline', 'name': 'rust-ci'})
        jobs = [{'id': name + '-id', 'name': name, 'status': 'success'} for name in self.names]
        jobs.append({'id': 'gate-id', 'name': 'required-rust-ci', 'status': 'success', 'job_number': 100,
                     'dependencies': [job['id'] for job in jobs]})
        self.write(self.circle / 'workflow-jobs.json', jobs)
        self.write(self.circle / 'api-pages/jobs-0.json', {'items': jobs, 'next_page_token': None})
        command = 'if [ "false" = "true" ]; then exit 0; fi\necho "All required jobs passed."'
        compiled = {'jobs': {'required-rust-ci': {'steps': [{'run': {'name': 'Verify all required jobs passed',
                     'command': command}}]}}, 'workflows': {'rust-ci': {'jobs': [{'required-rust-ci': {
                     'requires': self.names, 'upstream': {name: C.TERMINAL for name in self.names}}}]}}}
        self.write(self.circle / 'compiled.yml', compiled)
        source = (self.root / self.row['circle_config']).read_text()
        (self.circle / 'circle-config.yml').write_text(source)
        (self.circle / 'source.yml').write_text(source)
        (self.circle / 'source-gates.json').write_bytes((self.root / 'ops/ci/pr-gates.json').read_bytes())
        self.write(self.circle / 'pipeline-config.json', {'compiled': (self.circle / 'compiled.yml').read_text(), 'source': source})
        step = {'name': 'Verify all required jobs passed', 'actions': [{'status': 'success', 'output_url': 'https://fixture.invalid/log'}]}
        self.write(self.circle / 'gate-job.json', {'status': 'success', 'outcome': 'success', 'vcs_revision': self.sha,
                   'branch': self.env['CI_BRANCH'], 'build_num': 100, 'retry_of': None, 'retries': None,
                   'failed': False, 'timedout': False, 'canceled': False, 'infrastructure_fail': False,
                   'workflows': {'workflow_id': 'workflow', 'job_id': 'gate-id', 'job_name': 'required-rust-ci'}, 'steps': [step]})
        self.write(self.circle / 'step-index.json', [{'step': step['name'], 'status': 'success', 'log': 'step-logs/verifier.json'}])
        self.write(self.circle / 'step-logs/verifier.json', [{'message': 'Checking required jobs for required-rust-ci...\n' +
                   '\n'.join('  ok ' + name + ': success' for name in self.names) + '\nAll required jobs passed.\n',
                   'type': 'out', 'truncated': False}])
        self.write(self.circle / 'final.json', {'exit_code': 0, 'report_errors': [], 'tests': 0, 'original_sha256': {}})
        self.reseal(self.circle)
        def task(key, subtasks=None):
            return {'Key': key, 'ID': key + '-id', 'Status': {'Execution': 'finished', 'Result': 'succeeded',
                    'FinishedSubStatus': 'executed'}, 'Subtasks': subtasks or []}
        tasks = []
        for group in self.selection['groups']:
            definition = self.manifest['groups'][group]
            tasks.append(task(definition['embedded_task'], [task(key) for key in
                         G.group_tasks(self.manifest, group) + [definition['receipt_task']]]))
        tasks += [task(key) for key in ('aggregate', 'gate-failure', 'gate-status')]
        tasks[-2]['Status'] = {'Execution': 'skipped', 'Result': 'no_result', 'FinishedSubStatus': 'not_applicable'}
        self.run_path = self.root / '.ci/run.json'
        self.write(self.run_path, {'ID': self.env['RWX_RUN_ID'], 'RunID': self.env['RWX_RUN_ID'],
                   'CommitSha': self.sha, 'Branch': self.env['CI_BRANCH'], 'Trigger': 'github.push',
                   'Init': {'commit-sha': self.sha, 'branch': self.env['CI_BRANCH'], 'tag': ''},
                   'DefinitionPath': self.row['native_config'], 'TargetedTaskKeys': None, 'Tasks': tasks,
                   'ResultStatus': 'failed'})  # A Main-only failure must not change the Rust gate.
        self.github_path = self.root / '.ci/github.json'
        self.write(self.github_path, {'sha': self.sha, 'checks': [
                   {'name': 'RWX: ' + self.row['check_name'], 'state': 'success',
                    'link': 'https://cloud.rwx.com/optimism/runs/' + self.env['RWX_RUN_ID'] + '/latest/gate-status'},
                   {'name': 'ci/circleci: required-rust-ci', 'state': 'success',
                    'link': 'https://circleci.com/gh/ethereum-optimism/optimism/100'}]})

    def write(self, path, data):
        path.parent.mkdir(parents=True, exist_ok=True)
        G.S.write(path, data)

    def compare(self):
        return C.compare(self.circle, self.native, self.status, self.run_path, self.github_path)

    def test_complete_original_gate_and_main_only_failure(self):
        result = self.compare()
        self.assertEqual(result['state'], 'passed')
        self.assertEqual(result['dependencies'], self.names)
        self.assertEqual(result['tests'], 0)
        self.assertEqual(result['native']['workload_tasks'], 30)
        self.assertEqual(result['circle']['retry_history'], None)
        self.assertTrue(result['original_sha256']['circle'])

    def test_complete_contracts_gate_original_comparison_covers_every_prerequisite(self):
        gate='required-contracts-ci';row=self.manifest['gates'][gate]
        selection=G.configuration(gate)[1][gate];names=selection['requires']
        env=self.reports(gate=gate);result,self.native,_=self.run_aggregate(env,gate=gate);self.assertEqual(result,0)
        for alias,state in [('aggregate','succeeded'),('gate-failure','skipped')]:
            for attribute in C.PASS:env['OBSERVER_'+alias.upper().replace('-','_')+'_'+attribute.upper()]=str(state==attribute).lower()
        with patch.dict(os.environ,env):self.assertEqual(G.gate_status(gate),0)
        self.status=self.root/'.ci/pr-gates/status'/gate
        settings=G.read(self.circle/'settings.json');settings.update(gate=gate,workflow_name=row['workflow'])
        self.write(self.circle/'settings.json',settings)
        workflow=G.read(self.circle/'workflow.json');workflow['name']=row['workflow'];self.write(self.circle/'workflow.json',workflow)
        jobs=[{'id':name+'-id','name':name,'status':'success'} for name in names]
        jobs.append({'id':'gate-id','name':gate,'status':'success','job_number':100,'dependencies':[job['id'] for job in jobs]})
        self.write(self.circle/'workflow-jobs.json',jobs);self.write(self.circle/'api-pages/jobs-0.json',{'items':jobs,'next_page_token':None})
        command='if [ "false" = "true" ]; then exit 0; fi\necho "All required jobs passed."'
        self.write(self.circle/'compiled.yml',{'jobs':{gate:{'steps':[{'run':{'name':'Verify all required jobs passed','command':command}}]}},
            'workflows':{row['workflow']:{'jobs':[{gate:{'requires':names,'upstream':{name:C.TERMINAL for name in names}}}]}}})
        source=(self.root/row['circle_config']).read_text()
        (self.circle/'circle-config.yml').write_text(source);(self.circle/'source.yml').write_text(source)
        self.write(self.circle/'pipeline-config.json',{'compiled':(self.circle/'compiled.yml').read_text(),'source':source})
        job=G.read(self.circle/'gate-job.json');job['workflows']['job_name']=gate;self.write(self.circle/'gate-job.json',job)
        self.write(self.circle/'step-logs/verifier.json',[{'message':'Checking required jobs for '+gate+'...\n'+
            '\n'.join('  ok '+name+': success' for name in names)+'\nAll required jobs passed.\n','type':'out','truncated':False}])
        self.reseal(self.circle)
        def task(key,children=None):return {'Key':key,'ID':key+'-id','Status':{'Execution':'finished','Result':'succeeded','FinishedSubStatus':'executed'},'Subtasks':children or []}
        tasks=[]
        for group in selection['groups']:
            definition=self.manifest['groups'][group]
            tasks.append(task(definition['embedded_task'],[task(key) for key in G.group_tasks(self.manifest,group)+[definition['receipt_task']]]))
        tasks += [task(key) for key in row['observer_tasks'].values()]
        tasks[-2]['Status']={'Execution':'skipped','Result':'no_result','FinishedSubStatus':'not_applicable'}
        run=G.read(self.run_path);run['Tasks']=tasks;self.write(self.run_path,run)
        self.write(self.github_path,{'sha':self.sha,'checks':[
            {'name':'RWX: '+row['check_name'],'state':'success','link':'https://cloud.rwx.com/optimism/runs/'+self.env['RWX_RUN_ID']+'/latest/contracts-gate-status'},
            {'name':'ci/circleci: '+gate,'state':'success','link':'https://circleci.com/gh/ethereum-optimism/optimism/100'}]})
        result=self.compare();self.assertEqual(result['state'],'passed');self.assertEqual(result['native']['workload_tasks'],21)
        self.assertEqual(result['dependencies'],names)

    def test_resealed_revision_selection_retry_and_full_source_inputs(self):
        path = self.native / 'settings.json'
        original = G.read(path)
        changed_inputs = dict(original['input_sha256'])
        changed_inputs['ops/ci/pr-gates.json'] = 'f' * 64
        for field, value in [('source_sha', 'b' * 40), ('selected', False), ('selected', 1),
                             ('task_attempt', '2'), ('input_sha256', {}), ('input_sha256', changed_inputs)]:
            data = copy.deepcopy(original)
            data[field] = value
            self.write(path, data)
            self.reseal(self.native)
            with self.subTest(field=field, value=value), self.assertRaises(ValueError):
                self.compare()
        self.write(path, original)
        self.reseal(self.native)
        self.assertEqual(self.compare()['state'], 'passed')

    def test_missing_corrupt_extra_and_incorrectly_typed_originals(self):
        path = self.status / 'states.json'
        original = path.read_bytes()
        path.write_bytes(original + b' ')
        with self.assertRaises(ValueError):
            self.compare()
        path.unlink()
        with self.assertRaises((ValueError, FileNotFoundError)):
            self.compare()
        self.write(path, {'aggregate': {'succeeded': 1, 'failed': 0, 'skipped': 0}, 'gate-failure': C.SKIP})
        self.reseal(self.status)
        with self.assertRaisesRegex(ValueError, 'typed'):
            self.compare()
        path.write_bytes(original)
        self.reseal(self.status)
        (self.circle / 'extra.json').write_text('{}')
        with self.assertRaises(ValueError):
            self.compare()

    def test_circle_missing_failed_skipped_duplicate_or_incomplete_dependencies(self):
        jobs = G.read(self.circle / 'workflow-jobs.json')
        variants = []
        for status in ('failed', 'skipped', 'running'):
            data = copy.deepcopy(jobs)
            data[0]['status'] = status
            variants.append(data)
        variants += [jobs + [jobs[0]], jobs[1:]]
        data = copy.deepcopy(jobs)
        data[-1]['dependencies'] = data[-1]['dependencies'][:-1]
        variants.append(data)
        for data in variants:
            self.write(self.circle / 'workflow-jobs.json', data)
            self.write(self.circle / 'api-pages/jobs-0.json', {'items': data, 'next_page_token': None})
            self.reseal(self.circle)
            with self.subTest(jobs=data), self.assertRaises((ValueError, KeyError)):
                self.compare()

    def test_bypassed_duplicate_truncated_and_incomplete_circle_verifier(self):
        path = self.circle / 'step-logs/verifier.json'
        events = G.read(path)
        for message in ('Skipping ci-gate: always-succeed is true.\n', events[0]['message'] +
                        '  ok ' + self.names[0] + ': success\n', events[0]['message'].replace(self.names[0], 'foreign-job')):
            self.write(path, [events[0] | {'message': message}])
            self.reseal(self.circle)
            with self.assertRaises(ValueError):
                self.compare()
        self.write(path, [events[0] | {'truncated': True}])
        self.reseal(self.circle)
        with self.assertRaises(ValueError):
            self.compare()
        self.write(path, events)
        compiled = G.read(self.circle / 'compiled.yml')
        compiled['jobs']['required-rust-ci']['steps'][0]['run']['command'] = 'if [ "true" = "true" ]; then exit 0; fi'
        self.write(self.circle / 'compiled.yml', compiled)
        raw = G.read(self.circle / 'pipeline-config.json')
        self.write(self.circle / 'pipeline-config.json', raw | {'compiled': (self.circle / 'compiled.yml').read_text()})
        self.reseal(self.circle)
        with self.assertRaises(ValueError):
            self.compare()

    def test_native_cache_skip_failure_targeted_run_and_stale_github_status(self):
        original = G.read(self.run_path)
        for field, value in [('FinishedSubStatus', 'cached'), ('Result', 'failed'), ('Execution', 'skipped')]:
            data = copy.deepcopy(original)
            data['Tasks'][0]['Subtasks'][0]['Status'][field] = value
            self.write(self.run_path, data)
            with self.subTest(field=field), self.assertRaises(ValueError):
                self.compare()
        self.write(self.run_path, original | {'TargetedTaskKeys': ['aggregate']})
        with self.assertRaises(ValueError):
            self.compare()
        self.write(self.run_path, original)
        github = G.read(self.github_path)
        for field, value in [('state', 'pending'), ('link', 'https://cloud.rwx.com/optimism/runs/' + 'b' * 32 + '/latest/gate-status')]:
            data = copy.deepcopy(github)
            data['checks'][0][field] = value
            self.write(self.github_path, data)
            with self.subTest(field=field), self.assertRaises(ValueError):
                self.compare()

    def test_complete_git_source_includes_link_modes_and_ignores_new_checkout_edits(self):
        (self.root / 'target.txt').write_text('committed input\n')
        (self.root / 'fixture-link').symlink_to('target.txt')
        subprocess.run(['git', 'add', 'target.txt', 'fixture-link'], cwd=self.root, check=True)
        subprocess.run(['git', 'update-index', '--add', '--cacheinfo', '160000,' + self.sha + ',fixture-submodule'], cwd=self.root, check=True)
        subprocess.run(['git', '-c', 'user.name=CI fixture', '-c', 'user.email=ci-fixture@example.invalid',
                        'commit', '-qm', 'Actual source link modes'], cwd=self.root, check=True)
        sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=self.root, text=True).strip()
        inputs = C.source_inputs(sha)
        self.assertEqual(inputs['fixture-link'], {'symlink': 'target.txt'})
        self.assertEqual(inputs['fixture-submodule'], {'gitlink': self.sha})
        self.assertEqual(inputs['target.txt'], G.S.digest(self.root / 'target.txt'))
        (self.root / 'target.txt').write_text('new unrelated checkout edit\n')
        self.assertEqual(C.source_inputs(sha), inputs)
        self.assertEqual(self.compare()['state'], 'passed')


if __name__ == '__main__':
    unittest.main()
