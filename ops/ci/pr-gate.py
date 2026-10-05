#!/usr/bin/env python3
"""Verify Circle's exact gate dependencies against native RWX terminal receipts."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys

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
    if manifest['version']!=2 or manifest['repository']!='ethereum-optimism/optimism':raise ValueError('Wrong native gate manifest')
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
        caller=yaml(row['native_config']);caller_tasks={task['key']:task for task in caller['tasks']}
        if len(caller_tasks)!=len(caller['tasks']):raise ValueError('Duplicate native coordinator task')
        custom=caller['on']['github']['push']['status-checks']['custom']
        if custom.count({'name':row['check_name'],'tasks':'gate-status'})!=1:
            raise ValueError('Missing genuine native aggregate status binding')
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
            if receipt[0].get('outputs')!={'filesystem':False,'artifacts':[{'key':'receipt','path':'.ci/pr-gates/groups/'+group}]}:
                raise ValueError('Native gate receipt artifact does not match its declared producer')
            if 'github' in native['on']:raise ValueError('Native coordinator must execute each workload only once')
            embedded=caller_tasks[definition['embedded_task']]
            if embedded['call']!='${{ run.dir }}/'+definition['config'].removeprefix('.rwx/'):
                raise ValueError('Native coordinator calls a different workload')
            for param in ('commit-sha','branch','tag'):
                if embedded['init'].get(param)!='${{ init.'+param+' }}':raise ValueError('Native coordinator changes workload provenance')
            if embedded['init'].get('cache-warm','false')!='false':raise ValueError('Native gate must execute fresh workloads')
            if custom.count({'name':definition['check_name'],'tasks':definition['embedded_task']})!=1:
                raise ValueError('Native coordinator changes the existing workload check')
        definitions=[manifest['groups'][group] for group in sorted(selected_groups)]
        after=terminal_expression([definition['embedded_task'] for definition in definitions])
        passed='${{ '+' && '.join('tasks.'+d['embedded_task']+'.tasks.'+d['receipt_task']+'.succeeded' for d in definitions)+' }}'
        failed='${{ '+' || '.join('(tasks.'+d['embedded_task']+'.tasks.'+d['receipt_task']+'.failed || tasks.'+d['embedded_task']+'.tasks.'+d['receipt_task']+'.skipped)' for d in definitions)+' }}'
        for key,condition in [('aggregate',passed),('gate-failure',failed)]:
            observer=caller_tasks[key]
            if observer['after']!=after or observer['if']!=condition or observer.get('cache') is not False:
                raise ValueError('Native aggregate does not follow actual terminal receipts')
            expected_run='python3 ops/ci/pr-gate.py aggregate '+name+(' --failed' if key=='gate-failure' else '')
            if observer['run']!=expected_run or observer['use']!=['code','tools']:
                raise ValueError('Native aggregate bypasses its original receipt validator')
            for group in sorted(selected_groups):
                definition=manifest['groups'][group];prefix='GROUP_'+group.upper().replace('-','_')
                task='tasks.'+definition['embedded_task']+'.tasks.'+definition['receipt_task']
                for attribute in ('succeeded','failed','skipped'):
                    if observer['env'].get(prefix+'_'+attribute.upper())!='${{ '+task+'.'+attribute+' }}':
                        raise ValueError('Native aggregate state is not engine-bound')
                if key=='aggregate' and observer['env'].get(prefix+'_REPORT')!='${{ '+task+'.artifacts.receipt }}':
                    raise ValueError('Native aggregate report is not engine-bound')
        verdict=caller_tasks['gate-status']
        if (verdict['after']!=terminal_expression(['aggregate','gate-failure'])
                or 'if' in verdict or verdict.get('cache') is not False
                or verdict['run']!='python3 ops/ci/pr-gate.py status '+name
                or verdict['use']!=['code','tools']):
            raise ValueError('Native gate status does not follow both terminal observers')
        for task in ('aggregate','gate-failure'):
            for attribute in ('succeeded','failed','skipped'):
                key='OBSERVER_'+task.upper().replace('-','_')+'_'+attribute.upper()
                if verdict['env'].get(key)!='${{ tasks.'+task+'.'+attribute+' }}':
                    raise ValueError('Native gate status is not engine-bound')
        if verdict.get('outputs')!={'filesystem':False,'artifacts':[{'key':'status','path':'.ci/pr-gates/status/'+name}]}:
            raise ValueError('Native gate status originals are not retained')
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


def aggregate(gate,failed_only=False):
    output=ROOT/'.ci/pr-gates/aggregate'/gate;shutil.rmtree(output,ignore_errors=True);output.mkdir(parents=True)
    status,errors,record=1,[],{}
    try:
        manifest,selections=configuration(gate);selection=selections[gate];settings=binding()
        route=read(Path(os.environ['GATE_ROUTING']))
        selected=route['c-'+selection['route'].replace('-','_')]
        if type(selected) is not bool:raise ValueError('Missing authoritative native aggregate routing')
        S.write(output/'settings.json',{**settings,'gate':gate,'selection':selection,'selected':selected})
        shutil.copyfile(MANIFEST,output/'manifest.json')
        states={}
        for group in selection['groups']:
            prefix='GROUP_'+group.upper().replace('-','_');states[group]={}
            for attribute in ('succeeded','failed','skipped'):
                value=os.environ[prefix+'_'+attribute.upper()]
                if value not in ('true','false'):raise ValueError('Missing native aggregate engine state')
                states[group][attribute]=value=='true'
            if sum(states[group].values())!=1:raise ValueError('Inconsistent native aggregate engine state')
        S.write(output/'states.json',states)
        record={'state':'failed','requires':selection['requires'],'groups':states,'dependencies':[]}
        if failed_only or any(not state['succeeded'] for state in states.values()):
            raise ValueError('Native gate receipt failed, was canceled or never ran')
        for group in selection['groups']:
            artifact=Path(os.environ['GROUP_'+group.upper().replace('-','_')+'_REPORT'])
            final=originals(artifact);source=read(artifact/'settings.json');actual=read(artifact/'states.json')
            if set(final['original_sha256'])!={'settings.json','states.json'}:
                raise ValueError('Missing or extra native receipt original')
            if (any(source[k]!=settings[k] for k in ('source_sha','branch','input_sha256','native_run_id'))
                    or source['task_attempt']!='1' or source['group']!=group
                    or source['definition']!=manifest['groups'][group] or type(source['selected']) is not bool or source['selected']!=selected):
                raise ValueError('Stale, foreign or differently selected native aggregate receipt')
            if set(actual)!=set(group_tasks(manifest,group)):
                raise ValueError('Missing or duplicate original native aggregate workload')
            for value in actual.values():
                expected={'succeeded':selected,'failed':False,'skipped':not selected}
                if value!=expected or any(type(v) is not bool for v in value.values()):
                    raise ValueError('Failed or skipped original native aggregate workload')
            target=output/'groups'/group;shutil.copytree(artifact,target)
            for row in selection['dependencies']:
                if row['group']==group:
                    record['dependencies'].append({'name':row['name'],'group':group,'tasks':row['tasks'],
                            'outcome':'passed' if selected else 'safe_skip'})
        record['dependencies'].sort(key=lambda row:selection['requires'].index(row['name']))
        if [row['name'] for row in record['dependencies']]!=selection['requires']:
            raise ValueError('Incomplete original native aggregate dependency assignment')
        if M.inputs()!=settings['input_sha256']:raise ValueError('Native aggregate changed source inputs')
        record['state']='passed' if selected else 'safe_skip';status=0
    except Exception as error:errors.append(str(error));print(error,file=sys.stderr)
    finally:
        S.write(output/'verdict.json',record)
        S.write(output/'final.json',{'exit_code':status,'report_errors':errors,'tests':0,'original_sha256':seal(output)})
    return status


def gate_status(gate):
    """Publish one executed verdict after mutually exclusive report observers."""
    output=ROOT/'.ci/pr-gates/status'/gate;shutil.rmtree(output,ignore_errors=True);output.mkdir(parents=True)
    status,errors=1,[]
    try:
        _,selections=configuration(gate);settings=binding()
        S.write(output/'settings.json',{**settings,'gate':gate,'selection':selections[gate]})
        states={}
        for task in ('aggregate','gate-failure'):
            states[task]={}
            for attribute in ('succeeded','failed','skipped'):
                value=os.environ['OBSERVER_'+task.upper().replace('-','_')+'_'+attribute.upper()]
                if value not in ('true','false'):raise ValueError('Missing native gate observer terminal state')
                states[task][attribute]=value=='true'
            if sum(states[task].values())!=1:raise ValueError('Inconsistent native gate observer terminal state')
        S.write(output/'states.json',states)
        if states!={'aggregate':{'succeeded':True,'failed':False,'skipped':False},
                   'gate-failure':{'succeeded':False,'failed':False,'skipped':True}}:
            raise ValueError('Native gate aggregate failed, was canceled, skipped or contradicted by its failure observer')
        status=0
    except Exception as error:errors.append(str(error));print(error,file=sys.stderr)
    finally:S.write(output/'final.json',{'exit_code':status,'report_errors':errors,'tests':0,'original_sha256':seal(output)})
    return status


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);sub=parser.add_subparsers(dest='mode',required=True)
    group=sub.add_parser('receipt');group.add_argument('group')
    gate=sub.add_parser('aggregate');gate.add_argument('gate');gate.add_argument('--failed',action='store_true')
    published=sub.add_parser('status');published.add_argument('gate')
    args=parser.parse_args()
    sys.exit(receipt(args.group) if args.mode=='receipt' else gate_status(args.gate) if args.mode=='status' else aggregate(args.gate,args.failed))
