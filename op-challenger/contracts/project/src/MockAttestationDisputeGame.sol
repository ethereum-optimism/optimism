// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import { Claim } from "./Types.sol";

import { IDisputeGame } from "./IDisputeGame.sol";

/// @title MockAttestationDisputeGame
/// @dev Tests the `op-challenger` on a local devnet.
contract MockAttestationDisputeGame is IDisputeGame {
    Claim public immutable ROOT_CLAIM;
    uint256 public immutable L2_BLOCK_NUMBER;
    mapping(address => bool) public challenges;

    constructor(Claim _rootClaim, uint256 l2BlockNumber, address _creator) {
        ROOT_CLAIM = _rootClaim;
        L2_BLOCK_NUMBER = l2BlockNumber;
        challenges[_creator] = true;
    }

    function challenge(bytes calldata _signature) external {
        challenges[msg.sender] = true;
        _signature;
    }
}