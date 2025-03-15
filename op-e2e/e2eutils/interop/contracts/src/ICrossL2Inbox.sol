// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @notice Identifier of a cross chain message.
struct Identifier {
    address origin;
    uint256 blockNumber;
    uint256 logIndex;
    uint256 timestamp;
    uint256 chainId;
}

interface ICrossL2Inbox {
    error NoExecutingDeposits();
    error NotInAccessList();
    error BlockNumberTooHigh();
    error TimestampTooHigh();
    error LogIndexTooHigh();

    event ExecutingMessage(bytes32 indexed msgHash, Identifier id);

    function version() external view returns (string memory);

    function validateMessage(Identifier calldata _id, bytes32 _msgHash) external;
}
