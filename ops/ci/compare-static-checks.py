#!/usr/bin/env python3
"""Compare full original static selections, diagnostics, rule tests and findings."""
import argparse
import importlib.util
import json
from pathlib import Path
import re
import xml.etree.ElementTree as ET


def helper(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module); return module


C = helper('static-checks'); ORIGINALS = helper('compare-rust-e2e')


def report(directory, job, sha, provider, empty):
    required = {'settings.json', 'selection.json', 'check.stage.json', 'check.log', 'coverage.json', 'inputs-after.json', 'derived.junit.xml'}
    if job == 'shell-check': required |= {'discovery.log', 'discovery.stage.json', 'original.shellcheck.log'}
    else: required.add('check.json')
    if job == 'semgrep-scan-local': required.add('scanned-source-sha256.json')
    hashes = ORIGINALS.originals(directory, required, empty, provider + '/' + job)
    if {str(p.relative_to(directory)) for p in directory.rglob('*') if p.is_file() and p.name != 'final.json'} != set(hashes):
        raise ValueError('Unsealed or missing complete static originals')
    s = json.loads((directory / 'settings.json').read_text()); root = s['workspace_root']
    if (s['source_sha'], s['job'], s['provider'], s['command']) != (sha, job, 'circleci' if provider == 'circle' else 'rwx', C.COMMANDS[job]):
        raise ValueError('Wrong static source, job, provider or original command')
    if provider == 'rwx' and (not s['rwx_run_id'] or str(s['rwx_task_attempt']) != '1'):
        raise ValueError('Reused static verdict or uninvestigated retry')
    if not s['input_sha256'] or json.loads((directory / 'inputs-after.json').read_text()) != s['input_sha256']:
        raise ValueError('Static verdict changed source inputs')
    version = re.findall(r'^version: (\S+)$', s['tool_version'], re.M) if job == 'shell-check' else [s['tool_version']]
    if version != [C.VERSIONS[job]]: raise ValueError('Wrong original static tool version')
    stages = {p.name.removesuffix('.stage.json') for p in directory.glob('*.stage.json')}
    commands = {'check': C.COMMANDS[job]}
    if job == 'shell-check': commands['discovery'] = C.SHELL_FIND
    if stages != set(commands): raise ValueError('Missing original static command or unexplained retry')
    for name, argv in commands.items():
        row = json.loads((directory / (name + '.stage.json')).read_text())
        if (row['argv'], row['cwd'], row['exit_code'], row['stdin']) != (argv, root, 0, 'devnull') or \
           C.digest(directory / (name + '.log')) != row['log_sha256']:
            raise ValueError('Wrong or failed original static invocation/output')
    selected = json.loads((directory / 'selection.json').read_text()); recorded = json.loads((directory / 'coverage.json').read_text())
    if job == 'shell-check':
        if s['baseline'] is not None or s['environment'] != C.SHELL_ENV or selected['orb_command_sha256'] != C.ORB_SHA256 or \
           selected['orb'] != 'circleci/shellcheck@3.2.0' or s['input_sha256'].get(C.ORB) != C.ORB_SHA256:
            raise ValueError('Original ShellCheck orb parameters or command changed')
        files = C.shell_files((directory / 'discovery.log').read_text())
        log = re.sub(r'\x1b\[[0-9;]*m', '', (directory / 'check.log').read_text())
        actual = C.shell_files('\n'.join(s for s in log.splitlines() if s.startswith('./') and s.endswith('.sh')))
        if files != actual or files != selected['files'] or set(selected['sha256']) != set(files) or \
           any(selected['sha256'][n] != s['input_sha256'].get(n.removeprefix('./')) for n in files):
            raise ValueError('Missing, extra or changed original ShellCheck file assignment')
        coverage = {'files': files, 'outcome': 'pass'}
        if (directory / 'original.shellcheck.log').stat().st_size or 'No ShellCheck Errors Found' not in log:
            raise ValueError('Incomplete or failing original ShellCheck verdict')
        cases = {(n, None) for n in files}
    else:
        if s['environment'] != C.semgrep_environment(s['branch'], sha):
            raise ValueError('Original Semgrep scan environment changed')
        baseline = s['baseline']
        if (s['branch'] == 'develop' and baseline is not None) or (s['branch'] != 'develop' and
           (not isinstance(baseline, dict) or set(baseline) != {'ref', 'commit', 'merge_base', 'changed_files'} or baseline.get('ref') != 'develop' or
            not all(re.fullmatch('[0-9a-f]{40}', str(baseline.get(k))) for k in ('commit', 'merge_base')) or
            not isinstance(baseline.get('changed_files'), dict))):
            raise ValueError('Missing original Semgrep baseline provenance')
        if baseline is not None and any(status not in ('A','M','D','T') or Path(n).is_absolute() or '..' in Path(n).parts or
              ((n in s['input_sha256']) != (status != 'D')) for n,status in baseline['changed_files'].items()):
            raise ValueError('Invalid original Semgrep baseline change assignment')
        wanted = {'rules': {n:v for n,v in s['input_sha256'].items() if n.startswith('.semgrep/rules/')},
                  'fixtures': {n:v for n,v in s['input_sha256'].items() if n.startswith('.semgrep/tests/')},
                  'ignore': {n:v for n,v in s['input_sha256'].items() if Path(n).name in ('.gitignore', '.semgrepignore')}}
        if not wanted['rules'] or not wanted['fixtures'] or selected != wanted:
            raise ValueError('Incomplete original Semgrep rule/fixture/ignore selection')
        coverage = C.machine_report(job, json.loads((directory / 'check.json').read_text()), root)
        if job == 'semgrep-test':
            cases = {(config + ':' + name, None) for config,row in coverage['results'].items() for name in row['checks']}
            cases |= {(n, 'Original Semgrep config_missing_tests') for n in coverage['config_missing_tests']}
        else: cases = {(job, None)}
        if job == 'semgrep-scan-local' and json.loads((directory / 'scanned-source-sha256.json').read_text()) != \
           C.scanned_inputs(coverage, s['input_sha256']):
            raise ValueError('Original Semgrep scanned different source bytes')
    if recorded != coverage: raise ValueError('Static accounting differs from full original reports')
    suite = ET.parse(directory / 'derived.junit.xml').getroot(); found = set()
    for case in suite.findall('testcase'):
        skip = case.find('skipped'); reason = skip.get('message') if skip is not None else None
        key = (case.attrib['name'], reason)
        if case.attrib != {'name':key[0], 'classname':job} or key in found or \
           case.find('failure') is not None or case.find('error') is not None:
            raise ValueError('Duplicate or failing derived static case')
        found.add(key)
    if found != cases or suite.attrib != {'name':'static-checks/' + job,'tests':str(len(cases)), 'failures':'0',
                                          'skipped':str(sum(reason is not None for _,reason in cases))}:
        raise ValueError('Derived static JUnit differs from complete originals')
    return {'settings': s, 'selection': selected, 'coverage': coverage, 'original_sha256': hashes}


def compare(directories, job, sha):
    if not re.fullmatch('[0-9a-f]{40}', sha): raise ValueError('Expected full static benchmark SHA')
    empty = []; data = {p:report(d, job, sha, p, empty) for p,d in directories.items()}; a,b = data['circle'], data['rwx']
    for key in ('source_sha', 'branch', 'job', 'tool_version', 'environment', 'baseline', 'input_sha256', 'command'):
        if a['settings'][key] != b['settings'][key]: raise ValueError('Static settings differ at ' + key)
    for key in ('selection', 'coverage'):
        if a[key] != b[key]: raise ValueError('Complete original static parity differs at ' + key)
    return {'source_sha':sha, 'job':job, 'verified_parity':True, 'selection':a['selection'], 'coverage':a['coverage'],
            'settings':{p:d['settings'] for p,d in data.items()}, 'original_sha256':{p:d['original_sha256'] for p,d in data.items()},
            'manifest_declared_empty_logs':empty}


if __name__ == '__main__':
    p = argparse.ArgumentParser(description=__doc__); p.add_argument('--circle',type=Path,required=True); p.add_argument('--rwx',type=Path,required=True)
    p.add_argument('--job',choices=C.COMMANDS,required=True); p.add_argument('--sha',required=True); p.add_argument('--output',type=Path,required=True)
    a=p.parse_args(); result=compare({'circle':a.circle,'rwx':a.rwx},a.job,a.sha); C.S.write(a.output,result)
    print(json.dumps({'source_sha':a.sha,'job':a.job,'verified_parity':True}))
