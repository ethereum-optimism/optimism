# OP Stack Acceptance Tests

## Overview

This directory contains the OP Stack acceptance tests. They run against the in-process `sysgo` devstack and are executed as normal Go tests.

The supported execution path is:

- `just` or `just acceptance-test-all`
- `gotestsum -- go test ./op-acceptance-tests/tests/...`

`devtest.T.MarkFlaky(...)` is used for tests that should downgrade failures to skips in the normal acceptance run. Set `DEVNET_FAIL_FLAKY_TESTS=true` to force those tests to fail normally. Acceptance runs also emit a `flaky-tests.txt` artifact in `op-acceptance-tests/logs/...` listing the current `MarkFlaky(...)` call sites.

## Dependencies

Install repo tools via `mise` as documented in the repository root. Local acceptance runs also build contract and Rust dependencies when needed.

## Usage

### Quick Start

```bash
cd op-acceptance-tests
just
```

### Available Commands

```bash
# Default: run all acceptance test packages
just

# Explicit alias
just acceptance-test
just acceptance-test-all
```

### Direct CLI Usage

```bash
gotestsum --format testname --junitfile ./op-acceptance-tests/results/results.xml -- \
  -count=1 -p 4 -parallel 4 -timeout 2h ./op-acceptance-tests/tests/...
```

The `just` wrapper computes defaults from available CPUs:

- package jobs: CPU count
- in-package `t.Parallel`: half the CPU count
- timeout: `2h`

Override them with `ACCEPTANCE_TEST_JOBS`, `ACCEPTANCE_TEST_PARALLEL`, and `ACCEPTANCE_TEST_TIMEOUT`.

## Logging

When invoked with `go test`, devstack acceptance tests support configuring logging via CLI flags and environment variables:

- `--log.level LEVEL` or `LOG_LEVEL`
- `--log.format FORMAT` or `LOG_FORMAT`
- `--log.color` or `LOG_COLOR`
- `--log.pid` or `LOG_PID`

Example:

```bash
LOG_LEVEL=info go test -v ./op-acceptance-tests/tests/interop/sync/multisupervisor_interop/... -run TestL2CLAheadOfSupervisor
```

## Adding Tests

Add new acceptance tests as ordinary Go tests under [`tests`](./tests). There is no external gate or manifest to update.

If a test is currently flaky in the normal acceptance run, mark it in code with `devtest.T.MarkFlaky(...)`. That keeps the source of truth next to the test itself while the acceptance logs and flaky-test artifacts provide the review surface for recent failures.
