# Oneshot Builds

This build creates a single image that runs both `op-geth` and `op-node` against a specific network. It also exposes various environment variables that are useful to configure a Bedrock replica. 

## Usage

The only thing you need to set to get your replica working is the `OP_NODE_L1_ETH_RPC` environment variable. Set this to an L1 RPC you control, and the container will take care of the rest. The full list of env vars you can set is below:

**Opnode Configuration**

Env Var|Default|Usage
---|---|---
`OP_NODE_L1_ETH_RPC`|dummy|RPC URL for an L1 Ethereum node.
`OP_NODE_RPC_PORT`|9545|RPC port for the op node to listen on.
`OP_NODE_P2P_DISABLE`|false|Whether or not P2P should be disabled.
`OP_NODE_P2P_NO_DISCOVERY`|false|Whether or not peer discovery should be disabled.
`OP_NODE_P2P_LISTEN_IP`|0.0.0.0|P2P listen IP.
`OP_NODE_P2P_LISTEN_TCP_PORT`|9222|TCP port the P2P stack should listen on.
`OP_NODE_P2P_LISTEN_UDP_PORT`|9222|UDP port the P2P stack should listen on.
`OP_NODE_P2P_ADVERTISE_TCP`|9222|Port the P2P stack should listen on. Should usually be `OP_NODE_P2P_ADVERTISE_TCP`.
`OP_NODE_P2P_ADVERTISE_UDP`|9222|Port the P2P stack should listen on. Should usually be `OP_NODE_P2P_ADVERTISE_UDP`.
`OP_NODE_METRICS_ENABLED`|true|Enables Prometheus metrics.
`OP_NODE_METRICS_ADDR`|0.0.0.0|Address the metrics server should listen on.
`OP_NODE_METRICS_PORT`|7300|Port the metrics server should listen on.
`OP_NODE_LOG_FORMAT`|json|Log format. Can be JSON or text.
`OP_NODE_PPROF_ENABLED`|false|Enables `pprof` for profiling.
`OP_NODE_PPROF_PORT`|6666|Port `pprof` should listen on.
`OP_NODE_PPROF_ADDR`|0.0.0.0|Address `pprof` should listen on.
`OP_NODE_HEARTBEAT_ENABLED`|true|Whether or not to enable heartbeating.
`OP_NODE_HEARTBEAT_MONIKER`||Optional moniker to use while heartbeating.

**op-geth Configuration**

Env Var|Default|Usage
---|---|---
`OP_GETH_VERBOSITY`|3|Number 1-5 that controls how verbosely Geth should log.
`OP_GETH_HTTP_ADDR`|0.0.0.0|Address Geth should listen on.
`OP_GETH_HTTP_CORSDOMAIN`|*|CORS domain Geth should allow.
`OP_GETH_HTTP_PORT`|8545|HTTP port Geth should listen on.
`OP_GETH_WS_ADDR`|0.0.0.0|WS address Geth should listen on.
`OP_GETH_WS_PORT`|8546|WS port Geth should listeno n.
`OP_GETH_WS_ORIGINS`|*|WS origins Geth should allow.

**Other Configuration**

Additionally, the node exposes the following volumes should you wish to mount them somewhere yourself:

- `/db` contains Geth's database.
- `/p2p` contains the opnode's peer store.


## Architecture

Oneshot uses s6-overlay under the hood to supervise the processes it runs. It includes three services:

1. `op-init`: A oneshot service that `op-init` and `op-node` use to initialize themselves.
2. `op-geth`: A longrun service that manages the Geth node.
3. `op-node`: A longrun service that manages the opnode.