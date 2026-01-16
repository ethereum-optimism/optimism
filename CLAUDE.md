# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is the Optimism monorepo containing core components of the OP Stack - the decentralized software stack that powers Optimism and forms the backbone of OP Mainnet and Base. The codebase is primarily Go with Solidity smart contracts and a Rust subproject (Kona).

## Build Commands

### Initial Setup
```bash
# Install mise dependency manager (required)
mise trust mise.toml
mise install

# Initialize submodules
make submodules
```

### Building
```bash
# Build everything (Go + contracts)
make build

# Build only Go components
make build-go

# Build only contracts
make build-contracts

# Build specific components
make op-node
make op-batcher
make op-proposer
make op-challenger
make cannon
make op-program
```

### Linting
```bash
# Go linting
make lint-go
make lint-go-fix

# Solidity linting (from packages/contracts-bedrock)
just lint
just lint-fix
```

### Testing

**Go tests:**
```bash
# Run all Go tests
make go-tests

# Run tests with short flag (faster)
make go-tests-short

# Run tests for a specific package
go test ./op-node/...
go test ./op-batcher/...

# Run a single test
go test ./op-node/rollup/derive -run TestDerivation
```

**Solidity tests:**
```bash
cd packages/contracts-bedrock

# Run all contract tests
just test

# Run specific test
forge test --match-test testFunctionName

# Run tests with verbosity
forge test -vvv

# Run upgrade path tests (requires ETH_RPC_URL)
just test-upgrade
```

**E2E tests:**
```bash
# Action tests (deterministic, synchronous)
cd op-e2e && go test ./actions/...

# System tests (full system integration)
cd op-e2e && go test ./system/...
```

## Architecture Overview

### Core Go Services (op-*)

- **op-node**: Rollup consensus-layer client. Builds, relays, and verifies the canonical L2 chain. Communicates with execution layer (op-geth) via Engine API.
- **op-batcher**: Submits L2 transaction batches to L1 for data availability.
- **op-proposer**: Submits L2 output roots to L1 for withdrawals and dispute games.
- **op-challenger**: Dispute game challenge agent for fault proofs.
- **op-supervisor**: Monitors chains and determines cross-chain message safety for interop.
- **op-program**: Fault proof program that runs in the MIPS VM for onchain verification.

### Smart Contracts (packages/contracts-bedrock)

Located in `packages/contracts-bedrock/src/`. Key contract categories:
- `L1/`: L1 contracts (OptimismPortal, SystemConfig, bridges)
- `L2/`: L2 predeploys (CrossDomainMessenger, L2ToL1MessagePasser)
- `cannon/`: MIPS emulator contracts for fault proofs
- `dispute/`: Dispute game contracts

### Cannon (Fault Proofs)

Cannon is an onchain MIPS instruction emulator enabling fault proofs:
- `cannon/`: MIPS emulator CLI and `mipsevm` Go implementation
- Runs `op-program` one instruction at a time for dispute resolution
- Contracts in `packages/contracts-bedrock/src/cannon/`

### Kona (Rust Components)

The `kona/` directory contains Rust implementations of OP Stack components. See `kona/CLAUDE.md` for Kona-specific guidance:
```bash
cd kona
just build-native  # Build
just test          # Test
just lint-native   # Lint
```

### Testing Infrastructure

- **op-e2e/actions/**: Deterministic state-transition tests with mock clock. Tests run synchronously without parallelism.
- **op-e2e/system/**: Full system integration tests (deprecated for new tests).
- **op-acceptance-tests/**: New acceptance test framework.
- **kurtosis-devnet/**: Kurtosis-based devnet for testing.

## Solidity Style Guide

When working with Solidity files:

### Documentation
- Use `///` for NatSpec comments, `//` for regular inline comments
- `@notice` for external-facing docs, `@dev` for internal notes
- Line length: 100 characters

### Naming Conventions
- Function parameters: prefix with underscore (`_amount`)
- Return arguments: suffix with underscore (`result_`)
- Immutables: `SCREAMING_SNAKE_CASE`, `internal`, with hand-written getter
- Custom errors: `ContractName_ErrorDescription`

### Testing
- Test function naming: `test_functionName_reason_succeeds/reverts`
- Test contracts: `TargetContract_FunctionName_Test`

### Upgradeability
- Contracts should extend `Initializable` and use `reinitializer(initVersion())`
- Call `_disableInitializers()` in constructor
- All non-library contracts must implement `ISemver` with `version()`

## Key Configuration Files

- `mise.toml`: Tool versions (Go 1.24.10, Rust 1.92.0, Foundry, etc.)
- `go.mod`: Go module definition
- `packages/contracts-bedrock/foundry.toml`: Foundry configuration
- `docker-bake.hcl`: Docker build configuration

## Development Workflow

1. PRs should target the `develop` branch for backwards-compatible changes
2. Contract changes in `packages/contracts-bedrock/src` may require feature branches
3. Use `mise install` to ensure correct tool versions
4. Run `make build` after switching branches
5. Use Conventional Commits format for commit messages
