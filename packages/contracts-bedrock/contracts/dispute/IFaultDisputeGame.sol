// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Claim, ClaimHash, Clock, Bond, Position, Timestamp } from "../libraries/DisputeTypes.sol";

import { IDisputeGame } from "./IDisputeGame.sol";

/**
 * @title IFaultDisputeGame
 * @notice The interface for a fault proof backed dispute game.
 */
interface IFaultDisputeGame is IDisputeGame {
    /**
     * @notice Emitted when a subclaim is disagreed upon by `claimant`
     * @dev Disagreeing with a subclaim is akin to attacking it.
     * @param claimHash The unique ClaimHash that is being disagreed upon
     * @param pivot The claim for the following pivot (disagreement = go left)
     * @param claimant The address of the claimant
     */
    event Attack(ClaimHash indexed claimHash, Claim indexed pivot, address indexed claimant);

    /**
     * @notice Emitted when a subclaim is agreed upon by `claimant`
     * @dev Agreeing with a subclaim is akin to defending it.
     * @param claimHash The unique ClaimHash that is being agreed upon
     * @param pivot The claim for the following pivot (agreement = go right)
     * @param claimant The address of the claimant
     */
    event Defend(ClaimHash indexed claimHash, Claim indexed pivot, address indexed claimant);

    /**
     * @notice State variable of the starting timestamp of the game, set on deployment.
     * @return The starting timestamp of the game
     */
    function gameStart() external view returns (Timestamp);

    /**
     * @notice Maps a unique ClaimHash to a Claim.
     * @param claimHash The unique ClaimHash
     * @return claim The Claim associated with the ClaimHash
     */
    function claims(ClaimHash claimHash) external view returns (Claim claim);

    /**
     * @notice Maps a unique ClaimHash to its parent.
     * @param claimHash The unique ClaimHash
     * @return parent The parent ClaimHash of the passed ClaimHash
     */
    function parents(ClaimHash claimHash) external view returns (ClaimHash parent);

    /**
     * @notice Maps a unique ClaimHash to its Position.
     * @param claimHash The unique ClaimHash
     * @return position The Position associated with the ClaimHash
     */
    function positions(ClaimHash claimHash) external view returns (Position position);

    /**
     * @notice Maps a unique ClaimHash to a Bond.
     * @param claimHash The unique ClaimHash
     * @return bond The Bond associated with the ClaimHash
     */
    function bonds(ClaimHash claimHash) external view returns (Bond bond);

    /**
     * @notice Maps a unique ClaimHash its chess clock.
     * @param claimHash The unique ClaimHash
     * @return clock The chess clock associated with the ClaimHash
     */
    function clocks(ClaimHash claimHash) external view returns (Clock clock);

    /**
     * @notice Maps a unique ClaimHash to its reference counter.
     * @param claimHash The unique ClaimHash
     * @return _rc The reference counter associated with the ClaimHash
     */
    function rc(ClaimHash claimHash) external view returns (uint64 _rc);

    /**
     * @notice Maps a unique ClaimHash to a boolean indicating whether or not it has been countered.
     * @param claimHash The unique claimHash
     * @return _countered Whether or not `claimHash` has been countered
     */
    function countered(ClaimHash claimHash) external view returns (bool _countered);

    /**
     * @notice Disagree with a subclaim
     * @param disagreement The ClaimHash of the disagreement
     * @param pivot The claimed pivot
     */
    function attack(ClaimHash disagreement, Claim pivot) external;

    /**
     * @notice Agree with a subclaim
     * @param agreement The ClaimHash of the agreement
     * @param pivot The claimed pivot
     */
    function defend(ClaimHash agreement, Claim pivot) external;

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
