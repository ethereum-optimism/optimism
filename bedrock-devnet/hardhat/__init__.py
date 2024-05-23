import argparse
import logging
import os
import subprocess
import json
import socket
import time
import shutil
import http.client

pjoin = os.path.join

parser = argparse.ArgumentParser(description='Bedrock devnet launcher')
parser.add_argument('--monorepo-dir', help='Directory of the monorepo', default=os.getcwd())
parser.add_argument('--allocs', help='Only create the allocs and exit', type=bool, action=argparse.BooleanOptionalAction)
parser.add_argument('--test', help='Tests the deployment, must already be deployed', type=bool, action=argparse.BooleanOptionalAction)

log = logging.getLogger()

DEV_ACCOUNTS = [
    '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266',
    '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
    '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC',
    '0x90F79bf6EB2c4f870365E785982E1f101E93b906',
    '0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65'
]

# The original L1 starting block tag and timestamp
L1STARTINGBLOCKTAG = '0xb21fa192d3169c824801af37775514f246d96b906eff24849c5bd240ccb23557'
L2OUTPUTORACLESTARTINGTIMESTAMP = 1693950295
L1BOBATOKENADDRESS = '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266'
UPGRADETIMEOFFSET = '0x0'

class Bunch:
    def __init__(self, **kwds):
        self.__dict__.update(kwds)

def main():
    args = parser.parse_args()

    monorepo_dir = os.path.abspath(args.monorepo_dir)
    devnet_dir = pjoin(monorepo_dir, '.devnet')
    contracts_bedrock_dir = pjoin(monorepo_dir, 'packages', 'contracts-bedrock')
    deployments_dir = pjoin(contracts_bedrock_dir, 'deployments')
    deployment_dir = pjoin(contracts_bedrock_dir, 'deployments', 'hardhat-local')
    op_node_dir = pjoin(args.monorepo_dir, 'op-node')
    ops_bedrock_dir = pjoin(monorepo_dir, 'ops-bedrock')
    deploy_config_dir = pjoin(contracts_bedrock_dir, 'deploy-config'),
    devnet_config_path = pjoin(contracts_bedrock_dir, 'deploy-config', 'hardhat-local.json')
    devnet_config_export_path = pjoin(contracts_bedrock_dir, 'deploy-config', 'hardhat-local.ts')
    ops_chain_ops = pjoin(monorepo_dir, 'op-chain-ops')
    boba_chain_ops = pjoin(monorepo_dir, 'boba-chain-ops')
    sdk_dir = pjoin(monorepo_dir, 'packages', 'sdk')
    env_dir = pjoin(contracts_bedrock_dir, '.env')

    paths = Bunch(
      mono_repo_dir=monorepo_dir,
      devnet_dir=devnet_dir,
      contracts_bedrock_dir=contracts_bedrock_dir,
      deployments_dir=deployments_dir,
      deployment_dir=deployment_dir,
      l1_deployments_path=pjoin(deployment_dir, '.deploy'),
      deploy_config_dir=deploy_config_dir,
      devnet_config_path=devnet_config_path,
      devnet_config_export_path=devnet_config_export_path,
      op_node_dir=op_node_dir,
      ops_bedrock_dir=ops_bedrock_dir,
      ops_chain_ops=ops_chain_ops,
      boba_chain_ops=boba_chain_ops,
      sdk_dir=sdk_dir,
      env_dir=env_dir,
      genesis_l1_path=pjoin(devnet_dir, 'genesis-l1.json'),
      genesis_l2_path=pjoin(devnet_dir, 'genesis-l2.json'),
      allocs_path=pjoin(devnet_dir, 'allocs-l1.json'),
      addresses_json_path=pjoin(devnet_dir, 'addresses.json'),
      sdk_addresses_json_path=pjoin(devnet_dir, 'sdk-addresses.json'),
      rollup_config_path=pjoin(devnet_dir, 'rollup.json')
    )

    if args.test:
      log.info('Testing deployed devnet')
      devnet_test(paths)
      return

    os.makedirs(devnet_dir, exist_ok=True)

    log.info('Devnet starting')
    devnet_l1_genesis(paths)
    devnet_bring_l1(paths)
    devnet_write_env(paths)
    devnet_deploy(paths)
    devnet_generate_files(paths)
    devnet_bring_l2(paths)
    devnet_bring_op_node(paths)
    devnet_bring_batcher_proposer(paths)
    devnet_store_addresses(paths)
    devent_restore_configurations(paths)

def devnet_l1_genesis(paths):
    # Create the allocs
    allocs = {}
    for account in DEV_ACCOUNTS:
        allocs[account] = {"balance": "0x7C75D695C2706AC5E97044C3B2D3EF5929948000000000000000"}
    write_json(paths.allocs_path, allocs)
    outfile_l1 = pjoin(paths.devnet_dir, 'genesis-l1.json')

    run_command([
        'go', 'run', 'cmd/main.go', 'genesis', 'l1-clean',
        '--deploy-config', paths.devnet_config_path,
        '--l1-allocs', paths.allocs_path,
        '--outfile.l1', outfile_l1,
    ], cwd=paths.op_node_dir)

# Bring up the devnet where the contracts are deployed to L1
def devnet_bring_l1(paths):
    log.info('Starting L1.')
    run_command(['docker', 'compose', 'up', '-d', 'l1'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir
    })
    wait_up(8545)
    wait_for_rpc_server('127.0.0.1:8545')

def devnet_write_env(paths):
    log.info("Writing env file")
    with open(paths.env_dir, 'w+') as f:
        f.write("""
L1_RPC=http://127.0.0.1:8545
PRIVATE_KEY_DEPLOYER=ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
ENABLE_BOBA_TOKEN_DEPLOYMENT=true
""")

    block_info = eth_block("127.0.0.1:8545")
    block_info = json.loads(block_info)
    block_result = block_info['result']
    with open(paths.devnet_config_path, 'r') as f:
        config = json.load(f)
        config['l1StartingBlockTag'] = block_result['hash']
        config['l2OutputOracleStartingTimestamp'] = int(block_result['timestamp'], 0)
        config['controller'] = DEV_ACCOUNTS[0]
        config['l1BobaTokenAddress'] = DEV_ACCOUNTS[0]
        config['l2GenesisRegolithTimeOffset'] = block_result['timestamp']
        config['l2GenesisCanyonTimeOffset'] = block_result['timestamp']
        config['l2GenesisDeltaTimeOffset'] = block_result['timestamp']
        # L1 beacon endpoint is not available in the devnet
        if 'l2GenesisEcotoneTimeOffset' in config:
            del config['l2GenesisEcotoneTimeOffset']
    write_json(paths.devnet_config_path, config)

    with open(paths.devnet_config_export_path, 'w+') as f:
        f.write("""
import { DeployConfig } from '../scripts/deploy-config'
import config from './hardhat-local.json'

export default config satisfies DeployConfig
""")

def devnet_deploy(paths):
    if os.path.exists(paths.deployment_dir):
        shutil.rmtree(paths.deployment_dir)

    run_command([
        'yarn', 'deploy:hardhat', '--network', 'hardhat-local'
    ], cwd=paths.contracts_bedrock_dir)

    if os.path.exists(paths.devnet_config_export_path):
      os.remove(paths.devnet_config_export_path)

def devnet_generate_files(paths):
    log.info('Generating L2 genesis and rollup config')

    BOBA_deployment = read_json(pjoin(paths.deployment_dir, 'BOBA.json'))
    boba_token_address = BOBA_deployment['address']
    log.info(f'Using l1 boba token {boba_token_address}')

    with open(paths.devnet_config_path, 'r') as f:
        config = json.load(f)
        config['l1BobaTokenAddress'] = boba_token_address
    write_json(paths.devnet_config_path, config)

    run_command([
        'go', 'run', './cmd/boba-devnet',
        '--l1-rpc', 'http://localhost:8545',
        '--deploy-config', paths.devnet_config_path,
        '--hardhat-deployments', paths.deployments_dir,
        '--network', 'hardhat-local',
        '--outfile-l2', pjoin(paths.devnet_dir, 'genesis-l2.json'),
        '--outfile-rollup', pjoin(paths.devnet_dir, 'rollup.json')
    ], cwd=paths.boba_chain_ops)

    run_command(
        'openssl rand -hex 32 > test-jwt-secret.txt',
        shell=True,
        cwd=paths.devnet_dir
    )

def devnet_bring_l2(paths):
    log.info('Bringing up L2.')
    run_command(['docker', 'compose', 'up', '-d', 'l2'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir
    })
    wait_up(8546)
    wait_for_rpc_server('127.0.0.1:9545')

def devnet_bring_op_node(paths):
    log.info('Bringing up op-node.')
    run_command(['docker', 'compose', 'up', '-d', 'op-node'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir,
        'PLASMA_ENABLED': 'false',
        'PLASMA_DA_SERVICE': 'false',
    })

def devnet_bring_batcher_proposer(paths):
    log.info('Bringing up batcher and proposer.')

    L2OO_deployment = read_json(pjoin(paths.deployment_dir, 'L2OutputOracleProxy.json'))
    l2_output_oracle = L2OO_deployment['address']
    log.info(f'Using L2OutputOracle {l2_output_oracle}')

    rollup_config = read_json(paths.rollup_config_path)
    batch_inbox_address = rollup_config['batch_inbox_address']
    log.info(f'Using batch inbox {batch_inbox_address}')

    run_command(['docker', 'compose', 'up', '-d', 'op-batcher', 'op-proposer'], cwd=paths.ops_bedrock_dir, env={
        'PWD': paths.ops_bedrock_dir,
        'L2OO_ADDRESS': l2_output_oracle,
        'SEQUENCER_BATCH_INBOX_ADDRESS': batch_inbox_address,
        'PLASMA_ENABLED': 'false',
        'PLASMA_DA_SERVICE': 'false',
    })

def devnet_store_addresses(paths):
    addresses = {}
    addresses_name = {
        'AddressManager': 'Lib_AddressManager.json',
        'SuperchainConfig': 'SuperchainConfig.json',
        'SuperchainConfigProxy': 'SuperchainConfigProxy.json',
        'L1CrossDomainMessenger': 'L1CrossDomainMessenger.json',
        'L1CrossDomainMessengerProxy': 'Proxy__OVM_L1CrossDomainMessenger.json',
        'L1ERC721Bridge': 'L1ERC721Bridge.json',
        'L1ERC721BridgeProxy': 'L1ERC721BridgeProxy.json',
        'L1StandardBridge': 'L1StandardBridge.json',
        'L1StandardBridgeProxy': 'Proxy__OVM_L1StandardBridge.json',
        'L2OutputOracle': 'L2OutputOracle.json',
        'L2OutputOracleProxy': 'L2OutputOracleProxy.json',
        'OptimismMintableERC20Factory': 'OptimismMintableERC20Factory.json',
        'OptimismMintableERC20FactoryProxy': 'OptimismMintableERC20FactoryProxy.json',
        'OptimismPortal': 'OptimismPortal.json',
        'OptimismPortalProxy': 'OptimismPortalProxy.json',
        'ProxyAdmin': 'ProxyAdmin.json',
        'SystemConfig': 'SystemConfig.json',
        'SystemConfigProxy': 'SystemConfigProxy.json',
        'ProtocolVersions': 'ProtocolVersions.json',
        'ProtocolVersionsProxy': 'ProtocolVersionsProxy.json',
        'BOBA': 'BOBA.json',
    }

    for k, v in addresses_name.items():
        deployment = read_json(pjoin(paths.deployment_dir, v))
        addresses[k] = deployment['address']
    write_json(paths.addresses_json_path, addresses)

def devent_restore_configurations(paths):
    with open(paths.devnet_config_path, 'r') as f:
        config = json.load(f)
        config['l1StartingBlockTag'] = L1STARTINGBLOCKTAG
        config['l2OutputOracleStartingTimestamp'] = L2OUTPUTORACLESTARTINGTIMESTAMP
        config['l1BobaTokenAddress'] = L1BOBATOKENADDRESS
        config['l2GenesisRegolithTimeOffset'] = UPGRADETIMEOFFSET
        config['l2GenesisCanyonTimeOffset'] = UPGRADETIMEOFFSET
        config['l2GenesisDeltaTimeOffset'] = UPGRADETIMEOFFSET
        config['l2GenesisEcotoneTimeOffset'] = UPGRADETIMEOFFSET
    write_json(paths.devnet_config_path, config)

def eth_block(url):
    log.info(f'Fetch eth_getBlockByNumber {url}')
    conn = http.client.HTTPConnection(url)
    headers = {'Content-type': 'application/json'}
    body = '{"id":2, "jsonrpc":"2.0", "method": "eth_getBlockByNumber", "params":["latest", false]}'
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

def devnet_test(paths):
    run_command(
        ['go', 'run', './cmd/check-l2/main.go', '--l2-rpc-url', 'http://localhost:9545', '--l1-rpc-url', 'http://localhost:8545'],
        cwd=paths.boba_chain_ops,
    )

    run_command(
         ['npx', 'hardhat',  'deposit-eth', '--network',  'hardhat-local', '--l1-contracts-json-path', paths.addresses_json_path],
         cwd=paths.sdk_dir,
         timeout=12*60,
    )

    run_command(
         ['npx', 'hardhat',  'deposit-erc20', '--network',  'hardhat-local', '--l1-contracts-json-path', paths.addresses_json_path],
         cwd=paths.sdk_dir,
         timeout=12*60,
    )

    run_command(
         ['npx', 'hardhat',  'deposit-boba', '--network',  'hardhat-local', '--l1-contracts-json-path', paths.addresses_json_path],
         cwd=paths.sdk_dir,
         timeout=12*60,
    )

def run_command(args, check=True, shell=False, cwd=None, env=None, timeout=None):
    env = env if env else {}
    return subprocess.run(
        args,
        check=check,
        shell=shell,
        env={
            **os.environ,
            **env
        },
        cwd=cwd,
        timeout=timeout
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
