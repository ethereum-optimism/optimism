#!/usr/bin/env python3
"""Cache and verify the immutable image used by the original Kontrol scripts."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import time
import tomllib

ROOT = Path(__file__).resolve().parents[2]


def digest(path): return hashlib.sha256(path.read_bytes()).hexdigest()


def selection():
    value = json.loads((ROOT / 'ops/ci/kontrol-image.json').read_text())
    version = tomllib.loads((ROOT / 'mise.toml').read_text())['tools']['kontrol']
    if (value['version'] != version or value['tag'] != 'runtimeverificationinc/kontrol:ubuntu-jammy-' + version
            or value['platform'] != 'linux/amd64' or not re.fullmatch('sha256:[0-9a-f]{64}', value['config_digest'])
            or not re.fullmatch('runtimeverificationinc/kontrol@sha256:[0-9a-f]{64}', value['image'])):
        raise ValueError('Kontrol image differs from the pinned toolchain or platform')
    return value


def validate_inspection(result):
    value = selection()
    if len(result) != 1: raise ValueError('Missing or duplicate original Kontrol image')
    row = result[0]
    # Classic Docker reports the config digest as Id; the containerd image store
    # reports the manifest digest. Both are pinned by the same immutable manifest.
    ids = {value['config_digest'], value['image'].split('@', 1)[1]}
    if (row['Id'] not in ids or value['image'] not in row['RepoDigests']
            or row['Os'] + '/' + row['Architecture'] != value['platform']):
        raise ValueError('Wrong Kontrol image bytes or platform')
    return result


def inspect(report=None):
    value = selection()
    result = json.loads(subprocess.check_output(['docker', 'image', 'inspect', value['tag']], text=True))
    if report is not None:
        report.write_text(json.dumps(result, indent=2, sort_keys=True) + '\n')
    return validate_inspection(result)


def prepare():
    value = selection(); directory = ROOT / '.ci/kontrol-build/image'; directory.mkdir(parents=True, exist_ok=True)
    inputs = {name: digest(ROOT / name) for name in ('mise.toml', 'ops/ci/kontrol-image.json', 'ops/ci/kontrol-image.py')}
    (directory / 'inputs.json').write_text(json.dumps({'selection':value,'input_sha256':inputs},indent=2,sort_keys=True) + '\n')
    # Pull only an immutable image. Retried network operations never retry a verdict.
    attempts = []
    with (directory / 'pull.log').open('wb') as log:
        for attempt in range(1, 6):
            started = time.time()
            code = subprocess.call(['docker', 'pull', '--platform', value['platform'], value['image']], stdout=log, stderr=subprocess.STDOUT)
            attempts.append({'attempt': attempt, 'exit_code': code, 'elapsed_seconds': time.time() - started})
            if code == 0: break
            if attempt < 5: time.sleep(2 ** attempt)
    (directory / 'pull-attempts.json').write_text(json.dumps(attempts, indent=2) + '\n')
    if code: raise ValueError('Pinned Kontrol image download failed')
    # The original scripts use the mise-selected tag. Point that local tag at
    # the verified immutable image, without changing their Docker invocation.
    subprocess.run(['docker', 'tag', value['image'], value['tag']], check=True)
    inspect(directory / 'inspect.json')
    version = subprocess.check_output(['docker', 'run', '--rm', '--platform', value['platform'], value['image'], 'kontrol', 'version'], text=True).strip()
    if version != 'Kontrol version: ' + value['version']: raise ValueError('Wrong original Kontrol executable version')
    metadata = {'selection': value, 'kontrol_version': version,
        'input_sha256': inputs,
        'original_sha256': {p.name: digest(p) for p in directory.iterdir() if p.is_file() and p.name != 'metadata.json'}}
    (directory / 'metadata.json').write_text(json.dumps(metadata, indent=2, sort_keys=True) + '\n')


def verify(directory):
    if any(p.is_symlink() or not p.is_file() for p in directory.iterdir()):
        raise ValueError('Unsafe Kontrol image preparation input')
    value = selection(); metadata = json.loads((directory / 'metadata.json').read_text())
    if (metadata['selection'] != value or metadata['kontrol_version'] != 'Kontrol version: ' + value['version']
            or metadata['input_sha256'] != {name: digest(ROOT / name) for name in ('mise.toml', 'ops/ci/kontrol-image.json', 'ops/ci/kontrol-image.py')}):
        raise ValueError('Stale Kontrol image settings or toolchain inputs')
    if set(metadata['original_sha256']) != {'inputs.json','inspect.json', 'pull.log', 'pull-attempts.json'}:
        raise ValueError('Incomplete Kontrol image preparation originals')
    if {p.name for p in directory.iterdir()} != {*metadata['original_sha256'], 'metadata.json'}:
        raise ValueError('Extra or unsealed Kontrol image preparation input')
    for name, expected in metadata['original_sha256'].items():
        if (directory / name).is_symlink() or digest(directory / name) != expected:
            raise ValueError('Corrupt Kontrol image preparation original')
    if json.loads((directory / 'inputs.json').read_text()) != {'selection':value,'input_sha256':metadata['input_sha256']}:
        raise ValueError('Changed original Kontrol image selection')
    validate_inspection(json.loads((directory / 'inspect.json').read_text()))
    inspect()
    return metadata


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('prepare', 'verify'))
    parser.add_argument('--directory', type=Path, default=ROOT / '.ci/kontrol-build/image')
    args = parser.parse_args()
    if args.command == 'prepare': prepare()
    else: verify(args.directory)
