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

Formatting uses a pinned nightly toolchain (defined as `NIGHTLY` in `rust/justfile`). If the nightly isn't installed:

```bash
cd rust
just install-nightly
```

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

## Skills

- **Fix Rust Formatting** ([`.claude/skills/fix-rust-fmt/SKILL.md`](../../.claude/skills/fix-rust-fmt/SKILL.md)): Fixes `rust-fmt` CI failures by installing the pinned nightly toolchain and running `just fmt-fix`. Invoke with `/fix-rust-fmt`.
