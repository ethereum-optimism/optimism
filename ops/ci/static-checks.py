#!/usr/bin/env python3
"""Execute complete original ShellCheck/Semgrep selections and seal their reports."""
import argparse
import hashlib
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
S = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(S)
ROOT = Path(__file__).resolve().parents[2]
ORB = 'ops/ci/shellcheck-orb-3.2.0.bash'
ORB_SHA256 = 'd8f51c02b4a6ce3ebc32024dfaf8a16ec4a4445331a7832e8137d6afc00fb0a4'
VERSIONS = {'shell-check': '0.9.0', 'semgrep-test': '1.178.0', 'semgrep-scan-local': '1.178.0'}
COMMANDS = {'shell-check': ['bash', '-eo', 'pipefail', ORB],
            'semgrep-test': ['semgrep', 'scan', '--test', '--config', '.semgrep/rules/', '--json', '.semgrep/tests/'],
            'semgrep-scan-local': ['semgrep', 'scan', '--timeout=100', '--config', '.semgrep/rules/', '--error', '--json', '--verbose', '.']}
SHELL_ENV = {'SC_PARAM_DIR': '.', 'SC_PARAM_EXCLUDE': '', 'SC_PARAM_EXTERNAL_SOURCES': 'false',
             'SC_PARAM_FORMAT': 'tty', 'SC_PARAM_IGNORE_DIRS': './packages/contracts-bedrock/lib\n./docs/public-docs\n',
             'SC_PARAM_OUTPUT': 'shellcheck.log', 'SC_PARAM_PATTERN': '', 'SC_PARAM_SEVERITY': 'style', 'SC_PARAM_SHELL': ''}
SHELL_FIND = ['find', '.', '!', '-name', '*\n*', '-name', '*.sh', '-type', 'f',
              '!', '-path', './packages/contracts-bedrock/lib/*.sh', '!', '-path', './docs/public-docs/*.sh']


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def inputs():
    rows = subprocess.check_output(['git', 'ls-files', '--stage', '-z'], cwd=ROOT, text=True).split('\0')
    result = {}
    for row in rows:
        if not row: continue
        header, name = row.split('\t', 1); mode, revision, stage = header.split()
        if stage != '0' or name in result: raise ValueError('Unmerged or duplicate static-check source input')
        p = ROOT / name
        if mode == '160000': result[name] = {'gitlink': revision}
        elif p.is_symlink(): result[name] = {'symlink': os.readlink(p)}
        elif p.is_file(): result[name] = digest(p)
        else: raise ValueError('Missing static-check source: ' + name)
    return result


def normalized(value, root):
    # Original reports remain untouched. Only a bound absolute workspace prefix
    # is replaced, including Semgrep's path-keyed fixture results.
    if isinstance(value, str): return value.replace(root + '/', '<repo>/')
    if isinstance(value, list): return [normalized(v, root) for v in value]
    if isinstance(value, dict): return {normalized(k, root): normalized(v, root) for k, v in value.items()}
    return value


def shell_files(text):
    names = text.splitlines()
    if not names or len(names) != len(set(names)) or any(not n.startswith('./') or not n.endswith('.sh') or '..' in Path(n).parts for n in names):
        raise ValueError('Missing, duplicate or unsafe original ShellCheck discovery')
    return sorted(names)


def machine_report(job, data, root):
    if job == 'semgrep-test':
        required = {'config_missing_tests', 'config_missing_fixtests', 'config_with_errors', 'results', 'fixtest_results'}
        if set(data) != required or data['config_with_errors'] or not data['results']:
            raise ValueError('Incomplete or failed original Semgrep rule-test report')
        identities = set()
        for config, row in data['results'].items():
            if set(row) != {'checks'} or not row['checks']: raise ValueError('Empty original Semgrep test configuration')
            for name, case in row['checks'].items():
                identity = (config, name)
                if identity in identities or case['passed'] is not True or case['errors'] or not case['matches']:
                    raise ValueError('Duplicate, missing or failed original Semgrep rule verdict')
                identities.add(identity)
                for filename, match in case['matches'].items():
                    if match['expected_lines'] != match['reported_lines']:
                        raise ValueError('Original Semgrep annotations and matches differ')
        return normalized(data, root)
    fields = {'version', 'results', 'errors', 'paths', 'time', 'engine_requested', 'skipped_rules', 'profiling_results'}
    if set(data) != fields or data.get('version') != VERSIONS[job] or not isinstance(data.get('results'), list) or data['results'] or \
       not isinstance(data.get('paths', {}).get('scanned'), list) or not isinstance(data['paths'].get('skipped'), list) or \
       not isinstance(data.get('errors'), list) or not isinstance(data.get('skipped_rules'), list):
        raise ValueError('Incomplete, wrong-version or failing original Semgrep scan')
    scanned = data['paths']['scanned']
    if len(scanned) != len(set(scanned)): raise ValueError('Duplicate original Semgrep scan assignment')
    # The original CLI permits parser warnings. Preserve every diagnostic and
    # skipped target/rule rather than claiming those files were fully parsed.
    result = normalized({k: data[k] for k in ('version', 'results', 'errors', 'paths', 'engine_requested', 'skipped_rules')}, root)
    result['paths']['scanned'] = sorted(result['paths']['scanned'])
    result['paths']['skipped'] = sorted(result['paths']['skipped'], key=lambda r: (r['path'], r['reason']))
    return result


def baseline(branch):
    if branch == 'develop': return None
    commit = subprocess.check_output(['git', 'rev-parse', '--verify', 'develop^{commit}'], cwd=ROOT, text=True).strip()
    merge_base = subprocess.check_output(['git', 'merge-base', 'HEAD', commit], cwd=ROOT, text=True).strip()
    if not all(re.fullmatch('[0-9a-f]{40}', value) for value in (commit, merge_base)):
        raise ValueError('Missing or invalid original Semgrep develop baseline')
    fields = subprocess.check_output(['git', 'diff', '--name-status', '--no-renames', '-z', merge_base, 'HEAD', '--'],
                                     cwd=ROOT, text=True).split('\0')[:-1]
    if len(fields) % 2: raise ValueError('Truncated original Semgrep baseline change discovery')
    changed = {}
    for status, name in zip(fields[::2], fields[1::2]):
        if status not in ('A', 'M', 'D', 'T') or name in changed or Path(name).is_absolute() or '..' in Path(name).parts:
            raise ValueError('Invalid or duplicate original Semgrep baseline change')
        changed[name] = status
    return {'ref': 'develop', 'commit': commit, 'merge_base': merge_base, 'changed_files': changed}


def scanned_inputs(coverage, source):
    selected = {}
    for name in coverage['paths']['scanned']:
        relative = name.removeprefix('<repo>/').removeprefix('./')
        if relative not in source or not isinstance(source[relative], str):
            raise ValueError('Original Semgrep scanned an unbound source input: ' + name)
        selected[name] = source[relative]
    return selected


def semgrep_environment(branch, sha):
    metadata = {'SEMGREP_REPO_URL': 'https://github.com/ethereum-optimism/optimism',
                'SEMGREP_BRANCH': branch, 'SEMGREP_COMMIT': sha,
                'SEMGREP_REPO_NAME': 'ethereum-optimism/optimism', 'TEMPORARY_BASELINE_REF': 'develop'}
    if branch != 'develop': metadata['SEMGREP_BASELINE_REF'] = 'develop'
    return metadata


def configure(job, branch):
    if job == 'shell-check':
        for k in list(os.environ):
            if k.startswith('SC_PARAM_'): del os.environ[k]
        os.environ.update(SHELL_ENV)
        return SHELL_ENV
    # These are the original Circle adapter's local-scan metadata, without an
    # app token or an additional cloud policy. Both commands use local rules.
    for key in list(os.environ):
        if key.startswith('SEMGREP_'): del os.environ[key]
    metadata = semgrep_environment(branch, subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip())
    os.environ.update(metadata)
    return metadata


def selection(directory, job, before):
    if job == 'shell-check':
        if digest(ROOT / ORB) != ORB_SHA256: raise ValueError('Original ShellCheck orb command changed')
        if S.stage(directory, 'discovery', SHELL_FIND, cwd=str(ROOT), stdin=subprocess.DEVNULL):
            raise ValueError('Original ShellCheck find failed')
        files = shell_files((directory / 'discovery.log').read_text())
        # The original find skips an ignore directory only when it exists.
        for n in ('packages/contracts-bedrock/lib', 'docs/public-docs'):
            if not (ROOT / n).exists(): raise ValueError('Missing original ShellCheck ignore directory')
        return {'files': files, 'sha256': {n: digest(ROOT / n) for n in files},
                'orb': 'circleci/shellcheck@3.2.0', 'orb_command_sha256': digest(ROOT / ORB)}
    return {'rules': {n: v for n, v in before.items() if n.startswith('.semgrep/rules/')},
            'fixtures': {n: v for n, v in before.items() if n.startswith('.semgrep/tests/')},
            'ignore': {n: v for n, v in before.items() if Path(n).name in ('.gitignore', '.semgrepignore')}}


def execute(job):
    os.chdir(ROOT); d = ROOT / '.ci/static-checks' / job
    shutil.rmtree(d, ignore_errors=True); d.mkdir(parents=True)
    status, errors, cases, settings = 1, [], [], None
    try:
        sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip()
        if not re.fullmatch('[0-9a-f]{40}', sha) or sha != (os.environ.get('CI_COMMIT_SHA') or os.environ.get('CIRCLE_SHA1') or sha):
            raise ValueError('Wrong static-check source SHA')
        for argv in (['git', 'diff', '--exit-code', '--quiet'], ['git', 'diff', '--cached', '--exit-code', '--quiet']):
            if subprocess.call(argv): raise ValueError('Static-check source differs from its Git revision')
        branch = os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH')
        if not branch: raise ValueError('Missing tested static-check branch')
        before = inputs(); environment = configure(job, branch)
        tool = 'shellcheck' if job == 'shell-check' else 'semgrep'
        version = subprocess.check_output([tool, '--version'], text=True).strip()
        parsed = re.findall(r'^version: (\S+)$', version, re.M) if job == 'shell-check' else [version]
        if parsed != [VERSIONS[job]]: raise ValueError('Static-check tool differs from the recorded Circle image version')
        settings = {'source_sha': sha, 'branch': branch, 'job': job, 'provider': os.environ.get('CI_CHECK_PROVIDER', 'circleci'),
                    'workspace_root': str(ROOT), 'tool_version': version, 'environment': environment, 'input_sha256': before,
                    'baseline': baseline(branch) if job != 'shell-check' else None,
                    'command': COMMANDS[job], 'rwx_run_id': os.environ.get('RWX_RUN_ID'),
                    'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}
        S.write(d / 'settings.json', settings); selected = selection(d, job, before); S.write(d / 'selection.json', selected)
        if job == 'shell-check': (ROOT / 'shellcheck.log').unlink(missing_ok=True)
        status = S.stage(d, 'check', COMMANDS[job], stdout_json=job != 'shell-check', cwd=str(ROOT), stdin=subprocess.DEVNULL)
        if job == 'shell-check':
            shutil.copy2(ROOT / 'shellcheck.log', d / 'original.shellcheck.log')
            log = re.sub(r'\x1b\[[0-9;]*m', '', (d / 'check.log').read_text())
            actual = shell_files('\n'.join(s for s in log.splitlines() if s.startswith('./') and s.endswith('.sh')))
            if actual != selected['files']: raise ValueError('Original orb executed a different ShellCheck selection')
            if not status and ((d / 'original.shellcheck.log').stat().st_size or 'No ShellCheck Errors Found' not in log):
                raise ValueError('ShellCheck success lacks complete original verdict evidence')
            coverage = {'files': actual, 'outcome': 'pass' if status == 0 else 'fail'}
            cases = [{'name': n, 'outcome': 'pass'} for n in actual] if not status else [{'name': job, 'outcome': 'fail'}]
        elif not status:
            raw = json.loads((d / 'check.json').read_text()); coverage = machine_report(job, raw, str(ROOT))
            if job == 'semgrep-test':
                cases = [{'name': config + ':' + name, 'outcome': 'pass'} for config, row in coverage['results'].items() for name in row['checks']]
                cases += [{'name': n, 'outcome': 'skip', 'reason': 'Original Semgrep config_missing_tests'} for n in coverage['config_missing_tests']]
            else:
                S.write(d / 'scanned-source-sha256.json', scanned_inputs(coverage, before))
                cases = [{'name': job, 'outcome': 'pass'}]
        else: coverage = {'outcome': 'fail'}
        S.write(d / 'coverage.json', coverage)
        after = inputs(); S.write(d / 'inputs-after.json', after)
        if before != after or settings['source_sha'] != subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip():
            raise ValueError('Static verdict changed its original source inputs')
        if job != 'shell-check' and settings['baseline'] != baseline(branch):
            raise ValueError('Original Semgrep baseline changed during execution')
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as error:
        errors.append(str(error)); print(error, file=sys.stderr); status = status or 1
    if not cases or errors: cases = [{'name': job, 'outcome': 'fail'}]
    suite = ET.Element('testsuite', name='static-checks/' + job, tests=str(len(cases)),
                       failures=str(sum(c['outcome'] == 'fail' for c in cases)), skipped=str(sum(c['outcome'] == 'skip' for c in cases)))
    for c in cases:
        node = ET.SubElement(suite, 'testcase', classname=job, name=c['name'])
        if c['outcome'] != 'pass': ET.SubElement(node, 'skipped' if c['outcome'] == 'skip' else 'failure', message=c.get('reason', 'Original static check failed'))
    ET.ElementTree(suite).write(d / 'derived.junit.xml', encoding='utf-8', xml_declaration=True)
    S.write(d / 'final.json', {'exit_code': status, 'report_errors': errors,
            'original_sha256': {str(p.relative_to(d)): digest(p) for p in d.rglob('*') if p.is_file() and p.name != 'final.json'}})
    return status


if __name__ == '__main__':
    p = argparse.ArgumentParser(description=__doc__); p.add_argument('job', choices=COMMANDS); sys.exit(execute(p.parse_args().job))
