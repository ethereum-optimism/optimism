// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

interface IRandom {
    function revealAndCommit(bytes32, bytes32, address) external;
    function randomnessBlockRetentionWindow() external view returns (uint256);
    function random() external view returns (bytes32);
    function getBlockRandomness(uint256) external view returns (bytes32);
}
