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
- **`range-elf`**: Compiled range program. SP1 v6.2.4 no longer exposes a
  separate bump-allocator feature, so this port keeps one range artifact instead
  of separate bump and embedded variants.

In the optimism monorepo port, these files are generated on demand and ignored
by git, matching the Cannon prestate artifact workflow. Generate real v6.2.4
ELFs with `just build-elfs`. Host-toolchain workspace builds embed empty
build-output placeholders when generated ELFs are absent, so benchmarks fail fast
until the real artifacts are built.

## CI TODOs

TODO(#21418): the monorepo's CircleCI runs the
workspace-wide build, clippy, tests, cargo-hack, udeps, docs, typos, and zepter
gates over the SP1 crates now that they are workspace members. The following
standalone-kona GitHub workflow behavior is not yet reproduced:

- ELF build (`sp1/justfile build-elfs`, Dockerized `cargo-prove --tag v6.2.4`).
  This requires the SP1 v6.2.4 toolchain from `sp1up` and Docker in CI.
- Codecov flag wiring for SP1 coverage.
- no-std checks for the SP1/zkVM crates. The monorepo `rust-check-no-std` job is
  package-allowlisted and does not include SP1; add SP1 there if no-std coverage
  is wanted.

## Usage

The SP1 integration follows the same fault proof workflow as the native Kona implementation, but generates cryptographic proofs of execution:

1. **Range Proof Generation**: The `range` program executes state transitions for a block range in the zkVM, producing a validity proof
2. **Proof Aggregation**: The `aggregation` program combines multiple range proofs into a single proof for efficient on-chain verification
3. **On-chain Verification**: Proofs are submitted to the dispute game contract and verified on L1

### Benchmarking

The local range benchmark runs the range program against real RPC data with the
SP1 prover:

```bash
export L1_RPC=<l1-rpc-url>
export L2_RPC=<l2-rpc-url>
export L2_NODE_RPC=<op-node-rpc-url>
# Required for post-Ecotone/blob-backed ranges:
export L1_BEACON_RPC=<l1-beacon-rpc-url>

just build-elfs
just range-bench --start <l2-start-block> --end <l2-end-block>
```

By default, `just range-bench` executes the guest and prints cycle/SP1-gas
statistics. Pass `--prove` to additionally produce a compressed proof
and report the proving wall-clock time. Pass `--save-proof <path>` with
`--prove` to persist the compressed range proof for aggregation.

The range benchmark can also split RPC witness generation from proving. On a
machine with RPC access, save the generated SP1 stdin:

```bash
just range-bench --start <l2-start-block> --end <l2-end-block> --save-stdin /tmp/range.stdin
```

On another machine, load that stdin and prove it without any RPC environment
variables or block-range flags:

```bash
just range-bench --load-stdin /tmp/range.stdin --prove --save-proof /tmp/range.bin
```

`--save-stdin` can be combined with `--prove`, and it can also re-save a stdin
loaded with `--load-stdin`. When `--load-stdin` is used, per-block execution
statistics are skipped because they require RPC; the bench still reports
execution cycles and SP1 gas.

Aggregate consecutive saved range proofs with:

```bash
just range-bench --start <a> --end <b> --prove --save-proof /tmp/r1.bin
just range-bench --start <b> --end <c> --prove --save-proof /tmp/r2.bin
just agg-bench --proofs /tmp/r1.bin,/tmp/r2.bin
```

`agg-bench` expects compressed range proofs in ascending chain order. The ranges
must be consecutive, because the aggregation program asserts each proof's post
root is the next proof's pre root. Pass `--prove` to produce the compressed
recursion aggregation proof. The execute cycle report excludes recursive proof
verification, which appears in the `--prove` wall-clock time.

Produce the final PLONK aggregation proof with:

```bash
just plonk-prove-bench --proofs /tmp/r1.bin,/tmp/r2.bin --save-proof /tmp/agg.plonk.bin
```

`plonk-prove-bench` consumes the same consecutive compressed range proofs as
`agg-bench`, always proves, verifies the PLONK proof in-process, and reports the
prove wall-clock, local verify time, and on-chain calldata size. SP1 runs the full
pipeline for PLONK (core -> compress -> shrink -> wrap -> gnark), so estimate the
PLONK wrapping cost as the delta between `plonk-prove-bench` and
`agg-bench --prove` on the same inputs. The default local CPU PLONK path requires
Docker for the gnark container (`SP1_GNARK_IMAGE` overrides the default image)
and outbound network on first run to download circuit artifacts to `~/.sp1`
(`SP1_PLONK_CIRCUIT_PATH` overrides the cache path).

#### Hardware acceleration

All three benches build their prover with SP1's `ProverClient::from_env()`, so
the proving backend is selected by `SP1_PROVER`. If `SP1_PROVER` is unset, SP1
uses the local CPU prover, which is the default behavior.

`ProverClient::from_env()` also honors SP1's non-local network backends because
the workspace enables the `sp1-sdk/network` feature. Setting
`SP1_PROVER=network` or `SP1_PROVER=hosted` routes proof generation through the
SP1 prover network and requires `NETWORK_PRIVATE_KEY`. Use the unset/default CPU
backend or `SP1_PROVER=cuda` for local hardware benchmarking.

For CUDA proving, build the bench binary with the `gpu` feature and set
`SP1_PROVER=cuda`:

```bash
SP1_PROVER=cuda \
  cargo run --release -p kona-sp1-range-bench --features gpu --bin range-bench -- \
  --start <l2-start-block> --end <l2-end-block> --prove

# Or through just:
SP1_PROVER=cuda just features=gpu range-bench \
  --start <l2-start-block> --end <l2-end-block> --prove
```

The same `SP1_PROVER=cuda` plus `features=gpu` combination applies to
`agg-bench` and `plonk-prove-bench`. CUDA proving requires an NVIDIA GPU with
Compute Capability >= 8.6 and at least 24 GB VRAM, plus a compatible NVIDIA
driver/CUDA runtime. SP1 downloads and starts `~/.sp1/bin/sp1-gpu-server` on the
first run, sets `CUDA_VISIBLE_DEVICES=0`, and talks to that local server over a
Unix socket. For `SP1_PROVER=cuda`, the bench sends the selected proof mode,
including PLONK, to that server instead of running SP1's local CPU prover path.
The bench does not pass GPU flags into SP1's gnark Docker wrapper. Building
without `--features gpu` and setting `SP1_PROVER=cuda` fails with SP1's upstream
"requires the `cuda` feature" message.

CPU vector acceleration needs no cargo feature. Compile the CPU prover with
vector extensions via `RUSTFLAGS`:

```bash
# AVX2:
RUSTFLAGS="-C target-cpu=native" just range-bench \
  --start <l2-start-block> --end <l2-end-block> --prove

# AVX-512, on CPUs that support it:
RUSTFLAGS="-C target-cpu=native -C target-feature=+avx512f" just range-bench \
  --start <l2-start-block> --end <l2-end-block> --prove
```

SP1's Plonky3 field arithmetic uses AVX2 and AVX-512 automatically when present.
`RUSTFLAGS` and `SP1_PROVER=cuda` are orthogonal, so CPU codegen flags can be
combined with the CUDA backend build when useful.

The generated ELF files are ignored by git, so real `range-elf` and
`aggregation-elf` artifacts must be generated locally with `just build-elfs`
before running the benchmarks.

The benchmark crate is a native host tool. It is intentionally outside the
zkVM/no-std and WASM target allowlists.

## Building

Build utilities are provided in the `build` crate. Programs can be compiled for the zkVM target using the SP1 toolchain.

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
