# `kona-supervisor`

A supervisor implementation for the OP stack built in rust.

## Installation

Build from source 

```
cargo build --profile release-perf --bin kona-supervisor
```

### Usage

Run the `kona-supervisor` using the following command

```bash
kona-supervisor \
  --metrics.enabled \
  --metrics.port 9090 \
  --metrics.addr 127.0.0.1 \
  --l1-rpc http://localhost:8545 \
  --l2-consensus.nodes http://node1:8551,http://node2:8551 \
  --l2-consensus.jwt-secret secret1,secret2 \
  --datadir /supervisor_data \
  --dependency-set /path/to/deps.json \
  --rollup-config-paths /configs/rollup-*.json
```

### Configuration via Environment Variables

Many configuration options can be set via environment variables:

- `L1_RPC` - L1 RPC source
- `L2_CONSENSUS_NODES` - L2 consensus rollup node RPC addresses.
- `L2_CONSENSUS_JWT_SECRET` - JWT secrets for L2 consensus nodes.
- `DEPENDENCY_SET` - Path to the dependency-set JSON config file.
- `DATADIR` - Directory to store supervisor data.
- `ROLLUP_CONFIG_PATHS` - Path pattern to op-node rollup.json configs to load as a rollup config set.

### Help and Documentation

Use the `--help` flag to see all available options:

```
kona-supervisor --help
```

## Advanced Configuration

Coming soon
