#!/usr/bin/env python3
"""Regenerate every selected NUT pre-fork state with the original Just loop."""
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys

import importlib.util


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


G = helper('cannon-go')
MAIN = helper('main-checks')
S = G.S
ROOT = Path(__file__).resolve().parents[3]
# Circle's command. It regenerates each selected state, then runs git diff --exit-code on every state.
CHECK = ['just', '_check-nut-prefork-states']


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
            'authority': ' '.join(CHECK)}


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
    status, errors = 1, []
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
                ('forge', ['forge', '--version'])]},
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
        # Keep the recipe and the committed state bytes; the recipe rewrites the states in place.
        for name in sorted({'justfile', *(row['path'] for row in selected['states'].values())}):
            target = directory / 'source' / name; target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(ROOT / name, target)
        S.write(directory / 'selection.json', selected)
        # The recipe's exit status is the verdict. A passing run that changes a tracked input still fails.
        status = S.stage(directory, 'check', CHECK, cwd=str(ROOT), stdin=subprocess.DEVNULL)
        if status: raise ValueError('Original pre-fork check failed')
        generated = select_states(ROOT)
        S.write(directory / 'generated.json', generated)
        if generated != select_states(directory / 'source'): raise ValueError('Generated pre-fork states differ from committed bytes')
        after = G.inputs(); S.write(directory / 'inputs-after.json', after)
        if after != before or command('git', 'rev-parse', 'HEAD') != sha: raise ValueError('Pre-fork generation changed tracked source inputs')
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as error:
        errors.append(str(error)); print(error, file=sys.stderr); status = status or 1
    S.write(directory / 'final.json', {'exit_code': status, 'report_errors': errors,
        'original_sha256': {str(p.relative_to(directory)): S.digest(p) for p in directory.rglob('*') if p.is_file() and p.name != 'final.json'}})
    return status


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--provider', choices=('circleci', 'rwx'), required=True)
    for name in ('module', 'contract', 'bundle'): parser.add_argument('--' + name + '-artifact', type=Path)
    args = parser.parse_args()
    sys.exit(execute(args.provider, args.module_artifact, args.contract_artifact, args.bundle_artifact))
