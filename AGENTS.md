# Optimism Monorepo

This is the primary monorepo for the OP Stack, maintained by the Optimism Collective. The OP Stack is a decentralized software stack that powers Optimism and forms the backbone of blockchains like OP Mainnet and Base.

## Improving This Documentation

If during a session you learn something that would have helped you from the start, suggest updating these docs. Examples:

- User corrects an outdated or wrong command you tried
- User shows a better way to run tests, build, or debug
- User explains a pattern or convention not documented here
- Something you assumed from the docs turns out to be incorrect

When this happens, offer to submit the improvement to the relevant file in `docs/ai/` or to this file. If the topic doesn't fit existing docs (e.g., CI workflows, debugging techniques), suggest creating a new focused document. Keep these docs tight and well-scoped rather than sprawling. Small, incremental improvements compound over time.

## Repository Overview

This repository contains multiple components spanning different technologies:

### Go Services (`op-*` directories)

The rollup node software and associated services, including:

- **op-node**: Rollup consensus-layer client
- **op-batcher**: L2 batch submitter
- **op-proposer**: L2 output submitter
- **op-challenger**: Dispute game challenge agent
- **op-conductor**: High-availability sequencer service
- **op-supervisor**: Cross-chain message safety monitor

### Smart Contracts (`packages/contracts-bedrock`)

Solidity smart contracts for the OP Stack, including the core protocol contracts deployed on L1 and L2.

### Fault Proof System

- **cannon**: Onchain MIPS instruction emulator
- **op-program**: Fault proof program
- **kona**: Alternative fault proof implementation

### Development and Testing Infrastructure

- **devnet-sdk**: Toolkit for devnet interactions
- **kurtosis-devnet**: Kurtosis-based devnet environment
- **op-e2e**: End-to-end testing framework
- **op-acceptance-tests**: Acceptance test suite

## Additional Documentation

More detailed guidance for AI agents can be found in:

- [docs/ai/contract-dev.md](docs/ai/contract-dev.md) - Smart contract development
- [docs/ai/go-dev.md](docs/ai/go-dev.md) - Go service development
- [docs/ai/rust-dev.md](docs/ai/rust-dev.md) - Rust development (kona, op-reth, alloy crates)

## External References

- [Optimism Documentation](https://docs.optimism.io)
- [OP Stack Specifications](https://github.com/ethereum-optimism/specs)
- [Contributing Guide](CONTRIBUTING.md)
