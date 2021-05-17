// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { OVM_CrossDomainEnabled } from "../optimistic-ethereum/libraries/bridge/OVM_CrossDomainEnabled.sol";

/**
 * @title L2ChugSplashOwner
 * @dev This contract will be the owner of the L2ChugSplashDeployer contract on deployed networks.
 * By separating this from the L2ChugSplashDeployer, we can more easily test the core ChugSplash
 * logic. It's effectively just a proxy to the L2ChugSplashDeployer.
 */
contract L2ChugSplashOwner is OVM_CrossDomainEnabled {

    /**********
     * Events *
     **********/

    event OwnershipTransferred(
        address indexed previousOwner,
        address indexed newOwner
    );


    /*************
     * Variables *
     *************/

    address public owner;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _owner Address that will initially own the L2ChugSplashOwner.
     */
    constructor(
        address _owner
    )
        public
        OVM_CrossDomainEnabled(0x4200000000000000000000000000000000000007)
    {
        // Need to replicate the code from transferOwnership because transferOwnership can only be
        // called via an L1 => L2 message.
        require(
            _owner != address(0),
            "L2ChugSplashOwner: new owner is the zero address"
        );

        emit OwnershipTransferred(owner, _owner);
        owner = _owner;
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * Leaves the contract without owner.
     */
    function renounceOwnership()
        public
        onlyFromCrossDomainAccount(owner)
    {
        emit OwnershipTransferred(owner, address(0));
        owner = address(0);
    }

    /**
     * Transfers ownership to a new address.
     * @param _newOwner Address of the new owner.
     */
    function transferOwnership(
        address _newOwner
    )
        public
        onlyFromCrossDomainAccount(owner)
    {
        require(
            _newOwner != address(0),
            "L2ChugSplashOwner: new owner is the zero address"
        );

        emit OwnershipTransferred(owner, _newOwner);
        owner = _newOwner;
    }


    /*********************
     * Fallback Function *
     *********************/

    fallback()
        external
        onlyFromCrossDomainAccount(owner)
    {
        (bool success, bytes memory returndata) = address(
            0x420000000000000000000000000000000000000D
        ).call(msg.data);

        if (success) {
            assembly {
                return(add(returndata, 0x20), mload(returndata))
            }
        } else {
            assembly {
                revert(add(returndata, 0x20), mload(returndata))
            }
        }
    }
}
