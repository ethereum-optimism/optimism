# @eth-optimism/two-step-monitor

[![codecov](https://codecov.io/gh/ethereum-optimism/optimism/branch/develop/graph/badge.svg?token=0VTG7PG7YR&flag=two-step-monitor-tests)](https://codecov.io/gh/ethereum-optimism/optimism)

The `two-step-monitor` is a simple service for detecting discrepancies between withdrawals created on L2, and
withdrawals proven on L1.

## Installation

Clone, install, and build the Optimism monorepo:

```
git clone https://github.com/ethereum-optimism/optimism.git
yarn install
yarn build
```

## Running the service

Copy `.env.example` into a new file named `.env`, then set the environment variables listed there.
Once your environment variables have been set, run the service via:

```
yarn start
```

## Ports

- API is exposed at `$TWO_STEP_MONITOR__HOSTNAME:$TWO_STEP_MONITOR__PORT/api`
- Metrics are exposed at `$TWO_STEP_MONITOR__HOSTNAME:$TWO_STEP_MONITOR__PORT/metrics`
- `$TWO_STEP_MONITOR__HOSTNAME` defaults to `0.0.0.0`
- `$TWO_STEP_MONITOR__PORT` defaults to `7300`

## What this service does

The `two-step-monitor` detects when a withdrawal is proven on L1, and verifies that a corresponding withdrawal
has been created on L2.

We export a series of Prometheus metrics that you can use to trigger alerting when issues are detected.
Check the list of available metrics via `yarn start --help`:

```sh
> yarn start --help
yarn run v1.22.19
$ ts-node ./src/service.ts --help
Usage: service [options]

Options:
  --l1rpcprovider    Provider for interacting with L1 (env: TWO_STEP_MONITOR__L1_RPC_PROVIDER)
  --l2rpcprovider    Provider for interacting with L2 (env: TWO_STEP_MONITOR__L2_RPC_PROVIDER)
  --port             Port for the app server (env: TWO_STEP_MONITOR__PORT)
  --hostname         Hostname for the app server (env: TWO_STEP_MONITOR__HOSTNAME)
  -h, --help         display help for command

Metrics:
  l1_node_connection_failures   Number of times L1 node connection has failed (type: Gauge)
  l2_node_connection_failures   Number of times L2 node connection has failed (type: Gauge)
  metadata                      Service metadata (type: Gauge)
  unhandled_errors              Unhandled errors (type: Counter)

Done in 2.19s.
```
