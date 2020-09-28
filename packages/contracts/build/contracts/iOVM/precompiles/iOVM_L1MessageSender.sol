// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/**
 * @title iOVM_L1MessageSender
 */
interface iOVM_L1MessageSender {
    
    /********************
     * Public Functions *
     ********************/
    
    function getL1MessageSender() external returns (address _l1MessageSender);
}
