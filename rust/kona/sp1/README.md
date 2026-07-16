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

The super-root aggregation program currently accepts the range program verification
key as input to support development. This dynamic-vkey mode is not production sound
until the range vkey is embedded in the aggregation program or publicly bound by the
verifier path.

### Crates (`crates/`)

Supporting libraries for the SP1 fault proof system:

- **`build`**: Build utilities for compiling SP1 programs
- **`client`**: Client-side utilities and types for witness execution in the zkVM
- **`elfs`**: Management and references to compiled ELF binaries
- **`ethereum`**: Ethereum-specific data availability utilities
  - `client/`: Client-side Ethereum DA utilities
  - `host/`: Host-side Ethereum DA witness generation
- **`host`**: Host utilities for witness generation, proof orchestration, and preimage serving
- **`proof`**: High-level proof generation utilities and workflows

### ELF Binaries (`elf/`)

Compiled ELF binaries for the zkVM programs, used by the prover:

- **`aggregation-elf`**: Compiled aggregation program
- **`range-elf`**: Compiled range program. This port keeps one range artifact
  instead of separate bump and embedded variants.
- **`super-aggregation-elf`**: Compiled super-root aggregation program
- **`super-range-elf`**: Compiled unified super-root range/consolidation program

In the optimism monorepo port, these files are generated on demand and ignored
by git, matching the Cannon prestate artifact workflow. Generate real v6.3.1
ELFs with `just build-elfs`. Host-toolchain workspace builds embed empty
build-output placeholders when generated ELFs are absent, so proving fails fast
until the real artifacts are built.

## CI TODOs

TODO(#18326): the monorepo's CircleCI runs the
workspace-wide build, clippy, tests, cargo-hack, udeps, docs, typos, and zepter
gates over the SP1 host-side crates that are workspace members. The guest
program entrypoints live in their own workspace for SP1 patch scoping and are
not covered by those host workspace gates. The following standalone-kona GitHub
workflow behavior is not yet reproduced:

- ELF build (`sp1/justfile build-elfs`, Dockerized `cargo-prove --tag v6.3.1`).
  This requires the SP1 v6.3.1 toolchain from `sp1up` and Docker in CI.
- Codecov flag wiring for SP1 coverage.
- no-std checks for the SP1/zkVM crates. The monorepo `rust-check-no-std` job is
  package-allowlisted and does not include SP1; add SP1 there if no-std coverage
  is wanted.

### Guest Precompile Patches

The guest programs under `programs/` are isolated in `programs/Cargo.toml`, a
nested Cargo workspace with its own `Cargo.lock` and `[patch.crates-io]` table.
That workspace patches `sha2`, `sha3`, `crypto-bigint`, `k256`, `p256`, and
`substrate-bn` to the SP1 forks, so the generated ELFs get zkVM
precompile-accelerated crypto without changing the host `rust/` workspace
dependency graph.

The EVM-executing range and super-range guests also enable `revm`'s `bn` feature
in the nested workspace. That forwards to `revm-precompile`'s `substrate-bn`
backend for EIP-196/197 bn128 precompiles. EIP-2537 BLS pairing still uses
arkworks and is not SP1 accelerated.

## Usage

The SP1 integration follows the same fault proof workflow as the native Kona implementation, but generates cryptographic proofs of execution:

1. **Range Proof Generation**: The `range` program executes state transitions for a block range in the zkVM, producing a validity proof
2. **Proof Aggregation**: The `aggregation` program combines multiple range proofs into a single proof for efficient on-chain verification
3. **On-chain Verification**: Proofs are submitted to the dispute game contract and verified on L1

## Building

Build utilities are provided in the `build` crate. Programs can be compiled for the zkVM target using the SP1 toolchain.

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

That recipe builds the guest ELFs (`just build-elfs`, Dockerized SP1 toolchain), builds
the `range-executor` binary (which embeds the `range` ELF), and runs the test with
`KONA_SP1_RANGE_EXECUTOR_PATH` set. The test skips when that variable is unset, so the
heavy SP1 toolchain is only required when explicitly running the SP1 action tests.

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
