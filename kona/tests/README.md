# E2e testing for the kona-node

This repository contains the e2e testing resources for the kona-node. The e2e testing is done using the `op-devstack` from the [Optimism monorepo](https://github.com/ethereum-optimism/optimism) with the `sysgo` orchestrator.

## Prerequisites

Make sure to initialize the git submodules before running the tests:

```bash
git submodule init
git submodule update --recursive
```

Then run `go mod tidy` to fetch dependencies:

```bash
go mod tidy
```

## Description

The interactions with this repository are done through the [`justfile`](./justfile) recipes.

### Running E2E Tests

To run the e2e tests, use the following command:

```bash
just test-e2e-sysgo BINARY GO_PKG_NAME DEVNET FILTER
```

Where:
- `BINARY`: The binary to test (`node` or `supervisor`)
- `GO_PKG_NAME`: The Go package name to test (e.g., `node/common`)
- `DEVNET`: The devnet configuration (`simple-kona`, `simple-kona-geth`, `simple-kona-sequencer`, `large-kona-sequencer`)
- `FILTER`: Optional test filter

For example, to run the common node tests:

```bash
just test-e2e-sysgo node node/common simple-kona
```

### Acceptance Tests

To run acceptance tests for the Rust stack:

```bash
just acceptance-tests CL_TYPE EL_TYPE GATE
```

Where:
- `CL_TYPE`: Consensus layer type (`kona` or `op-node`)
- `EL_TYPE`: Execution layer type (`op-reth` or `op-geth`)
- `GATE`: The gate to run (default: `jovian`)

### Other Recipes

- `just build-devnet BINARY`: Builds the Docker image for the specified binary (`node` or `supervisor`).

- `just build-kona`: Builds the kona-node binary.

- `just build-reth`: Builds the op-reth binary.

- `just long-running-test FILTER OUTPUT_LOGS_DIR`: Runs long-running tests with optional filter and output directory.

- `just action-tests-single`: Runs action tests for the single-chain client program.

- `just action-tests-interop`: Runs action tests for the interop client program.

## Environment Variables

When using `op-devstack` for testing, the following environment variables are set automatically by the justfile recipes:

- `DEVSTACK_ORCHESTRATOR=sysgo`: Tells `op-devstack` to use the sysgo orchestrator.
- `DISABLE_OP_E2E_LEGACY=true`: Tells `op-devstack` not to use the `op-e2e` tests that rely on e2e config and contracts-bedrock artifacts.

## Contributing

We welcome contributions to this repository.
