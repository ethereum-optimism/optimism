#!/usr/bin/env python3
"""Retain and verify the complete original Circle Insights reporting workload."""
import argparse
import csv
import datetime
import hashlib
import html as html_format
from html.parser import HTMLParser
import json
import math
import os
from pathlib import Path
import re
import signal
import subprocess
import time
import urllib.parse

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = 'op-acceptance-tests/scripts/generate-flaky-tests-report.sh'
INPUTS = (SCRIPT, 'ops/ci/flaky-report.py', 'mise.toml')
PREFIX = 'github.com/ethereum-optimism/optimism/op-acceptance-tests/tests'
COLUMNS = ['times_flaked', 'test_name', 'classname', 'job_name', 'workflow_name', 'job_number',
           'pipeline_number', 'job_url', 'first_flaked_at', 'last_flaked_at']


def check(value, message):
    if not value:
        raise ValueError(message)


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + '\n')


def read(path):
    def pairs(rows):
        value = {}
        for key, item in rows:
            check(key not in value, 'Duplicate original JSON key')
            value[key] = item
        return value
    return json.loads(Path(path).read_bytes(), object_pairs_hook=pairs,
                      parse_constant=lambda _: (_ for _ in ()).throw(ValueError('Non-finite original JSON')))


def validate_api(path):
    value = read(path)
    check(isinstance(value, dict) and set(value) == {'flaky_tests', 'total_flaky_tests'}
          and isinstance(value['flaky_tests'], list) and type(value['total_flaky_tests']) is int
          and value['total_flaky_tests'] == len(value['flaky_tests']), 'Incomplete original Circle Insights response')
    identities = set()
    for row in value['flaky_tests']:
        check(isinstance(row, dict), 'Invalid original flaky-test row')
        for key in ('classname', 'test_name', 'job_name', 'workflow_name', 'workflow_id', 'workflow_created_at', 'file', 'source'):
            check(isinstance(row.get(key), str), 'Missing original flaky-test field: ' + key)
        for key in ('times_flaked', 'job_number', 'pipeline_number'):
            check(type(row.get(key)) is int and row[key] > 0, 'Invalid original flaky-test count: ' + key)
        check(type(row.get('time_wasted')) in (int, float) and math.isfinite(row['time_wasted']) and row['time_wasted'] >= 0
              and re.fullmatch('[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}', row['workflow_id']),
              'Invalid original flaky-test provenance')
        when = datetime.datetime.fromisoformat(row['workflow_created_at'].replace('Z', '+00:00'))
        check(when.tzinfo is not None, 'Missing original observation timezone')
        identity = tuple(row[key] for key in ('workflow_id', 'job_number', 'classname', 'test_name', 'source', 'file'))
        check(identity not in identities, 'Duplicate original flaky-test identity')
        identities.add(identity)
    return value


class Table(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.rows, self.row, self.cell = [], None, None
        self.links, self.link = [], None

    def handle_starttag(self, tag, attrs):
        if tag == 'tr': self.row = []; self.links = []
        if tag == 'td': self.cell = ''
        if tag == 'a' and self.cell is not None: self.links.append(dict(attrs).get('href'))

    def handle_data(self, data):
        if self.cell is not None: self.cell += data

    def handle_endtag(self, tag):
        if tag == 'td' and self.row is not None:
            self.row.append(self.cell); self.cell = None
        if tag == 'tr' and self.row:
            self.rows.append((self.row, self.links)); self.row = None


def url(row):
    return ('https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/' + str(row['pipeline_number'])
            + '/workflows/' + row['workflow_id'] + '/jobs/' + str(row['job_number']))


def validate_report(directory, branch):
    attempts = sorted(directory.glob('api-attempt-*.http-status.txt'), key=lambda p: int(p.name.split('-')[2].split('.')[0]))
    check(0 < len(attempts) <= 6 and [p.name for p in attempts]
          == ['api-attempt-' + str(i) + '.http-status.txt' for i in range(1, len(attempts) + 1)],
          'Missing or extra original API attempts')
    expected_files = set()
    for i, path in enumerate(attempts, 1):
        status = path.read_text()
        base = directory / ('api-attempt-' + str(i))
        code = base.with_suffix('.exit-code.txt').read_text()
        expected_files.update(base.with_suffix(suffix).name for suffix in ('.http-status.txt', '.exit-code.txt', '.stderr.log'))
        if status != '000\n' or base.with_suffix('.json').exists(): expected_files.add(base.with_suffix('.json').name)
        check(re.fullmatch(r'\d{3}\n', status) and re.fullmatch(r'\d+\n', code)
              and base.with_suffix('.stderr.log').is_file(), 'Incomplete original API attempt')
        if i == len(attempts):
            check(status == '200\n' and code == '0\n'
                  and base.with_suffix('.json').read_bytes() == (directory / 'flaky_tests.original.json').read_bytes(),
                  'Missing successful original API snapshot')
        else:
            check(status not in ('200\n', '400\n', '401\n', '403\n', '404\n')
                  and (status == '000\n' or base.with_suffix('.json').is_file()), 'Overwritten or invalid original failure evidence')
    check({p.name for p in directory.glob('api-attempt-*')} == expected_files, 'Missing or extra original API attempt files')
    original = validate_api(directory / 'flaky_tests.original.json')
    filtered = original | {'flaky_tests': [row for row in original['flaky_tests'] if row['classname'].startswith(PREFIX)]}
    check(read(directory / 'flaky_tests.json') == read(directory / 'flaky_tests.filtered.json') == filtered,
          'Omitted, extra or changed acceptance-test report rows')
    check((directory / 'http-status.txt').read_text() == '200\n', 'Missing successful original HTTP status')
    ordered = list(reversed(sorted(filtered['flaky_tests'], key=lambda row: row['times_flaked'])))
    with (directory / 'flaky_tests.csv').open(newline='') as file:
        csv_rows = list(csv.reader(file))
    check(csv_rows[0] == COLUMNS and len(csv_rows) == len(ordered) + 1, 'Incomplete original CSV report')
    for cells, row in zip(csv_rows[1:], ordered):
        # The original jq recipe JSON-quotes strings before CSV encoding.
        expected = [str(row['times_flaked']), row['test_name'], row['classname'], row['job_name'], row['workflow_name'],
                    str(row['job_number']), str(row['pipeline_number']), url(row), row['workflow_created_at'], row['workflow_created_at']]
        check(len(cells) == len(COLUMNS), 'Malformed CSV report row')
        decoded = [json.loads(cell) if i in (1, 2, 3, 4, 7, 8, 9) else cell for i, cell in enumerate(cells)]
        check(decoded == expected, 'CSV differs from complete original API response')
    html = (directory / 'flaky_tests.html').read_text()
    parser = Table(); parser.feed(html); parser.close()
    branches = re.findall(r'<h3>Branch: (.*?)</h3>', html)
    check(len(parser.rows) == len(ordered) and 'CircleCI\'s flaky-test API is project-wide and branch agnostic.' in html
          and len(branches) == 1 and '<' not in branches[0] and html_format.unescape(branches[0]) == branch,
          'Incomplete HTML report or incorrect branch scope')
    for (cells, links), row in zip(parser.rows, ordered):
        check(cells == [str(row['times_flaked']), row['test_name'], row['classname'], row['job_name'], row['workflow_name'],
              str(row['job_number']), str(row['pipeline_number']), 'View Job', row['workflow_created_at'], row['workflow_created_at']]
              and links == [url(row)], 'HTML differs from complete original API response')
    return {'source_rows': len(original['flaky_tests']), 'acceptance_rows': len(ordered), 'original': original,
            'filtered': filtered, 'branch': branch, 'scope': 'project-wide branch-agnostic Circle Insights'}


def files(directory):
    value = {}
    for path in sorted(directory.rglob('*')):
        check(not path.is_symlink(), 'Linked flaky-report evidence')
        if path.is_file() and path != directory / 'final.json': value[str(path.relative_to(directory))] = digest(path)
    return value


def run(directory):
    check(not directory.exists(), 'Flaky-report verdict already exists')
    directory.mkdir(parents=True)
    sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    check(re.fullmatch('[0-9a-f]{40}', sha) and os.environ.get('CI_COMMIT_SHA', os.environ.get('CIRCLE_SHA1', sha)) == sha,
          'Wrong flaky-report source revision')
    branch = os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH')
    check(branch, 'Missing original reporting branch')
    inputs = {}
    for name in INPUTS:
        raw = subprocess.check_output(['git', 'show', sha + ':' + name], cwd=ROOT)
        check((ROOT / name).read_bytes() == raw, 'Uncommitted report implementation')
        inputs[name] = hashlib.sha256(raw).hexdigest()
    provider = os.environ.get('CI_CHECK_PROVIDER', 'circleci')
    check(provider in ('circleci', 'rwx'), 'Unknown flaky-report provider')
    token = os.environ.get('CIRCLE_API_TOKEN', '') if provider == 'circleci' else ''
    settings = {'source_sha': sha, 'branch': branch, 'provider': provider, 'input_sha256': inputs,
                'authenticated': bool(token), 'api_url': 'https://circleci.com/api/v2/insights/gh/ethereum-optimism/optimism/flaky-tests?'
                    + urllib.parse.urlencode({'branch': branch}),
                'native_run_id': os.environ.get('RWX_RUN_ID'), 'native_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER'),
                'scope': 'project-wide branch-agnostic Circle Insights',
                'jq': subprocess.check_output(['jq', '--version'], text=True).strip(),
                'curl': subprocess.check_output(['curl', '--version'], text=True).splitlines()[0]}
    write(directory / 'settings.json', settings)
    report = directory / 'reports'
    argv = ['bash', str(ROOT / SCRIPT), '--branch', branch, '--org', 'ethereum-optimism', '--repo', 'optimism', '--output-dir', str(report)]
    safe_argv = argv.copy()
    if token: argv += ['--token', token]; safe_argv += ['--token', '[redacted]']
    status, errors, signals = 1, [], []
    started = time.time()
    with (directory / 'stdout.log').open('wb') as stdout, (directory / 'stderr.log').open('wb') as stderr:
        process = subprocess.Popen(argv, cwd=ROOT, stdout=stdout, stderr=stderr, start_new_session=True)
        def stop(signum, _):
            signals.append(signum)
            try: os.killpg(process.pid, signum)
            except ProcessLookupError: pass
        handlers = {sig: signal.signal(sig, stop) for sig in (signal.SIGTERM, signal.SIGINT)}
        timed_out = False
        try:
            try: status = process.wait(timeout=900)
            except subprocess.TimeoutExpired:
                timed_out = True; stop(signal.SIGTERM, None)
                try: status = process.wait(timeout=5)
                except subprocess.TimeoutExpired: os.killpg(process.pid, signal.SIGKILL); status = process.wait()
            check(status == 0 and not timed_out and not signals, 'Original flaky-report command failed or was cancelled')
            value = validate_report(report, branch)
            write(directory / 'coverage.json', {key: value[key] for key in ('source_rows', 'acceptance_rows', 'scope')})
            text = (directory / 'stdout.log').read_text()
            ordered = list(reversed(sorted(value['filtered']['flaky_tests'], key=lambda row: row['times_flaked'])))
            top = '\n'.join(str(row['times_flaked']) + 'x: ' + row['test_name'] for row in ordered[:10])
            check(text.endswith(top + '\n' if top else '==========================================\n'), 'Missing original text report')
        except BaseException as error:
            errors.append(str(error) or type(error).__name__)
            raise
        finally:
            for sig, handler in handlers.items(): signal.signal(sig, handler)
            write(directory / 'execution.json', {'argv': safe_argv, 'cwd': str(ROOT), 'started_at': started,
                  'elapsed_seconds': time.time() - started, 'exit_code': status, 'signals': signals, 'timed_out': timed_out,
                  'stdout_sha256': digest(directory / 'stdout.log'), 'stderr_sha256': digest(directory / 'stderr.log')})
            write(directory / 'final.json', {'state': 'passed' if status == 0 and not errors else 'failed',
                  'exit_code': status or 1 if errors else status, 'tests': 0, 'reports': 1 if status == 0 and not errors else 0,
                  'errors': errors, 'sha256': files(directory)})


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    modes = parser.add_subparsers(dest='mode', required=True)
    modes.add_parser('validate-api').add_argument('path', type=Path)
    modes.add_parser('run').add_argument('directory', type=Path)
    args = parser.parse_args()
    if args.mode == 'validate-api': validate_api(args.path)
    else: run(args.directory.resolve())
