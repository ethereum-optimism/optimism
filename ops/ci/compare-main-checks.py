#!/usr/bin/env python3
"""Compare complete original Main-validator commands, inputs and discoveries."""
import argparse
import importlib.util
import json
from pathlib import Path
import re
import tomllib
import xml.etree.ElementTree as ET

def helper(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module

MAIN = helper('main-checks')
REPORT = helper('ci-report')


def source_files(directory, settings, expected):
    root = directory / 'source'
    actual = {str(p.relative_to(root)) for p in root.rglob('*') if p.is_file()}
    if actual != set(expected): raise ValueError('Missing or extra original Main source bytes')
    for name in actual:
        p = root / name
        if p.is_symlink() or MAIN.STAGES.digest(p) != settings['input_sha256'].get(name):
            raise ValueError('Original Main source bytes differ from complete inputs')
    return root


def superchain(directory, selected, settings, provider):
    root = directory / 'dependencies'
    bundle = root / 'superchain-configs.zip'; pin = root / 'superchain-configs.zip.sha256'
    expected = pin.read_text().split()[0]
    if not re.fullmatch('[0-9a-f]{64}', expected) or MAIN.STAGES.digest(bundle) != expected or \
       MAIN.STAGES.digest(pin) != settings['input_sha256'][MAIN.BUNDLE + '.sha256']:
        raise ValueError('Missing or corrupt original pinned superchain bundle')
    identity = {'source_sha':settings['source_sha'], 'registry_revision':settings['input_sha256']['superchain-registry']['gitlink'],
                'sha256':expected,'expected_sha256':expected,
                'input_sha256':{name:settings['input_sha256'][name] for name in MAIN.BUNDLE_INPUTS},
                'tools':{name:settings['tools'][name] for name in ('go','just','jq','yq')}}
    if selected != identity: raise ValueError('Wrong superchain source, inputs or tools')
    if provider == 'rwx':
        root = directory / 'producer'; manifest = json.loads((root / 'manifest.json').read_text())
        if manifest['exit_code'] or manifest['report_errors'] or set(manifest['original_sha256']) != \
           {'bundle.json','coverage.json','superchain.log','superchain.stage.json'}:
            raise ValueError('Failed or incomplete reusable superchain producer')
        for name, value in manifest['original_sha256'].items():
            if MAIN.STAGES.digest(root / name) != value: raise ValueError('Corrupt original superchain producer')
        if json.loads((root / 'bundle.json').read_text()) != identity or \
           json.loads((root / 'coverage.json').read_text()) != {'tests':0,'verified_bundle':True}:
            raise ValueError('Wrong original reusable superchain binding')
        stage = json.loads((root / 'superchain.stage.json').read_text())
        if (stage['argv'],stage['cwd'],stage['exit_code'],stage.get('stdin')) != \
           (['just','build-superchain-go'],settings['workspace_root'],0,'devnull'):
            raise ValueError('Wrong original reusable superchain command')


def report(directory, job, sha, provider, empty):
    required = {'settings.json', 'selection.json', 'coverage.json', 'check.stage.json', 'check.log', 'check.junit.xml', 'inputs-after.json'}
    final = REPORT.read(directory / 'final.json')
    if final['exit_code'] != 0 or final['report_errors']:
        raise ValueError('Failed or incomplete original Main check report')
    hashes = REPORT.verify_files(directory, final['original_sha256'], required=required,
                                 missing_empty=empty if provider == 'circle' else None, label=provider + '/' + job)
    if {str(p.relative_to(directory)) for p in directory.rglob('*') if p.is_file() and p.name != 'final.json'} != set(hashes):
        raise ValueError('Unsealed or missing original Main report files')
    settings = json.loads((directory / 'settings.json').read_text())
    if (settings['source_sha'], settings['job'], settings['provider']) != (sha, job, 'circleci' if provider == 'circle' else 'rwx'):
        raise ValueError('Wrong Main source, job or provider')
    if provider == 'rwx' and (not settings['rwx_run_id'] or str(settings['rwx_task_attempt']) != '1'):
        raise ValueError('Main verdict is reused or a retry requires investigation')
    if not settings['input_sha256']: raise ValueError('Missing complete Main source inputs')
    if json.loads((directory / 'inputs-after.json').read_text()) != settings['input_sha256']:
        raise ValueError('Original Main command changed tracked inputs')
    if job in MAIN.GO_JOBS and settings['go_environment'] != json.loads((directory / 'go-env.json').read_text()):
        raise ValueError('Go environment differs from original')
    selected = json.loads((directory / 'selection.json').read_text())
    expected_coverage = {'job': job, 'outcome': 'pass', 'attempts': 1, 'complete_selection': True, 'command': MAIN.COMMANDS[job]}
    if json.loads((directory / 'coverage.json').read_text()) != expected_coverage: raise ValueError('Wrong original Main coverage')
    suite = ET.parse(directory / 'check.junit.xml').getroot(); junit = list(suite.iter('testcase'))
    if suite.attrib != {'name': 'main-checks', 'tests': '1', 'failures': '0'} or len(junit) != 1 or \
       junit[0].attrib != {'name': job, 'classname': 'main-checks'} or list(junit[0]):
        raise ValueError('Missing or failed original Main JUnit')
    expected = {'check': MAIN.COMMANDS[job]}
    if job in MAIN.GO_JOBS: expected['go-env'] = ['go', 'env', '-json', 'GOOS', 'GOARCH', 'CGO_ENABLED', 'GOFLAGS', 'GOTOOLCHAIN']
    if job in MAIN.MOCKS:
        expected.update(superchain=['just','build-superchain-go'], packages=['go', 'list', '-tags=generate', '-json', './...'], generators=['go', 'generate', '-n', '-v', './...'])
        superchain(directory, selected['superchain'], settings, provider)
        component = MAIN.MOCKS[job]
        rows = MAIN.PR.objects((directory / 'packages.json').read_text())
        names = {name for name in settings['input_sha256'] if name.startswith(component + '/') and name.endswith('.go')}
        root = source_files(directory, settings, names)
        discovered = MAIN.packages((directory / 'packages.json').read_text(), component,
                                   settings['workspace_root'], root)
        if selected['component'] != component or discovered != selected['packages']:
            raise ValueError('Generation selection differs from original packages and source directives')
        before = {name: MAIN.STAGES.digest(root / name) for name in names
                  if re.search(rb'^// Code generated .*DO NOT EDIT\.', (root / name).read_bytes(), re.M)}
        if not before or before != selected['generated_before'] or json.loads((directory / 'generated.json').read_text()) != before:
            raise ValueError('Original generated mocks changed or selection is incomplete')
    elif job == 'check-nut-locks':
        expected['fetch-develop'] = ['git', 'fetch', '--no-tags', 'origin', '+refs/heads/develop:refs/remotes/origin/develop']
        if not selected['locks'] or not re.fullmatch('[0-9a-f]{40}', selected['develop_sha']): raise ValueError('Missing original NUT locks or protected ancestry')
        names = {name for name in settings['input_sha256'] if re.fullmatch(r'op-core/nuts/(bundles/[^/]+_nut_bundle.json|state/[^/]+_state.json|fork_lock.toml)', name)}
        root = source_files(directory, settings, names)
        if MAIN.nut_selection(root, selected['develop_sha']) != selected: raise ValueError('NUT selection differs from original complete inputs')
    elif job == 'check-op-geth-version':
        expected['go-mod'] = ['go', 'mod', 'edit', '-json']
        source_files(directory, settings, ['go.mod'])
        if selected != json.loads((directory / 'go-mod.json').read_text()): raise ValueError('Module selection differs from original')
    elif job == 'op-deployer-forge-version':
        expected['mise-config'] = ['yq', '-o=json', '.', 'mise.toml']
        root = source_files(directory, settings, ['mise.toml', 'op-deployer/pkg/deployer/forge/version.json'])
        pin = tomllib.loads((root / 'mise.toml').read_text())['tools']['forge']
        if pin != json.loads((directory / 'mise-config.json').read_text())['tools']['forge'] or \
           selected != {'mise_forge':pin, 'deployer':json.loads((root / 'op-deployer/pkg/deployer/forge/version.json').read_text())}:
            raise ValueError('Forge version selection differs from original')
    elif job == 'l2-chains-sync-check':
        expected['configuration'] = ['yq', '-o=json', '.', '.circleci/continue/main.yml']
        root = source_files(directory, settings, ['.circleci/continue/main.yml', '.circleci/l2-rpcs.json'])
        discovery = MAIN.l2_selection(json.loads((directory / 'configuration.json').read_text()),
                                      json.loads((root / '.circleci/l2-rpcs.json').read_text()))
        if selected != discovery or discovery['rpc_chains'] != discovery['matrix_chains']:
            raise ValueError('L2 chain selection differs from original complete configuration')
    else:
        if not selected['files'] or selected['check_closed'] is not False or selected['near_misses']: raise ValueError('Incomplete or unsuccessful TODO discovery')
        if sorted((directory / 'files.log').read_text().splitlines()) != selected['files'] or \
           sorted((directory / 'todos.log').read_text().splitlines()) != selected['todos'] or \
           sorted((directory / 'near-misses.log').read_text().splitlines()) != selected['near_misses']:
            raise ValueError('TODO selection differs from complete original scans')
        for name in ('files', 'todos', 'near-misses'):
            row = json.loads((directory / (name + '.stage.json')).read_text())
            if row['exit_code'] not in (0, 1) or row['cwd'] != settings['workspace_root']: raise ValueError('Failed original TODO discovery')
        expected.update(MAIN.TODO_DISCOVERY)
    stages = {p.name.removesuffix('.stage.json') for p in directory.glob('*.stage.json')}
    if stages != set(expected): raise ValueError('Missing original command or unexplained Main retry')
    for name, argv in expected.items():
        row = json.loads((directory / (name + '.stage.json')).read_text())
        allowed = (0, 1) if name in ('todos','near-misses') else (0,)
        relative = MAIN.MOCKS[job] if job in MAIN.MOCKS and name in ('packages','generators') else 'op-deployer' if job == 'op-deployer-forge-version' and name == 'check' else ''
        cwd = str(Path(settings['workspace_root']) / relative)
        if row['argv'] != argv or row['exit_code'] not in allowed or row['cwd'] != cwd or row.get('stdin') != 'devnull':
            raise ValueError('Wrong or failed original Main command')
    return {'settings':settings, 'selection':selected, 'coverage':expected_coverage, 'original_sha256':hashes,
            'commands':expected, 'generator_plan':(directory/'generators.log').read_text() if job in MAIN.MOCKS else None}


def compare(directories, job, sha):
    if not re.fullmatch('[0-9a-f]{40}', sha): raise ValueError('Expected full Main benchmark revision')
    empty = []; data = {p:report(d,job,sha,p,empty) for p,d in directories.items()}; a,b = data['circle'],data['rwx']
    for key in ('source_sha','job','branch','tools','input_sha256','go_environment'):
        if a['settings'].get(key) != b['settings'].get(key): raise ValueError('Main settings differ at ' + key)
    for key in ('selection','coverage','commands','generator_plan'):
        if a[key] != b[key]: raise ValueError('Complete original Main parity differs at ' + key)
    return {'source_sha':sha,'job':job,'verified_parity':True,'selection':a['selection'],'coverage':a['coverage'],
            'settings':{p:d['settings'] for p,d in data.items()},'commands':a['commands'],
            'original_sha256':{p:d['original_sha256'] for p,d in data.items()},'manifest_declared_empty_logs':empty}


if __name__ == '__main__':
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--circle',type=Path,required=True);p.add_argument('--rwx',type=Path,required=True)
    p.add_argument('--sha',required=True);p.add_argument('--job',choices=MAIN.COMMANDS,required=True);p.add_argument('--output',type=Path,required=True)
    a=p.parse_args();result=compare({'circle':a.circle,'rwx':a.rwx},a.job,a.sha);MAIN.STAGES.write(a.output,result)
    print(json.dumps({'source_sha':a.sha,'job':a.job,'verified_parity':True}))
