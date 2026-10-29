#!/usr/bin/env python3
"""Collect the original acceptance reports even when the shared runner fails."""
from collections import Counter
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


MANIFEST = helper('acceptance-manifest')
PROJECT = helper('go-report')


def collect(destination, discovery, index, total):
    runs = sorted((ROOT / 'op-acceptance-tests/logs').glob('testrun-*'))
    if len(runs) != 1:
        raise ValueError('Expected exactly one acceptance test invocation')
    logs = runs[0]
    destination.mkdir(parents=True, exist_ok=True)
    shutil.copytree(logs, destination / 'logs', dirs_exist_ok=True)
    dependencies = ROOT / 'tmp/testlogs/dependencies'
    if dependencies.exists():
        shutil.copytree(dependencies, destination / 'dependencies', dirs_exist_ok=True)
    names = set(MANIFEST.select(discovery / 'manifest.json', index, total))
    tests = json.loads((discovery / 'manifest.json').read_text())['tests']
    expected = {(t['package'], t['name']) for t in tests if t['name'] in names}
    terminals = Counter()
    # The recipe runs nothing for an empty assignment. Any other shard must report.
    if expected:
        original = logs / f'raw_go_events-{index}.log'
        if not original.is_file() or not original.stat().st_size:
            raise ValueError('Missing original acceptance JSON')
        shutil.copyfile(original, destination / 'original.json')
        PROJECT.project(destination / 'original.json', destination / 'native.json')
        junit = ROOT / f'op-acceptance-tests/results/results-{index}.xml'
        if not junit.is_file() or not junit.stat().st_size:
            raise ValueError('Missing acceptance JUnit')
        (destination / 'junit').mkdir(exist_ok=True)
        shutil.copyfile(junit, destination / 'junit' / junit.name)
        for event in PROJECT.events(destination / 'original.json'):
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
        collect(Path(sys.argv[1]), Path(os.environ['CI_ACCEPTANCE_DISCOVERY']),
                os.environ['CI_SHARD_INDEX'], os.environ['CI_SHARD_TOTAL'])
    except (ValueError, OSError, KeyError, subprocess.CalledProcessError) as error:
        print(f'ERROR: {error}', file=sys.stderr)
        sys.exit(1)
