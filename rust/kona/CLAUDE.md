# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands
- Build workspace: `just b` or `just build-native`
- Lint: `just l` or `just lint-native`
- Format: `just f` or `just fmt-native-fix`
- Run all tests: `just t` or `just tests`
- Run specific test: `cargo nextest run --package [package-name] --test [test-name]`
- Run single test: `cargo nextest run --package [package-name] --test [test-name] -- [test_function_name]`
- Documentation: `just test-docs`

## Code Style
- MSRV: 1.88
- Format with nightly rustfmt: `cargo +nightly fmt`
- Imports: organized by crate, reordered automatically
- Error handling: use proper error types, prefer `Result<T, E>` over panics
- Naming: follow Rust conventions (snake_case for variables/functions, CamelCase for types)
- Prefer type-safe APIs and strong typing
- Documentation: rustdoc for public APIs, clear comments for complex logic
- Tests: write unit and integration tests for all functionality
- Performance: be mindful of allocations and copying, prefer references where appropriate
- No warnings policy: all clippy warnings are treated as errors (-D warnings)

## Architecture Overview

Kona is a monorepo for OP Stack types, components, and services built in Rust. The repository is organized into several major categories:

### Binaries (`bin/`)
- **`client`**: The fault proof program that executes state transitions on a prover
- **`host`**: Native program serving as the Preimage Oracle server
- **`node`**: Rollup Node implementation with flexible chain ID support
- **`supervisor`**: Supervisor implementation for interop coordination

### Protocol (`crates/protocol/`)
- **`derive`**: `no_std` compatible derivation pipeline implementation
- **`protocol`**: Core protocol types used across OP Stack rust crates
- **`genesis`**: Genesis types for OP Stack chains
- **`interop`**: Core functionality for OP Stack Interop features
- **`registry`**: Rust bindings for superchain-registry
- **`comp`**: Compression types and utilities
- **`hardforks`**: Consensus layer hardfork types and network upgrade transactions

### Proof (`crates/proof/`)
- **`executor`**: `no_std` stateless block executor
- **`proof`**: High-level OP Stack state transition proof SDK
- **`proof-interop`**: Extension of `kona-proof` with interop support
- **`mpt`**: Merkle Patricia Trie utilities for client program
- **`preimage`**: High-level PreimageOracle ABI interfaces
- **`std-fpvm`**: Platform-specific Fault Proof VM kernel APIs
- **`driver`**: Stateful derivation pipeline driver

### Node (`crates/node/`)
- **`service`**: OP Stack rollup node service implementation
- **`engine`**: Extensible rollup node engine client
- **`rpc`**: OP Stack RPC types and extensions
- **`p2p`**: P2P networking including Gossip and Discovery
- **`sources`**: Data source types and utilities

### Supervisor (`crates/supervisor/`)
- **`core`**: Core supervisor functionality
- **`service`**: Supervisor service implementation
- **`rpc`**: Supervisor RPC types and client
- **`storage`**: Database storage layer
- **`types`**: Common types for supervisor components

### Development Workflow

1. **Testing**: The project uses `nextest` for test execution. Online tests are excluded by default and can be run separately with `just test-online`
2. **Cross-compilation**: Docker-based builds for `cannon` (MIPS) and `asterisc` (RISC-V) targets
3. **Documentation**: Both rustdoc and a separate documentation site at rollup.yoga
4. **Monorepo Integration**: Pins and integrates with the Optimism monorepo for action tests

### Key Configuration Files
- `rust-toolchain.toml`: Pins Rust version to 1.88
- `rustfmt.toml`: Custom formatting configuration with crate-level import grouping
- `clippy.toml`: MSRV configuration for clippy
- `deny.toml`: Dependency auditing and license compliance
- `release.toml`: Configuration for `cargo-release` tool

### Target Architecture Support
- Native development on standard platforms
- Cross-compilation support for fault proof VMs:
  - MIPS64 (cannon target)
  - RISC-V (asterisc target)
- `no_std` compatibility for proof components

### Dependencies and Features
- Heavy use of Alloy ecosystem for Ethereum types
- OP-specific extensions via op-alloy
- Modular feature flags for different compilation targets
- Workspace-level dependency management with version pinning
