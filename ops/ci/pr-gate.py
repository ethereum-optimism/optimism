#!/usr/bin/env python3
"""Verify Circle's exact gate dependencies against native RWX terminal receipts."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
MANIFEST = ROOT/'ops/ci/pr-gates.json'


def helper(name):
    spec=importlib.util.spec_from_file_location(name,Path(__file__).with_name(name+'.py'))
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module);return module


S=helper('rust-workspace-report');M=helper('main-checks')
def read(path):return json.loads(path.read_text(),object_pairs_hook=unique)
def unique(pairs):
    result={}
    for key,value in pairs:
        if key in result:raise ValueError('Duplicate original gate key: '+key)
        result[key]=value
    return result
def seal(path):return {str(p.relative_to(path)):S.digest(p) for p in sorted(path.rglob('*')) if p.is_file()}
def yaml(path):return json.loads(subprocess.check_output(['yq','-o=json','.',str(ROOT/path)],text=True),object_pairs_hook=unique)


def originals(path):
    final=read(path/'final.json')
    if final['exit_code'] or final['report_errors'] or final['tests']!=0 or not final['original_sha256']:
        raise ValueError('Failed or incomplete native gate originals')
    if any(p.is_symlink() for p in path.rglob('*')):raise ValueError('Unsafe native gate original link')
    for name,digest in final['original_sha256'].items():
        if Path(name).is_absolute() or '..' in Path(name).parts or not re.fullmatch('[0-9a-f]{64}',digest):
            raise ValueError('Unsafe native gate original manifest')
        if not (path/name).is_file() or S.digest(path/name)!=digest:raise ValueError('Missing or corrupt native gate original')
    if seal(path)!=final['original_sha256']|{'final.json':S.digest(path/'final.json')}:
        raise ValueError('Extra or unsealed native gate original')
    return final


def group_tasks(manifest,group):
    tasks=[]
    for gate in manifest['gates'].values():
        tasks.extend(task for row in gate['dependencies'] if row['group']==group for task in row['tasks'])
    if not tasks or len(tasks)!=len(set(tasks)):raise ValueError('Missing or duplicate native gate task assignment')
    return sorted(tasks)


def terminal_expression(tasks):
    return '${{ '+' && '.join('('+task+'.succeeded || '+task+'.failed || '+task+'.skipped)' for task in tasks)+' }}'


def configuration(gate=None):
    manifest=read(MANIFEST)
    if manifest['version']!=1 or manifest['repository']!='ethereum-optimism/optimism':raise ValueError('Wrong native gate manifest')
    selections={}
    for name,row in manifest['gates'].items():
        if gate is not None and name!=gate:continue
        circle=yaml(row['circle_config']);jobs=circle['workflows'][row['workflow']]['jobs']
        matches=[job[name] for job in jobs if isinstance(job,dict) and name in job]
        if len(matches)!=1:raise ValueError('Missing or duplicate authoritative Circle gate')
        requires=matches[0]['requires'];dependencies=row['dependencies']
        names=[d['name'] for d in dependencies]
        if (not requires or any(not isinstance(item,dict) or len(item)!=1 or next(iter(item.values()))!='terminal' for item in requires)
                or len(names)!=len(set(names)) or names!=[next(iter(item)) for item in requires]
                or matches[0].get('always-succeed',False)):
            raise ValueError('Missing, duplicate, renamed or changed Circle gate dependency')
        selected_groups={d['group'] for d in dependencies}
        for group in selected_groups:
            definition=manifest['groups'][group];native=yaml(definition['config'])
            tasks=group_tasks(manifest,group);keys=[task['key'] for task in native['tasks']]
            if len(keys)!=len(set(keys)) or not set(tasks)<=set(keys):raise ValueError('Missing or duplicate native gate workload')
            receipt=[task for task in native['tasks'] if task['key']==definition['receipt_task']]
            if len(receipt)!=1 or receipt[0]['after']!=terminal_expression(tasks):raise ValueError('Native gate receipt does not wait for every terminal prerequisite')
            env=receipt[0]['env']
            for task in tasks:
                for attribute in ('succeeded','failed','skipped'):
                    key='TASK_'+task.upper().replace('-','_')+'_'+attribute.upper()
                    if env.get(key)!='${{ tasks.'+task+'.'+attribute+' }}':raise ValueError('Native gate task state is not engine-bound')
            if receipt[0].get('cache') is not False:raise ValueError('Native gate receipt must execute freshly')
            statuses=native['on']['github']['push']['status-checks']['custom']
            expected={'name':definition['status'].removeprefix('RWX: '),'tasks':definition['receipt_task']}
            if statuses.count(expected)!=1:raise ValueError('Missing or duplicate native terminal status binding')
        selections[name]={'requires':names,'groups':sorted(selected_groups),**row}
    if not selections:raise ValueError('Unknown or empty native gate selection')
    return manifest,selections


def binding():
    sha=subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip()
    if sha!=os.environ['CI_COMMIT_SHA'] or not re.fullmatch('[0-9a-f]{40}',sha) or subprocess.call(['git','diff','--quiet','HEAD'],cwd=ROOT):
        raise ValueError('Stale or changed native gate source')
    if (not os.environ['CI_BRANCH'] or not re.fullmatch('[0-9a-f]{32}',os.environ['RWX_RUN_ID'])
            or os.environ['RWX_TASK_ATTEMPT_NUMBER']!='1'):
        raise ValueError('Missing native gate identity or uninvestigated retry')
    return {'source_sha':sha,'branch':os.environ['CI_BRANCH'],'input_sha256':M.inputs(),
            'native_run_id':os.environ['RWX_RUN_ID'],'task_attempt':os.environ['RWX_TASK_ATTEMPT_NUMBER']}


def receipt(group):
    output=ROOT/'.ci/pr-gates/groups'/group;shutil.rmtree(output,ignore_errors=True);output.mkdir(parents=True)
    status,errors=1,[]
    try:
        manifest,_=configuration();settings=binding();definition=manifest['groups'][group]
        selected=os.environ['GROUP_SELECTED']
        if selected not in ('true','false'):raise ValueError('Missing authoritative native gate routing')
        states={}
        for task in group_tasks(manifest,group):
            states[task]={}
            for attribute in ('succeeded','failed','skipped'):
                value=os.environ['TASK_'+task.upper().replace('-','_')+'_'+attribute.upper()]
                if value not in ('true','false'):raise ValueError('Missing native gate terminal task state')
                states[task][attribute]=value=='true'
            if sum(states[task].values())!=1:raise ValueError('Inconsistent native gate terminal task state')
        S.write(output/'settings.json',{**settings,'group':group,'definition':definition,'selected':selected=='true'})
        S.write(output/'states.json',states)
        if selected=='true' and any(not row['succeeded'] for row in states.values()):raise ValueError('Failed, canceled or not-run native gate prerequisite')
        if selected=='false' and any(not row['skipped'] for row in states.values()):raise ValueError('Native safe skip executed or failed a selected workload')
        status=0
    except Exception as error:errors.append(str(error));print(error,file=sys.stderr)
    finally:S.write(output/'final.json',{'exit_code':status,'report_errors':errors,'tests':0,'original_sha256':seal(output)})
    return status


class Client:
    def __init__(self,directory,token,base='https://api.github.com'):
        self.directory=directory;self.token=token;self.base=base;self.requests=[]

    def get(self,path):
        url=self.base+path
        headers={'Accept':'application/vnd.github+json','X-GitHub-Api-Version':'2022-11-28','User-Agent':'optimism-rwx-native-gate'}
        if self.token:headers['Authorization']='Bearer '+self.token
        for attempt in range(5):
            try:
                with urllib.request.urlopen(urllib.request.Request(url,headers=headers),timeout=30) as response:
                    payload=response.read();code=response.status
                break
            except urllib.error.HTTPError as error:
                payload=error.read();code=error.code
                self.retain(url,payload,code)
                if code not in (429,500,502,503,504) or attempt==4:raise ValueError('Gate status API failed with HTTP '+str(code)) from None
                time.sleep(min(2**attempt,16))
        self.retain(url,payload,code)
        return json.loads(payload,object_pairs_hook=unique)

    def retain(self,url,payload,code):
        digest=hashlib.sha256(payload).hexdigest();path=self.directory/'responses'/digest
        path.parent.mkdir(parents=True,exist_ok=True);path.write_bytes(payload)
        self.requests.append({'url':url,'http_status':code,'response_sha256':digest,'observed_at':time.time()})

    def statuses(self,repository,sha):
        prefix='/repos/'+repository+'/commits/'+sha+'/statuses?per_page=100&page='
        for attempt in range(5):
            result=[];first=None
            for page in range(1,101):
                rows=self.get(prefix+str(page))
                if not isinstance(rows,list):raise ValueError('Invalid original commit-status page')
                if page==1:first=rows
                result.extend(rows)
                if len(rows)<100:break
            else:raise ValueError('Incomplete native gate commit-status pagination')
            # New statuses can shift page boundaries during collection. Retain
            # those originals, then fetch a complete consistent view again.
            ids=[row['id'] for row in result]
            if len(ids)==len(set(ids)) and self.get(prefix+'1')==first:return result
            if attempt<4:time.sleep(.25)
        raise ValueError('Native gate commit statuses changed during every complete snapshot')


def verdict(rows,manifest,selection):
    contexts={manifest['groups'][group]['status']:manifest['groups'][group] for group in selection['groups']};latest={};ids=set()
    for row in rows:
        if type(row.get('id')) is not int or row['id'] in ids:raise ValueError('Duplicate or invalid original commit status')
        ids.add(row['id'])
        if row['context'] in contexts and (row['context'] not in latest or row['id']>latest[row['context']]['id']):latest[row['context']]=row
    for name,row in latest.items():
        expected_url=r'https://cloud\.rwx\.com/optimism/runs/[0-9a-f]{32}/latest/'+re.escape(contexts[name]['receipt_task'])+r'\?external_source=github'
        if (any(row.get('creator',{}).get(k)!=v for k,v in manifest['trusted_actor'].items())
                or not re.fullmatch(expected_url,row.get('target_url') or '')
                or row['state'] not in ('pending','success','failure','error')):
            raise ValueError('Untrusted or malformed native gate prerequisite: '+name)
    if any(row['state'] in ('failure','error') for row in latest.values()):return 'failed',latest
    if set(latest)==set(contexts) and all(row['state']=='success' for row in latest.values()):return 'passed',latest
    return 'pending',latest


def wait(gate,seconds,previous=None,final_chunk=False,poll=15):
    output=ROOT/'.ci/pr-gates/wait';shutil.rmtree(output,ignore_errors=True)
    output.mkdir(parents=True)
    status,errors=1,[];client=None;record={}
    try:
        if previous:
            originals(previous);shutil.copytree(previous,output,dirs_exist_ok=True)
        manifest,selections=configuration(gate);selection=selections[gate];settings=binding()
        if settings['branch'] not in ('codex/rwx-ci-pilot','develop'):
            raise ValueError('Native gate API inputs are restricted to pilot/develop')
        route=read(Path(os.environ['GATE_ROUTING']))
        selected=route['c-'+selection['route'].replace('-','_')]
        if type(selected) is not bool:raise ValueError('Missing authoritative gate routing')
        if previous:
            old=read(output/'settings.json')
            if any(old[k]!=settings[k] for k in ('source_sha','branch','input_sha256','native_run_id')):raise ValueError('Stale native gate wait continuation')
            record=read(output/'verdict.json')
        S.write(output/'settings.json',{**settings,'gate':gate,'selection':selection,'selected':selected})
        shutil.copyfile(MANIFEST,output/'manifest.json')
        if not selected:record={'state':'safe_skip','requires':selection['requires'],'observed':{}}
        if record.get('state') not in ('passed','safe_skip'):
            client=Client(output,os.environ.get('GATE_GITHUB_TOKEN'))
            repository=manifest['repository'];sha=settings['source_sha']
            commit=client.get('/repos/'+repository+'/commits/'+sha)
            if commit['sha']!=sha:raise ValueError('Wrong commit returned by gate status API')
            deadline=time.monotonic()+seconds
            while True:
                state,observed=verdict(client.statuses(repository,sha),manifest,selection)
                record={'state':state,'requires':selection['requires'],'observed':observed}
                S.write(output/'verdict.json',record)
                if state=='failed':raise ValueError('Native gate prerequisite failed')
                if state=='passed':break
                if time.monotonic()>=deadline:
                    if final_chunk:raise ValueError('Native gate prerequisites missing or not terminal before timeout')
                    break
                time.sleep(min(poll,max(0,deadline-time.monotonic())))
        if M.inputs()!=settings['input_sha256']:raise ValueError('Native gate changed revision inputs')
        status=0
    except Exception as error:errors.append(str(error));print(error,file=sys.stderr)
    finally:
        if client:
            history=read(output/'requests.json') if (output/'requests.json').exists() else []
            S.write(output/'requests.json',history+client.requests)
        S.write(output/'verdict.json',record)
        (output/'final.json').unlink(missing_ok=True)
        S.write(output/'final.json',{'exit_code':status,'report_errors':errors,'tests':0,'original_sha256':seal(output)})
    return status


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);sub=parser.add_subparsers(dest='mode',required=True)
    group=sub.add_parser('receipt');group.add_argument('group')
    gate=sub.add_parser('wait');gate.add_argument('gate');gate.add_argument('--seconds',type=int,default=2400)
    gate.add_argument('--previous',type=Path);gate.add_argument('--final',action='store_true')
    args=parser.parse_args()
    sys.exit(receipt(args.group) if args.mode=='receipt' else wait(args.gate,args.seconds,args.previous,args.final))
