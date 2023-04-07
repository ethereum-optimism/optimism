// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import { IDisputeGame } from "src/interfaces/IDisputeGame.sol";

/// @title IValididtyDisputeGame
/// @notice The interface for a validity proof backed dispute game.
interface IValididtyDisputeGame is IDisputeGame {
    /// @notice Proves the root claim
    /// @dev Underneath the hood, the separate implementations will unpack the `data` differently
    /// due to the different proof verification algorithms for SNARKs, PLONKs, etc.
    function prove(bytes calldata input) external;
}
