// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/**
 * @title IOwnable
 * @notice An interface for ownable contracts.
 */
interface IOwnable {
    /**
     * @notice Returns the owner of the contract
     * @return _owner The address of the owner
     */
    function owner() external view returns (address _owner);

    /**
     * @notice Transfers ownership of the contract to a new address
     * @dev May only be called by the contract owner
     * @param newOwner The address to transfer ownership to
     */
    function transferOwnership(address newOwner) external;
}
