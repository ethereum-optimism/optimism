#!/usr/bin/env python3
"""Prepare the complete Circle Kontrol compiler graph without running tests."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name+'.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


M = helper('main-checks'); S = helper('rust-workspace-report'); A = helper('go-artifacts')
ROOT = Path(__file__).resolve().parents[2]
CONTRACTS = ROOT/'packages/contracts-bedrock'
POLICY = {'profile':'ci', 'build_args':['--skip','test'],
          'legacy_graph':{'source':'scripts/Artifacts.s.sol','compiler':'0.8.28'},
          'available_compilers':['0.8.15','0.8.19','0.8.25','0.8.28','0.8.30']}
COMMANDS = {'legacy-graph':['forge','build','scripts/Artifacts.s.sol','--use','0.8.28'],
            'ci-build':['just','forge-build','--skip','test'],
            'copy-artifacts':['just','copy-contract-artifacts']}


def read(path): return json.loads(path.read_text())
def seal(path): return {str(p.relative_to(path)):S.digest(p) for p in sorted(path.rglob('*')) if p.is_file()}
def command(*args): return subprocess.check_output(args,cwd=ROOT,text=True).strip()


def tools():
    result = {name:command(*argv) for name,argv in [('forge',['forge','--version']),('go',['go','version']),('just',['just','--version'])]}
    result['solc'] = {}
    for version in POLICY['available_compilers']:
        path = Path(command('svm','which',version))
        if not path.is_file(): raise ValueError('Missing actual Kontrol compiler '+version)
        result['solc'][version] = {'sha256':S.digest(path), 'version':subprocess.check_output([str(path),'--version'],text=True).strip()}
    return result


def execute():
    os.chdir(ROOT)
    output = ROOT/'.ci/go-tests/dependencies/contracts-kontrol'
    shutil.rmtree(output,ignore_errors=True); report = output/'preparation'; report.mkdir(parents=True)
    fingerprint = ROOT/'.ci/kontrol-contracts/target-inputs.json'
    status,errors = 1,[]
    try:
        sha = command('git','rev-parse','HEAD')
        if sha != os.environ['CI_COMMIT_SHA'] or subprocess.call(['git','diff','--quiet','HEAD'],cwd=ROOT):
            raise ValueError('Changed or stale Kontrol contract producer source')
        before = M.inputs(); settings = {'suite':'kontrol-contracts','source_sha':sha,'workspace_root':str(ROOT),'policy':POLICY,
                                        'input_sha256':before,'tools':tools()}
        S.write(report/'settings.json',settings)
        reused = fingerprint.is_file() and read(fingerprint) == settings
        S.write(report/'cache.json',{'reused':reused,'fingerprint':settings})
        if not reused:
            for name in ('cache','artifacts','forge-artifacts'):
                shutil.rmtree(CONTRACTS/name,ignore_errors=True)
        os.environ['FOUNDRY_PROFILE']='ci'
        os.environ['GOMODCACHE']=str(ROOT/'.ci/go-cache/full/modules')
        os.environ['GOCACHE']=str(ROOT/'.ci/go-cache/full/dependencies')
        for name,argv in COMMANDS.items():
            cwd = ROOT/'op-deployer' if name=='copy-artifacts' else CONTRACTS
            status = S.stage(report,name,argv,cwd=str(cwd),stdin=subprocess.DEVNULL)
            if status: raise ValueError('Kontrol contract preparation failed: '+name)
        after = M.inputs(); S.write(report/'inputs-after.json',after)
        if after != before: raise ValueError('Kontrol compiler preparation changed source inputs')
        A.pack('contracts-kontrol',['packages/contracts-bedrock/cache','packages/contracts-bedrock/artifacts',
               'packages/contracts-bedrock/forge-artifacts','op-deployer/pkg/deployer/artifacts/forge-artifacts'])
        fingerprint.parent.mkdir(parents=True,exist_ok=True); S.write(fingerprint,settings)
        status = 0
    except Exception as error:
        errors.append(str(error));status=status or 1;fingerprint.unlink(missing_ok=True);print(error,file=sys.stderr)
    finally:
        S.write(report/'final.json',{'exit_code':status,'report_errors':errors,'tests':0,'original_sha256':seal(report)})
        if not status:
            metadata = read(output/'metadata.json');metadata['preparation_sha256']=seal(report)
            S.write(output/'metadata.json',metadata)
    return status


def verify(directory,sha,inputs):
    report = directory/'preparation'; final = read(report/'final.json');settings=read(report/'settings.json')
    if final['exit_code'] or final['report_errors'] or final['tests'] != 0:
        raise ValueError('Failed or false Kontrol contract preparation')
    if (seal(report) != final['original_sha256'] | {'final.json':S.digest(report/'final.json')}
            or read(directory/'metadata.json')['preparation_sha256'] != seal(report)):
        raise ValueError('Missing, corrupt or extra original Kontrol preparation report')
    if (settings['suite'] != 'kontrol-contracts' or settings['source_sha'] != sha or settings['policy'] != POLICY
            or settings['input_sha256'] != inputs or read(report/'inputs-after.json') != inputs):
        raise ValueError('Stale Kontrol preparation source, policy or inputs')
    if set(settings['tools']['solc']) != set(POLICY['available_compilers']): raise ValueError('Incomplete Kontrol compiler set')
    metadata = read(directory/'metadata.json')
    if any(settings['tools'][name] != value for name,value in metadata['tool_versions'].items()):
        raise ValueError('Kontrol preparation tools differ from its artifact binding')
    expected_cache = {'reused':read(report/'cache.json')['reused'],'fingerprint':settings}
    if type(expected_cache['reused']) is not bool or read(report/'cache.json') != expected_cache:
        raise ValueError('Wrong Kontrol compiler cache binding')
    stages = {p.name.removesuffix('.stage.json') for p in report.glob('*.stage.json')}
    if stages != COMMANDS.keys(): raise ValueError('Missing or extra Kontrol preparation invocation')
    for name,argv in COMMANDS.items():
        row = read(report/(name+'.stage.json'))
        cwd = settings['workspace_root']+('/op-deployer' if name=='copy-artifacts' else '/packages/contracts-bedrock')
        if row['argv'] != argv or row['cwd'] != cwd or row['exit_code'] or row.get('stdin') != 'devnull' or row['log_sha256'] != S.digest(report/(name+'.log')):
            raise ValueError('Wrong or failed original Kontrol preparation command')
    return settings


if __name__=='__main__': sys.exit(execute())
