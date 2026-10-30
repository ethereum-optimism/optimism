#!/usr/bin/env python3
"""Compile native guests at the original Circle paths, preserving ELF identity."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
WORKSPACE = Path('/home/circleci/project')
CARGO = Path('/data/mise-data/.cargo')
TARGET = WORKSPACE/'rust/kona/sp1/programs/target'
POLICY = {'workspace_root':str(WORKSPACE),'cargo_home':str(CARGO),'target_directory':str(TARGET),'extra_rustflags':[]}


def helper(name):
    spec=importlib.util.spec_from_file_location(name.replace('-','_'),Path(__file__).with_name(name+'.py'))
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module);return module


S=helper('rust-workspace-report'); M=helper('main-checks')
def read(path):return json.loads(path.read_text())
def seal(path):return {str(p.relative_to(path)):S.digest(p) for p in sorted(path.rglob('*')) if p.is_file()}


def execute(features=""):
    cache=ROOT/'.ci/sp1-cache'
    staging=ROOT/'.ci/sp1-guest/native-workspace';shutil.rmtree(staging,ignore_errors=True);staging.mkdir(parents=True)
    output=ROOT/'.ci/sp1-guest/dependency';shutil.rmtree(output,ignore_errors=True)
    status,errors,owned_workspace=1,[],False
    try:
        sha=subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip()
        before=M.inputs()
        if sha!=os.environ['CI_COMMIT_SHA'] or subprocess.call(['git','diff','--quiet','HEAD'],cwd=ROOT):
            raise ValueError('Changed or stale native SP1 build source')
        if (WORKSPACE.exists() or WORKSPACE.is_symlink() or CARGO.exists() or CARGO.is_symlink()
                or WORKSPACE.resolve()!=WORKSPACE or CARGO.resolve()!=CARGO):
            raise ValueError('Native SP1 canonical directories are already occupied')
        subprocess.run(['sudo','mkdir','-p',str(WORKSPACE),str(CARGO)],check=True)
        subprocess.run(['sudo','chown',str(os.getuid())+':'+str(os.getgid()),str(WORKSPACE),str(CARGO)],check=True)
        for name,argv in [('clone',['git','clone','--shared','--no-checkout',str(ROOT),str(WORKSPACE)]),
                          ('checkout',['git','-C',str(WORKSPACE),'checkout','--detach',sha]),
                          ('trust',['mise','trust','--yes',str(WORKSPACE/'mise.toml')])]:
            status=S.stage(staging,name,argv,cwd=str(ROOT),stdin=subprocess.DEVNULL)
            if status:raise ValueError('Native SP1 workspace preparation failed: '+name)
            if name=='clone':owned_workspace=True
        for source,target in [(cache/'cargo',CARGO),(cache/'elf-target',TARGET)]:
            if source.exists():shutil.copytree(source,target,symlinks=True,dirs_exist_ok=True)
            else:target.mkdir(parents=True,exist_ok=True)
        S.write(staging/'settings.json',{'source_sha':sha,'input_sha256':before,'native_workspace_root':str(ROOT),'policy':POLICY})
        env={**os.environ,'SP1_GUEST_NATIVE_CACHE_ROOT':str(cache)}
        status=S.stage(staging,'toolchain',['python3','ops/ci/runtime/sp1-guest.py','--provider','rwx','--phase','toolchain'],
                       cwd=str(WORKSPACE),stdin=subprocess.DEVNULL)
        if status:raise ValueError('Native SP1 canonical toolchain verification failed')
        status=subprocess.call([sys.executable,'ops/ci/runtime/sp1-guest.py','--provider','rwx','--phase','build','--features',features],cwd=WORKSPACE,env=env)
        after=M.inputs();S.write(staging/'inputs-after.json',after)
        if before!=after:raise ValueError('Native SP1 workspace preparation changed original sources')
    except Exception as error:
        errors.append(str(error));status=status or 1;print(error,file=sys.stderr)
    finally:
        original=WORKSPACE/'.ci/sp1-guest/dependency'
        if owned_workspace and original.is_dir():shutil.copytree(original,output)
        else:output.mkdir(parents=True,exist_ok=True)
        if owned_workspace and not original.is_dir() and (WORKSPACE/'.ci/sp1-guest/toolchain').is_dir():
            shutil.copytree(WORKSPACE/'.ci/sp1-guest/toolchain',output/'toolchain')
        try:
            if not status:
                for source,target in [(CARGO,cache/'cargo'),(TARGET,cache/'elf-target')]:
                    shutil.rmtree(target,ignore_errors=True);shutil.copytree(source,target,symlinks=True)
        except Exception as error:
            errors.append(str(error));status=status or 1;print(error,file=sys.stderr)
        S.write(staging/'final.json',{'exit_code':status,'report_errors':errors,'tests':0,'original_sha256':seal(staging)})
        shutil.copytree(staging,output/'native-workspace')
        final=read(output/'final.json') if (output/'final.json').is_file() else {'phase':'build','tests':0,'report_errors':[]}
        final['exit_code']=status;final['report_errors']+=errors
        final['original_sha256']={n:h for n,h in seal(output).items() if n!='final.json'}
        S.write(output/'final.json',final)
    return status


def verify(directory,settings,native_workspace_root=None):
    native=directory/'native-workspace';final=read(native/'final.json');binding=read(native/'settings.json')
    if final['exit_code'] or final['report_errors'] or final['tests']!=0:
        raise ValueError('Failed native SP1 canonical workspace preparation')
    if seal(native)!=final['original_sha256']|{'final.json':S.digest(native/'final.json')}:
        raise ValueError('Missing, corrupt or extra native SP1 workspace originals')
    if (binding['source_sha']!=settings['source_sha'] or binding['input_sha256']!=settings['input_sha256']
            or binding['policy']!=POLICY or read(native/'inputs-after.json')!=settings['input_sha256']
            or settings['workspace_root']!=str(WORKSPACE) or settings['cargo_home']!=str(CARGO)
            or read(directory/'guest-workspace.json')['target_directory']!=str(TARGET)
            or (native_workspace_root is not None and binding['native_workspace_root']!=native_workspace_root)):
        raise ValueError('Wrong native SP1 build paths, revision or source inputs')
    root=binding['native_workspace_root'];expected={
        'clone':['git','clone','--shared','--no-checkout',root,str(WORKSPACE)],
        'checkout':['git','-C',str(WORKSPACE),'checkout','--detach',settings['source_sha']],
        'trust':['mise','trust','--yes',str(WORKSPACE/'mise.toml')],
        'toolchain':['python3','ops/ci/runtime/sp1-guest.py','--provider','rwx','--phase','toolchain']}
    if {p.name.removesuffix('.stage.json') for p in native.glob('*.stage.json')}!=expected.keys():
        raise ValueError('Missing or extra native SP1 workspace invocation')
    for name,argv in expected.items():
        row=read(native/(name+'.stage.json'))
        cwd=str(WORKSPACE) if name=='toolchain' else root
        if row['argv']!=argv or row['cwd']!=cwd or row['exit_code'] or row.get('stdin')!='devnull' or row['log_sha256']!=S.digest(native/(name+'.log')):
            raise ValueError('Wrong or failed native SP1 workspace command')


if __name__=='__main__':sys.exit(execute())
