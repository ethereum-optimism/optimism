# tools

A collection of Bedrock ops tools.

## generate_replica.py

This script generates a replica configuration suitable for use with a deployed Bedrock network. Given a network name and an output directory, it will:

1. Pull the network's genesis, rollup config, and contract addresses from GCP.
2. Generate P2P/JWT keys.
3. Generate a `docker-compose.yml` file that can be used to immediately start the replica.

The above files are outputted to a user-defined output directory in case further customization is desired.

The network must already have been deployed using `bedrock-regen`.

**Prerequisites**: Python 3.7 or above. No `pip` or `venv` is necessary, since the script does not have any dependencies outside of the standard library.

**Usage:**

Run `python3 generate.py <options>` to invoke the script. `python3 generate.py -h` will output the usage help text below. All configuration options except for the following are optional: `--network`, `--l1-rpc`, and `--outdir`.

**Example**:

```
python3 generate_replica.py --network <network-name> --op-node-tag 068113f255fa23edcd628ed853c6e5e616af7b77 --outdir ./replica-regenesis-447cda2 --l1-rpc <removed>
```

**CLI Helptext**:

```
generate.py [-h] --network NETWORK --l1-rpc L1_RPC --outdir OUTDIR [--geth-tag GETH_TAG] [--geth-http-port GETH_HTTP_PORT] [--geth-ws-port GETH_WS_PORT] [--op-node-tag OP_NODE_TAG]
                   [--op-node-http-port OP_NODE_HTTP_PORT] [--op-node-metrics-port OP_NODE_METRICS_PORT] [--op-node-pprof-port OP_NODE_PPROF_PORT] [--bucket BUCKET]

Configure an Optimism Bedrock replica using docker-compose.

optional arguments:
  -h, --help            show this help message and exit
  --network NETWORK     name for the network to create a replica for
  --l1-rpc L1_RPC       l1 RPC provider
  --outdir OUTDIR       output directory for the replica config
  --geth-tag GETH_TAG   docker tag to use with geth
  --geth-http-port GETH_HTTP_PORT
                        geth http port
  --geth-ws-port GETH_WS_PORT
                        geth ws port
  --op-node-tag OP_NODE_TAG
                        docker tag to use with the rollup node
  --op-node-http-port OP_NODE_HTTP_PORT
                        rollup node http port
  --op-node-metrics-port OP_NODE_METRICS_PORT
                        rollup node http port
  --op-node-pprof-port OP_NODE_PPROF_PORT
                        rollup node http port
  --bucket BUCKET       GCP bucket to pull network data from

```