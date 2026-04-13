// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @notice Features is a library that stores feature name constants. Can be used alongside the
///         feature flagging functionality in the SystemConfig contract to selectively enable or
///         disable customizable features of the OP Stack.
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

    /// @notice The INTEROP_BASE feature determines if the base interop contracts are deployed.
    ///         This is set on all chains at interop activation, regardless of dependency set size.
    bytes32 internal constant INTEROP_BASE = "INTEROP_BASE";

    /// @notice The INTEROP_CROSS_L2_INBOX feature determines if the CrossL2Inbox is activated.
    ///         This is only set when the chain is in a dependency set with at least two chains.
    bytes32 internal constant INTEROP_CROSS_L2_INBOX = "INTEROP_CROSS_L2_INBOX";
}
