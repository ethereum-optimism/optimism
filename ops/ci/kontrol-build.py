#!/usr/bin/env python3
"""Run the complete Kontrol summary/build workload with sealed original outputs."""
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tarfile

import importlib.util


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    value = importlib.util.module_from_spec(spec); spec.loader.exec_module(value)
    return value


F = helper('fetcher-artifacts'); G = F.G; S = F.S; IMAGE = helper('kontrol-image')
ROOT = Path(__file__).resolve().parents[2]
CONTRACTS = ROOT / 'packages/contracts-bedrock'
PROOFS = 'packages/contracts-bedrock/test/kontrol/proofs'
GENERATED = {v: [PROOFS + '/utils/' + name + suffix + '.sol' for suffix in ('', 'Code')]
             for v, name in [('default', 'DeploymentSummary'), ('fault-proofs', 'DeploymentSummaryFaultProofs')]}
SOURCES = ('mise.toml', 'packages/contracts-bedrock/justfile', 'packages/contracts-bedrock/foundry.toml',
    'packages/contracts-bedrock/test/kontrol/scripts/make-summary-deployment.sh',
    'packages/contracts-bedrock/test/kontrol/scripts/common.sh',
    'packages/contracts-bedrock/test/kontrol/scripts/json/clean_json.py',
    'packages/contracts-bedrock/test/kontrol/scripts/json/reverse_key_values.py',
    'ops/ci/kontrol-image.json', 'ops/ci/kontrol-image.py', 'ops/ci/kontrol-build.py')
COMMANDS = {'summaries': ['just', 'kontrol-summary-full'], 'proofs': ['just', 'forge-build', './test/kontrol/proofs']}
ENVIRONMENT = (*F.COMPILER_ENV, 'FOUNDRY_PROFILE', 'KONTROL_FP_DEPLOYMENT')


def command(*argv, cwd=ROOT): return subprocess.check_output(argv, cwd=cwd, text=True).strip()


def copy_files(directory, names):
    for name in names:
        path = ROOT / name
        if not path.is_file() or path.is_symlink(): raise ValueError('Missing or unsafe Kontrol output: ' + name)
        target = directory / name; target.parent.mkdir(parents=True, exist_ok=True); shutil.copyfile(path, target)


def seal(directory): return {str(p.relative_to(directory)): S.digest(p) for p in sorted(directory.rglob('*')) if p.is_file()}


def capture(phase):
    target = Path(os.environ['KONTROL_CI_REPORT_DIR']).resolve()
    if not target.is_relative_to(ROOT / '.ci') or phase not in ('deployment', 'load-inputs', 'generated'):
        raise ValueError('Invalid isolated Kontrol capture destination or phase')
    variant = 'fault-proofs' if os.environ.get('KONTROL_FP_DEPLOYMENT') == 'true' else 'default'
    names_file = 'packages/contracts-bedrock/deployments/kontrol' + ('-fp' if variant == 'fault-proofs' else '') + '.json'
    names = ['packages/contracts-bedrock/snapshots/state-diff/Kontrol-31337.json', names_file]
    if phase != 'deployment': names += [names_file + 'Reversed']
    if phase == 'generated': names += GENERATED[variant]
    directory = target / 'variants' / variant / phase
    if directory.exists(): raise ValueError('Duplicate Kontrol variant or capture phase')
    directory.mkdir(parents=True)
    copy_files(directory / 'files', names)
    summary = 'DeploymentSummary' + ('FaultProofs' if variant == 'fault-proofs' else '')
    load = ['kontrol', 'load-state', '--from-state-diff', summary, 'snapshots/state-diff/Kontrol-31337.json',
            '--contract-names', names_file.removeprefix('packages/contracts-bedrock/') + 'Reversed',
            '--output-dir', 'test/kontrol/proofs/utils', '--license', 'MIT']
    S.write(directory / 'manifest.json', {'variant': variant, 'phase': phase, 'source_sha': command('git', 'rev-parse', 'HEAD'),
        'profile': os.environ.get('FOUNDRY_PROFILE', 'default'), 'load_state_argv': load,
        'files': seal(directory / 'files')})


def collect_compiler(directory):
    directory.mkdir(parents=True, exist_ok=True); files = {}
    for relative in ('forge-artifacts', 'cache', 'artifacts/build-info'):
        for path in (CONTRACTS / relative).rglob('*'):
            if path.is_symlink(): raise ValueError('Unsafe Kontrol compiler output link')
            if path.is_file(): files[str(path.relative_to(ROOT))] = S.digest(path)
    S.write(directory / 'files.json', files)
    with tarfile.open(directory / 'files.tar.gz', 'w:gz') as archive:
        for name in sorted(files): archive.add(ROOT / name, arcname=name, recursive=False)


def select():
    names = command('git', 'ls-files', PROOFS).splitlines()
    proof_files = {name: S.digest(ROOT / name) for name in names if name.endswith('.sol')}
    if not proof_files or not set(sum(GENERATED.values(), [])) <= proof_files.keys():
        raise ValueError('Missing complete Kontrol proof source discovery')
    return {'authority': 'just kontrol-summary-full; just forge-build ./test/kontrol/proofs',
        'variants': ['default', 'fault-proofs'], 'proof_sources': proof_files,
        'generated': GENERATED, 'tests': 0}


def execute(provider, contract_artifact=None):
    os.chdir(ROOT); directory = ROOT / '.ci/kontrol-build/run'
    shutil.rmtree(directory, ignore_errors=True); directory.mkdir(parents=True)
    status, errors = 1, []
    try:
        sha = command('git', 'rev-parse', 'HEAD')
        if not re.fullmatch('[0-9a-f]{40}', sha) or sha != (os.environ.get('CI_COMMIT_SHA') or os.environ.get('CIRCLE_SHA1') or sha):
            raise ValueError('Wrong Kontrol source revision')
        for argv in (['git', 'diff', '--quiet'], ['git', 'diff', '--cached', '--quiet']):
            if subprocess.call(argv): raise ValueError('Kontrol source differs from its Git revision')
        before = G.inputs(); branch = os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH')
        if not branch or os.environ.get('FOUNDRY_PROFILE') not in (None, 'default') or os.environ.get('KONTROL_FP_DEPLOYMENT') not in (None, 'false'):
            raise ValueError('Missing branch or overridden original Kontrol mode/profile')
        submodules = command('git', 'submodule', 'status', '--recursive')
        (directory / 'submodules.txt').write_text(submodules + '\n')
        settings = {'suite': 'kontrol-build', 'source_sha': sha, 'branch': branch, 'provider': provider,
            'workspace_root': str(ROOT), 'input_sha256': before, 'submodules': F.SUBMODULES.revisions(submodules),
            'profile': 'default', 'environment': {name: os.environ.get(name) for name in ENVIRONMENT},
            'rerun_fails': 0, 'image': IMAGE.selection(), 'kontrol_version': 'Kontrol version: ' + IMAGE.selection()['version'],
            'tools': {name: command(*argv) for name, argv in [('forge',['forge','--version']), ('just',['just','--version']),
                ('jq',['jq','--version']), ('yq',['yq','--version'])]},
            'docker_version': command('docker', 'version', '--format', '{{json .}}'),
            'rwx_run_id': os.environ.get('RWX_RUN_ID'), 'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}
        S.write(directory / 'settings.json', settings)
        image = ROOT / '.ci/kontrol-build/image'
        if provider == 'circleci': IMAGE.prepare()
        record = IMAGE.verify(image)
        shutil.copytree(image, directory / 'image')
        with (directory / 'runtime-image.json').open('wb') as output:
            subprocess.run(['docker', 'image', 'inspect', record['selection']['tag']], stdout=output, check=True)
        if contract_artifact:
            G.ARTIFACTS.restore('contracts-kontrol', contract_artifact)
            metadata = G.read(contract_artifact / 'metadata.json')
            if metadata['tool_versions']['forge'] != settings['tools']['forge']:
                raise ValueError('Kontrol contract producer uses a different compiler toolchain')
            helper('kontrol-contracts').verify(contract_artifact,sha,before)
            shutil.copytree(contract_artifact, directory / 'dependencies/contracts-kontrol')
        collect_compiler(directory / 'initial-compiler')
        if not G.read(directory / 'initial-compiler/files.json'): raise ValueError('Missing initial CI contract artifacts')
        status = S.stage(directory, 'config', ['forge','config','--json'], stdout_json=True, cwd=str(CONTRACTS), stdin=subprocess.DEVNULL)
        if status: raise ValueError('Kontrol effective compiler configuration failed')
        selected = select(); S.write(directory / 'selection.json', selected)
        copy_files(directory / 'source', [*SOURCES, *selected['proof_sources']])
        os.environ['KONTROL_CI_REPORT_DIR'] = str(directory)
        for name, argv in COMMANDS.items():
            status = S.stage(directory, name, argv, cwd=str(CONTRACTS), stdin=subprocess.DEVNULL)
            collect_compiler(directory / (name + '-compiler'))
            if status: raise ValueError('Original Kontrol workload failed: ' + name)
        if sorted(p.name for p in (directory / 'variants').iterdir()) != selected['variants']:
            raise ValueError('Missing or extra original Kontrol summary variant')
        for variant in selected['variants']:
            if sorted(p.name for p in (directory / 'variants' / variant).iterdir()) != ['deployment','generated','load-inputs']:
                raise ValueError('Missing, duplicate or extra original Kontrol summary phase')
            if G.read(directory / 'variants' / variant / 'generated/manifest.json')['source_sha'] != sha:
                raise ValueError('Stale original Kontrol summary source')
        after = G.inputs(); S.write(directory / 'inputs-after.json', after)
        allowed = set(sum(GENERATED.values(), []))
        changes = {name: {'before': before[name], 'after': after[name]} for name in before if before[name] != after.get(name)}
        if before.keys() != after.keys() or set(changes) - allowed or command('git','rev-parse','HEAD') != sha:
            raise ValueError('Kontrol generation changed unrelated tracked source inputs')
        IMAGE.inspect()
        copy_files(directory / 'generated', sum(GENERATED.values(), []))
        S.write(directory / 'coverage.json', {'tests':0, 'variants':selected['variants'], 'proof_sources':len(selected['proof_sources']),
            'generated_files':4, 'tracked_generated_changes':changes, 'complete_original_commands':True})
        status = 0
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as error:
        errors.append(str(error)); print(error, file=sys.stderr); status = status or 1
    finally:
        # Failures and cancellation retain complete partial compiler/output state.
        try:
            image = ROOT / '.ci/kontrol-build/image'
            for path in image.iterdir() if image.exists() else []:
                if path.is_file() and not path.is_symlink():
                    (directory / 'image').mkdir(exist_ok=True)
                    shutil.copyfile(path, directory / 'image' / path.name)
            collect_compiler(directory / 'final-compiler')
            for relative in ('snapshots/state-diff', 'deployments', 'test/kontrol/proofs/utils'):
                names = [str(p.relative_to(ROOT)) for p in (CONTRACTS / relative).rglob('*') if p.is_file() and not p.is_symlink()]
                copy_files(directory / 'runtime-outputs', names)
        except (OSError, ValueError) as error:
            errors.append(str(error)); status = status or 1
    S.write(directory / 'final.json', {'exit_code':status,'report_errors':errors,
        'original_sha256':{n:h for n,h in seal(directory).items() if n!='final.json'}})
    return status


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument('--provider', choices=('circleci','rwx')); mode.add_argument('--capture', choices=('deployment','load-inputs','generated'))
    parser.add_argument('--contract-artifact', type=Path)
    args = parser.parse_args()
    if args.capture: capture(args.capture)
    else: sys.exit(execute(args.provider,args.contract_artifact))
