# Rust Development

Guidance for AI agents working with Rust code in the Optimism monorepo. See [dev-workflow.md](dev-workflow.md) for tool versions, PR workflow, and other cross-language guidance.

## Workspace Layout

All Rust code lives under `rust/`. This is a unified Cargo workspace — always run Rust commands from this directory. The workspace contains three main component groups:

- **Kona** — Proof system and rollup node (`rust/kona/`)
- **Op-Reth** — OP Stack execution client built on reth (`rust/op-reth/`)
- **Op-Alloy / Alloy extensions** — OP Stack types and providers

Check `rust/Cargo.toml` for the full workspace member list, dependency versions, and lint configuration. The Rust toolchain version is pinned in `rust/rust-toolchain.toml`.

## Build System

Run `just --list` in `rust/` to see all available targets. The key ones:

```bash
cd rust

# Build the workspace
just build

# Build the workspace excluding example crates with the fast-build profile
just build-no-examples

# Build in release mode
just build-release

# Build specific binaries
just build-node      # kona-node
just build-op-reth   # op-reth
```

### Running Tests

Tests use `cargo-nextest` (not `cargo test`) for unit tests:

```bash
cd rust

# Run all tests (unit + doc tests)
just test

# Unit tests only (excludes online tests)
just test-unit

# Doc tests only
just test-docs
```

### Running op-reth E2E Tests

The op-reth E2E tests (`rust/op-reth/tests/proofs/`) run a full devnet with op-geth (sequencer) and op-reth (validator). They require two build prerequisites:

1. **Forge artifacts** — the devnet deploys contracts from compiled artifacts:
   ```bash
   cd packages/contracts-bedrock
   mise exec -- just build-no-tests
   ```

2. **op-reth release binary** — the test harness (`op-devstack/sysgo/rust_binary.go`) only searches `target/release/`, not `target/debug/`. Options:
   ```bash
   # Option A: let the test build it (slow first run, cached after)
   RUST_JIT_BUILD=1 go test -v -run TestName ./rust/op-reth/tests/proofs/core/

   # Option B: pre-build the binary
   cd rust && just build-op-reth
   ```

Run from the monorepo root:
```bash
mise exec -- go test -v -run TestExecutePayloadSuccess -count=1 ./rust/op-reth/tests/proofs/core/
```

### Generating Prestates

Kona prestates are built via Docker:

```bash
cd rust
just build-kona-prestates
```

## Linting

```bash
cd rust

# Run all lints (format check + clippy + doc lints)
just lint

# Individual lint steps
just fmt-check      # formatting (requires nightly)
just lint-clippy    # clippy with all features, -D warnings
just lint-docs      # rustdoc warnings
```

Lint configuration lives in `rust/Cargo.toml` (workspace lints section), `rust/clippy.toml`, and `rust/rustfmt.toml`.

### Formatting Requires Nightly

Formatting uses a pinned nightly toolchain (defined as `NIGHTLY` in `rust/justfile`). It is installed via mise.

Then use `just fmt-fix` to auto-format, or `just fmt-check` to verify.

### no_std Compatibility

Many kona and alloy crates must compile without the standard library (for the fault proof VM). If you modify these crates, verify no_std builds:

```bash
cd rust
just check-no-std
```

This builds affected crates for the `riscv32imac-unknown-none-elf` target.

## Dependency Auditing

The workspace uses `cargo-deny` for license, advisory, and dependency checks. Configuration is in `rust/deny.toml`.

```bash
cd rust
just deny
```

## Before Every Commit

Run these checks from `rust/`. Fix all issues — CI enforces zero warnings.

1. **Lint** — this checks formatting, clippy, and doc lints:
   ```bash
   just lint
   ```

2. **Test** — run tests for changed packages:
   ```bash
   just test-unit
   ```

3. **no_std** — if you changed any proof, protocol, or alloy crate:
   ```bash
   just check-no-std
   ```

## CI

Op-reth requires `clang` / `libclang-dev` for reth-mdbx-sys bindgen. CI installs this automatically — if you see bindgen errors locally, install clang.

## Updating the reth dependency

The full guide lives at [`rust/UPDATING-RETH.md`](../../rust/UPDATING-RETH.md). Read it before bumping the reth rev in `rust/Cargo.toml`.

Agent-specific tips beyond what's in the guide:

- The bump is iterative — run `cargo check --workspace --tests`, fix the first batch of errors, re-run, repeat. Don't try to enumerate every API change up front and don't ask the user to confirm every line of adaptation; just iterate to a green compile and report the diff at the end.
- If you have a local checkout of `paradigmxyz/reth`, use it to look up upstream trait signatures and run `git log <old-rev>..<new-rev>` to find the commit that changed any given symbol — much faster and more reliable than hand-fetching raw GitHub URLs. If you don't know whether one is available, ask the user. Don't assume a path.
- For trait methods that gained an ignored parameter, prefix the new param with `_` (e.g. `_block_access_list_hash: Option<B256>`) so it doesn't generate an unused-variable warning. Don't invent a meaningful name unless you're actually plumbing the value through.
- If upstream removed a trait or re-export that op-reth still uses, vendor it locally with a comment pointing at the upstream removal PR — don't try to refactor op-reth to do without it without first confirming the consumer is actually unused. The "stale" label upstream doesn't mean unused downstream.
- When the new rev pulls in new transitive deps (visible as `Adding <crate>` lines from `cargo update`), check whether they're from upstream reth's own deps or from a misconfiguration on our side. `cargo tree -i <crate>` traces the path.

## Skills

- **Fix Rust Formatting** ([`.claude/skills/fix-rust-fmt/SKILL.md`](../../.claude/skills/fix-rust-fmt/SKILL.md)): Fixes `rust-fmt` CI failures by installing the pinned nightly toolchain and running `just fmt-fix`. Invoke with `/fix-rust-fmt`.
