#!/usr/bin/env python3
"""Collect the original acceptance reports even when the shared runner fails."""
import importlib.util
import json
from collections import Counter
from pathlib import Path
import shutil
import sys

ROOT = Path(__file__).resolve().parents[3]


def collect(destination):
    runs = sorted((ROOT / 'op-acceptance-tests/logs').glob('testrun-*'))
    if len(runs) != 1:
        raise ValueError('Expected exactly one acceptance test invocation')
    logs = runs[0]
    originals = list(logs.glob('raw_go_events*.log'))
    if len(originals) != 1 or not originals[0].stat().st_size:
        raise ValueError('Missing original acceptance JSON')
    destination.mkdir(parents=True, exist_ok=True)
    shutil.copytree(logs, destination / 'logs', dirs_exist_ok=True)
    results = ROOT / 'op-acceptance-tests/results'
    if not list(results.glob('*.xml')):
        raise ValueError('Missing acceptance JUnit')
    shutil.copytree(results, destination / 'junit', dirs_exist_ok=True)
    shutil.copyfile(originals[0], destination / 'original.json')
    dependencies = ROOT / 'tmp/testlogs/dependencies'
    if dependencies.exists():
        shutil.copytree(dependencies, destination / 'dependencies', dirs_exist_ok=True)
    spec = importlib.util.spec_from_file_location('report', Path(__file__).with_name('go-report.py'))
    report = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(report)
    report.project(destination / 'original.json', destination / 'native.json')
    selection = json.loads((logs / 'discovery/selection.json').read_text())
    expected = {(t['package'], t['name']) for t in selection['assigned_tests']}
    terminals = Counter()
    for event in report.events(destination / 'original.json'):
        name = event.get('Test', '')
        if name and '/' not in name and event.get('Action') in ('pass', 'skip', 'fail'):
            terminals[(event['Package'], name)] += 1
    coverage = {'expected': len(expected), 'observed': len(terminals),
                'missing': sorted(expected - terminals.keys()),
                'extra': sorted(terminals.keys() - expected),
                'duplicates': sorted(k for k, count in terminals.items() if count != 1)}
    (destination / 'coverage.json').write_text(json.dumps(coverage, indent=2) + '\n')
    if coverage['missing'] or coverage['extra'] or coverage['duplicates']:
        raise ValueError('Acceptance execution differs from assigned test identities')


if __name__ == '__main__':
    try:
        collect(Path(sys.argv[1]))
    except (ValueError, OSError) as error:
        print(f'ERROR: {error}', file=sys.stderr)
        sys.exit(1)
