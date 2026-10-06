#!/usr/bin/env python3
"""Compare every coverage case, LCOV entry and per-test attribution item."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
_SPEC_E = importlib.util.spec_from_file_location('report_evidence', Path(__file__).with_name('report-evidence.py'))
E = importlib.util.module_from_spec(_SPEC_E); _SPEC_E.loader.exec_module(E)
import re
import xml.etree.ElementTree as ET

SPEC = importlib.util.spec_from_file_location('coverage', Path(__file__).resolve().parents[1] / 'runtime' / 'contract-coverage.py')
C = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(C)
UP, CS = C.UP, C.CS


def derived(path, phase, cases):
    rows = []
    for suite in ET.parse(path).iter('testsuite'):
        children = suite.findall('testcase')
        if suite.attrib.get('tests') != str(len(children)) or \
           suite.attrib.get('failures') != str(sum(case.find('failure') is not None for case in children)) or \
           suite.attrib.get('skipped') != str(sum(case.find('skipped') is not None for case in children)):
            raise ValueError('Derived coverage JUnit totals differ from original cases')
        for case in children:
            identity = case.attrib['classname']; name = case.attrib['name']
            if not identity.startswith(phase + ':'): raise ValueError('Derived coverage case lacks its phase identity')
            identity = identity.removeprefix(phase + ':')
            if suite.attrib['name'] != phase + ':' + identity or case.find('error') is not None:
                raise ValueError('Invalid derived coverage JUnit identity')
            failure, skipped = case.find('failure'), case.find('skipped')
            child = failure if failure is not None else skipped
            reason = child.get('message') if child is not None else None
            if child is not None and (child.attrib != {'message': reason} or ''.join(child.itertext()) != (reason or '')):
                raise ValueError('Derived coverage JUnit reason differs from original text')
            rows.append({'class': identity, 'name': name, 'outcome': 'fail' if failure is not None else 'skip' if skipped is not None else 'pass',
                         'reason': reason or None})
    if sorted(rows, key=lambda r: (r['class'], r['name'])) != cases:
        raise ValueError('Derived coverage JUnit differs from original attribution/log verdicts')


def report(directory, feature, sha, provider, empty):
    required = C.PREPARED | {'block.json', 'source-after-verdict.json', 'runtime-fixtures.json'}
    for phase in C.PHASES:
        required |= {phase + '/' + name for name in ('tests.log', 'tests.stderr.log', 'tests.stage.json', 'original.lcov.info',
                     'original.attribution.json', 'original-cases.json', 'original-events.json', 'derived.junit.xml', 'lcov-records.json',
                     'coverage-source-sha256.json', 'coverage.json')}
    if provider == 'rwx': required |= {'preparation-manifest.json', 'preparation-settings.json', 'runtime-config.json',
                                       'runtime-config.stage.json', 'runtime-files.log', 'runtime-files.stage.json'}
    sealed_preflight = (directory / 'archive-preflight/preflight-manifest.json').exists()
    if sealed_preflight: required |= {'archive-preflight/block.json', 'archive-preflight/pinned-block.log',
                                      'archive-preflight/pinned-block.stage.json', 'archive-preflight/preflight-manifest.json'}
    else: required |= {'pinned-block.log', 'pinned-block.stage.json'}
    try:
        hashes = E.originals(directory, required, empty, provider + '/coverage-' + feature)
    except OSError as error:
        raise ValueError('Missing original contract coverage report file') from error
    if {str(p.relative_to(directory)) for p in directory.rglob('*') if p.is_file() and p.name != 'final.json'} != set(hashes):
        raise ValueError('Unsealed or missing complete coverage originals')
    settings = json.loads((directory / 'settings.json').read_text()); root = settings['workspace_root']
    seed = '0x' + hashlib.sha256((sha + ':' + feature + ':contract-coverage').encode()).hexdigest()
    if (settings['source_sha'], settings['feature'], settings['profile'], settings['source_build_profile'], settings['benchmark_seed'],
        settings['authority'], settings['phases'], settings['provider'], settings['rpc_input_name']) != \
       (sha, feature, 'cicoverage', 'default', seed, ['just', 'coverage-lcov-all'], list(C.PHASES), 'circleci' if provider == 'circle' else 'rwx', C.RPC_INPUT):
        raise ValueError('Wrong coverage source, occurrence, profile or opt-in replay benchmark seed')
    if (settings['ffi_replay_seed'], settings['ffi_replay_policy']) != (seed, 'chacha8-arguments-v1'):
        raise ValueError('Coverage FFI replay settings differ from the benchmark input')
    if provider == 'rwx' and (not settings['rwx_run_id'] or str(settings['rwx_task_attempt']) != '1'):
        raise ValueError('Reused coverage verdict or uninvestigated task retry')
    before = settings['input_sha256']
    if not before or json.loads((directory / 'source-after-preparation.json').read_text()) != before:
        raise ValueError('Coverage preparation changed source inputs')
    files = CS.file_selection((directory / 'files.log').read_text())
    expected = sorted(p.removeprefix('packages/contracts-bedrock/') for p in before
                      if p.startswith('packages/contracts-bedrock/test/') and p.endswith('.t.sol'))
    if files != expected or not files or json.loads((directory / 'file-selection.json').read_text()) != \
       {'files': files, 'partitions': [{'index': 0, 'files': files}]}:
        raise ValueError('Incomplete or duplicate coverage file selection/assignment')
    outputs = CS.runtime_outputs(files)
    if settings.get('runtime_output_paths') != outputs: raise ValueError('Undeclared coverage runtime fixture role')
    after = json.loads((directory / 'source-after-verdict.json').read_text())
    changes = {name: {'before': before.get(name), 'after': after.get(name)} for name in sorted(set(before) | set(after))
               if before.get(name) != after.get(name)}
    if set(changes) - set(outputs) or changes and json.loads((directory / 'source-changes-verdict.json').read_text()) != changes:
        raise ValueError('Unexpected or unbound coverage source mutation')
    fixtures = {}
    for name in outputs:
        for phase, wanted in [('before', before.get(name)), ('after', after.get(name))]:
            if not isinstance(wanted, str) or UP.digest(directory / 'tracked-fixtures' / phase / name) != wanted:
                raise ValueError('Corrupt original coverage tracked fixture')
        fixtures[name] = {'before_sha256': before[name], 'after_sha256': after[name]}
    if json.loads((directory / 'runtime-fixtures.json').read_text()) != fixtures: raise ValueError('Coverage runtime fixture hashes differ')
    config = json.loads((directory / 'foundry-config.json').read_text()); C.validate_config(config, seed)
    build_config = json.loads((directory / 'build-config.json').read_text())
    bindings = json.loads((directory / 'signature-bindings.json').read_text()); compiled = json.loads((directory / 'compiled.json').read_text())
    if 'packages/contracts-bedrock/scripts/go-ffi/go-ffi' not in compiled or not any('/forge-artifacts/' in name for name in compiled):
        raise ValueError('Missing complete coverage compiler bindings')
    for row in bindings.values():
        if not row['artifacts'] or any(compiled.get(name) != value for name, value in row['artifacts'].items()):
            raise ValueError('Coverage compiler signatures differ from bound artifacts')
    selection = {phase: UP.selection(json.loads((directory / (phase + '-discovery.json')).read_text()), bindings) for phase in C.PHASES}
    if selection != {k: [tuple(r) for r in v] for k, v in json.loads((directory / 'selection.json').read_text()).items()} or \
       any(identity.rsplit(':', 1)[0] not in files for rows in selection.values() for identity, _ in rows):
        raise ValueError('Coverage selection differs from complete discovery/compiler signatures')
    if json.loads((directory / 'compile-only.json').read_text()) != {'tests': 0, 'selected_cases': {phase: len(rows) for phase, rows in selection.items()}}:
        raise ValueError('Coverage producer executed tests or omitted its complete selection')
    commands = {'files': CS.FILE_COMMANDS['standard'], 'submodules-sync': ['git', '-C', root, 'submodule', 'sync', '--recursive'],
                'submodules-init': ['git', '-C', root, '-c', 'protocol.file.allow=never', 'submodule', 'update', '--init', '--recursive', '--jobs', '8'],
                'build-config': ['forge', 'config', '--json'], 'go-ffi': ['just', 'build-go-ffi'], 'source-build': ['just', 'build-source'],
                'foundry-config': ['forge', 'config', '--json'], 'coverage-build': ['forge', 'build'],
                **{phase + '-discovery': argv for phase, argv in C.DISCOVERY.items()},
                **{phase + '/tests': argv for phase, argv in C.COMMANDS.items()}}
    if provider == 'rwx':
        if json.loads((directory / 'runtime-config.json').read_text()) != config or CS.file_selection((directory / 'runtime-files.log').read_text()) != files:
            raise ValueError('Coverage runtime settings or selection differ from preparation')
        preparation = json.loads((directory / 'preparation-manifest.json').read_text())
        if preparation['exit_code'] or preparation['report_errors'] or not C.PREPARED <= set(preparation['original_sha256']):
            raise ValueError('Failed or incomplete coverage producer')
        for name, value in preparation['original_sha256'].items():
            if UP.digest(directory / ('preparation-settings.json' if name == 'settings.json' else name)) != value:
                raise ValueError('Corrupt original coverage producer report')
        old = json.loads((directory / 'preparation-settings.json').read_text())
        if any(old[k] != settings[k] for k in set(settings) - {'rwx_run_id', 'rwx_task_attempt'}):
            raise ValueError('Coverage producer/runtime settings differ')
        commands.update({'runtime-files': CS.FILE_COMMANDS['standard'], 'runtime-config': ['forge', 'config', '--json']})
    commands['archive-preflight/pinned-block' if sealed_preflight else 'pinned-block'] = ['just', 'print-pinned-block-number']
    stages = {str(p.relative_to(directory)).removesuffix('.stage.json') for p in directory.rglob('*.stage.json')}
    if stages != set(commands): raise ValueError('Missing original coverage command or unexplained retry')
    for name, argv in commands.items():
        row = json.loads((directory / (name + '.stage.json')).read_text())
        if (row['argv'], row['cwd'], row['exit_code']) != (argv, root + '/packages/contracts-bedrock', 0):
            raise ValueError('Wrong or failed original coverage command')
        if UP.digest(directory / (name + ('.json' if name.endswith(('config', 'discovery')) else '.log'))) != row['stdout_sha256'] or \
           UP.digest(directory / (name + '.stderr.log')) != row['stderr_sha256']:
            raise ValueError('Coverage stage output hashes differ from originals')
    block = json.loads((directory / 'block.json').read_text())
    if block['source_sha'] != sha or block['chain_id'] != 1 or block['policy'] != 'Just current-day 00:00 UTC' or \
       type(block['number']) is not int or block['number'] < 0 or not re.fullmatch('0x[0-9a-f]{64}', block['hash']):
        raise ValueError('Wrong or unbound original coverage archive block')
    prefix = directory / 'archive-preflight' if sealed_preflight else directory
    if (prefix / 'pinned-block.log').read_text().strip() != str(block['number']):
        raise ValueError('Original daily block discovery differs from executed archive block')
    if sealed_preflight:
        preflight = json.loads((prefix / 'preflight-manifest.json').read_text())
        if preflight['exit_code'] or preflight['report_errors'] or json.loads((prefix / 'block.json').read_text()) != block:
            raise ValueError('Failed or mismatched coverage archive preflight')
        for name, value in preflight['original_sha256'].items():
            if UP.digest(prefix / name) != value: raise ValueError('Corrupt original coverage preflight')
    phases = {}
    for phase in C.PHASES:
        d = directory / phase; attribution = json.loads((d / 'original.attribution.json').read_text())
        events = C.original_evidence((d / 'tests.log').read_text(), attribution); cases = events['cases']
        if events != json.loads((d / 'original-events.json').read_text()): raise ValueError('Coverage event accounting differs from complete originals')
        if cases != json.loads((d / 'original-cases.json').read_text()) or any(r['outcome'] == 'fail' for r in cases):
            raise ValueError('Coverage verdicts differ from original reports or contain a failure')
        derived(d / 'derived.junit.xml', phase, cases)
        coverage = C.accounting(d / 'derived.junit.xml', phase, selection[phase], bindings)
        if coverage != json.loads((d / 'coverage.json').read_text()): raise ValueError('Coverage case accounting differs from original reports')
        records = C.lcov((d / 'original.lcov.info').read_text(), Path(root) / 'packages/contracts-bedrock')
        if records != [{'source': r['source'], 'entries': [tuple(v) for v in r['entries']]} for r in json.loads((d / 'lcov-records.json').read_text())]:
            raise ValueError('LCOV entry accounting differs from original file')
        sources = json.loads((d / 'coverage-source-sha256.json').read_text()); wanted = {r['source'] for r in records}
        wanted.update(C.source_name(item['source'], Path(root) / 'packages/contracts-bedrock') for r in attribution['tests'] for item in r['covered'])
        if set(sources) != wanted or any(not re.fullmatch('[0-9a-f]{64}', value) or \
           ('packages/contracts-bedrock/' + name in before and sources[name] != before['packages/contracts-bedrock/' + name]) for name, value in sources.items()):
            raise ValueError('Coverage source items lack complete original input hashes')
        if phase == 'upgrade' and ('Running upgrade tests at block ' + str(block['number'])) not in (d / 'tests.log').read_text():
            raise ValueError('Original upgrade coverage did not confirm the pinned archive block')
        phases[phase] = {'coverage': coverage, 'attribution': C.attribution_summary(attribution), 'events': events,
                         'lcov': records, 'source_sha256': sources}
    methods = {name: {'methods': row['methods'], 'deployable': UP.deployable(row)} for name, row in bindings.items()
               if UP.deployable(row) or any(method.startswith(('test', 'invariant')) for method in row['methods'])}
    return {'settings': settings, 'files': files, 'selection': selection, 'phases': phases, 'block': block, 'source_after_verdict': after,
            'fixtures': fixtures, 'config': UP.REPORT.normalize(config, root), 'build_config': UP.REPORT.normalize(build_config, root),
            'methods': methods, 'submodules': UP.SUBMODULES.revisions((directory / 'submodules.txt').read_text()), 'original_sha256': hashes}


def compare(directories, feature, sha):
    if not re.fullmatch('[0-9a-f]{40}', sha): raise ValueError('Expected full coverage benchmark SHA')
    empty = []; data = {p: report(d, feature, sha, p, empty) for p, d in directories.items()}; a, b = data['circle'], data['rwx']
    for key in ('source_sha', 'feature', 'branch', 'profile', 'source_build_profile', 'benchmark_seed', 'authority', 'phases',
                'forge', 'go', 'just', 'input_sha256', 'runtime_output_paths', 'rpc_input_name'):
        if a['settings'][key] != b['settings'][key]: raise ValueError('Coverage settings differ at ' + key)
    for key in ('files', 'selection', 'phases', 'block', 'source_after_verdict', 'fixtures', 'config', 'build_config', 'methods', 'submodules'):
        if a[key] != b[key]: raise ValueError('Complete original coverage parity differs at ' + key)
    return {'source_sha': sha, 'feature': feature, 'verified_parity': True, 'selection': a['selection'], 'file_selection': a['files'],
            'phases': a['phases'], 'block': a['block'], 'settings': {p: d['settings'] for p, d in data.items()},
            'original_sha256': {p: d['original_sha256'] for p, d in data.items()}, 'manifest_declared_empty_logs': empty}


if __name__ == '__main__':
    p = argparse.ArgumentParser(description=__doc__); p.add_argument('--circle', type=Path, required=True); p.add_argument('--rwx', type=Path, required=True)
    p.add_argument('--sha', required=True); p.add_argument('--feature', choices=CS.FEATURES, required=True); p.add_argument('--output', type=Path, required=True)
    a = p.parse_args(); result = compare({'circle': a.circle, 'rwx': a.rwx}, a.feature, a.sha); UP.write(a.output, result)
    print(json.dumps({'source_sha': a.sha, 'feature': a.feature, 'verified_parity': True,
                      'outcomes': {phase: r['coverage']['outcomes'] for phase, r in result['phases'].items()}}))
