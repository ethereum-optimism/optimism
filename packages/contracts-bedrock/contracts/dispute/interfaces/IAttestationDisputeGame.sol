// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { IDisputeGame } from "./IDisputeGame.sol";

/**
 * @title IAttestationDisputeGame
 * @notice The interface for an attestation-based DisputeGame meant to contest output
 *         proposals in Optimism's `L2OutputOracle` contract.
 */
interface IAttestationDisputeGame is IDisputeGame {
    /**
     * @notice A mapping of addresses from the `attestorSet` to booleans signifying whether
     *         or not they have authorized the `rootClaim` to be invalidated.
     * @param challenger The address to check for authorization.
     * @return _challenged Whether or not the `challenger` has challenged the `rootClaim`.
     */
    function challenges(address challenger) external view returns (bool _challenged);

    /**
     * @notice The attestor set consists of authorized public keys that may challenge
     *         the `rootClaim`.
     * @param addr The address to check for authorization.
     * @return _isAuthorized Whether or not the `addr` is part of the attestor set.
     */
    function attestorSet(address addr) external view returns (bool _isAuthorized);

    /**
     * @notice The amount of signatures required to successfully challenge the `rootClaim`
     *         output proposal. Once this threshold is met by members of the `attestorSet`
     *         calling `challenge`, the game will be resolved to `CHALLENGER_WINS`.
     * @custom:invariant The `signatureThreshold` may never be greater than the length
     *                   of the `attestorSet`.
     * @return _signatureThreshold The amount of signatures required to successfully
     *         challenge the `rootClaim` output proposal.
     */
    function frozenSignatureThreshold() external view returns (uint256 _signatureThreshold);

    /**
     * @notice Returns the L2 Block Number that the `rootClaim` commits to.
     *         Exists within the `extraData`.
     * @return _l2BlockNumber The L2 Block Number that the `rootClaim` commits to.
     */
    function l2BlockNumber() external view returns (uint256 _l2BlockNumber);

    /**
     * @notice Challenge the `rootClaim`.
     * @dev - If the `ecrecover`ed address that created the signature is not a part of
     *        the attestor set returned by `attestorSet`, this function should revert.
     *      - If the `ecrecover`ed address that created the signature is not the
     *        msg.sender, this function should revert.
     *      - If the signature provided is the signature that breaches the signature
     *        threshold, the function should call the `resolve` function to resolve
     *        the game as `CHALLENGER_WINS`.
     *      - When the game resolves, the bond attached to the root claim should be
     *        distributed among the signers who participated in challenging the
     *        invalid claim.
     * @param signature An EIP-712 signature committing to the `rootClaim` and
     *        `l2BlockNumber` (within the `extraData`) from a key that exists
     *         within the `attestorSet`.
     */
    function challenge(bytes calldata signature) external;
}
