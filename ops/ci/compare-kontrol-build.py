#!/usr/bin/env python3
"""Compare every original Kontrol input, summary, generated file and compiler output."""
import argparse
from collections import Counter
import copy
import json
from pathlib import Path
import re
import tarfile

import importlib.util


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    value = importlib.util.module_from_spec(spec); spec.loader.exec_module(value)
    return value


K = helper('kontrol-build'); A = helper('compare-contract-artifacts'); G = K.G
EMPTY = A.digest(b'')
COMPILER_PHASES = ('initial', 'summaries', 'proofs', 'final')


def compiler(directory):
    manifest = G.read(directory / 'files.json'); files = {}
    prefixes = tuple('packages/contracts-bedrock/' + p for p in ('forge-artifacts/', 'cache/', 'artifacts/build-info/'))
    with tarfile.open(directory / 'files.tar.gz', 'r|gz') as archive:
        for member in archive:
            path = Path(member.name)
            if (path.is_absolute() or '..' in path.parts or not member.isfile() or member.name in files or not member.name.startswith(prefixes)):
                raise ValueError('Unsafe, duplicate or foreign Kontrol compiler output')
            data = archive.extractfile(member).read()
            if A.digest(data) != manifest.get(member.name): raise ValueError('Corrupt original Kontrol compiler output')
            files[member.name] = data
    if not files or files.keys() != manifest.keys(): raise ValueError('Incomplete original Kontrol compiler archive')
    return files


def report(directory, provider, sha):
    final = G.read(directory / 'final.json'); hashes = final['original_sha256']; empty = []
    required = {'settings.json','selection.json','coverage.json','inputs-after.json','submodules.txt','runtime-image.json',
                'config.json','config.log','config.stage.json','summaries.log','summaries.stage.json','proofs.log','proofs.stage.json'}
    required |= {phase + '-compiler/' + name for phase in COMPILER_PHASES for name in ('files.json','files.tar.gz')}
    if final['exit_code'] or final['report_errors'] or not required <= hashes.keys(): raise ValueError('Failed or incomplete original Kontrol workload')
    for name, value in hashes.items():
        path = Path(name); target = directory / path
        if path.is_absolute() or '..' in path.parts or name == 'final.json' or not re.fullmatch('[0-9a-f]{64}', value):
            raise ValueError('Unsafe original Kontrol report manifest')
        if provider == 'circleci' and not target.exists() and value == EMPTY:
            target.parent.mkdir(parents=True,exist_ok=True); target.write_bytes(b''); empty.append({'path':name,'sha256':value})
        if target.is_symlink() or K.S.digest(target) != value: raise ValueError('Missing or corrupt original Kontrol report: ' + name)
    if K.seal(directory).keys() != hashes.keys() | {'final.json'}: raise ValueError('Extra or unsealed original Kontrol report')
    settings = G.read(directory / 'settings.json'); root = settings['workspace_root']
    if (settings['suite'],settings['source_sha'],settings['provider']) != ('kontrol-build',sha,provider):
        raise ValueError('Wrong original Kontrol suite/source/provider')
    if (not settings['branch'] or settings['profile'] != 'default' or settings['rerun_fails'] != 0
            or set(settings['environment']) != set(K.ENVIRONMENT) or settings['image'] != K.IMAGE.selection()
            or settings['kontrol_version'] != 'Kontrol version: ' + settings['image']['version']):
        raise ValueError('Changed original Kontrol settings, retries or image')
    if provider == 'rwx' and (not settings['rwx_run_id'] or str(settings['rwx_task_attempt']) != '1'):
        raise ValueError('Missing native Kontrol identity or unexpected retry')
    if K.F.SUBMODULES.revisions((directory / 'submodules.txt').read_text()) != settings['submodules']:
        raise ValueError('Changed original Kontrol submodule selection')
    selected = G.read(directory / 'selection.json'); before = settings['input_sha256']
    proofs = {n:h for n,h in before.items() if n.startswith(K.PROOFS + '/') and n.endswith('.sol')}
    expected = {'authority':'just kontrol-summary-full; just forge-build ./test/kontrol/proofs','variants':['default','fault-proofs'],
        'proof_sources':proofs,'generated':K.GENERATED,'tests':0}
    if selected != expected or not proofs or not set(sum(K.GENERATED.values(),[])) <= proofs.keys():
        raise ValueError('Missing, extra or changed complete Kontrol proof discovery')
    source_names = {str(p.relative_to(directory / 'source')) for p in (directory / 'source').rglob('*') if p.is_file()}
    if source_names != set(K.SOURCES) | proofs.keys(): raise ValueError('Incomplete original Kontrol source selection')
    for name in source_names:
        if K.S.digest(directory / 'source' / name) != before.get(name): raise ValueError('Kontrol source differs from bound revision input')
    if {p.name.removesuffix('.stage.json') for p in directory.glob('*.stage.json')} != {'config',*K.COMMANDS}:
        raise ValueError('Missing original Kontrol command or extra retry')
    for name, argv in {'config':['forge','config','--json'],**K.COMMANDS}.items():
        stage = G.read(directory / (name + '.stage.json'))
        if (stage['argv'] != argv or stage['cwd'] != root + '/packages/contracts-bedrock' or stage['exit_code'] != 0
                or stage.get('stdin') != 'devnull' or stage['log_sha256'] != K.S.digest(directory / (name + '.log'))):
            raise ValueError('Changed or failed original Kontrol invocation')
    for variant in selected['variants']:
        base = directory / 'variants' / variant
        if sorted(p.name for p in base.iterdir()) != ['deployment','generated','load-inputs']:
            raise ValueError('Missing or extra original Kontrol capture phase')
        names_file = 'packages/contracts-bedrock/deployments/kontrol' + ('-fp' if variant == 'fault-proofs' else '') + '.json'
        summary = 'DeploymentSummary' + ('FaultProofs' if variant == 'fault-proofs' else '')
        load = ['kontrol','load-state','--from-state-diff',summary,'snapshots/state-diff/Kontrol-31337.json',
            '--contract-names',names_file.removeprefix('packages/contracts-bedrock/') + 'Reversed',
            '--output-dir','test/kontrol/proofs/utils','--license','MIT']
        for phase in ('deployment','load-inputs','generated'):
            row = G.read(base / phase / 'manifest.json'); names = ['packages/contracts-bedrock/snapshots/state-diff/Kontrol-31337.json',names_file]
            if phase != 'deployment': names += [names_file + 'Reversed']
            if phase == 'generated': names += K.GENERATED[variant]
            if (row != {'variant':variant,'phase':phase,'source_sha':sha,'profile':'default','load_state_argv':load,
                        'files':K.seal(base / phase / 'files')} or set(row['files']) != set(names)):
                raise ValueError('Wrong source, settings or original Kontrol phase inventory')
        for name in K.GENERATED[variant]:
            if (directory / 'generated' / name).read_bytes() != (base / 'generated/files' / name).read_bytes():
                raise ValueError('Final Kontrol summary differs from original generated bytes')
    if sorted(p.name for p in (directory / 'variants').iterdir()) != selected['variants']:
        raise ValueError('Missing or extra original Kontrol summary variant')
    after = G.read(directory / 'inputs-after.json'); generated = set(sum(K.GENERATED.values(),[]))
    if after.keys() != before.keys(): raise ValueError('Missing tracked Kontrol input after execution')
    changes = {n:{'before':before[n],'after':after[n]} for n in before if before[n] != after[n]}
    if set(changes) - generated or any(after[n] != K.S.digest(directory / 'generated' / n) for n in generated):
        raise ValueError('Kontrol execution changed unrelated inputs or generated wrong final bytes')
    coverage = {'tests':0,'variants':selected['variants'],'proof_sources':len(proofs),'generated_files':4,
                'tracked_generated_changes':changes,'complete_original_commands':True}
    if G.read(directory / 'coverage.json') != coverage: raise ValueError('False original Kontrol workload coverage')
    image_metadata = G.read(directory / 'image/metadata.json')
    if (image_metadata['selection'] != settings['image'] or image_metadata['kontrol_version'] != settings['kontrol_version']
            or image_metadata['input_sha256'] != {n:before[n] for n in ('mise.toml','ops/ci/kontrol-image.json','ops/ci/kontrol-image.py')}):
        raise ValueError('Stale original Kontrol image producer')
    for name, value in image_metadata['original_sha256'].items():
        if name not in ('inputs.json','inspect.json','pull.log','pull-attempts.json') or K.S.digest(directory / 'image' / name) != value:
            raise ValueError('Corrupt original Kontrol image producer')
    if set(image_metadata['original_sha256']) != {'inputs.json','inspect.json','pull.log','pull-attempts.json'}:
        raise ValueError('Incomplete original Kontrol image preparation')
    if G.read(directory / 'image/inputs.json') != {'selection':settings['image'],'input_sha256':image_metadata['input_sha256']}:
        raise ValueError('Changed original Kontrol image preparation selection')
    image = G.read(directory / 'runtime-image.json')
    K.IMAGE.validate_inspection(image)
    K.IMAGE.validate_inspection(G.read(directory / 'image/inspect.json'))
    files = {phase:compiler(directory / (phase + '-compiler')) for phase in COMPILER_PHASES}
    if provider == 'rwx':
        producer = directory / 'dependencies/contracts-kontrol'; metadata = G.read(producer / 'metadata.json')
        if (metadata['version'] != 1 or metadata['kind'] != 'contracts-kontrol' or metadata['commit_sha'] != sha
                or metadata['settings'] != G.ARTIFACTS.SETTINGS['contracts-kontrol'] or metadata['mise_sha256'] != before['mise.toml']
                or metadata['tool_versions']['forge'] != settings['tools']['forge']):
            raise ValueError('Wrong Kontrol contract producer revision/settings/toolchain')
        K.helper('kontrol-contracts').verify(producer,sha,before)
        original_inputs = A.archive(producer / 'files.tar.gz',metadata)
        if any(original_inputs.get(n) != data for n,data in files['initial'].items()):
            raise ValueError('Initial Kontrol compiler inputs differ from verified contract producer')
    if files['proofs'] != files['final']: raise ValueError('Kontrol compiler outputs changed after the original build')
    graph_paths = {p for prefix in ('packages/contracts-bedrock/artifacts/build-info/','packages/contracts-bedrock/forge-artifacts/build-info/')
                   for row in A.build_info(files['proofs'],prefix) for p in row['source_files']}
    if not {n.removeprefix('packages/contracts-bedrock/') for n in proofs} <= graph_paths:
        raise ValueError('Kontrol build did not retain every selected proof source graph')
    return {'settings':settings,'selection':selected,'coverage':coverage,'files':files,'image':image[0],
            'hashes':hashes,'declared_empty':empty}


def compare_compiler(x,y,roots,history=((),())):
    prefixes = ('packages/contracts-bedrock/artifacts/build-info/','packages/contracts-bedrock/forge-artifacts/build-info/')
    inventories = [compiler_bindings(files,earlier) for files,earlier in zip((x,y),history)]
    if inventories[0]['bindings'].keys() != inventories[1]['bindings'].keys():
        raise ValueError('Complete Kontrol logical compiler inventories differ')
    retained = [{(tuple(row['logical_key']),row['bound_phase']):row for row in value['retained']} for value in inventories]
    if retained[0].keys() != retained[1].keys():
        raise ValueError('Complete Kontrol retained compiler inventories differ')
    artifact_paths = [set(value['bindings'].values()) | {row['path'] for row in value['retained']} for value in inventories]
    other = [{n for n in files if not n.startswith(prefixes)} - paths for files,paths in zip((x,y),artifact_paths)]
    if other[0] != other[1]: raise ValueError('Complete Kontrol non-artifact compiler inventories differ')
    for prefix in prefixes:
        if A.build_info(x,prefix) != A.build_info(y,prefix): raise ValueError('Complete Kontrol compiler source graphs differ')
    counts = Counter(); contracts = 0; aliases = []
    for key in sorted(inventories[0]['bindings']):
        names = [value['bindings'][key] for value in inventories]
        A.contract(json.loads(x[names[0]],object_pairs_hook=G.C.no_duplicate_keys),
                   json.loads(y[names[1]],object_pairs_hook=G.C.no_duplicate_keys),roots,counts)
        contracts += 1
        if names[0] != names[1]: aliases.append({'logical_key':list(key),'circle':names[0],'rwx':names[1]})
    for key,phase in sorted(retained[0]):
        names = [value[(key,phase)]['path'] for value in retained]
        A.contract(json.loads(x[names[0]],object_pairs_hook=G.C.no_duplicate_keys),
                   json.loads(y[names[1]],object_pairs_hook=G.C.no_duplicate_keys),roots,counts)
        contracts += 1
        if names[0] != names[1]:
            aliases.append({'logical_key':list(key),'bound_phase':phase,'circle':names[0],'rwx':names[1]})
    for name in sorted(other[0]):
        if name.startswith(prefixes): continue
        if name == 'packages/contracts-bedrock/cache/solidity-files-cache.json':
            if A.normalize(A.cache_inputs(inventories[0]['cache']),roots[0]) != A.normalize(A.cache_inputs(inventories[1]['cache']),roots[1]):
                raise ValueError('Kontrol compiler content/configuration cache differs')
        elif x[name] != y[name]: raise ValueError('Unresolved original Kontrol compiler file difference: ' + name)
    return {'contract_artifacts':contracts,'resolved_metadata_differences':dict(counts),
            'verified_artifact_path_aliases':aliases,'complete_logical_inventory_equal':True,
            'verified_retained_previous_artifacts':{p:value['retained'] for p,value in zip(('circle','rwx'),inventories)}}


def compiler_bindings(files,history=()):
    """Bind every current or retained artifact to its original cache identity.

    Foundry can keep an unqualified filename when another compiler/profile is
    added later. Account for that filename only after checking every complete
    payload, compiler graph and cache setting; do not discard any output.
    Summary generation invalidates cache references before the final compile.
    Such retained outputs must have the exact bytes and a validated identity
    from the immediately preceding original phase. A newly compiled payload can
    replace that logical identity while the old qualified filename remains.
    Its original bound phase distinguishes the two; both payloads stay required.
    """
    prefix = 'packages/contracts-bedrock/forge-artifacts/'
    cache = copy.deepcopy(json.loads(files['packages/contracts-bedrock/cache/solidity-files-cache.json'],
                                    object_pairs_hook=G.C.no_duplicate_keys))
    selected = {n for n in files if n.startswith(prefix) and n.endswith('.json') and not n.startswith(prefix+'build-info/')}
    bindings = {}; paths = set()
    for source,row in cache['files'].items():
        if row['sourceName'] != source: raise ValueError('Kontrol compiler cache source identity differs')
        for name,versions in row['artifacts'].items():
            for version,profiles in versions.items():
                for profile,artifact in profiles.items():
                    key = (source,name,version,profile); relative = Path(artifact['path'])
                    path = prefix + artifact['path']
                    filenames = {name+'.json',name+'.'+version+'.json',name+'.'+profile+'.json',name+'.'+version+'.'+profile+'.json'}
                    if (relative.is_absolute() or '..' in relative.parts or path not in selected
                            or relative.name not in filenames or path in paths or key in bindings):
                        raise ValueError('Missing, duplicate or unsafe Kontrol logical artifact binding')
                    value = json.loads(files[path],object_pairs_hook=G.C.no_duplicate_keys)
                    metadata = value.get('metadata')
                    if metadata is not None:
                        if (metadata['compiler']['version'].split('+',1)[0] != version
                                or metadata['settings']['compilationTarget'] != {source:name}):
                            raise ValueError('Kontrol artifact payload differs from its compiler cache identity')
                    bindings[key] = path; paths.add(path)
                    artifact['path'] = {'source':source,'contract':name,'compiler':version,'profile':profile}
    retained = []
    if selected - paths:
        if not history: raise ValueError('Extra or unbound original Kontrol compiler artifacts')
        previous = history[-1]
        earlier = compiler_bindings(previous,history[:-1])
        identities = {path:(key,len(history)-1) for key,path in earlier['bindings'].items()}
        identities.update({row['path']:(tuple(row['logical_key']),row['bound_phase']) for row in earlier['retained']})
        retained_keys = set()
        for path in sorted(selected - paths):
            if path not in identities or previous[path] != files[path] or identities[path] in retained_keys:
                raise ValueError('Changed, duplicate or unbound retained Kontrol compiler artifact')
            key,phase = identities[path]; paths.add(path); retained_keys.add((key,phase))
            retained.append({'path':path,'logical_key':list(key),'bound_phase':phase,'sha256':A.digest(files[path])})
    if not bindings or paths != selected: raise ValueError('Extra or unbound original Kontrol compiler artifacts')
    return {'bindings':bindings,'cache':cache,'retained':retained}


def compare(directory,sha):
    if not re.fullmatch('[0-9a-f]{40}',sha): raise ValueError('Expected full Kontrol benchmark SHA')
    a,b = (report(directory / p,provider,sha) for p,provider in [('circle','circleci'),('rwx','rwx')])
    for key in ('source_sha','branch','input_sha256','submodules','profile','environment','rerun_fails','image','kontrol_version','tools'):
        if a['settings'][key] != b['settings'][key]: raise ValueError('Original Kontrol settings differ: ' + key)
    if a['selection'] != b['selection'] or a['coverage'] != b['coverage']: raise ValueError('Complete Kontrol selection or tracked changes differ')
    roots = [r['settings']['workspace_root'] for r in (a,b)]
    if A.normalize(G.read(directory / 'circle/config.json'),roots[0]) != A.normalize(G.read(directory / 'rwx/config.json'),roots[1]):
        raise ValueError('Complete original Kontrol compiler configuration differs')
    for key in ('RootFS','Config','Architecture','Os','Created'):
        if a['image'][key] != b['image'][key]: raise ValueError('Original immutable Kontrol image differs: ' + key)
    for prefix in ('variants/','generated/','runtime-outputs/'):
        x = {n:h for n,h in a['hashes'].items() if n.startswith(prefix)}
        y = {n:h for n,h in b['hashes'].items() if n.startswith(prefix)}
        if x != y: raise ValueError('Complete original Kontrol generated/state/output files differ: ' + prefix)
    counts = {}; history = ([],[])
    for phase in COMPILER_PHASES:
        current = (a['files'][phase],b['files'][phase])
        counts[phase] = compare_compiler(*current,roots,history)
        for earlier,files in zip(history,current): earlier.append(files)
    return {'source_sha':sha,'verified_parity':True,'native_run_id':b['settings']['rwx_run_id'],
        'selection':a['selection'],'coverage':a['coverage'],'compiler_phases':counts,
        'docker_versions':{p:r['settings']['docker_version'] for p,r in [('circle',a),('rwx',b)]},
        'immutable_image':a['settings']['image'],'docker_image_ids':{p:r['image']['Id'] for p,r in [('circle',a),('rwx',b)]},
        'original_sha256':{'circle':a['hashes'],'rwx':b['hashes']},
        'circle_manifest_declared_empty':a['declared_empty']}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('directory',type=Path); parser.add_argument('--sha',required=True); parser.add_argument('--output',required=True,type=Path)
    args = parser.parse_args(); result = compare(args.directory,args.sha); K.S.write(args.output,result)
    print({k:result[k] for k in ('source_sha','verified_parity','coverage','compiler_phases')})
