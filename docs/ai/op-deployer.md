# OP Deployer Knowledge Base

This is an AI-agent reference for `op-deployer`, based on the repository state inspected on 2026-05-05. It is meant to complement, not replace, the public operator docs in `docs/public-docs/chain-operators/tools/op-deployer/`.

## What It Is

`op-deployer` is the OP Stack contract deployment CLI and Go library. It deploys and configures L1 contracts for OP Chains, generates L2 genesis allocs, renders rollup/deploy config outputs, and exposes helper commands for bootstrapping shared contracts, verifying deployed contracts, and producing upgrade calldata.

Primary code lives in:

- `op-deployer/cmd/op-deployer/main.go` - binary entrypoint.
- `op-deployer/pkg/cli/app.go` - CLI command tree.
- `op-deployer/pkg/deployer/` - command implementations and deployment library.
- `op-deployer/pkg/env/host.go` - script host construction.
- `docs/public-docs/chain-operators/tools/op-deployer/` - canonical public docs source.

The public docs were relocated from an old mdBook into `docs/public-docs`; see `op-deployer/RELOCATED-DOCS.md`.

## Current Command Surface

The CLI app is named `op-deployer` and registers these top-level commands:

- `init` - creates `intent.toml` and `state.json`.
- `apply` - applies an intent through the deployment pipeline.
- `bootstrap` - deploys global Superchain contracts and implementation contracts.
- `inspect` - renders outputs from `state.json`.
- `verify` - verifies deployed contracts with Forge against Etherscan, Blockscout, or custom verifier APIs.
- `upgrade` - generates OPCM upgrade calldata for versioned upgrade scripts.
- `manage` - deprecated management commands for game-type/migration workflows.
- `validate` - validates deployment state and/or runs `op-validator`.
- `clean` - currently only cleans the cache.

Global flags:

- `--cache-dir`, default `~/.op-deployer/cache`, env `DEPLOYER_CACHE_DIR`.
- Standard OP logging flags: `--log.level`, `--log.format`, `--log.color`, `--log.pid`.

Environment variables mostly use the `DEPLOYER_` prefix, except `--l1-rpc-url`, which also accepts `L1_RPC_URL`.

## Build And Release

From `op-deployer/`:

```bash
just build
just test
just test ./pkg/deployer/state/...
```

`just build` runs:

1. `just ../packages/contracts-bedrock/build-no-tests`
2. Go build of `./cmd/op-deployer` into `./bin/op-deployer`

Contract artifacts are no longer embedded into release binaries. Commands that
need artifacts must receive a concrete `file://`, `http://`, or `https://`
locator.

Release details:

- GoReleaser config: `op-deployer/.goreleaser.yaml`
- Monorepo tag prefix: `op-deployer/`
- Binary name: `op-deployer`
- Supported release OS/arch: linux, darwin, windows on amd64/arm64 except windows/arm64.
- Docker target: `op-deployer` in `docker-bake.hcl`, built through `ops/docker/op-stack-go/Dockerfile`.

## Artifact Locators

Artifact locators feed Solidity artifacts into the in-memory script engine or Forge.

Current code supports:

- `file://...` - local artifact directory. If it contains `forge-artifacts/`, that subdir is used.
- `http://...` / `https://...` - downloads a tarball into the cache and extracts it.

Current code rejects `embedded` and `tag://` locators. Use explicit `file://`,
`http://`, or `https://` locators.

This differs from several public docs pages that still show
`tag://op-contracts/...` examples. For current-code work, use explicit
`file://` / `https://` locators unless you are intentionally working on an
older release lineage.

HTTP downloads are cached by SHA256 of the URL. HTTP extraction temp dirs are
registered for cleanup on process exit.

## Forge Usage

`op-deployer` has two Forge-related concepts:

- Deployment execution can use the default Go `script.Host` engine or `--use-forge`.
- Contract verification always uses Forge's `verify-contract`.

Forge binary behavior:

- Pinned version is `v1.2.3` in `op-deployer/pkg/deployer/forge/version.json`.
- If a matching `forge` is on `PATH`, it is used.
- Otherwise a checksummed Foundry release tarball is downloaded into `~/.op-deployer/cache/forge`.
- `FORGE_ENV=alpine` switches the expected OS key to `alpine`.

`--use-forge` runs Solidity scripts through `forge script --broadcast --private-key ...` with byte-encoded Go structs. The default path runs scripts inside `op-chain-ops/script.Host` and broadcasts through the Go broadcaster.

## Data Model

`intent.toml` describes desired chain configuration. It is represented by `state.Intent`:

- `configType`: `standard`, `standard-overrides`, or `custom`.
- `opDeployerVersion`
- `l1ChainID`
- `opcmAddress`
- `superchainConfigProxy`
- `superchainRoles`
- `fundDevAccounts`
- `l1ContractsLocator`
- `l2ContractsLocator`
- `chains`
- `globalDeployOverrides`
- `useInterop`
- `l1DevGenesisParams`

`state.json` records progress and final outputs. It is represented by `state.State`:

- `version` - currently only state version `1` is supported.
- `opDeployerVersion`
- `create2Salt`
- `appliedIntent`
- `prestateManifest`
- `interopDepSet`
- `superchainContracts`
- `superchainRoles`
- `implementationsDeployment`
- `opChainDeployments`
- `l1StateDump`
- `deploymentCalldata`

Each chain intent includes:

- Chain ID as a 256-bit hash.
- Fee vault recipients: base fee, L1 fee, sequencer fee, operator fee.
- EIP-1559 parameters.
- L2 gas limit.
- Roles: L1/L2 proxy admin owners, system config owner, unsafe block signer, batcher, proposer, challenger.
- `deployOverrides` and `globalDeployOverrides` for JSON merge overrides.
- Dangerous or specialized settings: Alt-DA, additional dispute games, operator fees, start block hash, min base fee, DA footprint scalar, custom gas token, L2 dev genesis params.

Important validation rules:

- L1 chain ID cannot be zero.
- L1 and L2 contract locators must be set.
- At least one L2 chain must be defined.
- Fee vault recipients cannot be the zero address.
- All chain roles must be non-zero.
- EIP-1559 denominator, Canyon denominator, and elasticity cannot be zero.
- Gas limit cannot be zero.
- `standard` intents reject non-standard values and only work on supported standard L1s.
- `standard-overrides` starts with standard defaults but validates through the custom path.
- `custom` requires either an OPCM address or Superchain roles. If OPCM is set, Superchain roles must not be set.
- `superchainRoles` includes `SuperchainProxyAdminOwner`, `SuperchainGuardian`, and `Challenger`; the challenger is needed by implementation deployment.
- Custom gas token is enabled only when both `name` and `symbol` are set. `initialLiquidity` defaults to `type(uint248).max`; `liquidityControllerOwner` defaults to `l2ProxyAdminOwner`.

## Intent Types

`standard`:

- Uses empty L1/L2 artifact locator placeholders; users must provide explicit artifact locators before apply.
- Resolves standard OPCM, challenger, and role addresses from the superchain registry validation data.
- Requires the user to manually fill L1 and L2 proxy admin owners before apply.
- Currently only standard L1 chain IDs supported by `standard.L1VersionsFor` work: mainnet `1` and Sepolia `11155111`.

`standard-overrides`:

- Generated from the standard template.
- Allows changing values because it validates through `validateCustomConfig`.
- Common for test deployments and examples that need standard defaults plus custom role or parameter changes.

`custom`:

- Starts with zero/empty fields and Superchain roles.
- Intended for standalone deployments, custom L1s, L3s, RaaS workflows, and non-Optimism-governed setups.
- Users must fill all required values before `apply`.

## Init

Command:

```bash
op-deployer init \
  --l1-chain-id 11155111 \
  --l2-chain-ids 1234,5678 \
  --workdir .deployer \
  --intent-type standard-overrides
```

Outputs:

- `.deployer/intent.toml`
- `.deployer/state.json`

`--outdir` is an alias for `--workdir`. L2 chain IDs are parsed with `op_service.Parse256BitChainID`, so decimal and 256-bit hex inputs are accepted.

## Apply

Command:

```bash
op-deployer apply \
  --workdir .deployer \
  --l1-rpc-url "$L1_RPC_URL" \
  --private-key "$DEPLOYER_PRIVATE_KEY"
```

Deployment targets:

- `live` - default. Forks a live L1, simulates scripts, and sends transactions. Requires `--l1-rpc-url` and `--private-key`.
- `genesis` - writes contracts into an L1 dev genesis state. `--l1-rpc-url` must be blank.
- `calldata` - simulates against an L1 RPC and stores calldata in `state.json`; it does not broadcast.
- `noop` - simulates with an RPC and does not broadcast.

Live apply requires the Forge deterministic deployer at `0x4e59b44847b379578588920ca78fbf26c0b4956c` to have code on the L1.

Extra apply flags:

- `--op-program-svc-url` - calls OP Program SVC to build prestates after genesis/rollup are available.
- `--verify` plus verifier flags - auto-verifies after deployment.
- `--use-forge` - executes scripts through Forge instead of the Go script host.
- `--validate` - empty disables validation; `auto` detects validator version from state/standard tag; explicit values are normalized to `op-contracts/<value>` if needed.

Apply writes `state.json` after every stage so recoverable failures can be resumed.

## Apply Pipeline

The pipeline is in `deployer.ApplyPipeline` and runs these stages:

1. `init`
2. `deploy-superchain`
3. `deploy-implementations`
4. Per chain: `deploy-opchain-<id>`
5. Per chain: `deploy-alt-da-<id>`
6. Per chain: `deploy-additional-dispute-games-<id>`
7. Per chain: `generate-l2-genesis-<id>`
8. Genesis target only: `prefund-l2-dev-genesis`
9. Genesis target only: `prefund-l1-dev-genesis`
10. Genesis target only: `preinstall-l1-dev-genesis`
11. Genesis target only: `seal-l1-dev-genesis`
12. Per chain: `set-start-block-<id>`
13. `generate-interop-depset`
14. `deploy-pre-state`

After each stage, the broadcaster is flushed and state is written. For `calldata`, `CalldataBroadcaster.Dump()` is stored at the end.

Stage notes:

- `init` checks state version, creates a random `create2Salt`, populates predeployed OPCM/Superchain state, checks live L1 chain ID, and verifies the deterministic deployer exists.
- If only `opcmAddress` is supplied, live init resolves `SuperchainConfig` from OPCM on-chain.
- `deploy-superchain` deploys `SuperchainConfig` and the Superchain proxy admin unless state already has a Superchain deployment.
- `deploy-implementations` deploys OPCM, implementation contracts, blueprints, validators, fault proof contracts, and related singletons unless already present.
- `deploy-opchain` calls OPCM to deploy chain-specific L1 contracts and then reads implementation addresses back from chain/script output.
- `deploy-alt-da` only runs when `dangerousAltDAConfig.useAltDA` is true and commitment type is Keccak.
- `deploy-additional-dispute-games` requires the deployer address to equal the chain L1 proxy admin owner. It supports Alphabet, Cannon, Cannon Next, and ZK dispute games.
- `generate-l2-genesis` calls `L2Genesis.s.sol`, dumps allocs, and applies L2 genesis overrides, fork schedule, custom gas token config, dev feature bitmap, and interop settings.
- `set-start-block` uses `l1StartBlockHash` if present, otherwise latest L1 block. Genesis target uses the sealed L1 genesis block.
- `generate-interop-depset` currently creates an entry for every chain in the intent.
- `deploy-pre-state` is skipped unless an OP Program SVC URL is supplied.

## Script Engine

Default deployments use `op-chain-ops/script.Host`:

- It runs Solidity deployment scripts in an in-memory EVM.
- It enables Foundry cheatcodes.
- It hooks `vm.broadcast` calls into an `op-deployer` broadcaster.
- It supports live L1 forking through RPC.
- It uses `script.WithCreate2Deployer()` for deterministic deployments.
- Go input/output structs are exposed as EVM precompiles, and ABI mismatches are caught when scripts load.

Core OPCM scripts are loaded from artifacts:

- `DeploySuperchain.s.sol`
- `DeployImplementations.s.sol`
- `DeployOPChain.s.sol`
- `DeployAltDA.s.sol`
- `DeployDisputeGame.s.sol`
- `DeployMIPS.s.sol`
- `DeployAlphabetVM.s.sol`
- `L2Genesis.s.sol`
- read-only helpers such as `ReadImplementationAddresses.s.sol` and `ReadSuperchainDeployment.s.sol`

## Broadcasters

`broadcaster.Broadcaster` has `Hook(script.Broadcast)` and `Broadcast(context.Context)`.

- `KeyedBroadcaster` signs and sends transactions through `txmgr`. It pads gas by `1.2`, clamps to block gas limit, uses one confirmation, and handles calls, creates, and CREATE2 through the deterministic deployer.
- `CalldataBroadcaster` converts broadcasts into `to`, `data`, `value` JSON entries. Its `Broadcast` is a no-op; callers must dump collected calldata.
- `NoopBroadcaster` discards broadcasts.

## Bootstrap

Use bootstrap when creating a new standalone deployment group or deploying a new OPCM/implementation set on an L1.

`bootstrap superchain`:

```bash
op-deployer bootstrap superchain \
  --l1-rpc-url "$L1_RPC_URL" \
  --private-key "$PRIVATE_KEY" \
  --outfile bootstrap_superchain.json \
  --superchain-proxy-admin-owner 0x... \
  --guardian 0x...
```

Requires non-zero Superchain proxy admin owner and guardian. Outputs Superchain proxy admin, SuperchainConfig implementation, and SuperchainConfig proxy addresses. The deployment key should not retain control after deployment.

`bootstrap implementations`:

```bash
op-deployer bootstrap implementations \
  --l1-rpc-url "$L1_RPC_URL" \
  --private-key "$PRIVATE_KEY" \
  --outfile bootstrap_implementations.json \
  --superchain-config-proxy 0x... \
  --superchain-proxy-admin 0x... \
  --l1-proxy-admin-owner 0x... \
  --challenger 0x...
```

Also accepts proof parameter flags, MIPS version, dev feature bitmap, `--use-forge`, and verifier flags. Outputs OPCM and implementation addresses.

Invariants:

- A deployment group should have one SuperchainConfig set for one L1.
- Each OPCM corresponds to a contract version and deployment group.
- For custom chains using a bootstrapped OPCM, set `configType = "standard-overrides"` or `custom`, set `opcmAddress`, and use matching artifact locators.

## Inspect

Subcommands:

- `inspect l1 <chain-id>` - writes all L1 contracts for a chain: Superchain, implementations, and OP Chain contracts.
- `inspect genesis <chain-id>` - renders L2 genesis from `state.json`.
- `inspect rollup <chain-id>` - renders rollup config from `state.json`.
- `inspect deploy-config <chain-id>` - renders the combined deploy config.
- `inspect l2-semvers <chain-id>` - reads L2 predeploy semvers from allocs with a script host.

Common flags:

```bash
op-deployer inspect genesis --workdir .deployer <chain-id> --outfile genesis.json
op-deployer inspect rollup --workdir .deployer <chain-id> --outfile rollup.json
```

`--outfile -` writes stdout.

## Verify

Manual verification:

```bash
op-deployer verify \
  --l1-rpc-url "$L1_RPC_URL" \
  --input-file .deployer/state.json \
  --artifacts-locator "file://$PWD/packages/contracts-bedrock" \
  --verifier etherscan \
  --verifier-api-key "$VERIFIER_API_KEY"
```

Supported `--input-file` formats:

- Bootstrap JSON output with contract name/address pairs.
- Full `state.json`; the verifier extracts Superchain, implementation, and chain contract addresses.

Supported verifiers:

- `etherscan` - requires API key and only supports chain IDs `1` and `11155111` in current helper mapping.
- `blockscout` - default URLs for mainnet and Sepolia; no API key required.
- `custom` - Etherscan v2-compatible custom verifier URL required.

Multiple verifiers are comma-separated, for example `--verifier etherscan,blockscout`.

Forge verification behavior:

- Uses `forge verify-contract`.
- Loads compiler metadata and source mappings from artifacts.
- Adds `--guess-constructor-args` outside local test environments.
- Rechecks API status for already-verified and partially verified contracts.
- Treats some Blockscout partial verification outcomes as expected because Forge may not upgrade partial verification to full verification.

Auto-verification after `apply` or `bootstrap` is best-effort: missing Etherscan API keys skip auto-verification instead of failing deployment, and later failures log warnings.

## Upgrade

The public notice says `op-deployer upgrade` is deprecated for newer L1 contract upgrade workflows. `op-deployer` itself is not deprecated.

Current code registers upgrade commands:

- `v2.0.0`
- `v3.0.0`
- `v4.0.0`
- `v4.1.0`
- `v5.0.0`
- `v6.0.0-rc.2`

Subcommands require:

```bash
op-deployer upgrade v6.0.0-rc.2 \
  --l1-rpc-url "$L1_RPC_URL" \
  --config upgrade.json \
  --override-artifacts-url "file://$PWD/packages/contracts-bedrock" \
  --outfile calldata.json
```

Important: versioned upgrade commands use `CalldataBroadcaster`. They fork/simulate the script and write calldata JSON; they do not sign or broadcast a transaction. The top-level `upgrade` command help still shows inherited `--private-key` and `--deployment-target`, but subcommands do not use them.

Upgrade config shape for current OPCM v2 helpers:

- `prank`
- `opcm`
- `upgradeInput`
- `upgradeInput.disputeGameConfigs`
- `upgradeInput.extraInstructions`

Supported current game type constants include Cannon `0`, Permissioned Cannon `1`, Super Permissioned Cannon `5`, Cannon Kona `8`, Super Cannon Kona `9`, and ZK `10`.

## Manage

The public deprecation notice says `manage` is deprecated and no longer maintained.

Current commands:

- `manage add-game-type-v2` - uses the current OPCMv2 calldata generation path and requires an artifacts locator.
- `manage migrate` - migrates a chain to superproofs through `InteropMigration.s.sol` helpers.

`manage migrate` required inputs include L1 RPC, private key, artifacts locator, L1 proxy admin owner, OPCM implementation, SystemConfig proxy, starting anchor root, bond, game type, and prestate. It rejects retired game type `4` (`SUPER_CANNON`) for both game type and starting respected game type.

Current source builds a `KeyedBroadcaster` in `manage migrate`, but does not call `Broadcast` or dump calldata before returning the script output. Confirm behavior before using it for a production migration.

## Validate

There are two validation paths.

`apply --validate`:

- Runs after successful live apply only.
- Skips non-live targets or missing L1 RPC.
- Uses `op-validator/pkg/service`.
- Supports `--validate auto` or an explicit contract version.
- Builds validator config from `state.json`, chain state, chain roles, absolute prestate, and OPCM standard validator address if available.
- Fails apply if validation returns errors.

`validate auto`:

```bash
op-deployer validate auto \
  --l1-rpc-url "$L1_RPC_URL" \
  --workdir .deployer \
  --fail \
  <optional-chain-id>
```

This simpler command checks selected deployed L1 contract addresses are non-zero and have code: SystemConfig, L1CrossDomainMessenger, L1StandardBridge, OptimismPortal.

## Standard Defaults

Important defaults in `pkg/deployer/standard/standard.go`:

- `CurrentTag = op-contracts/v6.0.0-rc.2`
- L2 gas limit: `60_000_000`
- base fee scalar: `1368`
- blob base fee scalar: `801949`
- withdrawal delay: `302400`
- preimage proposal size: `126000`
- challenge period: `86400`
- proof maturity delay: `604800`
- dispute game finality delay: `302400`
- MIPS version: `8`
- default dispute game type: `1` (permissioned)
- max game depth: `73`
- split depth: `30`
- clock extension: `10800`
- max clock duration: `302400`
- EIP-1559 Canyon denominator: `250`
- EIP-1559 denominator: `50`
- EIP-1559 elasticity: `6`
- default hardfork schedule activates Jovian at genesis.

Standard registry helpers only support mainnet and Sepolia. Unsupported L1 chain IDs need custom deployment paths.

## Custom Features

Custom gas token:

- Intent path: `[chains.customGasToken]`.
- Requires `name` and `symbol`.
- Optional `initialLiquidity`.
- Optional `liquidityControllerOwner`.
- Standard intents reject custom gas tokens.

Minimum base fee:

- Intent field: `minBaseFee`.
- Becomes part of `genesis.FeeMarketConfig`.
- Can later be updated through `SystemConfig.setMinBaseFee(uint64)` by the SystemConfig owner.

DA footprint gas scalar:

- Intent field: `daFootprintGasScalar`.
- Becomes part of `genesis.FeeMarketConfig`.
- Can later be updated through `SystemConfig.setDAFootprintGasScalar(uint16)`.

Alt-DA:

- Intent field: `dangerousAltDAConfig`.
- Only deploys challenge contracts for Keccak commitment mode.

Additional dispute games:

- Intent field: `dangerousAdditionalDisputeGames`.
- Deployer must be L1 proxy admin owner.
- Supports VM types `ALPHABET`, `CANNON`, `CANNON-NEXT`, and `ZK`.
- ZK games require verifier, absolute prestate, max challenge/prove durations, and positive challenger bond.

Interop:

- Intent field: `useInterop`.
- `devFeatureBitmap` in global deploy overrides must match the interop flag, or L2 genesis generation fails.
- Interop dependency set is generated into state.

Development genesis:

- `deployment-target=genesis` supports `l1DevGenesisParams` and per-chain `l2DevGenesisParams` prefunds.
- L1 genesis timestamp defaults to `time.Now()` if not specified.
- Sealed L1 dev genesis stores allocs in `l1StateDump` and stores the state root in `L1DevGenesis`.

## Known Pitfalls

- Public docs may be ahead of or behind current code. In this snapshot, `tag://` locators in docs are stale for current `develop` code.
- Some public examples omit `superchainRoles.Challenger`, but current custom intent validation and implementation deployment require it when deploying a new implementation set from intent.
- Some custom gas token docs show zero EIP-1559 values, but current `ChainIntent.Check` rejects zero EIP-1559 parameters.
- Standard and standard-overrides intent creation requires a supported standard L1 chain ID because it resolves standard OPCM data from registry validation tables.
- Live deployments require the deterministic deployer preinstall on L1.
- `apply` uses the intent's `l1ChainID` as the keyed broadcaster chain ID but separately checks the RPC chain ID during init. Mismatches fail early.
- `state.json` is the resume boundary. Do not delete or hand-edit it casually after partial deployments.
- `create2Salt` is generated once in state. Changing or deleting it changes deterministic addresses.
- `AppliedIntent` is written only after all stages complete. Some inspect commands require a fully applied state.
- `deployment-target=calldata` still needs an L1 RPC because it forks live state to simulate.
- `clean cache` takes `--cache-dir` as a global flag, so pass it before the subcommand: `op-deployer --cache-dir /tmp/cache clean cache`.
- `verify` needs artifacts with a usable `foundry.toml`; pass a concrete artifact locator.
- Running from source no longer requires `artifacts.tzst`; deployment and verification commands require an explicit artifact locator.

## Tests

Useful packages:

- `op-deployer/pkg/deployer/state/...` - intent/state validation and serialization.
- `op-deployer/pkg/deployer/artifacts/...` - locator and extraction behavior.
- `op-deployer/pkg/deployer/broadcaster/...` - calldata and keyed broadcaster behavior.
- `op-deployer/pkg/deployer/bootstrap/...` - bootstrap commands.
- `op-deployer/pkg/deployer/pipeline/...` - individual pipeline stages.
- `op-deployer/pkg/deployer/opcm/...` - script input/output bindings.
- `op-deployer/pkg/deployer/integration_test/...` - end-to-end pipeline and CLI workflows.
- `op-deployer/pkg/deployer/verify/...` - verifier and bundle extraction.
- `op-deployer/pkg/deployer/upgrade/...` and `manage/...` - upgrade and management calldata paths.

Common test commands:

```bash
cd op-deployer
just test ./pkg/deployer/state/...
just test ./pkg/deployer/artifacts/...
just test ./pkg/deployer/integration_test/cli/... -run TestCommandParsing
```

The repo root `justfile` includes op-deployer packages in `TEST_PKGS` and `RPC_TEST_PKGS`.

## Internal Consumers

Other repo areas use `op-deployer` as a library:

- `op-devstack/sysgo` calls `deployer.ApplyPipeline` with `DeploymentTargetGenesis` to build test worlds.
- `op-e2e/config` uses `ApplyPipeline` and `inspect` helpers for alloc/config generation.
- `op-e2e/e2eutils/intentbuilder` builds typed intents and is careful about override value types because JSON merge behavior depends on exact Go types.
- `op-validator` imports `standard` constants and validation data.
- `op-chain-ops/interopgen` imports `opcm` and `manage` helpers and is expected to consolidate with op-deployer over time.
- `op-fetcher` imports `opcm`, `broadcaster`, and `env` for script/fetch workflows.

## File Map

- CLI: `op-deployer/pkg/cli/app.go`
- Init/apply: `op-deployer/pkg/deployer/init.go`, `op-deployer/pkg/deployer/apply.go`
- Flags: `op-deployer/pkg/deployer/flags.go`, `op-deployer/pkg/deployer/flags/names.go`
- Intent/state: `op-deployer/pkg/deployer/state/`
- Pipeline: `op-deployer/pkg/deployer/pipeline/`
- OPCM script bindings: `op-deployer/pkg/deployer/opcm/`
- Script host: `op-deployer/pkg/env/host.go`
- Broadcasters: `op-deployer/pkg/deployer/broadcaster/`
- Artifacts: `op-deployer/pkg/deployer/artifacts/`
- Standard constants: `op-deployer/pkg/deployer/standard/standard.go`
- Bootstrap commands: `op-deployer/pkg/deployer/bootstrap/`
- Inspect commands: `op-deployer/pkg/deployer/inspect/`
- Verify commands: `op-deployer/pkg/deployer/verify/`
- Upgrade commands: `op-deployer/pkg/deployer/upgrade/`
- Deprecated manage commands: `op-deployer/pkg/deployer/manage/`
- Forge wrapper: `op-deployer/pkg/deployer/forge/`
- Public docs source: `docs/public-docs/chain-operators/tools/op-deployer/`
- Create-rollup tutorial: `docs/public-docs/chain-operators/tutorials/create-l2-rollup/op-deployer-setup.mdx`
- Upgrade deprecation notice: `docs/public-docs/notices/op-deployer-upgrade-deprecation.mdx`

## Suggested Follow-Up Doc Fixes

The public OP Deployer docs should be reviewed for current `develop` behavior:

- Replace or qualify `tag://` locator examples, since current locator parsing rejects them.
- Fix custom gas token examples that set EIP-1559 fields to zero.
- Clarify that upgrade subcommands generate calldata rather than broadcasting transactions.
- Clarify `manage` deprecation and the current behavior of `manage migrate`.
