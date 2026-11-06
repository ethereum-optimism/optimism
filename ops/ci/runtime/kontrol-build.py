#!/usr/bin/env python3
"""Run the complete Kontrol summary/build workload with sealed original outputs."""
import argparse
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


G = helper('cannon-go'); S = G.S; SUBMODULES = helper('git-submodule-report'); IMAGE = helper('kontrol-image')
ROOT = Path(__file__).resolve().parents[3]
CONTRACTS = ROOT / 'packages/contracts-bedrock'
PROOFS = 'packages/contracts-bedrock/test/kontrol/proofs'
GENERATED = {v: [PROOFS + '/utils/' + name + suffix + '.sol' for suffix in ('', 'Code')]
             for v, name in [('default', 'DeploymentSummary'), ('fault-proofs', 'DeploymentSummaryFaultProofs')]}
SOURCES = ('mise.toml', 'packages/contracts-bedrock/justfile', 'packages/contracts-bedrock/foundry.toml',
    'packages/contracts-bedrock/test/kontrol/scripts/make-summary-deployment.sh',
    'packages/contracts-bedrock/test/kontrol/scripts/common.sh',
    'packages/contracts-bedrock/test/kontrol/scripts/json/clean_json.py',
    'packages/contracts-bedrock/test/kontrol/scripts/json/reverse_key_values.py',
    'ops/ci/runtime/kontrol-image.json', 'ops/ci/runtime/kontrol-image.py')
COMMANDS = {'summaries': ['just', 'kontrol-summary-full'], 'proofs': ['just', 'forge-build', './test/kontrol/proofs']}
COMPILER_ENV = ('FOUNDRY_OPTIMIZER', 'FOUNDRY_OPTIMIZER_RUNS', 'FOUNDRY_VIA_IR', 'FOUNDRY_SOLC_VERSION',
                'FOUNDRY_EVM_VERSION', 'FOUNDRY_BYTECODE_HASH', 'FOUNDRY_CBOR_METADATA', 'FOUNDRY_REMAPPINGS',
                'FOUNDRY_AUTO_DETECT_REMAPPINGS', 'FOUNDRY_LIBRARIES', 'DAPP_OPTIMIZE', 'DAPP_OPTIMIZE_RUNS')
ENVIRONMENT = (*COMPILER_ENV, 'FOUNDRY_PROFILE', 'KONTROL_FP_DEPLOYMENT')


def command(*argv, cwd=ROOT): return subprocess.check_output(argv, cwd=cwd, text=True).strip()


def copy_files(directory, names):
    for name in names:
        path = ROOT / name
        if not path.is_file() or path.is_symlink(): raise ValueError('Missing or unsafe Kontrol output: ' + name)
        target = directory / name; target.parent.mkdir(parents=True, exist_ok=True); shutil.copyfile(path, target)


def seal(directory): return {str(p.relative_to(directory)): S.digest(p) for p in sorted(directory.rglob('*')) if p.is_file()}


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
            'workspace_root': str(ROOT), 'input_sha256': before, 'submodules': SUBMODULES.revisions(submodules),
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
        # Circle's two commands. Their exit statuses are the verdict.
        for name, argv in COMMANDS.items():
            status = S.stage(directory, name, argv, cwd=str(CONTRACTS), stdin=subprocess.DEVNULL)
            collect_compiler(directory / (name + '-compiler'))
            if status: raise ValueError('Original Kontrol workload failed: ' + name)
        after = G.inputs(); S.write(directory / 'inputs-after.json', after)
        allowed = set(sum(GENERATED.values(), []))
        changes = {name: {'before': before[name], 'after': after[name]} for name in before if before[name] != after.get(name)}
        if before.keys() != after.keys() or set(changes) - allowed or command('git','rev-parse','HEAD') != sha:
            raise ValueError('Kontrol generation changed unrelated tracked source inputs')
        IMAGE.inspect()
        # Retain the final summaries. The state diff is not per variant: the fault-proofs run
        # overwrites the default run's state diff.
        copy_files(directory / 'generated', sum(GENERATED.values(), []))
        S.write(directory / 'coverage.json', {'tests':0, 'proof_sources':len(selected['proof_sources']),
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
    parser.add_argument('--provider', choices=('circleci','rwx'), required=True)
    parser.add_argument('--contract-artifact', type=Path)
    args = parser.parse_args()
    sys.exit(execute(args.provider,args.contract_artifact))
