#!/usr/bin/env python3
"""Compare complete original module preparation and contract fast checks."""
import argparse
import importlib.util
import json
from pathlib import Path
import re
import xml.etree.ElementTree as ET


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module); return module


CHECKS = helper('pr-checks')
REPORT = helper('ci-report')
SUBMODULES = helper('git-submodule-report')


def report(directory, job, sha, provider, empty):
    required = {'settings.json', 'coverage.json', 'checks.junit.xml'}
    required |= {'discovery.json', 'discovery.stage.json', 'verify.stage.json', 'verify.log', 'download-0.stage.json', 'download-0.log'} if job == 'go-modules' else {
        'checks.log', 'checks.stage.json', 'selection.json', 'submodules.txt', 'foundry-config.json', 'foundry-config.stage.json', 'fetch-target.stage.json'}
    final = REPORT.read(directory / 'final.json')
    if final['exit_code'] != 0 or final['report_errors']:
        raise ValueError('Failed or incomplete original PR check report')
    hashes = REPORT.verify_files(directory, final['original_sha256'], required=required,
                                 missing_empty=empty if provider == 'circle' else None, label=provider + '/' + job)
    settings = json.loads((directory / 'settings.json').read_text()); coverage = json.loads((directory / 'coverage.json').read_text())
    if settings['source_sha'] != sha or settings['job'] != job or settings['provider'] != {'circle': 'circleci', 'rwx': 'rwx'}[provider]:
        raise ValueError('PR check source, job or provider differs')
    if provider == 'rwx' and job == 'contracts-fast' and (not settings['rwx_run_id'] or str(settings['rwx_task_attempt']) != '1'):
        raise ValueError('Missing fresh native check identity or uninvestigated task retry')
    if job == 'contracts-fast' and (not settings.get('target_branch') or any(
        not re.fullmatch('[0-9a-f]{40}', settings.get(field, '')) for field in ('target_sha', 'merge_base_sha'))):
        raise ValueError('Invalid bound contract target history')
    stages = {}
    for path in directory.glob('*.stage.json'):
        row = json.loads(path.read_text()); name = path.name.removesuffix('.stage.json')
        stages[name] = REPORT.normalize({k: row[k] for k in ('argv', 'cwd', 'exit_code')}, settings['workspace_root'])
        if row['exit_code'] != 0 and not name.startswith('download-'): raise ValueError('Failed original PR check stage')
    if job == 'go-modules':
        manifest = CHECKS.modules((directory / 'discovery.json').read_text())
        if manifest != coverage['modules'] or not coverage['verified']: raise ValueError('Module coverage differs from complete original discovery')
        downloads = sorted(n for n in stages if n.startswith('download-'))
        if downloads != ['download-' + str(i) for i in range(coverage['download_attempts'])] or not 1 <= len(downloads) <= 5:
            raise ValueError('Incomplete original download retry history')
        if stages[downloads[-1]]['exit_code'] != 0 or any(stages[n]['exit_code'] == 0 for n in downloads[:-1]):
            raise ValueError('Invalid original download retry history')
        expected = set(downloads) | {'discovery', 'verify'}
        if coverage['selected'] != len(manifest): raise ValueError('Incomplete module count')
        cases = {job}
    else:
        expected = {'checks', 'foundry-config', 'fetch-target'}
        configuration = json.loads((directory / 'selection.json').read_text())
        if CHECKS.check_verdicts((directory / 'checks.log').read_text(), configuration) != coverage:
            raise ValueError('Check coverage differs from original command output')
        cases = set(CHECKS.checks(configuration))
    if stages.keys() != expected: raise ValueError('Missing or extra original PR check stages')
    for name, row in stages.items():
        command = ['go', 'mod', 'download'] if name.startswith('download-') else {
            'verify': ['go', 'mod', 'verify'], 'discovery': ['go', 'list', '-m', '-json', 'all'],
            'checks': ['just', 'check-fast', '-verbose'], 'foundry-config': ['forge', 'config', '--json'],
            'fetch-target': ['git', 'fetch', '--no-tags', 'origin', '+refs/heads/' + settings.get('target_branch', '') +
                             ':refs/remotes/origin/' + settings.get('target_branch', '')]}[name]
        cwd = '<repo>/packages/contracts-bedrock' if job == 'contracts-fast' and name != 'fetch-target' else '<repo>'
        if row['argv'] != command or row['cwd'] != cwd: raise ValueError('Unexpected original PR check command or directory')
    junit = list(ET.parse(directory / 'checks.junit.xml').iter('testcase'))
    if len(junit) != len(cases) or {c.attrib['name'] for c in junit} != cases or any(
        c.find('failure') is not None or c.find('error') is not None or c.find('skipped') is not None for c in junit):
        raise ValueError('Original PR check JUnit differs from complete coverage')
    return settings, coverage, stages, hashes


def compare(directories, job, sha):
    if not re.fullmatch('[0-9a-f]{40}', sha): raise ValueError('Expected the full PR check source SHA')
    empty, data = [], {}
    for provider, directory in directories.items(): data[provider] = report(directory, job, sha, provider, empty)
    a, b = data['circle'], data['rwx']
    settings_fields = ('source_sha', 'job', 'branch', 'go', 'just', 'input_sha256') + (
        ('forge', 'semgrep', 'target_branch', 'target_sha', 'merge_base_sha') if job == 'contracts-fast' else ())
    for field in settings_fields:
        if a[0][field] != b[0][field]: raise ValueError('PR check binding differs at ' + field)
    if a[2] != b[2]: raise ValueError('PR check original commands or retry history differs')
    if job == 'go-modules':
        if a[1] != b[1]: raise ValueError('Complete original Go module graph or retries differ')
    else:
        for provider, directory in directories.items():
            config = json.loads((directory / 'foundry-config.json').read_text())
            data[provider] += (REPORT.normalize(config, data[provider][0]['workspace_root']),
                              SUBMODULES.revisions((directory / 'submodules.txt').read_text()))
        a, b = data['circle'], data['rwx']
        if a[4:] != b[4:]: raise ValueError('PR check Foundry configuration or submodule revisions differ')
        def outcomes(coverage):
            return {k: v for k, v in coverage.items() if k != 'checks'} | {
                'checks': [{**r, 'attempts': [{k: v for k, v in t.items() if k != 'elapsed_seconds'} for t in r['attempts']]} for r in coverage['checks']]}
        if outcomes(a[1]) != outcomes(b[1]): raise ValueError('Full original check outcomes or retry histories differ')
    return {'source_sha': sha, 'job': job, 'verified_parity': True, 'coverage': a[1], 'commands': a[2],
            'original_sha256': {p: data[p][3] for p in data}, 'manifest_declared_empty_logs': empty,
            'original_submodule_status': {p: (d / 'submodules.txt').read_text() for p,d in directories.items()} if job == 'contracts-fast' else None,
            'submodule_revisions': a[5] if job == 'contracts-fast' else None}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--circle', required=True, type=Path); parser.add_argument('--rwx', required=True, type=Path)
    parser.add_argument('--job', required=True, choices=('go-modules', 'contracts-fast')); parser.add_argument('--sha', required=True)
    parser.add_argument('--output', required=True, type=Path); args = parser.parse_args()
    result = compare({'circle': args.circle, 'rwx': args.rwx}, args.job, args.sha)
    args.output.write_text(json.dumps(result, indent=2, sort_keys=True) + '\n')
    print(json.dumps({'source_sha': args.sha, 'job': args.job, 'verified_parity': True, 'selected': result['coverage']['selected']}))
