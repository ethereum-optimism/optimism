#!/usr/bin/env python3
"""Verify complete original L2 fork executions and hosted same-SHA parity."""
import argparse
import hashlib
import json
import math
from pathlib import Path
import re
import subprocess
import tempfile

import importlib.util

def helper(name):
    spec=importlib.util.spec_from_file_location(name,Path(__file__).with_name(name+'.py'))
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module);return module

L=helper('contract-l2-fork');G=helper('compare-pr-gates')
read,check=L.read,L.check


def authority(settings):
    sha=settings['source_sha'];check(re.fullmatch('[0-9a-f]{40}',sha),'Invalid L2 source revision')
    expected=G.source_inputs(sha);inputs=settings['inputs']
    for entry in subprocess.check_output(['git','ls-tree','-rz',sha],cwd=G.ROOT).split(b'\0'):
        if not entry:continue
        header,name=entry.split(b'\t',1);mode,kind,oid=header.decode().split();name=name.decode();row=inputs[name]
        check(row['mode']==mode,'Changed L2 source mode')
        if kind=='commit':
            check(row=={'mode':mode,'gitlink':oid,'checkout':'initialized'}
                  and any(key.startswith(name+'/') for key in inputs),'Missing initialized L2 submodule')
        else:
            wanted=hashlib.sha256(expected[name]['symlink'].encode()).hexdigest() if mode=='120000' else expected[name]
            check(row=={'mode':mode,'sha256':wanted},'Changed committed L2 source')
    modules=[name for name,row in inputs.items() if row.get('mode')=='160000']
    check(set(inputs)-set(expected)=={name for name in inputs if any(name.startswith(m+'/') for m in modules)},
          'Foreign L2 source outside initialized submodules')
    check(settings['implementation']=={name:hashlib.sha256(G.source('ops/ci/'+name,sha)).hexdigest()
          for name in L.IMPLEMENTATION},'Uncommitted L2 implementation')


def positive(value):return type(value) in (int,float) and math.isfinite(value) and value>0


def rpc(directory,expected):
    """Bind all original request/response IDs before comparing result frames."""
    requests=sorted(directory.glob('rpc-*-request.json'),key=lambda p:int(p.name.split('-')[1]))
    check([p.name for p in requests]==['rpc-'+str(i)+'-request.json' for i in range(len(expected))],
          'Missing or extra original L2 RPC request')
    wanted_files=set();results=[]
    for i,(path,(method,params)) in enumerate(zip(requests,expected)):
        request=read(path)
        check(request=={'jsonrpc':'2.0','id':i+1,'method':method,'params':params},'Changed original L2 RPC request')
        wanted_files.add(path.name);stem='rpc-'+str(i)
        attempts=sorted(directory.glob(stem+'-attempt-*.metadata.json'),key=lambda p:int(p.name.split('-')[3].split('.')[0]))
        check(1<=len(attempts)<=3 and [p.name for p in attempts]==[stem+'-attempt-'+str(n)+'.metadata.json' for n in range(1,len(attempts)+1)],
              'Missing or excessive original L2 RPC attempt')
        history=[]
        for n,metadata in enumerate(attempts,1):
            body=directory/(stem+'-attempt-'+str(n)+'.json');wanted_files|={body.name,metadata.name}
            row=read(metadata)
            check(set(row)=={'url','http_status','error','started_at','elapsed_seconds','response_sha256'}
                  and row['url']==L.RPC and row['response_sha256']==L.digest(body)
                  and positive(row['started_at']) and positive(row['elapsed_seconds']), 'Changed original L2 RPC frame')
            if n==len(attempts):
                value=read(body)
                check(row['http_status']==200 and row['error'] is None and value.get('jsonrpc')=='2.0'
                      and type(value.get('id')) is int and value['id']==i+1 and 'result' in value and 'error' not in value,
                      'Failed or mismatched original L2 RPC response')
                response={key:item for key,item in value.items() if key!='id'}
            else:
                check(row['http_status'] not in (200,400,401,403,404),'Invalid L2 retry after success or permanent denial')
                response={'original_body_hex':body.read_bytes().hex()}
            history.append({'http_status':row['http_status'],'error':row['error'],'response':response})
        results.append({'method':method,'params':params,'attempts':history})
    check({p.name for p in directory.glob('rpc-*')}==wanted_files,'Extra original L2 RPC attempt or frame')
    return results


def preflight(directory,sha):
    L.original(directory);block=read(directory/'block.json')
    check(block['source_sha']==sha and block['chain_id']==10 and block['rpc_url']==L.RPC
          and type(block['number']) is int and block['number']>0 and type(block['timestamp']) is int and block['timestamp']>0
          and re.fullmatch('0x[0-9a-f]{64}',block['hash'])
          and block['policy']=='pinned original OP Mainnet L2 fork block'
          and block['requested'] in ('latest',str(block['number'])),'Changed L2 preflight identity')
    height=hex(block['number'])
    expected=[('eth_chainId',[]),('eth_getBlockByNumber',[height,False]),('eth_getCode',[L.ADDRESS,height]),
              ('eth_getStorageAt',[L.ADDRESS,L.SLOT,height]),('eth_call',[{'to':L.ADDRESS,'data':'0x54fd4d50'},height])]
    latest=block['requested']=='latest'
    rows=rpc(directory,([('eth_blockNumber',[])] if latest else [])+expected)
    if latest:
        check(rows[0]['attempts'][-1]['response']['result']==height,'Unbound latest L2 block discovery');rows=rows[1:]
    values=[row['attempts'][-1]['response']['result'] for row in rows]
    check(values[0]=='0xa' and values[1]['number']==height and values[1]['hash']==block['hash']
          and int(values[1]['timestamp'],16)==block['timestamp']
          and re.fullmatch('0x(?:[0-9a-f]{2})+',values[2]) and re.fullmatch('0x[0-9a-f]{64}',values[3])
          and int(values[3],16)>0 and re.fullmatch('0x(?:[0-9a-f]{2})+',values[4]),'Missing original L2 preflight state')
    return {'block':{k:v for k,v in block.items() if k!='requested'},'common_rpc':rows,
            'latest_discovery':latest,'original_sha256':L.files(directory)}


def runtime_transport(directory):
    """Validate the sealed policy and every recorded transport frame.

    A fresh Forge verdict can use only cached RPC state. Its relay report then
    contains zero requests. The verdict and pinned block checks remain separate.
    """
    final=read(directory/'final.json');settings=read(directory/'settings.json')
    check(set(final)=={'requests','sha256'} and type(final['requests']) is int and final['requests']>=0
          and final['sha256']==L.files(directory)
          and json.dumps(settings,sort_keys=True)==json.dumps({'upstream':L.RPC,'policy':L.P.POLICY},sort_keys=True),
          'Missing or corrupt L2 RPC transport originals')
    expected={'settings.json'};attempt_count=0;retry_count=0;statuses={};stable={};previous=None;live_requests=0
    def live(value):
        if isinstance(value,str):return value in ('latest','pending','safe','finalized')
        if isinstance(value,dict):return any(live(v) for v in value.values())
        if isinstance(value,list):return any(live(v) for v in value)
        return False
    for number in range(final['requests']):
        stem='request-'+str(number);path=directory/(stem+'.json');metadata=directory/(stem+'.metadata.json')
        expected|={path.name,metadata.name};request=read(path);identity=read(metadata)
        requests=request if isinstance(request,list) else [request]
        check(requests and all(isinstance(r,dict) and r.get('jsonrpc')=='2.0'
              and type(r.get('id')) in (int,str) and isinstance(r.get('method'),str)
              and (r.get('params') is None or isinstance(r['params'],(list,dict))) for r in requests)
              and len({r['id'] for r in requests})==len(requests)
              and identity=={'upstream':L.RPC,'request_sha256':L.digest(path)},'Changed original runtime RPC request')
        live_requests+=sum(live(r.get('params',[])) or r['method'] in ('eth_blockNumber','web3_clientVersion') for r in requests)
        attempts=sorted(directory.glob(stem+'-attempt-*.metadata.json'),key=lambda p:int(p.name.split('-')[3].split('.')[0]))
        check(1<=len(attempts)<=L.P.POLICY['max_attempts']
              and [p.name for p in attempts]==[stem+'-attempt-'+str(n)+'.metadata.json' for n in range(1,len(attempts)+1)],
              'Missing or excessive runtime RPC attempt')
        retry_count+=len(attempts)-1
        for n,metadata in enumerate(attempts,1):
            body=directory/(stem+'-attempt-'+str(n)+'.json');expected|={body.name,metadata.name};row=read(metadata)
            status=row['http_status'];attempt_count+=1;statuses[str(status)]=statuses.get(str(status),0)+1
            check(set(row)=={'http_status','error','started_at','elapsed_seconds','response_sha256'}
                  and (status is None or type(status) is int and 100<=status<=599)
                  and (row['error'] is None or isinstance(row['error'],str))
                  and row['response_sha256']==L.digest(body) and positive(row['started_at'])
                  and positive(row['elapsed_seconds']), 'Changed original runtime RPC attempt')
            check(previous is None or row['started_at']-previous>=0.4,'Runtime RPC requests exceeded the shared budget')
            previous=row['started_at']
            if n<len(attempts):
                check(status is None or status in L.P.POLICY['retry_http_statuses'],'Runtime RPC retried success or permanent denial')
            if status==200:
                value=read(body);responses=value if isinstance(value,list) else [value]
                by_id={r.get('id'):r for r in responses if isinstance(r,dict)}
                check(len(by_id)==len(responses)==len(requests) and set(by_id)=={r['id'] for r in requests}
                      and all(r.get('jsonrpc')=='2.0' and (('result' in r)!=('error' in r)) for r in responses),
                      'Mismatched original runtime RPC response IDs or results')
                for request in requests:
                    response=by_id[request['id']]
                    if 'result' not in response or live(request.get('params',[])) or request['method'] in ('eth_blockNumber','web3_clientVersion'):continue
                    key=json.dumps([request['method'],request.get('params',[])],sort_keys=True)
                    result=json.dumps(response['result'],sort_keys=True)
                    stable.setdefault(key,set()).add(result)
    check(set(final['sha256'])==expected,'Missing or extra complete runtime RPC frame')
    return {'requests':final['requests'],'attempts':attempt_count,'transport_retries':retry_count,
            'http_statuses':statuses,'live_metadata_requests':live_requests,
            'stable_results':{k:sorted(v) for k,v in stable.items()}}


def report(directory,validate_authority=True):
    final=L.original(directory);s=read(directory/'settings.json')
    check(s['branch']=='codex/rwx-ci-pilot' and s['provider'] in ('circleci','rwx')
          and (s['profile'],s['feature'],s['chain'],s['match_path'])==('ci','main','op-mainnet',L.MATCH)
          and json.dumps(s['runtime'],sort_keys=True)==json.dumps(L.RUNTIME,sort_keys=True),
          'Changed complete L2 workload or RPC concurrency')
    if validate_authority:authority(s)
    check(set(s['tools'])=={'forge','cast','go','just'} and all(re.fullmatch('[0-9a-f]{64}',row['sha256'])
          and row['version'] for row in s['tools'].values()),'Missing pinned L2 tool identity')
    if s['provider']=='rwx':check(re.fullmatch('[0-9a-f]{32}',s['rwx_run_id']) and s['rwx_task_attempt']=='1','Uninvestigated native L2 retry')
    config=read(directory/'foundry-config.json');L.effective(config)
    bindings=read(directory/'signature-bindings.json');compiled=read(directory/'compiled.json')
    check(compiled and all(name.startswith('packages/contracts-bedrock/') and '..' not in Path(name).parts
          and re.fullmatch('[0-9a-f]{64}',hashed) for name,hashed in compiled.items()),'Invalid L2 compiler inventory')
    artifacts=directory/'compiler-artifacts';previous=L.UP.ROOT
    try:
        L.UP.ROOT=artifacts
        rebuilt=L.UP.compiler_signatures(artifacts/'packages/contracts-bedrock/forge-artifacts')
    finally:L.UP.ROOT=previous
    check(bindings and all(rebuilt[key]==value for key,value in bindings.items()),'Changed original L2 compiler signatures or bytecode')
    captured={str(p.relative_to(artifacts)):L.digest(p) for p in artifacts.rglob('*') if p.is_file()}
    expected={name:hashed for value in bindings.values() for name,hashed in value['artifacts'].items()}
    check(captured==expected and all(compiled.get(name)==hashed for name,hashed in expected.items()),'Missing or extra L2 compiler artifact')
    discovery=read(directory/'discovery.json')
    check(all(path.startswith('test/L2/fork/') for path in discovery),'Foreign original L2 test discovery')
    selection=L.UP.selection(discovery,bindings)
    check(selection==[tuple(row) for row in read(directory/'selection.json')],'Changed original L2 selection')
    coverage=L.UP.junit(directory/'original.junit.xml',selection,bindings)
    check(coverage==read(directory/'coverage.json') and final['tests']==len(coverage['original_cases']), 'Incomplete original L2 verdicts')
    expected_commands={'foundry-config':['forge','config','--json'],'go-ffi':['just','build-go-ffi'],'contracts-build':['forge','build'],
       'discovery':['forge','test','--list','--json','--match-path',L.MATCH],
       'nut-bundle-check':['just','nut-bundle-check-no-build'],'tests':['just','test-l2-fork-upgrade'] + L.TEST_ARGS}
    check({p.name.removesuffix('.stage.json') for p in directory.glob('*.stage.json')}==set(expected_commands),
          'Missing L2 command or uninvestigated diagnostic rerun')
    for name,argv in expected_commands.items():
        stage=read(directory/(name+'.stage.json'));stdout=directory/(name+('.json' if name in ('foundry-config','discovery') else '.log'))
        check(stage['argv']==argv and stage['cwd']==s['workspace_root']+'/packages/contracts-bedrock'
              and type(stage['exit_code']) is int and stage['exit_code']==0 and positive(stage['started_at'])
              and positive(stage['elapsed_seconds']) and stage['stdout_sha256']==L.digest(stdout)
              and stage['stderr_sha256']==L.digest(directory/(name+'.stderr.log')), 'Failed or changed original L2 command')
    pinned=preflight(directory/'preflight',s['source_sha'])
    check(read(directory/'block.json')==read(directory/'preflight/block.json'),'Changed runtime L2 preflight block')
    height=hex(pinned['block']['number']);recheck=rpc(directory,[('eth_chainId',[]),('eth_getBlockByNumber',[height,False])])
    check(recheck==pinned['common_rpc'][:2],'Changed original L2 runtime block or retry history')
    transport=runtime_transport(directory/'runtime-rpc')
    return {'settings':s,'selection':selection,'coverage':coverage,'config':L.UP.ORIGINALS.normalize(config,s['workspace_root']),
            'submodules':L.UP.SUBMODULES.revisions((directory/'submodules.txt').read_text()),
            'compiler_methods':{k:{'methods':v['methods'],'deployable':L.UP.deployable(v)} for k,v in bindings.items()},
            'preflight':pinned,'runtime_rpc':recheck,'rpc_transport':transport,
            'commands':expected_commands,'original_sha256':L.files(directory)}


def equal(circle,native):
    c,n=report(circle),report(native)
    check(c['settings']['provider']=='circleci' and n['settings']['provider']=='rwx','Reversed L2 providers')
    for key in ('source_sha','branch','profile','feature','chain','match_path','runtime','tools','inputs','implementation'):
        check(c['settings'][key]==n['settings'][key],'L2 settings parity differs at '+key)
    for key in ('selection','coverage','config','submodules','compiler_methods','runtime_rpc','commands'):
        check(c[key]==n[key],'Complete original L2 parity differs at '+key)
    for key in ('block','common_rpc'):check(c['preflight'][key]==n['preflight'][key],'Original L2 preflight parity differs at '+key)
    cr,nr=(row['rpc_transport']['stable_results'] for row in (c,n))
    check(all(cr[key]==nr[key] for key in cr.keys()&nr.keys()),'Original stable runtime RPC results differ')
    check(not c['preflight']['latest_discovery'], 'Circle replay must use the shared original block')
    return c,n


def hosted(circle,native,run_path,github_path):
    s=read(native/'settings.json');run=read(run_path)
    check(run['ID']==run['RunID']==s['rwx_run_id'] and run['CommitSha']==s['source_sha'] and run['Branch']==s['branch']
          and run['Trigger']=='github.push' and run['TargetedTaskKeys'] is None,'Foreign or targeted native L2 execution')
    tasks=G.named(run['Tasks'],'Key')
    check_task=None
    if run['DefinitionPath']=='.rwx/pr-gates.yml':
        manifest=json.loads(G.source('ops/ci/pr-gates.json',s['source_sha']))
        definition=manifest['groups']['contracts-l2-fork'];check_task=definition['embedded_task']
        check(definition['config']=='.rwx/contract-l2-fork.yml'
              and definition['init']=={'cache-warm':'false','cache-epoch':'v1','fork-block':'latest'}, 'Changed embedded L2 mode')
        with tempfile.TemporaryDirectory() as temporary:
            path=Path(temporary)/'coordinator.yml';path.write_bytes(G.source(run['DefinitionPath'],s['source_sha']))
            coordinator=G.yaml(path)
        embedded=G.named(coordinator['tasks'],'key')[check_task]
        check(embedded['call']=='${{ run.dir }}/contract-l2-fork.yml'
              and embedded['init']=={key:'${{ init.'+key+' }}' for key in ('commit-sha','branch','tag')}|definition['init'],
              'Changed actual native L2 provenance or caller')
        tasks=G.named(tasks[check_task]['Subtasks'],'Key')
    else:check(run['DefinitionPath']=='.rwx/contract-l2-fork.yml','Foreign native L2 definition')
    for key in ('rpc-check','helper-tests','verdict'):G.fresh(tasks[key])
    identity=read(circle/'settings.json');sha=s['source_sha']
    check(identity=={'provider':'circleci','source_sha':sha,'branch':s['branch'],'pipeline_id':identity['pipeline_id'],
          'workflow_id':identity['workflow_id'],'job_number':identity['job_number']},'Foreign Circle L2 collection')
    check(read(circle/'final.json')['sha256']==L.files(circle),'Missing or corrupt complete Circle L2 collection')
    pipeline,workflow,job=(read(circle/name) for name in ('pipeline.json','workflow.json','job.json'))
    check(pipeline['id']==identity['pipeline_id'] and pipeline['vcs']['revision']==sha and pipeline['vcs']['branch']==s['branch']
          and not pipeline['errors'] and workflow['id']==identity['workflow_id'] and workflow['pipeline_id']==identity['pipeline_id']
          and workflow['name'] in ('l2-fork-parity-replay','contracts-feature-tests') and workflow['status']=='success','Foreign Circle L2 replay')
    jobs=[];pages=sorted((circle/'api-pages').glob('jobs-*.json'),key=lambda p:int(p.stem.split('-')[1]))
    check([p.name for p in pages]==['jobs-'+str(i)+'.json' for i in range(len(pages))] and pages,'Missing Circle L2 job page')
    for i,path in enumerate(pages):
        page=read(path);jobs.extend(page['items']);check(bool(page.get('next_page_token'))==(i<len(pages)-1),'Incomplete Circle L2 pagination')
    named=G.named(jobs,'name');name='contracts-bedrock-tests-l2-fork op-mainnet'
    expected={'prep-go-modules',name}
    if workflow['name']=='contracts-feature-tests':
        manifest=json.loads(G.source('ops/ci/pr-gates.json',sha))
        expected|={'required-contracts-ci'}|{row['name'] for row in manifest['gates']['required-contracts-ci']['dependencies']}
    check(set(named)==expected and all(row['status']=='success' for row in jobs)
          and named[name]['dependencies']==[named['prep-go-modules']['id']]
          and named[name]['job_number']==job['build_num']==identity['job_number'] and job['workflows']['job_id']==named[name]['id']
          and job['vcs_revision']==sha and job['branch']==s['branch'] and job['workflows']['workflow_id']==identity['workflow_id']
          and job['status']==job['outcome']=='success' and not job['retry_of']
          and all(job[key] is False for key in ('failed','timedout','canceled','infrastructure_fail')),'Failed, incomplete or retried Circle L2 workload')
    config=read(circle/'pipeline-config.json')
    check(config['compiled']==(circle/'compiled.yml').read_text() and config['source']==(circle/'source.yml').read_text(),'Changed Circle L2 compiled originals')
    compiled=G.yaml(circle/'compiled.yml');selected=compiled['workflows'][workflow['name']]['jobs']
    check(sum(isinstance(row,dict) and name in row for row in selected)==1
          and sum(isinstance(row,dict) and 'prep-go-modules' in row for row in selected)==1,'Changed actual Circle L2 workload selection')
    commands=[step['run']['command'] for step in compiled['jobs'][name]['steps'] if isinstance(step,dict) and isinstance(step.get('run'),dict)]
    check(any('python3 ../../ops/ci/contract-l2-fork.py run' in command for command in commands)
          and any('python3 ../../ops/ci/contract-l2-fork.py prepare' in command for command in commands), 'Circle L2 did not execute the complete shared runner')
    for name in ('prepare.console.log','run.console.log'):
        check((circle/'archive'/name).is_file() and (circle/'archive'/name).stat().st_size>0, 'Missing complete original Circle L2 console')
    index=read(circle/'step-index.json')
    check([(row['step'],row['action']) for row in index]==[(i,j) for i,step in enumerate(job['steps'])
          for j,action in enumerate(step['actions']) if action.get('output_url')],'Missing complete Circle L2 original steps')
    for row in index:
        check(all(event.get('truncated') is False and isinstance(event['message'],str) for event in read(circle/row['path'])), 'Truncated original Circle L2 console')
    github=read(github_path);checks=G.named(github['checks'],'name');optional=checks['RWX: optimism-contract-l2-fork-shadow']
    check(github['sha']==sha and optional['state']=='success' and '/runs/'+run['ID'] in optional['link']
          and (not check_task or '/latest/'+check_task in optional['link']), 'Missing successful same-SHA native L2 check')
    return {'circle_pipeline':pipeline['number'],'circle_job':job['build_num'],'native_run':run['ID']}


def compare(circle,native,run_path,github_path):
    ids=hosted(circle,native,run_path,github_path);c,n=equal(circle/'report',native)
    return {'state':'passed','source_sha':c['settings']['source_sha'],'selection':c['selection'],'coverage':c['coverage'],
            'block':c['preflight']['block'],'commands':c['commands'],'hosted':ids,
            'original_sha256':{'circle':L.files(circle),'native':n['original_sha256']},
            'runtime_rpc_transport':{provider:{k:v for k,v in row['rpc_transport'].items() if k!='stable_results'}
                for provider,row in (('circleci',c),('rwx',n))},
            'block_discovery':'Complete RPC originals retain either native head discovery or the committed pilot snapshot; both providers use the same original height'}


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    for name in ('circle','native','run_state','github_checks','output'):parser.add_argument(name,type=Path)
    args=parser.parse_args();result=compare(args.circle,args.native,args.run_state,args.github_checks)
    args.output.parent.mkdir(parents=True,exist_ok=True);L.write(args.output,result)
    print(json.dumps({key:result[key] for key in ('state','source_sha','hosted')}))
