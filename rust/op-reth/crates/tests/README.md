# E2E tests for op-reth

This folder contains the end-to-end testing resources for op-reth. Tests run against in-process `op-devstack` systems (sysgo).

This README documents common workflows and justfile recipes used to build artifacts and run e2e tests.

## Prerequisites

- Go (to run Go-based e2e tests)
- Rust toolchain (to build `op-reth`)
- Foundry (`forge`) for proof contract artifacts
- Docker (optional, only for `just build-docker`)

## Commands

List all available recipes:

```sh
just --list
```

Build the local `op-reth` binary:

```sh
just build
```

Run the e2e test suite in sysgo mode (Go tests):

```sh
# runs go test with a long timeout; defaults to GO_PKG_NAME=proofs/core
just test-e2e-sysgo

# run a specific test or package
GO_PKG_NAME=path/to/pkg just test-e2e-sysgo
```

Optional: build a local Docker image (`op-reth:local`):

```sh
just build-docker
```

## Implementation notes

- `just test-e2e-sysgo` depends on `build`, `unzip-contract-artifacts`, and `build-contracts`.
- The test target sets `OP_RETH_EXEC_PATH` to `../../../target/debug/op-reth`.
- You can override proof EL kinds with:
  - `OP_DEVSTACK_PROOF_SEQUENCER_EL`
  - `OP_DEVSTACK_PROOF_VALIDATOR_EL`

## Quick workflow example

```sh
# build op-reth
just build

# run tests (set GO_PKG_NAME if needed)
GO_PKG_NAME=proofs just test-e2e-sysgo
```
