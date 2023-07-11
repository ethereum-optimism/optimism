// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @title IPreimageOracle
/// @notice Interface for the `PreimageOracle` contract.
interface IPreimageOracle {
    function readPreimage(bytes32 key, uint256 offset) external view returns (bytes32 dat, uint256 datLen);

    function cheat(
        uint256 partOffset,
        bytes32 key,
        bytes32 part,
        uint256 size
    ) external;

    function loadKeccak256PreimagePart(uint256 partOffset, bytes calldata preimage) external;
}
