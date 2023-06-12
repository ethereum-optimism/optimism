// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/**
 * @title IBondManager
 * @notice The Bond Manager holds ether posted as a bond for a bond id.
 */
interface IBondManager {
    /**
     * @notice BondPosted is emitted when a bond is posted.
     * @param bondId is the id of the bond.
     * @param owner is the address that owns the bond.
     * @param expiration is the time at which the bond expires.
     * @param amount is the amount of the bond.
     */
    event BondPosted(bytes32 bondId, address owner, uint256 expiration, uint256 amount);

    /**
     * @notice BondSeized is emitted when a bond is seized.
     * @param bondId is the id of the bond.
     * @param owner is the address that owns the bond.
     * @param seizer is the address that seized the bond.
     * @param amount is the amount of the bond.
     */
    event BondSeized(bytes32 bondId, address owner, address seizer, uint256 amount);

    /**
     * @notice BondReclaimed is emitted when a bond is reclaimed by the owner.
     * @param bondId is the id of the bond.
     * @param claiment is the address that reclaimed the bond.
     * @param amount is the amount of the bond.
     */
    event BondReclaimed(bytes32 bondId, address claiment, uint256 amount);

    /**
     * @notice Post a bond with a given id and owner.
     * @dev This function will revert if the provided bondId is already in use.
     * @param _bondId is the id of the bond.
     * @param _bondOwner is the address that owns the bond.
     * @param _minClaimHold is the minimum amount of time the owner
     *        must wait before reclaiming their bond.
     */
    function post(
        bytes32 _bondId,
        address _bondOwner,
        uint128 _minClaimHold
    ) external payable;

    /**
     * @notice Seizes the bond with the given id.
     * @dev This function will revert if there is no bond at the given id.
     * @param _bondId is the id of the bond.
     */
    function seize(bytes32 _bondId) external;

    /**
     * @notice Seizes the bond with the given id and distributes it to recipients.
     * @dev This function will revert if there is no bond at the given id.
     * @param _bondId is the id of the bond.
     * @param _claimRecipients is a set of addresses to split the bond amongst.
     */
    function seizeAndSplit(bytes32 _bondId, address[] calldata _claimRecipients) external;

    /**
     * @notice Reclaims the bond of the bond owner.
     * @dev This function will revert if there is no bond at the given id.
     * @param _bondId is the id of the bond.
     */
    function reclaim(bytes32 _bondId) external;
}
