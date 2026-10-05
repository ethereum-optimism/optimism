#!/usr/bin/env python3
"""Prepare and execute the complete OP Mainnet L2 fork workload freshly."""
import argparse
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
import urllib.error
import urllib.request


def helper(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


UP = helper('contract-upgrades')
R = helper('selector-registry')
ROOT, CONTRACTS = UP.ROOT, UP.CONTRACTS
check, digest, write = R.check, R.digest, R.write
RPC = 'https://mainnet.optimism.io'
MATCH = 'test/L2/fork/**'
IMPLEMENTATION = ('contract-l2-fork.py', 'contract-upgrades.py', 'selector-registry.py',
                  'git-submodule-report.py', 'compare-rust-e2e.py')
ADDRESS = '0x4200000000000000000000000000000000000007'
SLOT = '0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc'


def read(path):
    def pairs(items):
        result = {}
        for key, value in items:
            check(key not in result, 'Duplicate original L2 JSON key'); result[key] = value
        return result
    return json.loads(Path(path).read_bytes(), object_pairs_hook=pairs,
        parse_constant=lambda _: (_ for _ in ()).throw(ValueError('Non-finite original L2 JSON')))


def files(directory):
    result = {}
    for path in sorted(directory.rglob('*')):
        check(not path.is_symlink(), 'Linked L2 evidence')
        if path.is_file() and path != directory / 'final.json': result[str(path.relative_to(directory))] = digest(path)
    return result


def seal(directory, status, errors, tests=0):
    write(directory / 'final.json', {'exit_code': status, 'errors': errors, 'tests': tests, 'sha256': files(directory)})


def original(directory):
    final = read(directory / 'final.json')
    check(type(final['exit_code']) is int and final['exit_code'] == 0 and final['errors'] == []
          and type(final['tests']) is int and final['tests'] >= 0 and final['sha256'] == files(directory),
          'Missing, failed or corrupt original L2 evidence')
    return final


def call(directory, method, params):
    number = len(list(directory.glob('rpc-*-request.json')))
    name = 'rpc-' + str(number)
    request = {'jsonrpc': '2.0', 'id': number + 1, 'method': method, 'params': params}
    write(directory / (name + '-request.json'), request)
    for attempt in range(1, 4):
        started = time.time(); status = None; raw = b''; error = None
        try:
            req = urllib.request.Request(RPC, data=json.dumps(request).encode(),
                headers={'Content-Type': 'application/json', 'User-Agent': 'Go-http-client/1.1'})
            with urllib.request.urlopen(req, timeout=30) as response: status, raw = response.status, response.read()
        except urllib.error.HTTPError as failure:
            try: status, raw, error = failure.code, failure.read(), 'HTTP ' + str(failure.code)
            finally: failure.close()
        except OSError as failure: error = type(failure).__name__
        path = directory / (name + '-attempt-' + str(attempt) + '.json'); path.write_bytes(raw)
        write(path.with_suffix('.metadata.json'), {'url': RPC, 'http_status': status, 'error': error,
              'started_at': started, 'elapsed_seconds': time.time() - started, 'response_sha256': digest(path)})
        if status == 200:
            value = read(path)
            check(value.get('jsonrpc') == '2.0' and value.get('id') == request['id'] and 'result' in value
                  and 'error' not in value, 'Invalid original public L2 RPC response')
            return value['result']
        if status in (400, 401, 403, 404) or attempt == 3: raise ValueError('Public L2 RPC unavailable: ' + str(status or error))
        time.sleep(2 ** attempt)


def block(directory, number):
    chain = call(directory, 'eth_chainId', [])
    check(isinstance(chain, str) and chain == '0xa', 'Wrong L2 fork chain')
    data = call(directory, 'eth_getBlockByNumber', [hex(number), False])
    check(isinstance(data, dict) and data.get('number') == hex(number)
          and re.fullmatch('0x[0-9a-f]{64}', data.get('hash', '')), 'Missing or mismatched public L2 block')
    return {'chain_id': 10, 'number': number, 'hash': data['hash'], 'timestamp': int(data['timestamp'], 16)}


def preflight(directory, requested):
    check(requested == 'latest' or re.fullmatch('[1-9][0-9]*', requested), 'Invalid selected L2 fork height')
    number = int(call(directory, 'eth_blockNumber', []), 16) if requested == 'latest' else int(requested)
    value = block(directory, number)
    code = call(directory, 'eth_getCode', [ADDRESS, hex(number)])
    storage = call(directory, 'eth_getStorageAt', [ADDRESS, SLOT, hex(number)])
    version = call(directory, 'eth_call', [{'to': ADDRESS, 'data': '0x54fd4d50'}, hex(number)])
    check(isinstance(code, str) and re.fullmatch('0x(?:[0-9a-f]{2})+', code)
          and isinstance(storage, str) and re.fullmatch('0x[0-9a-f]{64}', storage) and int(storage, 16) > 0
          and isinstance(version, str) and re.fullmatch('0x(?:[0-9a-f]{2})+', version), 'Public L2 state unavailable')
    value.update(source_sha=UP.revision(), rpc_url=RPC, policy='pinned original OP Mainnet L2 fork block', requested=requested)
    write(directory / 'block.json', value)
    return value


def configure():
    for name in list(os.environ):
        if name.startswith(('FOUNDRY_', 'DAPP_', 'DEV_FEATURE__', 'SYS_FEATURE__', 'FORK_', 'L2_FORK_', 'L2CM_')):
            del os.environ[name]
    for name in ('ETH_RPC_URL', 'ETH_RPC_JWT', 'ETH_RPC_HEADERS', 'ETHERSCAN_API_KEY', 'MAINNET_RPC_URL', 'L2_BLOCK_BEFORE_FORK', 'NUT_BUNDLE_PATH'):
        os.environ.pop(name, None)
    branch = os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH')
    check(branch, 'Missing L2 tested branch')
    os.environ.update(FOUNDRY_PROFILE='ci', FORK_TEST='false', L2_FORK_TEST='true', L2CM_ACTIVATION_TEST='false', CI='true')
    return branch


def settings():
    tools = {}
    for name in ('forge', 'cast', 'go', 'just'):
        binary = shutil.which(name); check(binary, 'Missing L2 runtime tool: ' + name)
        tools[name] = {'sha256': digest(binary), 'version': UP.command(name, 'version' if name == 'go' else '--version')}
    sha = UP.revision()
    return {'source_sha': sha, 'branch': configure(), 'profile': 'ci', 'feature': 'main', 'chain': 'op-mainnet',
            'match_path': MATCH, 'provider': os.environ.get('CI_CONTRACT_PROVIDER', 'circleci'), 'workspace_root': str(ROOT),
            'tools': tools, 'inputs': R.source_inputs(ROOT, sha, True),
            'implementation': {name: digest(Path(__file__).with_name(name)) for name in IMPLEMENTATION},
            'rwx_run_id': os.environ.get('RWX_RUN_ID'), 'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}


def effective(config):
    check(config['fuzz']['runs'] == 128 and config['invariant']['runs'] == 64 and config['invariant']['depth'] == 32
          and all(not config.get(k) for k in ('match_test', 'no_match_test', 'match_contract', 'no_match_contract', 'match_path', 'no_match_path', 'skip')),
          'Changed complete original L2 workload settings')


def prepare(directory):
    configure(); identity = settings(); write(directory / 'settings.json', identity)
    (directory / 'submodules.txt').write_text(UP.command('git', 'submodule', 'status', '--recursive') + '\n')
    for name, argv, as_json in [('foundry-config', ['forge', 'config', '--json'], True),
          ('go-ffi', ['just', 'build-go-ffi'], False), ('contracts-build', ['forge', 'build'], False),
          ('discovery', ['forge', 'test', '--list', '--json', '--match-path', MATCH], True)]:
        status = UP.stage(directory, name, argv, json_output=as_json)
        if status: return status
    config = read(directory / 'foundry-config.json'); effective(config)
    out = Path(config['out']); out = out if out.is_absolute() else CONTRACTS / out
    bindings = UP.compiler_signatures(out)
    selected = UP.selection(read(directory / 'discovery.json'), bindings)
    check(all(path.startswith('test/L2/fork/') for path in read(directory / 'discovery.json')), 'Foreign L2 discovery path')
    bindings = {identity: bindings[identity] for identity in sorted({row[0] for row in selected})}
    write(directory / 'signature-bindings.json', bindings); write(directory / 'selection.json', selected)
    compiled = {}
    for name in ('forge-artifacts', 'artifacts/build-info', 'cache/solidity-files-cache.json', 'scripts/go-ffi/go-ffi'):
        path = CONTRACTS / name
        for file in ([path] if path.is_file() else sorted(path.rglob('*'))):
            if file.is_file(): compiled[str(file.relative_to(ROOT))] = digest(file)
    check('packages/contracts-bedrock/scripts/go-ffi/go-ffi' in compiled and any('/forge-artifacts/' in p for p in compiled),
          'Missing original L2 compiler outputs')
    write(directory / 'compiled.json', compiled)
    for binding in bindings.values():
        for name in binding['artifacts']:
            target = directory / 'compiler-artifacts' / name; target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(ROOT / name, target)
    return 0


def restored(directory, prepared):
    original(prepared); previous = read(prepared / 'settings.json'); current = settings()
    check(all(previous[key] == current[key] for key in current if key not in ('rwx_run_id', 'rwx_task_attempt')),
          'Stale or mismatched L2 compiler inputs, settings or tools')
    check(UP.SUBMODULES.revisions((prepared / 'submodules.txt').read_text())
          == UP.SUBMODULES.revisions(UP.command('git', 'submodule', 'status', '--recursive')), 'Mismatched L2 source submodules')
    for name, hashed in read(prepared / 'compiled.json').items():
        check(name.startswith('packages/contracts-bedrock/') and '..' not in Path(name).parts and not Path(name).is_absolute()
              and digest(ROOT / name) == hashed, 'Missing or corrupt L2 compiled output')
    for path in prepared.iterdir():
        if path.name == 'final.json': continue
        if path.is_dir(): shutil.copytree(path, directory / path.name)
        else: shutil.copyfile(path, directory / path.name)
    write(directory / 'settings.json', current); effective(read(directory / 'foundry-config.json'))


def verdict(directory, pinned):
    original(pinned.parent)
    chosen = read(pinned)
    check(chosen['source_sha'] == UP.revision() and chosen['rpc_url'] == RPC and chosen['chain_id'] == 10
          and chosen['policy'] == 'pinned original OP Mainnet L2 fork block' and type(chosen['number']) is int and chosen['number'] > 0,
          'Stale or mismatched L2 preflight identity')
    check(block(directory, chosen['number']) == {k: chosen[k] for k in ('chain_id', 'number', 'hash', 'timestamp')},
          'Changed original L2 block')
    shutil.copytree(pinned.parent, directory / 'preflight')
    write(directory / 'block.json', chosen)
    os.environ.update(L2_FORK_RPC_URL=RPC, L2_FORK_BLOCK_NUMBER=str(chosen['number']), JUNIT_TEST_PATH=str(directory / 'original.junit.xml'))
    (CONTRACTS / 'l2ForkBlockNumber.txt').write_text(str(chosen['number']) + '\n')
    for name in ('cache/test-failures', 'cache/fuzz', 'cache/invariant'):
        path = CONTRACTS / name
        if path.is_dir(): shutil.rmtree(path)
        elif path.exists(): path.unlink()
    status = UP.stage(directory, 'nut-bundle-check', ['just', 'nut-bundle-check-no-build'])
    if status: return status, 0
    status = UP.stage(directory, 'tests', ['just', 'test-l2-fork-upgrade'])
    if status and status < 128:
        os.environ['JUNIT_TEST_PATH'] = str(directory / 'diagnostic.junit.xml')
        UP.stage(directory, 'rerun', ['just', 'test-l2-fork-upgrade-rerun'])
    if status: return status, 0
    coverage = UP.junit(directory / 'original.junit.xml', read(directory / 'selection.json'), read(directory / 'signature-bindings.json'))
    write(directory / 'coverage.json', coverage)
    return 0, len(coverage['original_cases'])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('preflight', 'prepare', 'run'))
    parser.add_argument('--fork-block', default='latest'); parser.add_argument('--prepared', type=Path); parser.add_argument('--block', type=Path)
    args = parser.parse_args()
    if args.prepared: args.prepared = args.prepared.resolve()
    if args.block: args.block = args.block.resolve()
    os.chdir(ROOT)
    directory = ROOT / '.ci/contract-l2-fork' / args.mode
    check(not directory.exists(), 'L2 verdict or preparation already exists'); directory.mkdir(parents=True)
    status, errors, tests = 1, [], 0
    try:
        if args.mode == 'preflight': preflight(directory, args.fork_block); status = 0
        else:
            configure()
            if args.prepared: restored(directory, args.prepared.resolve()); status = 0
            else: status = prepare(directory)
            if args.mode == 'run' and status == 0:
                check(args.block, 'Missing complete L2 preflight artifact')
                status, tests = verdict(directory, args.block.resolve())
            if (directory / 'settings.json').exists():
                check(read(directory / 'settings.json')['inputs'] == R.source_inputs(ROOT, UP.revision(), True), 'Source changed during L2 execution')
    except (OSError, ValueError, KeyError, UP.ET.ParseError, subprocess.CalledProcessError) as error:
        errors.append(str(error)); print(error, file=sys.stderr); status = 1
    finally: seal(directory, status, errors, tests)
    return status


if __name__ == '__main__': sys.exit(main())
