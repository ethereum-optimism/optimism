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

- **Default branch**: `develop` (not `main`)
- **Build system**: migrating from Make to [Just](https://github.com/casey/just) — shared justfile infra lives in `justfiles/`

This repository contains multiple components spanning different technologies:

### Go Services

The rollup node software and associated services, including:

- **op-node**: Rollup consensus-layer client
- **op-batcher**: L2 batch submitter
- **op-proposer**: L2 output submitter
- **op-challenger**: Dispute game challenge agent
- **op-conductor**: High-availability sequencer service
- **op-supernode**: Multi-chain consensus-layer host that runs multiple OP Stack chains in a single process and performs in-process cross-chain safety verification

### Smart Contracts (`packages/contracts-bedrock`)

Solidity smart contracts for the OP Stack, including the core protocol contracts deployed on L1 and L2.

### Rust Components

The OP Stack includes significant Rust implementations:

- **kona**: Rust implementation of the OP Stack rollup state transition, including fault proof program and rollup node
- **op-reth**: OP Stack execution client built on reth
- **op-alloy**: Rust crates providing OP Stack types and providers for the alloy ecosystem
- **alloy-op-hardforks** / **alloy-op-evm**: OP Stack hardfork and EVM support for alloy

### Fault Proof System

- **cannon**: Onchain MIPS instruction emulator (in Go)
- **rust/kona**: Fault proof program — client and host (in Rust)

### Development and Testing Infrastructure

- **op-e2e**: End-to-end testing framework
- **op-acceptance-tests**: Acceptance test suite

## Subdirectory Instructions

Some subdirectories have their own CLAUDE.md with domain-specific conventions. Read the relevant file before working in that area — do not read them all upfront.

- `rust/kona/CLAUDE.md` — Kona Rust workspace: build commands (`just b/t/l/f`), code style, architecture overview

## Additional Documentation

More detailed guidance for AI agents can be found in:

- [docs/ai/ci-ops.md](docs/ai/ci-ops.md) - CI/CD operations
- [docs/ai/ci-config-review.md](docs/ai/ci-config-review.md) - Reviewing changes to CI config (`.circleci/`, `.github/workflows/`): gate coverage, required checks, path filtering, caching, plus general CircleCI/GHA best practices
- [docs/ai/docker.md](docs/ai/docker.md) - Docker image builds: making every external fetch (apt/apk/curl/wget) retry so registry/CDN blips don't flake CI
- [docs/ai/contract-dev.md](docs/ai/contract-dev.md) - Smart contract development
- [docs/ai/dispute-game-investigation.md](docs/ai/dispute-game-investigation.md) - Investigating fault dispute games: challenger disagreements, excessive moves, self-contradiction, proposal validity, diagnosing the responsible op-node, and the bond outcome (read-only)
- [docs/ai/flake-prevention.md](docs/ai/flake-prevention.md) - Guidance for preventing flaky tests
- [docs/ai/dev-workflow.md](docs/ai/dev-workflow.md) - General development workflow: pinned tools via mise, Just usage, pre-PR checks, and CI caveats
- [docs/ai/go-dev.md](docs/ai/go-dev.md) - Go service development
- [docs/ai/rust-dev.md](docs/ai/rust-dev.md) - Rust development (kona, op-reth, alloy crates)
- [docs/ai/reth-update-review.md](docs/ai/reth-update-review.md) - Reviewing reth/revm/alloy dependency bumps: the risk guide for upstream changes that should force a change in our in-tree op- forks but produce no diff (silent overrides, new defaults/variants, sync drift, consensus-critical math). Pairs with the `reth-update-reviewer` agent
- [docs/ai/derivation.md](docs/ai/derivation.md) - Derivation pipeline development (op-node, kona-node)
- [docs/ai/execution-layer.md](docs/ai/execution-layer.md) - Execution layer development (op-reth / EVM, fees, deposits)
- [docs/ai/fault-proofs.md](docs/ai/fault-proofs.md) - Fault proof system (Cannon, kona-client, dispute games)
- [docs/ai/devfeatures.md](docs/ai/devfeatures.md) - The `DevFeatures` bitmap system gating in-development smart contract features: where the bitmap is supplied, composed, propagated, and read. Only relevant for contract development and op-deployer — not needed for client (op-node / op-reth / kona) work
- [docs/ai/acceptance-tests.md](docs/ai/acceptance-tests.md) - Building and running acceptance tests locally
- [docs/ai/writing-acceptance-tests.md](docs/ai/writing-acceptance-tests.md) - Writing new acceptance tests: DSL patterns, naming, what to avoid
- [docs/ai/opgeth-decoupling.md](docs/ai/opgeth-decoupling.md) - op-geth decoupling plan: consult when doing any op-geth decoupling work — migrating OP Stack–specific code out of op-geth into `op-core/*`, retiring op-geth-backed execution in tests/tooling, or removing fork-only API uses — so the monorepo can depend on upstream go-ethereum (scope: the whole monorepo — single go.mod; tracking issue #20257)

## External References

- [Optimism Documentation](https://docs.optimism.io)
- [OP Stack Specifications](https://github.com/ethereum-optimism/specs)
- [Contributing Guide](CONTRIBUTING.md)
