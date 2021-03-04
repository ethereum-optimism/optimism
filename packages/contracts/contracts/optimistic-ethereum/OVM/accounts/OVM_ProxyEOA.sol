// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_Bytes32Utils } from "../../libraries/utils/Lib_Bytes32Utils.sol";
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_ECDSAUtils } from "../../libraries/utils/Lib_ECDSAUtils.sol";
import { Lib_SafeExecutionManagerWrapper } from "../../libraries/wrappers/Lib_SafeExecutionManagerWrapper.sol";

/**
 * @title OVM_ProxyEOA
 * @dev The Proxy EOA contract uses a delegate call to execute the logic in an implementation contract.
 * In combination with the logic implemented in the ECDSA Contract Account, this enables a form of upgradable 
 * 'account abstraction' on layer 2. 
 * 
 * Compiler used: solc
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
        public
    {
        _setImplementation(_implementation);
    }


    /*********************
     * Fallback Function *
     *********************/

    fallback()
        external
    {
        (bool success, bytes memory returndata) = Lib_SafeExecutionManagerWrapper.safeDELEGATECALL(
            gasleft(),
            getImplementation(),
            msg.data
        );

        if (success) {
            assembly {
                return(add(returndata, 0x20), mload(returndata))
            }
        } else {
            Lib_SafeExecutionManagerWrapper.safeREVERT(
                string(returndata)
            );
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
        Lib_SafeExecutionManagerWrapper.safeREQUIRE(
            Lib_SafeExecutionManagerWrapper.safeADDRESS() == Lib_SafeExecutionManagerWrapper.safeCALLER(),
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
        return Lib_Bytes32Utils.toAddress(
            Lib_SafeExecutionManagerWrapper.safeSLOAD(
                IMPLEMENTATION_KEY
            )
        );
    }


    /**********************
     * Internal Functions *
     **********************/

    function _setImplementation(
        address _implementation
    )
        internal
    {
        Lib_SafeExecutionManagerWrapper.safeSSTORE(
            IMPLEMENTATION_KEY,
            Lib_Bytes32Utils.fromAddress(_implementation)
        );
    }
}
