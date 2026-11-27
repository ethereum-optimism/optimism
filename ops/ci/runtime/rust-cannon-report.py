#!/usr/bin/env python3
"""Record the settings and exit status of one Cannon job, and check the pinned witness."""
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import time
import xml.etree.ElementTree as ET

CONFIG = Path('ops/ci/runtime/cannon-witness.json')
DOCKERFILE = Path('rust/kona/docker/fpvm-prestates/cannon-repro.dockerfile')
# The JUnit case of each job. The witness and offline jobs name their case after the witness.
CASES = {'env': 'build/kona-cannon-env', 'go': 'go/cannon', 'lint': 'clippy/kona-std-fpvm',
         'build': 'build/kona-client'}


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


def begin(directory, job):
    settings = {'suite': 'rust-cannon', 'job': job, 'source_sha': command('git', 'rev-parse', 'HEAD'),
                'rwx_run_id': os.environ.get('RWX_RUN_ID'),
                'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER'),
                'rustc': command('rustc', '--version'), 'cargo': command('cargo', '--version'),
                'started_at': time.time()}
    if job in ('go', 'offline'):
        settings['go'] = command('go', 'version')
    if job in ('witness', 'offline'):
        settings['witness'] = witness_config()
    write(directory / 'settings.json', settings)


def check_witness():
    """Fail unless the fetched witness archive has the pinned checksum."""
    data = witness_config()
    if digest(Path('rust/kona/bin/client/testdata') / data['filename']) != data['sha256']:
        raise ValueError('Pinned Cannon witness archive checksum mismatch')


def finish(directory, status):
    """Write the job's exit status and stage records. The recipes own the verdict."""
    settings = read(directory / 'settings.json')
    job = settings['job']
    stages = {p.name.removesuffix('.stage.json'): read(p) for p in directory.glob('*.stage.json')}
    if job == 'witness':
        name = 'archive/' + settings['witness']['filename']
    elif job == 'offline':
        name = 'kona-client/' + settings['witness']['l2_chain_id'] + '/' + settings['witness']['block_number']
    else:
        name = CASES[job]
    suite = ET.Element('testsuite', name='cannon-' + job, tests='1')
    case = ET.SubElement(suite, 'testcase', classname='cannon-' + job, name=name)
    if status:
        ET.SubElement(case, 'failure', message='See original Cannon stages and final.json')
    ET.ElementTree(suite).write(directory / 'checks.junit.xml', encoding='utf-8', xml_declaration=True)
    write(directory / 'final.json', {'exit_code': status, 'stages': stages})
    return status


if __name__ == '__main__':
    mode = sys.argv[1]
    if mode == 'config':
        config = witness_config()
        keys = ['block_number', 'l2_claim', 'l2_output_root', 'l2_head', 'l1_head', 'l2_chain_id', 'filename', 'release_url']
        print('\n'.join(config[k] for k in keys))
    elif mode == 'base':
        print(re.search(r'^FROM (\S+) AS kona-build-env$', DOCKERFILE.read_text(), re.M).group(1))
    elif mode == 'witness':
        check_witness()
    else:
        directory = Path(sys.argv[2]).resolve()
        if mode == 'begin': begin(directory, sys.argv[3])
        elif mode == 'finish': sys.exit(finish(directory, int(sys.argv[3])))
        else: raise ValueError('Unknown Cannon report mode')
