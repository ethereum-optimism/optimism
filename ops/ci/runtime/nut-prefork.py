#!/usr/bin/env python3
"""Regenerate every selected NUT pre-fork state with the original Just loop."""
import argparse
import copy
from collections import Counter
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import xml.etree.ElementTree as ET

import importlib.util


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


G = helper('cannon-go')
MAIN = helper('main-checks')
S = G.S
ROOT = Path(__file__).resolve().parents[3]
PACKAGE_DIR = 'rust/kona/tests/proofs'
PACKAGE = 'github.com/ethereum-optimism/optimism/' + PACKAGE_DIR
PATTERN = 'TestGenerateForkState'
FLAGS = ['-count=1', '-run', PATTERN, './' + PACKAGE_DIR + '/']


def command(*argv): return subprocess.check_output(argv, cwd=ROOT, text=True).strip()


def select_states(root):
    paths = sorted((root / 'op-core/nuts/state').glob('*_state.json'))
    if not paths: raise ValueError('Empty complete pre-fork state selection')
    states = {}
    for path in paths:
        fork = path.name.removesuffix('_state.json')
        if not re.fullmatch('[a-z][a-z0-9-]*', fork) or path.is_symlink():
            raise ValueError('Unsafe pre-fork state input')
        # Reject invalid/empty state inputs before a costly runtime build.
        value = G.read(path)
        if not isinstance(value, dict) or not value: raise ValueError('Invalid pre-fork state input')
        states[fork] = {'path': str(path.relative_to(root)), 'sha256': S.digest(path)}
    forks = [name for name in states if name != 'jovian']
    if not forks: raise ValueError('Empty executable pre-fork state selection')
    return {'states': states, 'forks': forks,
            'excluded': [{'fork': 'jovian', 'reason': 'Original Just recipe excludes jovian'}] if 'jovian' in states else [],
            'authority': 'just _check-nut-prefork-states', 'package': PACKAGE, 'go_flags': FLAGS}


def package_selection(text, workspace, source_root=None):
    rows = MAIN.PR.objects(text)
    if len(rows) != 1: raise ValueError('Missing or duplicate pre-fork test package')
    row = rows[0]; directory = Path(row.get('Dir', '')).resolve()
    if (row.get('Error') or row.get('DepsErrors') or row.get('ImportPath') != PACKAGE
            or directory != Path(workspace).resolve() / PACKAGE_DIR):
        raise ValueError('Failed or foreign pre-fork package discovery')
    source_root = Path(workspace) if source_root is None else Path(source_root)
    files = {}
    for field in G.GO_FIELDS:
        names = row.get(field, [])
        if not isinstance(names, list) or len(set(names)) != len(names): raise ValueError('Duplicate pre-fork source files')
        for name in names:
            relative = Path(name)
            if relative.is_absolute() or '..' in relative.parts: raise ValueError('Unsafe pre-fork source path')
            path = str(Path(PACKAGE_DIR) / relative); files[path] = S.digest(source_root / path)
    if not files: raise ValueError('Empty complete pre-fork source discovery')
    return [{'package': PACKAGE, 'directory': PACKAGE_DIR, 'files': files,
             'go_files': {field: row.get(field, []) for field in G.GO_FIELDS}}]


def test_command(directory):
    return [str(ROOT / 'ops/scripts/gotestsum-split.sh'), '--format=testname',
            '--junitfile=' + str(directory / 'junit.xml'), '--jsonfile=' + str(directory / 'original.json'), '--', *FLAGS]


def run_fork(fork):
    directory = Path(os.environ['NUT_PREFORK_REPORT_ROOT']).resolve()
    if (directory != ROOT / '.ci/nut-prefork/run' or not re.fullmatch('[a-z][a-z0-9-]*', fork)
            or os.environ.get('OP_E2E_GEN_PREFORK_STATE') != fork):
        raise ValueError('Invalid original pre-fork invocation or report destination')
    directory = directory / 'forks' / fork
    directory.mkdir(parents=True, exist_ok=False)
    status = S.stage(directory, 'tests', test_command(directory), cwd=str(ROOT), stdin=subprocess.DEVNULL)
    if (directory / 'original.json').exists(): G.PROJECT.project(directory / 'original.json', directory / 'native.json')
    return status


def native_junit(directory, forks):
    """Group actual original cases by fork; never synthesize test verdicts."""
    result = ET.Element('testsuites'); origins = {}
    for fork in forks:
        source = directory / 'forks' / fork / 'junit.xml'
        if not source.exists(): continue  # Cancellation stays partial.
        root = ET.parse(source).getroot(); suites = [root] if root.tag == 'testsuite' else list(root)
        origins[fork] = S.digest(source)
        for suite in suites:
            suite = copy.deepcopy(suite)
            suite.set('name', fork + '/' + suite.get('name', ''))
            for case in suite.iter('testcase'): case.set('classname', fork + '/' + case.get('classname', ''))
            result.append(suite)
    ET.ElementTree(result).write(directory / 'native.junit.xml', encoding='unicode', xml_declaration=True)
    S.write(directory / 'native.metadata.json', {'original_junit_sha256': origins,
            'rule': 'Original case names/outcomes/output retained; suite and classname prefixed with selected fork'})


def verify_bundle(directory, artifact, settings):
    identity = MAIN.superchain()
    if artifact is not None:
        final = G.read(artifact / 'final.json')
        expected = {'bundle.json', 'coverage.json', 'superchain.log', 'superchain.stage.json'}
        if final.get('exit_code') != 0 or final.get('report_errors') or set(final['original_sha256']) != expected:
            raise ValueError('Failed or incomplete pre-fork superchain producer')
        for name, digest in final['original_sha256'].items():
            if S.digest(artifact / name) != digest: raise ValueError('Corrupt pre-fork superchain producer')
        if G.read(artifact / 'bundle.json') != identity: raise ValueError('Stale or mismatched pre-fork superchain bundle')
        target = directory / 'producer'; target.mkdir()
        for name in expected: shutil.copyfile(artifact / name, target / name)
        S.write(target / 'manifest.json', final)
    if identity['source_sha'] != settings['source_sha']: raise ValueError('Wrong pre-fork bundle revision')
    target = directory / 'dependencies'; target.mkdir(exist_ok=True)
    shutil.copyfile(ROOT / MAIN.BUNDLE, target / 'superchain-configs.zip')
    shutil.copyfile(ROOT / (MAIN.BUNDLE + '.sha256'), target / 'superchain-configs.zip.sha256')
    return identity


def execute(provider, module_artifact=None, contract_artifact=None, bundle_artifact=None):
    os.chdir(ROOT); directory = ROOT / '.ci/nut-prefork/run'
    shutil.rmtree(directory, ignore_errors=True); directory.mkdir(parents=True)
    status, errors, selection = 1, [], None
    try:
        sha = command('git', 'rev-parse', 'HEAD')
        if not re.fullmatch('[0-9a-f]{40}', sha) or sha != (os.environ.get('CI_COMMIT_SHA') or os.environ.get('CIRCLE_SHA1') or sha):
            raise ValueError('Wrong pre-fork source revision')
        for argv in (['git', 'diff', '--quiet'], ['git', 'diff', '--cached', '--quiet']):
            if subprocess.call(argv): raise ValueError('Pre-fork source differs from its Git revision')
        before = G.inputs()
        settings = {'suite': 'nut-prefork', 'source_sha': sha, 'provider': provider, 'workspace_root': str(ROOT),
            'branch': os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH'), 'input_sha256': before,
            'environment': {name: os.environ.get(name) for name in ('CI', 'GOMAXPROCS', 'FOUNDRY_PROFILE')},
            'go_environment': json.loads(command('go', 'env', '-json', 'GOOS', 'GOARCH', 'GOAMD64', 'CGO_ENABLED', 'GOFLAGS', 'GOEXPERIMENT', 'GOTOOLCHAIN', 'GO111MODULE')),
            'tool_versions': {name: command(*argv) for name, argv in [('go', ['go', 'version']), ('just', ['just', '--version']),
                ('gotestsum', ['gotestsum', '--version']), ('forge', ['forge', '--version'])]},
            'rwx_run_id': os.environ.get('RWX_RUN_ID'), 'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}
        if not settings['branch'] or settings['go_environment']['GOFLAGS']: raise ValueError('Missing pre-fork branch or overridden Go flags')
        S.write(directory / 'settings.json', settings)
        if provider == 'rwx' and any(p is None for p in (module_artifact, contract_artifact, bundle_artifact)):
            raise ValueError('RWX pre-fork requires verified modules, contracts and superchain')
        for kind, artifact in [('go-modules', module_artifact), ('contracts', contract_artifact)]:
            if artifact is None: continue
            G.ARTIFACTS.restore(kind, artifact)
            target = directory / 'dependencies'; target.mkdir(exist_ok=True)
            shutil.copyfile(ROOT / 'tmp/testlogs/dependencies' / (kind + '.json'), target / (kind + '.json'))
            record = G.read(target / (kind + '.json'))
            if any(settings['tool_versions'].get(k) != v for k, v in record['tool_versions'].items()):
                raise ValueError('Pre-fork producer toolchain differs from runtime')
        selected = select_states(ROOT)
        selected['superchain'] = verify_bundle(directory, bundle_artifact, settings)
        for name, argv in [('packages', ['go', 'list', '-e', '-json', './' + PACKAGE_DIR + '/']),
                           ('list', ['go', 'test', '-json', '-count=1', '-list', PATTERN, './' + PACKAGE_DIR + '/'])]:
            status = S.stage(directory, name, argv, stdout_json=True, cwd=str(ROOT), stdin=subprocess.DEVNULL)
            if status: raise ValueError('Original pre-fork discovery failed: ' + name)
        packages = package_selection((directory / 'packages.json').read_text(), ROOT)
        selected['packages'] = packages; selected['initial_tests'] = G.listing(directory / 'list.json', packages)
        names = {n for row in packages for n in row['files']} | {r['path'] for r in selected['states'].values()}
        names |= {'justfile', 'ops/ci/runtime/nut-prefork-test.sh'}
        for name in sorted(names):
            target = directory / 'source' / name; target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(ROOT / name, target)
        S.write(directory / 'selection.json', selected); selection = selected
        os.environ['NUT_PREFORK_REPORT_ROOT'] = str(directory)
        status = S.stage(directory, 'check', ['just', '_check-nut-prefork-states'], cwd=str(ROOT), stdin=subprocess.DEVNULL)
        actual = sorted(p.name for p in (directory / 'forks').iterdir()) if (directory / 'forks').exists() else []
        if actual != selected['forks']: raise ValueError('Missing, duplicate or extra original fork execution')
        coverage = {fork: G.observed(directory / 'forks' / fork, packages, selected['initial_tests']) for fork in actual}
        S.write(directory / 'coverage.json', {'forks': coverage, 'attempts_per_fork': 1,
            'outcomes': dict(Counter(c['outcome'] for body in coverage.values() for c in body['cases']))})
        if any(v['package_failures'] or v['outcomes'].get('fail') or v['outcomes'].get('skip') for v in coverage.values()):
            raise ValueError('Pre-fork generation failed or skipped an authoritative initial test')
        generated = select_states(ROOT)
        S.write(directory / 'generated.json', generated)
        if generated != select_states(directory / 'source'): raise ValueError('Generated pre-fork states differ from committed bytes')
        after = G.inputs(); S.write(directory / 'inputs-after.json', after)
        if after != before or command('git', 'rev-parse', 'HEAD') != sha: raise ValueError('Pre-fork generation changed tracked source inputs')
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as error:
        errors.append(str(error)); print(error, file=sys.stderr); status = status or 1
    if selection is not None:
        try: native_junit(directory, selection['forks'])
        except (OSError, ET.ParseError) as error: errors.append(str(error)); status = status or 1
    S.write(directory / 'final.json', {'exit_code': status, 'report_errors': errors,
        'original_sha256': {str(p.relative_to(directory)): S.digest(p) for p in directory.rglob('*') if p.is_file() and p.name != 'final.json'}})
    return status


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--provider', choices=('circleci', 'rwx'))
    parser.add_argument('--test-fork')
    for name in ('module', 'contract', 'bundle'): parser.add_argument('--' + name + '-artifact', type=Path)
    args = parser.parse_args()
    if args.test_fork:
        if args.provider: parser.error('A fork command does not run the full adapter')
        sys.exit(run_fork(args.test_fork))
    if not args.provider: parser.error('A provider is required')
    sys.exit(execute(args.provider, args.module_artifact, args.contract_artifact, args.bundle_artifact))
