# `kona-node` recipe

> [!WARNING]
>
> `kona-node` is in active development, and this recipe is subject to frequent change (and may not work!) For the time
> being, it is intended to be used for development purposes. Please [file an issue][new-issue] if you have any problems
> during development.

This directory contains a simple `docker-compose` setup for `kona-node` and `op-reth`, including example Grafana
dashboards and a default Prometheus configuration.

By default, this recipe is configured to sync the [`OP Sepolia`][op-sepolia] L2.

## Usage

### Running

An L1 Execution Client RPC and L1 Beacon API endpoint must be configured in your environment. The `L1_PROVIDER_RPC` and
`L1_BEACON_API` environment variables can be set in [`cfg.env`](./cfg.env).

Once these two environment variables are set, the environment can be spun up and shut down as follows:

```sh
# Start `kona-node`, `op-reth`, and `grafana` + `prometheus`
just up

# Shutdown the docker compose environment
just down

# Restart the docker compose environment
just restart
```

### Grafana

The grafana instance can be accessed at `http://localhost:3000` in your browser. The username and password, by default,
are both `admin`.

#### Adding a new visualization

The `kona-node` dashboard is provisioned within the grafana instance by default. A new visualization can be added to the
dashboard by navigating to the `Kona Node` dashboard, and then clicking `Add` > `Visualization` in the top right.

Once your visualization has been added, click `Share` > `Export` (tab), and toggle "Export for sharing externally" on.
Then, copy the JSON, and replace the contents of [`overview.json`](./grafana/dashboards/overview.json)
before making a PR.

## Default Ports

| Port    | Service                     |
|---------|-----------------------------|
| `9223`  | `kona-node` discovery       |
| `9002`  | `kona-node` metrics         |
| `5060`  | `kona-node` RPC             |
| `30303` | `op-reth` discovery         |
| `9001`  | `op-reth` metrics           |
| `8545`  | `op-reth` RPC               |
| `8551`  | `op-reth` engine            |
| `9090`  | `prometheus` metrics server |
| `3000`  | `grafana` dashboard UI      |

## Configuration

### Adjusting host ports

Host ports for both `op-reth` and `kona-node` can be configured in [`cfg.env`](./cfg.env).

### Syncing a different OP Stack chain

To adjust the chain that the node is syncing, you must modify the `docker-compose.yml` file to specify the desired
network parameters. Specifically:
1. Ensure `L1_PROVIDER_RPC` and `L1_BEACON_API` are set to L1 clients that represent the settlement layer of the L2.
1. `op-reth`
    - `--chain` must specify the desired chain.
    - `--rollup.sequencer-http` must specify the sequencer endpoint.
1. `kona-node`
    - `--chain` must specify the chain ID of the desired chain.

### Adjusting log filters

Log filters can be adjusted by setting the `RUST_LOG` environment variable. This environment variable will be forwarded
to the `kona-node` container's entrypoint.

Example: `export RUST_LOG=engine_builder=trace,runtime=debug`

[op-sepolia]: https://sepolia-optimism.etherscan.io
[op-reth]: https://github.com/paradigmxyz/reth
[new-issue]: https://github.com/op-rs/kona/issues/new
