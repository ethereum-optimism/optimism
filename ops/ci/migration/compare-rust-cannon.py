#!/usr/bin/env python3
"""Compare complete original Cannon lint, build and offline guest reports."""
import argparse
import gzip
import hashlib
import importlib.util
import json
from pathlib import Path
_SPEC_E=importlib.util.spec_from_file_location('report_evidence',Path(__file__).with_name('report-evidence.py'))
E=importlib.util.module_from_spec(_SPEC_E);_SPEC_E.loader.exec_module(E)
import re

JOBS = ('lint', 'build', 'offline')
BINDING = ('source_sha', 'input_sha256', 'source_trees', 'stable_rust', 'nightly',
           'go_pin', 'profile', 'target', 'rustflags', 'build_std', 'custom_configs', 'witness')
EMPTY_LOGS = {'guest.log'}
GO_FILES = {'cannon/bin/cannon', 'cannon/bin/cannon64-impl', 'cannon/multicannon/embeds/cannon-8'}


def read(path):
    return json.loads(path.read_text())


def digest(path):
    with path.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def command(stage, root):
    return {'argv': [arg.replace(root + '/', '<repo>/') for arg in stage['argv']],
            'cwd': stage['cwd'], 'exit_code': stage['exit_code']}


def compare(root, sha):
    if not re.fullmatch('[0-9a-f]{40}', sha):
        raise ValueError('Expected a full source SHA')
    evidence = {'source_sha': sha, 'verified_parity': False, 'jobs': {},
                'original_sha256': {}, 'manifest_declared_empty_logs': []}
    for provider in ('circle', 'rwx'):
        evidence['original_sha256'][provider] = {}
        for job in JOBS:
            directory = root / provider / ('cannon-' + job)
            settings, final = read(directory / 'settings.json'), read(directory / 'final.json')
            if settings['source_sha'] != sha or settings['job'] != job or settings['suite'] != 'rust-cannon':
                raise ValueError(f'{provider}/{job}: source or workload mismatch')
            if settings['provider'] != {'circle': 'circleci', 'rwx': 'rwx'}[provider]:
                raise ValueError('Provider provenance mismatch')
            if provider == 'rwx' and (not settings.get('rwx_run_id') or str(settings.get('rwx_task_attempt')) != '1'):
                raise ValueError('Missing fresh identity or task retries require investigation')
            if final['exit_code'] != 0 or final['report_errors'] or not final['stages']:
                raise ValueError('Failed or incomplete Cannon verdict')
            required = {'settings.json', 'checks.junit.xml', 'variants.log', 'variants.stage.json',
                        'image.json', 'image-elfs.json', job + '.log', job + '.stage.json'}
            if provider == 'rwx' and job != 'offline': required.add('verified-env.json')
            if job in ('build', 'offline'): required.add('elfs.json')
            if job == 'offline':
                required.update({'guest.json', 'guest.log', 'guest.stage.json', 'guest-coverage.json',
                                 'out.bin.gz', 'go-binaries.json', 'witness.json'})
                if provider == 'rwx': required.update('verified-' + kind + '.json' for kind in ('go', 'witness', 'build'))
            hashes = final['original_sha256']
            if not required <= hashes.keys():
                raise ValueError('Incomplete original Cannon artifact manifest')
            if provider == 'rwx':
                kinds = ('go', 'witness', 'build') if job == 'offline' else ('env',)
                for kind in kinds:
                    verified = read(directory / ('verified-' + kind + '.json'))
                    if verified['producer_source_sha'] != sha or not re.fullmatch('[0-9a-f]{64}', verified['manifest_sha256']):
                        raise ValueError('Inherited Cannon producer provenance mismatch')
                    if job == 'offline' and kind == 'build' and verified.get('image_manifest_sha256') != digest(root / provider / 'cannon-build/image.json'):
                        raise ValueError('Offline Cannon image handoff differs from the counted build producer')
            for name, stage in final['stages'].items():
                if stage['exit_code'] != 0 or not {name + '.stage.json', name + '.log'} <= hashes.keys():
                    raise ValueError('Unsuccessful or incomplete original Cannon stage')
                if stage != read(directory / (name + '.stage.json')):
                    raise ValueError('Stage summary differs from original Cannon stage')
            recovered=[]
            try:
                E.verify_files(directory, hashes, required=(),
                               missing_empty=recovered if provider == 'circle' else None,
                               recoverable=EMPTY_LOGS, label=provider+'/'+job)
            except OSError as error:
                raise ValueError('Missing nonempty original Cannon artifact: '+str(error)) from error
            except ValueError as error:
                raise ValueError('Corrupt original Cannon artifact: '+str(error)) from error
            evidence['manifest_declared_empty_logs'].extend(
                {'job':job,'name':row['path'],'sha256':row['sha256']} for row in recovered)
            evidence['original_sha256'][provider][job] = hashes
    for job in JOBS:
        a, b = (root / provider / ('cannon-' + job) for provider in ('circle', 'rwx'))
        sa, sb = read(a / 'settings.json'), read(b / 'settings.json')
        for key in BINDING + ('variants', 'rustc', 'cargo'):
            if sa[key] != sb[key]:
                raise ValueError(f'{job}: settings differ at {key}')
        variants = sa['variants']
        selected = {v['binary'] for v in variants}
        if not selected or len(selected) != len(variants):
            raise ValueError('Empty or duplicate Cannon variant selection')
        if (a / 'variants.log').read_bytes() != (b / 'variants.log').read_bytes():
            raise ValueError('Original authoritative Cannon variant discovery differs')
        expected_binding = {key: sa[key] for key in BINDING}
        ia, ib = read(a / 'image.json'), read(b / 'image.json')
        for key in ('binding', 'base', 'base_digests', 'tools'):
            if ia[key] != ib[key] or (key == 'binding' and ia[key] != expected_binding):
                raise ValueError('Cannon Docker input/toolchain provenance differs')
        stages = ('variants', job) + (('guest',) if job == 'offline' else ())
        primary = {}
        for name in stages:
            ca, cb = command(read(a / (name + '.stage.json')), sa['workspace_root']), command(read(b / (name + '.stage.json')), sb['workspace_root'])
            if ca != cb:
                raise ValueError('Cannon command, working directory or outcome differs: ' + name)
            primary[name] = ca
        info = {'variants': variants, 'outcome': 'pass', 'retries': 0, 'stages': primary,
                'images': {'circle': ia, 'rwx': ib}, 'elfs': {}}
        for filename, folder in [('image-elfs.json', 'image-elfs')] + ([('elfs.json', 'elfs')] if job in ('build', 'offline') else []):
            ma, mb = read(a / filename), read(b / filename)
            if ma != mb or ma['binding'] != expected_binding or ma['variants'] != variants or set(ma['files']) != selected:
                raise ValueError('Cannon ELF inventories, inputs or binary hashes differ')
            for binary, artifact in ma['files'].items():
                for directory in (a, b):
                    original = directory / folder / binary
                    header = original.read_bytes()[:20]
                    if (digest(original) != artifact['sha256'] or original.stat().st_size != artifact['size'] or
                        len(header) != 20 or header[:7] != b'\x7fELF\x02\x02\x01' or header[18:20] != b'\0\x08'):
                        raise ValueError('Corrupt or invalid original MIPS ELF')
            info['elfs'][filename] = ma
        if (a / 'checks.junit.xml').read_bytes() != (b / 'checks.junit.xml').read_bytes():
            raise ValueError('Original Cannon JUnit differs')
        if job == 'offline':
            wa, wb = read(a / 'witness.json'), read(b / 'witness.json')
            if wa != wb or wa['binding'] != expected_binding or wa['sha256'] != sa['witness']['sha256'] or wa['filename'] != sa['witness']['filename']:
                raise ValueError('Pinned Cannon witness provenance differs')
            ga, gb = read(a / 'go-binaries.json'), read(b / 'go-binaries.json')
            if ga['binding'] != expected_binding or gb['binding'] != expected_binding or ga['go'] != gb['go'] or set(ga['files']) != GO_FILES or set(gb['files']) != GO_FILES:
                raise ValueError('Native Cannon Go build provenance differs')
            info['native_go'] = {'circle': ga, 'rwx': gb, 'byte_identical': ga == gb,
                                 'comparison': 'Pinned Go, complete source inputs and original build coverage; native host debug paths may differ.'}
            ca, cb = read(a / 'guest-coverage.json'), read(b / 'guest-coverage.json')
            for key in ('boundary', 'final', 'fresh_execution'):
                if ca[key] != cb[key]: raise ValueError('Fresh Cannon guest coverage differs')
            final = ca['final']
            if (ca['boundary'] != sa['witness'] or ca['fresh_execution'] is not True or
                final != read(a / 'guest.json') or final != read(b / 'guest.json') or
                final['exited'] is not True or type(final['exitCode']) is not int or final['exitCode'] != 0 or
                type(final['step']) is not int or final['step'] <= 0 or final['stateVersion'] != 8 or
                not re.fullmatch('0x[0-9a-f]{64}', final['witnessHash']) or not re.fullmatch('0x(?:[0-9a-f]{2})+', final['witness'])):
                raise ValueError('Invalid or unsuccessful original Cannon guest verdict')
            for directory, coverage in ((a, ca), (b, cb)):
                if coverage['final_state_sha256'] != digest(directory / 'out.bin.gz'):
                    raise ValueError('Original final-state coverage hash differs')
                log = (directory / 'offline.log').read_text()
                if 'Successfully validated L2 block' not in log or sa['witness']['l2_claim'] not in log:
                    raise ValueError('Original output-root validation missing')
            if gzip.decompress((a / 'out.bin.gz').read_bytes()) != gzip.decompress((b / 'out.bin.gz').read_bytes()):
                raise ValueError('Complete uncompressed final Cannon state differs')
            info['guest'] = {'circle': ca, 'rwx': cb, 'complete_state_byte_identical': True,
                             'compressed_state_byte_identical': ca['final_state_sha256'] == cb['final_state_sha256']}
        evidence['jobs'][job] = info
    evidence['verified_parity'] = True
    return evidence


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('root', type=Path); parser.add_argument('sha')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    result = compare(args.root, args.sha)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(result, indent=2, sort_keys=True) + '\n')
    print(json.dumps({'verified_parity': True, 'sha': args.sha, 'jobs': list(result['jobs'])}))
