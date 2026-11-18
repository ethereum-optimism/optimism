#!/usr/bin/env python3
"""Compile, discover and freshly run the complete Rust E2E Go workloads."""
import argparse
from collections import Counter
import importlib.util
import os
from pathlib import Path
import re
import shutil
import signal
import sys

ROOT = Path(__file__).resolve().parents[3]
PREFIX = 'github.com/ethereum-optimism/optimism/'


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


GO = helper('go-compiled-tests')
PROJECT = helper('go-report')
REPORT = helper('ci-report')
# One entry per CircleCI E2E job. Each comment names the job and the recipe that it runs.
# RWX runs the compiled suite, not the recipe. `package` and `timeout` copy that recipe,
# `shards` copies the job's `parallelism`, and environment() copies the recipe's exports.
JOBS = {
    # kona-proof-action-single. rust/kona: just action-tests-single-run
    'proof': {'package': 'rust/kona/tests/proofs', 'shards': 8, 'timeout': '60m'},
    # rust-e2e-restart. rust/kona: just test-e2e-sysgo-run node node/restart
    'restart': {'package': 'rust/kona/tests/node/restart', 'shards': 3, 'timeout': '40m'},
    # rust-e2e-simple-kona. rust/kona: just test-e2e-sysgo-run node node/common simple-kona
    'simple-kona': {'package': 'rust/kona/tests/node/common', 'shards': 1, 'timeout': '40m'},
    # rust-e2e-simple-kona-sequencer. rust/kona: just test-e2e-sysgo-run node node/common simple-kona-sequencer
    'simple-kona-sequencer': {'package': 'rust/kona/tests/node/common', 'shards': 1, 'timeout': '40m'},
    # op-reth-e2e-sysgo-tests. rust/op-reth/tests: just test-e2e-sysgo
    'op-reth': {'package': 'rust/op-reth/tests/proofs/core', 'shards': 1, 'timeout': '40m'},
}


read = REPORT.read


write = REPORT.write


def environment(job):
    values = {'DISABLE_OP_E2E_LEGACY': 'true', 'DEVSTACK_ORCHESTRATOR': 'sysgo'} if job != 'proof' else {}
    if job == 'proof':
        values.update(KONA_HOST_PATH=str(ROOT / 'rust/target/release/kona-host'),
                      RUST_BINARY_PATH_OP_RETH_TEST_ENGINE=str(ROOT / 'rust/target/release/op-reth-test-engine'))
    elif job == 'op-reth':
        values.update(OP_RETH_ENABLE_PROOF_HISTORY='true',
                      RUST_BINARY_PATH_OP_RETH=str(ROOT / 'rust/target/release/op-reth'))
    else:
        sequencer = job == 'simple-kona-sequencer'
        values.update(KONA_SEQUENCER_WITH_RETH=str(int(sequencer)), KONA_VALIDATOR_WITH_RETH='1',
                      OP_SEQUENCER_WITH_RETH=str(int(not sequencer)), OP_VALIDATOR_WITH_RETH=str(int(not sequencer)),
                      RUST_BINARY_PATH_OP_RETH=str(ROOT / 'rust/target/release/op-reth'),
                      RUST_BINARY_PATH_KONA_NODE=str(ROOT / 'rust/target/release/kona-node'))
    return values


def partition(names, total):
    """Assign each name to exactly one of `total` shards, round-robin in the given order."""
    return [names[index::total] for index in range(total)]


def discovery(job, directory):
    """Return the tests that `job` runs and the listed names that it leaves out.

    The compiled suite wrote its test list to `listing.log`. Raise ValueError
    when a name is listed twice or when no test remains.
    """
    names = [line for line in (directory / 'listing.log').read_text().splitlines()
             if re.fullmatch(r'(Test|Example|Fuzz)\w*', line)]
    if len(set(names)) != len(names): raise ValueError('Duplicate E2E listed test')
    # Original Circle sharding keeps ^Test names only. Unsharded packages run
    # examples and fuzz seeds too; record that distinction instead of hiding it.
    excluded = [n for n in names if JOBS[job]['shards'] > 1 and not n.startswith('Test')]
    selected = sorted(set(names) - set(excluded))
    if not selected: raise ValueError('Empty E2E test discovery')
    return selected, excluded


def compile_suite(job):
    directory = ROOT / '.ci/rust-e2e/compiled' / job
    shutil.rmtree(directory, ignore_errors=True); directory.mkdir(parents=True)
    package = JOBS[job]['package']
    for name, command, cwd in (
        ('compile', ['go', 'test', '-c', '-o', str(directory / 'suite.test'), './' + package], ROOT),
        ('reporter', ['go', 'build', '-o', str(directory / 'test2json'), 'cmd/test2json'], ROOT),
        ('listing', [str(directory / 'suite.test'), '-test.list=.', '-test.timeout=5m'], ROOT / package)):
        status = REPORT.stage(directory, name, command, cwd=str(cwd))
        if status: return status
    names, excluded = discovery(job, directory)
    write(directory / 'manifest.json', {'tests': names, 'excluded_listing': excluded,
                                        'shards': partition(names, JOBS[job]['shards'])})
    return 0


def parallel_flag(job):
    # Only the proof recipe passes -parallel. The node and reth recipes keep Go's default.
    return int(os.environ['PARALLEL']) if job == 'proof' else None


def select(directory, reports, index):
    shards = read(directory / 'manifest.json')['shards']
    if not 0 <= index < len(shards): raise ValueError('Invalid E2E shard index')
    write(reports / 'selection.json', {'shard_index': index, 'assigned_tests': shards[index]})
    (reports / 'assigned.txt').write_text(''.join(name + '\n' for name in shards[index]))


def execute(job, directory, names):
    """Run `names` from the compiled suite in one fresh process and print its test2json events."""
    package = JOBS[job]['package']
    pattern = '^(' + '|'.join(re.escape(n) for n in names) + ')$'
    return GO.execute_binary(directory / 'test2json', directory / 'suite.test', PREFIX + package,
                             ROOT / package, parallel_flag(job), JOBS[job]['timeout'], pattern)


def report(job, reports, status):
    errors, cases, terminals = [], {}, Counter()
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
        PROJECT.project(combined, reports / 'native.json')
    except (ValueError, OSError, KeyError) as error: errors.append(str(error))
    # A reporting failure never replaces the first command status and never
    # fabricates a terminal test verdict. Partial and canceled originals stay.
    write(reports / 'final.json', {'exit_code': status, 'report_errors': errors})
    if errors: print('\n'.join(errors), file=sys.stderr)
    return status or bool(errors)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['compile', 'select', 'execute', 'report', 'env'])
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
    elif args.command == 'select':
        reports.mkdir(parents=True, exist_ok=True)
        select(directory, reports, int(os.environ['CI_SHARD_INDEX']))
    elif args.command == 'execute':
        signal.signal(signal.SIGINT, GO.cancel); signal.signal(signal.SIGTERM, GO.cancel)
        return execute(job, directory, args.arguments)
    elif args.command == 'report': return report(job, reports, int(args.arguments[0]))
    return 0


if __name__ == '__main__':
    try: sys.exit(main())
    except (ValueError, OSError, KeyError) as error:
        print(f'ERROR: {error}', file=sys.stderr); sys.exit(1)
