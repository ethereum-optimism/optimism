// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @notice Features is a library that stores feature name constants. Can be used alongside the
///         feature flagging functionality in the SystemConfig contract to selectively enable or
///         disable customizable features of the OP Stack.
///
/// @dev IMPORTANT: When adding a new feature to this library, its compatibility with ALL existing
///      features MUST be carefully evaluated. If the new feature is incompatible with any existing
///      feature, appropriate validation logic must be added to the validateCompatibility function
///      or a new validation mechanism must be implemented.
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

    /// @notice The REVENUE_SHARING feature determines if the system is configured to use
    ///         revenue sharing with fee vaults routing to the FeeSplitter predeploy.
    ///         This is an L2-only feature configured at genesis time.
    bytes32 internal constant REVENUE_SHARING = "REVENUE_SHARING";

    /// @notice Error thrown when custom gas token and revenue sharing are both enabled.
    error Features_CustomGasTokenAndRevenueShareIncompatible();

    /// @notice Validates that custom gas token and revenue sharing are not both enabled.
    ///         These features are mutually exclusive.
    ///
    /// @dev This validation approach is intentionally simple and not designed to scale
    ///      to a large number of features or complex compatibility rules. As new features are
    ///      added to the system, they should be evaluated for compatibility with all existing
    ///      features. If additional incompatibilities are discovered or the number of features
    ///      grows significantly, a more robust and extensible validation mechanism should be
    ///      implemented.
    /// @param _useCustomGasToken Whether custom gas token is enabled.
    /// @param _useRevenueShare Whether revenue sharing is enabled.
    function validateCompatibility(bool _useCustomGasToken, bool _useRevenueShare) internal pure {
        if (_useCustomGasToken && _useRevenueShare) {
            revert Features_CustomGasTokenAndRevenueShareIncompatible();
        }
    }
}
