// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../libraries/DisputeTypes.sol";

import { IDisputeGame } from "./IDisputeGame.sol";

/**
 * @title IFaultDisputeGame
 * @notice The interface for a fault proof backed dispute game.
 */
interface IFaultDisputeGame is IDisputeGame {
    /**
     * @notice The `ClaimData` struct represents the data associated with a Claim.
     * @dev TODO: Pack `Clock` and `Position` into the same slot. Should require 4 64 bit arms.
     * @dev TODO: Add bond ID information.
     */
    struct ClaimData {
        uint32 parentIndex;
        uint32 rc;
        bool countered;
        Claim claim;
        Position position;
        Clock clock;
    }

    /**
     * @notice Emitted when a subclaim is disagreed upon by `claimant`
     * @dev Disagreeing with a subclaim is akin to attacking it.
     * @param parentIndex The index within the `claimData` array of the parent claim
     * @param pivot The claim for the following pivot (disagreement = go left)
     * @param claimant The address of the claimant
     */
    event Attack(uint256 indexed parentIndex, Claim indexed pivot, address indexed claimant);

    /**
     * @notice Emitted when a subclaim is agreed upon by `claimant`
     * @dev Agreeing with a subclaim is akin to defending it.
     * @param parentIndex The index within the `claimData` array of the parent claim
     * @param pivot The claim for the following pivot (agreement = go right)
     * @param claimant The address of the claimant
     */
    event Defend(uint256 indexed parentIndex, Claim indexed pivot, address indexed claimant);

    /**
     * Attack a disagreed upon `Claim`.
     * @param parentIndex Index of the `Claim` to attack in `claimData`.
     * @param pivot The `Claim` at the relative attack position.
     */
    function attack(uint256 parentIndex, Claim pivot) external payable;

    /**
     * Defend an agreed upon `Claim`.
     * @param parentIndex Index of the claim to defend in `claimData`.
     * @param pivot The `Claim` at the relative defense position.
     */
    function defend(uint256 parentIndex, Claim pivot) external payable;

    /**
     * @notice Perform the final step via an on-chain fault proof processor
     * @dev This function should point to a fault proof processor in order to execute
     *      a step in the fault proof program on-chain. The interface of the fault proof
     *      processor contract should be generic enough such that we can use different
     *      fault proof VMs (MIPS, RiscV5, etc.)
     * @param disagreement The ClaimHash of the disagreement
     */
    function step(ClaimHash disagreement) external;
}
