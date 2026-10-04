#!/usr/bin/env python3
"""Compare complete L1 upgrade discovery, artifacts and original verdicts."""
import argparse
import importlib.util
import json
from pathlib import Path
import re

SPEC = importlib.util.spec_from_file_location('upgrade', Path(__file__).with_name('contract-upgrades.py'))
UP = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(UP)


def report(directory, variant, sha, provider, empty):
    required = {'settings.json', 'selection.json', 'foundry-config.json', 'foundry-config.stage.json',
                'discovery.json', 'discovery.stage.json', 'go-ffi.stage.json', 'contracts-build.stage.json', 'submodules.txt',
                'compiled.json', 'signature-bindings.json', 'block.json', 'coverage.json', 'tests.stage.json', 'original.junit.xml'}
    hashes = UP.ORIGINALS.originals(directory, required, empty, provider + '/' + variant)
    settings = json.loads((directory / 'settings.json').read_text())
    if (settings['source_sha'], settings['variant'], settings['provider']) != (sha, variant, 'circleci' if provider == 'circle' else 'rwx'):
        raise ValueError('Upgrade source, variant or provider differs')
    chain, feature = UP.VARIANTS[variant]
    if (settings['chain'], settings['feature'], settings['fuzz_seed'], settings['fuzz_runs']) != (chain, feature, '42424242', 1):
        raise ValueError('Unexpected effective upgrade workload')
    if settings['profile'] != ('ci' if settings['branch'] == 'develop' else 'liteci'): raise ValueError('Upgrade branch profile differs')
    if provider == 'rwx' and (not settings['rwx_run_id'] or str(settings['rwx_task_attempt']) != '1'):
        raise ValueError('Missing fresh native upgrade identity or uninvestigated task retry')
    methods = json.loads((directory / 'signature-bindings.json').read_text())
    compiled = json.loads((directory / 'compiled.json').read_text())
    for row in methods.values():
        if not row['artifacts'] or any(compiled.get(path) != hashed for path, hashed in row['artifacts'].items()):
            raise ValueError('Original compiler signature artifact binding differs')
    selected = UP.selection(json.loads((directory / 'discovery.json').read_text()), methods)
    if selected != [tuple(r) for r in json.loads((directory / 'selection.json').read_text())]:
        raise ValueError('Upgrade selection differs from original discovery and signatures')
    original = UP.junit(directory / 'original.junit.xml', selected)
    if original != json.loads((directory / 'coverage.json').read_text()): raise ValueError('Upgrade coverage differs from original JUnit')
    block = json.loads((directory / 'block.json').read_text())
    if block['source_sha'] != sha or block['chain_id'] != 1 or block['policy'] != 'Just current-day 00:00 UTC' or \
       type(block['number']) is not int or block['number'] <= 0 or not re.fullmatch('0x[0-9a-f]{64}', block['hash']):
        raise ValueError('Wrong bound archive block identity')
    commands = {}
    expected = {'foundry-config': ['forge', 'config', '--json'], 'go-ffi': ['just', 'build-go-ffi'],
                'contracts-build': ['forge', 'build'],
                'discovery': ['forge', 'test', '--list', '--json', '--match-path', UP.MATCH], 'tests': ['just', 'test-upgrade']}
    for name, argv in expected.items():
        row = json.loads((directory / (name + '.stage.json')).read_text())
        if row['argv'] != argv or row['exit_code'] != 0 or row['cwd'] != settings['workspace_root'] + '/packages/contracts-bedrock':
            raise ValueError('Unexpected original upgrade command, directory or failure')
        commands[name] = argv
    stages = {p.name.removesuffix('.stage.json') for p in directory.glob('*.stage.json')}
    if stages - set(expected) - {'pinned-block'}: raise ValueError('Uninvestigated extra upgrade command or diagnostic retry')
    if 'pinned-block' in stages:
        row = json.loads((directory / 'pinned-block.stage.json').read_text())
        if row['argv'] != ['just', 'print-pinned-block-number'] or row['exit_code'] != 0 or \
           (directory / 'pinned-block.log').read_text().strip() != str(block['number']):
            raise ValueError('Original archive block discovery differs')
    return {'settings': settings, 'selection': selected, 'coverage': original, 'block': block,
            'config': UP.ORIGINALS.normalize(json.loads((directory / 'foundry-config.json').read_text()), settings['workspace_root']),
            'submodules': UP.SUBMODULES.revisions((directory / 'submodules.txt').read_text()), 'methods': {k:v['methods'] for k,v in methods.items()},
            'commands': commands, 'original_sha256': hashes, 'compiled_sha256': compiled}


def compare(directories, variant, sha):
    if not re.fullmatch('[0-9a-f]{40}', sha): raise ValueError('Expected complete upgrade benchmark SHA')
    empty = []; data = {p:report(d, variant, sha, p, empty) for p,d in directories.items()}
    a,b = data['circle'],data['rwx']
    for key in ('source_sha','variant','branch','chain','feature','profile','fuzz_seed','fuzz_runs','forge','go','just','input_sha256'):
        if a['settings'][key] != b['settings'][key]: raise ValueError('Upgrade settings binding differs at ' + key)
    for key in ('selection','coverage','block','config','submodules','methods','commands'):
        if a[key] != b[key]: raise ValueError('Original upgrade parity differs at ' + key)
    return {'source_sha':sha,'variant':variant,'verified_parity':True,'selection':a['selection'],'coverage':a['coverage'],
            'block':a['block'],'commands':a['commands'],'settings':{p:d['settings'] for p,d in data.items()},
            'original_sha256':{p:d['original_sha256'] for p,d in data.items()},
            'compiled_sha256':{p:d['compiled_sha256'] for p,d in data.items()},'manifest_declared_empty_logs':empty}


if __name__ == '__main__':
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--circle',type=Path,required=True);p.add_argument('--rwx',type=Path,required=True)
    p.add_argument('--sha',required=True);p.add_argument('--variant',choices=UP.VARIANTS,required=True);p.add_argument('--output',type=Path,required=True)
    a=p.parse_args();result=compare({'circle':a.circle,'rwx':a.rwx},a.variant,a.sha);UP.write(a.output,result)
    print(json.dumps({'source_sha':a.sha,'variant':a.variant,'verified_parity':True,'outcomes':result['coverage']['outcomes']}))
