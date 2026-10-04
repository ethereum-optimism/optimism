#!/usr/bin/env python3
"""Bind the full E2E release workspace, effective Cargo targets and binaries."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time
import xml.etree.ElementTree as ET

INPUTS = ('mise.toml', 'rust/Cargo.toml', 'rust/Cargo.lock', 'rust/.cargo/config.toml',
          'ops/ci/rust-e2e-release.sh', 'ops/ci/rust-e2e-release-report.py',
          'ops/ci/rust-workspace-report.py', 'ops/ci/rust-target-cache.py', 'ops/ci/go-artifacts.py')


def read(path): return json.loads(path.read_text())


def write(path, value): path.write_text(json.dumps(value, indent=2, sort_keys=True) + '\n')


def digest(path):
    with path.open('rb') as source: return hashlib.file_digest(source, 'sha256').hexdigest()


def command(*args): return subprocess.check_output(args, text=True).strip()


def binding():
    sha = command('git', 'rev-parse', 'HEAD')
    if sha != os.environ['CI_COMMIT_SHA']: raise ValueError('E2E release source revision mismatch')
    return {'source_sha': sha, 'profile': 'release', 'features': ['default'], 'scope': 'workspace',
            'input_sha256': {p: digest(Path(p)) for p in INPUTS},
            'source_trees': {p: command('git', 'rev-parse', 'HEAD:' + p) for p in ('rust', 'op-core/nuts/bundles', 'superchain-registry')},
            'rustc': command('rustc', '--version'), 'cargo': command('cargo', '--version'),
            'mold': command('mold', '--version'), 'rustflags': os.environ.get('RUSTFLAGS', ''),
            'incremental': os.environ.get('CARGO_INCREMENTAL')}


def begin(directory):
    write(directory / 'settings.json', binding() | {'suite': 'rust-e2e', 'job': 'release',
          'provider': os.environ.get('CI_RUST_PROVIDER', 'circleci'), 'workspace_root': str(Path.cwd()),
          'started_at': time.time(), 'rwx_run_id': os.environ.get('RWX_RUN_ID'),
          'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER')})


def default_features(package):
    active, pending = set(), ['default'] if 'default' in package['features'] else []
    while pending:
        feature = pending.pop()
        if feature in active: continue
        active.add(feature)
        for child in package['features'].get(feature, []):
            if '/' not in child and not child.startswith('dep:'): pending.append(child)
            elif child.startswith('dep:'): active.add(child.removeprefix('dep:'))
    return active


def artifacts(directory):
    settings = read(directory / 'settings.json')
    metadata = read(directory / 'workspace.json')
    members = metadata['workspace_members']
    if not members or len(members) != len(set(members)): raise ValueError('Empty or duplicate release workspace selection')
    packages = {p['id']: p for p in metadata['packages']}
    if not set(members) <= packages.keys(): raise ValueError('Missing release workspace package')
    expected, excluded = {}, []
    for member in members:
        p = packages[member]; enabled = default_features(p)
        for target in p['targets']:
            kinds = target['kind']
            if not any(k in kinds for k in ('bin', 'lib', 'rlib', 'proc-macro', 'cdylib', 'dylib', 'staticlib')): continue
            key = (member, target['name'], tuple(kinds))
            if not set(target.get('required-features', [])) <= enabled:
                excluded.append({'package': p['name'], 'target': target['name'], 'reason': 'default feature set does not enable required target features'})
            else: expected[key] = target
    found, binaries, complete = {}, {}, False
    root = Path(settings['workspace_root'])
    for line in (directory / 'build.json').read_text().splitlines():
        m = json.loads(line)
        if m['reason'] == 'build-finished':
            if complete or m['success'] is not True: raise ValueError('Failed or duplicate release build completion')
            complete = True
        if m['reason'] != 'compiler-artifact' or m['package_id'] not in members: continue
        target = m['target']; key = (m['package_id'], target['name'], tuple(target['kind']))
        if 'custom-build' in target['kind']: continue
        if key not in expected or key in found: raise ValueError('Duplicate or unexpected compiled release target')
        files = {}
        for filename in m['filenames']:
            path = Path(filename)
            if not path.is_relative_to(root / 'rust/target') or not path.is_file(): raise ValueError('Missing or unsafe compiled workspace artifact')
            files[str(path.relative_to(root))] = {'sha256': digest(path), 'size': path.stat().st_size}
        if not files or m['profile']['test'] is not False or m['profile']['opt_level'] != '3':
            raise ValueError('Incomplete target artifacts or unexpected release profile')
        found[key] = {'package': packages[m['package_id']]['name'], 'target': target['name'], 'kind': target['kind'],
                      'features': sorted(m['features']), 'profile': m['profile'], 'files': files, 'fresh': m['fresh']}
        if m.get('executable'):
            path = Path(m['executable'])
            if str(path.relative_to(root)) not in files or 'bin' not in target['kind']: raise ValueError('Invalid workspace release executable')
            if target['name'] in binaries: raise ValueError('Duplicate release executable name')
            binaries[target['name']] = {'path': str(path.relative_to(root)), 'sha256': digest(path)}
    if not complete or found.keys() != expected.keys(): raise ValueError('Incomplete full workspace release build coverage')
    # op-reth-proof-v1 selects proof-history settings on the same op-reth ELF.
    required = {'kona-host', 'kona-node', 'op-reth'}
    if not required <= binaries.keys(): raise ValueError('Missing E2E runtime release binaries')
    write(directory / 'coverage.json', {'packages': sorted(packages[p]['name'] for p in members),
          'targets': sorted(found.values(), key=lambda t: (t['package'], t['target'], t['kind'])),
          'excluded_feature_gated_targets': excluded, 'binaries': binaries, 'binding': binding()})
    return binaries


def finish(directory, status):
    errors = []
    if status == 0:
        try:
            settings = read(directory / 'settings.json'); expected = binding()
            if {k: settings.get(k) for k in expected} != expected: raise ValueError('Release source/settings changed during build')
            for name in ('workspace', 'build'):
                if read(directory / (name + '.stage.json'))['exit_code'] != 0: raise ValueError('Unsuccessful release stage')
            artifacts(directory)
            dependency = read(directory / 'dependency.json')
            if dependency['commit_sha'] != expected['source_sha'] or dependency['kind'] != 'rust-e2e-release':
                raise ValueError('Release runtime dependency provenance differs')
        except (ValueError, OSError, KeyError) as error: errors.append(str(error))
    suite = ET.Element('testsuite', name='rust-e2e-release')
    case = ET.SubElement(suite, 'testcase', classname='rust-e2e', name='full-workspace-release')
    if status or errors: ET.SubElement(case, 'failure', message='See original release build evidence')
    ET.ElementTree(suite).write(directory / 'checks.junit.xml', encoding='utf-8', xml_declaration=True)
    write(directory / 'final.json', {'exit_code': status, 'report_errors': errors,
          'original_sha256': {p.name: digest(p) for p in directory.iterdir() if p.is_file() and p.name != 'final.json'}})
    if errors: print('\n'.join(errors), file=sys.stderr)
    return status or bool(errors)


if __name__ == '__main__':
    mode, directory = sys.argv[1], Path(sys.argv[2])
    if mode == 'begin': begin(directory)
    elif mode == 'artifacts': artifacts(directory)
    elif mode == 'finish': sys.exit(finish(directory, int(sys.argv[3])))
    elif mode == 'binary-paths': print('\n'.join(v['path'] for _, v in sorted(read(directory / 'coverage.json')['binaries'].items())))
    else: raise ValueError('Unknown release report mode')
