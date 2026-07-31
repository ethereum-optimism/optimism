# `sp1`

This directory contains an integration of [SP1](https://github.com/succinctlabs/sp1) zero-knowledge proof capabilities into Kona, enabling validity proofs for OP Stack state transitions. This integration is derived from the [OP-Succinct](https://github.com/succinctlabs/op-succinct) project.

> **⚠️ Experimental**: The SP1 fault proof integration is currently experimental and under active development. It is not yet recommended for production use.

## Overview

The SP1 integration provides zkVM-based fault proofs for the OP Stack, allowing verifiable state transitions to be proven on-chain. This enables trustless bridging and enhanced security for rollup chains.

## Structure

### Programs (`programs/`)

zkVM programs that execute inside the SP1 prover:

- **`range`**: Verifies OP Stack state transitions across a range of L2 blocks with Ethereum DA. Generates proofs for multi-block execution that can be verified on-chain.
- **`aggregation`**: Aggregates multiple range program proofs into a single proof, enabling efficient verification of longer block ranges.
- **`super-range`**: Scaffold for the unified multi-chain super-root range
  program, with modes for proving ranges and span-shaped consolidation.
- **`super-aggregation`**: Recursively verifies unified super-range proofs and
  commits the public values consumed by `ZKDisputeGame`.

### Crates (`crates/`)

Supporting libraries for the SP1 fault proof system:

- **`client`**: Client-side utilities and types for witness execution in the zkVM
- **`elfs`**: Runtime loading of compiled ELF binaries
- **`ethereum`**: Ethereum-specific data availability utilities
  - `client/`: Client-side Ethereum DA utilities
  - `host/`: Host-side Ethereum DA witness generation
- **`host`**: Host utilities for witness generation, proof orchestration, and preimage serving
- **`range-vkeys`**: Compile-time range and super-range guest verification keys, embedded from
  generated `elf/vkeys.toml` and used only by the aggregation guests
- **`proposer`**: The `kona-sp1-proposer` service: creates super-root ZK dispute games,
  defends challenged ones with SP1 super-aggregation proofs, resolves finished games, and
  claims bonds (see the Proposer section below)
- **`super-range-executor`**: Witness synthesis and execution engine for the super-root
  programs; used as a library by the proposer and as the `kona-sp1-super-range-executor`
  validity-checker binary in acceptance tests
- **`range-executor`**: Host binary running the single-chain `range` guest in SP1 execute
  mode for action tests

### ELF Binaries (`elf/`)

Compiled ELF binaries for the zkVM programs, used by the prover:

- **`aggregation-elf`**: Compiled aggregation program
- **`range-elf`**: Compiled range program. This port keeps one range artifact
  instead of separate bump and embedded variants.
- **`super-aggregation-elf`**: Compiled super-root aggregation program
- **`super-range-elf`**: Compiled unified super-root range/consolidation program

In the optimism monorepo port, these files and `elf/vkeys.toml` are generated on demand and
ignored by git, matching the Cannon prestate artifact workflow. Generate reproducible ELFs
on linux/amd64 with `just build-elfs`; it builds the leaf guests first, generates their vkeys, and
then builds the aggregation guests with those vkeys embedded through `kona-sp1-range-vkeys`. Use
`just build-elfs-native` for local iteration and the fast per-PR compile check; CI persists the
native manifest with the generated ELFs. Native ELF hashes may differ across build environments
because paths and other environment details are embedded. A Docker-based, uncached tag/release
reproducibility CI check is intentionally left to a future follow-up.

Host-toolchain workspace builds need neither ELFs nor `vkeys.toml`. Host binaries load guest
artifacts at runtime from `KONA_SP1_ELF_DIR`; a missing or empty artifact fails as an
infrastructure error. Release automation will eventually pin per-version vkeys from the generated
manifest into `superchain-registry/validation/standard/standard-prestates.toml` and verify
reproducible builds.

Custom chains and devnets can compile separate SP1 artifacts with custom kona
registry inputs:

```bash
KONA_CUSTOM_CONFIGS_DIR=/path/to/custom/configs just build-elfs
```

The directory must contain `chainList.json`, `configs.json`, and `depsets.json`.
Those files are compiled into the kona crates used by the guest programs, so
custom configs produce different ELFs and verification-key hashes.

## CI TODOs

TODO(#18326): the monorepo's CircleCI runs the
workspace-wide build, clippy, tests, cargo-hack, udeps, docs, typos, and zepter
gates over the SP1 host-side crates that are workspace members. The guest program entrypoints and
`range-vkeys` crate live outside that workspace. The `kona-build-sp1-elfs` rust-e2e job runs
`just build-elfs-native`, tests and lints all guests, and checks and tests `range-vkeys`; scheduled
vkey drift coverage is tracked in #21661. The following standalone-kona GitHub workflow behavior
is not yet reproduced:

- Codecov flag wiring for SP1 coverage.
- no-std checks for the SP1/zkVM crates. The monorepo `rust-check-no-std` job is
  package-allowlisted and does not include SP1; add SP1 there if no-std coverage
  is wanted.

### Guest Precompile Patches

All four guest programs are isolated in `programs/Cargo.toml`, a nested Cargo workspace with its
own `Cargo.lock` and `[patch.crates-io]` table. That workspace patches `sha2`, `sha3`,
`crypto-bigint`, `k256`, `p256`, and `substrate-bn` to the SP1 forks, so the
generated ELFs get zkVM precompile-accelerated crypto without changing the host
`rust/` workspace dependency graph.

The EVM-executing range and super-range guests also enable `revm`'s `bn` feature
in the nested workspace. That forwards to `revm-precompile`'s `substrate-bn`
backend for EIP-196/197 bn128 precompiles. EIP-2537 BLS pairing still uses
arkworks and is not SP1 accelerated.

## Usage

The SP1 integration follows the same fault proof workflow as the native Kona implementation, but generates cryptographic proofs of execution:

1. **Range Proof Generation**: The `range` program executes state transitions for a block range in the zkVM, producing a validity proof
2. **Proof Aggregation**: The `aggregation` program combines multiple range proofs into a single proof for efficient on-chain verification
3. **On-chain Verification**: Proofs are submitted to the dispute game contract and verified on L1

## Proposer (`kona-sp1-proposer`)

The proposer service (ported from op-succinct's fault-proof proposer) plays the
super-root `ZKDisputeGame` (game type 10) end to end:

1. **Create**: proposes the supernode's canonical super root on a fixed interval
   (`proposer.rs` creation scheduling).
2. **Defend**: when a game in the owned set is challenged, fetches the span's super
   roots, collects witnesses through the kona `InteropHost`
   (`super-range-executor` library), obtains SP1 proofs (`prover.rs` providers,
   `proving.rs` pipeline), and submits `prove()`.
3. **Resolve and claim**: resolves finished games and claims bonds (including the
   challenger bond earned by proving).

### Ownership (which games it defends)

Ownership is prestate-based: the
proposer proves, resolves, and claims every game whose `absolutePrestate()`
artifacts it can load, regardless of creator. The three sets are the same set.
Games whose prestate is unknown are skipped with the
`kona_sp1_proposer_unknown_prestate_challenged` gauge as the alarm.

**Operational requirement**: rotated-out prestate artifacts must remain published
under `PRESTATES_URL` for as long as games created under them can be live, or the
proposer loses the ability to defend, resolve, and claim those games.

### Proof providers

- `PROOF_PROVIDER=network`: real SP1 proving via the Succinct Prover Network.
  Proving keys are set up per prestate on first use, and the aggregation
  verifying key must hash to the on-chain prestate (mismatches poison the
  prestate and remove its games from the owned set). The registered prestate's
  keys are verified BEFORE any game is created on it, so the proposer never
  bonds a game it has not proven it can defend.
- `PROOF_PROVIDER=mock`: dev-only. Runs the full pipeline natively (witness
  collection computes the real range/consolidation outputs and the aggregation
  inputs are validated), then submits placeholder proof bytes. Only a deployment
  with a mock game verifier (devstack) accepts them. No ELFs, no SPN credentials.

### Restart behavior

There is no in-flight proof-request recovery (upstream parity): the task map is
in-memory, and a restart re-detects still-challenged games and re-requests their
proofs from scratch. A pre-flight check prevents duplicate `prove()` submissions.

### Operator alarms

`kona_sp1_proposer_game_proving_error` and
`kona_sp1_proposer_proving_timeout_error` are spend alarms in network mode: every
emergent retry after a post-proving failure (for example a misrouted
`AGG_PROOF_MODE` vs the on-chain verifier, or fee caps below basefee) re-purchases
the full proof set until the prove deadline expires. A sustained non-zero rate
means money burning, not a transient. `kona_sp1_proposer_game_unprovable` counts
games given up as permanently unprovable (kept in-memory until restart).

### Environment

Required:

| Variable | Purpose |
|---|---|
| `L1_RPC` | L1 execution RPC |
| `SUPERNODE_RPC` | supernode (or single-chain op-node) RPC serving `superroot_atTimestamp` |
| `FACTORY_ADDRESS` | `DisputeGameFactory` address |
| `PRESTATES_URL` | prestate artifact directory (`<vkey>.agg.bin.gz` + `<vkey>.range.bin.gz`) |
| `PROOF_PROVIDER` | `network` or `mock`; no default |
| `L1_BEACON_RPC` | L1 beacon API (blob sidecars for derivation witnesses) |
| `L2_RPCS` | comma-separated L2 EL RPCs, one per chain (order-irrelevant) |
| `PRIVATE_KEY` or `SIGNER_URL`+`SIGNER_ADDRESS` | L1 transaction signer |

Optional (defaults in parentheses):

| Variable | Purpose |
|---|---|
| `ROLLUP_CONFIG_PATHS`, `L1_CONFIG_PATH`, `DEPENDENCY_SET_PATH` | chain config files; absent = superchain-registry fallback, matching the executor CLI |
| `PROPOSAL_INTERVAL_SECONDS` (3600), `PROPOSAL_SAFETY` (finalized), `FETCH_INTERVAL` (30) | proposal cadence |
| `METRICS_PORT` (0 = disabled), `SYNC_L1_CONFIRMATIONS` (0), `TX_CONFIRMATION_TIMEOUT` (60) | operations |
| `MAX_FEE_PER_GAS`, `MAX_PRIORITY_FEE_PER_GAS` | L1 fee caps in wei (unset = uncapped) |
| `RANGE_SPLIT_COUNT` (1, max 16) | chunks a defended span is split into |
| `MAX_CONCURRENT_RANGE_PROOFS` (1) | child-proof concurrency within one game |
| `MAX_CONCURRENT_DEFENSE_TASKS` (8) | games defended concurrently (must be >= 1) |
| `NETWORK_PRIVATE_KEY` (network mode; `USE_KMS_REQUESTER` for AWS KMS) | SPN requester key |
| `RANGE_PROOF_STRATEGY`, `AGG_PROOF_STRATEGY` (reserved) | SPN fulfillment strategies |
| `AGG_PROOF_MODE` (plonk) | on-chain proof kind, `plonk` or `groth16` |
| `SP1_TIMEOUT_SECONDS` (14400), `NETWORK_CALLS_TIMEOUT` (15), `AUCTION_TIMEOUT` (60) | SPN timeouts |
| `RANGE_CYCLE_LIMIT`, `RANGE_GAS_LIMIT`, `AGG_CYCLE_LIMIT`, `AGG_GAS_LIMIT` (1e12) | SPN request limits |
| `MAX_PRICE_PER_PGU` (3e8), `MIN_AUCTION_PERIOD` (1) | SPN pricing |

Fast finality (proving at creation) is tracked in
ethereum-optimism/optimism#22112.

## Building

Programs are compiled for the zkVM target through the recipes in this directory's `justfile`.

The `cargo prove` subcommand is pinned by `mise.toml`. Install the native Succinct
toolchain for non-Docker local builds with:

```bash
just install-sp1-toolchain
```

## Testing (SP1 execute action tests)

The `range-executor` crate (`crates/range-executor`) builds a host binary,
`kona-sp1-range-executor`, that runs the `range` guest in SP1 **execute** mode (no
proving) against a real chain's witness. It accepts the same boot inputs as the native
kona-host `single` CLI, generates the witness via the kona-host preimage server, runs the
`range` ELF in the SP1 emulator, and exits `0` (valid claim) / `1` (invalid claim) / `2`
(infrastructure error) — mirroring the native fault-proof program convention.

The op-e2e action test `TestSP1RangeSimpleEmptyChain`
(`rust/kona/tests/proofs/sp1_simple_program_test.go`) drives this binary against an
in-process action-test chain, exercising the program end-to-end on real inputs. Run it
with:

```bash
cd rust/kona/tests && just action-tests-sp1
```

That recipe builds the guest ELFs (`just build-elfs`, Dockerized SP1 toolchain), builds the
`range-executor` binary, and runs the test with `KONA_SP1_RANGE_EXECUTOR_PATH` and
`KONA_SP1_ELF_DIR` set. The executor loads the `range` ELF at runtime. The test skips when the
executor-path variable is unset, so the heavy SP1 toolchain is only required when explicitly
running the SP1 action tests.

For faster coverage of the range-program logic, the same executor also supports
`--native-core`. This mode still generates the real witness, but runs the shared range
core natively instead of executing the SP1 ELF. Use the default SP1 execute path for a
small smoke test of the ELF, SP1 stdin, and public-values boundary; use `--native-core`
when broad action-test coverage would otherwise multiply SP1 emulator cost.

The test covers both an honest claim and an invalid claim. Note the invalid-claim path is
driven by **corrupting the claim in the witness**, not by passing a wrong claimed output
root: witness generation runs on the configured `--claimed-l2-output-root`, and the
host-side generator rejects a wrong one *before* the guest runs (a confusing infra error,
exit 2). So an invalid-claim test keeps the real claim and sets the `--corrupt-claimed-root`
flag (via `WithCorruptClaim()` in the Go harness), which tampers the claim in the generated
witness so the guest re-derives the real root, finds the mismatch, and aborts (exit 1) — a
soundness smoke test that a false transition cannot be executed (and thus could not be
proven). Do **not** write an SP1 negative test by passing a junk `WithL2Claim(...)`.

## Dependencies

This integration depends on:
- SP1 SDK and zkVM runtime
- Core Kona libraries (`kona-proof`, `kona-derive`, `kona-executor`, etc.)
- Alloy and OP-Alloy for Ethereum types
- RocksDB for witness data storage

## License and Attribution

This implementation is derived from [OP-Succinct](https://github.com/succinctlabs/op-succinct)
by Succinct Labs and incorporates code licensed under the MIT License and Apache License 2.0.
Significant modifications have been made to integrate with the Kona monorepo architecture.

See [LICENSE-THIRD-PARTY](./LICENSE-THIRD-PARTY) for full license details and attribution.
