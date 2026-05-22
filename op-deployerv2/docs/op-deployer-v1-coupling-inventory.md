# op-deployer v1 Coupling Inventory

This document lists the hard-coded coupling points that bind current
`op-deployer` behavior to the contracts release it was built with.

The recurring pattern is that contract release details are compiled into Go
code: release tags, script names, script input/output structs, OPCM input
structs, artifact names, standard constants, and legacy `intent.toml` /
`state.json` schemas.

## Upgrade command coupling

The upgrade CLI is a catalog of known releases:

- `op-deployer/pkg/deployer/upgrade/flags.go` imports every versioned upgrader:
  `embedded`, `v2_0_0`, `v3_0_0`, `v4_0_0`, `v4_1_0`, `v5_0_0`, and `v6_0_0`.
- The same file registers version-named subcommands: `v2.0.0`, `v3.0.0`,
  `v4.0.0`, `v4.1.0`, `v5.0.0`, `v6.0.0-rc.2`, and `embedded`.
- Adding a new named contracts release requires a new Go package import, a new
  CLI subcommand, and a new deployer release.

The upgrade input schemas are hardcoded:

- `upgrade/v2_0_0/upgrade.go` defines `UpgradeOPChainInput` and
  `OPChainConfig` manually, including the exact tuple encoder:
  `(address systemConfigProxy,address proxyAdmin,bytes32 absolutePrestate)[]`.
- `upgrade/v6_0_0/upgrade.go` defines a different `OPChainConfig` manually,
  with `systemConfigProxy`, `cannonPrestate`, and `cannonKonaPrestate`.
- `upgrade/current/upgrade.go` defines the current OPCMv2 `UpgradeInputV2`
  manually, including `systemConfig`, `disputeGameConfigs`, and
  `extraInstructions`.
- `upgrade/current/upgrade.go` also hardcodes game type IDs and specific game
  arg encoders for fault, permissioned, and ZK games.
- `upgrade/current/upgrade_superchainconfig.go` and
  `upgrade/v6_0_0/upgrade_superchainconfig.go` define
  `UpgradeSuperchainConfigInput` manually.

The upgrade script names are hardcoded:

- `UpgradeOPChain.s.sol`
- `UpgradeSuperchainConfig.s.sol`

The old upgrade packages also hardcode artifact locators:

- `v2_0_0`, `v3_0_0`, `v4_0_0`, `v4_1_0`, and `v5_0_0` each return a specific
  GCS artifact hash.
- `embedded` and `v6_0_0` return the embedded artifact locator.

## OPCM/script binding coupling

`op-deployer/pkg/deployer/opcm` is effectively a manually maintained generated
SDK for contracts-bedrock scripts.

These files define Go structs that must mirror Solidity script structs:

- `opcm/superchain.go`: `DeploySuperchainInput`,
  `DeploySuperchainOutput`.
- `opcm/implementations.go`: `DeployImplementationsInput`,
  `DeployImplementationsOutput`.
- `opcm/opchain.go`: `DeployOPChainInput`, `DeployOPChainOutput`,
  `ReadImplementationAddressesInput`, `ReadImplementationAddressesOutput`.
- `opcm/l2genesis.go`: `L2GenesisInput`.
- `opcm/dispute_game.go`: `DeployDisputeGameInput`,
  `DeployDisputeGameOutput`.
- `opcm/mips.go`: `DeployMIPSInput`, `DeployMIPSOutput`.
- `opcm/alphabet.go`: `DeployAlphabetVMInput`, `DeployAlphabetVMOutput`.
- `opcm/alt_da.go`: `DeployAltDAInput`, `DeployAltDAOutput`.
- `opcm/read_superchain_deployment.go`: `ReadSuperchainDeploymentInput`,
  `ReadSuperchainDeploymentOutput`.
- `opcm/dispute_game_factory.go`: `SetDisputeGameImplInput`.

Those files also hardcode script filenames, contract names, Forge script paths,
and `runWithBytes(bytes)` conventions, for example:

- `DeploySuperchain.s.sol:DeploySuperchain`
- `DeployImplementations.s.sol:DeployImplementations`
- `DeployOPChain.s.sol:DeployOPChain`
- `ReadImplementationAddresses.s.sol:ReadImplementationAddresses`
- `L2Genesis.s.sol:L2Genesis`
- `DeployDisputeGame.s.sol:DeployDisputeGame`
- `DeployMIPS.s.sol:DeployMIPS`
- `DeployAltDA.s.sol:DeployAltDA`
- `DeployAlphabetVM.s.sol:DeployAlphabetVM`
- `ReadSuperchainDeployment.s.sol:ReadSuperchainDeployment`
- `SetDisputeGameImpl.s.sol:SetDisputeGameImpl`
- `SetPreinstalls.s.sol:SetPreinstalls`

`opcm/scripts.go` compounds the coupling by collecting a fixed set of scripts
and failing if any are missing or their ABI no longer matches the Go types.

## Apply pipeline coupling

`op-deployer/pkg/deployer/apply.go` is not a generic deploy runner. It encodes
a fixed deployment pipeline:

- read `intent.toml`
- read `state.json`
- download L1 and L2 artifacts from intent locators
- create a script host
- load the fixed `opcm.Scripts`
- run fixed stages:
  - `init`
  - `deploy-superchain`
  - `deploy-implementations`
  - `deploy-opchain`
  - `deploy-alt-da`
  - `deploy-additional-dispute-games`
  - `generate-l2-genesis`
  - genesis-only prefund/preinstall/seal stages
  - `set-start-block`
  - `generate-interop-depset`
  - `deploy-pre-state`

Each pipeline stage maps the legacy intent/state model into hardcoded script
input structs:

- `pipeline/superchain.go` builds `opcm.DeploySuperchainInput`.
- `pipeline/implementations.go` builds `opcm.DeployImplementationsInput` and
  maps `DeployImplementationsOutput` into `addresses.ImplementationsContracts`.
- `pipeline/opchain.go` builds `opcm.DeployOPChainInput`, reads implementation
  addresses with `ReadImplementationAddresses`, and maps outputs into
  `addresses.OpChainContracts`.
- `pipeline/alt_da.go` builds `opcm.DeployAltDAInput`.
- `pipeline/dispute_games.go` selects fixed VM paths and builds
  `DeployAlphabetVMInput`, `DeployMIPSInput`, `DeployDisputeGameInput`, and
  `SetDisputeGameImplInput`.
- `pipeline/l2genesis.go` builds `opcm.L2GenesisInput`.
- `pipeline/preinstall_l1_dev_genesis.go` calls the fixed
  `SetPreinstalls.s.sol` script.

Any script input/output change can require Go changes in the pipeline, even if
the user workflow did not change.

## Legacy intent/state coupling

The legacy file formats are compiled into Go structs:

- `state/intent.go` defines `Intent`, `SuperchainProofParams`, and
  `L1DevGenesisParams`.
- `state/chain_intent.go` defines `ChainIntent`, `ChainProofParams`,
  `AdditionalDisputeGame`, `VMType`, `ZKDisputeGameParams`, and
  `CustomGasToken`.
- `state/state.go` defines `State`, `ChainState`, and
  `AdditionalDisputeGameState`.

The file names are hardcoded:

- `pipeline/env.go` reads `intent.toml`.
- `pipeline/env.go` reads and writes `state.json`.
- `init.go` writes both files.

The state version is hardcoded:

- `init.go` creates `State{Version: 1}`.
- `pipeline/init.go` only accepts state version `1`.

The legacy schemas contain specific contract names and current pipeline
assumptions, including `SuperchainDeployment`,
`ImplementationsDeployment`, `OpChainContracts`, L2 allocs, start blocks,
prestate manifests, interop dependency sets, and deployment calldata.

## Standard-release coupling

`op-deployer/pkg/deployer/standard/standard.go` centralizes release-specific
defaults and standard-chain assumptions:

- protocol constants such as gas limit, fee scalars, dispute parameters, MIPS
  version, dispute game type, and EIP-1559 values
- contract release tags from `op-contracts/v1.6.0` through
  `op-contracts/v6.0.0-rc.2`
- `CurrentTag`, currently set to `ContractsV600Tag`
- standard mainnet/sepolia role and OPCM address lookup
- hardcoded Superchain ProxyAdmin addresses
- default hardfork schedule

These constants are referenced by init, intent validation, bootstrap defaults,
pipeline input construction, L2 genesis generation, validation, tests, and
manual justfile flows.

## Artifact coupling

Before the embed removal, artifact handling was release-coupled in several
ways:

- `artifacts/embedded.go` embedded `pkg/deployer/artifacts/forge-artifacts`.
- The embedded artifact filename is fixed as `artifacts.tzst`.
- `artifacts/locator.go` defines `embedded` and uses it as both default L1 and
  L2 contracts locator.
- `flags.go` defaults `--artifacts-locator` to `embedded`.
- `artifacts/locator.go` builds old release artifact URLs under
  `https://storage.googleapis.com/oplabs-contract-artifacts/artifacts-v1-...`.
- `artifacts/download.go` assumes artifact bundles expose a `forge-artifacts`
  directory.
- `op-deployer/justfile` builds contracts from
  `../packages/contracts-bedrock` and copies them into the embedded artifact
  location.

This makes the binary carry a default contracts release and artifact layout.

## Bootstrap command coupling

The bootstrap commands expose current script input fields as fixed CLI flags:

- `bootstrap/flags.go` defines flags for current
  `DeployImplementationsInput`, including withdrawal delay, proposal size,
  challenge period, dispute game finality delay, MIPS version, fault game
  depths, superchain config proxy, proxy admins, and challenger.
- `bootstrap/implementations.go` maps those fixed flags to
  `opcm.DeployImplementationsInput`.
- `bootstrap/superchain.go` maps fixed flags to `opcm.DeploySuperchainInput`.

If a bootstrap script adds or changes an input, the CLI flags and Go config
structs need to change.

## Manage command coupling

`manage add-game-type-v2` is coupled directly to the embedded OPCMv2 upgrade
path:

- `manage/add_game_type.go` aliases the command to
  `upgrade.UpgradeCLI(current.DefaultUpgrader)`.

`manage migrate` hardcodes the migrator input schema:

- `manage/migrate.go` defines `InteropMigrationInput`, `MigrateInputV2`,
  `DisputeGameConfig`, and `Proposal`.
- It hardcodes the `IOPContractsManagerMigrator.MigrateInput` tuple encoder.
- It builds `gameArgs` manually from `bytes32 absolutePrestate`.
- It runs `InteropMigration.s.sol`.
- `manage/flags.go` hardcodes retired and default game type IDs.

## Verify, validate, and inspect coupling

Verification assumes current state and artifact names:

- `verify/state_bundle.go` parses `state.State` directly and reflects over
  `SuperchainDeployment`, `ImplementationsDeployment`, and `OpChainContracts`.
- `verify/artifacts.go` maps state field names to specific artifact paths like
  `OPContractsManagerV2.sol/OPContractsManagerV2.json`,
  `OptimismPortal2.sol/OptimismPortal2.json`, `MIPS64.sol/MIPS64.json`, and
  `Proxy.sol/Proxy.json`.

Validation assumes current state and current standard release:

- `validate/helpers.go` defaults `auto` validation to `standard.CurrentTag`
  unless it sees a legacy tag locator.
- It falls back to `standard.DisputeAbsolutePrestate`.
- It queries `opcmStandardValidator` from the OPCM using a hardcoded method
  name.
- `validate/validate.go` checks a fixed set of state contracts:
  `SystemConfig`, `L1CrossDomainMessenger`, `L1StandardBridge`, and
  `OptimismPortal`.

Inspect commands are state-schema coupled:

- `inspect/l1.go` emits `addresses.L1Contracts` from current state fields.
- `inspect/deploy_config.go`, `inspect/genesis.go`, and `inspect/rollup.go`
  rebuild deploy config/genesis/rollup via `state.CombineDeployConfig`.
- `state/deploy_config.go` hardcodes many deploy-config defaults and current
  genesis assumptions.
- `inspect/semvers.go` hardcodes the set of L2 predeploy semver addresses to
  inspect.

## Tests that lock in the coupling

Many tests are valuable but currently lock in release-specific assumptions.
Examples:

- upgrade CLI tests enumerate the same standard release tags and OPCM addresses.
- bootstrap and apply integration tests use `standard.*` defaults and embedded
  artifacts.
- script wrapper tests load current contracts-bedrock artifacts and assert that
  the hardcoded Go structs match current script ABIs.
- verify tests assert current artifact path exceptions.

These tests should not disappear, but in a source-driven design they should
become generator/adaptor freshness checks against a selected contracts source
instead of tests of core deployer release knowledge.
