// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @notice Features is a library that stores feature name constants. Can be used alongside the
///         feature flagging functionality in the SystemConfig contract to selectively enable or
///         disable customizable features of the OP Stack.
/// @dev ADDING A NEW SYSTEM FEATURE:
///      System feature values are read from SystemConfig on-chain.
///      Add the new feature name here first.
///
///      If the feature should be toggleable through contract-test env vars and the CI feature
///      matrix, update every applicable item below.
///
///        1. Add the env var reader in `packages/contracts-bedrock/scripts/libraries/Config.sol`.
///        2. Consume the reader in deployment and test setup code. Initialize SystemConfig state.
///           Search for existing `Config.sysFeature*` consumers. For `CUSTOM_GAS_TOKEN`, this includes
///           packages/contracts-bedrock/scripts/deploy/Deploy.s.sol and
///           packages/contracts-bedrock/test/setup/CommonTest.sol.
///        3. Add the feature name in `packages/contracts-bedrock/test/setup/FeatureFlags.sol`.
///           System features are not added to resolveFeaturesFromEnv(). Their value comes from
///           SystemConfig.
///        4. Register the feature as a system feature in `.circleci/continue/main.yml` under
///           `setup-features.system_features.default`. A missing entry exports DEV_FEATURE__<NAME>
///           instead of SYS_FEATURE__<NAME>.
///        5. Add CI coverage in `.circleci/continue/main.yml` under the `&features_matrix` anchor.
///           Add combination rows when this feature interacts with another feature. Document
///           intentionally unsupported combinations near the matrix.
///
///      System features have no Go mirror. For the parallel DEV feature checklist, see
///      packages/contracts-bedrock/src/libraries/DevFeatures.sol.
library Features {
    /// @notice The ETH_LOCKBOX feature determines if the system is configured to use the
    ///         ETHLockbox contract in the OptimismPortal. When the ETH_LOCKBOX feature is active
    ///         and the ETHLockbox contract has been configured, the OptimismPortal will use the
    ///         ETHLockbox to store ETH instead of storing ETH directly in the portal itself.
    bytes32 internal constant ETH_LOCKBOX = "ETH_LOCKBOX";

    /// @notice The CUSTOM_GAS_TOKEN feature determines if the system is configured to use a custom
    ///         gas token in the OptimismPortal. When the CUSTOM_GAS_TOKEN feature is active, the
    ///         deposits and withdrawals of native ETH are disabled.
    bytes32 internal constant CUSTOM_GAS_TOKEN = "CUSTOM_GAS_TOKEN";

    /// @notice The INTEROP feature determines if the system is configured to use interop.
    bytes32 internal constant INTEROP = "INTEROP";
}
