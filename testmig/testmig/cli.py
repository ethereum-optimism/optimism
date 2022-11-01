import logging
import os
import shutil
import time
from pathlib import Path

import click
import docker
import docker.errors

from web3 import Web3

from testmig.contracts import SLOTS_TO_MODIFY
from testmig.util import download_with_progress, run_command

import testmig.log_setup

URL_PREFIX = 'https://storage.googleapis.com/optimism/snapshots/mainnet/2022-10-24-1'
default_monorepo_dir = os.path.abspath(os.path.join(os.getcwd(), '..'))
log = logging.getLogger()


@click.group()
def group():
    pass


@group.command()
@click.option('--monorepo-dir', required=True, type=click.Path(exists=True), default=default_monorepo_dir)
@click.option('--snapshot-cache-dir', required=True, default='/tmp/testmig-snapshots')
@click.option('--work-dir', required=True, default='/tmp/testmig-workdir')
@click.option('--l1-url', required=True, default=os.getenv('TESTMIG_L1_URL'))
@click.option('--reset-work-dir', required=True, default=False)
def run_forked(monorepo_dir, snapshot_cache_dir, work_dir, l1_url, reset_work_dir):
    if not os.path.isdir(snapshot_cache_dir):
        os.makedirs(snapshot_cache_dir)

    if reset_work_dir:
        shutil.rmtree(work_dir)
    os.makedirs(work_dir, exist_ok=True)

    historical_mainnet_path = os.path.join(monorepo_dir, 'packages', 'contracts-bedrock', 'deployments', 'mainnet')
    if os.path.isdir(historical_mainnet_path):
        log.info('Removing historical mainnet deployment')
        print(historical_mainnet_path)
        shutil.rmtree(historical_mainnet_path)

    for archive in ('dtl', 'geth'):
        fp = os.path.join(snapshot_cache_dir, f'{archive}.tar.gz')
        if os.path.isfile(fp):
            log.info(f'{archive} archive already exists, not downloading')
        else:
            log.info(f'Downloading {archive} archive')
            with open(fp, 'wb+') as f:
                download_with_progress(f'{URL_PREFIX}/{archive}.tar.gz', f)

        outpath = os.path.join(work_dir, archive)
        donefile = os.path.join(outpath, 'DONE')
        if os.path.isfile(donefile):
            log.info(f'{archive} is already extracted')
        else:
            log.info(f'Extracting {archive} archive')
            os.makedirs(outpath)
            run_command(['tar', '-xzvf', fp, '--strip-components=6', '-C', outpath])
            Path(donefile).touch()

    client = docker.from_env()

    try:
        container = client.containers.get('testmig-l1')
        container.stop()
        container.remove()
        log.info('Stopped and removed old containers')
    except docker.errors.NotFound:
        pass

    log.info('Starting forked L1')
    container = client.containers.run('ethereumoptimism/hardhat-node:latest', detach=True, environment={
        'FORK_STARTING_BLOCK': '15822707',
        'FORK_URL': l1_url,
        'FORK_CHAIN_ID': '1'
    }, name='testmig-l1', ports={'8545/tcp': ('127.0.0.1', 8545)})

    w3 = None
    for i in range(0, 10):
        try:
            w3 = Web3(Web3.HTTPProvider('http://127.0.0.1:8545'))
            w3.eth.get_balance('0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266')
        except Exception as e:
            log.info('Polling for L1 to come up')
            w3 = None
            time.sleep(1)
    if w3 is None:
        raise Exception('Could not connect to Web3.')
    log.info('L1 is ready')

    for slot in SLOTS_TO_MODIFY:
        log.info(f'Setting storage slot on {slot[0]}')
        w3.provider.make_request('hardhat_setStorageAt', slot)

    log.info('Mining a block for good measure')
    w3.provider.make_request('evm_mine', [])

    log.info('Running L1 migration')
    run_command([
        'yarn',
        'hardhat',
        '--network',
        'mainnet-forked',
        'deploy',
        '--tags',
        'migration'
    ], env={
        'CHAIN_ID': '1',
        'L1_RPC': 'http://127.0.0.1:8545',
        'PRIVATE_KEY_DEPLOYER': 'ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
    }, cwd=os.path.join(monorepo_dir, 'packages', 'contracts-bedrock'))

    log.info('Cleaning up')
    container.stop()
    container.remove()


if __name__ == '__main__':
    group()
