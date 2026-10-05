#!/usr/bin/env python3
"""Verify complete original pre-fork generation parity on one source revision."""
import argparse
import json
from pathlib import Path
import re
import tempfile

import importlib.util

SPEC = importlib.util.spec_from_file_location('nut_prefork', Path(__file__).with_name('nut-prefork.py'))
N = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(N)
SPEC = importlib.util.spec_from_file_location('cannon_compare', Path(__file__).with_name('compare-cannon-go.py'))
C = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(C)
G = N.G
REQUIRED = {'settings.json', 'selection.json', 'coverage.json', 'generated.json', 'inputs-after.json',
    'packages.json', 'packages.log', 'packages.stage.json', 'list.json', 'list.log', 'list.stage.json',
    'check.log', 'check.stage.json', 'native.junit.xml', 'native.metadata.json'}


def stage(directory, name, argv, root):
    row = G.read(directory / (name + '.stage.json'))
    if row['argv'] != argv or row['cwd'] != root or row['exit_code'] != 0 or row.get('stdin') != 'devnull':
        raise ValueError('Changed or failed original pre-fork command: ' + name)
    if row['log_sha256'] != N.S.digest(directory / (name + '.log')): raise ValueError('Corrupt pre-fork stage log')
    return C.normalize({k: row[k] for k in ('argv', 'cwd', 'exit_code')}, root)


def report(directory, provider, sha):
    final = G.read(directory / 'final.json'); hashes = final['original_sha256']; empty = []
    if final['exit_code'] != 0 or final['report_errors'] or not REQUIRED <= hashes.keys():
        raise ValueError('Failed or incomplete original pre-fork report')
    for name, digest in hashes.items():
        relative = Path(name); path = directory / relative
        if relative.is_absolute() or '..' in relative.parts or not re.fullmatch('[0-9a-f]{64}', digest):
            raise ValueError('Unsafe original pre-fork file manifest')
        if provider == 'circleci' and not path.exists() and digest == C.EMPTY:
            path.parent.mkdir(parents=True, exist_ok=True); path.write_bytes(b'')
            empty.append({'path':name,'sha256':digest})
        if path.is_symlink() or N.S.digest(path) != digest: raise ValueError('Missing or corrupt original pre-fork report: ' + name)
    if {str(p.relative_to(directory)) for p in directory.rglob('*') if p.is_file() and p.name != 'final.json'} != set(hashes):
        raise ValueError('Unsealed or missing original pre-fork input')
    settings = G.read(directory / 'settings.json'); root = settings['workspace_root']
    if (settings['suite'], settings['source_sha'], settings['provider']) != ('nut-prefork', sha, provider):
        raise ValueError('Wrong original pre-fork suite, source or provider')
    if not settings['branch'] or settings['go_environment']['GOFLAGS']: raise ValueError('Missing branch or unaccounted Go flags')
    if provider == 'rwx' and (not settings['rwx_run_id'] or str(settings['rwx_task_attempt']) != '1'):
        raise ValueError('Missing original pre-fork native identity or unexpected retry')
    if G.read(directory / 'inputs-after.json') != settings['input_sha256']: raise ValueError('Original pre-fork inputs changed')
    selected = N.select_states(directory / 'source')
    packages = N.package_selection((directory / 'packages.json').read_text(), root, directory / 'source')
    selected['packages'] = packages; selected['initial_tests'] = G.listing(directory / 'list.json', packages)
    for row in packages:
        if any(settings['input_sha256'].get(k) != v for k, v in row['files'].items()):
            raise ValueError('Pre-fork package differs from bound source inputs')
    names = {n for row in packages for n in row['files']} | {r['path'] for r in selected['states'].values()}
    names |= {'justfile', 'ops/ci/nut-prefork-test.sh'}
    actual_source = {str(p.relative_to(directory / 'source')) for p in (directory / 'source').rglob('*') if p.is_file()}
    if actual_source != names: raise ValueError('Incomplete or extra original pre-fork source selection')
    for name in names:
        if N.S.digest(directory / 'source' / name) != settings['input_sha256'].get(name):
            raise ValueError('Pre-fork source bytes differ from complete input seal')
    bundle = directory / 'dependencies/superchain-configs.zip'
    pin = directory / 'dependencies/superchain-configs.zip.sha256'
    digest = pin.read_text().split()[0]
    if not re.fullmatch('[0-9a-f]{64}', digest) or N.S.digest(bundle) != digest:
        raise ValueError('Corrupt original pre-fork superchain bundle')
    if N.S.digest(pin) != settings['input_sha256'][N.MAIN.BUNDLE + '.sha256']:
        raise ValueError('Wrong pre-fork superchain checksum input')
    original_selected = G.read(directory / 'selection.json')
    selected['superchain'] = original_selected['superchain']
    identity = selected['superchain']
    if (identity['source_sha'] != sha or identity['sha256'] != digest or identity['expected_sha256'] != digest
            or identity['registry_revision'] != settings['input_sha256']['superchain-registry']['gitlink']
            or identity['input_sha256'] != {k:settings['input_sha256'][k] for k in N.MAIN.BUNDLE_INPUTS}
            or identity['tools']['go'] != settings['tool_versions']['go']
            or identity['tools']['just'] != settings['tool_versions']['just']):
        raise ValueError('Wrong original pre-fork bundle provenance')
    if original_selected != selected or G.read(directory / 'generated.json') != N.select_states(directory / 'source'):
        raise ValueError('Pre-fork discovery or generated state bytes differ')
    commands = {name: stage(directory, name, argv, root) for name, argv in [
        ('packages', ['go', 'list', '-e', '-json', './' + N.PACKAGE_DIR + '/']),
        ('list', ['go', 'test', '-json', '-count=1', '-list', N.PATTERN, './' + N.PACKAGE_DIR + '/']),
        ('check', ['just', '_check-nut-prefork-states'])]}
    if {p.name.removesuffix('.stage.json') for p in directory.glob('*.stage.json')} != set(commands):
        raise ValueError('Missing pre-fork command or unexplained retry')
    actual = sorted(p.name for p in (directory / 'forks').iterdir())
    if actual != selected['forks']: raise ValueError('Pre-fork assignment differs from complete original discovery')
    coverage = {}
    for fork in actual:
        child = directory / 'forks' / fork
        argv = [str(Path(root) / 'ops/scripts/gotestsum-split.sh'), '--format=testname',
            '--junitfile=' + str(Path(root) / '.ci/nut-prefork/run/forks' / fork / 'junit.xml'),
            '--jsonfile=' + str(Path(root) / '.ci/nut-prefork/run/forks' / fork / 'original.json'), '--', *N.FLAGS]
        commands[fork] = stage(child, 'tests', argv, root)
        if {p.name for p in child.glob('*.stage.json')} != {'tests.stage.json'}: raise ValueError('Unexpected pre-fork retry')
        coverage[fork] = G.observed(child, packages, selected['initial_tests'])
        if coverage[fork]['package_failures'] or coverage[fork]['outcomes'].get('fail') or coverage[fork]['outcomes'].get('skip'):
            raise ValueError('Original pre-fork generation failed or skipped')
        with tempfile.TemporaryDirectory() as temp:
            output = Path(temp) / 'native.json'; G.PROJECT.project(child / 'original.json', output)
            for name in ('native.json', 'native.metadata.json'):
                if N.S.digest(Path(temp) / name) != N.S.digest(child / name): raise ValueError('Changed pre-fork native projection')
    expected = {'forks':coverage, 'attempts_per_fork':1,
        'outcomes':dict(N.Counter(c['outcome'] for body in coverage.values() for c in body['cases']))}
    if expected != G.read(directory / 'coverage.json'): raise ValueError('Original pre-fork coverage differs from events')
    with tempfile.TemporaryDirectory() as temp:
        target = Path(temp); (target / 'forks').symlink_to(directory.resolve() / 'forks', target_is_directory=True)
        N.native_junit(target, actual)
        for name in ('native.junit.xml', 'native.metadata.json'):
            if N.S.digest(target / name) != N.S.digest(directory / name): raise ValueError('Changed native pre-fork grouping')
    dependencies = {}
    if provider == 'rwx':
        for kind in ('go-modules', 'contracts'):
            row = G.read(directory / 'dependencies' / (kind + '.json'))
            if (row['kind'] != kind or row['commit_sha'] != sha or row['settings'] != G.ARTIFACTS.SETTINGS[kind]
                    or row['mise_sha256'] != settings['input_sha256']['mise.toml'] or not row['files']
                    or any(settings['tool_versions'].get(k) != v for k,v in row['tool_versions'].items())):
                raise ValueError('Stale or mismatched pre-fork runtime artifact')
            dependencies[kind] = row
        producer = directory / 'producer'; final = G.read(producer / 'manifest.json')
        if final['exit_code'] != 0 or final['report_errors'] or set(final['original_sha256']) != {'bundle.json','coverage.json','superchain.log','superchain.stage.json'}:
            raise ValueError('Incomplete pre-fork superchain producer')
        for name, value in final['original_sha256'].items():
            if N.S.digest(producer / name) != value: raise ValueError('Corrupt original pre-fork superchain producer')
        if G.read(producer / 'bundle.json') != identity or G.read(producer / 'coverage.json') != {'tests':0,'verified_bundle':True}:
            raise ValueError('Wrong pre-fork superchain producer identity')
        producer_stage = G.read(producer / 'superchain.stage.json')
        if producer_stage['argv'] != ['just','build-superchain-go'] or producer_stage['exit_code'] != 0:
            raise ValueError('Wrong original pre-fork superchain producer command')
    return {'settings':settings, 'selection':selected, 'coverage':coverage, 'outcomes':expected['outcomes'],
            'commands':commands, 'dependencies':dependencies, 'original_sha256':hashes, 'declared_empty':empty}


def compare(directory, sha):
    if not re.fullmatch('[0-9a-f]{40}', sha): raise ValueError('Expected full pre-fork benchmark SHA')
    a = report(directory / 'circle', 'circleci', sha); b = report(directory / 'rwx', 'rwx', sha)
    for field in ('source_sha','branch','input_sha256','environment','go_environment','tool_versions'):
        if a['settings'][field] != b['settings'][field]: raise ValueError('Original pre-fork settings differ: ' + field)
    if a['selection'] != b['selection'] or a['commands'] != b['commands']: raise ValueError('Complete pre-fork discovery or invocation differs')
    def cases(value):
        return {fork:[{k:c[k] for k in ('suite','name','outcome','skip_reason')} |
                      {'attempts':[{k:h[k] for k in ('outcome','skip_reason')} for h in c['attempts']]}
                     for c in sorted(body['cases'], key=lambda r:(r['suite'],r['name']))]
                for fork,body in value.items()}
    if cases(a['coverage']) != cases(b['coverage']): raise ValueError('Original pre-fork cases, skips or retry histories differ')
    if any(a['coverage'][f]['package_attempts'] != b['coverage'][f]['package_attempts'] for f in a['coverage']):
        raise ValueError('Original pre-fork package attempts differ')
    return {'source_sha':sha,'verified_parity':True,'forks':a['selection']['forks'],
        'selection':a['selection'],'cases':cases(a['coverage']),'outcomes':a['outcomes'],
        'commands':a['commands'],'tool_versions':a['settings']['tool_versions'],
        'original_sha256':{'circle':a['original_sha256'],'rwx':b['original_sha256']},
        'native_dependencies':b['dependencies'],'circle_manifest_declared_empty':a['declared_empty'],
        'native_run_id':b['settings']['rwx_run_id']}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__); parser.add_argument('directory',type=Path)
    parser.add_argument('--sha',required=True); parser.add_argument('--output',type=Path,required=True)
    args = parser.parse_args()
    result = compare(args.directory,args.sha); N.S.write(args.output,result)
    print(json.dumps({k:result[k] for k in ('source_sha','verified_parity','forks','outcomes')}))
