#!/usr/bin/env python3
"""Verify complete original SP1 guest jobs, including byte-identical ELFs."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import xml.etree.ElementTree as ET

SPEC = importlib.util.spec_from_file_location('sp1_guest', Path(__file__).with_name('sp1-guest.py'))
G = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(G)
COMMON = ('source_sha', 'branch', 'suite', 'input_sha256', 'tools', 'programs',
          'build_rustflags', 'check_rustflags', 'vkey_prover', 'incremental', 'rerun_fails')
CACHE = {'sccache-start', 'sccache-zero', 'sccache-stats', 'sccache-stop', 'cache-prepare', 'cache-commit'}


def normalized(value, settings, metadata=None):
    """Account only for physical checkout, registry and compiler-output locations."""
    aliases = [(settings['cargo_home'], '<cargo>'), (settings['workspace_root'], '<workspace>')]
    if metadata is not None: aliases.insert(0, (metadata['target_directory'], '<target>'))
    if isinstance(value, str):
        for old, new in aliases: value = value.replace(old, new)
        return value
    if isinstance(value, list): return [normalized(v, settings, metadata) for v in value]
    if isinstance(value, dict):
        return {normalized(k, settings, metadata): normalized(v, settings, metadata) for k,v in value.items()}
    return value


def stages(directory, settings, phase):
    root, rust, sp1 = settings['workspace_root'], settings['workspace_root']+'/rust', settings['workspace_root']+'/rust/kona/sp1'
    expected = { 'install': (['bash', 'ops/ci/sp1-guest-toolchain.sh'], root) } if phase == 'toolchain' else {
        'guest-workspace': (['cargo', 'metadata', '--manifest-path', G.GUEST, '--locked', '--format-version', '1'], rust),
        **{'vkey-'+n: (['cargo', 'prove', 'vkey', '--elf', 'elf/'+n+'-elf'], sp1) for n in settings['programs']}}
    if phase == 'build':
        expected.update({'lock-before': (['just', 'check-sp1-guest-lock'], rust), 'build': (['just', 'build-elfs-native'], sp1)})
    elif phase == 'checks':
        expected.update({
            'guest-list': (['cargo', 'test', '--manifest-path', G.GUEST, '--workspace', '--locked', '--', '--list'], rust),
            'guest': (['just', 'test-sp1-guest'], rust), 'lint': (['just', 'lint-sp1-guest'], rust),
            'range-workspace': (['cargo', 'metadata', '--manifest-path', G.RANGE, '--locked', '--format-version', '1'], rust),
            'range-list': (['cargo', 'test', '--manifest-path', G.RANGE, '--locked', '--', '--list'], rust),
            'range': (['just', 'check-range-vkeys'], rust)})
    actual = {p.name.removesuffix('.stage.json') for p in directory.glob('*.stage.json')}
    cache = CACHE if settings['provider'] == 'rwx' and phase != 'toolchain' else set()
    if actual != expected.keys() | cache: raise ValueError('Missing, extra or retried original SP1 invocation')
    for name, (argv, cwd) in expected.items():
        row = G.read(directory/(name+'.stage.json'))
        if (row['argv'] != argv or row['cwd'] != cwd or row['exit_code'] != 0 or row.get('stdin') != 'devnull'
                or row['log_sha256'] != G.S.digest(directory/(name+'.log'))):
            raise ValueError('Wrong or failed original SP1 command: '+name)
    for name in cache:
        row = G.read(directory/(name+'.stage.json'))
        if row['exit_code'] != 0 or row['log_sha256'] != G.S.digest(directory/(name+'.log')):
            raise ValueError('Failed original SP1 compiler-cache operation')
    if cache:
        record=G.read(directory/'cache-settings.json'); kind='elf' if phase=='build' else 'checks'
        identity={'phase':kind,'tools':settings['tools'],'rustflags':'' if kind=='elf' else '-Dwarnings','incremental':'0'}
        digest=hashlib.sha256(json.dumps(identity,sort_keys=True).encode()).hexdigest()
        target=G.helper('sp1-guest-native-build').POLICY['target_directory'] if kind=='elf' else settings['workspace_root']+'/.ci/sp1-cache/checks-target'
        if (record['phase']!=kind or record['target_mode'] not in ('keep','sccache-only')
                or record['target_directory']!=target or record['sccache_version']!='sccache 0.18.0'
                or record['compiler_inputs']!=identity or record['compiler_input_sha256']!=digest):
            raise ValueError('Wrong original SP1 compiler cache identity or configuration')


def toolchain(directory, settings, circle):
    final = G.read(directory/'final.json')
    if final['phase'] != 'toolchain' or final['exit_code'] or final['report_errors']:
        raise ValueError('Failed complete SP1 compiler setup')
    G.verify_seals(directory, final, allow_circle_empty=circle)
    inputs = G.read(directory/'inputs.json')
    files = {str(p.relative_to(directory/'source')):G.S.digest(p)
             for p in (directory/'source').rglob('*') if p.is_file()}
    if files != inputs or set(files) != set(G.TOOL_INPUTS): raise ValueError('Incomplete original SP1 pin-check inputs')
    if any(settings['input_sha256'].get(n) != h for n,h in inputs.items()): raise ValueError('SP1 pins differ from revision inputs')
    if G.read(directory/'tools.json') != settings['tools']: raise ValueError('Wrong actual SP1 compiler identity')
    if G.programs((directory/'source/rust/kona/sp1/justfile').read_text()) != settings['programs']:
        raise ValueError('SP1 ELF selection differs from its authoritative recipe')
    stages(directory, settings, 'toolchain')


def report(directory, provider, sha):
    circle = provider == 'circleci'
    final, settings = G.read(directory/'final.json'), G.read(directory/'settings.json')
    if (final['phase'] != 'checks' or final['exit_code'] or final['report_errors']
            or (settings['suite'], settings['phase'], settings['source_sha'], settings['provider']) != ('sp1-guest', 'checks', sha, provider)):
        raise ValueError('Failed, stale or incorrect original SP1 job')
    if not settings['branch'] or settings['incremental'] != '0' or settings['rerun_fails'] != 0:
        raise ValueError('Changed SP1 original environment or retry policy')
    if provider == 'rwx' and (not settings.get('rwx_run_id') or str(settings.get('rwx_task_attempt')) != '1'):
        raise ValueError('Missing native fresh execution identity or uninvestigated retry')
    G.verify_seals(directory, final, allow_circle_empty=circle)
    producer = directory/'producer'; pf = G.read(producer/'final.json'); ps = G.read(producer/'settings.json')
    if pf['phase'] != 'build' or pf['exit_code'] or pf['report_errors'] or pf['tests'] != 0 or ps['phase'] != 'build':
        raise ValueError('Failed or false original SP1 ELF producer')
    G.verify_seals(producer, pf, allow_circle_empty=circle)
    if provider=='rwx':G.helper('sp1-guest-native-build').verify(producer,ps,settings['workspace_root'])
    if ps['provider'] != provider or any(ps[n] != settings[n] for n in COMMON):
        raise ValueError('SP1 producer provenance differs from its consumer')
    for path, original, phase in ((directory, settings, 'checks'), (producer, ps, 'build')):
        if G.read(path/'inputs-after.json') != original['input_sha256']: raise ValueError('Original SP1 job changed source inputs')
        toolchain(path/'toolchain', original, circle)
        stages(path, original, phase)
    elfs = G.elf_record(settings, producer/'files')
    if elfs != G.read(producer/'elfs.json'): raise ValueError('ELF manifest differs from complete actual guest bytes')
    for path in (producer, directory):
        for name in settings['programs']:
            if re.findall('0x[0-9a-fA-F]{64}', (path/('vkey-'+name+'.log')).read_text()) != [elfs[name]['vkey']]:
                raise ValueError('Original CPU verification key differs from the guest ELF')
    cases, metadata = {}, {}
    for phase in ('guest', 'range'):
        meta = G.read(directory/(phase+'-workspace.json'))
        coverage = G.coverage(directory, phase, meta, settings, emit=False)
        if (coverage != G.read(directory/(phase+'-coverage.json'))
                or G.junit(coverage) != (directory/(phase+'.junit.xml')).read_bytes()):
            raise ValueError('False original SP1 case report')
        if any(c['outcome'] == 'fail' or c['retries'] for c in coverage['cases']): raise ValueError('Unresolved original SP1 test failure or retry')
        tree = ET.parse(directory/(phase+'.junit.xml'))
        if len(list(tree.iter('testcase'))) != len(coverage['cases']): raise ValueError('Incomplete original SP1 JUnit')
        cases[phase] = coverage
        metadata[phase] = normalized(meta, settings, meta)
    build_meta = G.read(producer/'guest-workspace.json')
    if normalized(build_meta, ps, build_meta) != metadata['guest']: raise ValueError('SP1 build/check dependency graphs differ')
    return {'settings': settings, 'elfs': elfs, 'cases': cases, 'metadata': metadata,
            'original_sha256': final['original_sha256'], 'producer_original_sha256': pf['original_sha256']}


def compare(directory, sha):
    if not re.fullmatch('[0-9a-f]{40}', sha): raise ValueError('Expected full SP1 source SHA')
    a, b = (report(directory/p, provider, sha) for p,provider in [('circle','circleci'),('rwx','rwx')])
    if any(a['settings'][n] != b['settings'][n] for n in COMMON): raise ValueError('Original SP1 workload settings differ')
    for name in ('elfs', 'cases', 'metadata'):
        if a[name] != b[name]: raise ValueError('Complete original SP1 '+name+' differ')
    return {'source_sha': sha, 'verified_parity': True, 'native_run_id': b['settings']['rwx_run_id'],
        'elfs': a['elfs'], 'cases': a['cases'], 'complete_dependency_graphs_equal': True,
        'original_sha256': {p:r['original_sha256'] for p,r in [('circle',a),('rwx',b)]},
        'producer_original_sha256': {p:r['producer_original_sha256'] for p,r in [('circle',a),('rwx',b)]}}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('directory', type=Path); parser.add_argument('--sha', required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args(); result = compare(args.directory, args.sha); G.S.write(args.output, result)
    print({'source_sha': args.sha, 'verified_parity': True, 'elfs': len(result['elfs']),
           'tests': sum(len(c['cases']) for c in result['cases'].values())})
