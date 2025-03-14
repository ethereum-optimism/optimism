// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @notice The struct for a pointer to a message payload in a remote (or local) chain.
/// @custom:field origin The origin address of the message.
/// @custom:field blockNumber The block number of the message.
/// @custom:field logIndex The log index of the message.
/// @custom:field timestamp The timestamp of the message.
/// @custom:field chainId The origin chain ID of the message.
struct Identifier {
    address origin;
    uint256 blockNumber;
    uint256 logIndex;
    uint256 timestamp;
    uint256 chainId;
}

interface ICrossL2Inbox {
    /// @notice Thrown when trying to execute a cross chain message on a deposit transaction.
    error NoExecutingDeposits();

    /// @notice Thrown when trying to validate a cross chain message with an identifier checksum that is
    ///         invalid or was not provided in the transaction's access list to set the slot as warm.
    error NotInAccessList();

    /// @notice Thrown when trying to validate a cross chain message with a block number that is
    ///         greater than 2^64.
    error BlockNumberTooHigh();

    /// @notice Thrown when trying to validate a cross chain message with a timestamp that is
    ///         greater than 2^64.
    error TimestampTooHigh();

    /// @notice Thrown when trying to validate a cross chain message with a log index that is
    ///         greater than 2^32.
    error LogIndexTooHigh();

    /// @notice Emitted when a message is being executed.
    event ExecutingMessage(bytes32 indexed msgHash, Identifier id);

    /// @notice Returns the semantic version of the contract.
    /// @return version_ The semantic version.
    function version() external view returns (string memory version_);

    /// @notice Validates a message by checking that the identifier checksum slot is warm.
    /// @dev    To process the message, the tx must include the checksum composed by the message's
    ///         identifier and msgHash in the access list.
    /// @param _id The identifier of the message.
    /// @param _msgHash The hash of the message.
    function validateMessage(Identifier calldata _id, bytes32 _msgHash) external;

    /// @notice Calculates the checksum of an identifier and message hash.
    /// @param _id The identifier of the message.
    /// @param _msgHash The hash of the message.
    /// @return checksum_ The checksum of the identifier and message hash.
    function calculateChecksum(Identifier memory _id, bytes32 _msgHash) external pure returns (bytes32 checksum_);
}
