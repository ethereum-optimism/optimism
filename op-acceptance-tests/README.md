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

## Dependencies

* Docker
* Kurtosis
* Mise (install as instructed in CONTRIBUTING.md)

Dependencies are managed using the repo-wide `mise` config. So ensure you've first run `mise install` at the repo root. If you need to manually modify the version of op-acceptor you wish to run you'll need to do it within the _mise.toml_ file at the repo root.

## CI Usage

The tests can be run using the `just` command runner:

```bash
# Run the default acceptance tests against a simple devnet
just

# Run the acceptance tests against a specific devnet and gate
just acceptance-test <devnet> <gate>

# Run the acceptance tests using a specific version of op-acceptor
ACCEPTOR_IMAGE=op-acceptor:latest just acceptance-test
```

## Development Usage

The above command works great for CI but less well for development because it pessimistically rebuilds kurtosis each time, regardless of whether anything has changed in the underlying Optimism services build.
We will fix this in the future, but the workaround is to decouple your kurtosis deployment from test running by manually managing each.

1. Deploy kurtosis

   ```
   cd kurtosis-devnet
   just isthmus-devnet # or whatever devnet you intend to test
   # will take ~3-4m to spin up
   ```

2. Run tests with devnet descriptor env var configuration (instant)

   ```
   cd ../op-acceptance-tests
   DEVNET_ENV_URL=kt://isthmus-devnet/devnet-descriptor-0 go test -v ./tests/isthmus/...
   ```

This should allow you to tweak your test in a tight loop while the test subject remains the same.

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