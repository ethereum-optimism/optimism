# E2E tests for op-reth

This folder contains the end-to-end testing resources for op-reth. Tests run against in-process `op-devstack` systems (sysgo).

This README documents common workflows and Makefile commands used to build artifacts and run e2e tests.

## Prerequisites

- Go (to run Go-based e2e tests)
- Rust toolchain (to build `op-reth`)
- Foundry (`forge`) for proof contract artifacts
- Docker (optional, only for `make build-docker`)

## Commands (Makefile targets)

Build the local `op-reth` binary:

```sh
make build
```

Run the e2e test suite in sysgo mode (Go tests):

```sh
# runs go test with a long timeout; defaults to GO_PKG_NAME=proofs/core
make test-e2e-sysgo

# run a specific test or package
make test-e2e-sysgo GO_PKG_NAME=path/to/pkg
```

Optional: build a local Docker image (`op-reth:local`):

```sh
make build-docker
```

## Implementation notes

- `make test-e2e-sysgo` now builds `op-reth` before running tests.
- The test target sets `OP_RETH_EXEC_PATH` to `../../../target/debug/op-reth`.
- You can override proof EL kinds with:
  - `OP_DEVSTACK_PROOF_SEQUENCER_EL`
  - `OP_DEVSTACK_PROOF_VALIDATOR_EL`

## Quick workflow example

```sh
# build op-reth
make build

# run tests (set GO_PKG_NAME if needed)
make test-e2e-sysgo GO_PKG_NAME=proofs
```
