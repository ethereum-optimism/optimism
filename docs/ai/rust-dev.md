# Rust Development

Guidance for AI agents working with Rust code in the Optimism monorepo.

## Tool Versions

All tool versions are pinned in `mise.toml` at the repo root and `rust/rust-toolchain.toml` for the Rust toolchain. Always access tools through mise — never install or invoke system-global versions directly. Check these files for current pinned versions.

If mise reports the repo isn't trusted, ask the user to run `mise trust` — never trust it automatically.

## Workspace Layout

All Rust code lives under `rust/`. This is a unified Cargo workspace — always run Rust commands from this directory. The workspace contains three main component groups:

- **Kona** — Proof system and rollup node (`rust/kona/`)
- **Op-Reth** — OP Stack execution client built on reth (`rust/op-reth/`)
- **Op-Alloy / Alloy extensions** — OP Stack types and providers

Check `rust/Cargo.toml` for the full workspace member list, dependency versions, and lint configuration.

## Build System

Rust targets use Just. Run `just --list` in `rust/` to see all available targets. The key ones:

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

## Before Every PR

Everything in "Before Every Commit" plus:

1. **Run affected tests broadly** — don't just test the crate you changed. Test crates that depend on it too.

2. **Rebase on `develop`** — this is the default branch, not `main`:
   ```bash
   git fetch origin develop
   git rebase origin/develop
   ```

3. **Follow PR guidelines** — see `docs/handbook/pr-guidelines.md`.

## CI

Some tests require CI-only environment variables and are skipped locally. Check the test code for environment variable guards if a test behaves differently than expected.

Build dependencies: op-reth requires `clang` / `libclang-dev` for reth-mdbx-sys bindgen. CI installs this automatically — if you see bindgen errors locally, install clang.

## Skills

- **Fix Rust Formatting** ([`.claude/skills/fix-rust-fmt/SKILL.md`](../../.claude/skills/fix-rust-fmt/SKILL.md)): Fixes `rust-fmt` CI failures by installing the pinned nightly toolchain and running `just fmt-fix`. Invoke with `/fix-rust-fmt`.
