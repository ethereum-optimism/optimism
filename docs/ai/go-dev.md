# Go Service Development

Guidance for AI agents working with Go code in the Optimism monorepo. See [dev-workflow.md](dev-workflow.md) for tool versions, PR workflow, and other cross-language guidance.

## Build System

Each Go service has its own justfile — run `just --list` in any service directory to see available targets.

```bash
# Build a single service (pattern: just ./<service>/<binary>)
just ./op-node/op-node

# Build all Go components
just build-go
```

### Running Tests

```bash
# Test a single service
cd <service> && just test

# Test specific packages
go test ./op-node/rollup/derive/...

# Run the full test suite from the repo root
just go-tests
```

### Generating Mocks

Each service justfile has a `generate-mocks` target:

```bash
cd <service> && just generate-mocks
```

## Linting

The repo uses a **custom golangci-lint build** with additional analyzer plugins. The standard `golangci-lint` binary will not catch all issues — always lint through `just`.

```bash
# Lint (also verifies compilation and module tidiness)
just lint-go

# Lint with auto-fix
just lint-go-fix
```

The linter configuration is in `.golangci.yaml` — read it when you need specifics on which linters are enabled and how they're scoped.

## Before Every Commit

Run these checks before committing Go changes. Fix all issues — CI enforces zero warnings.

1. **Lint** — this also verifies the code compiles and modules are tidy:
   ```bash
   just lint-go
   ```

2. **Test** — run tests for changed packages:
   ```bash
   cd <service> && just test
   ```
