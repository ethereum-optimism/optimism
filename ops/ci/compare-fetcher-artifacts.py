#!/usr/bin/env python3
"""Compare complete original fetcher artifact builds on one source revision."""
import argparse
from collections import Counter
import importlib.util
import json
from pathlib import Path
import re
import tarfile


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


F = helper('fetcher-artifacts'); A = helper('compare-contract-artifacts'); G = F.G
EMPTY = A.digest(b'')
REQUIRED = {'settings.json', 'selection.json', 'coverage.json', 'submodules.txt', 'compiled.json',
            'compiler-files.json', 'compiler.tar.gz', 'inputs-after.json', 'portable.json',
            'build.stage.json', 'build.log', 'diff.stage.json', 'diff.log'}
SOURCES = ('op-fetcher/justfile', 'packages/contracts-bedrock/justfile', 'packages/contracts-bedrock/foundry.toml',
           'packages/contracts-bedrock/scripts/FetchChainInfo.s.sol')


def compiler(directory):
    manifest = G.read(directory / 'compiler-files.json'); files = {}
    with tarfile.open(directory / 'compiler.tar.gz', 'r|gz') as archive:
        for member in archive:
            path = Path(member.name)
            if (path.is_absolute() or '..' in path.parts or not member.isfile() or member.name in files
                    or not member.name.startswith(('packages/contracts-bedrock/forge-artifacts/', 'packages/contracts-bedrock/cache/'))):
                raise ValueError('Unsafe, duplicate or foreign original compiler output')
            data = archive.extractfile(member).read()
            if A.digest(data) != manifest.get(member.name): raise ValueError('Corrupt original fetcher compiler output')
            files[member.name] = data
    if not files or files.keys() != manifest.keys(): raise ValueError('Incomplete original fetcher compiler archive')
    return files


def report(directory, provider, sha):
    final = G.read(directory / 'final.json'); hashes = final['original_sha256']; empty = []
    if final['exit_code'] != 0 or final['report_errors'] or not REQUIRED <= hashes.keys():
        raise ValueError('Failed or incomplete original fetcher report')
    for name, value in hashes.items():
        path = Path(name); target = directory / path
        if path.is_absolute() or '..' in path.parts or name == 'final.json' or not re.fullmatch('[0-9a-f]{64}', value):
            raise ValueError('Unsafe original fetcher report manifest')
        if provider == 'circleci' and not target.exists() and value == EMPTY:
            target.parent.mkdir(parents=True, exist_ok=True); target.write_bytes(b'')
            empty.append({'path': name, 'sha256': value})
        if target.is_symlink() or F.S.digest(target) != value: raise ValueError('Missing or corrupt original fetcher report: ' + name)
    actual = {str(p.relative_to(directory)) for p in directory.rglob('*') if p.is_file() and p.name != 'final.json'}
    if actual != set(hashes): raise ValueError('Missing or unsealed original fetcher input')
    settings = G.read(directory / 'settings.json'); root = settings['workspace_root']
    if (settings['suite'], settings['source_sha'], settings['provider']) != ('fetcher-artifacts', sha, provider):
        raise ValueError('Wrong fetcher suite, source or provider')
    if (not settings['branch'] or settings['profile'] != 'lite' or settings['arguments'] != ['--deny-warnings', '--skip', 'test']
            or settings['rerun_fails'] != 0 or settings['portable_export'] != F.EXPORT_RULE
            or settings['fetcher_compiler'] != F.FETCHER_COMPILER
            or set(settings['environment']) != set(F.COMPILER_ENV)
            or G.read(directory / 'inputs-after.json') != settings['input_sha256']):
        raise ValueError('Changed original fetcher settings or source inputs')
    if provider == 'rwx' and (not settings['rwx_run_id'] or str(settings['rwx_task_attempt']) != '1'):
        raise ValueError('Missing native fetcher identity or unexpected retry')
    if F.SUBMODULES.revisions((directory / 'submodules.txt').read_text()) != settings['submodules']:
        raise ValueError('Changed original fetcher submodule selection')
    source_names = {str(p.relative_to(directory / 'source')) for p in (directory / 'source').rglob('*') if p.is_file()}
    if source_names != set(SOURCES): raise ValueError('Incomplete or extra original fetcher source selection')
    for name in SOURCES:
        if F.S.digest(directory / 'source' / name) != settings['input_sha256'].get(name):
            raise ValueError('Fetcher source differs from bound revision inputs')
    selected = F.inventory(directory / 'committed'); compiled = F.inventory(directory / 'compiled')
    portable = F.inventory(directory / 'portable')
    if selected != portable or portable != G.read(directory / 'portable.json') or compiled != G.read(directory / 'compiled.json'):
        raise ValueError('Fresh portable fetcher artifacts differ from committed bytes')
    if selected.keys() != compiled.keys(): raise ValueError('Original and portable fetcher selections differ')
    for name in compiled:
        if not G.read(directory / 'compiled' / name)['metadata']['compiler']['version'].startswith(F.FETCHER_COMPILER + '+'):
            raise ValueError('Original fetcher output uses a different compiler version')
        expected = F.portable_artifact(G.read(directory / 'compiled' / name), Path(root) / 'packages/contracts-bedrock')
        if (directory / 'portable' / name).read_bytes() != F.serialize(expected):
            raise ValueError('Portable fetcher export differs from the original compiler artifact')
    expected = {'authority': 'diff -qr', 'compiled_directory': F.BUILT, 'committed_directory': F.COMMITTED, 'committed_sha256': selected}
    if G.read(directory / 'selection.json') != expected or any(settings['input_sha256'].get(F.COMMITTED + '/' + n) != h for n,h in selected.items()):
        raise ValueError('Original fetcher selection differs from committed inputs')
    coverage = {'tests': 0, 'artifact_files': len(selected), 'complete_clean_build': True,
                'unchanged_committed_artifacts': True, 'identical_portable_artifacts': True}
    if G.read(directory / 'coverage.json') != coverage: raise ValueError('False original fetcher coverage')
    if {p.name.removesuffix('.stage.json') for p in directory.glob('*.stage.json')} != set(F.COMMANDS):
        raise ValueError('Missing original fetcher command or unexpected retry')
    for name, (argv, cwd) in F.COMMANDS.items():
        stage = G.read(directory / (name + '.stage.json'))
        if (stage['argv'] != argv or stage['cwd'] != str(Path(root) / cwd) or stage['exit_code'] != 0
                or stage.get('stdin') != 'devnull' or stage['log_sha256'] != F.S.digest(directory / (name + '.log'))):
            raise ValueError('Changed or failed original fetcher invocation')
    files = compiler(directory)
    for name, value in compiled.items():
        if files.get(F.BUILT + '/' + name) != (directory / 'compiled' / name).read_bytes():
            raise ValueError('Fetcher compiler archive differs from compared artifact bytes')
    return {'settings': settings, 'selection': expected, 'coverage': coverage, 'files': files,
            'original_sha256': hashes, 'declared_empty': empty}


def compare(directory, sha):
    if not re.fullmatch('[0-9a-f]{40}', sha): raise ValueError('Expected full fetcher benchmark SHA')
    a, b = (report(directory / p, provider, sha) for p, provider in [('circle', 'circleci'), ('rwx', 'rwx')])
    for key in ('source_sha', 'branch', 'input_sha256', 'submodules', 'environment', 'profile', 'arguments', 'rerun_fails', 'tool_versions', 'portable_export', 'fetcher_compiler'):
        if a['settings'][key] != b['settings'][key]: raise ValueError('Original fetcher settings differ: ' + key)
    if a['selection'] != b['selection'] or a['coverage'] != b['coverage']: raise ValueError('Complete fetcher artifact selection differs')
    x, y = a['files'], b['files']; info = 'packages/contracts-bedrock/forge-artifacts/build-info/'
    if {n for n in x if not n.startswith(info)} != {n for n in y if not n.startswith(info)}:
        raise ValueError('Complete fetcher compiler output inventories differ')
    if A.build_info(x, info) != A.build_info(y, info): raise ValueError('Complete fetcher compiler source graphs differ')
    counts = Counter(); contracts = 0; roots = [r['settings']['workspace_root'] for r in (a,b)]
    for name in x.keys() & y.keys():
        if name.startswith(info): continue
        if name.startswith('packages/contracts-bedrock/forge-artifacts/') and name.endswith('.json'):
            A.contract(json.loads(x[name], object_pairs_hook=G.C.no_duplicate_keys),
                       json.loads(y[name], object_pairs_hook=G.C.no_duplicate_keys), roots, counts); contracts += 1
        elif name == 'packages/contracts-bedrock/cache/solidity-files-cache.json':
            if A.normalize(A.cache_inputs(json.loads(x[name])), roots[0]) != A.normalize(A.cache_inputs(json.loads(y[name])), roots[1]):
                raise ValueError('Complete fetcher compiler cache inputs differ')
        elif x[name] != y[name]: raise ValueError('Unresolved original fetcher compiler output difference')
    return {'source_sha': sha, 'verified_parity': True, 'selection': a['selection'], 'coverage': a['coverage'],
        'complete_compiler_contracts': contracts, 'compiler_metadata_differences': dict(counts),
        'original_sha256': {'circle': a['original_sha256'], 'rwx': b['original_sha256']},
        'circle_manifest_declared_empty': a['declared_empty'], 'native_run_id': b['settings']['rwx_run_id']}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__); parser.add_argument('directory', type=Path)
    parser.add_argument('--sha', required=True); parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args(); result = compare(args.directory, args.sha); F.S.write(args.output, result)
    print({k:result[k] for k in ('source_sha', 'verified_parity', 'coverage', 'complete_compiler_contracts')})
