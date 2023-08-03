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
import http.client
import multiprocessing

import devnet.log_setup

pjoin = os.path.join

parser = argparse.ArgumentParser(description='Bedrock devnet launcher')
parser.add_argument('--monorepo-dir', help='Directory of the monorepo', default=os.getcwd())
parser.add_argument('--allocs', help='Only create the allocs and exit', type=bool, action=argparse.BooleanOptionalAction)

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
    ops_bedrock_dir = pjoin(monorepo_dir, 'ops-bedrock')
    deploy_config_dir = pjoin(contracts_bedrock_dir, 'deploy-config'),
    devnet_config_path = pjoin(contracts_bedrock_dir, 'deploy-config', 'devnetL1.json')

    paths = Bunch(
      mono_repo_dir=monorepo_dir,
      devnet_dir=devnet_dir,
      contracts_bedrock_dir=contracts_bedrock_dir,
      deployment_dir=deployment_dir,
      l1_deployments_path=pjoin(deployment_dir, '.deploy'),
      deploy_config_dir=deploy_config_dir,
      devnet_config_path=devnet_config_path,
      op_node_dir=op_node_dir,
      ops_bedrock_dir=ops_bedrock_dir,
      genesis_l1_path=pjoin(devnet_dir, 'genesis-l1.json'),
      genesis_l2_path=pjoin(devnet_dir, 'genesis-l2.json'),
      allocs_path=pjoin(devnet_dir, 'allocs-l1.json'),
      addresses_json_path=pjoin(devnet_dir, 'addresses.json'),
      sdk_addresses_json_path=pjoin(devnet_dir, 'sdk-addresses.json'),
      rollup_config_path=pjoin(devnet_dir, 'rollup.json')
    )

    os.makedirs(devnet_dir, exist_ok=True)

    if args.allocs:
        devnet_l1_genesis(paths)
        return

    log.info('Building docker images')
    run_command(['docker-compose', 'build', '--progress', 'plain'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir
    })

    log.info('Devnet starting')
    devnet_deploy(paths)


def deploy_contracts(paths):
    def internal():
        wait_up(8545)
        wait_for_rpc_server('127.0.0.1:8545')
        res = eth_accounts('127.0.0.1:8545')

        response = json.loads(res)
        account = response['result'][0]

        fqn = 'scripts/Deploy.s.sol:Deploy'
        run_command([
            'forge', 'script', fqn, '--sender', account,
            '--rpc-url', 'http://127.0.0.1:8545', '--broadcast',
            '--unlocked'
        ], env={}, cwd=paths.contracts_bedrock_dir)

        shutil.copy(paths.l1_deployments_path, paths.addresses_json_path)

        log.info('Syncing contracts.')
        run_command([
            'forge', 'script', fqn, '--sig', 'sync()',
            '--rpc-url', 'http://127.0.0.1:8545'
        ], env={}, cwd=paths.contracts_bedrock_dir)

    return internal


def devnet_l1_genesis(paths):
    log.info('Generating L1 genesis state')
    geth = subprocess.Popen([
        'geth', '--dev', '--http', '--http.api', 'eth,debug',
        '--verbosity', '4', '--gcmode', 'archive', '--dev.gaslimit', '30000000'
    ])

    forge = multiprocessing.Process(target=deploy_contracts(paths))
    forge.start()
    forge.join()

    res = debug_dumpBlock('127.0.0.1:8545')
    response = json.loads(res)
    allocs = response['result']

    write_json(paths.allocs_path, allocs)
    geth.terminate()


# Bring up the devnet where the contracts are deployed to L1
def devnet_deploy(paths):
    if os.path.exists(paths.genesis_l1_path):
        log.info('L1 genesis already generated.')
    else:
        log.info('Generating L1 genesis.')
        if os.path.exists(paths.allocs_path) == False:
            devnet_l1_genesis(paths)

        devnet_config_backup = pjoin(paths.devnet_dir, 'devnetL1.json.bak')
        shutil.copy(paths.devnet_config_path, devnet_config_backup)
        deploy_config = read_json(paths.devnet_config_path)
        deploy_config['l1GenesisBlockTimestamp'] = '{:#x}'.format(int(time.time()))
        write_json(paths.devnet_config_path, deploy_config)
        outfile_l1 = pjoin(paths.devnet_dir, 'genesis-l1.json')

        run_command([
            'go', 'run', 'cmd/main.go', 'genesis', 'l1',
            '--deploy-config', paths.devnet_config_path,
            '--l1-allocs', paths.allocs_path,
            '--l1-deployments', paths.addresses_json_path,
            '--outfile.l1', outfile_l1,
        ], cwd=paths.op_node_dir)

    log.info('Starting L1.')
    run_command(['docker-compose', 'up', '-d', 'l1'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir
    })
    wait_up(8545)
    wait_for_rpc_server('127.0.0.1:8545')

    if os.path.exists(paths.genesis_l2_path):
        log.info('L2 genesis and rollup configs already generated.')
    else:
        log.info('Generating L2 genesis and rollup configs.')
        run_command([
            'go', 'run', 'cmd/main.go', 'genesis', 'l2',
            '--l1-rpc', 'http://localhost:8545',
            '--deploy-config', paths.devnet_config_path,
            '--deployment-dir', paths.deployment_dir,
            '--outfile.l2', pjoin(paths.devnet_dir, 'genesis-l2.json'),
            '--outfile.rollup', pjoin(paths.devnet_dir, 'rollup.json')
        ], cwd=paths.op_node_dir)

    rollup_config = read_json(paths.rollup_config_path)
    addresses = read_json(paths.addresses_json_path)

    log.info('Bringing up L2.')
    run_command(['docker-compose', 'up', '-d', 'l2'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir
    })
    wait_up(9545)
    wait_for_rpc_server('127.0.0.1:9545')

    l2_output_oracle = addresses['L2OutputOracleProxy']
    log.info(f'Using L2OutputOracle {l2_output_oracle}')
    batch_inbox_address = rollup_config['batch_inbox_address']
    log.info(f'Using batch inbox {batch_inbox_address}')

    log.info('Bringing up everything else.')
    run_command(['docker-compose', 'up', '-d', 'op-node', 'op-proposer', 'op-batcher'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir,
        'L2OO_ADDRESS': l2_output_oracle,
        'SEQUENCER_BATCH_INBOX_ADDRESS': batch_inbox_address
    })

    log.info('Devnet ready.')


def eth_accounts(url):
    log.info(f'Fetch eth_accounts {url}')
    conn = http.client.HTTPConnection(url)
    headers = {'Content-type': 'application/json'}
    body = '{"id":2, "jsonrpc":"2.0", "method": "eth_accounts", "params":[]}'
    conn.request('POST', '/', body, headers)
    response = conn.getresponse()
    data = response.read().decode()
    conn.close()
    return data


def debug_dumpBlock(url):
    log.info(f'Fetch debug_dumpBlock {url}')
    conn = http.client.HTTPConnection(url)
    headers = {'Content-type': 'application/json'}
    body = '{"id":3, "jsonrpc":"2.0", "method": "debug_dumpBlock", "params":["latest"]}'
    conn.request('POST', '/', body, headers)
    response = conn.getresponse()
    data = response.read().decode()
    conn.close()
    return data


def wait_for_rpc_server(url):
    log.info(f'Waiting for RPC server at {url}')

    conn = http.client.HTTPConnection(url)
    headers = {'Content-type': 'application/json'}
    body = '{"id":1, "jsonrpc":"2.0", "method": "eth_chainId", "params":[]}'

    while True:
        try:
            conn.request('POST', '/', body, headers)
            response = conn.getresponse()
            conn.close()
            if response.status < 300:
                log.info(f'RPC server at {url} ready')
                return
        except Exception as e:
            log.info(f'Waiting for RPC server at {url}')
            time.sleep(1)


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
