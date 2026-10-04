#!/usr/bin/env python3
"""Run the original Main validators with complete discovery and sealed evidence."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tomllib
import xml.etree.ElementTree as ET

SPEC = importlib.util.spec_from_file_location('stages', Path(__file__).with_name('rust-workspace-report.py'))
STAGES = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(STAGES)
SPEC = importlib.util.spec_from_file_location('pr', Path(__file__).with_name('pr-checks.py'))
PR = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(PR)
ROOT = Path(__file__).resolve().parents[2]
MOCKS = {'check-generated-mocks-op-node': 'op-node', 'check-generated-mocks-op-service': 'op-service'}
COMMANDS = {
    'todo-issues-check': ['./ops/scripts/todo-checker.sh', '--verbose', '--strict'],
    'l2-chains-sync-check': ['bash', '.circleci/scripts/check-l2-chains-sync.sh'],
    'op-deployer-forge-version': ['just', 'check-forge-version'],
    'check-op-geth-version': ['just', 'check-op-geth-version'],
    'check-nut-locks': ['go', 'run', './ops/scripts/check-nut-locks'],
    **{job: ['bash', '-eo', 'pipefail', '-c', 'just generate-mocks-' + component + ' && git diff --exit-code']
       for job, component in MOCKS.items()},
}
GO_JOBS = {'check-op-geth-version', 'check-nut-locks', *MOCKS}
GO_FIELDS = ('GoFiles', 'CgoFiles', 'TestGoFiles', 'XTestGoFiles')
TODO_GLOBS = ['-g', '!ops/scripts/todo-checker.sh', '-g', '!packages/contracts-bedrock/lib']
TODO_DISCOVERY = {
    'files': ['rg', '--files', *TODO_GLOBS],
    'todos': ['rg', '-o', '--with-filename', '-i', '-n', *TODO_GLOBS, r'TODO\(([^)]+)\):?( [^,;]*)?'],
    'near-misses': ['bash', '-eo', 'pipefail', '-c',
        "rg --with-filename -i -n -g '!ops/scripts/todo-checker.sh' -g '!packages/contracts-bedrock/lib' "
        "'\\b(TODO|FIXME)\\b[^\\w\\n]{0,5}[\\w./-]*#[0-9]+' | rg -v -i '\\b(TODO|FIXME)\\([^)]+\\)'"],
}


def command(*args, cwd=None):
    return subprocess.check_output(args, cwd=ROOT if cwd is None else cwd, text=True).strip()


def inputs():
    result = {}
    for entry in command('git', 'ls-files', '--stage', '-z').split('\0'):
        if not entry: continue
        header, name = entry.split('\t', 1)
        mode, revision, merge_stage = header.split()
        if merge_stage != '0' or name in result: raise ValueError('Unmerged or duplicate tracked Main input')
        path = ROOT / name
        if mode == '160000': result[name] = {'gitlink': revision}
        elif path.is_symlink(): result[name] = {'symlink': os.readlink(path)}
        elif path.is_file(): result[name] = STAGES.digest(path)
        else: raise ValueError('Missing tracked Main input: ' + name)
    for name in ('main-checks.py', 'pr-checks.py', 'rust-workspace-report.py'):
        path = ROOT / 'ops/ci' / name
        result[str(path.relative_to(ROOT))] = STAGES.digest(path)
    return result


def tracked_generated(component):
    return {name: STAGES.digest(ROOT / name) for name in command('git', 'ls-files', component).splitlines()
            if name.endswith('.go') and re.search(rb'^// Code generated .*DO NOT EDIT\.', (ROOT / name).read_bytes(), re.M)}


def packages(text, component, workspace=None, source_root=None):
    workspace = ROOT if workspace is None else Path(workspace)
    source_root = ROOT if source_root is None else Path(source_root)
    rows = PR.objects(text); result = []
    for row in rows:
        if row.get('Error') or row.get('DepsErrors') or not row.get('ImportPath'):
            raise ValueError('Failed complete Go generation package discovery')
        directory = Path(row['Dir'])
        if not directory.is_relative_to(workspace / component): raise ValueError('Generation package outside selected component')
        relative = directory.relative_to(workspace)
        files, directives, names = {}, [], set()
        for field in GO_FIELDS:
            for filename in row.get(field, []):
                if Path(filename).name != filename: raise ValueError('Generation file outside selected package')
                names.add(filename)
        for filename in sorted(names):
            name = str(relative / filename); path = source_root / name
            files[name] = STAGES.digest(path)
            for number, line in enumerate(path.read_text().splitlines(), 1):
                if re.match(r'^//go:generate[ \t]', line): directives.append({'file': name, 'line': number, 'directive': line})
        result.append({'package': row['ImportPath'], 'directory': str(relative),
                       'files': files, 'directives': directives, 'ignored_go_files': row.get('IgnoredGoFiles', [])})
    if not result or len({r['package'] for r in result}) != len(result): raise ValueError('Empty or duplicate generation package discovery')
    if not any(r['directives'] for r in result): raise ValueError('Empty complete generator selection')
    return sorted(result, key=lambda r: r['package'])


def retain_sources(directory, names):
    """Retain original input bytes so derived selections can be revalidated."""
    for name in sorted(set(names)):
        relative = Path(name)
        if relative.is_absolute() or '..' in relative.parts: raise ValueError('Unsafe Main source path')
        destination = directory / 'source' / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes((ROOT / relative).read_bytes())


def l2_selection(configuration, rpcs):
    matrices = [body['contracts-bedrock-tests-l2-fork']['matrix']['parameters']['fork_op_chain']
                for body in configuration['workflows']['scheduled-daily-tests']['jobs']
                if isinstance(body, dict) and 'contracts-bedrock-tests-l2-fork' in body]
    if len(matrices) != 1 or not matrices[0] or len(set(matrices[0])) != len(matrices[0]):
        raise ValueError('Missing or duplicate complete scheduled L2 fork matrix')
    if not isinstance(rpcs, dict) or not rpcs: raise ValueError('Missing complete L2 RPC chain selection')
    return {'rpc_chains': sorted(rpcs), 'matrix_chains': sorted(matrices[0])}


def nut_selection(source_root, develop_sha):
    source_root = Path(source_root)
    locks = tomllib.loads((source_root / 'op-core/nuts/fork_lock.toml').read_text())
    bundles = {str(p.relative_to(source_root)): STAGES.digest(p) for p in sorted((source_root / 'op-core/nuts/bundles').glob('*_nut_bundle.json'))}
    states = {str(p.relative_to(source_root)): STAGES.digest(p) for p in sorted((source_root / 'op-core/nuts/state').glob('*_state.json'))}
    if not locks or not bundles or not states: raise ValueError('Empty complete NUT lock selection')
    return {'locks': locks, 'bundles': bundles, 'states': states, 'develop_sha': develop_sha}


def stage(directory, name, argv, cwd=None, json_output=False, allowed=(0,)):
    status = STAGES.stage(directory, name, argv, stdout_json=json_output, cwd=str(ROOT if cwd is None else cwd), stdin=subprocess.DEVNULL)
    if status not in allowed: raise ValueError('Original Main stage failed: ' + name + ' (exit ' + str(status) + ')')
    return status


def discovery(directory, job):
    if job == 'todo-issues-check':
        for name, argv in TODO_DISCOVERY.items():
            stage(directory, name, argv, allowed=(0,) if name == 'files' else (0, 1))
        files = (directory / 'files.log').read_text().splitlines()
        if not files or len(files) != len(set(files)): raise ValueError('Empty or duplicate complete TODO file discovery')
        return {'files': sorted(files), 'todos': sorted((directory / 'todos.log').read_text().splitlines()),
                'near_misses': sorted((directory / 'near-misses.log').read_text().splitlines()), 'check_closed': False}
    if job == 'l2-chains-sync-check':
        stage(directory, 'configuration', ['yq', '-o=json', '.', '.circleci/continue/main.yml'], json_output=True)
        configuration = json.loads((directory / 'configuration.json').read_text())
        retain_sources(directory, ['.circleci/continue/main.yml', '.circleci/l2-rpcs.json'])
        return l2_selection(configuration, json.loads((ROOT / '.circleci/l2-rpcs.json').read_text()))
    if job == 'op-deployer-forge-version':
        stage(directory, 'mise-config', ['yq', '-o=json', '.', 'mise.toml'], json_output=True)
        retain_sources(directory, ['mise.toml', 'op-deployer/pkg/deployer/forge/version.json'])
        return {'mise_forge': json.loads((directory / 'mise-config.json').read_text())['tools']['forge'],
                'deployer': json.loads((ROOT / 'op-deployer/pkg/deployer/forge/version.json').read_text())}
    if job == 'check-op-geth-version':
        stage(directory, 'go-mod', ['go', 'mod', 'edit', '-json'], json_output=True)
        retain_sources(directory, ['go.mod'])
        return json.loads((directory / 'go-mod.json').read_text())
    if job == 'check-nut-locks':
        stage(directory, 'fetch-develop', ['git', 'fetch', '--no-tags', 'origin', '+refs/heads/develop:refs/remotes/origin/develop'])
        selected = nut_selection(ROOT, command('git', 'rev-parse', 'origin/develop'))
        retain_sources(directory, ['op-core/nuts/fork_lock.toml', *selected['bundles'], *selected['states']])
        return selected
    component = MOCKS[job]
    stage(directory, 'packages', ['go', 'list', '-tags=generate', '-json', './...'], ROOT / component, True)
    selected = packages((directory / 'packages.json').read_text(), component)
    retain_sources(directory, command('git', 'ls-files', component + '/**/*.go', component + '/*.go').splitlines())
    stage(directory, 'generators', ['go', 'generate', '-n', '-v', './...'], ROOT / component)
    return {'component': component, 'packages': selected, 'generated_before': tracked_generated(component)}


def main():
    os.chdir(ROOT); parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('job', choices=COMMANDS); args = parser.parse_args(); job = args.job
    directory = ROOT / '.ci/main-checks' / job
    shutil.rmtree(directory, ignore_errors=True); directory.mkdir(parents=True)
    status, errors, settings = 1, [], None
    try:
        sha = command('git', 'rev-parse', 'HEAD'); expected = os.environ.get('CI_COMMIT_SHA') or os.environ.get('CIRCLE_SHA1') or sha
        if not re.fullmatch('[0-9a-f]{40}', expected) or sha != expected: raise ValueError('Main check source revision differs')
        tools = {'go': command('go', 'version'), 'just': command('just', '--version'),
                 'jq': command('jq', '--version'), 'yq': command('yq', '--version')}
        if job == 'todo-issues-check': tools['ripgrep'] = command('rg', '--version')
        if job in MOCKS:
            versions = re.findall(r'^v?\d+\.\d+\.\d+[^\n]*$', command('mockery', '--version'), re.M)
            if len(versions) != 1: raise ValueError('Missing authoritative mockery version')
            tools['mockery'] = versions[0]
        settings = {'source_sha': sha, 'job': job, 'branch': os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH'),
                    'provider': os.environ.get('CI_CHECK_PROVIDER', 'circleci'), 'workspace_root': str(ROOT),
                    'tools': tools, 'input_sha256': inputs(), 'rwx_run_id': os.environ.get('RWX_RUN_ID'),
                    'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}
        if job in GO_JOBS:
            stage(directory, 'go-env', ['go', 'env', '-json', 'GOOS', 'GOARCH', 'CGO_ENABLED', 'GOFLAGS', 'GOTOOLCHAIN'], json_output=True)
            settings['go_environment'] = json.loads((directory / 'go-env.json').read_text())
        STAGES.write(directory / 'settings.json', settings)
        selected = discovery(directory, job); STAGES.write(directory / 'selection.json', selected)
        cwd = ROOT / 'op-deployer' if job == 'op-deployer-forge-version' else ROOT
        os.environ.pop('CI_TODO_CHECKER_PAT', None)
        # Ripgrep otherwise consumes a piped parent stdin instead of searching
        # the checkout. These CI validators do not accept interactive input.
        status = STAGES.stage(directory, 'check', COMMANDS[job], cwd=str(cwd), stdin=subprocess.DEVNULL)
        if status: errors.append('Original Main command exited ' + str(status))
        after = inputs(); STAGES.write(directory / 'inputs-after.json', after)
        if job in MOCKS:
            generated = tracked_generated(MOCKS[job]); STAGES.write(directory / 'generated.json', generated)
        if status == 0:
            if job in MOCKS:
                if not generated or generated != selected['generated_before']: raise ValueError('Generated tracked mocks differ from committed inputs')
            if settings['input_sha256'] != after: raise ValueError('Main check changed tracked source inputs')
            STAGES.write(directory / 'coverage.json', {'job': job, 'outcome': 'pass', 'attempts': 1,
                         'complete_selection': True, 'command': COMMANDS[job]})
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as error:
        errors.append(str(error)); print(error, file=sys.stderr); status = 1
    suite = ET.Element('testsuite', name='main-checks', tests='1', failures='0' if status == 0 else '1')
    case = ET.SubElement(suite, 'testcase', classname='main-checks', name=job)
    if status: ET.SubElement(case, 'failure', message='Original Main command or evidence failed').text = '\n'.join(errors)
    ET.ElementTree(suite).write(directory / 'check.junit.xml', encoding='unicode', xml_declaration=True)
    STAGES.write(directory / 'final.json', {'exit_code': status, 'report_errors': errors,
        'original_sha256': {str(p.relative_to(directory)): STAGES.digest(p) for p in sorted(directory.rglob('*')) if p.is_file() and p.name != 'final.json'}})
    return status


if __name__ == '__main__': sys.exit(main())
