// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title iAtomicSwap
 */
interface iAtomicSwap {

    /********************
     *       Events     *
     ********************/

    event Open(bytes32 _swapID, address _closeTrader);

    event Expire(bytes32 _swapID);

    event Close(bytes32 _swapID);
    
    /***********************
     *       Functions     *
     ***********************/

    function open(
        bytes32 _swapID, 
        uint256 _openValue, 
        address _openContractAddress, 
        uint256 _closeValue, 
        address _closeTrader, 
        address _closeContractAddress
    ) 
        external;

    function close(
        bytes32 _swapID
    ) 
        external; 


    function expire(
        bytes32 _swapID
    ) 
        external;

    function check(
        bytes32 _swapID
    ) 
        external 
        view 
        returns (
            uint256 openValue, 
            address openContractAddress, 
            uint256 closeValue, 
            address closeTrader, 
            address closeContractAddress
        );
}
