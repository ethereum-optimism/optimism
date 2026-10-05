#!/usr/bin/env python3
"""Run complete original NUT regeneration with recorded Forge/Just provenance."""
import argparse
import base64
from collections import Counter
import importlib.util
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tomllib


def helper(name):
    spec=importlib.util.spec_from_file_location(name,Path(__file__).with_name(name+'.py'))
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module);return module


G=helper('cannon-go');M=helper('main-checks');S=G.S
ROOT=Path(__file__).resolve().parents[2]
LOCK='op-core/nuts/fork_lock.toml'
ENVIRONMENT=('CI','FOUNDRY_PROFILE','GOFLAGS','GOEXPERIMENT','GOTOOLCHAIN')


def command(*argv,root=None):return subprocess.check_output(argv,cwd=ROOT if root is None else root,text=True).strip()
def seal(path):return {str(p.relative_to(path)):S.digest(p) for p in sorted(path.rglob('*')) if p.is_file()}


def discover(root,full):
    current=(root/LOCK).read_bytes();locks=tomllib.loads(current.decode())
    if not locks:raise ValueError('Empty complete NUT lock selection')
    base_result=subprocess.run(['git','show','origin/develop:'+LOCK],cwd=root,capture_output=True)
    base=tomllib.loads(base_result.stdout.decode()) if base_result.returncode==0 else {}
    base_revision=subprocess.run(['git','rev-parse','origin/develop'],cwd=root,capture_output=True,text=True)
    entries={};selected=[];paths=set();historical={}
    for fork,entry in sorted(locks.items()):
        if (not re.fullmatch('[a-z][a-z0-9-]*',fork) or not isinstance(entry,dict)
                or set(entry)!={'bundle','hash','commit'} or any(not isinstance(v,str) for v in entry.values())):
            raise ValueError('Invalid complete NUT lock entry')
        path=Path(entry['bundle'])
        if (path.is_absolute() or '..' in path.parts or not str(path).startswith('op-core/nuts/bundles/')
                or path.suffix!='.json' or entry['bundle'] in paths or (root/path).is_symlink()
                or not re.fullmatch('[0-9a-f]{40}',entry['commit']) or not re.fullmatch('sha256:[0-9a-f]{64}',entry['hash'])):
            raise ValueError('Unsafe, duplicate or unpinned NUT provenance input')
        paths.add(entry['bundle'])
        if 'sha256:'+S.digest(root/path)!=entry['hash']:raise ValueError('Locked NUT bundle hash mismatch')
        if command('git','rev-parse',entry['commit']+'^{commit}',root=root)!=entry['commit']:
            raise ValueError('Missing or foreign recorded NUT source')
        raw=subprocess.check_output(['git','show',entry['commit']+':mise.toml'],cwd=root)
        tools=tomllib.loads(raw.decode())['tools'];pins={name:tools[name] for name in ('forge','just')}
        if any(not isinstance(pin,str) or not re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+',pin) for pin in pins.values()):
            raise ValueError('Historical NUT generator toolchain is not pinned')
        historical[fork]=raw
        entries[fork]={**entry,'historical_tools':pins,'historical_mise_sha256':hashlib.sha256(raw).hexdigest()}
        if full or base.get(fork,{}).get('hash','')!=entry['hash']:selected.append(fork)
    selection={'authority':'fork_lock.toml; original hash-changed policy vs origin/develop',
            'mode':'full' if full else 'changed','entries':entries,'selected_forks':selected,
            'excluded':[{'fork':fork,'reason':'Original unchanged bundle hash'} for fork in entries if fork not in selected],
            'base_revision':base_revision.stdout.strip() if base_revision.returncode==0 else None,
            'base_lock_available':base_result.returncode==0,'base_lock_exit_code':base_result.returncode,
            'generator_argv':['just','generate-nut-bundle'],'go_argv':['go','run','./ops/scripts/nut-provenance-verify'],
            'historical_pin_rule':'Forge and Just from each recorded source commit; current CI Go for the verifier',
            'tests':0,'retries':0}
    return selection,current,base_result.stdout,base_result.stderr,historical


def prefix(pins):return ['mise','exec','forge@'+pins['forge'],'just@'+pins['just'],'--']


def tool_identity(directory,pins):
    identity={}
    for name,args in [('forge',['--version']),('just',['--version']),('go',['version'])]:
        argv=prefix(pins)+[name,*args]
        if S.stage(directory,name+'-version',argv,cwd=str(ROOT),stdout_file=name+'-version.txt'):
            raise ValueError('NUT tool identity failed: '+name)
        version=(directory/(name+'-version.txt')).read_text().strip()
        if name=='forge' and not re.search(r'^forge Version: '+re.escape(pins[name])+r'(?:-|$)',version,re.M):
            raise ValueError('Wrong NUT generator tool version')
        if name=='just' and version!='just '+pins[name]:raise ValueError('Wrong NUT generator tool version')
        binary=Path(command(*prefix(pins),'which',name))
        identity[name]={'version':version,'binary_sha256':S.digest(binary),'binary_path':str(binary)}
    S.write(directory/'tools.json',identity);return identity


def compiler_identity(directory):
    versions=sorted(set(re.findall(r'Compiling [0-9]+ files with Solc ([0-9.]+)',(directory/'generation.log').read_text())))
    if not versions:raise ValueError('Original full NUT regeneration compiled no contracts')
    result={}
    for version in versions:
        name='solc-'+version
        if S.stage(directory,name+'-path',['mise','exec','svm-rs','--','svm','which',version],cwd=str(ROOT),stdout_file=name+'-path.txt'):
            raise ValueError('Missing original NUT Solidity compiler')
        path=Path((directory/(name+'-path.txt')).read_text().strip())
        if not path.is_absolute() or not path.is_file():raise ValueError('Invalid original NUT compiler path')
        if S.stage(directory,name+'-version',[str(path),'--version'],cwd=str(ROOT),stdout_file=name+'-version.txt'):
            raise ValueError('Original NUT compiler identity failed')
        actual=(directory/(name+'-version.txt')).read_text().strip()
        if not re.search(r'^Version: '+re.escape(version)+r'\+',actual,re.M):raise ValueError('Wrong original NUT compiler version')
        result[version]={'version':actual,'binary_sha256':S.digest(path),'binary_path':str(path)}
    S.write(directory/'compilers.json',result)


def originals(path,provider=None):
    final=G.read(path/'final.json')
    if (type(final['exit_code']) is not int or final['exit_code']!=0 or final['report_errors']!=[]
            or type(final['tests']) is not int or final['tests']!=0 or not final['original_sha256']):
        raise ValueError('Failed or incomplete original NUT provenance report')
    if any(p.is_symlink() for p in path.rglob('*')):raise ValueError('Unsafe original NUT report link')
    for name,digest in final['original_sha256'].items():
        if Path(name).is_absolute() or '..' in Path(name).parts or not re.fullmatch('[0-9a-f]{64}',digest):raise ValueError('Unsafe NUT report manifest')
        # Circle's artifact service omits zero-byte files. Reconstruct only
        # bytes established by the original sealed manifest, never a verdict.
        if provider=='circleci' and not (path/name).exists() and digest==hashlib.sha256(b'').hexdigest():
            (path/name).parent.mkdir(parents=True,exist_ok=True);(path/name).write_bytes(b'')
        if not (path/name).is_file() or S.digest(path/name)!=digest:raise ValueError('Missing or corrupt original NUT evidence')
    if seal(path)!=final['original_sha256']|{'final.json':S.digest(path/'final.json')}:raise ValueError('Extra or unsealed original NUT evidence')
    return final


def execute(provider,full,prepare=False,module_artifact=None,tool_artifact=None):
    os.chdir(ROOT);directory=ROOT/'.ci/nut-provenance'/('toolchain' if prepare else 'run')
    shutil.rmtree(directory,ignore_errors=True);directory.mkdir(parents=True)
    status,errors=1,[]
    try:
        sha=command('git','rev-parse','HEAD');expected=os.environ.get('CI_COMMIT_SHA') or os.environ.get('CIRCLE_SHA1')
        if sha!=expected or not re.fullmatch('[0-9a-f]{40}',sha) or subprocess.call(['git','diff','--quiet','HEAD']):
            raise ValueError('Stale or changed NUT provenance source')
        if provider=='rwx' and not full:raise ValueError('Native NUT verdicts require the complete selection')
        branch=os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH')
        if not branch or os.environ.get('GOFLAGS'):raise ValueError('Missing NUT branch or overridden Go flags')
        if provider=='rwx' and (not re.fullmatch('[0-9a-f]{32}',os.environ.get('RWX_RUN_ID',''))
                or not re.fullmatch('[1-9][0-9]*',os.environ.get('RWX_TASK_ATTEMPT_NUMBER',''))):
            raise ValueError('Missing native NUT run identity or invalid task attempt')
        before=M.inputs()
        settings={'suite':'nut-provenance','source_sha':sha,'branch':branch,'provider':provider,'prepare_only':prepare,
            'input_sha256':before,'workspace_root':str(ROOT),'environment':{k:os.environ.get(k) for k in ENVIRONMENT},
            'go_version':command('go','version'),'cpus':os.cpu_count(),'rwx_run_id':os.environ.get('RWX_RUN_ID'),
            'rwx_task_attempt':os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}
        S.write(directory/'settings.json',settings)
        selection,current,base,base_error,historical=discover(ROOT,full)
        S.write(directory/'selection.json',selection)
        (directory/'fork_lock.toml').write_bytes(current);(directory/'base-fork-lock.toml').write_bytes(base)
        (directory/'base-discovery.stderr').write_bytes(base_error)
        for fork,raw in historical.items():
            target=directory/'historical'/fork;target.mkdir(parents=True);(target/'mise.toml').write_bytes(raw)
        if provider=='rwx' and not prepare and (module_artifact is None or tool_artifact is None):
            raise ValueError('Native NUT runtime requires verified modules and historical tools')
        if module_artifact is not None:
            G.ARTIFACTS.restore('go-modules',module_artifact)
            shutil.copyfile(ROOT/'tmp/testlogs/dependencies/go-modules.json',directory/'go-modules.json')
            if G.read(directory/'go-modules.json')['tool_versions']!={'go':settings['go_version']}:
                raise ValueError('NUT module producer toolchain differs from runtime')
        if tool_artifact is not None:
            originals(tool_artifact);old=G.read(tool_artifact/'settings.json')
            if (old['source_sha']!=sha or old['input_sha256']!=before or old['prepare_only'] is not True
                    or old['branch']!=branch or old['provider']!='rwx' or old['go_version']!=settings['go_version']
                    or G.read(tool_artifact/'selection.json')!=selection):
                raise ValueError('Stale or differently selected historical NUT tool producer')
            shutil.copytree(tool_artifact,directory/'tool-producer')
        if (prepare or provider=='circleci') and selection['selected_forks']:
            if S.stage(directory,'compiler-prepare',['bash',str(ROOT/'ops/ci/nut-tools.sh'),'solc'],cwd=str(ROOT)):
                raise ValueError('Historical NUT compiler preparation failed')
        prepared=set()
        for fork in selection['selected_forks']:
            target=directory/'forks'/fork;target.mkdir(parents=True)
            pins=selection['entries'][fork]['historical_tools'];key=tuple(pins.items())
            if (prepare or provider=='circleci') and key not in prepared:
                # Install pinned historical generators explicitly. Native setup
                # disables mise auto-install; verdicts reuse the verified tools.
                argv=['bash',str(ROOT/'ops/ci/nut-tools.sh'),'generators',pins['forge'],pins['just']]
                if S.stage(target,'tool-prepare',argv,cwd=str(ROOT)):raise ValueError('Historical NUT tool installation failed')
                prepared.add(key)
            identity=tool_identity(target,pins)
            if tool_artifact is not None:
                old=G.read(tool_artifact/'forks'/fork/'tools.json')
                if any({k:v for k,v in row.items() if k!='binary_path'}!={k:v for k,v in old[name].items() if k!='binary_path'} for name,row in identity.items()):
                    raise ValueError('Historical NUT tools differ from producer')
            if prepare:continue
            os.environ['NUT_PROVENANCE_REPORT_DIR']=str(target/'originals')
            try:status=S.stage(target,'generation',prefix(pins)+selection['go_argv']+[fork],cwd=str(ROOT),stdin=subprocess.DEVNULL)
            finally:os.environ.pop('NUT_PROVENANCE_REPORT_DIR',None)
            if status:raise ValueError('Original NUT regeneration failed: '+fork)
            compiler_identity(target)
            validate_fork(target,selection['entries'][fork],fork)
        if M.inputs()!=before:raise ValueError('NUT verification changed source inputs')
        status=0
    except Exception as error:
        status=status or 1;errors.append(str(error));print(error,file=sys.stderr)
    finally:S.write(directory/'final.json',{'exit_code':status,'report_errors':errors,'tests':0,'original_sha256':seal(directory)})
    return status


def git_inventory(data):
    result={}
    for entry in data.split(b'\0'):
        if not entry:continue
        header,name=entry.decode().split('\t',1);mode,revision,stage=header.split()
        if (stage!='0' or mode not in ('100644','100755','120000','160000') or not re.fullmatch('[0-9a-f]{40}',revision)
                or name in result or Path(name).is_absolute() or '..' in Path(name).parts):raise ValueError('Invalid complete historical NUT Git inventory')
        result[name]={'mode':mode,'object':revision}
    if not result:raise ValueError('Empty historical NUT Git inventory')
    return result


def submodules(text):
    result={}
    for line in text.splitlines():
        match=re.fullmatch(r'([ -])([0-9a-f]{40}) ([^ ]+)(?: \(.*\))?',line)
        if (not match or match[3] in result or Path(match[3]).is_absolute() or '..' in Path(match[3]).parts):
            raise ValueError('Missing, duplicate or changed historical NUT submodule')
        result[match[3]]={'revision':match[2],'initialized':match[1]==' '}
    if not result:raise ValueError('Empty historical NUT submodule selection')
    return result


def validate_fork(directory,entry,fork):
    stage=G.read(directory/'generation.stage.json');raw=directory/'originals'
    if stage['exit_code']!=0 or stage['argv']!=prefix(entry['historical_tools'])+['go','run','./ops/scripts/nut-provenance-verify',fork]:
        raise ValueError('Wrong or incomplete original NUT invocation')
    metadata=G.read(raw/'source.json')
    if metadata!={'fork':fork,'entry':{k.title():entry[k] for k in ('bundle','hash','commit')},'generator_argv':['just','generate-nut-bundle']}:
        raise ValueError('Wrong historical NUT source or generator')
    for name in ('locked-bundle.json','regenerated-bundle.json'):
        if 'sha256:'+S.digest(raw/name)!=entry['hash']:raise ValueError('Original NUT regeneration bytes differ from lock')
    if S.digest(raw/'mise.toml')!=entry['historical_mise_sha256']:raise ValueError('Wrong historical NUT generator configuration')
    inventory=git_inventory((raw/'tracked-stage.bin').read_bytes())
    for name,path in [('mise.toml','mise.toml'),('contracts.justfile','packages/contracts-bedrock/justfile'),
                      ('foundry.toml','packages/contracts-bedrock/foundry.toml')]:
        data=(raw/name).read_bytes();blob=hashlib.sha1(b'blob '+str(len(data)).encode()+b'\0'+data).hexdigest()
        if inventory.get(path)!={'mode':'100644','object':blob}:
            raise ValueError('Retained NUT generator input differs from historical Git source')
    module_rows=submodules((raw/'submodules.txt').read_text())
    originals=G.read(raw/'submodule-inventories.json');module_inventories={}
    if set(originals)!={name for name,row in module_rows.items() if row['initialized']}:
        raise ValueError('Missing or extra original historical submodule inventory')
    for name,row in originals.items():
        if row['head']!=module_rows[name]['revision'] or base64.b64decode(row['status'],validate=True):
            raise ValueError('Changed historical NUT submodule source')
        module_inventories[name]=git_inventory(base64.b64decode(row['tracked_stage'],validate=True))
    # Each recorded gitlink is bound to its immediate parent's original index.
    # The actual Forge generator initializes contract libraries recursively;
    # unrelated root repositories remain explicitly uninitialized.
    for parent,index in [('',inventory),*module_inventories.items()]:
        for name,row in index.items():
            if row['mode']!='160000':continue
            path=str(Path(parent)/name)
            if path not in module_rows or module_rows[path]['revision']!=row['object']:
                raise ValueError('Missing or mismatched historical NUT gitlink')
            if path.startswith('packages/contracts-bedrock/lib/') and not module_rows[path]['initialized']:
                raise ValueError('Historical NUT contract gitlink was never initialized')
    for name in module_rows:
        parents=[p for p in module_inventories if name.startswith(p+'/')]
        parent=max(parents,key=len) if parents else ''
        relative=name[len(parent)+1:] if parent else name
        if (module_inventories[parent] if parent else inventory).get(relative)!={'mode':'160000','object':module_rows[name]['revision']}:
            raise ValueError('Extra historical NUT submodule outside original gitlinks')
    log=(directory/'generation.log').read_text()
    if log.count('PASS: regenerated bundle matches committed bundle')!=1 or log.count('PASS: bundle hash matches lock')!=1:
        raise ValueError('Missing or duplicate original NUT verdict')
    compilers=Counter(re.findall(r'Compiling ([0-9]+) files with Solc ([0-9.]+)',log))
    if not compilers:raise ValueError('Original full NUT regeneration compiled no contracts')
    identities=G.read(directory/'compilers.json')
    if (set(identities)!={version for _,version in compilers}
            or any(set(row)!={'version','binary_sha256','binary_path'} or not re.fullmatch('[0-9a-f]{64}',row['binary_sha256'])
                   or not Path(row['binary_path']).is_absolute() or not re.search(r'^Version: '+re.escape(version)+r'\+',row['version'],re.M)
                   for version,row in identities.items())):
        raise ValueError('Missing, extra or corrupt original NUT compiler identity')
    tools=G.read(directory/'tools.json')
    if (set(tools)!={'forge','just','go'} or any(set(row)!={'version','binary_sha256','binary_path'}
            or not re.fullmatch('[0-9a-f]{64}',row['binary_sha256']) or not Path(row['binary_path']).is_absolute() for row in tools.values())
            or tools['just']['version']!='just '+entry['historical_tools']['just']
            or not re.search(r'^forge Version: '+re.escape(entry['historical_tools']['forge'])+r'(?:-|$)',tools['forge']['version'],re.M)):
        raise ValueError('Missing or wrong original NUT generator tool identity')
    return {'source':metadata,'git_inventory':inventory,'submodules':module_rows,'submodule_inventories':module_inventories,
            'compiler_partitions':[{'files':int(files),'solc':version,'invocations':count} for (files,version),count in sorted(compilers.items())]}


def full_selection(directory):
    selection=G.read(directory/'selection.json')
    locks=tomllib.loads((directory/'fork_lock.toml').read_text())
    if (selection['mode']!='full' or not locks or selection['excluded']
            or set(selection['entries'])!=set(locks)
            or set(selection['selected_forks'])!=set(locks)
            or len(selection['selected_forks'])!=len(set(selection['selected_forks']))
            or any({key:entry[key] for key in ('bundle','hash','commit')}!=locks[fork]
                   for fork,entry in selection['entries'].items())):
        raise ValueError('Missing or different full NUT workload selection')
    base_keys=('base_revision','base_lock_available','base_lock_exit_code')
    base={key:selection[key] for key in base_keys}
    if (base['base_revision'] is not None and not re.fullmatch('[0-9a-f]{40}',base['base_revision'])
            or type(base['base_lock_available']) is not bool or type(base['base_lock_exit_code']) is not int
            or not 0<=base['base_lock_exit_code']<=255
            or base['base_lock_available'] and base['base_revision'] is None
            or base['base_lock_available']!=(base['base_lock_exit_code']==0)):
        raise ValueError('Invalid original NUT base discovery provenance')
    if base['base_lock_available']:
        tomllib.loads((directory/'base-fork-lock.toml').read_text())
    base.update(lock_sha256=S.digest(directory/'base-fork-lock.toml'),
                stderr_sha256=S.digest(directory/'base-discovery.stderr'),affects_full_selection=False)
    # The explicit full replay never consults base hashes to select work.
    # Preserve provider base observations; compare every effective input.
    return {key:value for key,value in selection.items() if key not in base_keys},base


def compare(circle,native,output):
    result={'passed':False,'errors':[],'full_original_comparison':True,'coverage':{}}
    try:
        originals(circle,'circleci');originals(native,'rwx')
        a,b=G.read(circle/'settings.json'),G.read(native/'settings.json')
        for key in ('source_sha','branch','input_sha256','environment','go_version','prepare_only'):
            if a[key]!=b[key]:raise ValueError('NUT provider settings differ: '+key)
        if a['provider']!='circleci' or b['provider']!='rwx' or a['prepare_only']:raise ValueError('Wrong original NUT provider or preparation-only evidence')
        selection,base_a=full_selection(circle);other,base_b=full_selection(native)
        result['base_discovery']={'circleci':base_a,'rwx':base_b}
        if selection!=other:
            raise ValueError('Missing or different full NUT workload selection')
        if (circle/'fork_lock.toml').read_bytes()!=(native/'fork_lock.toml').read_bytes():
            raise ValueError('Original complete NUT lock inputs differ')
        for fork,entry in selection['entries'].items():
            left,right=circle/'forks'/fork,native/'forks'/fork
            ca,cb=validate_fork(left,entry,fork),validate_fork(right,entry,fork)
            if ca!=cb:raise ValueError('Complete original NUT build selection differs: '+fork)
            for name in ('mise.toml','contracts.justfile','foundry.toml','locked-bundle.json','regenerated-bundle.json','tracked-stage.bin','worktree-status.txt','submodule-inventories.json'):
                if (left/'originals'/name).read_bytes()!=(right/'originals'/name).read_bytes():raise ValueError('Original NUT input or artifact differs: '+fork+'/'+name)
            tools_a,tools_b=G.read(left/'tools.json'),G.read(right/'tools.json')
            if tools_a['go']['version']!=a['go_version'] or tools_b['go']['version']!=b['go_version']:
                raise ValueError('Original NUT verifier Go differs from settings')
            if {k:{n:v for n,v in row.items() if n!='binary_path'} for k,row in tools_a.items()}!={k:{n:v for n,v in row.items() if n!='binary_path'} for k,row in tools_b.items()}:
                raise ValueError('Original NUT generator binary or toolchain differs')
            compilers_a,compilers_b=G.read(left/'compilers.json'),G.read(right/'compilers.json')
            if {k:{n:v for n,v in row.items() if n!='binary_path'} for k,row in compilers_a.items()}!={k:{n:v for n,v in row.items() if n!='binary_path'} for k,row in compilers_b.items()}:
                raise ValueError('Original NUT Solidity compiler binaries differ')
            result['coverage'][fork]={'recorded_commit':entry['commit'],'bundle_sha256':entry['hash'],'historical_tools':entry['historical_tools'],
                    'historical_source_files':len(ca['git_inventory']),'submodules':len(ca['submodules']),'compiler_partitions':ca['compiler_partitions'],'outcome':'passed','tests':0,'retries':0}
        result['passed']=True
    except Exception as error:result['errors'].append(str(error))
    S.write(output,result);return 0 if result['passed'] else 1


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--provider',choices=('circleci','rwx'))
    parser.add_argument('--full',action='store_true');parser.add_argument('--prepare-tools',action='store_true')
    parser.add_argument('--module-artifact',type=Path);parser.add_argument('--tool-artifact',type=Path)
    parser.add_argument('--compare',nargs=2,type=Path);parser.add_argument('--output',type=Path)
    args=parser.parse_args()
    if args.compare:
        if args.output is None:parser.error('--compare requires --output')
        sys.exit(compare(*args.compare,args.output))
    if args.provider is None:parser.error('Execution requires --provider')
    sys.exit(execute(args.provider,args.full,args.prepare_tools,args.module_artifact,args.tool_artifact))
