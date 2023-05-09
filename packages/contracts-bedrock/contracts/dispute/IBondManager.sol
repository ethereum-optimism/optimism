/// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/**
 * @title IBondManager
 * @notice The Bond Manager holds ether posted as a bond for a bond id.
 */
interface IBondManager {
    /**
     * @notice Post a bond with a given id and owner.
     * @dev This function will revert if the provided bondId is already in use.
     * @param bondId is the id of the bond.
     * @param owner is the address that owns the bond.
     * @param minClaimHold is the minimum amount of time the owner
     *        must wait before reclaiming their bond.
     */
    function post(
        bytes32 bondId,
        address owner,
        uint256 minClaimHold
    ) external payable;

    /**
     * @notice Seizes the bond with the given id.
     * @dev This function will revert if there is no bond at the given id.
     * @param bondId is the id of the bond.
     */
    function seize(bytes32 bondId) external;

    /**
     * @notice Seizes the bond with the given id and distributes it to recipients.
     * @dev This function will revert if there is no bond at the given id.
     * @param bondId is the id of the bond.
     * @param recipients is a set of addresses to split the bond amongst.
     */
    function seizeAndSplit(bytes32 bondId, address[] calldata recipients) external;

    /**
     * @notice Reclaims the bond of the bond owner.
     * @dev This function will revert if there is no bond at the given id.
     * @param bondId is the id of the bond.
     */
    function reclaim(bytes32 bondId) external;
}
