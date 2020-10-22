// SPDX-License-Identifier: UNLICENSED
pragma solidity >0.5.0 <0.8.0;

/**
 * @title iOVM_L2ToL1MessagePasser
 */
interface iOVM_L2ToL1MessagePasser {

    /**********
     * Events *
     **********/

    event L2ToL1Message(
       uint _nonce,
       address _sender,
       bytes _data
    );


    /********************
     * Public Functions *
     ********************/
    
    function passMessageToL1(bytes calldata _message) external;
}
