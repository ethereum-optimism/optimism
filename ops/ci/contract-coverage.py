#!/usr/bin/env python3
"""Run both original contract coverage passes with separately sealed reports."""
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
import xml.etree.ElementTree as ET

SPEC = importlib.util.spec_from_file_location('suites', Path(__file__).with_name('contract-suites.py'))
CS = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(CS)
UP, ROOT, CONTRACTS = CS.UP, CS.ROOT, CS.CONTRACTS
PHASES = ('ordinary', 'upgrade')
UPGRADE_CONTRACT = 'OPContractsManager.*_Upgrade_Test'
RPC_INPUT = 'OP_CI_MAINNET_L1_ARCHIVE_RPC_URL'
FFI_REPLAY_INPUT = 'OP_CI_FFI_REPLAY_SEED'
DISCOVERY = {'ordinary': ['forge', 'test', '--list', '--json'],
             'upgrade': ['forge', 'test', '--list', '--json', '--match-contract', UPGRADE_CONTRACT, '--match-path', UP.MATCH]}
# These are the two commands executed, in this order and with short-circuiting,
# by coverage-lcov-all. Attribution adds reporting only. Foundry ignores
# --report-file for multiple file reports, so each pass's lcov.info is moved to
# its original public filename before the next pass can overwrite it.
COMMANDS = {'ordinary': ['just', 'coverage-lcov', '--report', 'attribution'],
            'upgrade': ['just', 'coverage-lcov-upgrade', '--match-contract', UPGRADE_CONTRACT, '--report', 'attribution']}
PREPARED = {'settings.json', 'files.log', 'files.stage.json', 'file-selection.json', 'submodules.txt',
            'submodules-sync.stage.json', 'submodules-init.stage.json', 'build-config.json', 'build-config.stage.json',
            'go-ffi.stage.json', 'source-build.stage.json', 'coverage-build.stage.json', 'foundry-config.json', 'foundry-config.stage.json',
            'ordinary-discovery.json', 'ordinary-discovery.stage.json', 'upgrade-discovery.json', 'upgrade-discovery.stage.json',
            'selection.json', 'signature-bindings.json', 'compiled.json', 'compile-only.json', 'source-after-preparation.json'}


def configure(feature):
    if feature not in CS.FEATURES: raise ValueError('Unknown contract coverage feature')
    branch = os.environ.get('CI_BRANCH') or os.environ.get('CIRCLE_BRANCH')
    if not branch: raise ValueError('Missing tested coverage branch')
    replay = os.environ.get('CI_CONTRACT_COVERAGE_REPLAY', 'false').lower()
    if replay not in ('false', 'true', '0', '1'): raise ValueError('Invalid coverage replay benchmark parameter')
    if os.environ.get('CI_CONTRACT_PROFILE', 'cicoverage') != 'cicoverage':
        raise ValueError('Coverage profile differs from the original workflow matrix')
    for name in list(os.environ):
        if name.startswith(('FOUNDRY_', 'DAPP_', 'DEV_FEATURE__', 'SYS_FEATURE__')): del os.environ[name]
    for name in ('ETH_RPC_URL', 'FORK_RPC_URL', 'FORK_BLOCK_NUMBER', 'L2_FORK_RPC_URL', 'L2_FORK_BLOCK_NUMBER',
                 'ETH_RPC_JWT', 'ETH_RPC_HEADERS', 'ETHERSCAN_API_KEY', 'MAINNET_RPC_URL', FFI_REPLAY_INPUT):
        os.environ.pop(name, None)
    seed = '0x' + hashlib.sha256((UP.revision() + ':' + feature + ':contract-coverage').encode()).hexdigest() \
           if replay in ('true', '1') else None
    os.environ.update(FORK_TEST='false', L2_FORK_TEST='false', L2CM_ACTIVATION_TEST='false', CI='true', NO_COLOR='1')
    if seed: os.environ[FFI_REPLAY_INPUT] = seed
    if feature != 'main': os.environ[UP.FEATURES[feature]] = 'true'
    return branch, seed


def coverage_profile(seed):
    os.environ['FOUNDRY_PROFILE'] = 'cicoverage'
    if seed: os.environ['FOUNDRY_FUZZ_SEED'] = seed


def validate_config(config, seed):
    if (config['fuzz']['runs'], config['invariant']['runs'], config['invariant']['depth']) != (1, 1, 1) or \
       config.get('optimizer') is not False or config.get('compilation_restrictions') or config.get('threads') != 16 or \
       any(config.get(k) for k in ('match_test', 'no_match_test', 'match_contract', 'no_match_contract', 'match_path', 'no_match_path', 'skip')):
        raise ValueError('Unexpected effective original coverage workload settings')
    if (int(config['fuzz']['seed'], 16) if config['fuzz'].get('seed') else None) != (int(seed, 16) if seed else None):
        raise ValueError('Coverage benchmark seed differs from effective configuration')


def prepare(directory, feature):
    branch, seed = configure(feature)
    settings = {'source_sha': UP.revision(), 'feature': feature, 'branch': branch, 'profile': 'cicoverage',
                'source_build_profile': 'default', 'benchmark_seed': seed, 'rpc_input_name': RPC_INPUT,
                'ffi_replay_seed': seed, 'ffi_replay_policy': 'chacha8-arguments-v1',
                'authority': ['just', 'coverage-lcov-all'], 'phases': PHASES,
                'provider': os.environ.get('CI_CONTRACT_PROVIDER', 'circleci'), 'workspace_root': str(ROOT),
                'forge': UP.command('forge', '--version'), 'go': UP.command('go', 'version'), 'just': UP.command('just', '--version'),
                'input_sha256': CS.inputs(), 'rwx_run_id': os.environ.get('RWX_RUN_ID'),
                'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER')}
    UP.write(directory / 'settings.json', settings)
    for name, argv in [('submodules-sync', ['git', '-C', str(ROOT), 'submodule', 'sync', '--recursive']),
                       ('submodules-init', ['git', '-C', str(ROOT), '-c', 'protocol.file.allow=never', 'submodule',
                                            'update', '--init', '--recursive', '--jobs', '8']),
                       ('files', CS.FILE_COMMANDS['standard'])]:
        status = UP.stage(directory, name, argv)
        if status: return status
    files = CS.file_selection((directory / 'files.log').read_text())
    if not files: raise ValueError('Empty complete coverage file selection')
    settings['runtime_output_paths'] = CS.runtime_outputs(files); UP.write(directory / 'settings.json', settings)
    UP.write(directory / 'file-selection.json', {'files': files, 'partitions': [{'index': 0, 'files': files}]})
    for name in settings['runtime_output_paths']:
        source = ROOT / name
        if source.is_symlink() or UP.digest(source) != settings['input_sha256'].get(name):
            raise ValueError('Unbound original coverage runtime fixture')
        target = directory / 'tracked-fixtures/before' / name; target.parent.mkdir(parents=True, exist_ok=True); shutil.copy2(source, target)
    (directory / 'submodules.txt').write_text(UP.command('git', 'submodule', 'status', '--recursive') + '\n')
    UP.SUBMODULES.revisions((directory / 'submodules.txt').read_text())
    # Circle builds source before setting the coverage-only profile on its
    # verdict step. Preserve this default-profile prerequisite separately.
    for name, argv, as_json in [('build-config', ['forge', 'config', '--json'], True),
                              ('go-ffi', ['just', 'build-go-ffi'], False), ('source-build', ['just', 'build-source'], False)]:
        status = UP.stage(directory, name, argv, json_output=as_json)
        if status: return status
    coverage_profile(seed)
    status = UP.stage(directory, 'foundry-config', ['forge', 'config', '--json'], json_output=True)
    if status: return status
    config = json.loads((directory / 'foundry-config.json').read_text()); validate_config(config, seed)
    # test --list emits minimal artifacts when tests have not been fully built;
    # they omit compiler method identifiers and metadata. A real build supplies
    # the authoritative signature/abstract-bytecode bindings without tests.
    status = UP.stage(directory, 'coverage-build', ['forge', 'build'])
    if status: return status
    for phase, argv in DISCOVERY.items():
        status = UP.stage(directory, phase + '-discovery', argv, json_output=True)
        if status: return status
    out = Path(config['out']); out = out if out.is_absolute() else CONTRACTS / out
    bindings = UP.compiler_signatures(out); UP.write(directory / 'signature-bindings.json', bindings)
    selected = {phase: UP.selection(json.loads((directory / (phase + '-discovery.json')).read_text()), bindings) for phase in PHASES}
    if any(identity.rsplit(':', 1)[0] not in files for cases in selected.values() for identity, _ in cases):
        raise ValueError('Coverage discovery selected a file outside its complete manifest')
    UP.write(directory / 'selection.json', selected)
    compiled = {}
    for name in ('forge-artifacts', 'artifacts/build-info', 'cache/solidity-files-cache.json', 'scripts/go-ffi/go-ffi'):
        p = CONTRACTS / name
        for f in ([p] if p.is_file() else sorted(p.rglob('*'))):
            if f.is_file(): compiled[str(f.relative_to(ROOT))] = UP.digest(f)
    if 'packages/contracts-bedrock/scripts/go-ffi/go-ffi' not in compiled or not any('/forge-artifacts/' in p for p in compiled):
        raise ValueError('Missing coverage compiler outputs')
    UP.write(directory / 'compiled.json', compiled)
    UP.write(directory / 'compile-only.json', {'tests': 0, 'selected_cases': {phase: len(cases) for phase, cases in selected.items()}})
    CS.verify_inputs(directory, settings['input_sha256'], 'preparation')
    return 0


def restore(directory, prepared, feature):
    UP.originals(prepared, PREPARED, [], 'rwx/contract-coverage-prepare')
    old = json.loads((prepared / 'settings.json').read_text()); branch, seed = configure(feature)
    files = CS.file_selection((prepared / 'files.log').read_text())
    if (old['source_sha'], old['feature'], old['branch'], old['profile'], old['benchmark_seed'], old['workspace_root'], old['input_sha256'],
        old.get('runtime_output_paths')) != \
       (UP.revision(), feature, branch, 'cicoverage', seed, str(ROOT), CS.inputs(), CS.runtime_outputs(files)):
        raise ValueError('Stale coverage source, profile, benchmark seed or runtime fixture role')
    for key, argv in [('forge', ['forge', '--version']), ('go', ['go', 'version']), ('just', ['just', '--version'])]:
        if old[key] != UP.command(*argv): raise ValueError('Coverage runtime toolchain differs from preparation')
    if UP.SUBMODULES.revisions((prepared / 'submodules.txt').read_text()) != UP.SUBMODULES.revisions(UP.command('git', 'submodule', 'status', '--recursive')):
        raise ValueError('Coverage runtime submodules differ from preparation')
    compiled = json.loads((prepared / 'compiled.json').read_text())
    if not compiled: raise ValueError('Missing coverage compiler manifest')
    for name, value in compiled.items():
        if not name.startswith('packages/contracts-bedrock/') or '..' in Path(name).parts or UP.digest(ROOT / name) != value:
            raise ValueError('Missing or corrupt prepared coverage artifact')
    for source in prepared.iterdir():
        if source.is_file() and source.name != 'final.json': shutil.copy2(source, directory / source.name)
        elif source.is_dir(): shutil.copytree(source, directory / source.name, dirs_exist_ok=True)
    shutil.copy2(prepared / 'final.json', directory / 'preparation-manifest.json')
    shutil.copy2(prepared / 'settings.json', directory / 'preparation-settings.json')
    old.update(rwx_run_id=os.environ.get('RWX_RUN_ID'), rwx_task_attempt=os.environ.get('RWX_TASK_ATTEMPT_NUMBER'))
    UP.write(directory / 'settings.json', old); coverage_profile(seed)
    if UP.stage(directory, 'runtime-config', ['forge', 'config', '--json'], json_output=True):
        raise ValueError('Missing runtime coverage configuration')
    if json.loads((directory / 'runtime-config.json').read_text()) != json.loads((directory / 'foundry-config.json').read_text()):
        raise ValueError('Runtime coverage configuration differs')
    if UP.stage(directory, 'runtime-files', CS.FILE_COMMANDS['standard']) or \
       CS.file_selection((directory / 'runtime-files.log').read_text()) != files:
        raise ValueError('Runtime coverage file discovery changed')


def original_evidence(stdout, attribution):
    """Retain every predicate while binding merged campaigns to their real anchor."""
    suite, cases, summary, campaign, tag = None, {}, None, None, None
    campaigns, declared = [], {}
    states = {'PASS': 'pass', 'SKIP': 'skip', 'FAIL': 'fail'}
    pattern = re.compile(r'^\[(PASS|SKIP|FAIL)(?:: (.*))?\] (\w+\([^\s]*\))(?: \(.*\))?$')
    member_pattern = re.compile(r'^\[(PASS|SKIP|FAIL)(?:: (.*))?\] (invariant\w+)$')
    for line in stdout.splitlines():
        found = re.fullmatch(r'Ran (\d+) test suites? in .*?: (\d+) tests? passed, (\d+) failed, (\d+) skipped \((\d+) total tests?\)', line)
        if found:
            if campaign is not None: raise ValueError('Unclosed original invariant campaign')
            summary = dict(zip(('suites', 'pass', 'fail', 'skip', 'total'), map(int, found.groups()))); break
        found = re.fullmatch(r'Ran (\d+) tests? for (.+)', line)
        if found:
            if campaign is not None: raise ValueError('Unclosed original invariant campaign')
            count, suite = found.groups()
            if suite in declared: raise ValueError('Duplicate original coverage suite')
            declared[suite] = int(count); tag = None; continue
        if suite is not None:
            label = suite.rsplit(':', 1)[-1] + ' invariants'
            found = re.fullmatch(re.escape(label) + r':(?: (\d+)/(\d+) invariants broken)?', line)
            if found:
                if campaign is not None or tag is None and found[1] is None:
                    raise ValueError('Original invariant campaign lacks its aggregate verdict')
                campaign = {'class': suite, 'outcome': 'fail' if found[1] is not None else tag['outcome'],
                            'reason': tag['reason'] if tag is not None else None, 'members': [],
                            'broken': int(found[1]) if found[1] is not None else None,
                            'declared_predicates': int(found[2]) if found[2] is not None else None}
                tag = None; continue
            if campaign is not None:
                found = member_pattern.fullmatch(line)
                if found:
                    state, reason, name = found.groups(); name += '()'; key = (suite, name)
                    if key in cases: raise ValueError('Duplicate original invariant predicate')
                    cases[key] = {'outcome': states[state], 'reason': reason}; campaign['members'].append(name); continue
                if re.fullmatch(r' ?' + re.escape(label) + r'(?: \(block: \d+\))? \(.*\)', line):
                    members = sorted(campaign['members'])
                    if len(members) < 2 or campaign['declared_predicates'] is not None and \
                       (len(members) != campaign['declared_predicates'] or campaign['broken'] != sum(cases[suite, n]['outcome'] == 'fail' for n in members)):
                        raise ValueError('Incomplete original invariant campaign predicates')
                    campaign['members'] = members; campaign['anchor'] = members[0]; campaigns.append(campaign); campaign = None; continue
                if line.startswith('Suite result:'): raise ValueError('Missing original invariant campaign footer')
        found = re.fullmatch(r'\[(PASS|SKIP|FAIL)(?:: (.*))?\]', line)
        if found:
            state, reason = found.groups(); tag = {'outcome': states[state], 'reason': reason}; continue
        found = pattern.fullmatch(line)
        if not found: continue
        if suite is None: raise ValueError('Original coverage verdict lacks a suite')
        state, reason, name = found.groups()
        key = (suite, name)
        if key in cases: raise ValueError('Duplicate original coverage verdict')
        cases[key] = {'outcome': states[state], 'reason': reason}
    if summary is None or not cases: raise ValueError('Incomplete original coverage execution log')
    if set(attribution) != {'version', 'tests'} or attribution.get('version') != 1 or not isinstance(attribution.get('tests'), list):
        raise ValueError('Unsupported or missing original coverage attribution')
    machine, kinds = {}, {}
    for row in attribution['tests']:
        if not isinstance(row, dict) or not {'suite', 'test', 'status', 'kind', 'covered'} <= set(row):
            raise ValueError('Incomplete original coverage attribution entry')
        key = (row['suite'], row['test'])
        if key in machine or row['status'] not in ('success', 'failure', 'skipped') or not isinstance(row['covered'], list):
            raise ValueError('Invalid or duplicate original coverage attribution')
        machine[key] = {'success': 'pass', 'failure': 'fail', 'skipped': 'skip'}[row['status']]
        kinds[key] = row.get('kind')
    expected = {key: row['outcome'] for key, row in cases.items()}
    for group in campaigns:
        identity = group['class']; anchor = (identity, group['anchor'])
        if kinds.get(anchor) != 'invariant': raise ValueError('Original merged campaign lacks its invariant attribution anchor')
        for member in group['members']: expected.pop((identity, member))
        expected[anchor] = group['outcome']
        members = [cases[identity, member]['outcome'] for member in group['members']]
        if group['outcome'] == 'pass' and 'fail' in members or group['outcome'] == 'skip' and set(members) != {'skip'}:
            raise ValueError('Original invariant predicate and campaign outcomes disagree')
    if machine != expected:
        raise ValueError('Original coverage log and per-test attribution disagree')
    engine = {state: sum(value == state for value in machine.values()) for state in ('pass', 'fail', 'skip')}
    per_suite = {identity: sum(key[0] == identity for key in machine) for identity in declared}
    for group in campaigns:
        skipped = sum(cases[group['class'], n]['outcome'] == 'skip' for n in group['members'])
        extra = skipped - int(group['outcome'] == 'skip')
        engine['skip'] += extra; per_suite[group['class']] += extra
    if declared != per_suite or summary != {'suites': len(declared), **engine, 'total': sum(engine.values())}:
        raise ValueError('Original coverage engine totals disagree with complete attribution')
    return {'cases': [{'class': c, 'name': n, **cases[c, n]} for c, n in sorted(cases)],
            'invariant_campaigns': sorted(campaigns, key=lambda row: (row['class'], row['anchor'])), 'engine_summary': summary}


def original_cases(stdout, attribution): return original_evidence(stdout, attribution)['cases']


def attribution_summary(attribution):
    # Originals retain every item and hit. Fingerprint each complete canonical
    # row so the comparison index does not duplicate multi-gigabyte reports.
    return {'version': attribution['version'], 'tests': [{k: row[k] for k in ('suite', 'test', 'status', 'kind')} |
            {'covered_items': len(row['covered']), 'complete_row_sha256': hashlib.sha256(
             json.dumps(row, sort_keys=True, separators=(',', ':')).encode()).hexdigest()} for row in attribution['tests']]}


def derived_junit(path, cases, phase, campaigns=()):
    # Foundry coverage does not emit original JUnit. This reporting view is
    # explicitly derived from, and checked against, the retained two originals.
    cases = list(cases)
    for group in campaigns:
        if group['outcome'] == 'fail' and not any(row['class'] == group['class'] and row['name'] in group['members'] and row['outcome'] == 'fail' for row in cases):
            cases.append({'class': group['class'], 'name': '<invariant campaign ' + group['anchor'] + '>', 'outcome': 'fail',
                          'reason': group['reason'] or 'Original invariant campaign failed; predicate verdicts retained separately'})
    root = ET.Element('testsuites')
    for name in sorted({row['class'] for row in cases}):
        rows = [row for row in cases if row['class'] == name]
        suite = ET.SubElement(root, 'testsuite', name=phase + ':' + name, tests=str(len(rows)),
                              failures=str(sum(r['outcome'] == 'fail' for r in rows)), skipped=str(sum(r['outcome'] == 'skip' for r in rows)))
        for row in rows:
            case = ET.SubElement(suite, 'testcase', classname=phase + ':' + name, name=row['name'])
            if row['outcome'] != 'pass':
                message = row['reason'] or ''
                child = ET.SubElement(case, 'skipped' if row['outcome'] == 'skip' else 'failure', message=message)
                child.text = message
    ET.ElementTree(root).write(path, encoding='utf-8', xml_declaration=True)


def accounting(path, phase, selection, bindings):
    # Both passes may execute the same Solidity case. Keep their reporting
    # identities distinct while binding each to its original compiler identity.
    prefix = phase + ':'
    selected = [(prefix + identity, name) for identity, name in selection]
    methods = {prefix + identity: row for identity, row in bindings.items()}
    return UP.junit(path, selected, methods)


def source_name(name, contract_root=None):
    path = Path(name)
    if path.is_absolute():
        try: path = path.relative_to(contract_root or CONTRACTS)
        except ValueError: raise ValueError('Coverage source is outside the contract workspace') from None
    if '..' in path.parts or not path.parts or '\\' in name: raise ValueError('Unsafe original coverage source')
    return path.as_posix()


def lcov(text, contract_root=None):
    records, rows, source = [], [], None
    allowed = {'TN', 'FN', 'FNDA', 'DA', 'BRDA', 'FNF', 'FNH', 'LF', 'LH', 'BRF', 'BRH'}
    for line in text.splitlines():
        if line == 'end_of_record':
            if source is None: raise ValueError('LCOV record lacks a source')
            totals = {key: int(value) for key, value in rows if key in ('FNF', 'FNH', 'LF', 'LH', 'BRF', 'BRH')}
            if set(totals) != {'FNF', 'FNH', 'LF', 'LH', 'BRF', 'BRH'} or \
               any(totals[hit] > totals[found] for hit, found in [('FNH', 'FNF'), ('LH', 'LF'), ('BRH', 'BRF')]):
                raise ValueError('Incomplete or impossible original LCOV totals')
            records.append({'source': source, 'entries': sorted(rows)}); rows, source = [], None; continue
        key, sep, value = line.partition(':')
        if not sep: raise ValueError('Invalid original LCOV line')
        if key == 'SF':
            if source is not None: raise ValueError('Duplicate LCOV source within a record')
            source = source_name(value, contract_root)
        elif key in allowed:
            if key in ('FNF', 'FNH', 'LF', 'LH', 'BRF', 'BRH') and not value.isdecimal():
                raise ValueError('Invalid original LCOV count')
            rows.append((key, value))
        else: raise ValueError('Unsupported original LCOV field')
    if source is not None or rows or not records: raise ValueError('Missing or truncated original LCOV records')
    return sorted(records, key=lambda row: (row['source'], row['entries']))


def collect_pass(directory, phase, redactor, status, selection, bindings):
    for source, name in [('lcov.info', 'original.lcov.info'), ('coverage-attribution.json', 'original.attribution.json')]:
        p = CONTRACTS / source
        if p.exists(): (directory / name).write_bytes(redactor(p.read_bytes()))
    generated = {}
    for name in ('cache/test-failures', 'cache/fuzz', 'cache/invariant', '.testdata', '.resource-metering.csv'):
        p = CONTRACTS / name
        if p.exists():
            for source in ([p] if p.is_file() else sorted(p.rglob('*'))):
                if not source.is_file(): continue
                relative = str(source.relative_to(CONTRACTS)); before = source.read_bytes(); after = redactor(before)
                target = directory / 'generated' / relative; target.parent.mkdir(parents=True, exist_ok=True); target.write_bytes(after)
                generated[relative] = {'original_sha256': hashlib.sha256(before).hexdigest(), 'retained_sha256': UP.digest(target), 'redacted': before != after}
    UP.write(directory / 'generated-redaction.json', generated)
    if not (directory / 'original.lcov.info').exists() or not (directory / 'original.attribution.json').exists():
        raise ValueError('Coverage execution did not produce complete original file reports')
    attribution = json.loads((directory / 'original.attribution.json').read_text())
    evidence = original_evidence((directory / 'tests.log').read_text(), attribution); cases = evidence['cases']
    UP.write(directory / 'original-events.json', evidence)
    UP.write(directory / 'original-cases.json', cases); derived_junit(directory / 'derived.junit.xml', cases, phase, evidence['invariant_campaigns'])
    records = lcov((directory / 'original.lcov.info').read_text()); UP.write(directory / 'lcov-records.json', records)
    sources = {row['source'] for row in records}
    sources.update(source_name(item['source']) for row in attribution['tests'] for item in row['covered'])
    UP.write(directory / 'coverage-source-sha256.json', {name: UP.digest(CONTRACTS / name) for name in sorted(sources)})
    if not status:
        # Whole-contract setup skips and empty abstract declarations receive the
        # same complete accounting used for the other contract workloads.
        UP.write(directory / 'coverage.json', accounting(directory / 'derived.junit.xml', phase, selection, bindings))


def run(directory, url, redactor, block_path):
    if not url: raise ValueError('Missing test-only coverage archive RPC')
    os.environ['ETH_RPC_URL'] = url
    if block_path:
        if block_path.name != 'block.json': raise ValueError('Unexpected coverage preflight artifact')
        UP.originals(block_path.parent, {'block.json', 'pinned-block.log', 'pinned-block.stage.json'}, [], 'coverage/preflight')
        chosen = json.loads(block_path.read_text())
        if chosen.get('source_sha') != UP.revision() or chosen.get('policy') != 'Just current-day 00:00 UTC' or \
           UP.block(url, chosen['number']) != {k: chosen[k] for k in ('chain_id', 'number', 'hash', 'timestamp')}:
            raise ValueError('Stale, unavailable or mismatched coverage archive block')
        preflight = directory / 'archive-preflight'; preflight.mkdir()
        for source in block_path.parent.iterdir():
            if not source.is_file(): raise ValueError('Unexpected coverage preflight directory')
            shutil.copy2(source, preflight / ('preflight-manifest.json' if source.name == 'final.json' else source.name))
    else: chosen = UP.pinned_block(directory, url, redactor)
    UP.write(directory / 'block.json', chosen); os.environ['FORK_BLOCK_NUMBER'] = str(chosen['number'])
    selected = json.loads((directory / 'selection.json').read_text()); bindings = json.loads((directory / 'signature-bindings.json').read_text())
    status = 0
    for phase in PHASES:
        current = directory / phase; current.mkdir()
        for name in ('cache/test-failures', 'cache/fuzz', 'cache/invariant', 'lcov.info', 'lcov-upgrade.info', 'coverage-attribution.json'):
            p = CONTRACTS / name
            if p.is_dir(): shutil.rmtree(p)
            elif p.exists(): p.unlink()
        status = UP.stage(current, 'tests', COMMANDS[phase], redactor)
        collect_pass(current, phase, redactor, status, selected[phase], bindings)
        if status: break
    # Preserve coverage-lcov-all's two original public output paths as well as
    # the independently sealed originals. Verdicts and reports are never cached.
    for phase, name in [('ordinary', 'lcov.info'), ('upgrade', 'lcov-upgrade.info')]:
        source = directory / phase / 'original.lcov.info'
        if source.exists(): shutil.copy2(source, CONTRACTS / name)
    if status and status < 128 and (CONTRACTS / 'cache/test-failures').is_file() and (CONTRACTS / 'cache/test-failures').stat().st_size:
        UP.stage(directory, 'rerun', ['just', 'test-rerun'], redactor)
    return status


def main():
    os.chdir(ROOT); p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('mode', choices=('preflight', 'prepare', 'run')); p.add_argument('--feature', choices=CS.FEATURES)
    p.add_argument('--prepared', type=Path); p.add_argument('--block', type=Path); a = p.parse_args()
    if a.mode != 'preflight' and not a.feature: p.error('--feature is required')
    directory = ROOT / '.ci/contract-coverage' / ('preflight' if a.mode == 'preflight' else a.feature)
    if a.mode != 'preflight': directory /= a.mode
    shutil.rmtree(directory, ignore_errors=True); directory.mkdir(parents=True); status, errors = 1, []
    url = os.environ.get(RPC_INPUT)
    try:
        if os.environ.get('CI_CONTRACT_RPC_ENV_VAR', RPC_INPUT) != RPC_INPUT:
            raise ValueError('Coverage archive RPC parameter differs from the benchmark input')
        redactor = UP.Redactor(url) if url else lambda x: x
        if a.mode == 'preflight':
            if not url: raise ValueError('Missing test-only coverage archive RPC')
            os.environ['ETH_RPC_URL'] = url; UP.write(directory / 'block.json', UP.pinned_block(directory, url, redactor)); status = 0
        else:
            if a.mode == 'run' and not url: raise ValueError('Missing test-only coverage archive RPC')
            if a.prepared: restore(directory, a.prepared, a.feature); status = 0
            else: status = prepare(directory, a.feature)
            if a.mode == 'run' and not status: status = run(directory, url, redactor, a.block)
            settings = json.loads((directory / 'settings.json').read_text())
            CS.verify_inputs(directory, settings['input_sha256'], 'verdict' if a.mode == 'run' else 'preparation',
                             settings['runtime_output_paths'] if a.mode == 'run' else [])
    except (OSError, ValueError, KeyError, ET.ParseError, subprocess.CalledProcessError) as error:
        # RPC methods and subprocess output are redacted before retention.
        # Never include an exception's request object or URL in a report.
        message = str(error)
        if url:
            try: message = UP.Redactor(url)(message.encode()).decode()
            except ValueError: message = 'Invalid test-only coverage RPC configuration'
        errors.append(message); print(message, file=sys.stderr)
        if status == 0: status = 1
    return UP.finish(directory, status, errors)


if __name__ == '__main__': sys.exit(main())
