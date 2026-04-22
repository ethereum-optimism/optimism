# Pre IR Analysis

The aim of this document is to highlight key areas of risk and focus as well as any details about the project that could help guide reviewers. The idea is for reviewers to read this and know what areas they should triple check during the review.

### Project Details

Please complete this table:

| Project Name  | L2ContractsManager (L2CM)                                                                     |
| ------------- | --------------------------------------------------------------------------------------------- |
| Repository    | https://github.com/ethereum-optimism/optimism                                                 |
| Frozen Commit | https://github.com/ethereum-optimism/optimism/commit/b31687d863b5956c5aad2645b6ff1de5c82caae4 |

### Scope

**Core: L2**

- `packages/contracts-bedrock/src/L2/ConditionalDeployer.sol`: **New Contract** that idempotently deploys implementations via the Arachnid Deterministic Deployment Proxy. The **`deploy`** function first calculates the expected deployment address. If code already exists at that address, it returns the address. Otherwise, it forwards its parameters to the Deterministic Deployment contract.
- `packages/contracts-bedrock/src/L2/L2ContractsManager.sol`: **New contract** per-upgrade contract deployed fresh each fork that orchestrates predeploy upgrades via `DELEGATECALL` from `L2ProxyAdmin`, it reads network-specific config and reinitializes each predeploy atomically.
- `packages/contracts-bedrock/src/L2/L2ProxyAdmin.sol`: **New contract** new predeploy extending `ProxyAdmin` with an additional function `upgradePredeploys(address l2cm)` callable only by the `DEPOSITOR_ACCOUNT`.
- `packages/contracts-bedrock/src/L2/OptimismMintableERC721Factory.sol`: Converted from immutable constructor to `Initializable` proxy-upgradeable pattern; `BRIDGE`/`REMOTE_CHAIN_ID` become mutable storage; `isOptimismMintableERC721` mapping extracted to `OptimismMintableERC721FactoryLegacyMapping` base; added`_assertOnlyProxyAdminOrProxyAdminOwner()` in `initialize()`.

**Core: Libraries**

- `packages/contracts-bedrock/src/libraries/L2ContractsManagerTypes.sol`: **New library** defines `Implementations`, `FullConfig`, `MintableERC721FactoryConfig`, and related structs used by `L2ContractsManager`.
- `packages/contracts-bedrock/src/libraries/L2ContractsManagerUtils.sol`: **New library** config-reading utilities for `L2ContractsManager`; includes try/catch fallback in `readFeeVaultConfig`for pre-upgrade chains where `WITHDRAWAL_NETWORK()` is unavailable.
- `packages/contracts-bedrock/src/libraries/Predeploys.sol`: Added `CONDITIONAL_DEPLOYER` and `L2_DEV_FEATURE_FLAGS`; added `isUpgradeable()` and `getUpgradeablePredeploys()` helpers.
- `packages/contracts-bedrock/src/libraries/NetworkUpgradeTxns.sol`: **New library** contains the definition of the **`NetworkUpgradeTxn`** struct and functions to write and read them to/from JSON files.
- `packages/contracts-bedrock/scripts/libraries/UpgradeUtils.sol`: **New library** defining per-contract gas limits and configuration structs used by `GenerateNUTBundle.s.sol`.

**Core: Scripts**

- `packages/contracts-bedrock/scripts/deploy/Deploy.s.sol`: Removed unused `_isInterop` parameter from `deployImplementations()`.
- `packages/contracts-bedrock/scripts/L2Genesis.s.sol`: Added `ConditionalDeployer` and `L2DevFeatureFlags` to genesis; replaced`deployCrossL2Inbox` flag with `devFeatureBitmap` and `useInterop` flags. They must both match in order to enable the feature.
- `packages/contracts-bedrock/scripts/upgrade/ExecuteNUTBundle.s.sol`: **New script** reads a stored NUT bundle JSON and executes each transaction using `vm.prank` with the correct sender and gas limit.
- `packages/contracts-bedrock/scripts/upgrade/GenerateNUTBundle.s.sol`: **New script** deterministically generates the NUT bundle JSON from bytecode and CREATE2 addresses.

**L2DevFeatureFlags**

This feature aims to be the equivalent of the one we already have in L1. We decided to share the same flags namespace between L1 and L2 to avoid misconfigurations caused by having the same feature represented by two different flags. It includes a new contract, `packages/contracts-bedrock/src/L2/L2DevFeatureFlags.sol`, which is a predeploy that stores a per-chain dev feature bitmap, settable only by the `DEPOSITOR_ACCOUNT`. It also includes checks for `L2CM` and `OPTIMISM_PORTAL_INTEROP` in `L2Genesis.s.sol` and `Predeploys.sol`. `packages/contracts-bedrock/src/libraries/DevFeatures.sol` adds an `L2CM` feature flag constant.

- **NOTE:** There exists pending work to be performed in the `op-node`, outside of this scope, in order to inject dev flags bitmap on existing chains.

### Documentation

**Core:**

- Design doc: .claude/docs/l2-contract-upgrades.md
- Execution specs: .claude/docs/l2-upgrades-1-execution.md
- Contracts specs: .claude/docs/l2-upgrades-2-contracts.md

### Key Areas of Risk

**FMA:** .claude/docs/fma-l2cm.md

Extra areas of risk

1.  **Existing chain config is unconditionally re-applied to new implementations**

    `L2ContractsManager` does not accept new configuration as input. Its `upgrade()` function takes no parameters, and `_loadFullConfig()` reads all initialization values exclusively from the current on-chain state of existing predeploys. These values are then passed to the new implementations via `initialize()`.

    If the existing chain config is stale, misconfigured, or incompatible with new implementation logic (e.g., a new implementation introduces stricter validation or a different interpretation of a config field), the upgrade will either revert or silently apply incorrect configuration to the upgraded contracts.

    There is no mechanism to override or supplement the gathered config at upgrade time. Notably, for new predeploys being introduced by an upgrade (which have no prior on-chain state to read from), an external mechanism must supply the initial configuration, **this is currently out of scope for `L2ContractsManager` and must be handled separately before or alongside the upgrade transaction.**

2.  **`INITIALIZABLE_SLOT_OZ_V4 = bytes32(0)` — clearing slot 0**
    The clear-and-reinitialize pattern uses `StorageSetter` to reset the initializable flag at `INITIALIZABLE_SLOT_OZ_V4 = bytes32(0)` before calling `initialize()`. Slot 0 is also the first storage slot of the proxy implementation layout. For most predeploys this offset is `0`, meaning slot 0 of the implementation's storage layout is zeroed. Any contract that stores meaningful data at slot 0 (other than the OZ v4 initializable flag) would have that data silently cleared.

        The `CrossDomainMessenger` case passes `offset=20` to account for `CrossDomainMessengerLegacySpacer0`, which is the correct mitigation — but this offset must be verified to be correct for every initializable predeploy, as a wrong offset will either clear wrong data or miss the initializable flag.

### Common Questions

1. Will the project be deployed in multiple chains?

   Yes, in every OP-stack chain.

2. How many funds will the contract manage, a rough estimate works. If no idea, leave this empty.
   - `L2ContractsManager`: 0 funds.
   - `L2ProxyAdmin`: 0 funds.

     Note: `upgradePredeploys` (the functionality added by this project) is non-payable.

   - `ConditionalDeployer`: 0 funds.
   - The project involves upgrading other Predeploys that potentially hold a large amount of funds i.e. Fee Vaults.

3. Is the protocol upgradable? If so, how will the upgrade process work? Who will have access to execute the upgrade, a multisig?
   - It is upgradable. The upgrade process is done through `L2ProxyAdmin`'s new function `upgradePredeploys()`, which in turn is only callable by the `DEPOSITOR_ACCOUNT`.
4. Are there any centralized authorities like owners? What extent of permissions of the contracts they meant to have?
   - `L2ProxyAdmin`

     The `DEPOSITOR_ACCOUNT` is the authorized caller for `upgradePredeploys`.

   - NUT bundle:

     All the transactions (except for the individual upgrades) from the bundle are executed by the `DEPOSITOR_ACCOUNT`.

     Individual upgrades use `address(0)` to call `IProxy.upgradeTo(address)`.

5. Are there assumptions being made about _anything_ involving the code or the project in general? Be thorough. These are important for the reviewers to know.
   - All predeploy proxies follow the ERC-1967 pattern and accept `upgradeTo/upgradeToAndCall` from the `L2ProxyAdmin`
   - The `StorageSetter` offset is correct for each predeploy
   - We assume the Arachnid Deterministic Deployment Proxy exists at `0x4e59b44847b379578588920cA78FbF26c0B4956C` \*\*\*\*on every chain that is target for the L2CM project

### Extra information

[PR #19564](https://github.com/ethereum-optimism/optimism/pull/19564) moves `ProxyAdminOwnedBase` from `src/L1/` to `src/universal/` also four L2 contracts received a new `_assertOnlyProxyAdminOrProxyAdminOwner()` guard inside `initialize()`:

- `src/L2/FeeVault.sol`
- `src/L2/FeeSplitter.sol`
- `src/L2/LiquidityController.sol`
- `src/L2/OptimismMintableERC721Factory.sol`

These contracts can now be initialized only by the ProxyAdmin or its owner, which is important for the re-initialization pattern the `L2ContractsManager` follows.

**L2 Forked Test Framework**

> This section is listed here for completeness but it is **out of scope**

This aims to replicate a similar setup to L1 forked tests, executing against a configurable network and block number. The bulk of the work is done in `packages/contracts-bedrock/test/setup/CommonTest.sol`, `packages/contracts-bedrock/test/setup/FeatureFlags.sol`, `packages/contracts-bedrock/scripts/libraries/Config.sol`, `packages/contracts-bedrock/test/setup/ForkL2Live.s.sol`, and `packages/contracts-bedrock/test/setup/Setup.sol`, wiring up existing tests and adding some modifiers that can be used to skip tests in an L2 forked environment.

**L2 Fork Test**

Path: `packages/contracts-bedrock/test/L2/fork/L2ForkUpgrade.t.sol`

An end-to-end fork test suite that executes the full L2CM upgrade bundle against a live forked L2 network. The suite is split into five test contracts, each targeting a distinct correctness property:

- **`L2ForkUpgrade_Versions_Test`**: Asserts that every predeploy reports a newer version after the upgrade than it did before.
- **`L2ForkUpgrade_Initialization_Test`**: Asserts that all chain-specific configuration (bridge counterpart addresses, fee vault recipients and thresholds, factory settings, proxy admin ownership) is unchanged after the upgrade, and that every contract is fully initialized with no re-initialization in progress.
- **`L2ForkUpgrade_Implementations_Test`**: Asserts that each predeploy proxy now points to the expected new implementation and that the implementation has deployed code.
- **`L2ForkUpgrade_Events_Test`**: Asserts that each predeploy emitted an upgrade event pointing to the correct new implementation during the upgrade.
- **`L2ForkUpgrade_GasProfile_Test`**: Not a pass/fail correctness test — executes each upgrade transaction and prints a gas report, flagging any transaction whose gas usage is poorly matched to its configured limit.
