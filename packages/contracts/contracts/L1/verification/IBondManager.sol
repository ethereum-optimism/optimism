// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/**
 * @title IBondManager
 */
interface IBondManager {
    /********************
     * Public Functions *
     ********************/

    function isCollateralized(address _who) external view returns (bool);
}
