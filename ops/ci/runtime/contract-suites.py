#!/usr/bin/env python3
"""Run authoritative standard and changed-file contract suites with originals."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import subprocess
import sys


def helper(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


UP = helper('contract-upgrades')
ROOT = UP.ROOT
CONTRACTS = UP.CONTRACTS
FEATURES = ('main', *UP.FEATURES)
SUITES = ('standard', 'modified')
FILE_COMMANDS = {
    'standard': ['find', 'test', '-name', '*.t.sol'],
    'modified': ['bash', '-eo', 'pipefail', '-c',
                 "git diff origin/develop...HEAD --name-only --diff-filter=AM -- './test/**/*.t.sol' | sed 's|packages/contracts-bedrock/||'"],
}
SPLIT_COMMAND = ['bash', '-eo', 'pipefail', '-c', 'circleci tests split --split-by=timings < "$CI_CONTRACT_FILE_LIST"']
# GenerateNUTBundleTest calls the real script's run() and verifies the JSON it
# writes at Constants.CURRENT_BUNDLE_PATH. This tracked snapshot is both an
# initial fixture and an intentional runtime output, never a reusable verdict.
TRACKED_OUTPUTS = {'test/scripts/GenerateNUTBundle.t.sol':
                   'packages/contracts-bedrock/snapshots/upgrades/current-upgrade-bundle.json'}
PREPARED = {'settings.json', 'files.log', 'files.stage.json', 'file-selection.json', 'foundry-config.json',
            'foundry-config.stage.json', 'go-ffi.stage.json', 'contracts-build.stage.json', 'discovery.json',
            'discovery.stage.json', 'selection.json', 'signature-bindings.json', 'compiled.json', 'submodules.txt',
            'compile-only.json', 'source-after-preparation.json', 'submodules-sync.stage.json', 'submodules-init.stage.json'}


def inputs():
    result = {}
    for entry in UP.command('git', 'ls-files', '--stage', '-z').split('\0'):
        if not entry: continue
        header, name = entry.split('\t', 1); mode, sha, merge_stage = header.split()
        if merge_stage != '0' or name in result: raise ValueError('Unmerged or duplicate contract source input')
        path = ROOT / name
        if mode == '160000': result[name] = {'gitlink': sha}
        elif path.is_symlink(): result[name] = {'symlink': os.readlink(path)}
        elif path.is_file(): result[name] = UP.digest(path)
        else: raise ValueError('Missing tracked contract source input: ' + name)
    for name in ('contract-suites.py', 'contract-upgrades.py', 'git-submodule-report.py',
                 'ci-report.py'):
        path = ROOT / 'ops/ci/runtime' / name; result[str(path.relative_to(ROOT))] = UP.digest(path)
    return result


def runtime_outputs(files):
    return sorted({path for writer, path in TRACKED_OUTPUTS.items() if writer in files})


def verify_inputs(directory, before, phase, outputs=()):
    if outputs and phase != 'verdict': raise ValueError('Compilation cannot change tracked fixtures')
    after = inputs()
    UP.write(directory / ('source-after-' + phase + '.json'), after)
    changes = {name: {'before': before.get(name), 'after': after.get(name)}
               for name in sorted(set(before) | set(after)) if before.get(name) != after.get(name)}
    if changes:
        UP.write(directory / ('source-changes-' + phase + '.json'), changes)
    if phase == 'verdict':
        fixtures = {}
        for name in outputs:
            original = directory / 'tracked-fixtures/before' / name
            if not isinstance(before.get(name), str) or UP.digest(original) != before[name] or \
               not isinstance(after.get(name), str) or UP.digest(ROOT / name) != after[name]:
                raise ValueError('Missing or corrupt original tracked runtime fixture: ' + name)
            target = directory / 'tracked-fixtures/after' / name; target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(ROOT / name, target)
            fixtures[name] = {'before_sha256': before[name], 'after_sha256': after[name]}
        UP.write(directory / 'runtime-fixtures.json', fixtures)
    unexpected = set(changes) - set(outputs)
    if unexpected:
        raise ValueError('Contract ' + phase + ' changed source inputs: ' + ', '.join(sorted(unexpected)))


def configure(suite, feature):
    if suite not in SUITES or feature not in FEATURES: raise ValueError('Unknown contract suite or feature')
    branch = os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH')
    if not branch: raise ValueError('Missing tested contract branch')
    for name in list(os.environ):
        if name.startswith(('FOUNDRY_', 'DAPP_', 'DEV_FEATURE__', 'SYS_FEATURE__')): del os.environ[name]
    for name in ('ETH_RPC_URL', 'ETH_RPC_JWT', 'ETH_RPC_HEADERS', 'ETHERSCAN_API_KEY', 'MAINNET_RPC_URL',
                 'FORK_RPC_URL', 'FORK_BLOCK_NUMBER', 'L2_FORK_RPC_URL', 'L2_FORK_BLOCK_NUMBER'):
        os.environ.pop(name, None)
    profile = 'ciheavy' if suite == 'modified' else 'ci' if branch == 'develop' else 'liteci'
    if os.environ.get('CI_CONTRACT_PROFILE', profile) != profile: raise ValueError('Circle contract profile differs from suite and branch')
    os.environ.update(FOUNDRY_PROFILE=profile, FORK_TEST='false', L2_FORK_TEST='false', L2CM_ACTIVATION_TEST='false', CI='true')
    if feature != 'main': os.environ[UP.FEATURES[feature]] = 'true'
    return branch, profile


def file_selection(text):
    rows = text.splitlines()
    if len(rows) != len(set(rows)): raise ValueError('Duplicate contract test file discovery')
    for name in rows:
        p = Path(name)
        if p.is_absolute() or '..' in p.parts or not name.startswith('test/') or not name.endswith('.t.sol') or \
           any(c in name for c in '{},\r\t'):
            raise ValueError('Unsafe or invalid original contract test file')
    return sorted(rows)


def match_path(files):
    if not files: raise ValueError('Cannot execute an empty contract selection')
    return './test/{' + ','.join(name.removeprefix('test/') for name in files) + '}'


def file_command(suite, target=None):
    if target is None: return FILE_COMMANDS[suite]
    if suite != 'modified' or not re.fullmatch('[0-9a-f]{40}', target):
        raise ValueError('Invalid pinned contract target SHA')
    return [*FILE_COMMANDS[suite][:-1], FILE_COMMANDS[suite][-1].replace('origin/develop...HEAD', target + '...HEAD')]


def pinned_target(suite):
    target = os.environ.get('CI_CONTRACT_TARGET_SHA') or None
    if target is not None:
        file_command(suite, target)
        if os.environ.get('CI_CONTRACT_PROVIDER') != 'rwx':
            raise ValueError('Pinned contract target requires RWX provider')
        if UP.command('git', 'rev-parse', '--verify', target + '^{commit}') != target:
            raise ValueError('Pinned contract target is not an available commit')
    return target


def selected_files(directory, suite, provider, target=None):
    if UP.stage(directory, 'files', file_command(suite, target)): raise ValueError('Authoritative contract file discovery failed')
    files = file_selection((directory / 'files.log').read_text())
    if any(not (CONTRACTS / name).is_file() for name in files): raise ValueError('Missing discovered contract test file')
    assignment = files
    if provider == 'circleci' and files:
        if (os.environ.get('CIRCLE_NODE_TOTAL', '1'), os.environ.get('CIRCLE_NODE_INDEX', '0')) != ('1', '0'):
            raise ValueError('Circle contract partition count changed; complete collection is required')
        os.environ['CI_CONTRACT_FILE_LIST'] = str(directory / 'files.log')
        if UP.stage(directory, 'split', SPLIT_COMMAND): raise ValueError('Original Circle timing split failed')
        assignment = file_selection((directory / 'split.log').read_text())
        if assignment != files: raise ValueError('Circle split omitted or added contract test files')
    result = {'files': files, 'partitions': [{'index': 0, 'files': assignment}],
              'match_path': match_path(assignment) if files else None}
    UP.write(directory / 'file-selection.json', result)
    return result


def validate_config(config, suite):
    fuzz, invariant = config['fuzz'], config['invariant']
    expected = (20000, 128, 512) if suite == 'modified' else (128, 64, 32)
    if (fuzz['runs'], invariant['runs'], invariant['depth']) != expected or \
       any(config.get(k) for k in ('match_test', 'no_match_test', 'match_contract', 'no_match_contract', 'match_path', 'no_match_path', 'skip')):
        raise ValueError('Unexpected effective contract suite workload settings')
    if suite == 'modified' and (fuzz['timeout'], invariant['timeout']) != (300, 300):
        raise ValueError('Changed-file fuzz or invariant timeout differs')


def begin(directory, suite, feature):
    branch, profile = configure(suite, feature)
    test_list = 'find test -name "*.t.sol"' if suite == 'standard' else FILE_COMMANDS[suite][-1]
    if shlex.split(os.environ.get('CI_CONTRACT_TEST_LIST', test_list)) != shlex.split(test_list):
        raise ValueError('Circle test-list parameter differs from authoritative suite discovery')
    settings = {'source_sha': UP.revision(), 'suite': suite, 'feature': feature, 'branch': branch, 'profile': profile,
                'test_list': test_list,
                'provider': os.environ.get('CI_CONTRACT_PROVIDER', 'circleci'), 'workspace_root': str(ROOT),
                'forge': UP.command('forge', '--version'), 'go': UP.command('go', 'version'), 'just': UP.command('just', '--version'),
                'input_sha256': inputs(), 'rwx_run_id': os.environ.get('RWX_RUN_ID'),
                'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}
    UP.write(directory / 'settings.json', settings)
    target = pinned_target(suite)
    if suite == 'modified':
        if target is None:
            if UP.stage(directory, 'fetch-develop', ['git', 'fetch', '--no-tags', 'origin', '+refs/heads/develop:refs/remotes/origin/develop']):
                raise ValueError('Changed-file target history unavailable')
            target = UP.command('git', 'rev-parse', 'origin/develop')
        else:
            settings['target_binding'] = 'run-pinned'
        settings.update(target_sha=target, merge_base_sha=UP.command('git', 'merge-base', target, 'HEAD'))
        UP.write(directory / 'settings.json', settings)
    chosen = selected_files(directory, suite, settings['provider'], target if settings.get('target_binding') else None)
    settings['runtime_output_paths'] = runtime_outputs(chosen['files'])
    UP.write(directory / 'settings.json', settings)
    for name in settings['runtime_output_paths']:
        source = ROOT / name
        if source.is_symlink() or not isinstance(settings['input_sha256'].get(name), str) or \
           UP.digest(source) != settings['input_sha256'][name]:
            raise ValueError('Unbound tracked runtime fixture: ' + name)
        target = directory / 'tracked-fixtures/before' / name; target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, target)
    return settings


def prepare(directory, suite, feature):
    settings = begin(directory, suite, feature)
    chosen = json.loads((directory / 'file-selection.json').read_text())
    if not chosen['files']:
        UP.write(directory / 'coverage.json', {'tests': 0, 'eligible': False, 'reason': 'original changed-file selection is empty'})
        return 0
    for name, argv in [('submodules-sync', ['git', '-C', str(ROOT), 'submodule', 'sync', '--recursive']),
                       ('submodules-init', ['git', '-C', str(ROOT), '-c', 'protocol.file.allow=never', 'submodule',
                                            'update', '--init', '--recursive', '--jobs', '8'])]:
        status = UP.stage(directory, name, argv)
        if status: return status
    (directory / 'submodules.txt').write_text(UP.command('git', 'submodule', 'status', '--recursive') + '\n')
    UP.SUBMODULES.revisions((directory / 'submodules.txt').read_text())
    for name, argv, as_json in [('foundry-config', ['forge', 'config', '--json'], True),
                              ('go-ffi', ['just', 'build-go-ffi'], False), ('contracts-build', ['forge', 'build'], False),
                              ('discovery', ['forge', 'test', '--list', '--json', '--match-path', chosen['match_path']], True)]:
        status = UP.stage(directory, name, argv, json_output=as_json)
        if status: return status
    config = json.loads((directory / 'foundry-config.json').read_text()); validate_config(config, suite)
    out = Path(config['out']); out = out if out.is_absolute() else CONTRACTS / out
    bindings = UP.compiler_signatures(out); UP.write(directory / 'signature-bindings.json', bindings)
    discovered = UP.selection(json.loads((directory / 'discovery.json').read_text()), bindings)
    if any(identity.rsplit(':', 1)[0] not in chosen['files'] for identity, _ in discovered):
        raise ValueError('Forge selected a test outside the authoritative file manifest')
    UP.write(directory / 'selection.json', discovered)
    compiled = {}
    for name in ('forge-artifacts', 'artifacts/build-info', 'cache/solidity-files-cache.json', 'scripts/go-ffi/go-ffi'):
        p = CONTRACTS / name
        for f in ([p] if p.is_file() else sorted(p.rglob('*'))):
            if f.is_file(): compiled[str(f.relative_to(ROOT))] = UP.digest(f)
    if 'packages/contracts-bedrock/scripts/go-ffi/go-ffi' not in compiled or not any('/forge-artifacts/' in p for p in compiled):
        raise ValueError('Missing complete contract compiler outputs')
    UP.write(directory / 'compiled.json', compiled)
    UP.write(directory / 'compile-only.json', {'tests': 0, 'selected_cases': len(discovered), 'suite': suite, 'feature': feature})
    verify_inputs(directory, settings['input_sha256'], 'preparation')
    return 0


def restore(directory, prepared, suite, feature):
    UP.originals(prepared, PREPARED, 'rwx/contract-suites-prepare')
    old = json.loads((prepared / 'settings.json').read_text()); branch, profile = configure(suite, feature)
    chosen = json.loads((prepared / 'file-selection.json').read_text())
    if old.get('runtime_output_paths') != runtime_outputs(chosen['files']):
        raise ValueError('Unexpected compiled tracked runtime output role')
    if (old['source_sha'], old['suite'], old['feature'], old['branch'], old['profile'], old['workspace_root'], old['input_sha256']) != \
       (UP.revision(), suite, feature, branch, profile, str(ROOT), inputs()):
        raise ValueError('Stale contract compilation source or effective settings')
    for key, argv in [('forge', ['forge', '--version']), ('go', ['go', 'version']), ('just', ['just', '--version'])]:
        if old[key] != UP.command(*argv): raise ValueError('Contract runtime toolchain differs from compilation')
    if UP.SUBMODULES.revisions((prepared / 'submodules.txt').read_text()) != UP.SUBMODULES.revisions(UP.command('git', 'submodule', 'status', '--recursive')):
        raise ValueError('Contract runtime submodules differ from compilation')
    for name, value in json.loads((prepared / 'compiled.json').read_text()).items():
        if not name.startswith('packages/contracts-bedrock/') or '..' in Path(name).parts or UP.digest(ROOT / name) != value:
            raise ValueError('Missing or corrupt contract compiled output')
    for p in prepared.iterdir():
        if p.is_file() and p.name != 'final.json': shutil.copy2(p, directory / p.name)
        elif p.is_dir(): shutil.copytree(p, directory / p.name, dirs_exist_ok=True)
    shutil.copy2(prepared / 'final.json', directory / 'preparation-manifest.json')
    shutil.copy2(prepared / 'settings.json', directory / 'preparation-settings.json')
    old.update(rwx_run_id=os.environ.get('RWX_RUN_ID'), rwx_task_attempt=os.environ.get('RWX_TASK_ATTEMPT_NUMBER'))
    UP.write(directory / 'settings.json', old)
    if UP.stage(directory, 'runtime-config', ['forge', 'config', '--json'], json_output=True):
        raise ValueError('Runtime Foundry configuration unavailable')
    if json.loads((directory / 'runtime-config.json').read_text()) != json.loads((directory / 'foundry-config.json').read_text()):
        raise ValueError('Runtime Foundry settings differ from compilation')
    target = pinned_target(suite)
    if old.get('target_binding') == 'run-pinned':
        if target != old['target_sha'] or old['merge_base_sha'] != UP.command('git', 'merge-base', target, 'HEAD'):
            raise ValueError('Pinned contract target differs from compilation')
    elif target is not None:
        raise ValueError('Contract compilation lacks pinned target binding')
    elif suite == 'modified':
        if UP.stage(directory, 'runtime-fetch-develop', ['git', 'fetch', '--no-tags', 'origin', '+refs/heads/develop:refs/remotes/origin/develop']):
            raise ValueError('Runtime changed-file target history unavailable')
        if old['target_sha'] != UP.command('git', 'rev-parse', 'origin/develop'): raise ValueError('Changed-file target advanced after compilation')
    if UP.stage(directory, 'runtime-files', file_command(suite, target)): raise ValueError('Runtime contract file discovery failed')
    if file_selection((directory / 'runtime-files.log').read_text()) != json.loads((directory / 'file-selection.json').read_text())['files']:
        raise ValueError('Contract file selection changed after compilation')


def run(directory):
    chosen = json.loads((directory / 'file-selection.json').read_text())
    if not chosen['files']: return 0
    for name in ('cache/test-failures', 'cache/fuzz', 'cache/invariant'):
        p = CONTRACTS / name
        if p.is_dir(): shutil.rmtree(p)
        elif p.exists(): p.unlink()
    status = UP.stage(directory, 'tests', ['forge', 'test', '--match-path', chosen['match_path'], '--junit'], json_output=True)
    (directory / 'tests.json').rename(directory / 'original.junit.xml')
    (CONTRACTS / 'results').mkdir(exist_ok=True)
    shutil.copy2(directory / 'original.junit.xml', CONTRACTS / 'results/results.xml')
    if status and status < 128: UP.stage(directory, 'rerun', ['just', 'test-rerun'])
    if status == 0:
        status = UP.stage(directory, 'junit-nonempty', ['./scripts/checks/check-junit-tests-ran.sh', str(directory / 'original.junit.xml')])
    if status == 0:
        UP.write(directory / 'coverage.json', UP.junit(directory / 'original.junit.xml',
                 json.loads((directory / 'selection.json').read_text()), json.loads((directory / 'signature-bindings.json').read_text())))
        status = UP.stage(directory, 'lint-test-names', ['just', 'lint-forge-tests-check-no-build'])
    return status


def main():
    os.chdir(ROOT); p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('mode', choices=('prepare', 'run')); p.add_argument('--suite', choices=SUITES, required=True)
    p.add_argument('--feature', choices=FEATURES, required=True); p.add_argument('--prepared', type=Path)
    a = p.parse_args(); directory = ROOT / '.ci/contract-suites' / (a.suite + '-' + a.feature) / a.mode
    if a.prepared and a.mode != 'run': p.error('--prepared requires run mode')
    shutil.rmtree(directory, ignore_errors=True); directory.mkdir(parents=True); status, errors = 1, []
    if a.mode == 'run': (CONTRACTS / 'results/results.xml').unlink(missing_ok=True)
    try:
        if a.prepared: restore(directory, a.prepared, a.suite, a.feature); status = 0
        else: status = prepare(directory, a.suite, a.feature)
        if a.mode == 'run' and status == 0: status = run(directory)
        settings = json.loads((directory / 'settings.json').read_text())
        verify_inputs(directory, settings['input_sha256'], 'verdict' if a.mode == 'run' else 'preparation',
                      settings.get('runtime_output_paths', []) if a.mode == 'run' else [])
        if a.mode == 'prepare' and status == 0 and os.environ.get('RWX_VALUES'):
            chosen = json.loads((directory / 'file-selection.json').read_text())
            (Path(os.environ['RWX_VALUES']) / 'eligible').write_text('true\n' if chosen['files'] else 'false\n')
    except (OSError, ValueError, KeyError, UP.ET.ParseError, subprocess.CalledProcessError) as error:
        errors.append(str(error)); print(error, file=sys.stderr)
        if status == 0: status = 1
    finally:
        # Counterexamples and runtime fixtures are evidence, never compiler outputs.
        for name in ('cache/test-failures', 'cache/fuzz', 'cache/invariant', '.testdata', '.resource-metering.csv'):
            source = CONTRACTS / name
            if source.exists():
                target = directory / 'generated' / name; target.parent.mkdir(parents=True, exist_ok=True)
                if source.is_dir(): shutil.copytree(source, target, dirs_exist_ok=True)
                else: shutil.copy2(source, target)
    return UP.finish(directory, status, errors)


if __name__ == '__main__': sys.exit(main())
