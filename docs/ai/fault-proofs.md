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
- **Kona-SP1 acceptance boundary**: tests exercise the shipping super-root path through the
  `super-range` and `super-aggregation` programs. A single chain is represented by a dependency
  set of size one; the output-root-only programs are not acceptance targets.

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

## Running action tests

Action tests drive the sequencer/verifier/miner through scripted actions. They are Go
tests, but they need the L2 contract artifacts (the genesis the env deploys is built from
them). Tools (`forge`, `go`, `just`) come from mise — prefix commands with `mise x --` outside
a mise-activated shell, and run `mise trust` once in a fresh worktree.

**Build the contracts first (prerequisite for all action tests).** Build once, then reuse:

```bash
# from packages/contracts-bedrock — compile the contracts
mise x -- just forge-build --skip test
# from op-deployer — bundle the artifacts where the e2e env reads them
mise x -- just copy-contract-artifacts
```

**Plain Go action tests** (`op-e2e/actions/...`) then run directly:

```bash
go test ./op-e2e/actions/upgrades/ -run TestName -count=1
```

**Kona fault-proof action tests** (`rust/kona/tests/proofs/...`, e.g.
`TestActivationBlockNUTBundle`) additionally run kona-client through the native `kona-host`
binary, so they need it built and `KONA_HOST_PATH` pointing at it. These tests are excluded
from the normal `go test ./...` suite — they only run under kona-host.

> [!NOTE]
> These tests cover **op-node as well as kona-client**. The chain is built and derived by op-node
> — the action-test `L2Sequencer`/`L2Verifier` drives `op-node/rollup/derive` through
> `PreparePayloadAttributes` → `PayloadToSystemConfig` — and the `RunFaultProofProgram` step then
> has kona-client re-derive and prove that same chain. So a single test exercises both
> consensus-layer implementations of the state transition.

```bash
# build the native host (release; heavy build)
cargo build --release --manifest-path rust/Cargo.toml --bin kona-host

# run, pointing KONA_HOST_PATH at the binary
cd rust/kona/tests/proofs
KONA_HOST_PATH="$(git rev-parse --show-toplevel)/rust/target/release/kona-host" \
  go test -run TestActivationBlockNUTBundle -count=1 .
```

Or let the justfile build kona-host and run in one step (still needs the contracts built
above): from `rust/kona/tests`, `just action-tests-single TestActivationBlockNUTBundle`.
