# op-up

`op-up` starts local OP Stack devnets backed by `op-devstack` presets. It is intended for interactive local testing: choose a topology, get stable localhost RPC ports, and use the printed account and contract addresses from another terminal.

## Quick Start

```bash
cd op-up
just op-up
./bin/op-up presets
./bin/op-up up minimal
```

The default preset is `minimal`.

```bash
./bin/op-up
./bin/op-up --preset interop
./bin/op-up up multinode
```

When the devnet is ready, `op-up` prints:

- stable service RPC ports for L1/L2 EL nodes and L2 CL/rollup nodes
- a prefunded dev account and private key
- deployed L1 contract addresses for each L2, including `OptimismPortalProxy`, `SystemConfigProxy`, `L1StandardBridgeProxy`, and `DisputeGameFactoryProxy`
- a config export directory under `~/.op-up/configs/` with endpoint, contract, rollup, dependency-set, and generated JSON config files when available

Press `Ctrl+C` to stop the devnet and clean up resources.

## Presets

List available presets:

```bash
./bin/op-up presets
```

Start a preset:

```bash
./bin/op-up --preset two-l2
./bin/op-up up interop
./bin/op-up deploy conductors
```

The legacy `--interop` flag is still supported as an alias for `--preset interop`.

## Ports

Every exposed service binds a random available localhost port. `op-up` prints the actual endpoints after the devnet is ready.

## Runtime Selection

`op-up` exposes the common devstack implementation selectors as CLI flags:

```bash
./bin/op-up up --l2-el-kind op-geth minimal
./bin/op-up up --l2-el-kind op-reth-proof-v2 minimal
./bin/op-up up --l2-cl-kind kona-node minimal
```

These flags set `DEVSTACK_L2EL_KIND` and `DEVSTACK_L2CL_KIND` for the devnet startup. Binary path overrides such as `OP_RETH_EXEC_PATH`, `KONA_NODE_EXEC_PATH`, and `SYSGO_GETH_EXEC_PATH` are still read by devstack directly.

## Metrics

Enable devstack metrics and dashboards:

```bash
./bin/op-up --metrics
```

When supported components finish starting, Grafana is served at `http://localhost:3000`.
