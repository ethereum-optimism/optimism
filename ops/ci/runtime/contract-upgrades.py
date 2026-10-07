#!/usr/bin/env python3
"""Run the complete L1 contract upgrade matrices with bound original evidence."""
import argparse
import hashlib
import html
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
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET

SPEC = importlib.util.spec_from_file_location('ci_report', Path(__file__).with_name('ci-report.py'))
REPORT = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(REPORT)
SPEC = importlib.util.spec_from_file_location('submodules', Path(__file__).with_name('git-submodule-report.py'))
SUBMODULES = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(SUBMODULES)
ROOT = Path(__file__).resolve().parents[3]
CONTRACTS = ROOT / 'packages/contracts-bedrock'
VARIANTS = {'feature-main': ('op', 'main'), 'feature-CUSTOM_GAS_TOKEN': ('op', 'CUSTOM_GAS_TOKEN'),
            'feature-OPTIMISM_PORTAL_INTEROP': ('op', 'OPTIMISM_PORTAL_INTEROP'),
            'feature-ZK_DISPUTE_GAME': ('op', 'ZK_DISPUTE_GAME'),
            'chain-op': ('op', 'main'), 'chain-ink': ('ink', 'main'), 'chain-unichain': ('unichain', 'main')}
FEATURES = {'CUSTOM_GAS_TOKEN': 'SYS_FEATURE__CUSTOM_GAS_TOKEN',
            'OPTIMISM_PORTAL_INTEROP': 'DEV_FEATURE__OPTIMISM_PORTAL_INTEROP', 'ZK_DISPUTE_GAME': 'DEV_FEATURE__ZK_DISPUTE_GAME'}
MATCH = 'test/{L1,dispute,cannon}/**'


digest = REPORT.digest
write = REPORT.write


def command(*args, cwd=None):
    return REPORT.command(args, cwd=ROOT if cwd is None else cwd)


def revision():
    sha = command('git', 'rev-parse', 'HEAD')
    expected = os.environ.get('CI_COMMIT_SHA') or os.environ.get('CIRCLE_SHA1') or sha
    if not re.fullmatch('[0-9a-f]{40}', expected) or sha != expected: raise ValueError('Contract upgrade source revision differs')
    return sha


class Redactor:
    """Mask authentication data before streaming or saving RPC-dependent output."""
    def __init__(self, url):
        parsed = urllib.parse.urlsplit(url)
        if parsed.scheme not in ('http', 'https') or not parsed.hostname: raise ValueError('Invalid test-only RPC configuration')
        parts = [url, html.escape(url, quote=True), urllib.parse.quote(url, safe="")]
        parts += [v for _, v in urllib.parse.parse_qsl(parsed.query) if len(v) >= 8]
        parts += [v for v in (parsed.username, parsed.password) if v and len(v) >= 8]
        parts += [v for v in parsed.path.split('/') if len(v) >= 16]
        parts += [html.escape(v, quote=True) for v in parts] + [urllib.parse.quote(v, safe="") for v in parts]
        self.values = sorted({v.encode() for v in parts}, key=len, reverse=True)
    def __call__(self, data, replacement=b'<test-only-rpc>'):
        for value in self.values: data = data.replace(value, replacement)
        return data
    def xml(self, data):
        # XML text and attributes must remain parseable after authentication is
        # removed. The plain-log marker would introduce an unclosed XML tag.
        return self(data, b'&lt;test-only-rpc&gt;')


def stage(directory, name, argv, redactor=lambda x: x, json_output=False):
    return REPORT.stage(directory, name, argv, cwd=CONTRACTS, layout='split',
                        redactor=redactor, stdout_json=json_output)


def originals(directory, required, label):
    final = REPORT.read(directory / 'final.json')
    if final['exit_code'] != 0 or final['report_errors']:
        raise ValueError('Failed or incomplete original contract report: ' + label)
    return REPORT.verify_files(directory, final['original_sha256'], required=required,
                               label=label)


def rpc(url, method, params):
    body = json.dumps({'jsonrpc': '2.0', 'id': 1, 'method': method, 'params': params}).encode()
    for attempt in range(3):
        try:
            request = urllib.request.Request(url, data=body, headers={'Content-Type': 'application/json', 'User-Agent': 'Go-http-client/1.1'})
            with urllib.request.urlopen(request, timeout=30) as response: data = json.load(response)
            if data.get('error'):
                code = data['error'].get('code')
                raise ValueError('RPC error ' + str(code) if isinstance(code, int) else 'Invalid JSON-RPC response')
            if data.get('id') != 1 or 'result' not in data: raise ValueError('Invalid JSON-RPC response')
            return data['result']
        except (OSError, urllib.error.URLError, ValueError) as error:
            # Provider exceptions can include the URL or response body. Keep
            # only fixed categories and numeric codes, as in Go preflight.
            reason = 'HTTP ' + str(error.code) if isinstance(error, urllib.error.HTTPError) else str(error) if \
                isinstance(error, ValueError) and str(error).startswith(('RPC error ', 'Invalid JSON-RPC')) else type(error).__name__
            if attempt == 2: raise ValueError('Test-only archive RPC unavailable: ' + reason) from None
            time.sleep(2 ** attempt)


def block(url, number):
    chain_id = int(rpc(url, 'eth_chainId', []), 16)
    data = rpc(url, 'eth_getBlockByNumber', [hex(number), False])
    if chain_id != 1 or not data or int(data['number'], 16) != number or not re.fullmatch('0x[0-9a-fA-F]{64}', data['hash']):
        raise ValueError('Wrong archive chain, missing block or corrupt block identity')
    return {'chain_id': chain_id, 'number': number, 'hash': data['hash'].lower(), 'timestamp': int(data['timestamp'], 16)}


def pinned_block(directory, url, redactor):
    status = stage(directory, 'pinned-block', ['just', 'print-pinned-block-number'], redactor)
    if status: raise ValueError('Authoritative daily archive block discovery failed')
    text = (directory / 'pinned-block.log').read_text().strip()
    if not re.fullmatch('[0-9]+', text): raise ValueError('Invalid authoritative daily block number')
    data = block(url, int(text)); data['source_sha'] = revision(); data['policy'] = 'Just current-day 00:00 UTC'
    return data


def configure(variant):
    chain, feature = VARIANTS[variant]
    for name in list(os.environ):
        if name.startswith(('FOUNDRY_', 'DAPP_', 'DEV_FEATURE__', 'SYS_FEATURE__')): del os.environ[name]
    for name in ('ETH_RPC_URL', 'FORK_RPC_URL', 'FORK_BLOCK_NUMBER', 'L2_FORK_RPC_URL', 'L2_FORK_BLOCK_NUMBER',
                 'ETH_RPC_JWT', 'ETH_RPC_HEADERS', 'ETHERSCAN_API_KEY', 'MAINNET_RPC_URL'):
        os.environ.pop(name, None)
    branch = os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH')
    if not branch: raise ValueError('Missing tested contract branch')
    os.environ.update(FOUNDRY_PROFILE='ci' if branch == 'develop' else 'liteci',
                      FOUNDRY_FUZZ_SEED='42424242', FOUNDRY_FUZZ_RUNS='1', FORK_OP_CHAIN=chain,
                      FORK_BASE_CHAIN='mainnet', FORK_TEST='true', L2_FORK_TEST='false', L2CM_ACTIVATION_TEST='false', CI='true')
    if feature != 'main': os.environ[FEATURES[feature]] = 'true'
    return branch, chain, feature


def compiler_signatures(out):
    bindings = {}
    for path in sorted(out.rglob('*.json')):
        value = json.loads(path.read_text())
        targets = value.get('metadata', {}).get('settings', {}).get('compilationTarget', {})
        for source, contract in targets.items():
            if not source.startswith('test/'): continue
            methods = value.get('methodIdentifiers')
            if not isinstance(methods, dict): raise ValueError('Missing compiler test method identifiers')
            bytecode = value.get('bytecode', {}).get('object')
            if not isinstance(bytecode, str): raise ValueError('Missing compiler test creation bytecode')
            bytecode = bytecode.removeprefix('0x')
            if len(bytecode) % 2: raise ValueError('Invalid compiler test creation bytecode')
            creation = {'bytes': len(bytecode) // 2, 'sha256': hashlib.sha256(bytecode.encode()).hexdigest()}
            key = source + ':' + contract
            row = bindings.setdefault(key, {'methods': methods, 'creation_bytecode': {}, 'artifacts': {}})
            if row['methods'] != methods: raise ValueError('Conflicting compiler test signatures: ' + key)
            relative = str(path.relative_to(ROOT))
            row['creation_bytecode'][relative] = creation
            row['artifacts'][relative] = digest(path)
    return bindings


def selection(value, bindings=None):
    cases = []
    if not isinstance(value, dict): raise ValueError('Invalid complete Forge discovery')
    for path, contracts in value.items():
        if not isinstance(contracts, dict): raise ValueError('Invalid Forge contract discovery')
        for contract, tests in contracts.items():
            if not isinstance(tests, list) or any(not isinstance(t, str) for t in tests): raise ValueError('Invalid Forge test discovery')
            identity = path + ':' + contract
            if bindings is None:
                if any('(' not in test for test in tests): raise ValueError('Missing original compiler test signatures')
                cases.extend((identity, test) for test in tests)
            else:
                methods = bindings[identity]['methods']
                discovered = set(tests)
                selected = {name for name in methods if name.split('(', 1)[0] in discovered}
                if {name.split('(', 1)[0] for name in selected} != discovered:
                    raise ValueError('Original compiler signatures differ from Forge discovery')
                cases.extend((identity, test) for test in sorted(selected))
    if not cases or len(cases) != len(set(cases)): raise ValueError('Empty or duplicate upgrade test discovery')
    return sorted(cases)


def deployable(binding):
    creation = binding['creation_bytecode']
    if not creation or set(creation) != set(binding['artifacts']): raise ValueError('Compiler bytecode artifact binding differs')
    for row in creation.values():
        if type(row['bytes']) is not int or row['bytes'] < 0 or not re.fullmatch('[0-9a-f]{64}', row['sha256']):
            raise ValueError('Invalid compiler creation bytecode binding')
        if row['bytes'] == 0 and row['sha256'] != hashlib.sha256(b'').hexdigest():
            raise ValueError('Empty compiler bytecode hash differs')
    return any(row['bytes'] > 0 for row in creation.values())


def junit(path, discovered, bindings=None):
    selected = {tuple(r) for r in discovered}
    if not selected or len(selected) != len(discovered): raise ValueError('Empty or duplicate upgrade selection')
    non_executable = set()
    if bindings is not None:
        for identity, _ in selected:
            if not deployable(bindings[identity]):
                non_executable.update(r for r in selected if r[0] == identity)
    actual = {}; outcomes = {}; setup_skips = {}
    for suite in ET.parse(path).iter('testsuite'):
        for case in suite.findall('testcase'):
            key = (case.get('classname') or suite.get('name', ''), case.attrib['name'])
            if key in actual: raise ValueError('Duplicate original upgrade verdict')
            state = 'fail' if case.find('failure') is not None or case.find('error') is not None else 'skip' if case.find('skipped') is not None else 'pass'
            skipped = case.find('skipped')
            actual[key] = {'outcome': state, 'skip_reason': {'attributes': skipped.attrib, 'text': ''.join(skipped.itertext())} if state == 'skip' else None}
    original = [{'class': c, 'name': n, **actual[c, n]} for c, n in sorted(actual)]
    # Forge records one skipped setUp() instead of individual test verdicts
    # when a feature or fork guard skips the whole deployable contract.
    for key in set(actual) - selected:
        identity, name = key
        members = {r for r in selected - non_executable if r[0] == identity}
        if name != 'setUp()' or actual[key]['outcome'] != 'skip' or not members or any(r in actual for r in members):
            raise ValueError('Extra, failed or conflicting original upgrade setup verdict')
        value = actual.pop(key); setup_skips[identity] = value
        for member in members: actual[member] = value
    if set(actual) != selected - non_executable: raise ValueError('Missing or extra original upgrade verdict')
    for value in actual.values(): outcomes[value['outcome']] = outcomes.get(value['outcome'], 0) + 1
    if not outcomes.get('pass') or outcomes.get('fail'): raise ValueError('No successful complete upgrade execution')
    return {'cases': [{'class': c, 'name': n, 'verdict_source': 'setUp()' if c in setup_skips else n,
                       **actual[c, n]} for c, n in sorted(actual)],
            'non_executable': [{'class': c, 'name': n, 'reason': 'empty compiler creation bytecode'} for c, n in sorted(non_executable)],
            'original_cases': original, 'suite_setup_skips': setup_skips, 'outcomes': outcomes, 'retries': 0}


def inputs():
    paths = command('git', 'ls-files').splitlines()
    helpers = ['ops/ci/runtime/contract-upgrades.py', 'ops/ci/runtime/git-submodule-report.py', 'ops/ci/runtime/ci-report.py']
    for helper in helpers:
        if helper not in paths: paths.append(helper)
    chosen = [p for p in paths if p.startswith('packages/contracts-bedrock/') or p in helpers + ['mise.toml', 'go.mod', 'go.sum']]
    return {p: digest(ROOT / p) for p in chosen if (ROOT / p).is_file()}


def prepare(directory, variant):
    branch, chain, feature = configure(variant)
    settings = {'source_sha': revision(), 'variant': variant, 'branch': branch, 'chain': chain, 'feature': feature,
                'profile': os.environ['FOUNDRY_PROFILE'], 'fuzz_seed': '42424242', 'fuzz_runs': 1,
                'provider': os.environ.get('CI_CONTRACT_PROVIDER', 'circleci'), 'workspace_root': str(ROOT),
                'forge': command('forge', '--version'), 'go': command('go', 'version'), 'just': command('just', '--version'),
                'input_sha256': inputs(), 'rwx_run_id': os.environ.get('RWX_RUN_ID'),
                'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}
    write(directory / 'settings.json', settings)
    (directory / 'submodules.txt').write_text(command('git', 'submodule', 'status', '--recursive') + '\n')
    for name, argv, as_json in [('foundry-config', ['forge', 'config', '--json'], True),
                                ('go-ffi', ['just', 'build-go-ffi'], False),
                                ('contracts-build', ['forge', 'build'], False),
                                ('discovery', ['forge', 'test', '--list', '--json', '--match-path', MATCH], True)]:
        status = stage(directory, name, argv, json_output=as_json)
        if status: return status
    config = json.loads((directory / 'foundry-config.json').read_text())
    out = Path(config['out']); out = out if out.is_absolute() else CONTRACTS / out
    signatures = compiler_signatures(out); write(directory / 'signature-bindings.json', signatures)
    discovered = selection(json.loads((directory / 'discovery.json').read_text()), signatures)
    write(directory / 'selection.json', discovered)
    if config['fuzz']['runs'] != 1 or int(config['fuzz']['seed'], 16) != 42424242 or any(config.get(k) for k in
        ('match_test', 'no_match_test', 'match_contract', 'no_match_contract', 'match_path', 'no_match_path', 'skip')):
        raise ValueError('Unexpected effective upgrade workload settings')
    compiled = {}
    for name in ('forge-artifacts', 'artifacts/build-info', 'cache/solidity-files-cache.json', 'scripts/go-ffi/go-ffi'):
        path = CONTRACTS / name
        for file in ([path] if path.is_file() else sorted(path.rglob('*'))):
            if file.is_file(): compiled[str(file.relative_to(ROOT))] = digest(file)
    if 'packages/contracts-bedrock/scripts/go-ffi/go-ffi' not in compiled or not any('/forge-artifacts/' in p for p in compiled):
        raise ValueError('Missing complete upgrade compiler outputs')
    write(directory / 'compiled.json', compiled)
    return 0


def finish(directory, status, errors):
    write(directory / 'final.json', {'exit_code': status, 'report_errors': errors,
          'original_sha256': REPORT.file_hashes(directory, exclude=('final.json',))})
    return status


def main():
    os.chdir(ROOT); parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('mode', choices=('preflight', 'prepare', 'run')); parser.add_argument('--variant', choices=VARIANTS)
    parser.add_argument('--prepared', type=Path); parser.add_argument('--block', type=Path); args = parser.parse_args()
    if args.mode != 'preflight' and not args.variant: parser.error('--variant is required')
    directory = ROOT / '.ci/contract-upgrades' / ('preflight' if args.mode == 'preflight' else args.variant)
    if args.mode != 'preflight': directory /= args.mode
    shutil.rmtree(directory, ignore_errors=True); directory.mkdir(parents=True)
    status, errors = 1, []
    try:
        url = os.environ.get('OP_CI_MAINNET_L1_ARCHIVE_RPC_URL')
        redactor = Redactor(url) if url else lambda x: x
        redact_xml = redactor.xml if url else lambda x: x
        if args.mode == 'preflight':
            if not url: raise ValueError('Missing test-only archive RPC')
            os.environ['ETH_RPC_URL'] = url; write(directory / 'block.json', pinned_block(directory, url, redactor)); status = 0
        else:
            if args.prepared:
                required = {'settings.json', 'selection.json', 'foundry-config.json', 'foundry-config.stage.json',
                            'contracts-build.stage.json', 'discovery.json', 'discovery.stage.json', 'go-ffi.stage.json',
                            'submodules.txt', 'compiled.json', 'signature-bindings.json'}
                originals(args.prepared, required, 'rwx/upgrade-prepare')
                old = json.loads((args.prepared / 'settings.json').read_text())
                if old['source_sha'] != revision() or old['variant'] != args.variant or old['input_sha256'] != inputs():
                    raise ValueError('Stale or mismatched contract upgrade compilation inputs')
                if any(old[k] != command(*argv) for k, argv in [('forge', ['forge', '--version']), ('go', ['go', 'version']), ('just', ['just', '--version'])]):
                    raise ValueError('Contract upgrade runtime toolchain differs from compilation')
                if SUBMODULES.revisions((args.prepared / 'submodules.txt').read_text()) != SUBMODULES.revisions(command('git', 'submodule', 'status', '--recursive')):
                    raise ValueError('Contract upgrade runtime submodules differ from compilation')
                for name, hashed in json.loads((args.prepared / 'compiled.json').read_text()).items():
                    path = ROOT / name
                    if not name.startswith('packages/contracts-bedrock/') or '..' in Path(name).parts or digest(path) != hashed:
                        raise ValueError('Missing or corrupt contract upgrade compiler output')
                for p in args.prepared.iterdir():
                    if p.is_file() and p.name != 'final.json': shutil.copy2(p, directory / p.name)
                branch, chain, feature = configure(args.variant)
                if (old['branch'], old['chain'], old['feature'], old['profile'], old['fuzz_runs'], old['fuzz_seed']) != (
                    branch, chain, feature, os.environ['FOUNDRY_PROFILE'], 1, '42424242') or old['workspace_root'] != str(ROOT):
                    raise ValueError('Contract upgrade compiled settings differ from runtime')
                old.update(rwx_run_id=os.environ.get('RWX_RUN_ID'), rwx_task_attempt=os.environ.get('RWX_TASK_ATTEMPT_NUMBER'))
                write(directory / 'settings.json', old); status = 0
            else: status = prepare(directory, args.variant)
            if args.mode == 'run' and status == 0:
                if not url: raise ValueError('Missing test-only archive RPC')
                os.environ['ETH_RPC_URL'] = url
                chosen = json.loads(args.block.read_text()) if args.block else pinned_block(directory, url, redactor)
                if chosen['source_sha'] != revision() or block(url, chosen['number']) != {k:chosen[k] for k in ('chain_id','number','hash','timestamp')}:
                    raise ValueError('Stale, unavailable or mismatched archive block')
                write(directory / 'block.json', chosen); os.environ['FORK_BLOCK_NUMBER'] = str(chosen['number'])
                os.environ['JUNIT_TEST_PATH'] = str(directory / 'original.junit.xml')
                for path in ('cache/test-failures', 'cache/fuzz', 'cache/invariant'):
                    p = CONTRACTS / path
                    if p.is_dir(): shutil.rmtree(p)
                    elif p.exists(): p.unlink()
                status = stage(directory, 'tests', ['just', 'test-upgrade'], redactor)
                # Authentication strings can occur in failing test reasons.
                if (p := directory / 'original.junit.xml').exists(): p.write_bytes(redact_xml(p.read_bytes()))
                if status and status < 128:
                    os.environ['JUNIT_TEST_PATH'] = str(directory / 'diagnostic.junit.xml')
                    stage(directory, 'rerun', ['just', 'test-upgrade-rerun'], redactor)
                elif status == 0: write(directory / 'coverage.json', junit(directory / 'original.junit.xml',
                    json.loads((directory / 'selection.json').read_text()), json.loads((directory / 'signature-bindings.json').read_text())))
            for xml in directory.glob('*.xml'): xml.write_bytes(redact_xml(xml.read_bytes()))
            settings = json.loads((directory / 'settings.json').read_text())
            if settings['input_sha256'] != inputs(): raise ValueError('Contract upgrade source changed during execution')
    except (OSError, ValueError, KeyError, ET.ParseError, subprocess.CalledProcessError) as error:
        # RPC exceptions use safe messages; never print request objects or URLs.
        errors.append(str(error)); print(error, file=sys.stderr); status = 1
    return finish(directory, status, errors)


if __name__ == '__main__': sys.exit(main())
