// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title ICrossL2Inbox
/// @notice Interface for the CrossL2Inbox contract.
interface ICrossL2Inbox {
    /// @notice The struct for a pointer to a message payload in a remote (or local) chain.
    struct Identifier {
        address origin;
        uint256 blockNumber;
        uint256 logIndex;
        uint256 timestamp;
        uint256 chainId;
    }

    /// @notice Returns the origin address of the Identifier.
    /// @return _origin The origin address of the Identifier.
    function origin() external view returns (address _origin);

    /// @notice Returns the block number of the Identifier.
    /// @return _blockNumber The block number of the Identifier.
    function blockNumber() external view returns (uint256 _blockNumber);

    /// @notice Returns the log index of the Identifier.
    /// @return _logIndex The log index of the Identifier.
    function logIndex() external view returns (uint256 _logIndex);

    /// @notice Returns the timestamp of the Identifier.
    /// @return _timestamp The timestamp of the Identifier.
    function timestamp() external view returns (uint256 _timestamp);

    /// @notice Returns the chain ID of the Identifier.
    /// @return _chainId The chain ID of the Identifier.
    function chainId() external view returns (uint256 _chainId);

    /// @notice Executes a cross chain message on the destination chain.
    /// @param _id An Identifier pointing to the initiating message.
    /// @param _target Account that is called with _msg.
    /// @param _msg The message payload, matching the initiating message.
    function executeMessage(
        ICrossL2Inbox.Identifier calldata _id,
        address _target,
        bytes calldata _msg
    )
        external
        payable;
}
