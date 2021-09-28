// SPDX-License-Identifier: MIT
pragma solidity ^0.8.8;

/**
 * @title iOVM_L1MessageSender
 */
interface iOVM_L1MessageSender {

    /********************
     * Public Functions *
     ********************/

    function getL1MessageSender() external view returns (address);
}
