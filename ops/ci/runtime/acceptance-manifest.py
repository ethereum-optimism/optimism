#!/usr/bin/env python3
"""Retain acceptance discovery and validate exhaustive test-name partitions."""
import argparse
from collections import Counter
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
PREFIX = 'github.com/ethereum-optimism/optimism/op-acceptance-tests/tests'
SPEC = importlib.util.spec_from_file_location('shards', Path(__file__).with_name('go-package-shards.py'))
SHARDS = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SHARDS)


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def settings():
    return {'count': 1, 'tags': [], 'package_parallelism': int(os.environ.get('ACCEPTANCE_TEST_JOBS', '12')),
            'parallel': int(os.environ.get('ACCEPTANCE_TEST_PARALLEL', '1')),
            'timeout': os.environ.get('ACCEPTANCE_TEST_TIMEOUT', '30m'), 'rerun_fails': 0,
            'l1_fork': os.environ.get('DEVSTACK_L1_FORK', 'fusaka'),
            'l2_cl_kind': os.environ.get('DEVSTACK_L2CL_KIND', 'op-node'),
            'l2_el_kind': os.environ.get('DEVSTACK_L2EL_KIND', 'op-reth'),
            'log_level': os.environ.get('LOG_LEVEL', 'info'),
            'stub_sp1_elf': not bool(os.environ.get('KONA_SP1_ELF_DIR'))}


def discovery(packages_path, listing_path):
    packages = SHARDS.read_go_packages(packages_path.read_text(), PREFIX)
    tests, completed = set(), set()
    for line in listing_path.read_text().splitlines():
        event = json.loads(line)
        package = event.get('Package')
        if package not in packages:
            raise ValueError('Test listing contains an unknown package')
        if event.get('Action') == 'fail':
            raise ValueError('Test listing failed')
        if event.get('Action') in ('pass', 'skip') and not event.get('Test'):
            completed.add(package)
        output = event.get('Output', '').strip()
        if re.fullmatch(r'Test\w*', output):
            identity = (package, output)
            if identity in tests:
                raise ValueError('Duplicate test identity in discovery')
            tests.add(identity)
    if completed != set(packages) or not tests:
        raise ValueError('Incomplete or empty test discovery')
    return packages, [{'package': p, 'name': n} for p, n in sorted(tests)]


def create(packages_path, listing_path, total):
    total = SHARDS.shard_number(total, total=True)
    packages, tests = discovery(packages_path, listing_path)
    # Circle splits by testname. Keep identical names from different packages
    # together, so a global -run regexp cannot accidentally duplicate a case.
    weights = Counter(t['name'] for t in tests)
    groups, loads = [[] for _ in range(total)], [0] * total
    for name in sorted(weights, key=lambda n: (-weights[n], n)):
        index = min(range(total), key=lambda i: (loads[i], len(groups[i]), i))
        groups[index].append(name)
        loads[index] += weights[name]
    sha = os.environ.get('CI_COMMIT_SHA') or os.environ.get('CIRCLE_SHA1')
    if not sha:
        sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    return {'version': 1, 'suite': 'acceptance', 'commit_sha': sha,
            'packages': packages, 'tests': tests, 'total': total, 'shards': groups,
            'settings': settings(), 'packages_sha256': sha256(packages_path),
            'listing_sha256': sha256(listing_path),
            'go_version': subprocess.check_output(['go', 'version'], cwd=ROOT, text=True).strip()}


def validate(manifest, total):
    if manifest.get('version') != 1 or manifest.get('suite') != 'acceptance':
        raise ValueError('Invalid acceptance manifest')
    total = SHARDS.shard_number(total, total=True)
    if manifest.get('total') != total:
        raise ValueError('Acceptance shard total differs')
    SHARDS.validate_packages(manifest['packages'], PREFIX)
    tests = manifest['tests']
    identities = [(t['package'], t['name']) for t in tests]
    if not identities or len(set(identities)) != len(identities):
        raise ValueError('Missing or duplicate test discovery')
    for package, name in identities:
        if package not in manifest['packages'] or not re.fullmatch(r'Test\w*', name):
            raise ValueError('Invalid test identity')
    groups = manifest['shards']
    assigned = [n for group in groups for n in group]
    if len(groups) != total or len(set(assigned)) != len(assigned) or set(assigned) != {n for _, n in identities}:
        raise ValueError('Missing, duplicate or unknown shard assignment')
    return manifest


def select(path, index, total):
    manifest = validate(json.loads(path.read_text()), total)
    index = SHARDS.shard_number(index)
    if index >= manifest['total']:
        raise ValueError('Shard index out of range')
    sha = os.environ.get('CI_COMMIT_SHA') or os.environ.get('CIRCLE_SHA1')
    if manifest.get('commit_sha') != sha or manifest.get('settings') != settings():
        raise ValueError('Acceptance revision or settings differ')
    if manifest.get('go_version') != subprocess.check_output(['go', 'version'], cwd=ROOT, text=True).strip():
        raise ValueError('Acceptance toolchain differs')
    for name, field in [('packages.json', 'packages_sha256'), ('listing.json', 'listing_sha256')]:
        if sha256(path.with_name(name)) != manifest.get(field):
            raise ValueError('Acceptance discovery changed')
    return sorted(manifest['shards'][index])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['create', 'names', 'select', 'record'])
    parser.add_argument('--directory', type=Path, required=True)
    parser.add_argument('--total', default='8')
    parser.add_argument('--index', default='0')
    parser.add_argument('--assigned', type=Path)
    args = parser.parse_args()
    path = args.directory / 'manifest.json'
    if args.command in ('create', 'names'):
        manifest = create(args.directory / 'packages.json', args.directory / 'listing.json', args.total)
        validate(manifest, args.total)
        path.write_text(json.dumps(manifest, indent=2) + '\n')
        if args.command == 'names':
            print('\n'.join(sorted({t['name'] for t in manifest['tests']})))
    else:
        manifest = validate(json.loads(path.read_text()), args.total)
        if args.command == 'select':
            print('\n'.join(select(path, args.index, args.total)), end='\n' if manifest['shards'][int(args.index)] else '')
        else:
            names = args.assigned.read_text().splitlines() if args.assigned else sorted({t['name'] for t in manifest['tests']})
            if len(names) != len(set(names)) or not set(names) <= {t['name'] for t in manifest['tests']}:
                raise ValueError('Invalid effective acceptance selection')
            record = {**manifest, 'shard_index': int(args.index), 'assigned_names': names,
                      'assigned_tests': [t for t in manifest['tests'] if t['name'] in names]}
            (args.directory / 'selection.json').write_text(json.dumps(record, indent=2) + '\n')


if __name__ == '__main__':
    try:
        main()
    except (ValueError, OSError, KeyError, subprocess.CalledProcessError) as error:
        print(f'ERROR: {error}', file=sys.stderr)
        sys.exit(1)
