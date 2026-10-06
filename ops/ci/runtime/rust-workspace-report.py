#!/usr/bin/env python3
"""Retain original Rust evidence and reject missing or duplicate verdicts."""
import collections
import importlib.util
import json
import os
from pathlib import Path
import re
import shlex
import sys
import time
import xml.etree.ElementTree as ET


SPEC = importlib.util.spec_from_file_location('ci_report', Path(__file__).with_name('ci-report.py'))
REPORT = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(REPORT)

digest = REPORT.digest
write = REPORT.write


def command(*args):
    return REPORT.command(args)


def inputs(job=None):
    paths = ['mise.toml', 'rust/Cargo.lock', 'rust/justfile',
             'rust/.config/nextest.toml', 'rust/.cargo/config.toml', 'ops/ci/runtime/ci-report.py']
    if job in ('wasm-unknown', 'wasm-wasi', 'zepter', 'typos', 'registry', 'interop'):
        # Bind every workspace manifest as well as the shared commands. Source
        # identity is the Git SHA; these hashes make settings drift explicit.
        paths += ['rust/Cargo.toml', 'ops/ci/runtime/rust-workspace.sh', 'ops/ci/runtime/rust-workspace-report.py']
        paths += sorted(command('git', 'ls-files', 'rust/**/Cargo.toml').splitlines())
    if job == 'zepter':
        paths += ['rust/.config/zepter.yaml']
    if job == 'typos':
        paths += sorted(command('git', 'ls-files', '*typos*').splitlines())
    if job == 'registry':
        paths += ['rust/kona/crates/protocol/registry/build.rs']
    if job == 'interop':
        paths += ['go.mod', 'go.sum', 'ops/scripts/test-interop-deposits-diff.sh',
                  'op-node/cmd/interop-deposits-dump/main.go',
                  'rust/kona/crates/protocol/hardforks/examples/interop-deposits-dump.rs']
    return {p: digest(p) for p in paths}


def begin(directory, job):
    sha = command('git', 'rev-parse', 'HEAD')
    expected = os.environ.get('CI_COMMIT_SHA') or os.environ.get('CIRCLE_SHA1') or sha
    if not re.fullmatch('[0-9a-f]{40}', expected) or sha != expected:
        raise ValueError('Rust source revision differs from expected SHA')
    write(directory / 'settings.json', {
        'suite': 'rust-workspace', 'job': job, 'source_sha': sha,
        'workspace_root': str(Path.cwd()),
        'cargo_home': os.environ.get('CARGO_HOME', str(Path.home() / '.cargo')),
        'provider': os.environ.get('CI_RUST_PROVIDER', 'circleci'),
        'rwx_run_id': os.environ.get('RWX_RUN_ID'),
        'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER'),
        'input_sha256': inputs(job), 'rustc': command('rustc', '--version'),
        'cargo': command('cargo', '--version'), 'nextest': command('cargo', 'nextest', '--version'),
        'feature_seed': sha, 'feature_partitions':
            int(os.environ.get('CI_RUST_PARTITION_TOTAL', os.environ.get('CIRCLE_NODE_TOTAL', '10')))
            if job == 'features' else 10,
        'feature_partition_index': int(os.environ.get('CI_RUST_PARTITION_INDEX', os.environ.get('CIRCLE_NODE_INDEX', '0'))),
        'test_filter': '!test(test_online)', 'incremental': os.environ.get('CARGO_INCREMENTAL'),
        'rustflags': os.environ.get('RUSTFLAGS', ''), 'rustdocflags': os.environ.get('RUSTDOCFLAGS', ''),
        'started_at': time.time(), 'cpus': os.cpu_count(),
        'superchain_revision': command('git', 'rev-parse', 'HEAD:superchain-registry')})
    tools = {'zepter': ('zepter', '--version'), 'typos': ('typos', '--version'),
             'wasm-unknown': ('cargo', 'hack', '--version'),
             'wasm-wasi': ('cargo', 'hack', '--version'), 'interop': ('go', 'version')}
    if job in tools:
        settings = json.loads((directory / 'settings.json').read_text())
        settings['extra_tool'] = command(*tools[job])
        write(directory / 'settings.json', settings)


def stage(directory, name, args, stdout_json=False, cwd='rust', stdin=None, stdout_file=None):
    return REPORT.stage(directory, name, args, cwd=cwd, stdout_json=stdout_json,
                        stdin=stdin, stdout_file=stdout_file)


def unit_report(directory):
    discovery = json.loads((directory / 'unit-list.json').read_text())
    expected, excluded = set(), {}
    for suite, body in discovery['rust-suites'].items():
        for name, case in body['testcases'].items():
            key = (suite, name)
            match = case['filter-match']
            if match == {'status': 'matches'} and case['ignored'] is False:
                expected.add(key)
            elif match == {'status': 'mismatch', 'reason': 'ignored'} and case['ignored'] is True:
                excluded[key] = 'ignored'
            elif match == {'status': 'mismatch', 'reason': 'expression'} and 'test_online' in name:
                excluded[key] = 'online test excluded by the shared Just filter'
            else:
                raise ValueError(f'Unrecognized nextest selection: {key}: {case}')
    if not expected:
        raise ValueError('Empty runnable nextest discovery')
    actual, retries = {}, {}
    tree = ET.parse(directory / 'junit.xml')
    for case in tree.iter('testcase'):
        key = (case.get('classname', ''), case.attrib['name'])
        if key in actual:
            raise ValueError(f'Duplicate nextest verdict: {key}')
        actual[key] = ('skip' if case.find('skipped') is not None else
                       'fail' if case.find('failure') is not None or case.find('error') is not None else 'pass')
        retries[key] = len([n for n in case if n.tag in ('flakyFailure', 'flakyError', 'rerunFailure', 'rerunError')])
    # Pinned nextest omits explicitly ignored/filtered cases from JUnit.
    missing, extra = expected - actual.keys(), actual.keys() - expected - excluded.keys()
    unexpected_skips = [key for key, result in actual.items() if (result == 'skip') != (key in excluded)]
    report = {'selected': len(expected), 'excluded': [dict(suite=k[0], name=k[1], reason=v)
                 for k, v in sorted(excluded.items())], 'missing': sorted(missing), 'extra': sorted(extra),
              'unexpected_skips': sorted(unexpected_skips), 'outcomes': dict(collections.Counter(actual.values())),
              'cases': [dict(suite=k[0], name=k[1], outcome=v, retries=retries[k]) for k, v in sorted(actual.items())]}
    write(directory / 'unit-coverage.json', report)
    if missing or extra or unexpected_skips:
        raise ValueError('Incomplete or mismatched nextest coverage; see unit-coverage.json')
    return report


def libtest_report(directory, phase):
    listing = (directory / (phase + '-list.log')).read_text()
    log = (directory / (phase + '.log')).read_text()
    settings_path = directory / 'settings.json'
    settings = json.loads(settings_path.read_text()) if settings_path.exists() else {}
    def identity(name):
        # Rustdoc decorates no_run verdicts but not their --list identities.
        name = re.sub(r'(\(line \d+\)) - compile$', r'\1', name)
        home = settings.get('cargo_home')
        if home and name.startswith(home + '/'):
            name = '<cargo>/' + name[len(home) + 1:]
        return name
    expected = []
    for line in listing.splitlines():
        match = re.fullmatch(r'(.+): test', line)
        if match:
            expected.append(identity(match[1]))
    if not expected or len(expected) != len(set(expected)):
        raise ValueError(f'Empty or duplicate {phase} discovery')
    actual, compile_only = {}, set()
    for line in log.splitlines():
        match = re.fullmatch(r'test (.+) \.\.\. (ok|FAILED|ignored.*)', line)
        if match:
            original, result = match.groups()
            name = identity(original)
            if re.search(r'\(line \d+\) - compile$', original):
                compile_only.add(name)
            if name in actual:
                raise ValueError(f'Duplicate {phase} verdict: {name}')
            actual[name] = 'pass' if result == 'ok' else 'fail' if result == 'FAILED' else 'skip'
    missing, extra = set(expected) - actual.keys(), actual.keys() - set(expected)
    report = {'selected': len(expected), 'missing': sorted(missing), 'extra': sorted(extra),
              'cases': [dict(suite=phase, name=k, outcome=v, retries=0, compile_only=k in compile_only)
                        for k, v in sorted(actual.items())],
              'outcomes': dict(collections.Counter(actual.values()))}
    write(directory / (phase + '-coverage.json'), report)
    suite = ET.Element('testsuite', name=phase, tests=str(len(actual)))
    for name, result in sorted(actual.items()):
        case = ET.SubElement(suite, 'testcase', classname=phase, name=name)
        if result != 'pass':
            ET.SubElement(case, 'skipped' if result == 'skip' else 'failure', message=result)
    ET.ElementTree(suite).write(directory / (phase + '.junit.xml'), encoding='utf-8', xml_declaration=True)
    if missing or extra:
        raise ValueError(f'Incomplete {phase} coverage')
    return report


def commands(text):
    # Authoritative unpartitioned cargo-hack dry-run output, including manifests.
    return [m.group(1) for m in re.finditer(r'(?:^|running `)(cargo (?:check|build) [^\n`]+)', text, re.M)]


def feature_report(directory):
    report = {}
    settings = json.loads((directory / 'settings.json').read_text())
    workspace = json.loads((directory / 'workspace.json').read_text())
    root = Path(settings['workspace_root']) / 'rust'
    packages = {str(Path(p['manifest_path']).relative_to(root)): p['name'] for p in workspace['packages']}
    index, total = settings['feature_partition_index'], settings['feature_partitions']
    for phase in ('features', 'feature-tests'):
        plan = []
        for line in commands((directory / (phase + '-list.log')).read_text()):
            args = shlex.split(line)
            location = args.index('--manifest-path')
            package = packages[args[location + 1]]
            del args[location:location + 2]
            plan.append({'crate': package, 'argv': args})
        identities = [json.dumps(c, sort_keys=True) for c in plan]
        if not plan or len(identities) != len(set(identities)):
            raise ValueError(f'Empty or duplicate {phase} commands')
        width = (len(plan) + total - 1) // total
        expected = set(range(index * width + 1, min((index + 1) * width, len(plan)) + 1))
        actual, seen, errors = [], set(), []
        pattern = r'info: (running|skipping) `(cargo (?:check|build)(?: [^`]+)?)` on ([^\s]+) \((\d+)/(\d+)\)'
        for match in re.finditer(pattern, (directory / (phase + '.log')).read_text()):
            action, cargo, package, number, count = match.groups()
            number = int(number)
            identity = {'crate': package, 'argv': shlex.split(cargo)}
            if number in seen or int(count) != len(plan) or not 1 <= number <= len(plan):
                errors.append(f'Duplicate or invalid command index: {number}/{count}')
                continue
            seen.add(number)
            if identity != plan[number - 1] or (action == 'running') != (number in expected):
                errors.append(f'Wrong command or partition ownership at {number}')
            if action == 'running':
                actual.append({'index': number, **identity})
        missing = sorted(set(range(1, len(plan) + 1)) - seen)
        report[phase] = {'command_count': len(plan), 'partition_index': index, 'partitions': total,
                         'planned': [{'index': n, **plan[n - 1]} for n in sorted(expected)],
                         'executed': actual, 'missing': missing, 'errors': errors}
    write(directory / 'feature-coverage.json', report)
    if any(v['missing'] or v['errors'] for v in report.values()):
        raise ValueError('Incomplete feature command execution')


def no_std_report(directory):
    source = Path('rust/justfile').read_text()
    array = re.search(r'no_std_packages=\((.*?)\n\s*\)', source, re.S)
    if not array:
        raise ValueError('Missing authoritative no_std selection')
    selected = shlex.split(array.group(1), comments=True)
    log = (directory / 'no-std.log').read_text()
    attempted = re.findall(r'^Checking no_std build for: (.+)$', log, re.M)
    completed = re.findall(r'^Successfully checked no_std build for: (.+)$', log, re.M)
    write(directory / 'no-std-coverage.json', {'selected': selected, 'attempted': attempted, 'completed': completed})
    if not selected or len(selected) != len(set(selected)) or attempted != selected or completed != selected:
        raise ValueError('Incomplete no_std package coverage')


WASM = {
    'wasm-unknown': ('wasm32-unknown-unknown',
                     ['op-alloy-consensus', 'op-alloy-rpc-types', 'op-alloy-rpc-types-engine', 'alloy-op-evm'], True),
    'wasm-wasi': ('wasm32-wasip1', ['op-alloy-consensus', 'op-alloy-rpc-types-engine', 'alloy-op-evm'], False),
}


def wasm_report(directory, job):
    target, selected, no_defaults = WASM[job]
    settings = json.loads((directory / 'settings.json').read_text())
    workspace = json.loads((directory / 'workspace.json').read_text())
    root = Path(settings['workspace_root']) / 'rust'
    packages = {str(Path(p['manifest_path']).relative_to(root)): p['name'] for p in workspace['packages']}
    plan = []
    for line in commands((directory / 'wasm-list.log').read_text()):
        args = shlex.split(line)
        location = args.index('--manifest-path')
        package = packages[args[location + 1]]
        del args[location:location + 2]
        if args[args.index('--target') + 1] != target or ('--no-default-features' in args) != no_defaults:
            raise ValueError('WASM discovery target or default features mismatch')
        plan.append({'crate': package, 'argv': args})
    if sorted(p['crate'] for p in plan) != sorted(selected):
        raise ValueError('Missing, extra or duplicate WASM package selection')
    actual = []
    pattern = r'info: running `(cargo build(?: [^`]+)?)` on ([^\s]+) \((\d+)/(\d+)\)'
    for match in re.finditer(pattern, (directory / 'wasm.log').read_text()):
        cargo, package, index, count = match.groups()
        if int(index) != len(actual) + 1 or int(count) != len(plan):
            raise ValueError('Duplicate or invalid WASM command index')
        actual.append({'crate': package, 'argv': shlex.split(cargo)})
    write(directory / 'wasm-coverage.json', {'target': target, 'no_default_features': no_defaults,
                                           'selected': selected, 'planned': plan, 'executed': actual})
    if actual != plan:
        raise ValueError('Incomplete or mismatched WASM command execution')
    artifacts = {}
    for package in selected:
        path = Path(workspace['target_directory']) / target / 'debug' / ('lib' + package.replace('-', '_') + '.rlib')
        with path.open('rb') as source:
            if source.read(8) != b'!<arch>\n':
                raise ValueError(f'Corrupt WASM library archive: {package}')
        artifacts[path.name] = {'sha256': digest(path), 'size': path.stat().st_size}
    write(directory / 'wasm-artifacts.json', {'source_sha': settings['source_sha'], 'target': target,
                                             'rustc': settings['rustc'], 'artifacts': artifacts})
    return selected


REGISTRY = 'rust/kona/crates/protocol/registry/etc'


def registry_snapshot(directory, phase):
    for name in ('chainList.json', 'configs.json', 'depsets.json'):
        path = Path(REGISTRY) / name
        data = json.loads(path.read_text())
        if not isinstance(data, (list, dict)) or not data:
            raise ValueError(f'Empty or invalid registry snapshot: {name}')
        (directory / f'registry-{phase}-{name}').write_bytes(path.read_bytes())


def registry_report(directory):
    pairs = {}
    for name in ('chainList.json', 'configs.json', 'depsets.json'):
        pairs[name] = {phase: digest(directory / f'registry-{phase}-{name}') for phase in ('before', 'after')}
    write(directory / 'registry-coverage.json', {'snapshots': pairs, 'sync_superchain': True,
                                               'fresh_build_script': True})
    if any(p['before'] != p['after'] for p in pairs.values()):
        raise ValueError('Regenerated registry snapshots differ from committed inputs')


def interop_report(directory):
    for language in ('go', 'rust'):
        if (directory / f'interop-{language}.exit').read_text().strip() != '0':
            raise ValueError(f'Interop {language} dumper did not succeed')
        # Retain stderr even when it contains no diagnostics.
        digest(directory / f'interop-{language}.stderr')
    go = (directory / 'interop-go.stdout').read_bytes()
    rust = (directory / 'interop-rust.stdout').read_bytes()
    if not go or go != rust:
        raise ValueError('Interop dump outputs are empty or differ')
    cases, variants = [], []
    current, index, fields = None, None, []
    required = ['source_hash', 'from', 'to', 'mint', 'value', 'gas_limit', 'is_system_tx', 'data']
    def complete_tx():
        if index is not None and fields != required:
            raise ValueError('Missing or extra Interop deposit fields')
    for line in go.decode().splitlines():
        if line.startswith('activate='):
            complete_tx()
            current = line.removeprefix('activate=')
            variants.append(current)
            index, fields = None, []
        elif line.startswith('gas='):
            if not re.fullmatch(r'gas=0x[0-9a-f]{16}', line) or index is not None:
                raise ValueError('Invalid Interop activation gas')
            cases.append(f'activate={current}/gas')
        elif re.fullmatch(r'--- tx \d+ ---', line):
            complete_tx()
            next_index = int(line.split()[2])
            if next_index != (0 if index is None else index + 1):
                raise ValueError('Missing or duplicate Interop deposit')
            index, fields = next_index, []
            cases.append(f'activate={current}/tx={index}')
        elif '=' in line and index is not None:
            fields.append(line.split('=', 1)[0])
        else:
            raise ValueError('Unrecognized Interop dump record')
    complete_tx()
    if variants != ['false', 'true'] or any(f'activate={v}/gas' not in cases or
        f'activate={v}/tx=0' not in cases for v in variants) or len(cases) != len(set(cases)):
        raise ValueError('Incomplete Interop activation selection')
    write(directory / 'interop-coverage.json', {'cases': cases, 'byte_identical': True,
                                               'stdout_sha256': digest(directory / 'interop-go.stdout')})
    return cases


def check_junit(directory, job, cases, passed):
    suite = ET.Element('testsuite', name=job, tests=str(len(cases)))
    for name in cases:
        case = ET.SubElement(suite, 'testcase', classname=job, name=name)
        if not passed:
            ET.SubElement(case, 'failure', message='See original stage logs and final.json')
    ET.ElementTree(suite).write(directory / 'checks.junit.xml', encoding='utf-8', xml_declaration=True)


def artifact(directory, verify=False):
    path = directory / 'tests.tar.zst'
    expected = {'source_sha': command('git', 'rev-parse', 'HEAD'), 'input_sha256': inputs(),
                'rustc': command('rustc', '--version'), 'nextest': command('cargo', 'nextest', '--version'),
                'archive_sha256': digest(path), 'workspace': True, 'all_features': True}
    if verify:
        if json.loads((directory / 'archive.json').read_text()) != expected:
            raise ValueError('Test archive revision, settings, toolchain or checksum mismatch')
    else:
        write(directory / 'archive.json', expected)


def finish(directory, status):
    errors = []
    job = json.loads((directory / 'settings.json').read_text())['job']
    reports = []
    if job in ('tests', 'doctest'):
        checks = [('doctests', lambda: libtest_report(directory, 'doctests'))]
        if job == 'tests':
            checks = [('unit', lambda: unit_report(directory)),
                      ('beacon', lambda: libtest_report(directory, 'beacon'))] + checks
        for name, check in checks:
            try:
                reports.append(check())
            except (ValueError, KeyError, OSError, ET.ParseError) as error:
                errors.append(f'{name}: {error}')
    if job == 'features':
        try:
            feature_report(directory)
        except (ValueError, OSError) as error:
            errors.append(str(error))
    if job == 'no-std':
        try:
            no_std_report(directory)
        except (ValueError, OSError) as error:
            errors.append(str(error))
    cases = [job]
    extra_jobs = (*WASM, 'zepter', 'typos', 'registry', 'interop')
    if job in extra_jobs:
        try:
            if job in WASM:
                cases = wasm_report(directory, job)
            elif job == 'registry':
                registry_report(directory)
            elif job == 'interop':
                cases = interop_report(directory)
        except (ValueError, KeyError, OSError) as error:
            errors.append(str(error))
    stages = {p.stem.removesuffix('.stage'): json.loads(p.read_text()) for p in directory.glob('*.stage.json')}
    required = {
        'tests-build': ['archive', 'beacon-build'],
        'tests': ['unit-list', 'unit', 'beacon-list', 'beacon', 'doctests-list', 'doctests'],
        'doctest': ['doctests-list', 'doctests'], 'docs': ['docs'], 'clippy': ['clippy'],
        'build': ['build'], 'features': ['features-list', 'features', 'feature-tests-list', 'feature-tests'],
        'feature-plan': ['features-list', 'feature-tests-list'], 'no-std': ['no-std'], 'udeps': ['udeps'],
        'wasm-unknown': ['wasm-target', 'wasm-list', 'wasm'], 'wasm-wasi': ['wasm-target', 'wasm-list', 'wasm'],
        'zepter': ['zepter'], 'typos': ['typos'],
        'registry': ['registry-clean', 'registry', 'registry-diff'],
        'interop': ['superchain-go', 'interop']}
    if status == 0:
        for name in ['workspace'] + required[job]:
            if stages.get(name, {}).get('exit_code') != 0:
                errors.append(f'Missing or unsuccessful stage: {name}')
    if job in extra_jobs:
        check_junit(directory, job, cases, status == 0 and not errors)
    files = {p.name: digest(p) for p in directory.iterdir() if p.is_file() and p.name != 'final.json'}
    write(directory / 'final.json', {'exit_code': status, 'report_errors': errors, 'stages': stages,
                                     'original_sha256': files, 'reports': reports})
    if errors:
        print('\n'.join(errors), file=sys.stderr)
    return status or bool(errors)


if __name__ == '__main__':
    mode, name = sys.argv[1:3]
    directory = Path(name).resolve()
    if mode == 'begin':
        begin(directory, sys.argv[3])
    elif mode in ('stage', 'json-stage'):
        sys.exit(stage(directory, sys.argv[3], sys.argv[4:], mode == 'json-stage'))
    elif mode == 'stage-at':
        sys.exit(stage(directory, sys.argv[4], sys.argv[5:], cwd=sys.argv[3]))
    elif mode == 'registry-snapshot':
        registry_snapshot(directory, sys.argv[3])
    elif mode in ('artifact', 'verify-artifact'):
        artifact(directory, mode == 'verify-artifact')
    elif mode == 'finish':
        sys.exit(finish(directory, int(sys.argv[3])))
    else:
        raise ValueError('Unknown Rust report mode')
