# E2E tests for op-reth

This folder contains the end-to-end testing resources for op-reth. Tests use the Optimism "devstack" (from the Optimism monorepo) and Kurtosis to deploy ephemeral devnets.

This README documents common workflows and Makefile commands used to build the local Docker image, start the devnet with Kurtosis, run e2e tests, and clean up resources.

## Prerequisites

- Docker (Desktop) running on your machine
- Kurtosis CLI installed and able to reach the Kurtosis engine
- Go (to run Go-based e2e tests)

## Commands (Makefile targets)

Build the Docker image used by the devnet (tags `op-reth:local`):

```sh
make build
```

Start the Optimism devnet (default: `simple-historical-proof`):

```sh
# uses the Makefile's DEVNET variable (devnets/<DEVNET>.yaml)
make run

# or with a custom devnet YAML path
make run DEVNET_CUSTOM_PATH=/absolute/path/to/devnet.yaml
```

Run the e2e test suite that exercises the deployed devnet (Go tests):

```sh
# runs go test with a long timeout; set GO_PKG_NAME to the package to test
make test-e2e-kurtosis

# run a specific test or package
make test-e2e-kurtosis GO_PKG_NAME=path/to/pkg
```

Stop and remove Kurtosis resources (cleanup):

```sh
make clean
```

## Implementation notes

- The Makefile in this directory calls the repository root `DockerfileOp` to build an op-reth image tagged `op-reth:local`.
- The default Kurtosis package used is `github.com/ethpandaops/optimism-package@1.4.0`. The Makefile passes the YAML under `devnets/$(DEVNET).yaml` to `kurtosis run`.

## Quick workflow example

```sh
# build image
make build

# start devnet
make run

# run tests (set GO_PKG_NAME if needed)
make test-e2e-kurtosis GO_PKG_NAME=proofs

# cleanup
make clean
```
