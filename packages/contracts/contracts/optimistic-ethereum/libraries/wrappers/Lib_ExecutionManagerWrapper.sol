// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_ErrorUtils } from "../utils/Lib_ErrorUtils.sol";

/**
 * @title Lib_ExecutionManagerWrapper
 *
 * Compiler used: solc
 * Runtime target: OVM
 */
library Lib_ExecutionManagerWrapper {

    /**********************
     * Internal Functions *
     **********************/

    /**
     * Performs a safe ovmCHAINID call.
     * @return _CHAINID Result of calling ovmCHAINID.
     */
    function ovmCHAINID()
        internal
        returns (
            uint256 _CHAINID
        )
    {
        bytes memory returndata = _safeExecutionManagerInteraction(
            abi.encodeWithSignature(
                "ovmCHAINID()"
            )
        );

        return abi.decode(returndata, (uint256));
    }

    /**
     * Performs a safe ovmGETNONCE call.
     * @return _nonce Result of calling ovmGETNONCE.
     */
    function ovmGETNONCE()
        internal
        returns (
            uint256 _nonce
        )
    {
        bytes memory returndata = _safeExecutionManagerInteraction(
            abi.encodeWithSignature(
                "ovmGETNONCE()"
            )
        );

        return abi.decode(returndata, (uint256));
    }

    /**
     * Performs a safe ovmINCREMENTNONCE call.
     */
    function ovmINCREMENTNONCE()
        internal
    {
        _safeExecutionManagerInteraction(
            abi.encodeWithSignature(
                "ovmINCREMENTNONCE()"
            )
        );
    }

    /**
     * Performs a safe ovmCREATEEOA call.
     * @param _messageHash Message hash which was signed by EOA
     * @param _v v value of signature (0 or 1)
     * @param _r r value of signature
     * @param _s s value of signature
     */
    function ovmCREATEEOA(
        bytes32 _messageHash,
        uint8 _v,
        bytes32 _r,
        bytes32 _s
    )
        internal
    {
        _safeExecutionManagerInteraction(
            abi.encodeWithSignature(
                "ovmCREATEEOA(bytes32,uint8,bytes32,bytes32)",
                _messageHash,
                _v,
                _r,
                _s
            )
        );
    }


    /*********************
     * Private Functions *
     *********************/

    /**
     * Performs an ovm interaction and the necessary safety checks.
     * @param _calldata Data to send to the OVM_ExecutionManager (encoded with sighash).
     * @return _returndata Data sent back by the OVM_ExecutionManager.
     */
    function _safeExecutionManagerInteraction(
        bytes memory _calldata
    )
        private
        returns (
            bytes memory
        )
    {
        bytes memory returndata;
        assembly {
            kall(add(_calldata, 0x20), mload(_calldata), 0x0, 0x0)
            let size := returndatasize()
            returndata := mload(0x40)
            mstore(0x40, add(add(returndata, 0x20), size))
            mstore(returndata, size)
            returndatacopy(add(returndata, 0x20), 0x0, size)
        }

        return returndata;
    }
}
