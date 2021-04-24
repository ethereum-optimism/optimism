// SPDX-License-Identifier: MIT
// @unsupported: evm
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
     * Performs a safe ovmCREATE call.
     * @param _bytecode Code for the new contract.
     * @return _contract Address of the created contract.
     */
    function ovmCREATE(
        bytes memory _bytecode
    )
        internal
        returns (
            address,
            bytes memory
        )
    {
        bytes memory returndata = _safeExecutionManagerInteraction(
            abi.encodeWithSignature(
                "ovmCREATE(bytes)",
                _bytecode
            )
        );

        return abi.decode(returndata, (address, bytes));
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

    /**
     * Calls the ovmL1TXORIGIN opcode.
     * @return Address that sent this message from L1.
     */
    function ovmL1TXORIGIN()
        internal
        returns (
            address
        )
    {
        bytes memory returndata = _safeExecutionManagerInteraction(
            abi.encodeWithSignature(
                "ovmL1TXORIGIN()"
            )
        );

        return abi.decode(returndata, (address));
    }

    /**
     * Calls the ovmCHAINID opcode.
     * @return Chain ID of the current network.
     */
    function ovmCHAINID()
        internal
        returns (
            uint256
        )
    {
        bytes memory returndata = _safeExecutionManagerInteraction(
            abi.encodeWithSignature(
                "ovmCHAINID()"
            )
        );

        return abi.decode(returndata, (uint256));
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
            // kall is a custom yul builtin within optimistic-solc that allows us to directly call
            // the execution manager (since `call` would be compiled).
            kall(add(_calldata, 0x20), mload(_calldata), 0x0, 0x0)
            let size := returndatasize()
            returndata := mload(0x40)
            mstore(0x40, add(returndata, and(add(add(size, 0x20), 0x1f), not(0x1f))))
            mstore(returndata, size)
            returndatacopy(add(returndata, 0x20), 0x0, size)
        }
        return returndata;
    }
}
