# OP Stack Acceptance Tests

## Overview

This directory contains the acceptance tests and configuration for the OP Stack. These tests are executed by `op-acceptor`, which serves as an automated gatekeeper for OP Stack network promotions.

Think of acceptance testing as Gandalf 🧙, standing at the gates and shouting, "You shall not pass!" to networks that don't meet our standards. It enforces the "Don't trust, verify" principle by:

- Running automated acceptance tests
- Providing clear pass/fail results (and tracking these over time)
- Gating network promotions based on test results
- Providing insight into test feature/functional coverage

The `op-acceptor` ensures network quality and readiness by running a comprehensive suite of acceptance tests before features can advance through the promotion pipeline:

Localnet → Alphanet → Betanet → Testnet

This process helps maintain high-quality standards across all networks in the OP Stack ecosystem.

## Architecture

The acceptance testing system uses `sysgo` in-process orchestration:

### **sysgo (In-process)**
- **Use case**: Fast, isolated testing without external dependencies
- **Benefits**: Quick startup, no external infrastructure needed
- **Dependencies**: None (pure Go services)

## Dependencies

### Basic Dependencies
* Mise (install as instructed in CONTRIBUTING.md)

Dependencies are managed using the repo-wide `mise` config. Run `mise install` at the repo root to install `op-acceptor` and other tools.

## Usage

### Quick Start

```bash
# Run in-process tests (fast, no external dependencies)
just acceptance-test base
```

### Available Commands

```bash
# Default: run in-process tests with base gate
just

# Run a specific gate
just acceptance-test <gate>

# Run all tests (gateless)
just acceptance-test-all
```

### Direct CLI Usage

You can also run `op-acceptor` directly:

```bash
cd op-acceptance-tests

# In-process testing
op-acceptor \
  --gate base \
  --testdir .. \
  --validators ./acceptance-tests.yaml \
  --log.level info \
  --allow-skips \
  --exclude-gates flake-shake
```

## Development Usage

### Fast Development Loop (In-process)

For rapid test development, use in-process testing:

```bash
cd op-acceptance-tests
just acceptance-test base
```

### Configuration

- `acceptance-tests.yaml`: Defines the validation gates and the suites and tests that should be run for each gate.
- `justfile`: Contains the commands for running the acceptance tests.

### Logging Configuration

When invoked with `go test`, devstack acceptance tests support configuring logging via CLI flags and environment variables. The following options are available:

* `--log.level LEVEL` (env: `LOG_LEVEL`): Sets the minimum log level. Supported levels: `trace`, `debug`, `info`, `warn`, `error`, `crit`. Default: `trace`.
* `--log.format FORMAT` (env: `LOG_FORMAT`): Chooses the log output format. Supported formats: `text`, `terminal`, `logfmt`, `json`, `json-pretty`. Default: `text`.
* `--log.color` (env: `LOG_COLOR`): Enables colored output in terminal mode. Default: `true` if STDOUT is a TTY.
* `--log.pid` (env: `LOG_PID`): Includes the process ID in each log entry. Default: `false`.

Environment variables override CLI flags. For example:
```bash
# Override log level via flag
go test -v ./op-acceptance-tests/tests/interop/sync/multisupervisor_interop/... -run TestL2CLAheadOfSupervisor -log.format=json | logdy

# Override via env var
LOG_LEVEL=info go test -v ./op-acceptance-tests/tests/interop/sync/multisupervisor_interop/... -run TestL2CLAheadOfSupervisor
```

## Adding New Tests

To add new acceptance tests:

1. Create your test in the appropriate Go package under `tests` (as a regular Go test)
2. Register the test in `acceptance-tests.yaml` under the appropriate gate
3. Follow the existing pattern for test registration:
   ```yaml
   - name: YourTestName
     package: github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/your/package/path
   ```

## Flake-Shake: Test Stability Validation

Flake-shake is a test stability validation system that runs tests multiple times to detect flakiness before they reach production gates. It serves as a quarantine area where new or potentially unstable tests must prove their reliability.

### Purpose

- Detect flaky tests through repeated execution (100+ iterations)
- Prevent unstable tests from disrupting CI/CD pipelines
- Provide data-driven decisions for test promotion to production gates

### How It Works

Flake-shake runs tests multiple times and aggregates results to determine stability:
- **STABLE**: Tests with 100% pass rate across all iterations
- **UNSTABLE**: Tests with any failures (<100% pass rate)

### Running Flake-Shake

Flake-shake is integrated into op-acceptor and can be run locally or in CI:

```bash
# Run flake-shake with op-acceptor (requires op-acceptor v3.4.0+)
op-acceptor \
  --validators ./acceptance-tests.yaml \
  --gate flake-shake \
  --flake-shake \
  --flake-shake-iterations 10

# Run with more iterations for thorough testing
op-acceptor \
  --validators ./acceptance-tests.yaml \
  --gate flake-shake \
  --flake-shake \
  --flake-shake-iterations 100
```

### Adding Tests to Flake-Shake

Add new or suspicious tests to the flake-shake gate in `acceptance-tests.yaml`:

```yaml
gates:
  - id: flake-shake
    description: "Test stability validation gate"
    tests:
      - package: github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/yourtest
        timeout: 10m
        metatada:
          owner: stefano
```

### Understanding Reports

Flake-shake stores a daily summary artifact per run:
- **`final-report/daily-summary.json`**: Aggregated counts of stable/unstable tests and per-test pass/fail tallies.

### CI Integration

In CI, flake-shake runs tests across multiple parallel workers:
- 10 workers each run 10 iterations (100 total by default)
- Results are aggregated using the `flake-shake-aggregator` tool
- Reports are stored as CircleCI artifacts

### Automated Promotion (Promoter CLI)

We provide a small CLI that aggregates the last N daily summaries from CircleCI and proposes YAML edits to promote stable tests out of the `flake-shake` gate:

```bash
export CIRCLE_API_TOKEN=...  # CircleCI API token (read artifacts)
go build -o ./op-acceptance-tests/flake-shake-promoter ./op-acceptance-tests/cmd/flake-shake-promoter/main.go
./op-acceptance-tests/flake-shake-promoter \
  --org ethereum-optimism --repo optimism --branch develop \
  --workflow scheduled-flake-shake --report-job op-acceptance-tests-flake-shake-report \
  --days 3 --gate flake-shake --min-runs 300 --max-failure-rate 0.01 --min-age-days 3 \
  --out ./final-promotion --dry-run
```

Outputs written to `--out`:
- `aggregate.json`: Per-test aggregated totals across days
- `promotion-ready.json`: Candidates and skip reasons
- `promotion.yaml`: Proposed edits to `op-acceptance-tests/acceptance-tests.yaml`

### Promotion Criteria

Tests should remain in flake-shake until they demonstrate consistent stability:
- **Immediate promotion**: 100% pass rate across 100+ iterations
- **Investigation needed**: Any failures require fixing before promotion
- **Minimum soak time**: 3 days in flake-shake gate recommended

### Quick Development

For rapid development and testing:

```bash
cd op-acceptance-tests

# Run all tests (gateless mode) - most comprehensive coverage
just acceptance-test-all

# Run specific gate-based tests (traditional mode)
just acceptance-test base
```

## Further Information

For more details about `op-acceptor` and the acceptance testing process, refer to the main documentation or ask the team for guidance.

The source code for `op-acceptor` is available at [github.com/ethereum-optimism/infra/op-acceptor](https://github.com/ethereum-optimism/infra/tree/main/op-acceptor). If you discover any bugs or have feature requests, please open an issue in that repository.
