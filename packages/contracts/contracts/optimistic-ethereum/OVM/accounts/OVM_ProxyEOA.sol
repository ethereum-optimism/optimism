// SPDX-License-Identifier: MIT
// @unsupported: evm
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_Bytes32Utils } from "../../libraries/utils/Lib_Bytes32Utils.sol";

/**
 * @title OVM_ProxyEOA
 * @dev The Proxy EOA contract uses a delegate call to execute the logic in an implementation contract.
 * In combination with the logic implemented in the ECDSA Contract Account, this enables a form of upgradable
 * 'account abstraction' on layer 2.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_ProxyEOA {

    /**********
     * Events *
     **********/
    
    event Upgraded(
        address indexed implementation
    );


    /*************
     * Constants *
     *************/

    address constant DEFAULT_IMPLEMENTATION = 0x4200000000000000000000000000000000000003;
    bytes32 constant IMPLEMENTATION_KEY = 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc; //bytes32(uint256(keccak256('eip1967.proxy.implementation')) - 1);


    /*********************
     * Fallback Function *
     *********************/

    fallback()
        external
    {
        (bool success, bytes memory returndata) = getImplementation().delegatecall(msg.data);

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

    // WARNING: We use the deployed bytecode of this contract as a template to create ProxyEOA
    // contracts. As a result, we must *not* perform any constructor logic. Use initialization
    // functions if necessary.


    /********************
     * Public Functions *
     ********************/

    /**
     * Changes the implementation address.
     * @param _implementation New implementation address.
     */
    function upgrade(
        address _implementation
    )
        external
    {
        require(
            msg.sender == address(this),
            "EOAs can only upgrade their own EOA implementation"
        );

        _setImplementation(_implementation);
        emit Upgraded(_implementation);
    }

    /**
     * Gets the address of the current implementation.
     * @return Current implementation address.
     */
    function getImplementation()
        public
        returns (
            address
        )
    {
        bytes32 addr32;
        assembly {
            addr32 := sload(IMPLEMENTATION_KEY)
        }

        address implementation = Lib_Bytes32Utils.toAddress(addr32);
        if (implementation == address(0)) {
            return DEFAULT_IMPLEMENTATION;
        } else {
            return implementation;
        }
    }


    /**********************
     * Internal Functions *
     **********************/

    function _setImplementation(
        address _implementation
    )
        internal
    {
        bytes32 addr32 = Lib_Bytes32Utils.fromAddress(_implementation);
        assembly {
            sstore(IMPLEMENTATION_KEY, addr32)
        }
    }
}
