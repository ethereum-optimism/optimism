# Acceptance Tests

Guidance for AI agents building and running acceptance tests in the Optimism monorepo. See [dev-workflow.md](dev-workflow.md) for tool versions and general workflow.

## What Are Acceptance Tests?

Acceptance tests live in `op-acceptance-tests/tests/` and run full in-process devnet scenarios. They exercise the entire stack — contracts, Go services, and Rust binaries — in a single `go test` process. This means all dependencies must be built locally before running them.

## Running Tests

All `just` targets below automatically build dependencies (contracts, cannon prestates, Rust binaries) before running tests. The builds are incremental — re-running is fast when nothing changed.

**Prerequisites:** mise tools must be installed (see [dev-workflow.md](dev-workflow.md#setup)), and a working C toolchain (`clang` or `gcc`) must be available for Rust builds.

Always set `RUST_JIT_BUILD=1` when running locally. This lets the test framework automatically build any Rust binaries it needs (e.g. op-reth) using cargo's rebuild detection, so you don't have to pre-build them separately.

Run from `op-acceptance-tests/`:

### Specific Tests or Packages (recommended)

```bash
# Run a single test
RUST_JIT_BUILD=1 cd op-acceptance-tests && mise exec -- just test -run TestMyTest ./op-acceptance-tests/tests/base/

# Run a package
RUST_JIT_BUILD=1 cd op-acceptance-tests && mise exec -- just test ./op-acceptance-tests/tests/base/...
```

The `just test` target builds deps, then runs `go test -count=1 -timeout 30m` with your arguments.

### All Tests

```bash
RUST_JIT_BUILD=1 cd op-acceptance-tests && mise exec -- just acceptance-test
```

Runs all test packages with gotestsum, structured logging, and auto-tuned parallelism.

### Gated Subsets

Gate files in `op-acceptance-tests/gates/` list package subsets:

```bash
RUST_JIT_BUILD=1 cd op-acceptance-tests && mise exec -- just acceptance-test base
```

This runs only packages listed in `gates/base.txt`.

### Kona Reproducible Prestate

Some tests (e.g. superfaultproofs, interop fault proofs) require a reproducible kona prestate. This is **not** handled by `build-deps` or `RUST_JIT_BUILD`:

```bash
mise exec -- just reproducible-prestate-kona
```

**Requires Docker.** If Docker is not available in your environment, ask the user to run this command for you.

## What `build-deps` Does

The `just build-deps` target (called automatically by `just test` and `just acceptance-test`) runs these steps when not in CI:

1. **mise** — `mise install` (ensures gotestsum, forge, etc. are available)
2. **Contracts** — `cd packages/contracts-bedrock && just install && just build-no-tests`
3. **Cannon prestates** — `just cannon-prestates` (builds cannon, op-program, and prestate artifacts)
4. **Rust binaries** — `just build-rust-release` (kona-node, op-rbuilder, rollup-boost)

You can also run `just build-deps` directly to pre-build without running tests.

## Tuning Parallelism (`acceptance-test` only)

When using `just acceptance-test`, the runner auto-detects CPU count and sets:
- `ACCEPTANCE_TEST_JOBS` — number of packages to test in parallel (default: CPU count)
- `ACCEPTANCE_TEST_PARALLEL` — `go test -parallel` value per package (default: CPU count / 2)
- `ACCEPTANCE_TEST_TIMEOUT` — per-package timeout (default: 2h)

Override with environment variables:

```bash
ACCEPTANCE_TEST_PARALLEL=2 ACCEPTANCE_TEST_TIMEOUT=1h cd op-acceptance-tests && mise exec -- just acceptance-test
```

## Log Output (`acceptance-test` only)

When using `just acceptance-test`, logs are written to `op-acceptance-tests/logs/testrun-<timestamp>/`:
- `all.log` — full test output
- `raw_go_events.log` — JSON test events
- `flaky-tests.txt` — tests marked with `MarkFlaky()`

Results XML goes to `op-acceptance-tests/results/results.xml`.

When using `just test`, output goes to stdout only.

## Common Issues

- **Missing prestates** — Run `cd op-acceptance-tests && mise exec -- just build-deps` or `mise exec -- just cannon-prestates` from the repo root.
- **Stale contracts** — Rebuild with `cd packages/contracts-bedrock && mise exec -- just build-no-tests`.
- **Missing Rust binaries** — Run `mise exec -- just build-rust-release` from the repo root.
- **gotestsum not found** — Run `mise install` to install all pinned tools.
