// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_ErrorUtils } from "../utils/Lib_ErrorUtils.sol";

/**
 * @title Lib_ExecutionManagerWrapper
 * @dev This library acts as a utility for easily calling the OVM_ExecutionManagerWrapper, the
 *  predeployed contract which exposes the `kall` builtin. Effectively, this contract allows the
 *  user to trigger OVM opcodes by directly calling the OVM_ExecutionManger.
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
     * @return Address of the created contract.
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
        bytes memory returndata = _callWrapperContract(
            abi.encodeWithSignature(
                "ovmCREATE(bytes)",
                _bytecode
            )
        );

        return abi.decode(returndata, (address, bytes));
    }

    /**
     * Performs a safe ovmGETNONCE call.
     * @return Result of calling ovmGETNONCE.
     */
    function ovmGETNONCE()
        internal
        returns (
            uint256
        )
    {
        bytes memory returndata = _callWrapperContract(
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
        _callWrapperContract(
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
        _callWrapperContract(
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
        bytes memory returndata = _callWrapperContract(
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
        bytes memory returndata = _callWrapperContract(
            abi.encodeWithSignature(
                "ovmCHAINID()"
            )
        );

        return abi.decode(returndata, (uint256));
    }

    /**
     * Performs a safe ovmADDRESS call.
     * @return Result of calling ovmADDRESS.
     */
    function ovmADDRESS()
        internal
        returns (
            address
        )
    {
        bytes memory returndata = _callWrapperContract(
            abi.encodeWithSignature(
                "ovmADDRESS()"
            )
        );

        return abi.decode(returndata, (address));
    }

    /**
     * Calls the ovmSETCODE opcode. Only callable by the upgrade deployer.
     * @param _address Address to set the code of.
     * @param _code New code for the address.
     */
    function ovmSETCODE(
        address _address,
        bytes memory _code
    )
        internal
    {
        _callWrapperContract(
            abi.encodeWithSignature(
                "ovmSETCODE(address,bytes)",
                _address,
                _code
            )
        );
    }

    /**
     * Calls the ovmSETSTORAGE opcode. Only callable by the upgrade deployer.
     * @param _address Address to set a storage slot for.
     * @param _key Storage slot key to modify.
     * @param _value Storage slot value.
     */
    function ovmSETSTORAGE(
        address _address,
        bytes32 _key,
        bytes32 _value
    )
        internal
    {
        _callWrapperContract(
            abi.encodeWithSignature(
                "ovmSETSTORAGE(address,bytes32,bytes32)",
                _address,
                _key,
                _value
            )
        );
    }


    /*********************
     * Private Functions *
     *********************/

    /**
     * Performs an ovm interaction and the necessary safety checks.
     * @param _calldata Data to send to the OVM_ExecutionManager (encoded with sighash).
     * @return Data sent back by the OVM_ExecutionManager.
     */
    function _callWrapperContract(
        bytes memory _calldata
    )
        private
        returns (
            bytes memory
        )
    {
        (bool success, bytes memory returndata) = 0x420000000000000000000000000000000000000B.delegatecall(_calldata);

        if (success == true) {
            return returndata;
        } else {
            assembly {
                revert(add(returndata, 0x20), mload(returndata))
            }
        }
    }
}
