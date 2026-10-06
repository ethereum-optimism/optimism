#!/usr/bin/env python3
"""Compile complete ABI evidence and run the real selector publisher privately."""
import argparse
from collections import Counter
from concurrent.futures import ThreadPoolExecutor
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import time
import urllib.parse
import urllib.request


def helper(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


R = helper('selector-registry')
UP = helper('contract-upgrades')
ROOT, CONTRACTS = UP.ROOT, UP.CONTRACTS
COMMAND = ['just', 'update-selectors']
COMPILERS = ('0.8.15', '0.8.19', '0.8.25', '0.8.28', '0.8.30')
IMPLEMENTATION = ('selector-upload.py', 'selector-registry.py', 'contract-upgrades.py',
                  'git-submodule-report.py', 'ci-report.py')


def read(path):
    def pairs(items):
        result = {}
        for key, value in items:
            R.check(key not in result, 'Duplicate original JSON key')
            result[key] = value
        return result
    return json.loads(Path(path).read_bytes(), object_pairs_hook=pairs,
                      parse_constant=lambda _: (_ for _ in ()).throw(ValueError('Non-finite original JSON')))


def files(directory):
    result = {}
    for path in sorted(Path(directory).rglob('*')):
        R.check(not path.is_symlink(), 'Linked selector evidence')
        if path.is_file() and path != Path(directory) / 'final.json':
            result[str(path.relative_to(directory))] = R.digest(path)
    return result


def seal(directory, state, status, errors, uploads=0):
    R.write(directory / 'final.json', {'state': state, 'exit_code': status, 'tests': 0,
            'uploads': uploads, 'errors': errors, 'sha256': files(directory)})


def original(directory, state):
    final = read(directory / 'final.json')
    R.check(final['state'] == state and type(final['exit_code']) is int and final['exit_code'] == 0
            and type(final['tests']) is int and final['tests'] == 0
            and type(final['uploads']) is int and final['uploads'] == (1 if state == 'passed' else 0)
            and final['errors'] == []
            and final['sha256'] == files(directory), 'Incomplete or corrupt selector originals')
    return final


def configuration():
    return json.loads(UP.command('forge', 'config', '--json', cwd=CONTRACTS))


def compiler_tools(versions):
    result = {}
    for version in sorted(versions):
        R.check(re.fullmatch(r'0\.\d+\.\d+', version), 'Invalid selected compiler version')
        path = UP.command('svm', 'which', version)
        result[version] = {'version': UP.command(path, '--version'), 'sha256': R.digest(path)}
    return result


def identity():
    R.check(os.environ.get('FOUNDRY_PROFILE', 'default') == 'default', 'Selector publisher requires the default profile')
    R.check(not any(key.startswith(('FOUNDRY_', 'DAPP_', 'DEV_FEATURE__', 'SYS_FEATURE__'))
              and key != 'FOUNDRY_PROFILE' for key in os.environ), 'Unexpected selector compiler override')
    sha = UP.revision()
    branch = os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH')
    R.check(branch, 'Missing selector branch metadata')
    tools = {}
    for name in ('forge', 'cast', 'just'):
        binary = shutil.which(name)
        R.check(binary, 'Missing selector tool: ' + name)
        tools[name] = {'version': UP.command(name, '--version'), 'sha256': R.digest(binary)}
    compilers = {}
    for version in COMPILERS:
        path = UP.command('svm', 'which', version)
        compilers[version] = {'version': UP.command(path, '--version'), 'sha256': R.digest(path)}
    return {'version': 1, 'source_sha': sha, 'branch': branch, 'profile': 'default',
            'provider': os.environ.get('CI_CHECK_PROVIDER', 'circleci'), 'workspace_root': str(ROOT),
            'tools': tools, 'compilers': compilers, 'command': COMMAND,
            'configuration': configuration(), 'inputs': R.source_inputs(ROOT, sha, True),
            'implementation': {name: R.digest(Path(__file__).with_name(name)) for name in IMPLEMENTATION},
            'native_run_id': os.environ.get('RWX_RUN_ID'),
            'native_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}


def canonical_type(parameter):
    kind = parameter.get('type')
    R.check(isinstance(kind, str), 'Missing canonical ABI parameter type')
    if kind.startswith('tuple'):
        suffix = kind.removeprefix('tuple')
        R.check(re.fullmatch(r'(?:\[\d*\])*', suffix) is not None
                and isinstance(parameter.get('components'), list), 'Invalid ABI tuple')
        return '(' + ','.join(canonical_type(p) for p in parameter['components']) + ')' + suffix
    R.check(re.fullmatch(r'(?:address|bool|string|bytes\d*|u?int\d+|u?fixed\d+x\d+|function)(?:\[\d*\])*', kind),
            'Noncanonical ABI parameter type')
    return kind


def signatures(abi):
    R.check(isinstance(abi, list), 'Missing compiler ABI')
    rows = []
    for item in abi:
        kind = item.get('type')
        R.check(kind in ('constructor', 'fallback', 'receive', 'function', 'event', 'error'), 'Unknown ABI declaration')
        if kind not in ('function', 'event', 'error'):
            continue
        name, inputs = item.get('name'), item.get('inputs')
        R.check(isinstance(name, str) and re.fullmatch(r'[A-Za-z_$][A-Za-z0-9_$]*', name)
                and isinstance(inputs, list), 'Invalid ABI signature')
        rows.append({'kind': kind, 'signature': name + '(' + ','.join(canonical_type(p) for p in inputs) + ')'})
    R.check(len({(r['kind'], r['signature']) for r in rows}) == len(rows), 'Duplicate compiler ABI signature')
    return sorted(rows, key=lambda r: (r['kind'], r['signature']))


def safe_relative(name):
    path = Path(name)
    R.check(not path.is_absolute() and '..' not in path.parts and path.parts, 'Unsafe compiler source or artifact path')
    return path


def catalog(cache, build_info, artifacts):
    """Use the original compiler cache's source/version/profile assignments."""
    R.check(cache.get('preprocessed') is True and isinstance(cache.get('files'), dict)
            and isinstance(cache.get('profiles'), dict) and isinstance(cache.get('builds'), list)
            and len(cache['builds']) == len(set(cache['builds'])) == len(build_info), 'Incomplete compiler ABI inventory')
    R.check(set(cache['builds']) == set(build_info), 'Missing or extra original compiler unit')
    source_ids = {}
    for build_id, unit in build_info.items():
        R.check(unit['id'] == build_id and unit['language'] == 'Solidity'
                and unit['solcVersion'] == unit['input']['version'], 'Foreign compiler unit identity')
        unit_ids = {str(row['id']): source for source, row in unit['output']['sources'].items()}
        R.check(len(unit_ids) == len(unit['output']['sources']) and unit_ids == unit['source_id_to_path'],
                'Corrupt compiler unit source map')
        for source, row in unit['output']['sources'].items():
            source_ids.setdefault((source, unit['solcVersion']), set()).add(row['id'])
    source_prefix = safe_relative(cache['paths']['sources'])
    selected, bindings, consumed = [], {}, set()
    for source, row in sorted(cache['files'].items()):
        source_path = safe_relative(source)
        R.check(row['sourceName'] == source and row['seenByCompiler'] is True, 'Uncompiled or foreign ABI source')
        in_sources = source_path.is_relative_to(source_prefix)
        is_test = source.endswith('.t.sol')
        for contract, versions in sorted(row['artifacts'].items()):
            for version, profiles in sorted(versions.items()):
                for profile, ref in sorted(profiles.items()):
                    R.check(profile in cache['profiles'] and ref['build_id'] in build_info, 'Foreign ABI compiler profile')
                    name = str(safe_relative(ref['path']))
                    R.check(name in artifacts and name not in consumed, 'Missing or duplicate compiler ABI artifact')
                    consumed.add(name)
                    unit = build_info[ref['build_id']]
                    R.check(unit['solcVersion'] == version and unit['input']['version'] == version
                            and source in unit['input']['sources'], 'ABI artifact compiler provenance differs')
                    artifact = artifacts[name]
                    compiled = unit['output']['contracts'][source][contract]
                    # Foundry 1.8.3 takes artifact.id from the first SourceFile
                    # matching path/version, without considering its profile.
                    # Use the exact build unit's own source map for provenance;
                    # retain and verify the artifact's separate original ID.
                    unit_id = unit['output']['sources'][source]['id']
                    R.check(artifact['abi'] == compiled['abi']
                            and type(artifact['id']) is int
                            and artifact['id'] in source_ids[(source, version)]
                            and unit['source_id_to_path'][str(unit_id)] == source,
                            'Corrupt compiler ABI binding: ' + source + ':' + contract + ':' + version + ':' + profile)
                    entries = signatures(artifact['abi'])
                    reason = 'selected' if in_sources and not is_test and entries else \
                             'outside-source-path' if not in_sources else 'test-source' if is_test else 'empty-selector-abi'
                    key = source + ':' + contract + ':' + version + ':' + profile
                    R.check(key not in bindings, 'Duplicate ABI declaration identity')
                    binding = {'source': source, 'contract': contract, 'compiler': version, 'profile': profile,
                               'artifact': name, 'build_id': ref['build_id'], 'abi': artifact['abi'],
                               'artifact_source_id': artifact['id'], 'unit_source_id': unit_id,
                               'signatures': entries, 'selection': reason}
                    bindings[key] = binding
                    if reason == 'selected':
                        selected.append(key)
    R.check(consumed == set(artifacts) and selected, 'Extra artifacts or empty selector workload')
    return {'declarations': bindings, 'selected': sorted(selected),
            'compiler_profiles': cache['profiles'], 'compiled_sources': sorted(cache['files'])}


def keccak(signature):
    value = UP.command('cast', 'keccak', signature)
    R.check(re.fullmatch(r'0x[0-9a-f]{64}', value), 'Invalid independent selector digest')
    return signature, value


def expected(catalogue):
    sets = {'function': set(), 'event': set()}
    for key in catalogue['selected']:
        for row in catalogue['declarations'][key]['signatures']:
            sets['event' if row['kind'] == 'event' else 'function'].add(row['signature'])
    with ThreadPoolExecutor(max_workers=8) as pool:
        hashes = dict(pool.map(keccak, sorted(set.union(*sets.values()))))
    return {'signatures': {kind: {sig: hashes[sig] if kind == 'event' else hashes[sig][:10]
                         for sig in sorted(values)} for kind, values in sets.items()}, 'hash32': hashes}


def discover(directory, settings):
    directory.mkdir(parents=True, exist_ok=True)
    # build_info only persists the ABI compiler's original inputs/outputs.
    # The upload below keeps the publisher's original build_info=false.
    for name in ('out', 'cache_path'):
        path = CONTRACTS / safe_relative(settings['configuration'][name])
        shutil.rmtree(path, ignore_errors=True)
    info_path = settings['configuration']['build_info_path'] or settings['configuration']['out'] + '/build-info'
    shutil.rmtree(CONTRACTS / safe_relative(info_path), ignore_errors=True)
    os.environ['FOUNDRY_BUILD_INFO'] = 'true'
    try:
        status = UP.stage(directory, 'abi-oracle', ['forge', 'selectors', 'list', '--no-group'])
        R.check(status == 0, 'Complete compiler ABI discovery failed')
    finally:
        os.environ.pop('FOUNDRY_BUILD_INFO', None)
    cache_path = CONTRACTS / settings['configuration']['cache_path'] / 'solidity-files-cache.json'
    cache = read(cache_path)
    info_root = CONTRACTS / cache['paths']['build_infos']
    infos = {p.stem: read(p) for p in sorted(info_root.glob('*.json'))}
    artifact_root = CONTRACTS / cache['paths']['artifacts']
    artifacts = {str(p.relative_to(artifact_root)): read(p) for p in sorted(artifact_root.rglob('*.json'))
                 if not p.is_relative_to(info_root)}
    # Preserve original compiler evidence even when validation rejects it.
    shutil.copy2(cache_path, directory / 'compiler-cache.json')
    shutil.copytree(info_root, directory / 'compiler-units')
    shutil.copytree(artifact_root, directory / 'compiler-artifacts')
    selection = catalog(cache, infos, artifacts)
    R.write(directory / 'selection.json', selection)
    return selection, compiler_tools({unit['solcVersion'] for unit in infos.values()})


def prepare_compilers(directory):
    # The complete graph requires 0.8.30. Installing it before auto detection
    # makes a cold worker choose the same versions as a restored SVM cache.
    for version in COMPILERS:
        try: UP.command('svm', 'which', version)
        except subprocess.CalledProcessError:
            for attempt in range(1, 6):
                status = UP.stage(directory, 'solc-' + version + '-attempt-' + str(attempt), ['svm', 'install', version])
                if status == 0: break
                R.check(attempt < 5, 'Pinned selector compiler installation failed')
                time.sleep(2 ** attempt)


def prepare(directory):
    R.check(not directory.exists(), 'Selector preparation already exists')
    directory.mkdir(parents=True)
    status, errors = 1, []
    try:
        prepare_compilers(directory)
        settings = identity()
        R.write(directory / 'settings.json', settings)
        R.write(directory / 'foundry-config.json', settings['configuration'])
        # Auto detection favors installed compatible solc versions. Its first
        # complete pass can install a further version required by dependencies.
        # Retain that pass, then bind the selection with all those compilers
        # already available to both the cached producer and fresh publisher.
        _, resolved = discover(directory / 'initial-discovery', settings)
        R.write(directory / 'initial-discovery/selected-compilers.json', resolved)
        selection, selected = discover(directory, settings)
        R.check(set(selected) <= set(resolved) and compiler_tools(resolved) == resolved,
                'Compiler resolution did not stabilize after complete discovery')
        R.write(directory / 'selected-compilers.json', resolved)
        R.write(directory / 'expected.json', expected(selection))
        R.check(identity() == settings, 'ABI discovery changed effective settings or source inputs')
        status = 0
    except BaseException as error:
        errors.append(str(error) or type(error).__name__)
        if isinstance(error, KeyboardInterrupt):
            status = 130
        raise
    finally:
        seal(directory, 'compiled' if not errors and status == 0 else 'failed', status or 1 if errors else status, errors)


def imports(directory, expectation):
    result = {kind: {'imported': {}, 'duplicated': {}} for kind in ('function', 'event')}
    sent = {kind: {} for kind in result}
    metadata = sorted(directory.glob('http-*.json'), key=lambda p: int(p.stem.split('-')[1])
                      if re.fullmatch(r'http-\d+', p.stem) else -1)
    metadata = [p for p in metadata if re.fullmatch(r'http-\d+', p.stem)]
    R.check(metadata, 'No actual selector POSTs')
    for number, path in enumerate(metadata):
        row = read(path)
        R.check(path.stem == 'http-' + str(number) and row['method'] == 'POST' and row['host'] == R.HOST
                and row['path'] == R.IMPORT and row['status'] == 200, 'Failed, foreign or incomplete original selector request')
        request, response = (directory / (path.stem + suffix) for suffix in ('-request.json', '-response.json'))
        R.check(R.digest(request) == row['request_sha256'] and R.digest(response) == row['response_sha256'],
                'Corrupt original selector payload')
        payload, verdict = read(request), read(response)
        R.check(set(payload) == set(result) and verdict['ok'] is True and set(verdict['result']) == set(result),
                'Invalid original selector protocol')
        for kind in result:
            values, outcome = payload[kind], verdict['result'][kind]
            R.check(isinstance(values, list) and len(values) <= 1000 and len(set(values)) == len(values)
                    and not (set(values) & set(sent[kind])) and set(outcome) == {'imported', 'duplicated', 'invalid'}
                    and outcome['invalid'] == [] and not (set(outcome['imported']) & set(outcome['duplicated']))
                    and set(outcome['imported']) | set(outcome['duplicated']) == set(values), 'Omitted, repeated or rejected selectors')
            for state in ('imported', 'duplicated'):
                for signature, digest in outcome[state].items():
                    R.check(expectation['signatures'][kind].get(signature) == digest, 'Unexpected selector import digest')
                result[kind][state].update(outcome[state])
            sent[kind].update(outcome['imported'] | outcome['duplicated'])
    R.check(sent == expectation['signatures'], 'Actual Forge upload differs from complete compiler ABI selection')
    return {'requests': len(metadata), 'outcomes': result}


def lookup_client(expectation_path, output):
    expectation = read(expectation_path)
    hashes = expectation['hash32']
    expected_maps = {'function': {}, 'event': {}}
    # The official DB is type-independent: include every row sharing a hash,
    # including legitimate four-byte collisions and event-only signatures.
    for signature, digest in hashes.items():
        expected_maps['function'].setdefault(digest[:10], []).append(signature)
        expected_maps['event'].setdefault(digest, []).append(signature)
    observed = {'function': {}, 'event': {}}
    for kind, wanted in expected_maps.items():
        keys = sorted(wanted)
        for start in range(0, len(keys), 50):
            query = urllib.parse.urlencode({kind: ','.join(keys[start:start + 50]), 'filter': 'false'})
            with urllib.request.urlopen('https://' + R.HOST + R.LOOKUP + '?' + query, timeout=15) as response:
                row = json.loads(response.read())
            R.check(row['ok'] is True and set(row['result']) == set(observed)
                    and set(row['result'][kind]) == set(keys[start:start + 50])
                    and row['result']['event' if kind == 'function' else 'function'] == {}, 'Incomplete registry lookup')
            for digest, values in row['result'][kind].items():
                R.check(isinstance(values, list) and all(set(v) == {'name', 'filtered', 'hasVerifiedContract'}
                        and type(v['filtered']) is bool and v['hasVerifiedContract'] is False for v in values)
                        and sorted(v['name'] for v in values) == sorted(wanted[digest]), 'Registry lookup omitted or added signatures')
                observed[kind][digest] = values
    R.write(Path(output), observed)


def run(directory, prepared, registry_source, images):
    original(prepared, 'compiled')
    settings, wanted = read(prepared / 'settings.json'), read(prepared / 'expected.json')
    compilers = read(prepared / 'selected-compilers.json')
    R.check(compiler_tools(compilers) == compilers, 'Stale selected selector compiler')
    current = identity()
    for key in ('version', 'source_sha', 'branch', 'profile', 'workspace_root', 'tools', 'compilers',
                'command', 'configuration', 'inputs', 'implementation'):
        R.check(current[key] == settings[key], 'Stale selector preparation: ' + key)
    R.check(not directory.exists(), 'Selector verdict already exists')
    directory.mkdir(parents=True)
    R.write(directory / 'settings.json', current)
    shutil.copytree(prepared, directory / 'prepared')
    status, errors, uploads = 1, [], 0
    registry = R.Registry(registry_source, directory / 'registry', read(images))
    try:
        if UP.stage(directory, 'recipe', ['just', '--dry-run', 'update-selectors']):
            raise ValueError('Cannot inspect actual publisher recipe')
        R.check((directory / 'recipe.stderr.log').read_text().strip() == 'forge selectors up --all'
                and (directory / 'recipe.log').read_text().strip() == '', 'Original selector recipe changed')
        registry.start()
        status = registry.run('upload', COMMAND, CONTRACTS)
        R.check(status == 0, 'Real selector publisher failed')
        catalogue = read(prepared / 'selection.json')
        names = Counter(catalogue['declarations'][key]['contract'] for key in catalogue['selected'])
        text = (directory / 'registry/upload/stdout.log').read_text()
        statuses = (directory / 'registry/upload/stderr.log').read_text()
        observed = Counter(re.findall(r'^Uploading selectors for ([A-Za-z_$][A-Za-z0-9_$]*)\.\.\.$', statuses, re.MULTILINE))
        R.check(observed == names and 'Selectors successfully uploaded to OpenChain' in text,
                'Publisher omitted selected declarations or its real success output')
        R.write(directory / 'imports.json', imports(directory / 'registry/upload', wanted))
        rows = registry.rows()
        R.check({row['signature']: row['hash32'] for row in rows} == wanted['hash32']
                and all(row['hash4'] == row['hash32'][:10] and isinstance(row['created_at'], str) for row in rows),
                'Real database does not contain exactly the complete uploaded selection')
        lookup = directory / 'lookup.json'
        status = registry.run('lookup', [sys.executable, str(Path(__file__).resolve()), 'lookup-client',
                                        str(prepared / 'expected.json'), str(lookup)], CONTRACTS)
        R.check(status == 0 and lookup.is_file(), 'Real registry readback failed')
        after = identity()
        R.check(after == current, 'Selector execution changed source inputs or effective settings')
        uploads, status = 1, 0
    except BaseException as error:
        errors.append(str(error) or type(error).__name__)
        if isinstance(error, KeyboardInterrupt):
            status = 130
        raise
    finally:
        registry.close()
        seal(directory, 'passed' if not errors and status == 0 and uploads == 1 else 'failed',
             status or 1 if errors or uploads != 1 else status,
             errors, uploads)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    modes = parser.add_subparsers(dest='mode', required=True)
    modes.add_parser('prepare').add_argument('directory', type=Path)
    verdict = modes.add_parser('run')
    for name in ('directory', 'prepared', 'registry_source', 'images'):
        verdict.add_argument(name, type=Path)
    lookup = modes.add_parser('lookup-client')
    lookup.add_argument('expectation', type=Path)
    lookup.add_argument('output', type=Path)
    args = parser.parse_args()
    if args.mode == 'prepare':
        prepare(args.directory.resolve())
    elif args.mode == 'run':
        run(args.directory.resolve(), args.prepared.resolve(), args.registry_source.resolve(), args.images.resolve())
    else:
        lookup_client(args.expectation, args.output)
