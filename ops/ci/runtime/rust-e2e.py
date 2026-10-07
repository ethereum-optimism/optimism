#!/usr/bin/env python3
"""Compile, discover and freshly run the complete Rust E2E Go workloads."""
import argparse
from collections import Counter
import importlib.util
import json
import math
import os
from pathlib import Path
import re
import shutil
import signal
import statistics
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[3]
PREFIX = 'github.com/ethereum-optimism/optimism/'


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


GO = helper('go-compiled-tests')
STAGE = helper('rust-workspace-report')
PROJECT = helper('go-report')
REPORT = helper('ci-report')
JOBS = {
    'proof': {'circle_job': 'kona-proof-action-single', 'package': 'rust/kona/tests/proofs', 'shards': 8,
              'timeout': '60m', 'per_test': True, 'recipe': ['just', 'action-tests-single-run']},
    'restart': {'circle_job': 'rust-e2e-restart', 'package': 'rust/kona/tests/node/restart', 'shards': 3,
                'timeout': '40m', 'per_test': False, 'recipe': ['just', 'test-e2e-sysgo-run', 'node', 'node/restart']},
    'simple-kona': {'circle_job': 'rust-e2e-simple-kona', 'package': 'rust/kona/tests/node/common', 'shards': 1,
                    'timeout': '40m', 'per_test': False, 'recipe': ['just', 'test-e2e-sysgo-run', 'node', 'node/common', 'simple-kona']},
    'simple-kona-sequencer': {'circle_job': 'rust-e2e-simple-kona-sequencer', 'package': 'rust/kona/tests/node/common', 'shards': 1,
                    'timeout': '40m', 'per_test': False, 'recipe': ['just', 'test-e2e-sysgo-run', 'node', 'node/common', 'simple-kona-sequencer']},
    'op-reth': {'circle_job': 'op-reth-e2e-sysgo-tests', 'package': 'rust/op-reth/tests/proofs/core', 'shards': 1,
                'timeout': '40m', 'per_test': False, 'recipe': ['just', 'test-e2e-sysgo']},
}
INPUTS = ('go.mod', 'go.sum', 'mise.toml', 'rust/kona/tests/justfile', 'rust/op-reth/tests/justfile',
          'ops/ci/runtime/rust-e2e.py', 'ops/ci/runtime/rust-e2e.sh', 'ops/ci/runtime/rust-e2e-timings.json',
          'ops/ci/runtime/go-compiled-tests.py', 'ops/ci/runtime/go-report.py', 'ops/ci/runtime/ci-report.py')


read = REPORT.read


write = REPORT.write


def settings(job):
    return {'suite': 'rust-e2e', 'job': job, **JOBS[job], 'count': 1, 'tags': [], 'rerun_fails': 0}


def environment(job):
    values = {'DISABLE_OP_E2E_LEGACY': 'true', 'DEVSTACK_ORCHESTRATOR': 'sysgo'} if job != 'proof' else {}
    if job == 'proof': values['KONA_HOST_PATH'] = str(ROOT / 'rust/target/release/kona-host')
    elif job == 'op-reth':
        values.update(OP_RETH_ENABLE_PROOF_HISTORY='true', OP_DEVSTACK_PROOF_SEQUENCER_EL='op-reth',
                      OP_DEVSTACK_PROOF_VALIDATOR_EL='op-reth',
                      RUST_BINARY_PATH_OP_RETH=str(ROOT / 'rust/target/release/op-reth'))
    else:
        sequencer = job == 'simple-kona-sequencer'
        values.update(KONA_SEQUENCER_WITH_RETH=str(int(sequencer)), KONA_VALIDATOR_WITH_RETH='1',
                      OP_SEQUENCER_WITH_RETH=str(int(not sequencer)), OP_VALIDATOR_WITH_RETH=str(int(not sequencer)),
                      RUST_BINARY_PATH_OP_RETH=str(ROOT / 'rust/target/release/op-reth'),
                      RUST_BINARY_PATH_KONA_NODE=str(ROOT / 'rust/target/release/kona-node'))
    return values


def binding(job):
    sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    if sha != os.environ['CI_COMMIT_SHA']: raise ValueError('Rust E2E source revision differs')
    go_env = json.loads(subprocess.check_output(['go', 'env', '-json', 'GOOS', 'GOARCH', 'CGO_ENABLED',
                       'GOFLAGS', 'GOEXPERIMENT', 'GOTOOLCHAIN'], cwd=ROOT, text=True))
    return {'source_sha': sha, 'settings': settings(job), 'workspace_root': str(ROOT),
            'input_sha256': {name: GO.digest(ROOT / name) for name in INPUTS},
            'go_version': subprocess.check_output(['go', 'version'], text=True).strip(), 'go_environment': go_env}


def partition(names, total, durations=None):
    if type(total) is not int or not 1 <= total <= 256: raise ValueError('Invalid E2E shard total')
    if not names or sorted(set(names)) != names or any(not re.fullmatch(r'(Test|Example|Fuzz)\w*', n) for n in names):
        raise ValueError('Empty, duplicate or invalid E2E test discovery')
    durations = {} if durations is None else durations
    if not isinstance(durations, dict) or any(type(v) not in (float, int) or not math.isfinite(v) or v < 0 for v in durations.values()):
        raise ValueError('Invalid E2E test durations')
    known = [durations[n] for n in names if n in durations]
    fallback = max(.001, statistics.median(known)) if known else 1.
    weights = {n: durations.get(n, fallback) for n in names}
    groups, loads = [[] for _ in range(total)], [0.] * total
    for name in sorted(names, key=lambda n: (-weights[n], n)):
        index = min(range(total), key=lambda i: (loads[i], len(groups[i]), i))
        groups[index].append(name); loads[index] += weights[name]
    return [sorted(group) for group in groups]


def discovery(job, directory):
    package = read(directory / 'packages.json')
    expected = PREFIX + JOBS[job]['package']
    if (package.get('ImportPath') != expected or package.get('Error') or package.get('DepsErrors')
            or not (package.get('TestGoFiles') or package.get('XTestGoFiles'))):
        raise ValueError('Invalid E2E package discovery')
    names = [line for line in (directory / 'listing.log').read_text().splitlines()
             if re.fullmatch(r'(Test|Example|Fuzz)\w*', line)]
    if len(set(names)) != len(names): raise ValueError('Duplicate E2E listed test')
    # Original Circle sharding keeps ^Test names only. Unsharded packages run
    # examples and fuzz seeds too; record that distinction instead of hiding it.
    excluded = [n for n in names if JOBS[job]['shards'] > 1 and not n.startswith('Test')]
    selected = sorted(set(names) - set(excluded))
    for name in ('packages', 'listing'):
        if read(directory / (name + '.stage.json'))['exit_code'] != 0: raise ValueError('E2E discovery failed')
    partition(selected, JOBS[job]['shards'])
    return selected, excluded


def compile_suite(job):
    directory = ROOT / '.ci/rust-e2e/compiled' / job
    shutil.rmtree(directory, ignore_errors=True); directory.mkdir(parents=True)
    current = binding(job)
    # Query defaults in the verdict container using the same Go toolchain and
    # module language version, without invoking any test package's TestMain.
    (directory / 'runtime-defaults.go').write_text('package main\nimport ("encoding/json"; "os"; "runtime")\n'
        'func main() { json.NewEncoder(os.Stdout).Encode(map[string]int{"gomaxprocs":runtime.GOMAXPROCS(0),"num_cpu":runtime.NumCPU()}) }\n')
    for name, command, is_json, cwd in (
        ('packages', ['go', 'list', '-e', '-json', './' + JOBS[job]['package']], True, str(ROOT)),
        ('compile', ['go', 'test', '-c', '-o', str(directory / 'suite.test'), './' + JOBS[job]['package']], False, str(ROOT)),
        ('reporter', ['go', 'build', '-o', str(directory / 'test2json'), 'cmd/test2json'], False, str(ROOT)),
        ('defaults', ['go', 'build', '-o', str(directory / 'runtime-defaults'), str(directory / 'runtime-defaults.go')], False, str(ROOT)),
        ('listing', [str(directory / 'suite.test'), '-test.list=.', '-test.timeout=5m'], False, str(ROOT / JOBS[job]['package']))):
        status = STAGE.stage(directory, name, command, stdout_json=is_json, cwd=cwd)
        if status: return status
    names, excluded = discovery(job, directory)
    durations = None
    timing_file = ROOT / 'ops/ci/runtime/rust-e2e-timings.json'
    if timing_file.is_file(): durations = read(timing_file).get(job, {})
    manifest = {'version': 1, **current, 'tests': names, 'excluded_listing': excluded, 'durations': durations,
                'shards': partition(names, JOBS[job]['shards'], durations)}
    write(directory / 'manifest.json', manifest)
    hashes = {p.name: GO.digest(p) for p in directory.iterdir() if p.is_file()}
    write(directory / 'metadata.json', {'version': 1, **current, 'files': hashes})
    return 0


def verify(job, directory):
    current = binding(job)
    metadata = read(directory / 'metadata.json'); manifest = read(directory / 'manifest.json')
    for data in (metadata, manifest):
        if data.get('version') != 1 or {k: data.get(k) for k in current} != current:
            raise ValueError('E2E compilation source, paths, toolchain or settings differ')
    names = {'packages.json', 'packages.stage.json', 'packages.log', 'compile.log', 'compile.stage.json',
             'reporter.stage.json', 'reporter.log', 'listing.stage.json', 'listing.log',
             'manifest.json', 'suite.test', 'test2json', 'runtime-defaults.go', 'runtime-defaults', 'defaults.log', 'defaults.stage.json'}
    if set(metadata['files']) != names: raise ValueError('Incomplete E2E compilation artifact')
    for name, sha in metadata['files'].items():
        if GO.digest(directory / name) != sha: raise ValueError('Corrupt E2E compilation or discovery artifact')
    selected, excluded = discovery(job, directory)
    if (manifest['tests'] != selected or manifest['excluded_listing'] != excluded
            or manifest['shards'] != partition(selected, JOBS[job]['shards'], manifest['durations'])):
        raise ValueError('E2E shards do not partition complete discovery exactly once')
    for name in ('compile', 'reporter', 'defaults'):
        if read(directory / (name + '.stage.json'))['exit_code'] != 0: raise ValueError('E2E compilation failed')
    return manifest


def parallel_setting(job, directory):
    defaults = json.loads(subprocess.check_output([directory / 'runtime-defaults'], text=True))
    if job == 'proof':
        parallel = int(os.environ['PARALLEL'])
        if parallel <= 0: raise ValueError('Invalid explicit E2E test parallelism')
        return parallel, parallel, defaults
    # The original node/reth recipes omit -parallel. Keep Go's actual default;
    # nproc can see the host instead of the container's GOMAXPROCS quota.
    return None, defaults['gomaxprocs'], defaults


def select(job, directory, reports, index, provider):
    manifest = verify(job, directory)
    total = JOBS[job]['shards']
    if type(index) is not int or not 0 <= index < total: raise ValueError('Invalid E2E shard index')
    if provider == 'circleci' and total > 1:
        names_path = reports / 'all-tests.txt'; names_path.write_text('\n'.join(manifest['tests']) + '\n')
        result = subprocess.run(['circleci', 'tests', 'split', '--split-by=timings', '--timings-type=testname', str(names_path)],
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        (reports / 'circle-split.log').write_text(result.stderr)
        if result.returncode: raise ValueError('Circle E2E timing split failed')
        selected = result.stdout.splitlines()
    else: selected = manifest['shards'][index]
    if len(set(selected)) != len(selected) or not set(selected) <= set(manifest['tests']):
        raise ValueError('Invalid effective E2E selection')
    parallel, effective, defaults = parallel_setting(job, directory)
    record = {**manifest, 'provider': provider, 'shard_index': index, 'assigned_tests': selected,
              'compiled_metadata_sha256': GO.digest(directory / 'metadata.json'),
              'binaries_sha256': {name: GO.digest(directory / name) for name in ('suite.test', 'test2json')},
              'environment': environment(job), 'parallel_flag': parallel, 'effective_parallel': effective, 'go_runtime_defaults': defaults,
              'branch': os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH'),
              'runtime_settings': {k: os.environ.get(k) for k in ('CI', 'GOMAXPROCS', 'GODEBUG', 'FOUNDRY_PROFILE',
                                    'LOG_LEVEL', 'OP_E2E_CANNON_ENABLED', 'ENABLE_ANVIL', 'ENABLE_KURTOSIS')},
              'rwx_run_id': os.environ.get('RWX_RUN_ID'), 'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER'),
              'circle_workflow_id': os.environ.get('CIRCLE_WORKFLOW_ID'), 'started_at': time.time()}
    write(reports / 'selection.json', record)
    return record


def execute(job, directory, reports, names):
    manifest = verify(job, directory); record = read(reports / 'selection.json')
    if record['settings'] != settings(job) or record['source_sha'] != manifest['source_sha'] or not set(names) <= set(record['assigned_tests']):
        raise ValueError('Invalid E2E execution selection')
    if record['compiled_metadata_sha256'] != GO.digest(directory / 'metadata.json'): raise ValueError('E2E compilation changed after selection')
    if record['environment'] != environment(job) or any(os.environ.get(k) != v for k, v in environment(job).items()):
        raise ValueError('E2E runtime environment differs')
    parallel, effective, defaults = parallel_setting(job, directory)
    if parallel != record['parallel_flag'] or effective != record['effective_parallel'] or defaults != record['go_runtime_defaults']:
        raise ValueError('E2E runtime parallelism differs')
    if not names: return 0
    pattern = '^(' + '|'.join(re.escape(n) for n in names) + ')$'
    invocation_dir = reports / 'invocations'; invocation_dir.mkdir(exist_ok=True)
    invocation_path = invocation_dir / ((names[0] if JOBS[job]['per_test'] else 'package') + '.json')
    if invocation_path.exists(): raise ValueError('Duplicate E2E binary invocation')
    args = (directory / 'test2json', directory / 'suite.test', PREFIX + JOBS[job]['package'])
    invocation = {'source_sha': manifest['source_sha'], 'assigned_tests': names,
                  'argv': GO.binary_command(*args, parallel, JOBS[job]['timeout'], pattern),
                  'cwd': str(ROOT / JOBS[job]['package']), 'started_at': time.time(), 'exit_code': None,
                  'binary_sha256': GO.digest(directory / 'suite.test'), 'reporter_sha256': GO.digest(directory / 'test2json')}
    write(invocation_path, invocation)
    try:
        status = GO.execute_binary(*args, ROOT / JOBS[job]['package'], parallel, JOBS[job]['timeout'], pattern)
        invocation['exit_code'] = status
        return status
    finally:
        exception = sys.exception()
        if isinstance(exception, SystemExit): invocation['exit_code'] = exception.code
        invocation['finished_at'] = time.time(); write(invocation_path, invocation)


def invocations(record, directory):
    """Check actual process provenance independently of projected test results."""
    job = record['settings']['job']
    rows = [read(path) for path in sorted((directory / 'invocations').glob('*.json'))]
    expected = len(record['assigned_tests']) if JOBS[job]['per_test'] else int(bool(record['assigned_tests']))
    if len(rows) != expected or any(i['exit_code'] != 0 or i['source_sha'] != record['source_sha'] for i in rows):
        raise ValueError('Missing, duplicate or failed original E2E binary invocation')
    if Counter(n for i in rows for n in i['assigned_tests']) != Counter(record['assigned_tests']):
        raise ValueError('Original binary invocations do not cover the complete E2E assignment')
    if job != 'proof' and record['parallel_flag'] is not None:
        raise ValueError('Node/reth E2E must preserve default Go parallelism')
    for row in rows:
        if row['cwd'] != str(Path(record['workspace_root']) / JOBS[job]['package']):
            raise ValueError('E2E binary working directory differs')
        if (row['binary_sha256'] != record['binaries_sha256']['suite.test']
                or row['reporter_sha256'] != record['binaries_sha256']['test2json']):
            raise ValueError('Original E2E invocation binaries differ from verified selection')
        expected_flags = {'-test.v=test2json', '-test.count=1', '-test.timeout=' + JOBS[job]['timeout'], '-test.paniconexit0',
                          '-test.run=^(' + '|'.join(re.escape(n) for n in row['assigned_tests']) + ')$'}
        if record['parallel_flag'] is not None: expected_flags.add('-test.parallel=' + str(record['parallel_flag']))
        flags = [a for a in row['argv'] if a.startswith('-test.')]
        if len(flags) != len(set(flags)) or set(flags) != expected_flags:
            raise ValueError('Original E2E execution flags differ')
        if JOBS[job]['per_test'] and len(row['assigned_tests']) != 1:
            raise ValueError('Proof E2E tests must use separate processes')
    return rows


def report(job, reports, status):
    errors, cases, terminals, originals = [], {}, Counter(), {}
    record = read(reports / 'selection.json') if (reports / 'selection.json').exists() else None
    logs = sorted((reports / 'events').glob('*.json'))
    combined = reports / 'original.json'
    combined.write_bytes(b''.join(p.read_bytes() for p in logs))
    try:
        if record is None: raise ValueError('Missing E2E selection')
        if not logs and record['assigned_tests']: raise ValueError('Missing original E2E events')
        for event in PROJECT.events(combined):
            if event.get('Package') != PREFIX + JOBS[job]['package']: raise ValueError('Unexpected E2E event package')
            name, action = event.get('Test'), event['Action']
            if name and action in ('pass', 'skip', 'fail'):
                if '/' not in name: terminals[name] += 1
                if name in cases: raise ValueError('Duplicate E2E case verdict; retries are disabled')
                cases[name] = {'outcome': action, 'elapsed': event.get('Elapsed')}
        coverage = {'assigned': record['assigned_tests'], 'missing': sorted(set(record['assigned_tests']) - terminals.keys()),
                    'extra': sorted(terminals.keys() - set(record['assigned_tests'])),
                    'duplicates': sorted(n for n, count in terminals.items() if count != 1), 'cases': cases}
        write(reports / 'coverage.json', coverage)
        if coverage['missing'] or coverage['extra'] or coverage['duplicates']: raise ValueError('Incomplete or duplicate E2E test verdicts')
        if status == 0 and any(c['outcome'] == 'fail' for c in cases.values()): raise ValueError('Successful E2E command contains failed verdicts')
        if status == 0: invocations(record, reports)
        junit_cases = {}
        import xml.etree.ElementTree as ET
        for path in (reports / 'junit').glob('*.xml'):
            for case in ET.parse(path).getroot().iter('testcase'):
                key = case.attrib['name']
                if key in junit_cases: raise ValueError('Duplicate E2E JUnit case')
                junit_cases[key] = 'fail' if case.find('failure') is not None or case.find('error') is not None else 'skip' if case.find('skipped') is not None else 'pass'
        if junit_cases != {k: v['outcome'] for k, v in cases.items()}: raise ValueError('E2E JUnit and original events differ')
        PROJECT.project(combined, reports / 'native.json')
    except (ValueError, OSError, KeyError) as error: errors.append(str(error))
    # Keep partial/canceled originals; a reporting failure never replaces the
    # first command status or fabricates a terminal test verdict.
    for path in sorted(reports.rglob('*')):
        if path.is_file() and path.name != 'final.json': originals[str(path.relative_to(reports))] = GO.digest(path)
    write(reports / 'final.json', {'exit_code': status, 'report_errors': errors, 'finished_at': time.time(),
                                 'original_sha256': originals})
    if errors: print('\n'.join(errors), file=sys.stderr)
    return status or bool(errors)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['compile', 'verify', 'select', 'execute', 'report', 'env'])
    parser.add_argument('job', choices=JOBS)
    parser.add_argument('arguments', nargs='*')
    args = parser.parse_args()
    job = args.job
    directory = Path(os.environ.get('CI_E2E_COMPILED', ROOT / '.ci/rust-e2e/compiled' / job))
    reports = ROOT / '.ci/rust-e2e/reports' / job
    if args.command == 'env':
        # NUL pairs are consumed by Bash without eval or command substitution.
        for key, value in environment(job).items(): sys.stdout.buffer.write(key.encode() + b'\0' + value.encode() + b'\0')
    elif args.command == 'compile': return compile_suite(job)
    elif args.command == 'verify': verify(job, directory)
    elif args.command == 'select':
        reports.mkdir(parents=True, exist_ok=True)
        record = select(job, directory, reports, int(os.environ['CI_SHARD_INDEX']), os.environ['CI_E2E_PROVIDER'])
        (reports / 'assigned.txt').write_text('\n'.join(record['assigned_tests']) + ('\n' if record['assigned_tests'] else ''))
    elif args.command == 'execute':
        signal.signal(signal.SIGINT, GO.cancel); signal.signal(signal.SIGTERM, GO.cancel)
        return execute(job, directory, reports, args.arguments)
    elif args.command == 'report': return report(job, reports, int(args.arguments[0]))
    return 0


if __name__ == '__main__':
    try: sys.exit(main())
    except (ValueError, OSError, KeyError, subprocess.CalledProcessError) as error:
        print(f'ERROR: {error}', file=sys.stderr); sys.exit(1)
