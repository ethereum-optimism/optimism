#!/usr/bin/env python3
"""Build the complete SP1 guest workload and run its fresh original checks."""
import argparse
import collections
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import shlex
import struct
import subprocess
import sys
import tomllib
import xml.etree.ElementTree as ET

ROOT = Path(__file__).resolve().parents[3]
RUST = ROOT / 'rust'
SP1 = RUST / 'kona/sp1'
GUEST = 'kona/sp1/programs/Cargo.toml'
RANGE = 'kona/sp1/crates/range-vkeys/Cargo.toml'
TEST_CONFIG_MARKER = b'KONA_SP1_UNSAFE_TEST_CONFIG_FALLBACK{fd6d88e711058eef5eff1512237c8ad3}'
REPORTS = ROOT / '.ci/sp1-guest'
TOOL_INPUTS = ('mise.toml', 'rust/Cargo.toml', 'rust/Cargo.lock',
    'rust/kona/sp1/programs/Cargo.toml', 'rust/kona/sp1/programs/Cargo.lock',
    'rust/kona/sp1/justfile', 'rust/rust-toolchain.toml', 'rust/.cargo/config.toml',
    'ops/ci/runtime/sp1-guest-toolchain.sh', 'ops/ci/runtime/sp1-guest.py', 'ops/ci/runtime/sp1-guest-native-build.py',
    'ops/ci/runtime/rust-target-cache.py', 'ops/ci/runtime/rust-workspace-report.py', 'ops/ci/runtime/ci-report.py')


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    value = importlib.util.module_from_spec(spec); spec.loader.exec_module(value)
    return value


S = helper('rust-workspace-report')


def inputs(): return helper('main-checks').inputs()


def read(path): return json.loads(Path(path).read_text())
def command(*args, cwd=ROOT): return subprocess.check_output(args, cwd=cwd, text=True).strip()
def seal(path): return {str(p.relative_to(path)): S.digest(p) for p in sorted(path.rglob('*')) if p.is_file()}


def programs(source=None):
    source = (SP1 / 'justfile').read_text() if source is None else source
    recipe = re.search(r'^build-elfs-native(?: features="")?:\n', source, re.M)
    if not recipe: raise ValueError('Missing authoritative native SP1 build recipe')
    body = source[recipe.end():]
    match = re.search(r'^    for name in ([a-z0-9 -]+); do$', body, re.M)
    if not match: raise ValueError('Missing authoritative complete SP1 ELF loop')
    names = shlex.split(match[1])
    if (not names or len(names) != len(set(names)) or not {'super-range', 'super-aggregation'} <= set(names)
            or any(not re.fullmatch('[a-z0-9]+(?:-[a-z0-9]+)*', n) for n in names)):
        raise ValueError('Empty, duplicate or invalid original SP1 ELF selection')
    return names


def tools():
    version = tomllib.loads((ROOT / 'mise.toml').read_text())['tools']['github:succinctlabs/sp1']['version']
    result = {name: command(*argv) for name, argv in {
        'rustc': ['rustc', '--version'], 'cargo': ['cargo', '--version'],
        'cargo-prove': ['cargo', 'prove', '--version'], 'succinct-rustc': ['rustc', '+succinct', '--version'],
        'nightly-rustfmt': ['cargo', '+' + tomllib.loads((ROOT / 'mise.toml').read_text())['tools']['rust'][1]['version'], 'fmt', '--version'],
        'just': ['just', '--version'], 'protoc': ['protoc', '--version']}.items()}
    directory=Path(command('mise','where','github:succinctlabs/sp1')).resolve()
    binary=Path(shutil.which('cargo-prove') or '/missing-cargo-prove').resolve()
    if directory.name!=version or not binary.is_file() or not binary.is_relative_to(directory):
        raise ValueError('Cargo-prove executable does not come from the pinned mise installation')
    # This release reports a Git descriptor ('sp1'), not the numeric SDK version.
    # Retain that original descriptor and verify the actual installed binary bytes.
    result['cargo-prove-sha256']=S.digest(binary)
    rustc = Path(command('rustup', 'which', '--toolchain', 'succinct', 'rustc'))
    if not rustc.is_file(): raise ValueError('Missing actual Succinct compiler')
    result['succinct-rustc-sha256'] = S.digest(rustc)
    library = rustc.parent.parent/'lib'
    result['succinct-library-sha256'] = {str(p.relative_to(library)):S.digest(p) for p in sorted(library.rglob('*')) if p.is_file()}
    if not result['succinct-library-sha256']:raise ValueError('Missing actual Succinct linker and standard-library inputs')
    result['sp1-version'] = version
    return result


def toolchain(provider):
    path = REPORTS / 'toolchain'; shutil.rmtree(path, ignore_errors=True); path.mkdir(parents=True)
    status, errors = 1, []
    try:
        inputs = {name: S.digest(ROOT / name) for name in TOOL_INPUTS}
        S.write(path / 'inputs.json', inputs)
        for name in TOOL_INPUTS:
            target = path / 'source' / name
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(ROOT / name, target)
        status = S.stage(path, 'install', ['bash', 'ops/ci/runtime/sp1-guest-toolchain.sh'], cwd=str(ROOT), stdin=subprocess.DEVNULL)
        if status: raise ValueError('Original SP1 toolchain pin checks or installation failed')
        S.write(path / 'tools.json', tools())
    except Exception as error:
        errors.append(str(error)); status = status or 1; print(str(error), file=sys.stderr)
    finally:
        S.write(path / 'final.json', {'phase': 'toolchain', 'exit_code': status, 'report_errors': errors, 'original_sha256': seal(path)})
    return status


def verified_toolchain():
    path = REPORTS / 'toolchain'; final = read(path / 'final.json')
    if final['exit_code'] or final['report_errors']: raise ValueError('Failed SP1 toolchain preparation')
    verify_seals(path, final)
    if read(path / 'inputs.json') != {name:S.digest(ROOT / name) for name in TOOL_INPUTS}:
        raise ValueError('Stale SP1 toolchain settings or pin inputs')
    if {str(p.relative_to(path/'source')): S.digest(p) for p in (path/'source').rglob('*') if p.is_file()} != read(path/'inputs.json'):
        raise ValueError('Incomplete or corrupt original SP1 pin-check sources')
    result = read(path / 'tools.json')
    if result != tools(): raise ValueError('Changed actual SP1 compiler or toolchain')
    return result


def verify_seals(path, final):
    if any(p.is_symlink() for p in path.rglob('*')): raise ValueError('Unsafe SP1 original report link')
    hashes = final['original_sha256']
    for name, value in hashes.items():
        relative = Path(name)
        if relative.is_absolute() or '..' in relative.parts or not re.fullmatch('[0-9a-f]{64}', value):
            raise ValueError('Unsafe SP1 original report manifest')
        target = path / relative
        if target.is_symlink() or not target.is_file() or S.digest(target) != value:
            raise ValueError('Missing or corrupt original SP1 file: ' + name)
    if seal(path).keys() != hashes.keys() | {'final.json'}: raise ValueError('Extra or unsealed original SP1 files')


def binding(provider, phase, features=""):
    if features not in ("", "test-config-fallback"): raise ValueError("Unsupported SP1 guest features")
    sha = command('git', 'rev-parse', 'HEAD')
    if sha != (os.environ.get('CI_COMMIT_SHA') or os.environ.get('CIRCLE_SHA1') or sha) or not re.fullmatch('[0-9a-f]{40}', sha):
        raise ValueError('Wrong SP1 source revision')
    if subprocess.call(['git', 'diff', '--quiet'], cwd=ROOT) or subprocess.call(['git', 'diff', '--cached', '--quiet'], cwd=ROOT):
        raise ValueError('SP1 tracked source differs from its Git revision')
    branch = os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH')
    if not branch or any(os.environ.get(n) for n in ('KONA_CUSTOM_CONFIGS_DIR', 'KONA_CUSTOM_CONFIGS', 'KONA_SP1_GIT_SHA', 'CARGO_ENCODED_RUSTFLAGS')):
        raise ValueError('Missing SP1 branch or overridden original source/compiler inputs')
    if os.environ.get('CARGO_INCREMENTAL') != '0': raise ValueError('SP1 compiler caching requires the original nonincremental setting')
    return {'suite':'sp1-guest', 'phase':phase, 'provider':provider, 'source_sha':sha, 'branch':branch,
        'features':features, 'elf_source_sha':sha+('-test' if features else ''),
        'workspace_root':str(ROOT), 'cargo_home':os.environ.get('CARGO_HOME', str(Path.home()/'.cargo')),
        'input_sha256':inputs(), 'tools':verified_toolchain(), 'programs':programs(),
        'build_rustflags':'', 'check_rustflags':'-Dwarnings', 'vkey_prover':'cpu', 'incremental':'0', 'rerun_fails':0,
        'rwx_run_id':os.environ.get('RWX_RUN_ID'), 'rwx_task_attempt':os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}


def workspace(metadata):
    rows = {p['id']:p for p in metadata['packages']}; members = metadata['workspace_members']
    if not members or len(members) != len(set(members)) or not set(members) <= rows.keys():
        raise ValueError('Empty, duplicate or missing complete SP1 workspace selection')
    result = {rows[k]['name']:rows[k] for k in members}
    if len(result) != len(members): raise ValueError('Duplicate selected SP1 package name')
    return result


def elf_record(settings, directory):
    if directory.is_symlink() or any(p.is_symlink() or not p.is_file() for p in directory.iterdir()):
        raise ValueError('Unsafe original guest ELF inventory')
    names = settings['programs']; expected = {n+'-elf' for n in names} | {'vkeys.toml'}
    files = {p.name for p in directory.iterdir() if p.name != '.gitignore'}
    if files != expected: raise ValueError('Missing or extra complete original guest ELF inventory')
    values = tomllib.loads((directory / 'vkeys.toml').read_text())
    if (set(values) != {'git_sha','sp1_toolchain_tag',*names} or values['git_sha'] != settings['elf_source_sha']
            or values['sp1_toolchain_tag'] != 'v'+settings['tools']['sp1-version']):
        raise ValueError('Wrong source, version or complete vkeys manifest selection')
    result = {}
    for name in names:
        path = directory / (name+'-elf'); data = path.read_bytes()
        if (path.is_symlink() or len(data)<64 or data[:6] != b'\x7fELF\x02\x01'
                or struct.unpack_from('<HH', data, 16) != (2,243)):
            raise ValueError('Missing or corrupt original RISC-V 64-bit executable')
        markers = set(m.decode() for m in re.findall(rb'KONA_SP1_BUILD\{git_sha=[0-9A-Za-z._-]*\}', data))
        if markers != {'KONA_SP1_BUILD{git_sha='+settings['elf_source_sha']+'}'}:
            raise ValueError('Guest ELF has missing, extra or stale source markers')
        if name == 'super-range' and (TEST_CONFIG_MARKER in data) != bool(settings['features']):
            raise ValueError('Guest ELF test-config-fallback marker differs from selected features')
        if not re.fullmatch('0x[0-9a-fA-F]{64}', values[name]): raise ValueError('Invalid complete guest verification key')
        result[name] = {'sha256':S.digest(path), 'size':len(data), 'vkey':values[name], 'markers':sorted(markers)}
    return result


class StageFailure(ValueError):
    def __init__(self, name, status):
        self.status=status
        super().__init__('Original SP1 stage failed: '+name+' (exit '+str(status)+')')


def stage(path, name, argv, cwd=RUST, json_output=False):
    status = S.stage(path, name, argv, stdout_json=json_output, cwd=str(cwd), stdin=subprocess.DEVNULL)
    if status: raise StageFailure(name,status)


def cache_start(path, provider, phase, features=""):
    if provider != 'rwx': return False
    cache = ROOT / '.ci/sp1-cache'
    target = cache / (phase + '-target')
    cargo_home = cache/'cargo'
    if phase=='elf':
        native=helper('sp1-guest-native-build')
        if ROOT!=native.WORKSPACE:raise ValueError('Native SP1 ELF compilation must use the verified canonical workspace')
        cache=Path(os.environ['SP1_GUEST_NATIVE_CACHE_ROOT'])
        if not cache.is_absolute() or cache.name!='sp1-cache' or cache.parent.name!='.ci' or cache.is_relative_to(ROOT):
            raise ValueError('Invalid isolated native SP1 compiler cache root')
        target=native.TARGET;cargo_home=native.CARGO
    mode = os.environ.get('SP1_GUEST_TARGET_MODE', 'keep')
    if mode not in ('keep', 'sccache-only'): raise ValueError('Invalid SP1 compiler-target probe mode')
    identity={'phase':phase,'features':features,'tools':tools(),'rustflags':'' if phase=='elf' else '-Dwarnings','incremental':'0'}
    identity_sha256=hashlib.sha256(json.dumps(identity,sort_keys=True).encode()).hexdigest()
    identity_path=target/'.sp1-compiler-inputs.json'
    if not identity_path.is_file() or read(identity_path)!=identity:
        shutil.rmtree(target,ignore_errors=True)
    if mode == 'sccache-only': shutil.rmtree(target, ignore_errors=True)
    target.mkdir(parents=True,exist_ok=True);S.write(identity_path,identity)
    S.write(path/'cache-settings.json', {'phase':phase, 'target_mode':mode,
        'target_directory':str(target), 'sccache_version':command('sccache','--version'),
        'compiler_input_sha256':identity_sha256,'compiler_inputs':identity})
    os.environ.update(CARGO_HOME=str(cargo_home), CARGO_TARGET_DIR=str(target),
        RUSTC_WRAPPER='sccache', SCCACHE_DIR=str(cache/(phase+'-sccache')/identity_sha256),
        SCCACHE_BASEDIRS=str(ROOT), SCCACHE_CACHE_SIZE='10G', SCCACHE_IDLE_TIMEOUT='0')
    for name in ('CARGO_HOME', 'CARGO_TARGET_DIR', 'SCCACHE_DIR'):
        Path(os.environ[name]).mkdir(parents=True, exist_ok=True)
    stage(path, 'sccache-start', ['sccache', '--start-server'], ROOT)
    stage(path, 'sccache-zero', ['sccache', '--zero-stats'], ROOT)
    stage(path, 'cache-prepare', ['python3', 'ops/ci/runtime/rust-target-cache.py', 'prepare'], ROOT)
    return True


def cache_finish(path, active, status, errors):
    if not active: return status
    for name, argv, json_output in (
            ('sccache-stats', ['sccache', '--show-stats', '--stats-format', 'json'], True),
            ('sccache-stop', ['sccache', '--stop-server'], False)):
        try: stage(path, name, argv, ROOT, json_output)
        except Exception as error:
            errors.append(str(error)); status = status or getattr(error, 'status', 1)
    if not status:
        try: stage(path, 'cache-commit', ['python3', 'ops/ci/runtime/rust-target-cache.py', 'commit'], ROOT)
        except Exception as error:
            errors.append(str(error)); status = getattr(error, 'status', 1)
    return status


def build(provider, features=""):
    if provider=='rwx' and ROOT!=helper('sp1-guest-native-build').WORKSPACE:
        return helper('sp1-guest-native-build').execute(features)
    path = REPORTS/'dependency'; shutil.rmtree(path, ignore_errors=True); path.mkdir(parents=True)
    status, errors, cached = 1, [], False
    try:
        if os.environ.get('RUSTFLAGS'):raise ValueError('Overridden original SP1 build compiler flags')
        os.environ['SP1_PROVER']='cpu'
        cached = cache_start(path, provider, 'elf', features)
        settings = binding(provider,'build', features); S.write(path/'settings.json',settings)
        shutil.copytree(REPORTS/'toolchain',path/'toolchain')
        stage(path,'lock-before',['just','check-sp1-guest-lock'])
        stage(path,'guest-workspace',['cargo','metadata','--manifest-path',GUEST,'--locked','--all-features','--format-version','1'],json_output=True)
        selected = workspace(read(path/'guest-workspace.json'))
        manifests = {p['manifest_path'] for p in selected.values()}
        if not {str(SP1/'programs'/p/'Cargo.toml') for p in settings['programs']} <= manifests:
            raise ValueError('Original ELF selection missing from complete guest workspace')
        stage(path,'build',['just','build-elfs-native',features],SP1)
        elves = elf_record(settings,SP1/'elf')
        for name in settings['programs']:
            stage(path,'vkey-'+name,['cargo','prove','vkey','--elf','elf/'+name+'-elf'],SP1)
            found = re.findall('0x[0-9a-fA-F]{64}',(path/('vkey-'+name+'.log')).read_text())
            if found != [elves[name]['vkey']]: raise ValueError('Original ELF and verification-key bytes differ')
        for name in (*[p+'-elf' for p in settings['programs']],'vkeys.toml'):
            target=path/'files'/name;target.parent.mkdir(exist_ok=True);shutil.copyfile(SP1/'elf'/name,target)
        S.write(path/'elfs.json',elves)
        after=inputs();S.write(path/'inputs-after.json',after)
        if after != settings['input_sha256']: raise ValueError('SP1 build changed tracked source or lockfiles')
        status=0
    except Exception as error:
        status=getattr(error,'status',status or 1)
        errors.append(str(error));print(str(error),file=sys.stderr)
    finally:
        status = cache_finish(path, cached, status, errors)
        S.write(path/'final.json',{'phase':'build','exit_code':status,'report_errors':errors,'tests':0,'original_sha256':seal(path)})
    return status


def restore_artifact(directory, settings):
    final=read(directory/'final.json')
    if final['exit_code'] or final['report_errors'] or final['phase']!='build' or final['tests']!=0:
        raise ValueError('Failed or false SP1 ELF producer')
    verify_seals(directory,final)
    original=read(directory/'settings.json')
    for name in ('source_sha','branch','input_sha256','tools','programs','features','elf_source_sha','build_rustflags','check_rustflags','vkey_prover','incremental','rerun_fails'):
        if original[name] != settings[name]: raise ValueError('Stale SP1 producer revision, source or settings: '+name)
    if original['provider']=='rwx':helper('sp1-guest-native-build').verify(directory,original,settings['workspace_root'])
    if read(directory/'elfs.json') != elf_record(settings,directory/'files'):
        raise ValueError('Wrong or incomplete actual SP1 producer binaries')
    (SP1/'elf').mkdir(exist_ok=True)
    for p in (directory/'files').iterdir(): shutil.copyfile(p,SP1/'elf'/p.name)
    return original


def normalize_name(name, root, cargo_home):
    name=re.sub(r'(\(line \d+\)) - compile$',r'\1',name)
    return name.replace(root,'<workspace>').replace(cargo_home,'<cargo>')


def libtest(text, metadata, listing, root, cargo_home, allow_partial=False):
    packages=workspace(metadata);targets=collections.defaultdict(list)
    expected_suites=set()
    for package in packages.values():
        for target in package['targets']:
            if target.get('test') and set(target['kind']) & {'bin','lib','test','example'}:
                key=package['name']+':'+target['name']+':unit';expected_suites.add(key)
                targets[('unit',target['name'].replace('-','_'))].append(key)
            if target.get('doctest') and 'lib' in target['kind']:
                key=package['name']+':'+target['name']+':doc';expected_suites.add(key)
                targets[('doc',target['name'].replace('-','_'))].append(key)
    suites={};current=None;active=False;rows={};advertised=None
    def finish():
        if current is not None:
            if current in suites or (not allow_partial and (advertised is None or len(rows)!=advertised)):
                raise ValueError('Duplicate or incomplete original libtest suite')
            suites[current]={'cases':dict(rows),'count':advertised}
    for line in text.splitlines():
        match=re.match(r'^\s*Running (?:unittests .+|tests/\S+) \((.+)\)$',line)
        doc=re.match(r'^\s*Doc-tests (\S+)$',line)
        if match or doc:
            finish();rows={};advertised=None;active=False
            name=re.sub(r'-[0-9a-f]{16}$','',Path(match[1]).name) if match else doc[1]
            candidates=targets[('unit' if match else 'doc',name.replace('-','_'))]
            if len(candidates)!=1:raise ValueError('Missing or ambiguous selected Rust target: '+name)
            current=candidates[0]
        elif listing:
            case=re.fullmatch(r'(.+): (test|benchmark)',line)
            total=re.fullmatch(r'(\d+) tests?, (\d+) benchmarks?',line)
            if case:
                if current is None:raise ValueError('Original test discovery lacks target identity')
                name=normalize_name(case[1],root,cargo_home)
                if name in rows:raise ValueError('Duplicate original test discovery')
                rows[name]=case[2]
            elif total:
                advertised=int(total[1])+int(total[2])
        else:
            total=re.fullmatch(r'running (\d+) tests?',line)
            if total:advertised=int(total[1]);active=True
            elif line=='failures:':active=False
            case=re.fullmatch(r'test (.+) \.\.\. (ok|FAILED|ignored.*)',line)
            if case and active:
                name=normalize_name(case[1],root,cargo_home)
                if name in rows:raise ValueError('Duplicate original Rust verdict')
                rows[name]='pass' if case[2]=='ok' else 'fail' if case[2]=='FAILED' else 'skip'
    finish()
    if (not suites.keys()<=expected_suites or (not allow_partial and suites.keys()!=expected_suites)):
        raise ValueError('Missing or extra complete Rust target suites')
    return suites


def junit(record):
    tree=ET.Element('testsuite',name=record['suite'],tests=str(len(record['cases'])))
    for row in record['cases']:
        case=ET.SubElement(tree,'testcase',classname=row['suite'],name=row['name'])
        if row['outcome']!='pass':ET.SubElement(case,'skipped' if row['outcome']=='skip' else 'failure',message=row['outcome'])
    return ET.tostring(tree,encoding='utf-8',xml_declaration=True)


def coverage(path, phase, metadata, settings, emit=True):
    expected=libtest((path/(phase+'-list.log')).read_text(),metadata,True,settings['workspace_root'],settings['cargo_home'])
    status=read(path/(phase+'.stage.json'))['exit_code']
    actual=libtest((path/(phase+'.log')).read_text(),metadata,False,settings['workspace_root'],settings['cargo_home'],allow_partial=status!=0)
    cases=[];missing=[];extra=[]
    for suite in expected:
        observed=actual.get(suite,{'cases':{}})['cases']
        missing += [{'suite':suite,'name':n} for n in sorted(expected[suite]['cases'].keys()-observed.keys())]
        extra += [{'suite':suite,'name':n} for n in sorted(observed.keys()-expected[suite]['cases'].keys())]
        cases += [{'suite':suite,'name':n,'outcome':o,'retries':0,'original_log':phase+'.log'} for n,o in sorted(observed.items())]
    record={'suite':phase,'packages':sorted(workspace(metadata)),'targets':expected,'cases':cases,
            'missing':missing,'extra':extra,'missing_target_suites':sorted(expected.keys()-actual.keys()),
            'outcomes':dict(collections.Counter(c['outcome'] for c in cases))}
    if emit:
        S.write(path/(phase+'-coverage.json'),record)
        (path/(phase+'.junit.xml')).write_bytes(junit(record))
    if missing or extra or record['missing_target_suites']:raise ValueError('Missing or extra complete original Rust cases/targets')
    return record


def checks(provider, artifact, features=""):
    path=REPORTS/'run';shutil.rmtree(path,ignore_errors=True);path.mkdir(parents=True)
    status,errors,cached=1,[],False
    try:
        cached = cache_start(path, provider, 'checks', features)
        settings=binding(provider,'checks', features);S.write(path/'settings.json',settings)
        restore_artifact(artifact,settings);shutil.copytree(artifact,path/'producer')
        shutil.copytree(REPORTS/'toolchain',path/'toolchain')
        os.environ['SP1_PROVER']='cpu'
        for name in settings['programs']:
            stage(path,'vkey-'+name,['cargo','prove','vkey','--elf','elf/'+name+'-elf'],SP1)
            if re.findall('0x[0-9a-fA-F]{64}',(path/('vkey-'+name+'.log')).read_text()) != [read(artifact/'elfs.json')[name]['vkey']]:
                raise ValueError('Verified runtime ELF differs from the selected actual verification key')
        os.environ['RUSTFLAGS']='-Dwarnings'
        stage(path,'guest-workspace',['cargo','metadata','--manifest-path',GUEST,'--locked','--all-features','--format-version','1'],json_output=True)
        stage(path,'guest-list',['cargo','test','--manifest-path',GUEST,'--workspace','--locked','--all-features','--','--list'])
        stage(path,'guest',['just','test-sp1-guest'])
        coverage(path,'guest',read(path/'guest-workspace.json'),settings)
        stage(path,'lint',['just','lint-sp1-guest'])
        stage(path,'range-workspace',['cargo','metadata','--manifest-path',RANGE,'--locked','--format-version','1'],json_output=True)
        stage(path,'range-list',['cargo','test','--manifest-path',RANGE,'--locked','--','--list'])
        stage(path,'range',['just','check-range-vkeys'])
        coverage(path,'range',read(path/'range-workspace.json'),settings)
        after=inputs();S.write(path/'inputs-after.json',after)
        if after!=settings['input_sha256']:raise ValueError('Fresh SP1 checks changed tracked sources or lockfiles')
        if elf_record(settings,SP1/'elf')!=read(artifact/'elfs.json'):raise ValueError('Runtime checks changed verified SP1 ELFs or keys')
        status=0
    except Exception as error:
        status=getattr(error,'status',status or 1)
        errors.append(str(error));print(str(error),file=sys.stderr)
    finally:
        for phase in ('guest','range'):
            if all((path/n).is_file() for n in ('settings.json',phase+'.log',phase+'-list.log',phase+'-workspace.json')) and not (path/(phase+'-coverage.json')).exists():
                try:coverage(path,phase,read(path/(phase+'-workspace.json')),read(path/'settings.json'))
                except Exception as error:errors.append(str(error))
        status = cache_finish(path, cached, status, errors)
        S.write(path/'final.json',{'phase':'checks','exit_code':status,'report_errors':errors,'original_sha256':seal(path)})
    return status


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--provider',choices=('circleci','rwx'),required=True)
    parser.add_argument('--phase',choices=('toolchain','build','checks','full'),default='full')
    parser.add_argument('--features',choices=('', 'test-config-fallback'),default='')
    parser.add_argument('--artifact',type=Path,default=REPORTS/'dependency')
    args=parser.parse_args();os.chdir(ROOT)
    if args.phase=='toolchain':status=toolchain(args.provider)
    elif args.phase=='build':status=build(args.provider,args.features)
    elif args.phase=='checks':status=checks(args.provider,args.artifact,args.features)
    else:
        status=build(args.provider,args.features)
        if not status:status=checks(args.provider,args.artifact,args.features)
    sys.exit(status)
