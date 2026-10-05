#!/usr/bin/env python3
"""Compile the original fetcher workload and compare untouched embedded artifacts."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


G = helper('cannon-go'); S = G.S; SUBMODULES = helper('git-submodule-report')
ROOT = Path(__file__).resolve().parents[2]
CONTRACTS = ROOT / 'packages/contracts-bedrock'
BUILT = 'packages/contracts-bedrock/forge-artifacts/FetchChainInfo.s.sol'
COMMITTED = 'op-fetcher/pkg/fetcher/fetch/forge-artifacts/FetchChainInfo.s.sol'
EXPORT_RULE = 'Retain every compiler field; relativize checkout-scoped remappings, sort and remove duplicate remappings; canonical JSON'
FETCHER_COMPILER = '0.8.30'
COMPILER_ENV = ('FOUNDRY_OPTIMIZER', 'FOUNDRY_OPTIMIZER_RUNS', 'FOUNDRY_VIA_IR', 'FOUNDRY_SOLC_VERSION',
                'FOUNDRY_EVM_VERSION', 'FOUNDRY_BYTECODE_HASH', 'FOUNDRY_CBOR_METADATA', 'FOUNDRY_REMAPPINGS',
                'FOUNDRY_AUTO_DETECT_REMAPPINGS', 'FOUNDRY_LIBRARIES', 'DAPP_OPTIMIZE', 'DAPP_OPTIMIZE_RUNS')
COMMANDS = {'build': (['just', 'compile-contracts'], 'op-fetcher'),
            'diff': (['diff', '-qr', '.ci/fetcher-artifacts/run/portable', COMMITTED], '.')}


def command(*argv): return subprocess.check_output(argv, cwd=ROOT, text=True).strip()


def inventory(directory):
    if directory.is_symlink() or not directory.is_dir(): raise ValueError('Missing or unsafe fetcher artifact directory')
    result = {}
    for path in sorted(directory.rglob('*')):
        if path.is_symlink(): raise ValueError('Unsafe fetcher artifact link')
        if not path.is_file(): continue
        if path.suffix != '.json': raise ValueError('Unexpected fetcher artifact file')
        value = G.read(path)
        target = value.get('metadata', {}).get('settings', {}).get('compilationTarget', {})
        if target != {'scripts/FetchChainInfo.s.sol': path.stem}:
            raise ValueError('Fetcher compiler artifact belongs to a different source or contract')
        result[str(path.relative_to(directory))] = S.digest(path)
    if not result: raise ValueError('Empty fetcher artifact selection')
    return result


def copy_tree(source, target):
    shutil.copytree(source, target)


def serialize(value): return (json.dumps(value, sort_keys=True, separators=(',', ':')) + '\n').encode()


def portable_artifact(value, contracts_root):
    """Retain every compiler field; normalize only checkout-scoped remappings."""
    value = json.loads(json.dumps(value))
    metadata = value.get('metadata')
    raw = json.loads(value.get('rawMetadata', '{}'), object_pairs_hook=G.C.no_duplicate_keys)
    if not isinstance(metadata, dict) or not isinstance(raw, dict): raise ValueError('Missing original fetcher compiler metadata')
    # Foundry's typed metadata omits some fields present in the original solc
    # JSON (for example error userdocs and empty ABI outputs). Keep both original
    # representations; never rebuild rawMetadata from the lossy typed object.
    if any(metadata.get(k) != raw.get(k) for k in ('compiler', 'sources', 'language', 'version')):
        raise ValueError('Inconsistent original fetcher compiler source metadata')
    prefix = str(contracts_root).rstrip('/') + '/'
    for original in (metadata, raw):
        remappings = original.get('settings', {}).get('remappings')
        if not isinstance(remappings, list): raise ValueError('Missing original fetcher compiler remappings')
        result = []
        for row in remappings:
            if not isinstance(row, str) or '=' not in row: raise ValueError('Invalid original fetcher compiler remapping')
            if row.startswith(prefix): row = row[len(prefix):]
            if row.startswith('/'): raise ValueError('Foreign absolute fetcher compiler remapping')
            result.append(row)
        original['settings']['remappings'] = sorted(set(result))
    value['rawMetadata'] = json.dumps(raw, sort_keys=True, separators=(',', ':'))
    return value


def export(source, target):
    inventory(source)
    # Validate every compiler artifact before changing any existing export.
    values = {p.name: portable_artifact(G.read(p), CONTRACTS) for p in sorted(source.glob('*.json'))}
    if target.is_symlink(): raise ValueError('Unsafe fetcher export destination')
    if target.exists() and any(p.is_symlink() or not p.is_file() or p.suffix != '.json' for p in target.iterdir()):
        raise ValueError('Unexpected file in fetcher export destination')
    target.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(dir=target.parent, prefix='.fetcher-export-') as temp:
        for name, value in values.items(): (Path(temp) / name).write_bytes(serialize(value))
        for name in values: os.replace(Path(temp) / name, target / name)
    for path in target.glob('*.json'):
        if path.name not in values: path.unlink()
    return inventory(target)


def collect_compiler(directory):
    artifacts = CONTRACTS / 'forge-artifacts'; cache = CONTRACTS / 'cache'
    compiler_files = {}
    for base in (artifacts, cache):
        for path in base.rglob('*'):
            if path.is_symlink(): raise ValueError('Unsafe original fetcher compiler output link')
            if path.is_file(): compiler_files[str(path.relative_to(ROOT))] = S.digest(path)
    S.write(directory / 'compiler-files.json', compiler_files)
    with tarfile.open(directory / 'compiler.tar.gz', 'w:gz') as archive:
        for name in sorted(compiler_files): archive.add(ROOT / name, arcname=name, recursive=False)


def execute(provider):
    os.chdir(ROOT); directory = ROOT / '.ci/fetcher-artifacts/run'
    shutil.rmtree(directory, ignore_errors=True); directory.mkdir(parents=True)
    status, errors = 1, []
    try:
        sha = command('git', 'rev-parse', 'HEAD')
        if not re.fullmatch('[0-9a-f]{40}', sha) or sha != (os.environ.get('CI_COMMIT_SHA') or os.environ.get('CIRCLE_SHA1') or sha):
            raise ValueError('Wrong fetcher artifact source revision')
        for argv in (['git', 'diff', '--quiet'], ['git', 'diff', '--cached', '--quiet']):
            if subprocess.call(argv): raise ValueError('Fetcher artifact inputs differ from their Git revision')
        before = G.inputs(); branch = os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH')
        if not branch: raise ValueError('Missing fetcher source branch')
        original_submodules = command('git', 'submodule', 'status', '--recursive')
        (directory / 'submodules.txt').write_text(original_submodules + '\n')
        settings = {'suite': 'fetcher-artifacts', 'provider': provider, 'source_sha': sha, 'branch': branch,
            'workspace_root': str(ROOT), 'input_sha256': before, 'submodules': SUBMODULES.revisions(original_submodules),
            'environment': {name: os.environ.get(name) for name in COMPILER_ENV},
            'profile': 'lite', 'arguments': ['--deny-warnings', '--skip', 'test'], 'rerun_fails': 0,
            'fetcher_compiler': FETCHER_COMPILER,
            'portable_export': EXPORT_RULE,
            'tool_versions': {name: command(*argv) for name, argv in [('forge', ['forge', '--version']), ('just', ['just', '--version'])]},
            'rwx_run_id': os.environ.get('RWX_RUN_ID'), 'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}
        S.write(directory / 'settings.json', settings)
        selected = inventory(ROOT / COMMITTED)
        if any(before.get(COMMITTED + '/' + name) != value for name, value in selected.items()):
            raise ValueError('Embedded fetcher artifacts differ from committed inputs')
        S.write(directory / 'selection.json', {'authority': 'diff -qr', 'compiled_directory': BUILT,
            'committed_directory': COMMITTED, 'committed_sha256': selected})
        copy_tree(ROOT / COMMITTED, directory / 'committed')
        for name in ('op-fetcher/justfile', 'packages/contracts-bedrock/justfile', 'packages/contracts-bedrock/foundry.toml',
                     'packages/contracts-bedrock/scripts/FetchChainInfo.s.sol'):
            target = directory / 'source' / name; target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(ROOT / name, target)
        for name, (argv, cwd) in COMMANDS.items():
            status = S.stage(directory, name, argv, cwd=str(ROOT / cwd), stdin=subprocess.DEVNULL)
            if name == 'build':
                collect_compiler(directory)  # Also retain complete partial output on failure/cancellation.
                if (ROOT / BUILT).exists():
                    copy_tree(ROOT / BUILT, directory / 'compiled')
                    S.write(directory / 'compiled.json', inventory(ROOT / BUILT))
                if status: raise ValueError('Original fetcher compilation failed')
                if any(not G.read(p)['metadata']['compiler']['version'].startswith(FETCHER_COMPILER + '+')
                       for p in (ROOT / BUILT).glob('*.json')):
                    raise ValueError('Fresh fetcher artifacts use the wrong pinned compiler')
                if inventory(ROOT / COMMITTED) != selected:
                    raise ValueError('Fetcher compilation overwrote the committed comparison input')
                S.write(directory / 'portable.json', export(ROOT / BUILT, directory / 'portable'))
            if status: raise ValueError('Fresh fetcher artifacts differ from committed bytes')
        after = G.inputs(); S.write(directory / 'inputs-after.json', after)
        if after != before or command('git', 'rev-parse', 'HEAD') != sha:
            raise ValueError('Fetcher compilation changed tracked source inputs')
        if not G.read(directory / 'compiler-files.json'): raise ValueError('Empty complete fetcher compiler output')
        S.write(directory / 'coverage.json', {'tests': 0, 'artifact_files': len(selected), 'complete_clean_build': True,
            'unchanged_committed_artifacts': True, 'identical_portable_artifacts': inventory(directory / 'portable') == selected})
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as error:
        errors.append(str(error)); print(error, file=sys.stderr); status = status or 1
    S.write(directory / 'final.json', {'exit_code': status, 'report_errors': errors,
        'original_sha256': {str(p.relative_to(directory)): S.digest(p) for p in directory.rglob('*') if p.is_file() and p.name != 'final.json'}})
    return status


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument('--provider', choices=('circleci', 'rwx')); mode.add_argument('--export', action='store_true')
    args = parser.parse_args()
    if args.export: export(ROOT / BUILT, ROOT / COMMITTED)
    else: sys.exit(execute(args.provider))
