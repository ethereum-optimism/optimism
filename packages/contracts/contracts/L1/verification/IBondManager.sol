// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/**
 * @title IBondManager
 */
interface IBondManager {
    /********************
     * Public Functions *
     ********************/

    /**
     * Checks whether a given address is properly collateralized and can perform actions within
     * the system.
     * @param _who Address to check.
     * @return true if the address is properly collateralized, false otherwise.
     */
    function isCollateralized(address _who) external view returns (bool);
}
