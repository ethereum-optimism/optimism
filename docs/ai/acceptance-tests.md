# Running Acceptance Tests

Guidance for AI agents building and running acceptance tests in the Optimism monorepo. For guidance on *writing* new tests — DSL patterns, naming, what to avoid — see [writing-acceptance-tests.md](writing-acceptance-tests.md). See [dev-workflow.md](dev-workflow.md) for tool versions and general workflow.

## What Are Acceptance Tests?

Acceptance tests live in `op-acceptance-tests/tests/` and run full in-process devnet scenarios. They exercise the entire stack — contracts, Go services, and Rust binaries — in a single `go test` process. This means all dependencies must be built locally before running them.

## Running Tests

The `just test` target automatically builds dependencies (contracts, cannon prestates, Rust binaries) before running tests. The builds are incremental — re-running is fast when nothing changed.

**Prerequisites:** mise tools must be installed (see [dev-workflow.md](dev-workflow.md#setup)), and a working C toolchain (`clang` or `gcc`) must be available for Rust builds.

Always set `RUST_JIT_BUILD=1` when running locally. This lets the test framework automatically build any Rust binaries it needs (e.g. op-reth) using cargo's rebuild detection, so you don't have to pre-build them separately.

Run from `op-acceptance-tests/`:

### Specific Tests or Packages (recommended)

```bash
# Run a single test
cd op-acceptance-tests && RUST_JIT_BUILD=1 mise exec -- just test -run TestMyTest ./op-acceptance-tests/tests/base/

# Run a package
cd op-acceptance-tests && RUST_JIT_BUILD=1 mise exec -- just test ./op-acceptance-tests/tests/base/...
```

The `just test` target builds deps, then runs `go test -count=1 -timeout 30m` with your arguments.

### All Tests

```bash
cd op-acceptance-tests && RUST_JIT_BUILD=1 mise exec -- just acceptance-test
```

Runs all test packages with gotestsum, structured logging, and bounded parallelism.

### Selecting the L2 Clients

Every run covers the whole `./op-acceptance-tests/tests/...` tree; there is no per-package
subset target. Which clients the devstack starts is chosen by two environment variables, read
in `op-devstack/sysgo/mixed_runtime.go`:

- `DEVSTACK_L2CL_KIND` — `op-node` (the default) or `kona-node`
- `DEVSTACK_L2EL_KIND` — `op-reth` (the default) or `op-geth`

```bash
cd op-acceptance-tests && DEVSTACK_L2CL_KIND=kona-node RUST_JIT_BUILD=1 mise exec -- just acceptance-test
```

CI runs two variants of the `op-acceptance-tests` job: `memory-all-opn-op-reth-<l1_fork>`
(op-node + op-reth) and `memory-all-kona-op-reth-<l1_fork>` (kona-node + op-reth).

A test that cannot run under a given client skips itself in code rather than being left out of
a list — `sysgo.SkipOnKonaNode`, `sysgo.SkipOnOpReth` and `sysgo.SkipOnOpGeth`.

### Kona Prestate

Some tests (e.g. superfaultproofs, interop fault proofs) require a kona prestate. This is **not** handled by `build-deps` or `RUST_JIT_BUILD`. There are two ways to build it:

**Reproducible build** (preferred when Docker is available):

```bash
mise exec -- just reproducible-prestate-kona
```

This produces a prestate whose hash matches CI/release builds. It works on any host with Docker installed.

**Native build** (fallback when Docker is not available):

```bash
cd rust && mise exec -- just build-kona-prestates
```

Only works on **Linux** with the **MIPS cross-compile toolchain** installed. The produced hash will not match release builds, so this is only suitable for local test runs where the hash doesn't need to match a deployed release. If neither Docker nor the MIPS toolchain is available, ask the user to build the prestate for you.

## What `build-deps` Does

The `just build-deps` target (called automatically by `just test`) runs these steps when not in CI:

1. **mise** — `mise install` (ensures gotestsum, forge, etc. are available)
2. **Contracts** — `cd packages/contracts-bedrock && just install && just build-no-tests`
3. **Cannon prestates** — `just cannon-prestates` (builds the kona prestate artifacts)
4. **Rust binaries** — `just build-rust-release` (kona-node, kona-host, op-reth, and the test-only
   `op-reth-sdm-fixture`). The fixture is built only for tests and is excluded from production
   packages and images.

You can also run `just build-deps` directly to pre-build without running tests.

## Tuning Parallelism (`acceptance-test` only)

When using `just acceptance-test`, the runner sets:
- `ACCEPTANCE_TEST_JOBS` — number of packages to test in parallel (default: 12)
- `ACCEPTANCE_TEST_PARALLEL` — `go test -parallel` value per package (default: 1)
- `ACCEPTANCE_TEST_TIMEOUT` — per-package timeout (default: 30m)

Override with environment variables:

```bash
cd op-acceptance-tests && ACCEPTANCE_TEST_PARALLEL=2 ACCEPTANCE_TEST_TIMEOUT=1h mise exec -- just acceptance-test
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
