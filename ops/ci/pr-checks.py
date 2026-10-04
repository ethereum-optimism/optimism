#!/usr/bin/env python3
"""Run complete PR checks and retain their original commands and coverage."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import xml.etree.ElementTree as ET

SPEC = importlib.util.spec_from_file_location('stages', Path(__file__).with_name('rust-workspace-report.py'))
STAGES = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(STAGES)
ROOT = Path(__file__).resolve().parents[2]
CONTRACTS = ROOT / 'packages/contracts-bedrock'
ANSI = re.compile(r'\x1b\[[0-?]*[ -/]*[@-~]')


def objects(text):
    decoder, offset, result = json.JSONDecoder(), 0, []
    while offset < len(text):
        if text[offset].isspace(): offset += 1; continue
        value, offset = decoder.raw_decode(text, offset)
        if not isinstance(value, dict): raise ValueError('Expected complete Go module objects')
        result.append(value)
    return result


def modules(text):
    rows = objects(text)
    if not rows or any(row.get('Error') for row in rows): raise ValueError('Empty or failed Go module discovery')
    def identity(row):
        fields = ('Path', 'Version', 'Sum', 'GoModSum', 'Main', 'Indirect', 'GoVersion')
        result = {k: row[k] for k in fields if k in row}
        if not result.get('Path'): raise ValueError('Missing module identity')
        if 'Replace' in row: result['Replace'] = identity(row['Replace'])
        return result
    result = sorted((identity(row) for row in rows), key=lambda r: r['Path'])
    if len({r['Path'] for r in result}) != len(result) or sum(r.get('Main') is True for r in result) != 1:
        raise ValueError('Duplicate module discovery or missing main module')
    return result


def checks(configuration):
    phases = configuration.get('phases', [])
    if not phases or not isinstance(phases, list): raise ValueError('Empty check-runner phases')
    names = []
    for phase in phases:
        if not phase.get('name') or not phase.get('checks'): raise ValueError('Incomplete check-runner phase')
        for check in phase['checks']:
            if not check.get('name') or not check.get('command'): raise ValueError('Incomplete check command')
            names.append(check['name'])
    if len(names) != len(set(names)): raise ValueError('Duplicate authoritative check selection')
    for phase in phases:
        selected = {check['name'] for check in phase['checks']}
        for check in phase['checks']:
            if not set(check.get('depends', [])) <= selected: raise ValueError('Missing check dependency')
    return names


def check_verdicts(text, configuration):
    names = checks(configuration); plain = ANSI.sub('', text).replace('\r', '\n')
    selected, histories, retries = set(names), {n: [] for n in names}, set()
    for line in plain.splitlines():
        skipped = re.search(r'✗\s+([\w-]+) \(skipped\)', line)
        if skipped:
            name = skipped[1]
            if name not in selected: raise ValueError('Unexpected original skipped check')
            row = {'outcome': 'skip', 'role': 'initial', 'reason': 'dependency failed', 'elapsed_seconds': None}
            if row not in histories[name]: histories[name].append(row)
            continue
        match = re.search(r'(✓|✗|↻) ([\w-]+)(?: \((retry|unblocked)\))? (\d+\.\d+)s(?: |$)', line)
        if not match: continue
        sign, name, role, seconds = match.groups()
        if name not in selected: raise ValueError('Unexpected original check verdict: ' + name)
        row = {'outcome': 'pass' if sign == '✓' else 'fail', 'role': role or 'initial', 'elapsed_seconds': float(seconds)}
        # Spinner redraws repeat a status. Distinct initial/retry verdicts remain.
        if row not in histories[name]: histories[name].append(row)
        if sign == '↻' or role == 'retry': retries.add(name)
    for name, history in histories.items():
        if len([r for r in history if r['role'] == 'initial']) != 1 or len(history) > 2:
            raise ValueError('Missing or duplicate initial check invocation')
        if len(history) == 2:
            if (history[0]['outcome'], history[1]['role']) not in (('fail', 'retry'), ('skip', 'unblocked')):
                raise ValueError('Unexpected check retry or unblocked history')
    if any(not history or history[-1]['outcome'] != 'pass' for history in histories.values()):
        raise ValueError('Missing or unsuccessful original check verdict')
    summary = re.findall(r'✓ All checks passed \((\d+)/(\d+)\)', plain)
    if summary != [(str(len(names)), str(len(names)))]: raise ValueError('Incomplete full check-runner summary')
    return {'checks': [{'name': n, 'outcome': 'pass', 'attempts': histories[n]} for n in names],
            'selected': len(names), 'retries': sorted(retries), 'configuration': configuration}


def begin(directory, job):
    sha = STAGES.command('git', 'rev-parse', 'HEAD')
    expected = os.environ.get('CI_COMMIT_SHA', os.environ.get('CIRCLE_SHA1', sha))
    if not re.fullmatch('[0-9a-f]{40}', expected) or sha != expected: raise ValueError('PR check source revision differs')
    paths = ['go.mod', 'go.sum', 'mise.toml', 'ops/ci/pr-checks.py', 'ops/ci/rust-workspace-report.py']
    if job == 'contracts-fast': paths += ['packages/contracts-bedrock/checks.yaml', 'packages/contracts-bedrock/justfile',
                                        'packages/contracts-bedrock/scripts/check-runner/main.go',
                                        'packages/contracts-bedrock/scripts/ops/get-target-branch.sh']
    settings = {'source_sha': sha, 'job': job, 'provider': os.environ.get('CI_CHECK_PROVIDER', 'circleci'),
                'workspace_root': str(ROOT), 'branch': os.environ.get('CI_BRANCH', os.environ.get('CIRCLE_BRANCH')),
                'go': STAGES.command('go', 'version'), 'just': STAGES.command('just', '--version'),
                'input_sha256': {name: STAGES.digest(ROOT / name) for name in paths},
                'rwx_run_id': os.environ.get('RWX_RUN_ID'), 'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}
    if job == 'contracts-fast':
        settings.update(forge=STAGES.command('forge', '--version'), semgrep=STAGES.command('semgrep', '--version'))
        settings.update(target(directory))
    STAGES.write(directory / 'settings.json', settings)
    return settings


def target(directory):
    # Resolve through the original helper. The isolated clone contains HEAD's
    # full history, but can lack the protected remote ref used by semver-diff.
    branch = subprocess.check_output(['bash', '-c',
        'source ./scripts/ops/get-target-branch.sh; printf "%s\\n" "$TARGET_BRANCH"'], cwd=CONTRACTS, text=True).strip()
    subprocess.check_call(['git', 'check-ref-format', '--branch', branch], cwd=ROOT, stdout=subprocess.DEVNULL)
    ref = 'origin/' + branch
    status = STAGES.stage(directory, 'fetch-target', ['git', 'fetch', '--no-tags', 'origin',
                           '+refs/heads/' + branch + ':refs/remotes/' + ref], cwd=str(ROOT))
    if status: raise ValueError('Authoritative contract target history unavailable')
    return {'target_branch': branch, 'target_sha': STAGES.command('git', 'rev-parse', ref),
            'merge_base_sha': STAGES.command('git', 'merge-base', ref, 'HEAD')}


def execute(directory, job):
    if job == 'go-modules':
        for attempt in range(5):
            status = STAGES.stage(directory, 'download-' + str(attempt), ['go', 'mod', 'download'], cwd=str(ROOT))
            if status == 0: break
            if status >= 128 or attempt == 4: return status
            __import__('time').sleep(5 * 2 ** attempt)
        status = STAGES.stage(directory, 'verify', ['go', 'mod', 'verify'], cwd=str(ROOT))
        if status: return status
        return STAGES.stage(directory, 'discovery', ['go', 'list', '-m', '-json', 'all'], stdout_json=True, cwd=str(ROOT))
    for name in list(os.environ):
        if name.startswith(('FOUNDRY_', 'DAPP_', 'DEV_FEATURE__', 'SYS_FEATURE__')): del os.environ[name]
    for name in ('ETH_RPC_URL', 'ETH_RPC_JWT', 'ETH_RPC_HEADERS', 'ETHERSCAN_API_KEY',
                 'MAINNET_RPC_URL', 'FORK_RPC_URL', 'FORK_BLOCK_NUMBER', 'L2_FORK_RPC_URL', 'L2_FORK_BLOCK_NUMBER'):
        os.environ.pop(name, None)
    (directory / 'submodules.txt').write_text(STAGES.command('git', 'submodule', 'status', '--recursive') + '\n')
    status = STAGES.stage(directory, 'foundry-config', ['forge', 'config', '--json'], stdout_json=True, cwd=str(CONTRACTS))
    if status: return status
    configuration = json.loads(subprocess.check_output(['yq', '-o=json', '.', 'checks.yaml'], cwd=CONTRACTS))
    checks(configuration); STAGES.write(directory / 'selection.json', configuration)
    # CI disables the local lint-fix command; the complete checked-in phase,
    # build, command, dependency and retry behavior remains authoritative.
    os.environ['CI'] = 'true'
    return STAGES.stage(directory, 'checks', ['just', 'check-fast', '-verbose'], cwd=str(CONTRACTS))


def finish(directory, job, status, errors):
    coverage = None
    try:
        if status == 0 and not errors:
            if job == 'go-modules':
                rows = modules((directory / 'discovery.json').read_text())
                coverage = {'modules': rows, 'selected': len(rows), 'verified': True,
                            'download_attempts': len(list(directory.glob('download-*.stage.json')))}
            else: coverage = check_verdicts((directory / 'checks.log').read_text(),
                                          json.loads((directory / 'selection.json').read_text()))
            STAGES.write(directory / 'coverage.json', coverage)
    except (OSError, ValueError, KeyError) as error: errors.append(str(error))
    if status == 0 and errors: status = 1
    cases = coverage['checks'] if coverage and job == 'contracts-fast' else [{'name': job, 'outcome': 'pass' if status == 0 else 'fail'}]
    suite = ET.Element('testsuite', name='pr-checks/' + job, tests=str(len(cases)))
    for row in cases:
        case = ET.SubElement(suite, 'testcase', classname=job, name=row['name'])
        if row['outcome'] != 'pass': ET.SubElement(case, 'failure', message='Original PR check failed')
    ET.ElementTree(suite).write(directory / 'checks.junit.xml', encoding='utf-8', xml_declaration=True)
    STAGES.write(directory / 'final.json', {'exit_code': status, 'report_errors': errors,
        'original_sha256': {str(p.relative_to(directory)): STAGES.digest(p) for p in sorted(directory.rglob('*'))
                            if p.is_file() and p.name != 'final.json'}})
    return status


def main():
    os.chdir(ROOT)
    parser = argparse.ArgumentParser(description=__doc__); parser.add_argument('job', choices=('go-modules', 'contracts-fast'))
    args = parser.parse_args(); directory = ROOT / '.ci/pr-checks' / args.job
    shutil.rmtree(directory, ignore_errors=True); directory.mkdir(parents=True)
    status, errors = 1, []
    try:
        settings = begin(directory, args.job); status = execute(directory, args.job)
        if any(STAGES.digest(ROOT / name) != digest for name, digest in settings['input_sha256'].items()):
            raise ValueError('PR check source changed during execution')
        if args.job == 'contracts-fast' and STAGES.command('git', 'rev-parse', 'origin/' + settings['target_branch']) != settings['target_sha']:
            raise ValueError('Contract target revision changed during execution')
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as error: errors.append(str(error)); print(error, file=sys.stderr)
    return finish(directory, args.job, status, errors)


if __name__ == '__main__': sys.exit(main())
