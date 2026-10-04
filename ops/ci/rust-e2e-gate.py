#!/usr/bin/env python3
"""Require complete fresh E2E shard coverage and a successful full release build."""
import argparse
from collections import Counter
import importlib.util
import json
from pathlib import Path
import re
import sys

SPEC = importlib.util.spec_from_file_location('e2e', Path(__file__).with_name('rust-e2e.py'))
E2E = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(E2E)


def originals(directory):
    final = E2E.read(directory / 'final.json')
    if final['exit_code'] != 0 or final['report_errors']: raise ValueError('Unsuccessful E2E dependency or verdict')
    if not final['original_sha256']: raise ValueError('Empty E2E original-report manifest')
    for name, sha in final['original_sha256'].items():
        if Path(name).is_absolute() or '..' in Path(name).parts or not re.fullmatch('[0-9a-f]{64}', sha):
            raise ValueError('Unsafe original E2E report path or digest')
        if E2E.GO.digest(directory / name) != sha: raise ValueError('Missing or corrupt original E2E report')
    return final


def validate(release, reports, sha, *, provider='rwx'):
    originals(release)
    settings, coverage = E2E.read(release / 'settings.json'), E2E.read(release / 'coverage.json')
    if (settings['source_sha'] != sha or settings['suite'] != 'rust-e2e' or settings['job'] != 'release'
            or settings['profile'] != 'release' or settings['features'] != ['default']
            or coverage['binding']['source_sha'] != sha or not coverage['packages'] or not coverage['targets']):
        raise ValueError('Incomplete E2E full-release provenance')
    grouped, summary = {job: [] for job in E2E.JOBS}, {}
    for directory in reports:
        originals(directory)
        record, observed = E2E.read(directory / 'selection.json'), E2E.read(directory / 'coverage.json')
        job = record['settings']['job']
        if job not in grouped or record['settings'] != E2E.settings(job) or record['source_sha'] != sha:
            raise ValueError('E2E source, job or settings mismatch')
        if record['provider'] != provider: raise ValueError('E2E provider mismatch')
        if provider == 'rwx' and (not record['rwx_run_id'] or str(record['rwx_task_attempt']) != '1'):
            raise ValueError('Missing fresh E2E run identity or uninvestigated task retry')
        E2E.invocations(record, directory)
        if observed['missing'] or observed['extra'] or observed['duplicates'] or observed['assigned'] != record['assigned_tests']:
            raise ValueError('Incomplete E2E test coverage')
        cases, terminal = {}, Counter()
        for event in E2E.PROJECT.events(directory / 'original.json'):
            if event.get('Package') != E2E.PREFIX + E2E.JOBS[job]['package']: raise ValueError('Unexpected original E2E package')
            if event['Action'] == 'fail': raise ValueError('Failed original E2E package or test verdict')
            name = event.get('Test')
            if name and event['Action'] in ('pass', 'skip', 'fail'):
                if name in cases or event['Action'] == 'fail': raise ValueError('Failed or duplicate original E2E verdict')
                cases[name] = {'outcome': event['Action'], 'elapsed': event.get('Elapsed')}
                if '/' not in name: terminal[name] += 1
        if cases != observed['cases'] or terminal != Counter(record['assigned_tests']):
            raise ValueError('E2E coverage differs from complete original verdicts')
        expected_dependencies = {'go', 'contracts-e2e', 'rust-e2e-release'}
        if job in ('restart', 'simple-kona', 'simple-kona-sequencer'): expected_dependencies.add('prestate')
        if provider == 'rwx':
            paths = {p.stem: p for p in (directory / 'dependencies').glob('*.json')}
            if paths.keys() != expected_dependencies: raise ValueError('Missing or extra E2E dependency inputs')
            for kind, path in paths.items():
                dependency = E2E.read(path)
                if dependency['kind'] != kind or dependency['commit_sha'] != sha or not dependency['files']:
                    raise ValueError('E2E runtime dependency provenance differs')
        grouped[job].append(record)
    for job, rows in grouped.items():
        if sorted(r['shard_index'] for r in rows) != list(range(E2E.JOBS[job]['shards'])):
            raise ValueError('Missing or duplicate E2E shard reports')
        names = rows[0]['tests']; assigned = Counter(n for r in rows for n in r['assigned_tests'])
        if any(r['tests'] != names or r['excluded_listing'] != rows[0]['excluded_listing'] for r in rows):
            raise ValueError('E2E shard discovery differs')
        if assigned != Counter(names): raise ValueError('E2E initial selection is not assigned exactly once')
        summary[job] = {'top_level_tests': len(names), 'shards': len(rows), 'fresh': True, 'retries': 0}
    return {'source_sha': sha, 'suite': 'rust-e2e', 'jobs': summary,
            'release_packages': len(coverage['packages']), 'release_targets': len(coverage['targets']), 'passed': True}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--release', required=True, type=Path)
    parser.add_argument('--sha', required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('reports', nargs='+', type=Path)
    args = parser.parse_args()
    try: result = validate(args.release, args.reports, args.sha)
    except (ValueError, OSError, KeyError) as error:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        E2E.write(args.output, {'source_sha': args.sha, 'passed': False, 'error': str(error)})
        print(str(error), file=sys.stderr); sys.exit(1)
    args.output.parent.mkdir(parents=True, exist_ok=True); E2E.write(args.output, result)
