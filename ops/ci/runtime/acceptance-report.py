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


def report_id(index, package, name):
    # The recipe names a test's JSON and JUnit after "<shard>-<package, / as .>.<test>".
    return f"{index}-{package.replace('/', '.')}.{name}"


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
    # Like the recipe, treat an empty assignment as a failed split.
    if not expected:
        raise ValueError('Empty acceptance shard assignment')
    ids = {report_id(index, *pair) for pair in expected}
    originals = {path.name.removeprefix('raw_go_events-').removesuffix('.log'): path for path in logs.glob('raw_go_events-*.log')}
    junits = sorted((ROOT / 'op-acceptance-tests/results').glob('results-*.xml'))
    (destination / 'junit').mkdir(exist_ok=True)
    for junit in junits:
        shutil.copyfile(junit, destination / 'junit' / junit.name)
    # The recipe runs one go test per test. Join their events in report order; an
    # empty stream adds no events, so its test is reported missing below.
    with (destination / 'original.json').open('wb') as joined:
        for key in sorted(originals):
            data = originals[key].read_bytes().strip(b'\n')
            if data: joined.write(data + b'\n')
    PROJECT.project(destination / 'original.json', destination / 'native.json')
    terminals = Counter()
    for event in PROJECT.events(destination / 'original.json'):
        name = event.get('Test', '')
        if name and '/' not in name and event.get('Action') in ('pass', 'skip', 'fail'):
            terminals[(event['Package'], name)] += 1
    coverage = {'expected': len(expected), 'observed': len(terminals),
                'missing': sorted(expected - terminals.keys()),
                'extra': sorted(terminals.keys() - expected),
                'duplicates': sorted(k for k, count in terminals.items() if count != 1),
                'missing_json': sorted(ids - originals.keys()), 'extra_json': sorted(originals.keys() - ids),
                'missing_junit': sorted(ids - {j.name.removeprefix('results-').removesuffix('.xml') for j in junits})}
    (destination / 'coverage.json').write_text(json.dumps(coverage, indent=2) + '\n')
    if any(coverage[key] for key in ('missing', 'extra', 'duplicates', 'missing_json', 'extra_json', 'missing_junit')):
        raise ValueError('Acceptance execution differs from assigned test identities')


if __name__ == '__main__':
    try:
        collect(Path(sys.argv[1]), Path(os.environ['CI_ACCEPTANCE_DISCOVERY']),
                os.environ['CI_SHARD_INDEX'], os.environ['CI_SHARD_TOTAL'])
    except (ValueError, OSError, KeyError, subprocess.CalledProcessError) as error:
        print(f'ERROR: {error}', file=sys.stderr)
        sys.exit(1)
