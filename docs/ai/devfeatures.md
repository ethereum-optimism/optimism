# DevFeatures System

Reference for the `DevFeatures` bitmap system. The bitmap is transitional infrastructure that lets in-development features ride alongside production code without a full hardfork. It is expected to disappear once each feature it gates either ships as a real hardfork or is abandoned (see #20084 for the L2CM/CannonKona cleanup).

## A. What it is

A 32-byte bitmap (`bytes32` on chain, `common.Hash` in Go) where each feature occupies its own nibble — `0x...01`, `0x...10`, `0x...100`, etc. — giving 64 concurrent feature slots. Defined identically on both sides:

- **Go**: `op-core/devfeatures/devfeatures.go` — flag constants, `IsDevFeatureEnabled(bitmap, flag)`, `EnableDevFeature(bitmap, flag)`
- **Solidity**: `packages/contracts-bedrock/src/libraries/DevFeatures.sol` — same constants, same predicate

The two sides are independent literals with **no compile-time sync check** — a mismatch silently diverges Go and Solidity behavior. The constant values are canonical in those two files; don't trust copies elsewhere.

Active flags:

| Flag | Gates |
|------|-------|
| `OptimismPortalInterop` | Interop migration functions on OptimismPortal2 |
| `CannonKona` | Respected-game-type override for CANNON_KONA during upgrades — **enabled by default** |
| `DeployV2DisputeGames` | Legacy, no longer used; constant kept for historical reasons |
| `L2CM` | L2ContractsManager + supporting L2 predeploys — **enabled by default** |
| `ZKDisputeGame` | ZK dispute game system |
| `SuperRootGamesMigration` | Super-root games migration path in OPCM upgrade |

The predicate is a bitwise AND (`(bitmap & flag) == flag && flag != 0`) — except that `IsDevFeatureEnabled` short-circuits to `true` for `L2CM` and `CannonKona`, which are enabled by default on both the Go and Solidity sides (the bitmap no longer acts as a circuit breaker for them; removal tracked in #20084).

**Adding a new dev feature**: the full checklist lives in the `DevFeatures.sol` natspec — both constant files, the env-var reader in `scripts/libraries/Config.sol`, the test assembler in `test/setup/FeatureFlags.sol`, and the CI `&features_matrix` anchor in `.circleci/continue/main.yml` all need updating; there is no compile-time link between them.

## B. Inputs — where operators supply the bitmap

The bitmap has **two operator-facing input surfaces**, both in op-deployer:

1. **CLI flag on `op-deployer bootstrap implementations`**
   - `--dev-feature-bitmap` (env: `OP_DEPLOYER_DEV_FEATURE_BITMAP`), defined in `op-deployer/pkg/deployer/bootstrap/flags.go`
   - Raw 32-byte hex; default empty
   - Flows into `ImplementationsConfig.DevFeatureBitmap` and on into `DeployImplementationsInput` for L1 implementation deployment.

2. **Intent file (`globalDeployOverrides.devFeatureBitmap`)**
   - Schema field on `Intent`, `op-deployer/pkg/deployer/state/intent.go`
   - Lives in the operator's intent TOML/JSON
   - Read by the L2 genesis pipeline.

There is no other production operator-facing surface. `op-node`, `op-program`, `kona`, and rollup config do not take a bitmap at runtime.

### Developer-only: `op-node/rollup/toggles.go`

A third input mechanism, distinct from the bitmap, lives in `op-node/rollup/toggles.go`. It is a **source-code-edit-only** developer toggle — not exposed via config, CLI, or env var. The file holds ephemeral `Is<Feature>(time)` methods on `*Config` that map an in-development feature to its eventual hardfork timestamp:

```go
func (c *Config) IsL2CM(time uint64) bool {
    return c.IsKarst(time)
}
```

The file's comment instructs developers to "replace with `return false` to disable NUT bundle execution during development." This is a temporary scaffold for the active fork scope; entries are removed after the fork is locked.

What it gates is **runtime activation timing** in op-node (when NUT bundle deposit transactions are emitted at the activation block), which is orthogonal to what the bitmap gates (whether the supporting predeploys and contracts exist on a given chain at all).

### Test-only: per-feature env vars

A separate, **test-only** assembler exists for Foundry tests and fork scripts. It is isolated from the production path — no `src/` contract and no production deploy script reads these env vars.

- `DEV_FEATURE__OPTIMISM_PORTAL_INTEROP`
- `DEV_FEATURE__L2CM`
- `DEV_FEATURE__ZK_DISPUTE_GAME`
- `DEV_FEATURE__CANNON_KONA`
- `DEV_FEATURE__SUPER_ROOT_GAMES_MIGRATION`

Each is read via `vm.envOr(..., false)` in `packages/contracts-bedrock/scripts/libraries/Config.sol` (functions `devFeatureInterop()`, `devFeatureL2CM()`, etc.). The only callers are under `test/`:

- `test/setup/FeatureFlags.sol` — `resolveFeaturesFromEnv()` OR-s each enabled flag into `devFeatureBitmap`
- `test/setup/CommonTest.sol`, `test/setup/ForkL1Live.s.sol`, `test/setup/ForkL2Live.s.sol` — branch on individual `Config.devFeature*` returns
- `test/L1/OPContractsManagerStandardValidator.t.sol` — `vm.skip()` based on flag state

These env vars exist purely to set up local test fixtures and to skip / branch tests. They never reach a deployed chain. To exercise a feature in production you must set the bitmap via op-deployer.

## C. Composition — how those inputs become a single bitmap

All composition happens in op-deployer at deploy / genesis time. The central function is:

- **`buildDevFeatureBitmap()`** in `op-deployer/pkg/deployer/pipeline/l2genesis.go`
  - Reads `intent.GlobalDeployOverrides["devFeatureBitmap"]`
  - **Cross-validates** the interop bit against the separate `intent.UseInterop` boolean — fails the deploy if they disagree
  - Returns the bitmap to the genesis builder.

What this does **not** do: it does not OR together a list of named feature toggles. The operator hands in a raw bitmap; op-deployer just validates one bit (interop) against a parallel boolean, then passes the bitmap through verbatim. The CLI flag and the intent override are independent surfaces — they are not merged; whichever subcommand runs uses its own input.

The Foundry test-only assembler in `FeatureFlags.sol` is a separate composition layer entirely; it is never invoked by op-deployer.

## D. Propagation — where the bitmap travels

From op-deployer the bitmap fans out three ways:

1. **Into L2 genesis state** — `scripts/L2Genesis.s.sol` (Foundry script) writes the bitmap into the **`L2DevFeatureFlags` predeploy at `0x42...2D`** via `setDevFeatureBitmap()`. Only `DEPOSITOR_ACCOUNT` can write; effectively write-once at genesis.
2. **Into L1 implementation deployment** — `scripts/deploy/DeployImplementations.s.sol` consults the bitmap to decide which implementation contracts to deploy / configure.
3. **Into NUT bundle generation** — `scripts/upgrade/GenerateNUTBundle.s.sol` uses it to decide which predeploy upgrades to include.

It does **not** flow to op-node, op-program, or kona at runtime. They learn about feature activation through other channels (hardfork timestamps in rollup config, primarily).

## E. Activation — who reads it and what they gate

**On L2 (Solidity)**, readers consult either the in-memory bitmap (during deploy scripts) or the `L2DevFeatureFlags` predeploy (at runtime):

- `Predeploys.isSupportedPredeploy(...)` in `src/libraries/Predeploys.sol` takes the bitmap as a parameter. Gates whether `CONDITIONAL_DEPLOYER` (0x2C), `L2_DEV_FEATURE_FLAGS` (0x2D), and the interop predeploys are considered "real" predeploys.
- `scripts/L2Genesis.s.sol` — gates which predeploys get installed at genesis.
- `L2ContractsManager._isDevFeatureEnabled()` in `src/L2/L2ContractsManager.sol` — at runtime, calls `IL2DevFeatureFlags.isDevFeatureEnabled(...)` on the predeploy to decide whether to execute upgrade flows.

**On Go**, the bitmap is essentially deploy-time-only:

- The interop cross-check in `buildDevFeatureBitmap()` noted above.
- No runtime callers. (op-node / op-program use hardfork timestamps, not the bitmap.)

**The L2DevFeatureFlags predeploy API** (`src/L2/L2DevFeatureFlags.sol`):

- `setDevFeatureBitmap(bytes32)` — DEPOSITOR_ACCOUNT only, set at genesis
- `devFeatureBitmap()` — read full bitmap
- `isDevFeatureEnabled(bytes32 feature)` — per-flag query

## F. Hardfork interaction

For **L2CM specifically** there are two independent gates:

1. **Hardfork timestamp** — `IsL2CM(time)` in `op-node/rollup/toggles.go` returns `IsKarst(time)`. This decides **when** L2CM upgrade transactions are executed across the network.
2. **Bitmap** — gates **whether the L2CM machinery exists at all** on a given chain (predeploys installed, implementations deployed, upgrade paths active). Since #20439 the L2CM bit is enabled by default, so on new deploys the machinery is always provisioned.

So: hardfork is the network-wide "go" signal; bitmap is the per-chain "is this feature provisioned" switch. Other flags (interop, super-root migration) are bitmap-only, no parallel hardfork timestamp.

## G. Lifecycle direction

- L2CM and CannonKona are now **default-on** (#20439): `IsDevFeatureEnabled` / `isDevFeatureEnabled` return `true` for them regardless of the bitmap.
- **#20084** tracks removing the L2CM and CannonKona flags (and eventually the wider DevFeatures scaffolding) once L2CM is proven in production.

## File index

| Concern | Path |
|---|---|
| Go constants & predicate | `op-core/devfeatures/devfeatures.go` |
| Solidity constants & predicate | `packages/contracts-bedrock/src/libraries/DevFeatures.sol` |
| CLI input | `op-deployer/pkg/deployer/bootstrap/flags.go` |
| Intent input + composition | `op-deployer/pkg/deployer/pipeline/l2genesis.go` |
| Genesis writer | `packages/contracts-bedrock/scripts/L2Genesis.s.sol` |
| L2 storage | `packages/contracts-bedrock/src/L2/L2DevFeatureFlags.sol` |
| L2 runtime reader | `packages/contracts-bedrock/src/L2/L2ContractsManager.sol` |
| Predeploy gating | `packages/contracts-bedrock/src/libraries/Predeploys.sol` |
| L2CM hardfork gate | `op-node/rollup/toggles.go` |
| Test-only env vars | `packages/contracts-bedrock/scripts/libraries/Config.sol` |
| Test bitmap assembler | `packages/contracts-bedrock/test/setup/FeatureFlags.sol` |
