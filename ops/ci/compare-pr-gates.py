#!/usr/bin/env python3
"""Compare complete original Circle and native RWX gate evidence on one SHA."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import subprocess
import tempfile
from urllib.parse import urlparse

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location('original_pr_gate', Path(__file__).with_name('pr-gate.py'))
G = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(G)
PASS = {'succeeded': True, 'failed': False, 'skipped': False}
SKIP = {'succeeded': False, 'failed': False, 'skipped': True}
TERMINAL = ['success', 'failed', 'canceled', 'unauthorized', 'skipped']


def require(condition, message):
    if not condition:
        raise ValueError(message)


def original(path, expected=None):
    final = G.originals(path)
    require(type(final['exit_code']) is int and type(final['tests']) is int,
            'Incorrectly typed gate verdict')
    if expected is not None:
        require(set(final['original_sha256']) == set(expected), 'Missing or extra gate originals')
    return final['original_sha256'] | {'final.json': G.S.digest(path / 'final.json')}


def source(path, sha):
    return subprocess.check_output(['git', 'show', sha + ':' + path], cwd=ROOT)


def source_inputs(sha):
    """Hash the complete committed tree, including links, without a checkout."""
    tree = subprocess.check_output(['git', 'ls-tree', '-rz', sha], cwd=ROOT)
    entries, blobs, result, seen = [], set(), {}, set()
    for entry in tree.split(b'\0'):
        if not entry:
            continue
        header, raw_name = entry.split(b'\t', 1)
        mode, kind, oid = header.decode().split()
        name = raw_name.decode()
        require(name not in seen, 'Duplicate source input')
        seen.add(name)
        if mode == '160000':
            require(kind == 'commit', 'Wrong gitlink input')
            result[name] = {'gitlink': oid}
        else:
            require(kind == 'blob' and mode in ('100644', '100755', '120000'), 'Wrong source input mode')
            entries.append((name, mode, oid))
            blobs.add(oid)
    # A file-backed stdin avoids pipe deadlocks while hashing the complete tree.
    contents = {}
    with tempfile.TemporaryFile() as requests:
        requests.write(('\n'.join(sorted(blobs)) + '\n').encode())
        requests.seek(0)
        with subprocess.Popen(['git', 'cat-file', '--batch'], cwd=ROOT, stdin=requests,
                              stdout=subprocess.PIPE) as process:
            for oid in sorted(blobs):
                header = process.stdout.readline().decode().split()
                require(len(header) == 3 and header[:2] == [oid, 'blob'], 'Missing source blob')
                size = int(header[2])
                data = process.stdout.read(size)
                require(len(data) == size and process.stdout.read(1) == b'\n', 'Truncated source blob')
                contents[oid] = data
            require(not process.stdout.read() and process.wait() == 0, 'Source discovery failed')
    for name, mode, oid in entries:
        result[name] = ({'symlink': contents[oid].decode()} if mode == '120000'
                        else hashlib.sha256(contents[oid]).hexdigest())
    require(result, 'Empty source discovery')
    return result


def yaml(path):
    return json.loads(subprocess.check_output(['yq', '-o=json', '.', str(path)], text=True),
                      object_pairs_hook=G.unique)


def named(rows, key):
    require(isinstance(rows, list) and rows, 'Empty gate task/job discovery')
    result = {row[key]: row for row in rows}
    require(len(result) == len(rows), 'Duplicate gate task/job identity')
    return result


def fresh(task):
    require(task['Status']['Execution'] == 'finished' and task['Status']['Result'] == 'succeeded'
            and task['Status']['FinishedSubStatus'] == 'executed', 'Gate prerequisite did not execute freshly')


def states(actual, expected):
    require(actual == expected and all(isinstance(row, dict) and set(row) == set(PASS)
            and all(type(value) is bool for value in row.values()) for row in actual.values()),
            'Missing, failed or incorrectly typed original gate states')


def compare(circle, native, status, run_path, github_path):
    circle, native, status = Path(circle), Path(native), Path(status)
    originals = {'circle': original(circle), 'native': original(native), 'status': original(status, ['settings.json', 'states.json'])}
    settings, published = G.read(native / 'settings.json'), G.read(status / 'settings.json')
    c = G.read(circle / 'settings.json')
    sha, gate = settings['source_sha'], settings['gate']
    require(re.fullmatch('[0-9a-f]{40}', sha) and c['source_sha'] == sha, 'Different gate source revisions')
    require(c['provider'] == 'circleci' and c['gate'] == gate and c['branch'] == settings['branch'],
            'Different gate provider, branch or selection')
    require(type(settings['selected']) is bool and settings['selected'], 'Unselected gate adds no coverage')
    require(settings['task_attempt'] == '1', 'Uninvestigated native gate retry')
    manifest = G.read(native / 'manifest.json')
    require(manifest['version'] in (2, 3) and manifest['repository'] == 'ethereum-optimism/optimism',
            'Wrong original gate manifest')
    require((native / 'manifest.json').read_bytes() == source('ops/ci/pr-gates.json', sha)
            == (circle / 'source-gates.json').read_bytes(), 'Stale or resealed gate authority')
    row = manifest['gates'][gate]
    names = [d['name'] for d in row['dependencies']]
    groups = sorted({d['group'] for d in row['dependencies']})
    require(names and len(names) == len(set(names)), 'Empty or duplicate required dependency')
    selection = row | {'requires': names, 'groups': groups}
    require(settings['selection'] == selection and published['selection'] == selection,
            'Changed original gate selection')
    identity = {k: settings[k] for k in ('source_sha', 'branch', 'input_sha256', 'native_run_id', 'task_attempt', 'gate')}
    require(all(published[k] == value for k, value in identity.items()), 'Foreign native final verdict')
    require(settings['input_sha256'] == source_inputs(sha), 'Stale, missing or corrupt committed gate inputs')
    require((circle / 'circle-config.yml').read_bytes() == source(row['circle_config'], sha),
            'Stale original Circle gate configuration')
    config = yaml(circle / 'circle-config.yml')
    authority = [item[gate] for item in config['workflows'][row['workflow']]['jobs'] if isinstance(item, dict) and gate in item]
    require(len(authority) == 1 and not authority[0].get('always-succeed', False)
            and authority[0]['requires'] == [{name: 'terminal'} for name in names], 'Changed authoritative Circle prerequisites')
    compiled_raw = G.read(circle / 'pipeline-config.json')
    require(compiled_raw['compiled'] == (circle / 'compiled.yml').read_text()
            and compiled_raw['source'] == (circle / 'source.yml').read_text(), 'Changed compiled Circle originals')
    compiled = yaml(circle / 'compiled.yml')
    gates = [item[gate] for item in compiled['workflows'][row['workflow']]['jobs'] if isinstance(item, dict) and gate in item]
    require(len(gates) == 1 and gates[0]['requires'] == names
            and gates[0]['upstream'] == {name: TERMINAL for name in names}, 'Changed compiled terminal dependency set')
    commands = [step['run']['command'] for step in compiled['jobs'][gate]['steps']
                if isinstance(step, dict) and isinstance(step.get('run'), dict)
                and step['run'].get('name') == 'Verify all required jobs passed']
    require(len(commands) == 1 and 'if [ "false" = "true" ]; then' in commands[0]
            and 'All required jobs passed.' in commands[0], 'Always-successful or omitted Circle gate verifier')
    pipeline, workflow = G.read(circle / 'pipeline.json'), G.read(circle / 'workflow.json')
    require(pipeline['id'] == c['pipeline_id'] and not pipeline['errors']
            and pipeline['vcs']['revision'] == sha and pipeline['vcs']['branch'] == settings['branch']
            and workflow['id'] == c['workflow_id'] and workflow['pipeline_id'] == c['pipeline_id']
            and workflow['name'] == c['workflow_name'] == row['workflow'], 'Foreign original Circle workflow')
    pages = sorted((circle / 'api-pages').glob('jobs-*.json'), key=lambda p: int(p.stem.split('-')[1]))
    require([p.name for p in pages] == ['jobs-' + str(i) + '.json' for i in range(len(pages))], 'Missing Circle job page')
    jobs = []
    for i, path in enumerate(pages):
        page = G.read(path)
        require(bool(page.get('next_page_token')) == (i != len(pages) - 1), 'Incomplete Circle job pagination')
        jobs.extend(page['items'])
    require(jobs == G.read(circle / 'workflow-jobs.json'), 'Changed original Circle job collection')
    by_id, by_name = named(jobs, 'id'), named(jobs, 'name')
    circle_gate = by_name[gate]
    require(circle_gate['status'] == 'success' and circle_gate['job_number'] == c['job_number'], 'Circle gate failed or never ran')
    dependencies = circle_gate['dependencies']
    require(len(dependencies) == len(set(dependencies)) == len(names)
            and {by_id[key]['name'] for key in dependencies} == set(names), 'Missing, duplicate or foreign Circle prerequisite')
    require(all(by_id[key]['status'] == 'success' for key in dependencies), 'Failed or skipped Circle prerequisite')
    job = G.read(circle / 'gate-job.json')
    require(job['status'] == job['outcome'] == 'success' and job['vcs_revision'] == sha
            and job['branch'] == settings['branch'] and job['build_num'] == c['job_number']
            and job['workflows']['workflow_id'] == c['workflow_id']
            and job['workflows']['job_id'] == circle_gate['id']
            and job['workflows']['job_name'] == gate and not job['retry_of']
            and all(job[key] is False for key in ('failed', 'timedout', 'canceled', 'infrastructure_fail')),
            'Failed, stale or retried original Circle verdict')
    steps = [step for step in G.read(circle / 'step-index.json') if step['step'] == 'Verify all required jobs passed']
    actual_steps = [step for step in job['steps'] if step['name'] == 'Verify all required jobs passed']
    require(len(steps) == len(actual_steps) == 1 and len(actual_steps[0]['actions']) == 1
            and actual_steps[0]['actions'][0]['status'] == steps[0]['status'] == 'success'
            and actual_steps[0]['actions'][0]['output_url'], 'Missing original executed Circle verifier')
    events = G.read(circle / steps[0]['log'])
    require(events and all(event.get('truncated') is False and isinstance(event['message'], str) for event in events),
            'Truncated or missing original Circle verdict log')
    lines = [line.strip() for line in ''.join(event['message'] for event in events).splitlines() if line.strip()]
    require(lines[0] == 'Checking required jobs for ' + gate + '...' and lines[-1] == 'All required jobs passed.'
            and sorted(lines[1:-1]) == sorted('ok ' + name + ': success' for name in names),
            'Incomplete, duplicate or bypassed original Circle verification')
    states(G.read(status / 'states.json'), {'aggregate': PASS, 'gate-failure': SKIP})
    group_states, verdict = G.read(native / 'states.json'), G.read(native / 'verdict.json')
    states(group_states, {group: PASS for group in groups})
    states(verdict['groups'], group_states)
    require(verdict['state'] == 'passed'
            and verdict['requires'] == names
            and verdict['dependencies'] == [d | {'outcome': 'passed'} for d in row['dependencies']],
            'Incomplete original native gate verdict')
    run = G.read(run_path)
    run_id = settings['native_run_id']
    require(re.fullmatch('[0-9a-f]{32}', run_id) and run['ID'] == run['RunID'] == run_id
            and run['CommitSha'] == sha and run['Branch'] == settings['branch']
            and run['Init'] == {'commit-sha': sha, 'branch': settings['branch'], 'tag': ''}
            and run['Trigger'] == 'github.push' and run['DefinitionPath'] == row['native_config']
            and run['TargetedTaskKeys'] is None, 'Foreign, manually targeted or unexecuted native gate run')
    engine = named(run['Tasks'], 'Key')
    observers = row.get('observer_tasks', {'aggregate': 'aggregate', 'failure': 'gate-failure', 'status': 'gate-status'})
    fresh(engine[observers['aggregate']])
    fresh(engine[observers['status']])
    require(engine[observers['failure']]['Status']['Execution'] == 'skipped', 'Unexpected native failure observer')
    expected = {'settings.json', 'manifest.json', 'states.json', 'verdict.json'}
    for group in groups:
        artifact = native / 'groups' / group
        original(artifact, ['settings.json', 'states.json'])
        expected |= {'groups/' + group + '/' + name for name in ('settings.json', 'states.json', 'final.json')}
        group_settings = G.read(artifact / 'settings.json')
        require(all(group_settings[k] == settings[k] for k in ('source_sha', 'branch', 'input_sha256', 'native_run_id', 'task_attempt'))
                and group_settings['selected'] is True and group_settings['group'] == group
                and group_settings['definition'] == manifest['groups'][group], 'Stale or foreign native group receipt')
        tasks = G.group_tasks(manifest, group)
        states(G.read(artifact / 'states.json'), {task: PASS for task in tasks})
        child = named(engine[manifest['groups'][group]['embedded_task']]['Subtasks'], 'Key')
        fresh(child[manifest['groups'][group]['receipt_task']])
        for task in tasks:
            fresh(child[task])
    original(native, expected)
    github = G.read(github_path)
    require(github['sha'] == sha, 'Stale GitHub gate observation')
    checks = named(github['checks'], 'name')
    native_check = checks['RWX: ' + row['check_name']]
    circle_check = checks['ci/circleci: ' + gate]
    link = urlparse(native_check['link'])
    require(native_check['state'] == circle_check['state'] == 'success'
            and link.scheme == 'https' and link.netloc == 'cloud.rwx.com'
            and link.path == '/optimism/runs/' + run_id + '/latest/' + observers['status']
            and circle_check['link'] == 'https://circleci.com/gh/ethereum-optimism/optimism/' + str(c['job_number']),
            'Missing, failed or foreign terminal GitHub gate check')
    return {'state': 'passed', 'source_sha': sha, 'branch': settings['branch'], 'gate': gate,
            'tests': 0, 'dependencies': names, 'circle': {'pipeline_id': c['pipeline_id'], 'workflow_id': c['workflow_id'],
            'job_number': c['job_number'], 'retry_of': job['retry_of'], 'retry_history': job['retries']},
            'native': {'run_id': run_id, 'aggregate_task': engine[observers['aggregate']]['ID'],
            'status_task': engine[observers['status']]['ID'], 'task_attempt': settings['task_attempt'],
            'workload_tasks': sum(len(G.group_tasks(manifest, group)) for group in groups)},
            'original_sha256': originals | {'run_state': G.S.digest(run_path), 'github_checks': G.S.digest(github_path)}}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('circle', 'native', 'status', 'run_state', 'github_checks', 'output'):
        parser.add_argument(name, type=Path)
    args = parser.parse_args()
    report = compare(args.circle, args.native, args.status, args.run_state, args.github_checks)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    G.S.write(args.output, report)
    print(json.dumps({key: report[key] for key in ('state', 'source_sha', 'gate', 'tests')}))
