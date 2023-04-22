// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/// @title IDisputeGame
/// @notice An interface for the dispute game contract.
interface IDisputeGame {
    function challenge(bytes calldata _signature) external;
}