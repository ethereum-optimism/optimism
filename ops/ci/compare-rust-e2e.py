#!/usr/bin/env python3
"""Compare complete original Rust E2E selection, builds and fresh Go verdicts."""
import argparse
from collections import Counter
import importlib.util
import json
from pathlib import Path
import re


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


E2E = helper('rust-e2e')
COMPARE = helper('compare-ci')
EXTRA = helper('compare-rust-extra')
CONTRACTS = helper('compare-contract-artifacts')
EMPTY = __import__('hashlib').sha256(b'').hexdigest()
RELEASE_BINDING = ('source_sha', 'profile', 'features', 'scope', 'input_sha256', 'source_trees',
                   'rustc', 'cargo', 'mold', 'rustflags', 'incremental')


def originals(directory, required, empty, label):
    final = E2E.read(directory / 'final.json')
    if final['exit_code'] != 0 or final['report_errors']: raise ValueError('Failed or incomplete original E2E report: ' + label)
    hashes = final['original_sha256']
    if not required <= hashes.keys(): raise ValueError('Incomplete original E2E file manifest: ' + label)
    for name, sha in hashes.items():
        path = Path(name)
        if path.is_absolute() or '..' in path.parts or not re.fullmatch('[0-9a-f]{64}', sha):
            raise ValueError('Invalid original E2E report path or hash')
        if not (directory / path).exists() and sha == EMPTY and label.startswith('circle/'):
            empty.append({'report': label, 'path': name, 'sha256': sha})
            # Provider manifests establish an empty file, never a missing verdict.
            (directory / path).parent.mkdir(parents=True, exist_ok=True); (directory / path).write_bytes(b'')
        if E2E.GO.digest(directory / path) != sha: raise ValueError('Missing or corrupt original E2E report: ' + label + '/' + name)
    return hashes


def normalize(value, root):
    if isinstance(value, str): return value.replace(root, '<repo>')
    if isinstance(value, dict): return {k: normalize(v, root) for k, v in value.items()}
    if isinstance(value, list): return [normalize(v, root) for v in value]
    return value


def test_report(directory, job, sha, provider, empty, originals_index):
    required = {'selection.json', 'coverage.json', 'original.json', 'native.json', 'native.metadata.json', 'assigned.txt', 'environment.nul'}
    hashes = originals(directory, required, empty, provider + '/' + directory.name)
    originals_index[directory.name] = hashes
    record = E2E.read(directory / 'selection.json')
    if (record['source_sha'] != sha or record['settings'] != E2E.settings(job)
            or record['provider'] != {'circle': 'circleci', 'rwx': 'rwx'}[provider]):
        raise ValueError('E2E job, source or effective workload differs')
    if provider == 'rwx' and (not record['rwx_run_id'] or str(record['rwx_task_attempt']) != '1'):
        raise ValueError('Missing fresh native E2E identity or task retry needs investigation')
    metadata = {'package_prefix': E2E.PREFIX + E2E.JOBS[job]['package']}
    source = {'feature': job, 'id': directory.name}
    cases, failures = COMPARE.go_cases(directory / 'original.json', source, metadata)
    if failures or any(c['outcome'] == 'fail' or len(c['attempts']) != 1 for c in cases):
        raise ValueError('E2E package/case failure or unexpected retry')
    keys = {(c['suite'], c['name']): c for c in cases}
    junit = []
    for path in (directory / 'junit').glob('*.xml'):
        junit.extend(COMPARE.junit_cases(path, {'feature': job, 'id': directory.name}, metadata))
    jk = {(c['suite'], c['name']): c for c in junit}
    if len(jk) != len(junit) or jk.keys() != keys.keys() or any(jk[k]['outcome'] != keys[k]['outcome'] for k in keys):
        raise ValueError('E2E original JUnit and JSON verdict identities/outcomes differ')
    observed = E2E.read(directory / 'coverage.json')
    assigned = set(record['assigned_tests'])
    if {c['name'] for c in cases if '/' not in c['name']} != assigned:
        raise ValueError('E2E original verdicts differ from complete assigned selection')
    if observed['assigned'] != record['assigned_tests'] or observed['missing'] or observed['extra'] or observed['duplicates']:
        raise ValueError('Incomplete E2E observed coverage')
    E2E.invocations(record, directory)
    return record, keys, source['package_attempts']


def compare(root, sha):
    if not re.fullmatch('[0-9a-f]{40}', sha): raise ValueError('Expected full E2E source SHA')
    result = {'source_sha': sha, 'verified_parity': False, 'jobs': {}, 'original_sha256': {'circle': {}, 'rwx': {}},
              'manifest_declared_empty_logs': [], 'settings_differences': []}
    empty = result['manifest_declared_empty_logs']
    release_dirs = {p: root / p / 'release' for p in ('circle', 'rwx')}
    releases = {}
    for provider, directory in release_dirs.items():
        hashes = originals(directory, {'settings.json', 'coverage.json', 'workspace.json', 'workspace.stage.json', 'workspace.log',
            'build.json', 'build.log', 'build.stage.json', 'dependency.json', 'checks.junit.xml'}, empty, provider + '/release')
        result['original_sha256'][provider]['release'] = hashes
        settings, coverage = E2E.read(directory / 'settings.json'), E2E.read(directory / 'coverage.json')
        if settings['source_sha'] != sha or settings['suite'] != 'rust-e2e' or settings['job'] != 'release': raise ValueError('Release source or workload differs')
        if settings['provider'] != {'circle': 'circleci', 'rwx': 'rwx'}[provider]: raise ValueError('Release provider provenance differs')
        if coverage['binding'] != {k: settings[k] for k in RELEASE_BINDING}: raise ValueError('Release build inputs changed')
        stages = {}
        for name in ('workspace', 'build'):
            stage = E2E.read(directory / (name + '.stage.json'))
            if stage['exit_code'] != 0: raise ValueError('Failed original release stage')
            stages[name] = normalize({k: stage[k] for k in ('argv', 'cwd', 'exit_code')}, settings['workspace_root'])
        packages = EXTRA.package_manifest(E2E.read(directory / 'workspace.json'))
        targets = [{k: t[k] for k in ('package', 'target', 'kind', 'role', 'features', 'profile')} for t in coverage['targets']]
        if not targets or len({json.dumps(t, sort_keys=True) for t in targets}) != len(targets): raise ValueError('Empty or duplicate release target coverage')
        dependency = E2E.read(directory / 'dependency.json')
        if dependency['commit_sha'] != sha or dependency['kind'] != 'rust-e2e-release': raise ValueError('Release artifact provenance differs')
        if set(dependency['files']) != {v['path'] for v in coverage['binaries'].values()}: raise ValueError('Release binary archive coverage differs')
        if any(dependency['files'][v['path']] != v['sha256'] for v in coverage['binaries'].values()): raise ValueError('Release binary hashes differ from compiled outputs')
        releases[provider] = {'binding': coverage['binding'], 'packages': packages, 'targets': targets,
            'excluded': coverage['excluded_feature_gated_targets'], 'commands': stages, 'dependency': dependency, 'coverage': coverage}
    a, b = releases['circle'], releases['rwx']
    for field in ('binding', 'packages', 'targets', 'excluded', 'commands'):
        if a[field] != b[field]: raise ValueError('Complete release parity differs at ' + field)
    if a['coverage']['binaries'].keys() != b['coverage']['binaries'].keys(): raise ValueError('Full workspace release binary selection differs')
    result['jobs']['release'] = {'packages': len(a['packages']), 'targets': len(a['targets']), 'commands': a['commands'],
        'binaries': {name: {p: releases[p]['coverage']['binaries'][name] for p in releases} for name in a['coverage']['binaries']},
        'comparison': 'Full original Cargo selection, settings and target completion; ELF byte hashes are retained separately.'}
    for job, config in E2E.JOBS.items():
        selections, all_cases, histories = {}, {}, {}
        for provider in ('circle', 'rwx'):
            directories = sorted((root / provider).glob(job + '-[0-9]*'))
            if len(directories) != config['shards']: raise ValueError('Missing E2E shard report: ' + provider + '/' + job)
            rows, cases, history = [], {}, []
            for directory in directories:
                record, current, package_attempts = test_report(directory, job, sha, provider, empty, result['original_sha256'][provider])
                if cases.keys() & current.keys(): raise ValueError('Duplicate E2E case across shards')
                cases.update(current); rows.append(record); history.extend(package_attempts)
            if sorted(r['shard_index'] for r in rows) != list(range(config['shards'])): raise ValueError('Duplicate or missing E2E initial shard')
            names = rows[0]['tests']
            if not names or any(r['tests'] != names or r['excluded_listing'] != rows[0]['excluded_listing'] for r in rows): raise ValueError('E2E complete shard selection differs')
            if Counter(n for r in rows for n in r['assigned_tests']) != Counter(names): raise ValueError('E2E initial test assignment is not exactly once')
            selections[provider], all_cases[provider], histories[provider] = rows, cases, history
        a, b = selections['circle'][0], selections['rwx'][0]
        for field in ('source_sha', 'settings', 'tests', 'excluded_listing', 'durations', 'input_sha256', 'go_version', 'go_environment', 'branch'):
            if a[field] != b[field]: raise ValueError(job + ': E2E selection or settings differ at ' + field)
        if normalize(a['environment'], a['workspace_root']) != normalize(b['environment'], b['workspace_root']): raise ValueError('E2E runtime roles or binaries differ')
        # Proofs may deliberately use fewer workers than Circle's host nproc.
        # Other recipes retain Go's default; arbitrary flag drift is an error.
        if job != 'proof' and (a['parallel_flag'] is not None or b['parallel_flag'] is not None):
            raise ValueError('Node/reth E2E must preserve default Go parallelism')
        for field in ('effective_parallel', 'go_runtime_defaults'):
            if a[field] != b[field]: result['settings_differences'].append({'job': job, 'field': field, 'circle': a[field], 'rwx': b[field]})
        if a['runtime_settings'] != b['runtime_settings']: raise ValueError('E2E runtime setting differences need investigation: ' + job)
        ac, bc = all_cases['circle'], all_cases['rwx']
        if ac.keys() != bc.keys(): raise ValueError(job + ': Missing or extra original E2E case identities')
        skips = []
        for key in ac:
            if ac[key]['outcome'] != bc[key]['outcome']: raise ValueError(job + ': E2E case outcome differs: ' + str(key))
            if ac[key]['outcome'] == 'skip':
                sa, sb = (normalize(c[key]['skip_reason'], selections[p][0]['workspace_root']) for p, c in [('circle', ac), ('rwx', bc)])
                if sa is None or sa != sb: raise ValueError(job + ': Original E2E skip reason differs or is unknown: ' + str(key))
                skips.append({'package': key[0], 'name': key[1], 'reason': sa})
        result['jobs'][job] = {'settings': a['settings'], 'top_level_selection': a['tests'], 'cases': len(ac),
            'outcomes': dict(Counter(c['outcome'] for c in ac.values())), 'skips': skips, 'retries': 0,
            'shard_assignments': {p: [{'index': r['shard_index'], 'tests': r['assigned_tests']} for r in rows] for p, rows in selections.items()},
            'package_invocations': histories}
    for kind in ('contracts', 'prestate'):
        a, b = (E2E.read(root / p / kind / 'metadata.json') for p in ('circle', 'rwx'))
        for field in ('version', 'kind', 'commit_sha', 'settings', 'mise_sha256'):
            if a[field] != b[field] or (field == 'commit_sha' and a[field] != sha): raise ValueError(kind + ': dependency input binding differs')
        if kind == 'prestate' and a['files'] != b['files']: raise ValueError('Complete reproducible prestate file hashes differ')
        result['jobs'][kind] = {'metadata': {'circle': a, 'rwx': b}, 'file_hashes_equal': a['files'] == b['files']}
    result['jobs']['contracts']['complete_artifact_comparison'] = CONTRACTS.compare(
        {p: root / p / 'contracts' for p in ('circle', 'rwx')}, sha)
    # Setting differences remain explicit; they are not textual setting parity.
    result['verified_parity'] = True
    return result


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, required=True); parser.add_argument('--sha', required=True)
    parser.add_argument('--output', type=Path, required=True); args = parser.parse_args()
    result = compare(args.root, args.sha); args.output.write_text(json.dumps(result, indent=2, sort_keys=True) + '\n')
    print(json.dumps({'source_sha': args.sha, 'verified_parity': result['verified_parity'],
                      'jobs': {k: {p: v[p] for p in ('cases', 'outcomes', 'packages', 'targets') if p in v} for k, v in result['jobs'].items()},
                      'settings_differences': result['settings_differences']}))
