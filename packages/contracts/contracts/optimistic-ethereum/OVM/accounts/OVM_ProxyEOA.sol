// SPDX-License-Identifier: MIT
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

    /*************
     * Constants *
     *************/

    bytes32 constant IMPLEMENTATION_KEY = 0xdeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddead;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _implementation Address of the initial implementation contract.
     */
    constructor(
        address _implementation
    )
    {
        _setImplementation(_implementation);
    }


    /*********************
     * Fallback Function *
     *********************/

    fallback()
        external
    {
        (bool success, bytes memory returndata) = getImplementation().call(msg.data);

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
        return Lib_Bytes32Utils.toAddress(addr32);
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
