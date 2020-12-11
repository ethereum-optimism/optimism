// SPDX-License-Identifier: MIT
// +build ovm
pragma solidity >0.6.0 <0.8.0;

/**
 * @title Ownable
 * @dev Adapted from https://github.com/OpenZeppelin/openzeppelin-contracts/blob/master/contracts/access/Ownable.sol
 */
abstract contract Ownable {

    /*************
     * Variables *
     *************/

    address public owner;


    /**********
     * Events *
     **********/

    event OwnershipTransferred(
        address indexed previousOwner,
        address indexed newOwner
    );


    /***************
     * Constructor *
     ***************/

    constructor() internal {
        owner = msg.sender;
        emit OwnershipTransferred(address(0), owner);
    }


    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyOwner() {
        require(
            owner == msg.sender,
            "Ownable: caller is not the owner"
        );
        _;
    }


    /********************
     * Public Functions *
     ********************/

    function renounceOwnership()
        public
        virtual
        onlyOwner 
    {
        emit OwnershipTransferred(owner, address(0));
        owner = address(0);
    }

    function transferOwnership(address _newOwner)
        public
        virtual
        onlyOwner
    {
        require(
            _newOwner != address(0),
            "Ownable: new owner cannot be the zero address"
        );

        emit OwnershipTransferred(owner, _newOwner);
        owner = _newOwner;
    }
}
