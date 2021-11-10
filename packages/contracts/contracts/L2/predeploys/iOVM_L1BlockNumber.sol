// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/**
 * @title iOVM_L1BlockNumber
 */
interface iOVM_L1BlockNumber {
    /********************
     * Public Functions *
     ********************/

    function getL1BlockNumber() external view returns (uint256);
}
