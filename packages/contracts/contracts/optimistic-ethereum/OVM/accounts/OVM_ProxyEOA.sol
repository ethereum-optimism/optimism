pragma solidity ^0.7.0;

/* Library Imports */
import { Lib_BytesUtils } from "../../libraries/utils/Lib_BytesUtils.sol";
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_ECDSAUtils } from "../../libraries/utils/Lib_ECDSAUtils.sol";
import { Lib_SafeExecutionManagerWrapper } from "../../libraries/wrappers/Lib_SafeExecutionManagerWrapper.sol";

/**
 * @title OVM_ProxyEOA
 */
contract OVM_ProxyEOA {

    /***************
     * Constructor *
     ***************/

    constructor(
        address _implementation
    ) {
        _setImplementation(_implementation);
    }


    /*********************
     * Fallback Function *
     *********************/

    fallback()
        external
    {
        (bool success, bytes memory returndata) = Lib_SafeExecutionManagerWrapper.safeDELEGATECALL(
            msg.sender,
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
                msg.sender,
                string(returndata)
            );
        }
    }


    /********************
     * Public Functions *
     ********************/

    function upgrade(
        address _implementation
    )
        external
    {
        Lib_SafeExecutionManagerWrapper.safeREQUIRE(
            msg.sender,
            Lib_SafeExecutionManagerWrapper.safeADDRESS(msg.sender) == Lib_SafeExecutionManagerWrapper.safeCALLER(msg.sender),
            "EOAs can only upgrade their own EOA implementation"
        );

        _setImplementation(_implementation);
    }

    function getImplementation()
        public
        returns (
            address _implementation
        )
    {
        return address(uint160(uint256(
            Lib_SafeExecutionManagerWrapper.safeSLOAD(
                msg.sender,
                bytes32(uint256(0))
            )
        )));
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
            msg.sender,
            bytes32(uint256(0)),
            bytes32(uint256(uint160(_implementation)))
        );
    }
}