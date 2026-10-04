#!/usr/bin/env python3
"""Retain original Rust evidence and reject missing or duplicate verdicts."""
import collections
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import shlex
import subprocess
import sys
import time
import xml.etree.ElementTree as ET


def digest(path):
    with Path(path).open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def write(path, data):
    Path(path).write_text(json.dumps(data, indent=2, sort_keys=True) + '\n')


def command(*args):
    return subprocess.check_output(args, text=True).strip()


def inputs():
    return {p: digest(p) for p in ('mise.toml', 'rust/Cargo.lock', 'rust/justfile',
                                  'rust/.config/nextest.toml', 'rust/.cargo/config.toml')}


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
        'input_sha256': inputs(), 'rustc': command('rustc', '--version'),
        'cargo': command('cargo', '--version'), 'nextest': command('cargo', 'nextest', '--version'),
        'feature_seed': sha, 'feature_partitions': 10,
        'feature_partition_index': int(os.environ.get('CI_RUST_PARTITION_INDEX', os.environ.get('CIRCLE_NODE_INDEX', '0'))),
        'test_filter': '!test(test_online)', 'incremental': os.environ.get('CARGO_INCREMENTAL'),
        'rustflags': os.environ.get('RUSTFLAGS', ''), 'rustdocflags': os.environ.get('RUSTDOCFLAGS', ''),
        'started_at': time.time(), 'cpus': os.cpu_count(),
        'superchain_revision': command('git', 'rev-parse', 'HEAD:superchain-registry')})


def stage(directory, name, args, stdout_json=False):
    """Keep the real exit and signal, including when a subprocess is canceled."""
    started = time.time()
    data = {'argv': args, 'started_at': started, 'exit_code': None}
    record = directory / (name + '.stage.json')
    write(record, data)
    with (directory / (name + '.log')).open('wb') as log:
        # JSON discovery needs stdout separate from compiler diagnostics.
        with (directory / (name + '.json')).open('wb') if stdout_json else open(os.devnull, 'wb') as out:
            child = subprocess.Popen(args, cwd='rust', start_new_session=True,
                                     stdout=out if stdout_json else subprocess.PIPE,
                                     stderr=log if stdout_json else subprocess.STDOUT)
            previous = {}
            def cancel(signum, _frame):
                os.killpg(child.pid, signum)
            for signum in (signal.SIGINT, signal.SIGTERM):
                previous[signum] = signal.signal(signum, cancel)
            try:
                if not stdout_json:
                    for line in iter(child.stdout.readline, b''):
                        log.write(line)
                        log.flush()
                        sys.stdout.buffer.write(line)
                        sys.stdout.buffer.flush()
                status = child.wait()
            finally:
                for signum, handler in previous.items():
                    signal.signal(signum, handler)
    data.update(exit_code=status, elapsed_seconds=time.time() - started,
                log_sha256=digest(directory / (name + '.log')))
    write(record, data)
    if status < 0:
        status = 128 - status
    return status


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
    stages = {p.stem.removesuffix('.stage'): json.loads(p.read_text()) for p in directory.glob('*.stage.json')}
    required = {
        'tests-build': ['archive', 'beacon-build'],
        'tests': ['unit-list', 'unit', 'beacon-list', 'beacon', 'doctests-list', 'doctests'],
        'doctest': ['doctests-list', 'doctests'], 'docs': ['docs'], 'clippy': ['clippy'],
        'build': ['build'], 'features': ['features-list', 'features', 'feature-tests-list', 'feature-tests'],
        'feature-plan': ['features-list', 'feature-tests-list'], 'no-std': ['no-std'], 'udeps': ['udeps']}
    if status == 0:
        for name in ['workspace'] + required[job]:
            if stages.get(name, {}).get('exit_code') != 0:
                errors.append(f'Missing or unsuccessful stage: {name}')
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
    elif mode in ('artifact', 'verify-artifact'):
        artifact(directory, mode == 'verify-artifact')
    elif mode == 'finish':
        sys.exit(finish(directory, int(sys.argv[3])))
    else:
        raise ValueError('Unknown Rust report mode')
