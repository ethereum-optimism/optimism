#!/usr/bin/env python3
"""Compare complete original Circle/RWX reports for the remaining Rust checks."""
import argparse
import hashlib
import json
from pathlib import Path
import re

JOBS = ('wasm-unknown', 'wasm-wasi', 'zepter', 'typos', 'registry', 'interop')
STAGES = {'wasm-unknown': ('wasm-target', 'wasm-list', 'wasm'),
          'wasm-wasi': ('wasm-target', 'wasm-list', 'wasm'),
          'zepter': ('zepter',), 'typos': ('typos',),
          'registry': ('registry-clean', 'registry', 'registry-diff'),
          'interop': ('superchain-go', 'interop')}
EMPTY_LOGS = {'workspace.log', 'typos.log', 'registry-diff.log',
              'interop-go.stderr', 'interop-rust.stderr'}


def read(path):
    return json.loads(path.read_text())


def digest(path):
    with path.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def package_manifest(metadata):
    packages = {p['id']: p for p in metadata['packages']}
    members = metadata['workspace_members']
    if not members or len(members) != len(set(members)):
        raise ValueError('Empty or duplicate workspace membership')
    result = {}
    for member in members:
        p = packages[member]
        if p['name'] in result:
            raise ValueError('Duplicate workspace package name')
        result[p['name']] = {k: p[k] for k in ('version', 'features')} | {
            'targets': [{k: t[k] for k in ('name', 'kind', 'required-features') if k in t}
                        for t in p['targets']]}
    return result


def compare(root, sha):
    if not re.fullmatch('[0-9a-f]{40}', sha):
        raise ValueError('Expected a full source SHA')
    result = {'source_sha': sha, 'verified_parity': False, 'jobs': {},
              'original_sha256': {}, 'manifest_declared_empty_logs': []}
    for provider in ('circle', 'rwx'):
        result['original_sha256'][provider] = {}
        for job in JOBS:
            directory = root / provider / job
            settings, final = read(directory / 'settings.json'), read(directory / 'final.json')
            if settings['source_sha'] != sha or settings['job'] != job:
                raise ValueError(f'{provider}/{job}: source or job mismatch')
            if final['exit_code'] != 0 or final['report_errors'] or not final['stages']:
                raise ValueError(f'{provider}/{job}: unsuccessful or incomplete report')
            if provider == 'rwx' and (not settings.get('rwx_run_id') or str(settings.get('rwx_task_attempt')) != '1'):
                raise ValueError(f'{provider}/{job}: missing fresh identity or task retries require investigation')
            hashes = final['original_sha256']
            required = {'settings.json', 'workspace.json', 'checks.junit.xml'}
            stages = ('workspace',) + STAGES[job]
            required.update(name + suffix for name in stages for suffix in ('.log', '.stage.json'))
            if job.startswith('wasm-'):
                required.update({'wasm-coverage.json', 'wasm-artifacts.json'})
            if job == 'registry':
                required.add('registry-coverage.json')
                required.update(f'registry-{phase}-{name}.json' for phase in ('before', 'after')
                                for name in ('chainList', 'configs', 'depsets'))
            if job == 'interop':
                required.add('interop-coverage.json')
                required.update(f'interop-{language}.{suffix}' for language in ('go', 'rust')
                                for suffix in ('stdout', 'stderr', 'exit'))
            if not required <= hashes.keys() or set(final['stages']) != set(stages):
                raise ValueError(f'{provider}/{job}: incomplete original manifest')
            for name in stages:
                if final['stages'][name] != read(directory / (name + '.stage.json')):
                    raise ValueError(f'{provider}/{job}: summary differs from original stage {name}')
            for name, expected in hashes.items():
                path = Path(name)
                if path.is_absolute() or '..' in path.parts or not re.fullmatch('[0-9a-f]{64}', expected):
                    raise ValueError('Invalid original artifact path or checksum')
                original = directory / path
                if not original.exists():
                    # Circle's artifact API omits empty files. Never reconstruct
                    # originals or accept a missing nonempty stream.
                    if provider != 'circle' or name not in EMPTY_LOGS or expected != hashlib.sha256(b'').hexdigest():
                        raise ValueError(f'{provider}/{job}: missing original {name}')
                    result['manifest_declared_empty_logs'].append({'job': job, 'name': name, 'sha256': expected})
                elif digest(original) != expected:
                    raise ValueError(f'{provider}/{job}: corrupt original {name}')
            result['original_sha256'][provider][job] = hashes
    for job in JOBS:
        a, b = (root / provider / job for provider in ('circle', 'rwx'))
        sa, sb = read(a / 'settings.json'), read(b / 'settings.json')
        for key in ('job', 'suite', 'source_sha', 'input_sha256', 'rustc', 'cargo', 'nextest',
                    'incremental', 'rustflags', 'rustdocflags', 'superchain_revision', 'extra_tool'):
            if sa.get(key) != sb.get(key):
                raise ValueError(f'{job}: settings differ at {key}')
        packages = package_manifest(read(a / 'workspace.json'))
        if packages != package_manifest(read(b / 'workspace.json')):
            raise ValueError(f'{job}: complete workspace manifests differ')
        stages_a, stages_b = read(a / 'final.json')['stages'], read(b / 'final.json')['stages']
        projection = lambda stages: {name: {k: s[k] for k in ('argv', 'cwd', 'exit_code')}
                                     for name, s in stages.items()}
        if projection(stages_a) != projection(stages_b) or any(s['exit_code'] != 0 for s in stages_a.values()):
            raise ValueError(f'{job}: original commands, working directories or outcomes differ')
        info = {'packages': len(packages), 'stages': projection(stages_a), 'outcome': 'pass', 'retries': 0}
        filename = {'wasm-unknown': 'wasm-coverage.json', 'wasm-wasi': 'wasm-coverage.json',
                    'registry': 'registry-coverage.json', 'interop': 'interop-coverage.json'}.get(job)
        if filename:
            coverage = read(a / filename)
            if coverage != read(b / filename):
                raise ValueError(f'{job}: full coverage differs')
            info['coverage'] = coverage
        if job.startswith('wasm-'):
            artifacts_a, artifacts_b = read(a / 'wasm-artifacts.json'), read(b / 'wasm-artifacts.json')
            for key in ('source_sha', 'target', 'rustc'):
                if artifacts_a[key] != artifacts_b[key]:
                    raise ValueError('WASM artifact provenance differs')
            if artifacts_a['artifacts'].keys() != artifacts_b['artifacts'].keys():
                raise ValueError('WASM artifact inventories differ')
            info['library_artifacts'] = {'circle': artifacts_a, 'rwx': artifacts_b,
                'byte_identical': artifacts_a == artifacts_b,
                'comparison': 'Inputs, settings, package execution and successful library production; host archive byte reproducibility is not required.'}
        if job == 'interop':
            if (a / 'interop-go.stdout').read_bytes() != (b / 'interop-go.stdout').read_bytes():
                raise ValueError('Interop original stdout differs between providers')
        if (a / 'checks.junit.xml').read_bytes() != (b / 'checks.junit.xml').read_bytes():
            raise ValueError(f'{job}: original JUnit differs')
        result['jobs'][job] = info
    result['verified_parity'] = True
    return result


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('root', type=Path)
    parser.add_argument('sha')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    evidence = compare(args.root, args.sha)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(evidence, indent=2, sort_keys=True) + '\n')
    print(json.dumps({'verified_parity': True, 'sha': args.sha, 'jobs': list(evidence['jobs'])}))
