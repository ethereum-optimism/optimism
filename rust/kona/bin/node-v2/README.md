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
kona-node [GLOBAL_OPTIONS] <SUBCOMMAND> [SUBCOMMAND_OPTIONS]
```

### Available Subcommands

- **`node`** (alias: `n`) - Runs the consensus node for OP Stack rollup validation
- **`net`** (aliases: `p2p`, `network`) - Runs the networking stack for the node
- **`registry`** (aliases: `r`, `scr`) - Lists OP Stack chains available in the superchain-registry
- **`bootstore`** (aliases: `b`, `boot`, `store`) - Utility tool to interact with local bootstores
- **`info`** - Get information about OP Stack chains

### Running the Consensus Node

The primary use case is running the consensus node with the `node` subcommand:

```bash
kona-node \
  --chain 11155420 \
  --metrics.enabled \
  --metrics.port 9002 \
  node \
  --l1 <L1_PROVIDER_RPC> \
  --l1-beacon <L1_BEACON_API> \
  --l2 <L2_ENGINE_RPC> \
  --l2-engine-jwt-secret /path/to/jwt.hex \
  --port 5060 \
  --p2p.listen.tcp 9223 \
  --p2p.listen.udp 9223 \
  --p2p.scoring off \
  --p2p.bootstore /path/to/bootstore
```

### Example: OP Sepolia Configuration

Here's a complete example for running a kona-node connected to OP Sepolia:

```bash
# Set required environment variables
export L1_PROVIDER_RPC="https://your-l1-rpc-endpoint"
export L1_BEACON_API="https://your-l1-beacon-api-endpoint"

# Run the node
kona-node \
  --chain 11155420 \
  --metrics.enabled \
  --metrics.port 9002 \
  node \
  --l1 $L1_PROVIDER_RPC \
  --l1-beacon $L1_BEACON_API \
  --l2 http://localhost:8551 \
  --l2-engine-jwt-secret ./jwt.hex \
  --port 5060 \
  --p2p.listen.tcp 9223 \
  --p2p.listen.udp 9223 \
  --p2p.scoring off \
  --p2p.bootstore ./bootstore
```

### Configuration via Environment Variables

Many configuration options can be set via environment variables:

- `KONA_NODE_L1_ETH_RPC` - L1 execution client RPC URL
- `KONA_NODE_L1_TRUST_RPC` - Whether to trust the L1 RPC without verification (default: true)
- `KONA_NODE_L1_BEACON` - L1 beacon API URL
- `KONA_NODE_L2_ENGINE_RPC` - L2 engine API URL
- `KONA_NODE_L2_TRUST_RPC` - Whether to trust the L2 RPC without verification (default: true)
- `KONA_NODE_L2_ENGINE_AUTH` - Path to L2 engine JWT secret file
- `KONA_NODE_MODE` - Node operation mode (default: validator)
- `RUST_LOG` - Logging configuration

Example using environment variables:

```bash
export KONA_NODE_L1_ETH_RPC="https://your-l1-rpc"
export KONA_NODE_L1_BEACON="https://your-l1-beacon-api"
export KONA_NODE_L2_ENGINE_RPC="http://localhost:8551"
export RUST_LOG="kona_node=info,kona_derive=debug"

kona-node node --port 5060
```

### Help and Documentation

Use the `--help` flag to see all available options and subcommands:

```bash
# General help
kona-node --help

# Help for specific subcommands
kona-node node --help
kona-node net --help
kona-node registry --help
```

### Networking and P2P

Run just the networking stack:

```bash
kona-node net \
  --p2p.listen.tcp 9223 \
  --p2p.listen.udp 9223 \
  --p2p.bootstore ./bootstore
```

### Registry Information

List available OP Stack chains:

```bash
kona-node registry
```

Get information about a specific chain:

```bash
kona-node info --help
```

## Requirements

- **L1 Execution Client**: Access to an Ethereum L1 execution client RPC endpoint
- **L1 Beacon API**: Access to an Ethereum L1 beacon chain API endpoint
- **L2 Execution Client**: Access to an OP Stack L2 execution client (e.g., op-reth)
- **JWT Secret**: A JWT secret file for authenticated communication with the L2 execution client

## Advanced Configuration

### RPC Trust Configuration

By default, Kona trusts RPC providers and does not perform additional block hash verification, optimizing for performance. This can be configured using trust flags:

```bash
# For untrusted/public RPC providers (adds verification)
kona-node node \
  --l1 https://public-rpc-endpoint.com \
  --l1-trust-rpc false \
  --l2 https://another-public-rpc.com \
  --l2-trust-rpc false \
  # ... other options
```

**Security Considerations:**
- Default behavior (`true`): No additional verification, assumes RPC is trustworthy
- Verification mode (`false`): All block hashes are verified against requested hashes
- Use verification (`false`) for public or third-party RPC endpoints
- Default trust (`true`) is suitable for local nodes and trusted infrastructure

### Production Deployments

For production deployments and advanced configurations, refer to the docker recipe in the main repository at `docker/recipes/kona-node/` which provides a complete setup example with monitoring and multiple services.
