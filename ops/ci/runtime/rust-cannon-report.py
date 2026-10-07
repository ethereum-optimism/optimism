#!/usr/bin/env python3
"""Bind Cannon dependencies and retain the original VM guest verdict."""
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import time
import tomllib
import xml.etree.ElementTree as ET

IMAGE = 'kona-cannon-env:local'
CONFIG = Path('ops/ci/runtime/cannon-witness.json')
DOCKERFILE = Path('rust/kona/docker/fpvm-prestates/cannon-repro.dockerfile')
JOBS = {'env', 'go', 'witness', 'lint', 'build', 'offline'}
GO_FILES = {'cannon/bin/cannon', 'cannon/bin/cannon64-impl', 'cannon/multicannon/embeds/cannon-8'}
INPUTS = ['mise.toml', '.dockerignore', str(DOCKERFILE), 'ops/scripts/install_mise.sh',
          'rust/Cargo.toml', 'rust/Cargo.lock', 'rust/.cargo/config.toml',
          'rust/justfile', 'rust/kona/justfile', 'rust/kona/docker/cannon/mips64-unknown-none.json',
          'rust/kona/bin/client/justfile', 'rust/kona/bin/client/scripts/fetch-witness-tar.sh',
          str(CONFIG), 'ops/ci/runtime/rust-cannon.sh', 'ops/ci/runtime/rust-cannon-report.py',
          'ops/ci/runtime/rust-workspace-report.py', 'ops/ci/runtime/ci-report.py', 'go.mod', 'go.sum', 'cannon/justfile',
          'justfiles/go.just', 'justfiles/git.just', 'ops/ci/runtime/rust-target-cache.py']


def write(path, data):
    path.write_text(json.dumps(data, indent=2, sort_keys=True) + '\n')


def read(path):
    return json.loads(path.read_text())


def digest(path):
    with Path(path).open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def command(*args, cwd=None):
    return subprocess.check_output(args, cwd=cwd, text=True).strip()


def witness_config():
    data = read(CONFIG)
    if not re.fullmatch('[0-9a-f]{64}', data['sha256']) or Path(data['filename']).name != data['filename']:
        raise ValueError('Invalid witness archive pin')
    if not data['release_url'].startswith('https://github.com/ethereum-optimism/chain-test-data/releases/download/'):
        raise ValueError('Witness must use a pinned chain-test-data release')
    for key in ('l2_claim', 'l2_output_root', 'l2_head', 'l1_head'):
        if not re.fullmatch('0x[0-9a-f]{64}', data[key]):
            raise ValueError('Invalid witness boundary hash')
    for key in ('block_number', 'l2_chain_id'):
        if not re.fullmatch('[1-9][0-9]*', data[key]):
            raise ValueError('Invalid witness chain or block')
    return data


def binding():
    sha = command('git', 'rev-parse', 'HEAD')
    expected = os.environ.get('CI_COMMIT_SHA') or os.environ.get('CIRCLE_SHA1') or sha
    if not re.fullmatch('[0-9a-f]{40}', expected) or sha != expected:
        raise ValueError('Cannon source revision mismatch')
    tools = tomllib.loads(Path('mise.toml').read_text())['tools']
    nightly = next(t['version'] for t in tools['rust'] if t['version'].startswith('nightly-'))
    return {'source_sha': sha, 'input_sha256': {p: digest(p) for p in INPUTS},
            'source_trees': {p: command('git', 'rev-parse', f'HEAD:{p}') for p in
                             ('rust', 'cannon', 'op-service', 'op-preimage', 'op-core/nuts/bundles')},
            'stable_rust': tools['rust'][0]['version'], 'nightly': nightly, 'go_pin': tools['go'],
            'profile': 'release-client-lto', 'target': 'mips64-unknown-none',
            'rustflags': '-Clink-arg=-e_start -Cllvm-args=-mno-check-zero-division',
            'build_std': 'core,alloc', 'custom_configs': False, 'witness': witness_config()}


def begin(directory, job):
    if job not in JOBS:
        raise ValueError('Unknown Cannon job')
    write(directory / 'settings.json', binding() | {
        'suite': 'rust-cannon', 'job': job, 'provider': os.environ.get('CI_RUST_PROVIDER', 'circleci'),
        'workspace_root': str(Path.cwd()), 'started_at': time.time(),
        'rwx_run_id': os.environ.get('RWX_RUN_ID'), 'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER'),
        'rustc': command('rustc', '--version'), 'cargo': command('cargo', '--version')})


def inventory(directory):
    variants = []
    for line in (directory / 'variants.log').read_text().splitlines():
        parts = line.split()
        if len(parts) != 2 or not re.fullmatch('[a-zA-Z0-9_-]+', parts[0]) or not re.fullmatch('prestate-artifacts-[a-zA-Z0-9_-]+', parts[1]):
            raise ValueError('Invalid authoritative Cannon variant discovery')
        variants.append({'binary': parts[0], 'prestate_directory': parts[1]})
    if not variants or len({v['binary'] for v in variants}) != len(variants) or len({v['prestate_directory'] for v in variants}) != len(variants):
        raise ValueError('Empty or duplicate Cannon variant discovery')
    settings = read(directory / 'settings.json')
    settings['variants'] = variants
    write(directory / 'settings.json', settings)


def elf_manifest(directory, files, name):
    settings = read(directory / 'settings.json')
    selected = [v['binary'] for v in settings['variants']]
    if set(p.name for p in files.iterdir()) != set(selected):
        raise ValueError('Missing or extra Cannon ELF variants')
    artifacts = {}
    for binary in selected:
        path = files / binary
        with path.open('rb') as source:
            header = source.read(20)
        if len(header) != 20 or header[:7] != b'\x7fELF\x02\x02\x01' or header[18:20] != b'\x00\x08':
            raise ValueError(f'Invalid ELF64 big-endian MIPS artifact: {binary}')
        artifacts[binary] = {'sha256': digest(path), 'size': path.stat().st_size}
    write(directory / name, {'binding': binding(), 'variants': settings['variants'], 'files': artifacts})


def image_manifest(directory):
    settings = read(directory / 'settings.json')
    image = json.loads(command('docker', 'image', 'inspect', IMAGE))[0]
    base = re.search(r'^FROM (\S+) AS kona-build-env$', DOCKERFILE.read_text(), re.M).group(1)
    base_image = json.loads(command('docker', 'image', 'inspect', base))[0]
    if not base_image['RepoDigests']:
        raise ValueError('Missing Cannon base-image digest')
    nightly = settings['nightly']
    tools = command('docker', 'run', '--rm', IMAGE, 'bash', '-c',
                    f'cd /app/rust && rustc +{nightly} --version && cargo +{nightly} --version && mips64-linux-gnuabi64-gcc --version')
    write(directory / 'image.json', {'binding': binding(), 'image_id': image['Id'], 'base': base,
                                   'base_digests': base_image['RepoDigests'], 'tools': tools})
    container = command('docker', 'create', IMAGE)
    try:
        dest = directory / 'image-elfs'
        dest.mkdir()
        subprocess.run(['docker', 'cp', container + ':/app/elf/.', str(dest)], check=True)
        elf_manifest(directory, dest, 'image-elfs.json')
    finally:
        subprocess.run(['docker', 'rm', container], check=True, stdout=subprocess.DEVNULL)


def verify_dependency(directory, producer, kind):
    filenames = {'env': 'image.json', 'go': 'go-binaries.json', 'witness': 'witness.json', 'build': 'elfs.json'}
    manifest = read(producer / filenames[kind])
    if manifest['binding'] != binding():
        raise ValueError('Cannon dependency revision, settings or toolchain mismatch')
    settings = read(producer / 'settings.json')
    expected = binding()
    if settings.get('suite') != 'rust-cannon' or settings.get('job') != kind or {k: settings.get(k) for k in expected} != expected:
        raise ValueError('Cannon dependency producer identity mismatch')
    if settings.get('variants') != read(directory / 'settings.json')['variants']:
        raise ValueError('Cannon dependency variant selection mismatch')
    final = read(producer / 'final.json')
    if final['exit_code'] != 0 or final['report_errors']:
        raise ValueError('Cannon dependency producer failed')
    required = {'settings.json', 'variants.log', 'variants.stage.json', 'checks.junit.xml', filenames[kind]}
    if kind == 'build': required.add('image.json')
    if not required <= final['original_sha256'].keys():
        raise ValueError('Incomplete Cannon dependency original manifest')
    for name, expected in final['original_sha256'].items():
        path = Path(name)
        if path.is_absolute() or '..' in path.parts or not re.fullmatch('[0-9a-f]{64}', expected) or digest(producer / path) != expected:
            raise ValueError('Cannon dependency original artifact checksum mismatch')
    if kind in {'env', 'build'}:
        image = manifest if kind == 'env' else read(producer / 'image.json')
        if image['binding'] != manifest['binding']:
            raise ValueError('Cannon build image revision, settings or toolchain mismatch')
        actual = json.loads(command('docker', 'image', 'inspect', IMAGE))[0]['Id']
        if image['image_id'] != actual:
            raise ValueError(f"Inherited Cannon Docker image mismatch: expected {image['image_id']}, got {actual}")
    if kind == 'witness':
        config = witness_config()
        if manifest['filename'] != config['filename'] or manifest['sha256'] != config['sha256']:
            raise ValueError('Cannon dependency witness pin mismatch')
        path = Path('rust/kona/bin/client/testdata') / manifest['filename']
        if digest(path) != manifest['sha256'] or path.stat().st_size != manifest['size']:
            raise ValueError('Inherited Cannon witness archive corrupt')
    elif kind in {'go', 'build'}:
        selected = GO_FILES if kind == 'go' else {v['binary'] for v in settings['variants']}
        if set(manifest['files']) != selected or (kind == 'build' and manifest['variants'] != settings['variants']):
            raise ValueError('Cannon dependency binary inventory mismatch')
        for name, file in manifest['files'].items():
            path = Path(name) if kind == 'go' else Path('rust/target/mips64-unknown-none/release-client-lto') / name
            if digest(path) != file['sha256'] or path.stat().st_size != file['size']:
                raise ValueError('Inherited Cannon build binary checksum mismatch')
    verified = {'producer_source_sha': manifest['binding']['source_sha'],
                'manifest_sha256': digest(producer / filenames[kind])}
    if kind == 'build': verified['image_manifest_sha256'] = final['original_sha256']['image.json']
    write(directory / f'verified-{kind}.json', verified)


def witness_manifest(directory):
    data = witness_config()
    path = Path('rust/kona/bin/client/testdata') / data['filename']
    if digest(path) != data['sha256']:
        raise ValueError('Pinned Cannon witness archive checksum mismatch')
    write(directory / 'witness.json', {'binding': binding(), 'filename': data['filename'],
                                      'sha256': data['sha256'], 'size': path.stat().st_size})


def go_manifest(directory):
    write(directory / 'go-binaries.json', {'binding': binding(), 'go': command('go', 'version'),
        'files': {p: {'sha256': digest(p), 'size': Path(p).stat().st_size} for p in sorted(GO_FILES)}})


def guest_report(directory):
    state = read(directory / 'guest.json')
    if state['exited'] is not True or type(state['exitCode']) is not int or state['exitCode'] != 0:
        raise ValueError('Cannon guest did not exit successfully')
    if (type(state['step']) is not int or state['step'] <= 0 or
        type(state['stateVersion']) is not int or state['stateVersion'] != 8 or
        not re.fullmatch('0x[0-9a-f]{64}', state['witnessHash']) or
        not re.fullmatch('0x(?:[0-9a-f]{2})+', state['witness'])):
        raise ValueError('Invalid or empty Cannon final guest state')
    log = (directory / 'offline.log').read_text()
    config = witness_config()
    if 'Successfully validated L2 block' not in log or config['l2_claim'] not in log:
        raise ValueError('Missing original guest output-root validation')
    with (directory / 'out.bin.gz').open('rb') as source:
        if source.read(2) != b'\x1f\x8b':
            raise ValueError('Missing compressed Cannon final state')
    write(directory / 'guest-coverage.json', {'boundary': config, 'final': state,
        'final_state_sha256': digest(directory / 'out.bin.gz'), 'fresh_execution': True})
    return state


def finish(directory, status):
    settings = read(directory / 'settings.json')
    job = settings['job']
    stages = {p.name.removesuffix('.stage.json'): read(p) for p in directory.glob('*.stage.json')}
    required = {'env': ['base-pull', 'env'], 'go': ['go'], 'witness': ['witness'],
                'lint': ['lint'], 'build': ['build'], 'offline': ['offline', 'guest']}[job]
    errors = []
    if status == 0:
        for name in ['variants'] + required:
            if stages.get(name, {}).get('exit_code') != 0:
                errors.append('Missing or unsuccessful Cannon stage: ' + name)
        try:
            expected = binding()
            if {k: settings[k] for k in expected} != expected:
                raise ValueError('Cannon source/settings changed during execution')
            if job in ('env', 'lint', 'build', 'offline'):
                if read(directory / 'image.json')['binding'] != binding():
                    raise ValueError('Cannon image binding mismatch')
                read(directory / 'image-elfs.json')
            if job in ('build', 'offline'):
                read(directory / 'elfs.json')
            if job in ('go', 'offline'):
                read(directory / 'go-binaries.json')
            if job in ('witness', 'offline'):
                witness_manifest(directory)
            if job == 'offline':
                guest_report(directory)
        except (ValueError, KeyError, OSError) as error:
            errors.append(str(error))
    suite = ET.Element('testsuite', name='cannon-' + job)
    cases = {'go': ['go/cannon'], 'witness': ['archive/' + settings['witness']['filename']],
             'lint': ['clippy/kona-std-fpvm'],
             'offline': ['kona-client/' + settings['witness']['l2_chain_id'] + '/' + settings['witness']['block_number']]}
    names = cases.get(job, ['build/' + v['binary'] for v in settings.get('variants', [])] or ['build/discovery'])
    for name in names:
        case = ET.SubElement(suite, 'testcase', classname='cannon-' + job, name=name)
        if status or errors:
            ET.SubElement(case, 'failure', message='See original Cannon stages and final.json')
    ET.ElementTree(suite).write(directory / 'checks.junit.xml', encoding='utf-8', xml_declaration=True)
    files = {str(p.relative_to(directory)): digest(p) for p in directory.rglob('*')
             if p.is_file() and p != directory / 'final.json'}
    write(directory / 'final.json', {'exit_code': status, 'report_errors': errors,
                                    'stages': stages, 'original_sha256': files})
    if errors:
        print('\n'.join(errors), file=sys.stderr)
    return status or bool(errors)


if __name__ == '__main__':
    mode = sys.argv[1]
    if mode == 'config':
        config = witness_config()
        keys = ['block_number', 'l2_claim', 'l2_output_root', 'l2_head', 'l1_head', 'l2_chain_id', 'filename', 'release_url']
        print('\n'.join(config[k] for k in keys))
    elif mode == 'base':
        print(re.search(r'^FROM (\S+) AS kona-build-env$', DOCKERFILE.read_text(), re.M).group(1))
    else:
        directory = Path(sys.argv[2]).resolve()
        if mode == 'begin': begin(directory, sys.argv[3])
        elif mode == 'inventory': inventory(directory)
        elif mode == 'image': image_manifest(directory)
        elif mode == 'elfs': elf_manifest(directory, Path(sys.argv[3]), 'elfs.json')
        elif mode == 'witness': witness_manifest(directory)
        elif mode == 'go': go_manifest(directory)
        elif mode == 'verify': verify_dependency(directory, Path(sys.argv[3]), sys.argv[4])
        elif mode == 'finish': sys.exit(finish(directory, int(sys.argv[3])))
        else: raise ValueError('Unknown Cannon report mode')
