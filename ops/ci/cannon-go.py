#!/usr/bin/env python3
"""Run the complete Cannon Go PR workload and seal its original evidence."""
import argparse
from collections import Counter
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
PREFIX = 'github.com/ethereum-optimism/optimism/cannon'
LIST_PATTERN = '^(Test|Example|Fuzz)'
GO_FIELDS = ('GoFiles', 'CgoFiles', 'TestGoFiles', 'XTestGoFiles', 'EmbedFiles',
             'TestEmbedFiles', 'XTestEmbedFiles', 'IgnoredGoFiles')


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


S = helper('rust-workspace-report')
C = helper('compare-ci')
PROJECT = helper('go-report')
ARTIFACTS = helper('go-artifacts')


def read(path): return C.read_json(path)


def command(*argv): return subprocess.check_output(argv, cwd=ROOT, text=True).strip()


def boolean(value):
    # Circle renders boolean environment values as 0/1; CLI flags also accept
    # their human-readable spelling. Reject every other value explicitly.
    if value in ('true', '1'): return True
    if value in ('false', '0'): return False
    raise argparse.ArgumentTypeError('Expected true/false or 1/0')


def settings(provider, slow, fresh, cpus):
    if provider not in ('circleci', 'rwx') or type(slow) is not bool or type(fresh) is not bool:
        raise ValueError('Invalid Cannon provider, slow-test or freshness setting')
    if type(cpus) is not int or cpus < 1 or (provider == 'rwx' and not fresh):
        raise ValueError('Invalid Cannon CPU count or nonfresh RWX verdict')
    return {'tags': [], 'skip_slow_tests': slow, 'timeout': '10m' if slow else '45m',
            'parallel_rule': 'nproc', 'parallel': cpus, 'package_parallelism': None,
            'count': 1 if fresh else None, 'short': False, 'rerun_fails': 0,
            'format': 'testname', 'selector': './...', 'working_directory': 'cannon'}


def inputs():
    result = {}
    for entry in command('git', 'ls-files', '--stage', '-z').split('\0'):
        if not entry: continue
        header, name = entry.split('\t', 1); mode, revision, stage = header.split()
        if stage != '0' or name in result: raise ValueError('Unmerged or duplicate Cannon source input')
        path = ROOT / name
        if mode == '160000': result[name] = {'gitlink': revision}
        elif path.is_symlink(): result[name] = {'symlink': os.readlink(path)}
        elif path.is_file(): result[name] = S.digest(path)
        else: raise ValueError('Missing tracked Cannon source input: ' + name)
    return result


def packages(text, root, source_root=None):
    root = root.resolve()
    source_root = root if source_root is None else source_root
    decoder = json.JSONDecoder(object_pairs_hook=C.no_duplicate_keys); offset, rows = 0, []
    while offset < len(text):
        if text[offset].isspace(): offset += 1; continue
        row, offset = decoder.raw_decode(text, offset)
        if not isinstance(row, dict) or row.get('Error') or row.get('DepsErrors'):
            raise ValueError('Failed complete Cannon package discovery')
        package = C.required_text(row.get('ImportPath'), 'Cannon package')
        directory = Path(row.get('Dir', '')).resolve()
        if not (package == PREFIX or package.startswith(PREFIX + '/')) or not directory.is_relative_to(root / 'cannon'):
            raise ValueError('Cannon package outside authoritative selection')
        suffix = directory.relative_to(root / 'cannon').as_posix()
        if package != PREFIX + ('' if suffix == '.' else '/' + suffix):
            raise ValueError('Cannon package identity differs from its source directory')
        files = {}
        for field in GO_FIELDS:
            names = row.get(field, [])
            if not isinstance(names, list) or len(names) != len(set(names)): raise ValueError('Duplicate Cannon package files')
            for name in names:
                relative = Path(name)
                if relative.is_absolute() or '..' in relative.parts: raise ValueError('Unsafe Cannon package source path')
                path = directory / relative
                files[str(path.relative_to(root))] = S.digest(source_root / path.relative_to(root))
        rows.append({'package': package, 'directory': str(directory.relative_to(root)),
                     'files': files, 'go_files': {f: row.get(f, []) for f in GO_FIELDS}})
    if not rows or len({r['package'] for r in rows}) != len(rows): raise ValueError('Empty or duplicate Cannon package discovery')
    return sorted(rows, key=lambda r: r['package'])


def listing(path, selected):
    source = {'feature': 'cannon-go', 'id': 'run'}; cases, failures = C.go_cases(path, source, {})
    names = {r['package']: [] for r in selected}
    if cases or failures or source['observed_packages'] != sorted(names):
        raise ValueError('Failed, incomplete or executing Cannon test listing')
    for line in path.read_text().splitlines():
        event = json.loads(line, object_pairs_hook=C.no_duplicate_keys)
        if event.get('Package') not in names: raise ValueError('Unexpected Cannon listing package')
        if event.get('Action') == 'output':
            for name in event.get('Output', '').splitlines():
                if re.fullmatch(r'(Test|Example|Fuzz)\S*', name): names[event['Package']].append(name)
    if not any(names.values()) or any(len(v) != len(set(v)) for v in names.values()):
        raise ValueError('Empty or duplicate complete Cannon test listing')
    return {p: sorted(v) for p, v in sorted(names.items())}


def go_flags(effective, *, list_tests=False):
    args = ['-timeout=' + effective['timeout'], '-parallel=' + str(effective['parallel'])]
    if effective['count'] is not None: args.append('-count=1')
    if list_tests: args += ['-list', LIST_PATTERN]
    return args + ['./...']


def test_command(directory, effective):
    return [str(ROOT / 'ops/scripts/gotestsum-split.sh'), '--format=testname',
            '--junitfile=' + str(directory / 'junit.xml'), '--jsonfile=' + str(directory / 'original.json'),
            '--', *go_flags(effective)]


def observed(directory, selected, tests):
    source = {'feature': 'cannon-go', 'id': 'run'}; cases, failures = C.go_cases(directory / 'original.json', source, {})
    if source['observed_packages'] != [r['package'] for r in selected]: raise ValueError('Cannon verdict package selection differs')
    expected = {(p, n) for p, names in tests.items() for n in names}
    actual = {(c['suite'], c['name']) for c in cases if '/' not in c['name']}
    if expected != actual: raise ValueError('Missing or extra initially selected Cannon test')
    if any(len(c['attempts']) != 1 for c in cases) or any(len(r['outcomes']) != 1 for r in source['package_attempts']):
        raise ValueError('Unexpected Cannon test or package retry')
    junit = C.junit_cases(directory / 'junit.xml', {'feature': 'cannon-go', 'id': 'run'}, {})
    keys = {(c['suite'], c['name']): c['outcome'] for c in cases}
    junit_keys = {(c['suite'], c['name']): c['outcome'] for c in junit}
    if len(junit_keys) != len(junit) or keys != junit_keys: raise ValueError('Original Cannon JUnit and JSON differ')
    # Validate every split log against the original events, including failed
    # cases. Never compare a fabricated/shortened per-test report.
    outputs = {}
    for line in (directory / 'original.json').read_text().splitlines():
        event = json.loads(line)
        if event.get('Test') and event['Action'] == 'output':
            safe = lambda value: re.sub(r'[<>:"|?*]', '_', value)
            name = 'per-test/' + safe(event['Package'].replace('/', '.')) + '/' + safe(event['Test']) + '.log'
            outputs.setdefault(name, []).append(event['Output'])
    logs = {str(p.relative_to(directory)) for p in (directory / 'per-test').rglob('*.log')}
    if logs != outputs.keys(): raise ValueError('Incomplete original Cannon per-test logs')
    for name, values in outputs.items():
        if (directory / name).read_bytes() != ''.join(values).encode(): raise ValueError('Corrupt original Cannon per-test output')
    return {'packages': source['observed_packages'], 'package_attempts': source['package_attempts'],
            'initial_tests': [{'package': p, 'name': n} for p, n in sorted(actual)],
            'cases': cases, 'outcomes': dict(Counter(c['outcome'] for c in cases)),
            'package_failures': failures, 'per_test_logs': len(logs)}


def execute(provider, slow, fresh, module_artifact=None, contract_artifact=None):
    os.chdir(ROOT); directory = ROOT / '.ci/cannon-go/run'
    shutil.rmtree(directory, ignore_errors=True); directory.mkdir(parents=True)
    status, errors, before = 1, [], None
    try:
        sha = command('git', 'rev-parse', 'HEAD')
        if not re.fullmatch('[0-9a-f]{40}', sha) or sha != (os.environ.get('CI_COMMIT_SHA') or os.environ.get('CIRCLE_SHA1') or sha):
            raise ValueError('Cannon source differs from expected revision')
        for argv in (['git', 'diff', '--exit-code', '--quiet'], ['git', 'diff', '--cached', '--exit-code', '--quiet']):
            if subprocess.call(argv): raise ValueError('Cannon source differs from its Git revision')
        branch = os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH')
        if not branch: raise ValueError('Missing Cannon tested branch')
        status = S.stage(directory, 'cpus', ['nproc'], stdout_json=True, cwd=str(ROOT / 'cannon'), stdin=subprocess.DEVNULL)
        if status: raise ValueError('Cannon CPU discovery failed')
        effective = settings(provider, slow, fresh, int((directory / 'cpus.json').read_text().strip()))
        os.environ['SKIP_SLOW_TESTS'] = str(slow).lower()
        before = inputs()
        record = {'version': 1, 'suite': 'cannon-go', 'source_sha': sha, 'branch': branch,
            'provider': provider, 'workspace_root': str(ROOT), 'settings': effective, 'input_sha256': before,
            'environment': {name: os.environ.get(name) for name in ('CI', 'SKIP_SLOW_TESTS', 'GOMAXPROCS')},
            'go_environment': json.loads(command('go', 'env', '-json', 'GOOS', 'GOARCH', 'GOAMD64', 'CGO_ENABLED',
                                                 'GOFLAGS', 'GOEXPERIMENT', 'GOTOOLCHAIN', 'GO111MODULE')),
            'tool_versions': {name: command(*argv) for name, argv in [('go', ['go', 'version']),
                ('gotestsum', ['gotestsum', '--version']), ('just', ['just', '--version']), ('forge', ['forge', '--version'])]},
            'rwx_run_id': os.environ.get('RWX_RUN_ID'), 'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}
        S.write(directory / 'settings.json', record)
        if provider == 'rwx' and (module_artifact is None or contract_artifact is None):
            raise ValueError('RWX Cannon requires verified module and ci-contract dependencies')
        for kind, artifact in (('go-modules', module_artifact), ('contracts', contract_artifact)):
            if artifact is None: continue
            ARTIFACTS.restore(kind, artifact)
            target = directory / 'dependencies'; target.mkdir(exist_ok=True)
            shutil.copy2(ROOT / 'tmp/testlogs/dependencies' / (kind + '.json'), target / (kind + '.json'))
            dependency = read(target / (kind + '.json'))
            if any(record['tool_versions'].get(name) != version for name, version in dependency['tool_versions'].items()):
                raise ValueError('Cannon dependency producer used a different toolchain')
        for name, argv, json_output in [('lint', ['just', 'lint'], False),
            ('packages', ['go', 'list', '-e', '-json', './...'], True),
            ('list', ['go', 'test', '-json', *go_flags(effective, list_tests=True)], True)]:
            status = S.stage(directory, name, argv, stdout_json=json_output, cwd=str(ROOT / 'cannon'), stdin=subprocess.DEVNULL)
            if status: raise ValueError('Original Cannon stage failed: ' + name)
            if name == 'packages': selected = packages((directory / 'packages.json').read_text(), ROOT)
        tests = listing(directory / 'list.json', selected)
        for name in sorted({n for row in selected for n in row['files']}):
            destination = directory / 'source' / name; destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(ROOT / name, destination)
        S.write(directory / 'selection.json', {'packages': selected, 'initial_tests': tests,
                                               'authority': 'go list ./... and go test -list ' + LIST_PATTERN})
        status = S.stage(directory, 'tests', test_command(directory, effective), cwd=str(ROOT / 'cannon'), stdin=subprocess.DEVNULL)
        if (directory / 'original.json').exists():
            PROJECT.project(directory / 'original.json', directory / 'native.json')
        coverage = observed(directory, selected, tests); S.write(directory / 'coverage.json', coverage)
        if not status and (coverage['package_failures'] or coverage['outcomes'].get('fail')):
            raise ValueError('Cannon command success conflicts with original failure evidence')
        after = inputs(); S.write(directory / 'inputs-after.json', after)
        if before != after or sha != command('git', 'rev-parse', 'HEAD'): raise ValueError('Cannon execution changed tracked source inputs')
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as error:
        errors.append(str(error)); print(error, file=sys.stderr); status = status or 1
    S.write(directory / 'final.json', {'exit_code': status, 'report_errors': errors,
        'original_sha256': {str(p.relative_to(directory)): S.digest(p) for p in directory.rglob('*') if p.is_file() and p.name != 'final.json'}})
    return status


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--provider', choices=('circleci', 'rwx'), required=True)
    parser.add_argument('--skip-slow-tests', type=boolean, default='true')
    parser.add_argument('--fresh-tests', type=boolean, default=os.environ.get('CI_GO_FRESH_TESTS', 'false'))
    parser.add_argument('--module-artifact', type=Path); parser.add_argument('--contract-artifact', type=Path)
    args = parser.parse_args()
    sys.exit(execute(args.provider, args.skip_slow_tests, args.fresh_tests, args.module_artifact, args.contract_artifact))
