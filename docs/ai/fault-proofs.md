# Fault Proofs Development

This document provides guidance for AI agents working with the fault proof system in
the Optimism monorepo: the Cannon MIPS VM, kona-client, and the dispute game contracts.
See [go-dev.md](go-dev.md) for Go workflow, [rust-dev.md](rust-dev.md) for the Rust
kona-client workflow, and [contract-dev.md](contract-dev.md) for the Solidity dispute game
contracts.

kona-client is the only fault proof program. op-program was deleted in
ethereum-optimism/optimism#21271 and is not supported. Do not compare against op-program or
ask for op-program changes.

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
  `super-range` and `super-aggregation` programs. Single-chain coverage uses a dependency set of
  size one.
- **Kona-SP1 proof limits**: if an honest kona-sp1 proof exceeds a resource limit, the proposer
  loses its `ZKDisputeGame`. See [kona-sp1-proof-limits.md](kona-sp1-proof-limits.md).

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

**Plain Go action tests** (`op-e2e/actions/...`) run their L2 execution engine as an
`op-reth-test-engine` subprocess, so they also need that binary. `just test-actions` (from
`op-e2e`) builds it and points the tests at it. Run directly, `go test` uses
`RUST_BINARY_PATH_OP_RETH_TEST_ENGINE` if set, otherwise the newest release or debug build in the
cargo target directory, and otherwise fails with instructions. Build it once:

```bash
# from rust/ — release build, lands in rust/target/release/ (or under CARGO_TARGET_DIR)
mise x -- just build-op-reth-test-engine
# from the repo root
go test ./op-e2e/actions/upgrades/ -run TestName -count=1
```

Alternatively `RUST_JIT_BUILD=1` has the tests run the release build themselves; a cold build
takes long, so give `go test` a `-timeout` that covers it (e.g. `-timeout 60m`).

**Kona fault-proof action tests** (`rust/kona/tests/proofs/...`, e.g.
`TestActivationBlockNUTBundle`) run their L2 chain on the same `op-reth-test-engine` and
additionally run kona-client through the native `kona-host` binary, which reads the L2 chain from
the engine over its IPC socket. So they need both binaries built, and `KONA_HOST_PATH` pointing at
kona-host. These tests are excluded from the normal `go test ./...` suite — they only run under
kona-host.

A kona-host run that does not finish within 10 minutes fails the test. The host retries a
preimage it cannot fetch indefinitely, and the engine's `debug_dbGet` serves contract code only,
so a trie node missing from an execution witness shows up as that timeout.

> [!NOTE]
> These tests cover **op-node as well as kona-client**. The chain is built and derived by op-node
> — the action-test `L2Sequencer`/`L2Verifier` drives `op-node/rollup/derive` through
> `PreparePayloadAttributes` → `PayloadToSystemConfig` — and the `RunFaultProofProgram` step then
> has kona-client re-derive and prove that same chain. So a single test exercises both
> consensus-layer implementations of the state transition.

```bash
# build the native host and the engine (release; heavy builds) from rust/, where the toolchain is pinned
cd rust
mise x -- cargo build --release --bin kona-host
mise x -- just build-op-reth-test-engine

# run, pointing KONA_HOST_PATH at the host binary
cd kona/tests/proofs
KONA_HOST_PATH="$(git rev-parse --show-toplevel)/rust/target/release/kona-host" \
  go test -run TestActivationBlockNUTBundle -count=1 .
```

Or let the justfile build both binaries and run in one step (still needs the contracts built
above): from `rust/kona/tests`, `just action-tests-single TestActivationBlockNUTBundle`.
