# `kona-node`

A modular, robust rollup node implementation for the OP Stack built in rust.

## Installation

You can install `kona-node` directly from crates.io using Cargo:

```bash
cargo install kona-node
```

## Usage

### Basic Command Structure

```bash
kona-node <SUBCOMMAND> [OPTIONS]
```

### Available Subcommands

- **`node`** - Runs the consensus node for OP Stack rollup validation

### Running the Consensus Node

The primary use case is running the consensus node with the `node` subcommand:

```bash
kona-node node \
  --logs.enabled \
  --chain 11155420 \
  --metrics.enabled \
  --metrics.port 9002 \
  --l1.eth-url <L1_PROVIDER_RPC> \
  --l1.beacon-url <L1_BEACON_API> \
  --l2.engine.url <L2_ENGINE_RPC> \
  --l2.engine.jwt-secret-path /path/to/jwt.hex \
  --rpc.enabled \
  --rpc.port 5060 \
  --p2p.listen.tcp-port 9223 \
  --p2p.listen.udp-port 9223 \
  --p2p.peer-scoring none \
  --p2p.bootstore /path/to/bootstore
```

### Example: OP Sepolia Configuration

Here's a complete example for running a kona-node connected to OP Sepolia:

```bash
# Set required environment variables
export L1_PROVIDER_RPC="https://your-l1-rpc-endpoint"
export L1_BEACON_API="https://your-l1-beacon-api-endpoint"

# Run the node
kona-node node \
  --logs.enabled \
  --chain 11155420 \
  --metrics.enabled \
  --metrics.port 9002 \
  --l1.eth-url $L1_PROVIDER_RPC \
  --l1.beacon-url $L1_BEACON_API \
  --l2.engine.url http://localhost:8551 \
  --l2.engine.jwt-secret-path ./jwt.hex \
  --rpc.enabled \
  --rpc.port 5060 \
  --p2p.listen.tcp-port 9223 \
  --p2p.listen.udp-port 9223 \
  --p2p.peer-scoring none \
  --p2p.bootstore ./bootstore
```

### Configuration via Environment Variables

Many configuration options can be set via environment variables:

- `KONA_NODE_CHAIN` - L2 chain to load from the superchain registry (or set `KONA_NODE_CHAIN_ROLLUP_CONFIG_PATH` to a rollup config file)
- `KONA_NODE_L1_ETH_URL` - L1 execution client RPC URL
- `KONA_NODE_L1_BEACON_URL` - L1 beacon API URL
- `KONA_NODE_L2_ENGINE_URL` - L2 engine API URL
- `KONA_NODE_L2_ENGINE_JWT_SECRET_PATH` - Path to L2 engine JWT secret file
- `RUST_LOG` - Logging configuration

Example using environment variables:

```bash
export KONA_NODE_CHAIN="optimism"
export KONA_NODE_L1_ETH_URL="https://your-l1-rpc"
export KONA_NODE_L1_BEACON_URL="https://your-l1-beacon-api"
export KONA_NODE_L2_ENGINE_URL="http://localhost:8551"
export KONA_NODE_L2_ENGINE_JWT_SECRET_PATH="/path/to/jwt.hex"
export RUST_LOG="kona_node=info,kona_derive=debug"

kona-node node --logs.enabled --rpc.enabled --rpc.port 5060
```

### Help and Documentation

Use the `--help` flag to see all available options and subcommands:

```bash
# General help
kona-node --help

# Help for the node subcommand
kona-node node --help
```

## Requirements

- **L1 Execution Client**: Access to an Ethereum L1 execution client RPC endpoint
- **L1 Beacon API**: Access to an Ethereum L1 beacon chain API endpoint
- **L2 Execution Client**: Access to an OP Stack L2 execution client (e.g., op-reth)
- **JWT Secret**: A JWT secret file for authenticated communication with the L2 execution client

## Advanced Configuration

### Production Deployments

For production deployments and advanced configurations, refer to the docker recipe in the main repository at `docker/recipes/kona-node/` which provides a complete setup example with monitoring and multiple services.
