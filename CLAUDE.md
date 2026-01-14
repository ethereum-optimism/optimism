# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is the Optimism monorepo containing core components of the OP Stack - the decentralized software stack that powers Optimism and forms the backbone of blockchains like OP Mainnet and Base.

## Build Commands

### Setup
```bash
# Install mise for dependency management
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
cd packages/contracts-bedrock && just build

# Build specific Go component
make op-node
make op-batcher
make op-proposer
make op-challenger
make cannon
make op-program
```

### Testing

**Go tests:**
```bash
# Run all Go tests
make go-tests

# Run tests for a specific package
go test ./op-node/...
go test ./op-batcher/...

# Run a single test
go test -run TestName ./path/to/package/...

# Run tests with verbose output
go test -v ./op-node/...
```

**Solidity tests:**
```bash
cd packages/contracts-bedrock

# Run all contract tests
just test

# Run tests in developer mode (faster)
just test-dev

# Run specific test
forge test --match-test testFunctionName

# Run tests for specific contract
forge test --match-contract ContractName
```

**E2E tests:**
```bash
# Action tests (deterministic, DSL-style)
cd op-e2e && make test-actions

# System tests (full integration)
cd op-e2e && make test-ws
```

### Linting
```bash
# Lint Go code
make lint-go

# Lint and fix Go code
make lint-go-fix

# Lint Solidity
cd packages/contracts-bedrock && just lint-check

# Fix Solidity formatting
cd packages/contracts-bedrock && just lint-fix
```

### Pre-PR Checks
```bash
# For contracts - runs all checks
cd packages/contracts-bedrock && just pre-pr

# Fast parallel check runner for contracts
cd packages/contracts-bedrock && just check-fast
```

## Architecture

### Go Components (op-*)
- **op-node**: Rollup consensus-layer client - derives L2 chain from L1
- **op-batcher**: Submits L2 transaction batches to L1
- **op-proposer**: Submits L2 output roots to L1
- **op-challenger**: Dispute game challenge agent for fault proofs
- **op-program**: Fault proof program executed in MIPS VM
- **op-supervisor**: Monitors chains for cross-chain message safety (interop)
- **op-conductor**: High-availability sequencer coordination
- **cannon**: Onchain MIPS instruction emulator for fault proofs
- **op-service**: Shared utilities across Go components
- **op-e2e**: End-to-end and integration tests

### Smart Contracts (packages/contracts-bedrock)
Located in `packages/contracts-bedrock/src/`. Key contracts:
- L1 contracts: Bridge, rollup, and dispute game contracts deployed on Ethereum
- L2 contracts: Predeploys and system contracts on L2

### Rust Components
- **kona/**: Fault proof program and OP Stack types in Rust
- **reth/**: Modified Reth execution client (op-reth)
- **op-rbuilder/**: Block builder
- **rollup-boost/**: Rollup boost service

Build Rust components:
```bash
just build-rust-release
```

## Code Style

### Go
- Use `golangci-lint` for linting
- Follow standard Go conventions
- Run `go mod tidy` to clean dependencies

### Solidity
See `.cursor/rules/solidity-styles.mdc` for detailed guidelines:
- NatSpec: Use `///` for documentation, `//` for inline comments
- Custom errors: `ContractName_ErrorDescription`
- Function parameters: prefix with underscore (`_param`)
- Return values: suffix with underscore (`result_`)
- Immutables: `SCREAMING_SNAKE_CASE`, `internal` with getter
- All non-library contracts must inherit `ISemver` with `version()`
- Test naming: `test_functionName_reason_succeeds/reverts`

### Rust
Each Rust subproject has its own CLAUDE.md:
- `kona/CLAUDE.md` - Kona fault proof components
- `reth/CLAUDE.md` - Reth execution client

## Key Files and Directories

- `mise.toml`: Tool versions (Go 1.24, Rust 1.92, Foundry, etc.)
- `Makefile`: Root build commands
- `justfile`: Just commands for various tasks
- `packages/contracts-bedrock/justfile`: Contract-specific commands
- `op-e2e/`: Integration and E2E tests
- `ops/`: Operational scripts and tooling

## Development Workflow

1. Create branch from `develop` (main development branch)
2. Make changes following code style guidelines
3. Run tests for affected components
4. Run linting (`make lint-go` or `just lint-check`)
5. For contracts: run `just pre-pr` or `just check-fast`
6. Submit PR against `develop`

## Versioning

- Go releases: `<component-name>/v<semver>` (e.g., `op-node/v1.1.2`)
- Contract releases: `op-contracts/v<semver>`
- Tags like `v1.1.4` indicate Go-only releases (no contracts)
