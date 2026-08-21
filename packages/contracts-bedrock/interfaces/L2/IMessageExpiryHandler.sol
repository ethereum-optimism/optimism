// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IMessageExpiryHandler
/// @notice Interface that applications implement to be notified by the MessageExpiryRelay when a
///         message they recorded has verifiably expired undelivered on its destination chain.
interface IMessageExpiryHandler {
    /// @notice Called by the MessageExpiryRelay exactly once per expired recorded message.
    ///         Implementations MUST restrict the caller to the MessageExpiryRelay predeploy.
    /// @param _msgHash         Hash of the expired message, as returned by `recordSentMessage`.
    /// @param _attestorChainId Chain ID of the destination chain that attested non-delivery.
    /// @param _attestedAt      Timestamp of the attestation on the destination chain.
    function onMessageExpired(bytes32 _msgHash, uint256 _attestorChainId, uint256 _attestedAt) external;
}
