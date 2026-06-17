# Execution Layer Development

This document provides guidance for AI agents working with the OP Stack execution layer:
EVM execution, state transitions, block processing, and the L2-specific modifications the OP
Stack carries on top of upstream Ethereum execution.

## Execution-Layer Clients

- **op-reth** is the primary execution-layer implementation. It lives in the monorepo at
  `rust/op-reth`, built on [reth](https://github.com/paradigmxyz/reth) with OP Stack
  extensions (deposit transactions, L1 fee model, OP hardforks).
- **kona-client** carries its own execution path for fault proofs (and future zk proofs).
  It does **not** embed op-reth: the proof program executes blocks via `op-revm` and
  `op-alloy` directly through `kona-executor` (`rust/kona/crates/proof/executor`), keeping the
  proof binary minimal. See [fault-proofs.md](fault-proofs.md).
- **op-geth** is a **deprecated** execution-layer client (`ethereum-optimism/op-geth`, a fork
  of go-ethereum). It is being removed in favor of op-reth, and OP-Stack-specific code is
  being extracted out of op-geth into `op-core/*` so the monorepo can depend on upstream
  go-ethereum directly. Only read [opgeth-decoupling.md](opgeth-decoupling.md) when doing
  op-geth-decoupling / op-core-extraction work — it is not needed for general execution-layer
  development.

## Scope

The execution layer: EVM execution, state transitions, block processing, and the L2-specific
modifications the OP Stack adds on top of upstream Ethereum execution.

## Key Concepts

- **Deposit transactions**: system-level transactions (type `0x7E`) originating from L1
  deposits.
- **L1 fee computation**: an additional fee component charged on each L2 transaction based on
  its L1 data cost.
- **Sequencer fee vault**: collection of L2 execution fees.
- **EIP implementation**: carrying upstream EIPs with L2 adaptations.

## Invariants

- **Deposit success**: deposit transactions always succeed at the execution level — they
  do not revert on gas. Deposit transaction handling must not break the standard EVM
  execution path.
- **L1 fee accuracy**: the L1 fee calculation must match the on-chain L1 oracle exactly.
- **Determinism**: the state transition function is deterministic. The same inputs must
  produce identical results across op-reth and kona-client's execution path.
- **Gas limit enforcement**: block gas limit enforcement must account for deposit
  transactions.

## Key Differences from Upstream Execution

- Deposit transaction type (`0x7E`) handling.
- Fee model modifications: L1 data fee and operator fee.
- Sequencer-specific block building.
- OP hardfork schedule (Canyon, Ecotone, Fjord, Granite, Holocene, Isthmus, …).

## Testing Requirements

- The upstream execution test suite must continue passing.
- Deposit transaction tests covering all edge cases.
- L1 fee calculation tests against known reference values.
- State transition consistency between op-reth and kona-client's execution path.
