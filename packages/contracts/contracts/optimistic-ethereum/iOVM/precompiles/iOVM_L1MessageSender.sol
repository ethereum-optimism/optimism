// SPDX-License-Identifier: UNLICENSED
pragma solidity >0.5.0 <0.8.0;

/**
 * @title iOVM_L1MessageSender
 */
interface iOVM_L1MessageSender {
    
    /********************
     * Public Functions *
     ********************/
    
    function getL1MessageSender() external returns (address _l1MessageSender);
}
