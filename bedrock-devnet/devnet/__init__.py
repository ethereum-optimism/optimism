import argparse
import logging
import os
import subprocess
import json
import socket
import calendar
import datetime
import time
import shutil

import devnet.log_setup
from devnet.genesis import GENESIS_TMPL

pjoin = os.path.join

parser = argparse.ArgumentParser(description='Bedrock devnet launcher')
parser.add_argument('--monorepo-dir', help='Directory of the monorepo', default=os.getcwd())
parser.add_argument('--deploy', help='Whether the contracts should be predeployed or deployed', type=bool, action=argparse.BooleanOptionalAction)

log = logging.getLogger()

class Bunch:
    def __init__(self, **kwds):
        self.__dict__.update(kwds)

def main():
    args = parser.parse_args()

    monorepo_dir = os.path.abspath(args.monorepo_dir)
    devnet_dir = pjoin(monorepo_dir, '.devnet')
    contracts_bedrock_dir = pjoin(monorepo_dir, 'packages', 'contracts-bedrock')
    deployment_dir = pjoin(contracts_bedrock_dir, 'deployments', 'devnetL1')
    op_node_dir = pjoin(args.monorepo_dir, 'op-node')
    ops_bedrock_dir=pjoin(monorepo_dir, 'ops-bedrock')

    paths = Bunch(
      mono_repo_dir=monorepo_dir,
      devnet_dir=devnet_dir,
      contracts_bedrock_dir=contracts_bedrock_dir,
      deployment_dir=deployment_dir,
      deploy_config_dir=pjoin(contracts_bedrock_dir, 'deploy-config'),
      op_node_dir=op_node_dir,
      ops_bedrock_dir=ops_bedrock_dir,
      genesis_l1_path=pjoin(devnet_dir, 'genesis-l1.json'),
      genesis_l2_path=pjoin(devnet_dir, 'genesis-l2.json'),
      addresses_json_path=pjoin(devnet_dir, 'addresses.json'),
      sdk_addresses_json_path=pjoin(devnet_dir, 'sdk-addresses.json'),
      rollup_config_path=pjoin(devnet_dir, 'rollup.json')
    )

    os.makedirs(devnet_dir, exist_ok=True)

    if args.deploy:
      log.info('Devnet with upcoming smart contract deployments')
      devnet_deploy(paths)
    else:
      log.info('Devnet with smart contracts pre-deployed')
      devnet_prestate(paths)

# Bring up the devnet where the L1 contracts are in the genesis state
def devnet_prestate(paths):
    date = datetime.datetime.utcnow()
    utc_time = hex(calendar.timegm(date.utctimetuple()))

    done_file = pjoin(paths.devnet_dir, 'done')
    if os.path.exists(done_file):
        log.info('Genesis files already exist')
    else:
        log.info('Creating genesis files')
        deploy_config_path = pjoin(paths.deploy_config_dir, 'devnetL1.json')

        # read the json file
        deploy_config = read_json(deploy_config_path)
        deploy_config['l1GenesisBlockTimestamp'] = utc_time
        temp_deploy_config = pjoin(paths.devnet_dir, 'deploy-config.json')
        write_json(temp_deploy_config, deploy_config)

        outfile_l1 = paths.genesis_l1_path
        outfile_l2 = paths.genesis_l2_path
        outfile_rollup = paths.rollup_config_path

        run_command(['go', 'run', 'cmd/main.go', 'genesis', 'devnet', '--deploy-config', temp_deploy_config, '--outfile.l1', outfile_l1, '--outfile.l2', outfile_l2, '--outfile.rollup', outfile_rollup], cwd=paths.op_node_dir)
        write_json(done_file, {})

    log.info('Bringing up L1.')
    run_command(['docker-compose', 'up', '-d', 'l1'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir
    })
    wait_up(8545)

    log.info('Bringing up L2.')
    run_command(['docker-compose', 'up', '-d', 'l2'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir
    })
    wait_up(9545)

    log.info('Bringing up the services.')
    run_command(['docker-compose', 'up', '-d', 'op-proposer', 'op-batcher'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir,
        'L2OO_ADDRESS': '0x6900000000000000000000000000000000000000'
    })

# Bring up the devnet where the contracts are deployed to L1
def devnet_deploy(paths):
    if os.path.exists(paths.genesis_l1_path):
        log.info('L1 genesis already generated.')
    else:
        log.info('Generating L1 genesis.')
        write_json(paths.genesis_l1_path, GENESIS_TMPL)

    log.info('Starting L1.')
    run_command(['docker-compose', 'up', '-d', 'l1'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir
    })
    wait_up(8545)

    log.info('Generating network config.')
    devnet_cfg_orig = pjoin(paths.contracts_bedrock_dir, 'deploy-config', 'devnetL1.json')
    devnet_cfg_backup = pjoin(paths.devnet_dir, 'devnetL1.json.bak')
    shutil.copy(devnet_cfg_orig, devnet_cfg_backup)
    deploy_config = read_json(devnet_cfg_orig)
    deploy_config['l1GenesisBlockTimestamp'] = GENESIS_TMPL['timestamp']
    deploy_config['l1StartingBlockTag'] = 'earliest'
    write_json(devnet_cfg_orig, deploy_config)

    if os.path.exists(paths.addresses_json_path):
        log.info('Contracts already deployed.')
        addresses = read_json(paths.addresses_json_path)
    else:
        log.info('Deploying contracts.')
        run_command(['yarn', 'hardhat', '--network', 'devnetL1', 'deploy', '--tags', 'l1'], env={
            'CHAIN_ID': '900',
            'L1_RPC': 'http://localhost:8545',
            'PRIVATE_KEY_DEPLOYER': 'ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80'
        }, cwd=paths.contracts_bedrock_dir)
        contracts = os.listdir(paths.deployment_dir)
        addresses = {}
        for c in contracts:
            if not c.endswith('.json'):
                continue
            data = read_json(pjoin(paths.deployment_dir, c))
            addresses[c.replace('.json', '')] = data['address']
        sdk_addresses = {}
        sdk_addresses.update({
            'AddressManager': '0x0000000000000000000000000000000000000000',
            'StateCommitmentChain': '0x0000000000000000000000000000000000000000',
            'CanonicalTransactionChain': '0x0000000000000000000000000000000000000000',
            'BondManager': '0x0000000000000000000000000000000000000000',
        })
        sdk_addresses['L1CrossDomainMessenger'] = addresses['Proxy__OVM_L1CrossDomainMessenger']
        sdk_addresses['L1StandardBridge'] = addresses['Proxy__OVM_L1StandardBridge']
        sdk_addresses['OptimismPortal'] = addresses['OptimismPortalProxy']
        sdk_addresses['L2OutputOracle'] = addresses['L2OutputOracleProxy']
        write_json(paths.addresses_json_path, addresses)
        write_json(paths.sdk_addresses_json_path, sdk_addresses)

    if os.path.exists(paths.genesis_l2_path):
        log.info('L2 genesis and rollup configs already generated.')
    else:
        log.info('Generating L2 genesis and rollup configs.')
        run_command([
            'go', 'run', 'cmd/main.go', 'genesis', 'l2',
            '--l1-rpc', 'http://localhost:8545',
            '--deploy-config', devnet_cfg_orig,
            '--deployment-dir', paths.deployment_dir,
            '--outfile.l2', pjoin(paths.devnet_dir, 'genesis-l2.json'),
            '--outfile.rollup', pjoin(paths.devnet_dir, 'rollup.json')
        ], cwd=paths.op_node_dir)

    rollup_config = read_json(paths.rollup_config_path)

    if os.path.exists(devnet_cfg_backup):
        shutil.move(devnet_cfg_backup, devnet_cfg_orig)

    log.info('Bringing up L2.')
    run_command(['docker-compose', 'up', '-d', 'l2'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir
    })
    wait_up(9545)

    log.info('Bringing up everything else.')
    run_command(['docker-compose', 'up', '-d', 'op-node', 'op-proposer', 'op-batcher'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir,
        'L2OO_ADDRESS': addresses['L2OutputOracleProxy'],
        'SEQUENCER_BATCH_INBOX_ADDRESS': rollup_config['batch_inbox_address']
    })

    log.info('Devnet ready.')


def run_command(args, check=True, shell=False, cwd=None, env=None):
    env = env if env else {}
    return subprocess.run(
        args,
        check=check,
        shell=shell,
        env={
            **os.environ,
            **env
        },
        cwd=cwd
    )


def wait_up(port, retries=10, wait_secs=1):
    for i in range(0, retries):
        log.info(f'Trying 127.0.0.1:{port}')
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        try:
            s.connect(('127.0.0.1', int(port)))
            s.shutdown(2)
            log.info(f'Connected 127.0.0.1:{port}')
            return True
        except Exception:
            time.sleep(wait_secs)

    raise Exception(f'Timed out waiting for port {port}.')


def write_json(path, data):
    with open(path, 'w+') as f:
        json.dump(data, f, indent='  ')


def read_json(path):
    with open(path, 'r') as f:
        return json.load(f)
