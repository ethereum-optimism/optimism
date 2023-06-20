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
        bool countered;
        Claim claim;
        Position position;
        Clock clock;
    }

    /**
     * @notice Emitted when a new claim is added to the DAG by `claimant`
     * @param parentIndex The index within the `claimData` array of the parent claim
     * @param pivot The claim being added
     * @param claimant The address of the claimant
     */
    event Move(uint256 indexed parentIndex, Claim indexed pivot, address indexed claimant);

    /**
     * @notice Attack a disagreed upon `Claim`.
     * @param _parentIndex Index of the `Claim` to attack in `claimData`.
     * @param _pivot The `Claim` at the relative attack position.
     */
    function attack(uint256 _parentIndex, Claim _pivot) external payable;

    /**
     * @notice Defend an agreed upon `Claim`.
     * @param _parentIndex Index of the claim to defend in `claimData`.
     * @param _pivot The `Claim` at the relative defense position.
     */
    function defend(uint256 _parentIndex, Claim _pivot) external payable;

    /**
     * @notice Perform the final step via an on-chain fault proof processor
     * @dev This function should point to a fault proof processor in order to execute
     *      a step in the fault proof program on-chain. The interface of the fault proof
     *      processor contract should be generic enough such that we can use different
     *      fault proof VMs (MIPS, RiscV5, etc.)
     * @param _prestateIndex The index of the prestate of the step within `claimData`.
     * @param _parentIndex The index of the parent claim within `claimData`.
     * @param _stateData The stateData of the step is the preimage of the claim @ `prestateIndex`
     * @param _proof Proof to access memory leaf nodes in the VM.
     */
    function step(
        uint256 _prestateIndex,
        uint256 _parentIndex,
        bytes calldata _stateData,
        bytes calldata _proof
    ) external;
}
