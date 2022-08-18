import argparse
import hashlib
import logging
import os
import shutil
import sys
from logging.config import dictConfig
import urllib.request
from secrets import token_bytes

log_level = os.getenv('LOG_LEVEL')

log_config = {
    'version': 1,
    'loggers': {
        '': {
            'handlers': ['console'],
            'level': log_level if log_level is not None else 'INFO'
        },
    },
    'handlers': {
        'console': {
            'formatter': 'stderr',
            'class': 'logging.StreamHandler',
            'stream': 'ext://sys.stdout'
        }
    },
    'formatters': {
        'stderr': {
            'format': '[%(levelname)s|%(asctime)s] %(message)s',
            'datefmt': '%m-%d-%Y %I:%M:%S'
        }
    },
}

dictConfig(log_config)

log = logging.getLogger()

parser = argparse.ArgumentParser(description='Configure an Optimism Bedrock replica using docker-compose.')
parser.add_argument('--network', type=str, help='name for the network to create a replica for', required=True)
parser.add_argument('--l1-rpc', type=str, help='l1 RPC provider', required=True)
parser.add_argument('--outdir', type=str, help='output directory for the replica config', required=True)
parser.add_argument('--geth-tag', type=str, help='docker tag to use with geth', default='optimism-history')
parser.add_argument('--geth-http-port', type=int, help='geth http port', default=8545)
parser.add_argument('--geth-ws-port', type=int, help='geth ws port', default=8546)
parser.add_argument('--op-node-tag', type=str, help='docker tag to use with the rollup node', default='develop')
parser.add_argument('--op-node-http-port', type=int, help='rollup node http port', default=9545)
parser.add_argument('--op-node-metrics-port', type=int, help='rollup node http port', default=7300)
parser.add_argument('--op-node-pprof-port', type=int, help='rollup node http port', default=6300)
parser.add_argument('--bucket', type=str, help='GCP bucket to pull network data from',
                    default='https://storage.googleapis.com/bedrock-goerli-regenesis-data')


def main():
    args = parser.parse_args()
    network = args.network
    l1_rpc = args.l1_rpc
    outdir = args.outdir
    bucket = args.bucket

    if not os.path.isdir(outdir):
        log.info(f'Output directory {outdir} does not exist, creating it.')
        os.makedirs(outdir, exist_ok=False)

    if len(os.listdir(outdir)) > 0 and not confirm(
            f'Output directory {outdir} is not empty. Files may be overwritten. Proceed?'):
        log.error('Aborted.')
        sys.exit(1)

    log.info(f'Using network {network}.')
    log.info(f'Downloading genesis.')
    get_network_file(outdir, bucket, network, 'genesis.json')
    log.info(f'Downloading contracts.')
    get_network_file(outdir, bucket, network, 'contracts.json')
    log.info(f'Downloading rollup config.')
    get_network_file(outdir, bucket, network, 'rollup.json')

    log.info('Writing JWT secret.')
    m = hashlib.sha3_256()
    m.update(token_bytes(32))
    dump_file(outdir, 'jwt-secret.txt', m.hexdigest())

    log.info('Writing P2P secret.')
    m = hashlib.sha3_256()
    m.update(token_bytes(32))
    dump_file(outdir, 'p2p-node-key.txt', m.hexdigest())

    log.info('Writing opnode environment.')
    dump_file(outdir, 'op-node.env', op_node_env_tmpl(l1_rpc, f'ws://l2:{args.geth_ws_port}', args.op_node_http_port))

    log.info('Writing entrypoint.')
    dump_file(outdir, 'entrypoint.sh', ENTRYPOINT)

    log.info('Writing compose config.')
    dump_file(outdir, 'docker-compose.yml', docker_compose_tmpl(
        network,
        args.geth_tag,
        args.geth_http_port,
        args.geth_ws_port,
        args.op_node_tag,
        args.op_node_http_port,
        args.op_node_pprof_port,
        args.op_node_metrics_port
    ))


def get_network_file(outdir, bucket, network, filename):
    outfile, _ = urllib.request.urlretrieve(
        f'{bucket}/{network}/{filename}'
    )
    shutil.move(outfile, os.path.join(outdir, filename))
    return outfile


def confirm(msg):
    while True:
        res = input(f'{msg} y/n ')
        if res in 'y':
            return True
        elif res in 'n':
            return False
        else:
            print('\nInvalid option, please try again.')


def dump_file(outdir, filename, content):
    with open(os.path.join(outdir, filename), 'w+') as f:
        f.write(content)


def op_node_env_tmpl(l1_rpc, l2_rpc, op_node_http_port):
    return f"""
OP_NODE_L1_ETH_RPC={l1_rpc}
OP_NODE_L2_ETH_RPC={l2_rpc}
OP_NODE_ROLLUP_CONFIG=/config/rollup.json
OP_NODE_L2_ENGINE_RPC={l2_rpc}
OP_NODE_RPC_ADDR=0.0.0.0
OP_NODE_RPC_PORT={op_node_http_port}
OP_NODE_P2P_LISTEN_IP=0.0.0.0
OP_NODE_P2P_LISTEN_TCP_PORT=9003
OP_NODE_P2P_LISTEN_UDP_PORT=9003
OP_NODE_P2P_PRIV_PATH=/config/p2p-node-key.txt
OP_NODE_P2P_PEERSTORE_PATH=/p2p/peerstore
OP_NODE_P2P_DISCOVERY_PATH=/p2p/discovery
OP_NODE_L2_ENGINE_AUTH=/config/jwt-secret.txt
OP_NODE_VERIFIER_L1_CONFS=3
OP_NODE_LOG_FORMAT=json

# OP_NODE_P2P_ADVERTISE_IP=
# OP_NODE_P2P_ADVERTISE_TCP=9003
# OP_NODE_P2P_ADVERTISE_TCP=9003

OP_NODE_METRICS_ENABLED=true
OP_NODE_METRICS_ADDR=127.0.0.1
OP_NODE_METRICS_PORT=7300

OP_NODE_PPROF_ENABLED=true
OP_NODE_PPROF_PORT=6666
OP_NODE_PPROF_ADDR=127.0.0.1
    """


def docker_compose_tmpl(network, geth_tag, geth_http_port, geth_ws_port, op_node_tag, op_node_http_port,
                        op_node_pprof_port,
                        op_node_metrics_port):
    return f"""
version: '3.4'

volumes:
  {network}_l2_data:
  {network}_op_log:

services:
  l2:
    image: ethereumoptimism/reference-optimistic-geth:{geth_tag}
    ports:
      - "{geth_http_port}:8545"
      - "{geth_ws_port}:8546"
    volumes:
      - "{network}_l2_data:/db"
      - ./genesis.json:/genesis.json
      - ./jwt-secret.txt:/jwt-secret.txt
      - ./entrypoint.sh:/entrypoint.sh
    entrypoint:
      - "/bin/sh"
      - "/entrypoint.sh"
      - "--authrpc.jwtsecret=/jwt-secret.txt"

  op-node:
    depends_on:
      - l2
    image: us-central1-docker.pkg.dev/bedrock-goerli-development/images/op-node:{op_node_tag}
    command: op-node
    ports:
      - "{op_node_http_port}:8545"
      - "{op_node_pprof_port}:6666"
      - "{op_node_metrics_port}:7300"
    env_file:
      - ./op-node.env
    volumes:
      - ./jwt-secret.txt:/config/jwt-secret.txt
      - ./rollup.json:/config/rollup.json
      - ./p2p-node-key.txt:/config/p2p-node-key.txt
      - {network}_op_log:/op_log
    """


ENTRYPOINT = """
#!/bin/sh
set -exu

apk add jq

VERBOSITY=${GETH_VERBOSITY:-3}
GETH_DATA_DIR=/db
GETH_CHAINDATA_DIR="$GETH_DATA_DIR/geth/chaindata"
GETH_KEYSTORE_DIR="$GETH_DATA_DIR/keystore"
GENESIS_FILE_PATH="${GENESIS_FILE_PATH:-/genesis.json}"
CHAIN_ID=$(cat "$GENESIS_FILE_PATH" | jq -r .config.chainId)
BLOCK_SIGNER_PRIVATE_KEY="3e4bde571b86929bf08e2aaad9a6a1882664cd5e65b96fff7d03e1c4e6dfa15c"
BLOCK_SIGNER_ADDRESS="0xca062b0fd91172d89bcd4bb084ac4e21972cc467"
RPC_PORT="${RPC_PORT:-8545}"
WS_PORT="${WS_PORT:-8546}"

if [ ! -d "$GETH_KEYSTORE_DIR" ]; then
    echo "$GETH_KEYSTORE_DIR missing, running account import"
    echo -n "pwd" > "$GETH_DATA_DIR"/password
    echo -n "$BLOCK_SIGNER_PRIVATE_KEY" | sed 's/0x//' > "$GETH_DATA_DIR"/block-signer-key
    geth account import \\
        --datadir="$GETH_DATA_DIR" \\
        --password="$GETH_DATA_DIR"/password \\
        "$GETH_DATA_DIR"/block-signer-key
else
    echo "$GETH_KEYSTORE_DIR exists."
fi

if [ ! -d "$GETH_CHAINDATA_DIR" ]; then
    echo "$GETH_CHAINDATA_DIR missing, running init"
    echo "Initializing genesis."
    geth --verbosity="$VERBOSITY" init \\
        --datadir="$GETH_DATA_DIR" \\
        "$GENESIS_FILE_PATH"
else
    echo "$GETH_CHAINDATA_DIR exists."
fi

# Warning: Archive mode is required, otherwise old trie nodes will be
# pruned within minutes of starting the devnet.

exec geth \\
    --datadir="$GETH_DATA_DIR" \\
    --verbosity="$VERBOSITY" \\
    --http \\
    --http.corsdomain="*" \\
    --http.vhosts="*" \\
    --http.addr=0.0.0.0 \\
    --http.port="$RPC_PORT" \\
    --http.api=web3,debug,eth,txpool,net,engine \\
    --ws \\
    --ws.addr=0.0.0.0 \\
    --ws.port="$WS_PORT" \\
    --ws.origins="*" \\
    --ws.api=debug,eth,txpool,net,engine \\
    --syncmode=full \\
    --nodiscover \\
    --maxpeers=1 \\
    --networkid=$CHAIN_ID \\
    --unlock=$BLOCK_SIGNER_ADDRESS \\
    --mine \\
    --miner.etherbase=$BLOCK_SIGNER_ADDRESS \\
    --password="$GETH_DATA_DIR"/password \\
    --allow-insecure-unlock \\
    --gcmode=archive \\
    "$@"
"""

if __name__ == '__main__':
    main()
