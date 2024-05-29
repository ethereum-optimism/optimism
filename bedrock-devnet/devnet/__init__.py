import argparse
import dataclasses
import logging
import os
import subprocess
import json
import socket
import datetime
import time
import shutil
import http.client
import concurrent.futures
from collections import namedtuple

import devnet.log_setup

pjoin = os.path.join
parser = argparse.ArgumentParser(description='Bedrock devnet launcher')
parser.add_argument('--monorepo-dir', help='Directory of the monorepo', default=os.getcwd())
parser.add_argument('--allocs', help='Only create the allocs and exit', type=bool, action=argparse.BooleanOptionalAction)
parser.add_argument('--test', help='Tests the deployment, must already be deployed', type=bool, action=argparse.BooleanOptionalAction)

devnet.log_setup.setup_logging()
log = logging.getLogger()

# Global environment variables
DEVNET_NO_BUILD = os.getenv('DEVNET_NO_BUILD') == "true"
DEVNET_FPAC = os.getenv('DEVNET_FPAC') == "true"
DEVNET_PLASMA = os.getenv('DEVNET_PLASMA') == "true"


@dataclasses.dataclass
class Config:
    mono_repo_dir: str
    devnet_dir: str
    contracts_bedrock_dir: str
    deployment_dir: str
    forge_l1_dump_path: str
    l1_deployments_path: str
    deploy_config_dir: str
    devnet_config_path: str
    devnet_config_template_path: str
    op_node_dir: str
    ops_bedrock_dir: str
    ops_chain_ops: str
    sdk_dir: str
    genesis_l1_path: str
    genesis_l2_path: str
    allocs_l1_path: str
    addresses_json_path: str
    sdk_addresses_json_path: str
    rollup_config_path: str


def main():
    args = parser.parse_args()
    monorepo_dir = os.path.abspath(args.monorepo_dir)
    devnet_dir = pjoin(monorepo_dir, '.devnet')
    contracts_bedrock_dir = pjoin(monorepo_dir, 'packages', 'contracts-bedrock')
    deployment_dir = pjoin(contracts_bedrock_dir, 'deployments', 'devnetL1')
    deploy_config_dir = pjoin(contracts_bedrock_dir, 'deploy-config')

    config = Config(
        mono_repo_dir=os.path.abspath(args.monorepo_dir),
        devnet_dir=devnet_dir,
        contracts_bedrock_dir=contracts_bedrock_dir,
        deployment_dir=pjoin(contracts_bedrock_dir, 'deployments', 'devnetL1'),
        forge_l1_dump_path=pjoin(contracts_bedrock_dir, 'state-dump-900.json'),
        l1_deployments_path=pjoin(deployment_dir, '.deploy'),
        deploy_config_dir=pjoin(contracts_bedrock_dir, 'deploy-config'),
        devnet_config_path=pjoin(deploy_config_dir, 'devnetL1.json'),
        devnet_config_template_path=pjoin(deploy_config_dir, 'devnetL1-template.json'),
        op_node_dir=pjoin(args.monorepo_dir, 'op-node'),
        ops_bedrock_dir=pjoin(monorepo_dir, 'ops-bedrock'),
        ops_chain_ops=pjoin(monorepo_dir, 'op-chain-ops'),
        sdk_dir=pjoin(monorepo_dir, 'packages', 'sdk'),
        genesis_l1_path=pjoin(devnet_dir, 'genesis-l1.json'),
        genesis_l2_path=pjoin(devnet_dir, 'genesis-l2.json'),
        allocs_l1_path=pjoin(devnet_dir, 'allocs-l1.json'),
        addresses_json_path=pjoin(devnet_dir, 'addresses.json'),
        sdk_addresses_json_path=pjoin(devnet_dir, 'sdk-addresses.json'),
        rollup_config_path=pjoin(devnet_dir, 'rollup.json')
    )

    if args.test:
        log.info('Testing deployed devnet')
        devnet_test(config)
        return

    os.makedirs(devnet_dir, exist_ok=True)

    if args.allocs:
        devnet_l1_allocs(config)
        devnet_l2_allocs(config)
        return

    git_commit = subprocess.run(['git', 'rev-parse', 'HEAD'], capture_output=True, text=True).stdout.strip()
    git_date = subprocess.run(['git', 'show', '-s', "--format=%ct"], capture_output=True, text=True).stdout.strip()

    # Create shared network device for multiple containers to communicate
    run_command(['docker', 'network', 'create', '-d', 'bridge', 'ops-bedrock-shared'], cwd=config.ops_bedrock_dir)

    # CI loads the images from workspace, and does not otherwise know the images are good as-is
    if DEVNET_NO_BUILD:
        log.info('Skipping docker images build')
    else:
        log.info(f'Building docker images for git commit {git_commit} ({git_date})')
        log.info('Building docker images for L1')
        run_command(['docker', 'compose',
                     '-f', 'docker-compose.l1.yml',
                     'build',
                     '--progress', 'plain',
                     '--build-arg', f'GIT_COMMIT={git_commit}',
                     '--build-arg', f'GIT_DATE={git_date}'
                     ],
                    cwd=config.ops_bedrock_dir, env={
            'PWD': config.ops_bedrock_dir,
            'DOCKER_BUILDKIT': '1', # (should be available by default in later versions, but explicitly enable it anyway)
            'COMPOSE_DOCKER_CLI_BUILD': '1'  # use the docker cache
        })

        log.info('Building docker images for L2')
        run_command(['docker', 'compose',
                     '-f', 'docker-compose.l2.yml',
                     'build',
                     '--progress', 'plain',
                     '--build-arg', f'GIT_COMMIT={git_commit}',
                     '--build-arg', f'GIT_DATE={git_date}'
                     ],
                    cwd=config.ops_bedrock_dir, env={
            'PWD': config.ops_bedrock_dir,
            'DOCKER_BUILDKIT': '1', # (should be available by default in later versions, but explicitly enable it anyway)
            'COMPOSE_DOCKER_CLI_BUILD': '1'  # use the docker cache
        })


    log.info('Devnet starting')
    devnet_deploy(config)


def init_devnet_l1_deploy_config(config, update_timestamp=False):
    deploy_config = read_json(config.devnet_config_template_path)
    if update_timestamp:
        deploy_config['l1GenesisBlockTimestamp'] = '{:#x}'.format(int(time.time()))
    if DEVNET_FPAC:
        deploy_config['useFaultProofs'] = True
        deploy_config['faultGameMaxClockDuration'] = 10
        deploy_config['faultGameWithdrawalDelay'] = 0
    if DEVNET_PLASMA:
        deploy_config['usePlasma'] = True
    write_json(config.devnet_config_path, deploy_config)


def devnet_l1_allocs(config):
    log.info('Generating L1 genesis allocs')
    init_devnet_l1_deploy_config(config)

    fqn = 'scripts/Deploy.s.sol:Deploy'
    run_command([
        # We need to set the sender here to an account we know the private key of,
        # because the sender ends up being the owner of the ProxyAdmin SAFE
        # (which we need to enable the Custom Gas Token feature).
        'forge', 'script', fqn, "--sig", "runWithStateDump()", "--sender", "0x90F79bf6EB2c4f870365E785982E1f101E93b906"
    ], env={
        'DEPLOYMENT_OUTFILE': config.l1_deployments_path,
        'DEPLOY_CONFIG_PATH': config.devnet_config_path,
    }, cwd=config.contracts_bedrock_dir)

    shutil.move(src=config.forge_l1_dump_path, dst=config.allocs_l1_path)

    shutil.copy(config.l1_deployments_path, config.addresses_json_path)


def devnet_l2_allocs(config):
    log.info('Generating L2 genesis allocs, with L1 addresses: '+config.l1_deployments_path)

    fqn = 'scripts/L2Genesis.s.sol:L2Genesis'
    run_command([
        'forge', 'script', fqn, "--sig", "runWithAllUpgrades()"
    ], env={
      'CONTRACT_ADDRESSES_PATH': config.l1_deployments_path,
      'DEPLOY_CONFIG_PATH': config.devnet_config_path,
    }, cwd=config.contracts_bedrock_dir)

    # For the previous forks, and the latest fork (default, thus empty prefix),
    # move the forge-dumps into place as .devnet allocs.
    for suffix in ["-delta", "-ecotone", ""]:
        input_path = pjoin(config.contracts_bedrock_dir, f"state-dump-901{suffix}.json")
        output_path = pjoin(config.devnet_dir, f'allocs-l2{suffix}.json')
        shutil.move(src=input_path, dst=output_path)
        log.info("Generated L2 allocs: "+output_path)


def devnet_deploy_l1(config):
    if os.path.exists(config.genesis_l1_path):
        log.info('L1 genesis already generated.')
    else:
        log.info('Generating L1 genesis.')
        if not os.path.exists(config.allocs_l1_path) or DEVNET_FPAC or DEVNET_PLASMA:
            # If this is the FPAC devnet then we need to generate the allocs
            # file here always. This is because CI will run devnet-allocs
            # without DEVNET_FPAC=true which means the allocs will be wrong.
            # Re-running this step means the allocs will be correct.
            devnet_l1_allocs(config)
        else:
            log.info('Re-using existing L1 allocs.')

        # It's odd that we want to regenerate the devnetL1.json file with
        # an updated timestamp different than the one used in the devnet_l1_allocs
        # function.  But, without it, CI flakes on this test rather consistently.
        # If someone reads this comment and understands why this is being done, please
        # update this comment to explain.
        init_devnet_l1_deploy_config(config, update_timestamp=True)
        run_command([
            'go', 'run', 'cmd/main.go', 'genesis', 'l1',
            '--deploy-config', config.devnet_config_path,
            '--l1-allocs', config.allocs_l1_path,
            '--l1-deployments', config.addresses_json_path,
            '--outfile.l1', config.genesis_l1_path,
        ], cwd=config.op_node_dir)

    log.info('Starting L1.')
    run_command(['docker', 'compose', 'up', '-d', 'l1'], cwd=config.ops_bedrock_dir, env={
        'PWD': config.ops_bedrock_dir
    })
    wait_up(8545)
    wait_for_rpc_server('127.0.0.1:8545')


def devnet_deploy_l2(config):
    if os.path.exists(config.genesis_l2_path):
        log.info('L2 genesis and rollup configs already generated.')
    else:
        log.info('Generating L2 genesis and rollup configs.')
        l2_allocs_path = pjoin(config.devnet_dir, 'allocs-l2.json')
        if os.path.exists(l2_allocs_path) == False or DEVNET_FPAC == True:
            # Also regenerate if FPAC.
            # The FPAC flag may affect the L1 deployments addresses, which may affect the L2 genesis.
            devnet_l2_allocs(config)
        else:
            log.info('Re-using existing L2 allocs.')

        run_command([
            'go', 'run', 'cmd/main.go', 'genesis', 'l2',
            '--l1-rpc', 'http://localhost:8545',
            '--deploy-config', config.devnet_config_path,
            '--l2-allocs', l2_allocs_path,
            '--l1-deployments', config.addresses_json_path,
            '--outfile.l2', config.genesis_l2_path,
            '--outfile.rollup', config.rollup_config_path
        ], cwd=config.op_node_dir)

    rollup_config = read_json(config.rollup_config_path)
    addresses = read_json(config.addresses_json_path)

    # Start the L2.
    log.info('Bringing up L2.')
    run_command(['docker', 'compose', 'up', '-d', 'l2'], cwd=config.ops_bedrock_dir, env={
        'PWD': config.ops_bedrock_dir
    })

    # Wait for the L2 to be available.
    wait_up(9545)
    wait_for_rpc_server('127.0.0.1:9545')

    # Print out the addresses being used for easier debugging.
    l2_output_oracle = addresses['L2OutputOracleProxy']
    dispute_game_factory = addresses['DisputeGameFactoryProxy']
    batch_inbox_address = rollup_config['batch_inbox_address']
    log.info(f'Using L2OutputOracle {l2_output_oracle}')
    log.info(f'Using DisputeGameFactory {dispute_game_factory}')
    log.info(f'Using batch inbox {batch_inbox_address}')

    # Set up the base docker environment.
    docker_env = {
        'PWD': config.ops_bedrock_dir,
        'SEQUENCER_BATCH_INBOX_ADDRESS': batch_inbox_address
    }

    # Selectively set the L2OO_ADDRESS or DGF_ADDRESS if using FPAC.
    # Must be done selectively because op-proposer throws if both are set.
    if DEVNET_FPAC:
        docker_env['DGF_ADDRESS'] = dispute_game_factory
        docker_env['DG_TYPE'] = '0'
        docker_env['PROPOSAL_INTERVAL'] = '10s'
    else:
        docker_env['L2OO_ADDRESS'] = l2_output_oracle

    if DEVNET_PLASMA:
        docker_env['PLASMA_ENABLED'] = 'true'
        docker_env['PLASMA_DA_SERVICE'] = 'false'
    else:
        docker_env['PLASMA_ENABLED'] = 'false'
        docker_env['PLASMA_DA_SERVICE'] = 'false'

    # Bring up the rest of the services.
    log.info('Bringing up `op-node`, `op-proposer` and `op-batcher`.')
    run_command(['docker', 'compose', 'up', '-d', 'op-node', 'op-proposer', 'op-batcher', 'artifact-server'], cwd=config.ops_bedrock_dir, env=docker_env)

    # Optionally bring up op-challenger.
    if DEVNET_FPAC:
        log.info('Bringing up `op-challenger`.')
        run_command(['docker', 'compose', 'up', '-d', 'op-challenger'], cwd=config.ops_bedrock_dir, env=docker_env)

    # Optionally bring up OP Plasma.
    if DEVNET_PLASMA:
        log.info('Bringing up `da-server`, `sentinel`.') # TODO(10141): We don't have public sentinel images yet
        run_command(['docker', 'compose', 'up', '-d', 'da-server'], cwd=config.ops_bedrock_dir, env=docker_env)


# Bring up the devnet where the contracts are deployed to L1
def devnet_deploy(config):
    devnet_deploy_l1(config)
    devnet_deploy_l2(config)

    # Fin.
    log.info('Devnet ready.')

def wait_for_rpc_server(url):
    log.info(f'Waiting for RPC server at {url}')

    headers = {'Content-type': 'application/json'}
    body = '{"id":1, "jsonrpc":"2.0", "method": "eth_chainId", "params":[]}'

    while True:
        try:
            conn = http.client.HTTPConnection(url)
            conn.request('POST', '/', body, headers)
            response = conn.getresponse()
            if response.status < 300:
                log.info(f'RPC server at {url} ready')
                return
        except Exception as e:
            log.info(f'Waiting for RPC server at {url}')
            time.sleep(1)
        finally:
            if conn:
                conn.close()


CommandPreset = namedtuple('Command', ['name', 'args', 'cwd', 'timeout'])


def devnet_test(config):
    # Run the two commands with different signers, so the ethereum nonce management does not conflict
    # And do not use devnet system addresses, to avoid breaking fee-estimation or nonce values.
    run_commands([
        CommandPreset('erc20-test',
          ['npx', 'hardhat',  'deposit-erc20', '--network',  'devnetL1',
           '--l1-contracts-json-path', config.addresses_json_path, '--signer-index', '14'],
          cwd=config.sdk_dir, timeout=8*60),
        CommandPreset('eth-test',
          ['npx', 'hardhat',  'deposit-eth', '--network',  'devnetL1',
           '--l1-contracts-json-path', config.addresses_json_path, '--signer-index', '15'],
          cwd=config.sdk_dir, timeout=8*60)
    ], max_workers=1)


def run_commands(commands: list[CommandPreset], max_workers=2):
    with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = [executor.submit(run_command_preset, cmd) for cmd in commands]

        for future in concurrent.futures.as_completed(futures):
            result = future.result()
            if result:
                print(result.stdout)


def run_command_preset(command: CommandPreset):
    with subprocess.Popen(command.args, cwd=command.cwd,
                          stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True) as proc:
        try:
            # Live output processing
            for line in proc.stdout:
                # Annotate and print the line with timestamp and command name
                timestamp = datetime.datetime.utcnow().strftime('%H:%M:%S.%f')
                # Annotate and print the line with the timestamp
                print(f"[{timestamp}][{command.name}] {line}", end='')

            stdout, stderr = proc.communicate(timeout=command.timeout)

            if proc.returncode != 0:
                raise RuntimeError(f"Command '{' '.join(command.args)}' failed with return code {proc.returncode}: {stderr}")

        except subprocess.TimeoutExpired:
            raise RuntimeError(f"Command '{' '.join(command.args)}' timed out!")

        except Exception as e:
            raise RuntimeError(f"Error executing '{' '.join(command.args)}': {e}")

        finally:
            # Ensure process is terminated
            proc.kill()
    return proc.returncode


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
