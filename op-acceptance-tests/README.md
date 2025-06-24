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

The acceptance testing system supports two orchestrator modes:

### **sysgo (In-process)**
- **Use case**: Fast, isolated testing without external dependencies
- **Benefits**: Quick startup, no external infrastructure needed
- **Dependencies**: None (pure Go services)

### **sysext (External)**
- **Use case**: Testing against Kurtosis-managed devnets or persistent networks
- **Benefits**: Testing against realistic network conditions
- **Dependencies**: Docker, Kurtosis (for Kurtosis devnets)

The system automatically selects the appropriate orchestrator based on your usage pattern.

## Dependencies

### Basic Dependencies
* Mise (install as instructed in CONTRIBUTING.md)

### Additional Dependencies (for external devnet testing)
* Docker
* Kurtosis

Dependencies are managed using the repo-wide `mise` config. Run `mise install` at the repo root to install `op-acceptor` and other tools.

## Usage

### Quick Start

```bash
# Run in-process tests (fast, no external dependencies)
just acceptance-test "" base

# Run against Kurtosis devnets (requires Docker + Kurtosis)
just acceptance-test simple base
just acceptance-test isthmus isthmus
just acceptance-test interop interop
```

### Available Commands

```bash
# Default: Run tests against simple devnet with base gate
just

# Run specific devnet and gate combinations
just acceptance-test <devnet> <gate>

# Use specific op-acceptor version
ACCEPTOR_VERSION=v1.0.0 just acceptance-test simple base
```

### Direct CLI Usage

You can also run the acceptance test wrapper directly:

```bash
cd op-acceptance-tests

# In-process testing (sysgo orchestrator)
go run cmd/main.go --orchestrator sysgo --gate base --testdir .. --validators ./acceptance-tests.yaml --acceptor op-acceptor

# External devnet testing (sysext orchestrator)
go run cmd/main.go --orchestrator sysext --devnet simple --gate base --testdir .. --validators ./acceptance-tests.yaml --kurtosis-dir ../kurtosis-devnet --acceptor op-acceptor

# Remote network testing
go run cmd/main.go --orchestrator sysext --devnet "kt://my-network" --gate base --testdir .. --validators ./acceptance-tests.yaml --acceptor op-acceptor
```

## Development Usage

### Fast Development Loop (In-process)

For rapid test development, use in-process testing:

```bash
cd op-acceptance-tests
just acceptance-test "" base  # Uses sysgo orchestrator - faster!
```

### Testing Against External Devnets

For integration testing against realistic networks:

1. **Automated approach** (rebuilds devnet each time):
   ```bash
   just acceptance-test isthmus isthmus
   ```

2. **Manual approach** (once-off)
   ```bash
   cd op-acceptance-tests
   # This spins up a devnet, then runs op-acceptor
   go run cmd/main.go --orchestrator sysext --devnet "interop" --gate interop --testdir .. --validators ./acceptance-tests.yaml
   ```

3. **Manual approach** (faster for repeated testing):
   ```bash
   # Deploy devnet once
   cd kurtosis-devnet
   just isthmus-devnet

   # Run tests multiple times against the same devnet
   cd op-acceptance-tests
   # This runs op-acceptor (devnet spin up is skipped due to `--reuse-devnet`)
   go run cmd/main.go --orchestrator sysext --devnet "interop" --gate interop --testdir .. --validators ./acceptance-tests.yaml --reuse-devnet
   ```

### Configuration

- `acceptance-tests.yaml`: Defines the validation gates and the suites and tests that should be run for each gate.
- `justfile`: Contains the commands for running the acceptance tests.
- `cmd/main.go`: Wrapper binary that handles orchestrator selection and devnet management.

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

1. Create your test in the appropriate Go package (as a regular Go test)
2. Register the test in `acceptance-tests.yaml` under the appropriate gate
3. Follow the existing pattern for test registration:
   ```yaml
   - name: YourTestName
     package: github.com/ethereum-optimism/optimism/your/package/path
   ```

## Further Information

For more details about `op-acceptor` and the acceptance testing process, refer to the main documentation or ask the team for guidance.

The source code for `op-acceptor` is available at [github.com/ethereum-optimism/infra/op-acceptor](https://github.com/ethereum-optimism/infra/tree/main/op-acceptor). If you discover any bugs or have feature requests, please open an issue in that repository.