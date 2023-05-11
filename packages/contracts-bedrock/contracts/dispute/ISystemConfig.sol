// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/**
 * @title ISystemConfig
 * @notice A minimal interface for the SystemConfig contract.
 */
interface ISystemConfig {
    /**
     * @notice The `signerSet` is a set of addresses that are allowed to issue positive attestations
     *         for alternative output proposals in the `AttestationDisputeGame`.
     */
    function signerSet() external view returns (address[] memory);

    /**
     * @notice The `signatureThreshold` is the number of positive attestations that must be issued
     *         for a given alternative output proposal in the `AttestationDisputeGame` before it is
     *         considered to be the canonical output.
     */
    function signatureThreshold() external view returns (uint256);

    /**
     * @notice An external setter for the `signerSet` mapping. This method is used to
     *         authenticate or deauthenticate a signer in the `AttestationDisputeGame`.
     * @param _signer Address of the signer to authenticate or deauthenticate.
     * @param _authenticated True if the signer should be authenticated, false if the
     *        signer should be removed.
     */
    function authenticateSigner(address _signer, bool _authenticated) external;

    /**
     * @notice An external setter for the `signatureThreshold` variable. This method is used to
     *         set the number of signatures required to invalidate an output proposal
     *         in the `AttestationDisputeGame`.
     */
    function setSignatureThreshold(uint256 _signatureThreshold) external;
}
