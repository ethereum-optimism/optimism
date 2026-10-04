#!/usr/bin/env python3
"""Compare complete standard/changed-file originals, settings and assignments."""
import argparse
import json
from pathlib import Path
import re

import importlib.util
SPEC = importlib.util.spec_from_file_location('suites', Path(__file__).with_name('contract-suites.py'))
CS = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(CS)
UP = CS.UP


def report(directory, suite, feature, sha, provider, empty):
    required = CS.PREPARED | {'original.junit.xml', 'coverage.json', 'tests.stage.json', 'junit-nonempty.stage.json',
                              'lint-test-names.stage.json', 'source-after-verdict.json', 'runtime-fixtures.json'}
    if provider == 'rwx': required |= {'runtime-config.json', 'runtime-config.stage.json', 'runtime-files.log', 'runtime-files.stage.json',
                                       'preparation-manifest.json', 'preparation-settings.json'}
    else: required |= {'split.log', 'split.stage.json'}
    try:
        hashes = UP.ORIGINALS.originals(directory, required, empty, provider + '/' + suite + '-' + feature)
    except OSError as error:
        raise ValueError('Missing original contract suite report file') from error
    physical = {str(p.relative_to(directory)) for p in directory.rglob('*') if p.is_file() and p.name != 'final.json'}
    if physical != set(hashes): raise ValueError('Unsealed or missing original contract suite files')
    settings = json.loads((directory / 'settings.json').read_text())
    if (settings['source_sha'], settings['suite'], settings['feature'], settings['provider']) != \
       (sha, suite, feature, 'circleci' if provider == 'circle' else 'rwx'):
        raise ValueError('Wrong contract suite source, occurrence or provider')
    profile = 'ciheavy' if suite == 'modified' else 'ci' if settings['branch'] == 'develop' else 'liteci'
    if settings['profile'] != profile or not settings['input_sha256']: raise ValueError('Wrong effective contract branch profile or missing source inputs')
    if json.loads((directory / 'source-after-preparation.json').read_text()) != settings['input_sha256']:
        raise ValueError('Contract source inputs changed during preparation')
    test_list = 'find test -name "*.t.sol"' if suite == 'standard' else CS.FILE_COMMANDS[suite][-1]
    if settings['test_list'] != test_list: raise ValueError('Wrong original Circle contract test-list parameter')
    if provider == 'rwx' and (not settings['rwx_run_id'] or str(settings['rwx_task_attempt']) != '1'):
        raise ValueError('Reused contract verdict or uninvestigated task retry')
    files = CS.file_selection((directory / 'files.log').read_text())
    if not files: raise ValueError('Empty original contract selection cannot establish coverage')
    chosen = {'files': files, 'partitions': [{'index': 0, 'files': files}], 'match_path': CS.match_path(files)}
    outputs = CS.runtime_outputs(files)
    if settings.get('runtime_output_paths') != outputs: raise ValueError('Unexpected tracked runtime output role')
    after = json.loads((directory / 'source-after-verdict.json').read_text())
    changes = {name: {'before': settings['input_sha256'].get(name), 'after': after.get(name)}
               for name in sorted(set(settings['input_sha256']) | set(after))
               if settings['input_sha256'].get(name) != after.get(name)}
    if set(changes) - set(outputs): raise ValueError('Contract source inputs changed during verdict')
    if changes and json.loads((directory / 'source-changes-verdict.json').read_text()) != changes:
        raise ValueError('Tracked runtime fixture change evidence differs')
    fixtures = {}
    for name in outputs:
        for phase, expected in [('before', settings['input_sha256'].get(name)), ('after', after.get(name))]:
            if not isinstance(expected, str) or UP.digest(directory / 'tracked-fixtures' / phase / name) != expected:
                raise ValueError('Corrupt original tracked runtime fixture')
        fixtures[name] = {'before_sha256': settings['input_sha256'][name], 'after_sha256': after[name]}
    if json.loads((directory / 'runtime-fixtures.json').read_text()) != fixtures:
        raise ValueError('Runtime fixture hashes differ from originals')
    if json.loads((directory / 'file-selection.json').read_text()) != chosen: raise ValueError('Missing, duplicate or wrong contract file assignment')
    if provider == 'circle' and CS.file_selection((directory / 'split.log').read_text()) != files:
        raise ValueError('Original Circle timing split differs from complete file selection')
    expected_files = {p.removeprefix('packages/contracts-bedrock/') for p in settings['input_sha256']
                      if p.startswith('packages/contracts-bedrock/test/') and p.endswith('.t.sol')}
    if (suite == 'standard' and set(files) != expected_files) or not set(files) <= expected_files:
        raise ValueError('Incomplete original standard discovery or missing changed-file inputs')
    config = json.loads((directory / 'foundry-config.json').read_text()); CS.validate_config(config, suite)
    if provider == 'rwx':
        if json.loads((directory / 'runtime-config.json').read_text()) != config or \
           CS.file_selection((directory / 'runtime-files.log').read_text()) != files:
            raise ValueError('Runtime contract settings or file selection changed')
        preparation = json.loads((directory / 'preparation-manifest.json').read_text())
        if preparation['exit_code'] or preparation['report_errors'] or not CS.PREPARED <= set(preparation['original_sha256']):
            raise ValueError('Failed or incomplete contract compilation producer')
        for name, digest in preparation['original_sha256'].items():
            path = directory / ('preparation-settings.json' if name == 'settings.json' else name)
            if UP.digest(path) != digest: raise ValueError('Corrupt original contract preparation')
        before = json.loads((directory / 'preparation-settings.json').read_text())
        for key in set(settings) - {'rwx_run_id', 'rwx_task_attempt'}:
            if settings[key] != before[key]: raise ValueError('Contract producer settings differ from runtime')
    methods = json.loads((directory / 'signature-bindings.json').read_text())
    compiled = json.loads((directory / 'compiled.json').read_text())
    if not compiled or 'packages/contracts-bedrock/scripts/go-ffi/go-ffi' not in compiled:
        raise ValueError('Missing complete contract compiler artifact binding')
    for row in methods.values():
        if not row['artifacts'] or any(compiled.get(name) != digest for name, digest in row['artifacts'].items()):
            raise ValueError('Compiler test signature binding differs from original artifacts')
    selected = UP.selection(json.loads((directory / 'discovery.json').read_text()), methods)
    if selected != [tuple(row) for row in json.loads((directory / 'selection.json').read_text())] or \
       any(identity.rsplit(':', 1)[0] not in files for identity, _ in selected):
        raise ValueError('Contract test selection differs from complete original discovery')
    coverage = UP.junit(directory / 'original.junit.xml', selected, methods)
    if json.loads((directory / 'coverage.json').read_text()) != coverage: raise ValueError('Contract coverage differs from original JUnit')
    if json.loads((directory / 'compile-only.json').read_text()) != {'tests': 0, 'selected_cases': len(selected), 'suite': suite, 'feature': feature}:
        raise ValueError('Contract producer did not retain a compile-only observation')
    root = settings['workspace_root']; cwd = root + '/packages/contracts-bedrock'
    commands = {'files': CS.FILE_COMMANDS[suite], 'foundry-config': ['forge', 'config', '--json'],
                'submodules-sync': ['git', '-C', root, 'submodule', 'sync', '--recursive'],
                'submodules-init': ['git', '-C', root, '-c', 'protocol.file.allow=never', 'submodule',
                                    'update', '--init', '--recursive', '--jobs', '8'],
                'go-ffi': ['just', 'build-go-ffi'], 'contracts-build': ['forge', 'build'],
                'discovery': ['forge', 'test', '--list', '--json', '--match-path', chosen['match_path']],
                'tests': ['forge', 'test', '--match-path', chosen['match_path'], '--junit'],
                'junit-nonempty': ['./scripts/checks/check-junit-tests-ran.sh', str(directory / 'original.junit.xml')],
                'lint-test-names': ['just', 'lint-forge-tests-check-no-build']}
    if provider == 'circle': commands['split'] = CS.SPLIT_COMMAND
    else: commands.update({'runtime-files': CS.FILE_COMMANDS[suite], 'runtime-config': ['forge', 'config', '--json']})
    if suite == 'modified':
        fetch = ['git', 'fetch', '--no-tags', 'origin', '+refs/heads/develop:refs/remotes/origin/develop']
        commands['fetch-develop'] = fetch
        if provider == 'rwx': commands['runtime-fetch-develop'] = fetch
        if any(not re.fullmatch('[0-9a-f]{40}', settings[k]) for k in ('target_sha', 'merge_base_sha')):
            raise ValueError('Missing original changed-file target revisions')
    stages = {p.name.removesuffix('.stage.json') for p in directory.glob('*.stage.json')}
    if stages != set(commands): raise ValueError('Missing original contract command or uninvestigated retry')
    for name, argv in commands.items():
        row = json.loads((directory / (name + '.stage.json')).read_text())
        # The report location is provider-specific; the original guard's path
        # must still refer to this invocation's retained original XML.
        if name == 'junit-nonempty':
            expected = root + '/.ci/contract-suites/' + suite + '-' + feature + '/run/original.junit.xml'
            argv = ['./scripts/checks/check-junit-tests-ran.sh', expected]
        if (row['argv'], row['cwd'], row['exit_code']) != (argv, cwd, 0): raise ValueError('Wrong or failed original contract command')
    return {'settings': settings, 'files': chosen, 'selection': selected, 'coverage': coverage,
            'config': UP.ORIGINALS.normalize(config, root), 'original_sha256': hashes, 'compiled_sha256': compiled,
            'source_after_verdict': after, 'runtime_fixtures': fixtures,
            'submodules': UP.SUBMODULES.revisions((directory / 'submodules.txt').read_text()),
            'methods': {name: {'methods': row['methods'], 'deployable': UP.deployable(row)} for name, row in methods.items()}}


def compare(directories, suite, feature, sha):
    if not re.fullmatch('[0-9a-f]{40}', sha): raise ValueError('Expected full contract suite benchmark SHA')
    empty = []; data = {p: report(d, suite, feature, sha, p, empty) for p, d in directories.items()}; a, b = data['circle'], data['rwx']
    for key in ('source_sha', 'suite', 'feature', 'branch', 'profile', 'test_list', 'forge', 'go', 'just', 'input_sha256',
                'runtime_output_paths', 'target_sha', 'merge_base_sha'):
        if a['settings'].get(key) != b['settings'].get(key): raise ValueError('Contract settings differ at ' + key)
    for key in ('files', 'selection', 'coverage', 'config', 'submodules', 'methods', 'source_after_verdict', 'runtime_fixtures'):
        if a[key] != b[key]: raise ValueError('Complete original contract parity differs at ' + key)
    return {'source_sha': sha, 'suite': suite, 'feature': feature, 'verified_parity': True,
            'file_selection': a['files'], 'selection': a['selection'], 'coverage': a['coverage'],
            'settings': {p: d['settings'] for p, d in data.items()}, 'original_sha256': {p: d['original_sha256'] for p, d in data.items()},
            'compiled_sha256': {p: d['compiled_sha256'] for p, d in data.items()},
            'source_after_verdict': a['source_after_verdict'], 'runtime_fixtures': a['runtime_fixtures'],
            'manifest_declared_empty_logs': empty}


if __name__ == '__main__':
    p = argparse.ArgumentParser(description=__doc__); p.add_argument('--circle', type=Path, required=True); p.add_argument('--rwx', type=Path, required=True)
    p.add_argument('--sha', required=True); p.add_argument('--suite', choices=CS.SUITES, required=True); p.add_argument('--feature', choices=CS.FEATURES, required=True)
    p.add_argument('--output', type=Path, required=True); a = p.parse_args()
    result = compare({'circle': a.circle, 'rwx': a.rwx}, a.suite, a.feature, a.sha); UP.write(a.output, result)
    print(json.dumps({'source_sha': a.sha, 'suite': a.suite, 'feature': a.feature, 'verified_parity': True, 'outcomes': result['coverage']['outcomes']}))
