# Execution Layer Development

This document provides guidance for AI agents working with the OP Stack execution layer:
EVM execution, state transitions, block processing, and the L2-specific modifications
op-geth carries on top of upstream go-ethereum.

The execution client itself (op-geth) lives in a separate repository
(`ethereum-optimism/op-geth`), but its behavior is load-bearing for the services in this
monorepo. For the ongoing effort to move OP-Stack-specific code out of op-geth and into
`op-core/*` so the monorepo can depend on upstream go-ethereum directly, see
[opgeth-decoupling.md](opgeth-decoupling.md).

## Scope

The execution layer in op-geth: EVM execution, state transitions, block processing, and
L2-specific modifications.

## Key Concepts

- **Deposit transactions**: system-level transactions originating from L1 deposits.
- **L1 fee computation**: an additional fee component based on L1 data cost.
- **Sequencer fee vault**: collection of L2 execution fees.
- **EIP implementation**: carrying upstream EIPs with L2 adaptations.

## Invariants

- **Deposit success**: deposit transactions always succeed at the execution level — they
  do not revert on gas. Deposit transaction handling must not break the standard EVM
  execution path.
- **L1 fee accuracy**: the L1 fee calculation must match the on-chain L1 oracle exactly.
- **Determinism**: the state transition function is deterministic.
- **Gas limit enforcement**: block gas limit enforcement must account for deposit
  transactions.

## Key Differences from Upstream geth

- Deposit transaction type handling in `core/types/`.
- Fee model modifications in `core/` and `params/`.
- Sequencer-specific block building in `miner/`.
- **Minimal diff principle**: keep OP-specific changes small for upstream rebasing. Every
  OP-specific change must be justified.

## Testing Requirements

- The upstream geth test suite must continue passing.
- Deposit transaction tests covering all edge cases.
- L1 fee calculation tests against known reference values.
- State transition differential tests between op-geth and upstream geth.
