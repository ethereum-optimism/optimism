#!/usr/bin/env python3
"""Verify complete original reporting executions and same-SHA hosted parity."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import re


def helper(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


F = helper('flaky-report')
G = helper('compare-pr-gates')
check, read = F.check, F.read


def report(directory, authority=True):
    final = read(directory / 'final.json')
    check(final['state'] == 'passed' and type(final['exit_code']) is int and final['exit_code'] == 0
          and type(final['tests']) is int and final['tests'] == 0 and type(final['reports']) is int
          and final['reports'] == 1 and final['errors'] == [] and final['sha256'] == F.files(directory),
          'Missing, failed or corrupt original report')
    settings = read(directory / 'settings.json')
    sha, branch, provider = settings['source_sha'], settings['branch'], settings['provider']
    check(re.fullmatch('[0-9a-f]{40}', sha) and branch == 'codex/rwx-ci-pilot' and provider in ('circleci', 'rwx')
          and settings['scope'] == 'project-wide branch-agnostic Circle Insights'
          and type(settings['authenticated']) is bool
          and (provider != 'rwx' or settings['authenticated'] is False), 'Wrong original reporting identity')
    check(settings['api_url'] == 'https://circleci.com/api/v2/insights/gh/ethereum-optimism/optimism/flaky-tests?branch=codex%2Frwx-ci-pilot'
          and settings['jq'] == 'jq-1.7.1' and settings['curl'].startswith('curl '), 'Changed original API or tools')
    if authority:
        check(settings['input_sha256'] == {name: hashlib.sha256(G.source(name, sha)).hexdigest() for name in F.INPUTS},
              'Stale or uncommitted reporting inputs')
    execution = read(directory / 'execution.json')
    cwd, argv = execution['cwd'], execution['argv']
    # Downloaded evidence has a different local prefix; the original process
    # cwd and exact argv identify its actual provider workspace and output.
    expected_output = cwd + '/.ci/flaky-report/run/reports'
    wanted = ['bash', cwd + '/' + F.SCRIPT, '--branch', branch, '--org', 'ethereum-optimism',
              '--repo', 'optimism', '--output-dir', expected_output]
    if settings['authenticated']: wanted += ['--token', '[redacted]']
    check(argv == wanted and cwd.startswith('/') and type(execution['exit_code']) is int and execution['exit_code'] == 0
          and execution['signals'] == [] and execution['timed_out'] is False
          and type(execution['started_at']) in (int, float) and execution['started_at'] > 0
          and type(execution['elapsed_seconds']) in (int, float) and execution['elapsed_seconds'] > 0
          and execution['stdout_sha256'] == F.digest(directory / 'stdout.log')
          and execution['stderr_sha256'] == F.digest(directory / 'stderr.log'), 'Missing real original reporting execution')
    value = F.validate_report(directory / 'reports', branch)
    check(read(directory / 'coverage.json') == {key: value[key] for key in ('source_rows', 'acceptance_rows', 'scope')}
          and value['source_rows'] > 0 and value['acceptance_rows'] > 0, 'Empty or incomplete executed workload')
    stdout = (directory / 'stdout.log').read_text()
    ordered = list(reversed(sorted(value['filtered']['flaky_tests'], key=lambda row: row['times_flaked'])))
    top = '\n'.join(str(row['times_flaked']) + 'x: ' + row['test_name'] for row in ordered[:10])
    check(stdout.endswith(top + '\n'), 'Missing original top-ten text report')
    attempts = []
    for path in sorted((directory / 'reports').glob('api-attempt-*.http-status.txt')):
        stem = path.name.removesuffix('.http-status.txt')
        attempts.append({'number': int(stem.split('-')[2]), 'status': path.read_text().strip(),
                         'exit_code': int((path.parent / (stem + '.exit-code.txt')).read_text()),
                         'body_sha256': F.digest(path.parent / (stem + '.json')) if (path.parent / (stem + '.json')).exists() else None,
                         'stderr_sha256': F.digest(path.parent / (stem + '.stderr.log'))})
    return {'settings': settings, 'original': value['original'], 'filtered': value['filtered'],
            'csv': (directory / 'reports/flaky_tests.csv').read_bytes(),
            'html': (directory / 'reports/flaky_tests.html').read_bytes(),
            'stdout': stdout.replace(expected_output, '$REPORTS'), 'attempts': attempts,
            'rows': {key: value[key] for key in ('source_rows', 'acceptance_rows')}}


def equal(circle, native):
    for key in ('source_sha', 'branch', 'input_sha256', 'api_url', 'scope', 'jq'):
        check(circle['settings'][key] == native['settings'][key], 'Original reporting identity differs: ' + key)
    # No row, date, order, count, skip or retry data are normalized away.
    for key in ('original', 'filtered', 'csv', 'html', 'stdout', 'rows'):
        check(circle[key] == native[key], 'Complete original flaky-report parity differs: ' + key)
    check(all(row['status'] == '200' and row['exit_code'] == 0 for row in
              (circle['attempts'][-1], native['attempts'][-1])), 'Failed final original API attempt')
    return {'rows': circle['rows'], 'attempts': {'circleci': circle['attempts'], 'rwx': native['attempts']}}


def hosted(circle, native, run_path, github_path):
    settings = read(native / 'settings.json')
    sha, branch = settings['source_sha'], settings['branch']
    run = read(run_path)
    check(run['ID'] == run['RunID'] == settings['native_run_id'] and run['CommitSha'] == sha
          and run['Branch'] == branch and run['Trigger'] == 'github.push' and run['DefinitionPath'] == '.rwx/flaky-report.yml'
          and run['TargetedTaskKeys'] is None and settings['native_task_attempt'] == '1',
          'Foreign, targeted or retried native reporter')
    tasks = G.named(run['Tasks'], 'Key')
    G.fresh(tasks['helper-tests']); G.fresh(tasks['report'])
    identity = read(circle / 'settings.json')
    check(identity == {'provider': 'circleci', 'source_sha': sha, 'branch': branch,
          'pipeline_id': identity['pipeline_id'], 'workflow_id': identity['workflow_id'], 'job_number': identity['job_number']},
          'Foreign Circle collection identity')
    envelope = read(circle / 'final.json')
    check(envelope['source_sha'] == sha and envelope['sha256'] == F.files(circle), 'Missing or corrupt Circle collection')
    pipeline, workflow, job = (read(circle / name) for name in ('pipeline.json', 'workflow.json', 'job.json'))
    check(pipeline['id'] == identity['pipeline_id'] and not pipeline['errors'] and pipeline['vcs']['revision'] == sha
          and pipeline['vcs']['branch'] == branch and workflow['id'] == identity['workflow_id']
          and workflow['pipeline_id'] == identity['pipeline_id'] and workflow['name'] == 'flaky-report-replay'
          and workflow['status'] == 'success', 'Foreign or failed Circle reporting workflow')
    pages = sorted((circle / 'api-pages').glob('jobs-*.json'), key=lambda p: int(p.stem.split('-')[1]))
    check(pages and [p.name for p in pages] == ['jobs-' + str(i) + '.json' for i in range(len(pages))], 'Missing Circle job page')
    jobs = []
    for i, path in enumerate(pages):
        page = read(path); jobs.extend(page['items'])
        check(bool(page.get('next_page_token')) == (i < len(pages) - 1), 'Incomplete Circle pagination')
    check(len(jobs) == 1 and jobs[0]['name'] == 'generate-flaky-tests-report' and jobs[0]['status'] == 'success'
          and jobs[0]['job_number'] == identity['job_number'] and job['workflows']['job_id'] == jobs[0]['id']
          and job['workflows']['job_name'] == jobs[0]['name'] and job['build_num'] == identity['job_number']
          and job['vcs_revision'] == sha and job['branch'] == branch and job['workflows']['workflow_id'] == identity['workflow_id']
          and job['status'] == job['outcome'] == 'success' and not job['retry_of']
          and all(job[key] is False for key in ('failed', 'timedout', 'canceled', 'infrastructure_fail')),
          'Failed, skipped or retried original Circle report')
    config = read(circle / 'pipeline-config.json')
    check(config['compiled'] == (circle / 'compiled.yml').read_text() and config['source'] == (circle / 'source.yml').read_text(),
          'Changed Circle compiled originals')
    compiled = G.yaml(circle / 'compiled.yml')
    selected = compiled['workflows']['flaky-report-replay']['jobs']
    check(len(selected) == 1 and set(selected[0]) == {'generate-flaky-tests-report'}, 'Wrong original reporting workload selection')
    commands = [step['run']['command'] for step in compiled['jobs']['generate-flaky-tests-report']['steps']
                if isinstance(step, dict) and isinstance(step.get('run'), dict)]
    check(any(command.strip() == 'python3 ops/ci/flaky-report.py run .ci/flaky-report/run' for command in commands),
          'Missing original executed reporting wrapper')
    authority = G.source('.circleci/continue/main.yml', sha).decode()
    check('generate-flaky-report:' in authority and 'python3 ops/ci/flaky-report.py run .ci/flaky-report/run' in authority,
          'Uncommitted Circle report adapter')
    index = read(circle / 'step-index.json')
    actions = [(i, j) for i, step in enumerate(job['steps']) for j, action in enumerate(step['actions']) if action.get('output_url')]
    check([(row['step'], row['action']) for row in index] == actions, 'Missing complete original Circle logs')
    for row in index:
        events = read(circle / row['path'])
        check(all(event.get('truncated') is False and isinstance(event['message'], str) for event in events), 'Truncated Circle originals')
    executed = [row for row in index if row['name'] == 'Generate flaky acceptance tests report']
    check(len(executed) == 1 and executed[0]['status'] == 'success', 'Missing actual Circle report execution')
    github = read(github_path)
    native_check = G.named(github['checks'], 'name')['RWX: optimism-flaky-report-shadow']
    check(github['sha'] == sha and native_check['state'] == 'success' and '/runs/' + run['ID'] in native_check['link'],
          'Missing successful same-SHA optional reporting check')
    return {'circle_pipeline': pipeline['number'], 'circle_job': job['build_num'], 'native_run': run['ID']}


def compare(circle, native, run_path, github_path):
    ids = hosted(circle, native, run_path, github_path)
    c, n = report(circle / 'report'), report(native)
    check(c['settings']['provider'] == 'circleci' and n['settings']['provider'] == 'rwx', 'Wrong original providers')
    parity = equal(c, n)
    return {'verified_parity': True, 'source_sha': c['settings']['source_sha'], **ids, **parity,
            'original_manifest_sha256': {'circleci': F.digest(circle / 'report/final.json'), 'rwx': F.digest(native / 'final.json')},
            'complete_original_files': {'circleci': len(F.files(circle / 'report')) + 1, 'rwx': len(F.files(native)) + 1},
            'normalizations': ['provider workspace prefix in report output paths'],
            'provider_differences': ['authenticated Circle API request and anonymous native request to the same public project']}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ('circle', 'native', 'run', 'github', 'output'): parser.add_argument('--' + key, type=Path, required=True)
    args = parser.parse_args()
    value = compare(args.circle, args.native, args.run, args.github)
    F.write(args.output, value)
    print(json.dumps(value))
