#!/usr/bin/env python3
"""Compare complete original Cannon Go reports on an identical source revision."""
import argparse
import importlib.util
import json
from pathlib import Path
_SPEC_E = importlib.util.spec_from_file_location('report_evidence', Path(__file__).with_name('report-evidence.py'))
E = importlib.util.module_from_spec(_SPEC_E); _SPEC_E.loader.exec_module(E)
import re
import sys
import tempfile

SPEC = importlib.util.spec_from_file_location('cannon_go', Path(__file__).resolve().parents[1] / 'runtime' / 'cannon-go.py')
G = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(G)
EMPTY = __import__('hashlib').sha256(b'').hexdigest()
REQUIRED = {'settings.json', 'selection.json', 'coverage.json', 'cpus.json', 'cpus.log', 'cpus.stage.json', 'packages.json', 'packages.log',
            'packages.stage.json', 'lint.log', 'lint.stage.json', 'list.json', 'list.log',
            'list.stage.json', 'tests.log', 'tests.stage.json', 'original.json', 'junit.xml',
            'native.json', 'native.metadata.json', 'inputs-after.json'}


def originals(directory, provider):
    final = G.read(directory / 'final.json'); hashes = final.get('original_sha256', {})
    if final.get('exit_code') != 0 or final.get('report_errors') != [] or not REQUIRED <= hashes.keys():
        raise ValueError('Failed or incomplete original Cannon report')
    empty = []
    if 'final.json' in hashes: raise ValueError('Self-sealed original report manifest')
    E.verify_files(directory, hashes, required=REQUIRED,
                   missing_empty=empty if provider == 'circleci' else None, label=None)
    actual = {str(p.relative_to(directory)) for p in directory.rglob('*') if p.is_file() and p.name != 'final.json'}
    if actual != hashes.keys(): raise ValueError('Unexpected unsealed Cannon report input')
    return hashes, empty


normalize = E.normalize


def verdict(directory, provider, sha):
    hashes, empty = originals(directory, provider); settings = G.read(directory / 'settings.json')
    if (settings['suite'] != 'cannon-go' or settings['source_sha'] != sha or settings['provider'] != provider
            or not settings['branch'] or not settings['input_sha256']):
        raise ValueError('Wrong Cannon source, provider, branch or workload')
    effective = settings['settings']
    if effective != G.settings(provider, effective['skip_slow_tests'], True, effective['parallel']):
        raise ValueError('Cannon benchmark must execute freshly with original settings')
    cpus = G.read(directory / 'cpus.json')
    if type(cpus) is not int or cpus != effective['parallel']:
        raise ValueError('Cannon concurrency differs from original nproc discovery')
    if settings['environment']['SKIP_SLOW_TESTS'] != str(effective['skip_slow_tests']).lower():
        raise ValueError('Cannon slow-test environment differs from recorded setting')
    if settings['go_environment']['GOFLAGS']:
        raise ValueError('Unaccounted Cannon Go flags override original selection')
    if provider == 'rwx' and (not settings['rwx_run_id'] or str(settings['rwx_task_attempt']) != '1'):
        raise ValueError('Missing native Cannon run identity or unexpected task retry')
    root = Path(settings['workspace_root']); selection = G.read(directory / 'selection.json')
    packages = G.packages((directory / 'packages.json').read_text(), root, directory / 'source')
    for row in packages:
        for name, digest in row['files'].items():
            if settings['input_sha256'].get(name) != digest:
                raise ValueError('Cannon package source differs from tracked revision inputs')
    tests = G.listing(directory / 'list.json', packages)
    if selection != {'packages': packages, 'initial_tests': tests,
                     'authority': 'go list ./... and go test -list ' + G.LIST_PATTERN}:
        raise ValueError('Cannon selection differs from complete original discovery')
    expected = {'cpus': ['nproc'], 'lint': ['just', 'lint'], 'packages': ['go', 'list', '-e', '-json', './...'],
                'list': ['go', 'test', '-json', *G.go_flags(effective, list_tests=True)]}
    # Reconstruct commands using the bound original absolute workspace and
    # report path. Collection directories can differ from execution directories.
    report = root / '.ci/cannon-go/run'
    expected['tests'] = [str(root / 'ops/scripts/gotestsum-split.sh'), '--format=testname',
        '--junitfile=' + str(report / 'junit.xml'), '--jsonfile=' + str(report / 'original.json'),
        '--', *G.go_flags(effective)]
    stages = {}
    for name, argv in expected.items():
        row = G.read(directory / (name + '.stage.json'))
        if row['exit_code'] != 0 or row['argv'] != argv or row['cwd'] != str(root / 'cannon') or row.get('stdin') != 'devnull':
            raise ValueError('Failed or changed original Cannon stage: ' + name)
        if row['log_sha256'] != hashes[name + '.log']: raise ValueError('Cannon stage log seal differs')
        stages[name] = normalize({k: row[k] for k in ('argv', 'cwd', 'exit_code')}, str(root))
    coverage = G.observed(directory, packages, tests)
    if coverage != G.read(directory / 'coverage.json') or coverage['package_failures'] or coverage['outcomes'].get('fail'):
        raise ValueError('Incomplete or unsuccessful original Cannon coverage')
    if G.read(directory / 'inputs-after.json') != settings['input_sha256']:
        raise ValueError('Cannon execution changed original tracked inputs')
    with tempfile.TemporaryDirectory() as tmp:
        native = Path(tmp) / 'native.json'; G.PROJECT.project(directory / 'original.json', native)
        for path in (native, native.with_suffix('.metadata.json')):
            if G.S.digest(path) != hashes[path.name]: raise ValueError('Native Cannon report differs from original projection')
    dependencies = {}
    if provider == 'rwx':
        for kind in ('go-modules', 'contracts'):
            record = G.read(directory / 'dependencies' / (kind + '.json'))
            if (record['kind'] != kind or record['commit_sha'] != sha or record['settings'] != G.ARTIFACTS.SETTINGS[kind]
                    or record['mise_sha256'] != settings['input_sha256']['mise.toml'] or not record['files']
                    or record['tool_versions']['go'] != settings['tool_versions']['go']):
                raise ValueError('Stale or mismatched Cannon dependency artifact')
            if kind == 'contracts' and record['tool_versions']['forge'] != settings['tool_versions']['forge']:
                raise ValueError('Cannon contract producer used a different Forge toolchain')
            dependencies[kind] = record
    return settings, selection, coverage, stages, dependencies, hashes, empty


def compare(root, sha):
    if not re.fullmatch('[0-9a-f]{40}', sha): raise ValueError('Expected full Cannon benchmark SHA')
    a = verdict(root / 'circle', 'circleci', sha); b = verdict(root / 'rwx', 'rwx', sha)
    settings_a, settings_b = a[0], b[0]
    for field in ('source_sha', 'branch', 'input_sha256', 'environment', 'go_environment', 'tool_versions'):
        if settings_a[field] != settings_b[field]: raise ValueError('Complete Cannon settings parity differs: ' + field)
    # Both providers honor the original nproc rule. Circle exposes the host's
    # 32 CPUs inside an 8-vCPU Docker job; RWX exposes the requested 8 CPUs.
    # Retain the actual values and argv, and compare every other setting.
    if ({k:v for k,v in settings_a['settings'].items() if k != 'parallel'} !=
            {k:v for k,v in settings_b['settings'].items() if k != 'parallel'}):
        raise ValueError('Complete Cannon effective workload settings differ')
    def commands(value):
        return {k: {**v, 'argv': ['-parallel=<nproc>' if re.fullmatch(r'-parallel=\d+', arg) else arg
                                 for arg in v['argv']]} for k,v in value.items()}
    if a[1] != b[1] or commands(a[3]) != commands(b[3]): raise ValueError('Complete Cannon discovery or invocation parity differs')
    differences = []
    if settings_a['settings']['parallel'] != settings_b['settings']['parallel']:
        differences.append({'setting': 'parallel', 'rule': 'nproc',
            'circle': settings_a['settings']['parallel'], 'rwx': settings_b['settings']['parallel'],
            'reason': 'Original CPU discovery differs by execution environment; requested workers remain 8 CPU / 16 GiB'})
    def cases(coverage):
        return [{k: c[k] for k in ('suite', 'name', 'outcome', 'skip_reason')} |
                {'attempts': [{k: h[k] for k in ('outcome', 'skip_reason')} for h in c['attempts']]}
                for c in sorted(coverage['cases'], key=lambda c: (c['suite'], c['name']))]
    original_a, original_b = cases(a[2]), cases(b[2])
    if original_a != original_b or a[2]['package_attempts'] != b[2]['package_attempts']:
        raise ValueError('Original Cannon outcomes, skips or retry histories differ')
    return {'source_sha': sha, 'verified_parity': True, 'packages': len(a[1]['packages']),
            'initial_tests': sum(map(len, a[1]['initial_tests'].values())), 'case_identities': len(original_a),
            'outcomes': a[2]['outcomes'], 'settings': {'circle': settings_a['settings'], 'rwx': settings_b['settings']},
            'settings_differences': differences, 'tool_versions': settings_a['tool_versions'],
            'selection': a[1], 'cases': original_a, 'package_attempts': a[2]['package_attempts'], 'commands': {'circle': a[3], 'rwx': b[3]},
            'native_dependencies': b[4], 'per_test_logs': {'circle': a[2]['per_test_logs'], 'rwx': b[2]['per_test_logs']},
            'original_sha256': {'circle': a[5], 'rwx': b[5]}, 'circle_manifest_declared_empty': a[6],
            'providers': {p: {'workspace_root': x[0]['workspace_root'], 'branch': x[0]['branch'],
                            'rwx_run_id': x[0]['rwx_run_id'], 'rwx_task_attempt': x[0]['rwx_task_attempt']}
                          for p, x in (('circle', a), ('rwx', b))}}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__); parser.add_argument('root', type=Path)
    parser.add_argument('--sha', required=True); parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    try:
        result = compare(args.root, args.sha); G.S.write(args.output, result)
        print(json.dumps({k: result[k] for k in ('source_sha', 'verified_parity', 'packages', 'initial_tests', 'case_identities', 'outcomes')}))
    except (OSError, ValueError, KeyError) as error:
        print('Cannon parity failed: ' + str(error), file=sys.stderr); sys.exit(1)
