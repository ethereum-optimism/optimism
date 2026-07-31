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

### The `op-core/superchain` bundle

`op-core/superchain` `//go:embed`s `superchain-configs.zip`, which is **gitignored** (only
its `.sha256` is committed). Any package that transitively imports it — op-node and most
binaries, plus `packages/contracts-bedrock/scripts/go-ffi`, `op-e2e`, `op-acceptance-tests`,
op-deployer, and the kona/op-reth Go tests — won't compile until the bundle is built:

```
op-core/superchain/chain.go:NN: pattern superchain-configs.zip: no matching files found
```

The `just` targets handle this: `just build-go`, `just go-tests`, and `just lint-go` all
depend on `build-superchain-go`, which builds the bundle from the superchain-registry
submodule and verifies it against the committed `.sha256`. You only hit the embed error on
a **bare** `go build`/`go test` or a fresh checkout. To prep it manually:

```bash
just build-superchain-go   # Go bundle only (fast; verify mode)
just sync-superchain        # all superchain bundles (Go + kona/op-reth Rust) — the registry-bump command
```

Because the zip is usually already on disk, a missing-bundle problem is invisible locally
and only surfaces in CI's clean checkout (see [ci-ops.md](ci-ops.md)).

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

## Conventions

- **Pointers to values**: use `ptr.New(v)` from `github.com/ethereum-optimism/optimism/op-service/ptr` to take the address of a literal or expression — common for optional `*uint64` config fields like fork-activation times (`cfg.SomeTime = ptr.New(uint64(123))`). Don't define a local `ptr`/`ptrTo` helper; the shared one avoids per-package duplicates, and a local `func ptr` collides with importing the `ptr` package in the same package.

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
