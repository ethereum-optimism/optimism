# Fault Proofs Development

This document provides guidance for AI agents working with the fault proof system in
the Optimism monorepo: the Cannon MIPS VM, kona-client, and the dispute game contracts.
See [go-dev.md](go-dev.md) for Go workflow, [rust-dev.md](rust-dev.md) for the Rust
kona-client workflow, and [contract-dev.md](contract-dev.md) for the Solidity dispute game
contracts.

## Scope

- `cannon/` — MIPS32 VM that executes kona-client in single-step mode.
- `rust/kona/bin/client/` — kona-client, the primary fault proof program: a Rust binary
  (`kona-client`) that derives and executes L2 state inside the VM.
- `packages/contracts-bedrock/src/dispute/` — on-chain dispute game contracts.

## Key Concepts

- **Cannon**: a MIPS32 VM that executes kona-client in single-step mode, producing a
  reproducible execution trace.
- **kona-client**: the primary fault proof program — a Rust binary that derives and executes
  L2 state. It executes blocks via `op-revm`/`op-alloy` through `kona-executor`, not op-reth.
- **Dispute game**: an on-chain bisection game to resolve output root disputes.
- **Preimage oracle**: the mechanism for the VM to load external data (L1 blocks, L2 state).

## Invariants

- **kona-client determinism**: the same inputs always produce the same output root. kona-client
  must be fully deterministic — no network calls, no filesystem access at runtime; all external
  data is served through the preimage oracle.
- **Trace reproducibility**: the Cannon execution trace must be reproducible from any
  starting state.
- **Preimage fidelity**: the preimage oracle must serve exactly the data requested, with
  no corruption. Preimage key computation must match exactly between Go and Solidity.
- **Resolution finality**: dispute game resolution must be final and correct.

## Security Considerations

- VM instruction handling must produce identical results on-chain (Solidity) and
  off-chain (Rust/Go). Memory access in Cannon must be bounds-checked.
- Preimage key collision resistance.
- Game clock management and bond economics.
- Any change to dispute game mechanics requires formal security review.

## Testing Requirements

- Differential testing between the Cannon Go and Solidity VM implementations.
- End-to-end dispute game tests covering honest and dishonest scenarios.
- Fuzz testing for VM instruction handlers.
- Preimage oracle consistency tests.
