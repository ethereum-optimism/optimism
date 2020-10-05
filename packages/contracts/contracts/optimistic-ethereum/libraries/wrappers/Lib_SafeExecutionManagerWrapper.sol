// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/**
 * @title Lib_SafeExecutionManagerWrapper
 */
library Lib_SafeExecutionManagerWrapper {

    /**********************
     * Internal Functions *
     **********************/

    /**
     * Makes an ovmCALL and performs all the necessary safety checks.
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     * @param _gasLimit Gas limit for the call.
     * @param _target Address to call.
     * @param _calldata Data to send to the call.
     * @return _success Whether or not the call reverted.
     * @return _returndata Data returned by the call.
     */
    function safeCALL(
        address _ovmExecutionManager,
        uint256 _gasLimit,
        address _target,
        bytes memory _calldata
    )
        internal
        returns (
            bool _success,
            bytes memory _returndata
        )
    {
        bytes memory returndata = _safeExecutionManagerInteraction(
            _ovmExecutionManager,
            abi.encodeWithSignature(
                "ovmCALL(uint256,address,bytes)",
                _gasLimit,
                _target,
                _calldata
            )
        );

        return abi.decode(returndata, (bool, bytes));
    }

    /**
     * Performs an ovmCREATE and the necessary safety checks.
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     * @param _gasLimit Gas limit for the creation.
     * @param _bytecode Code for the new contract.
     * @return _contract Address of the created contract.
     */
    function safeCREATE(
        address _ovmExecutionManager,
        uint256 _gasLimit,
        bytes memory _bytecode
    )
        internal
        returns (
            address _contract
        )
    {
        bytes memory returndata = _safeExecutionManagerInteraction(
            _ovmExecutionManager,
            _gasLimit,
            abi.encodeWithSignature(
                "ovmCREATE(bytes)",
                _bytecode
            )
        );

        return abi.decode(returndata, (address));
    }

    /**
     * Performs a safe ovmCHAINID call.
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     * @return _CHAINID Result of calling ovmCHAINID.
     */
    function safeCHAINID(
        address _ovmExecutionManager
    )
        internal
        returns (
            uint256 _CHAINID
        )
    {
        bytes memory returndata = _safeExecutionManagerInteraction(
            _ovmExecutionManager,
            abi.encodeWithSignature(
                "ovmCHAINID()"
            )
        );

        return abi.decode(returndata, (uint256));
    }

    /**
     * Performs a safe ovmADDRESS call.
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     * @return _ADDRESS Result of calling ovmADDRESS.
     */
    function safeADDRESS(
        address _ovmExecutionManager
    )
        internal
        returns (
            address _ADDRESS
        )
    {
        bytes memory returndata = _safeExecutionManagerInteraction(
            _ovmExecutionManager,
            abi.encodeWithSignature(
                "ovmADDRESS()"
            )
        );

        return abi.decode(returndata, (address));
    }

    /**
     * Performs a safe ovmGETNONCE call.
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     * @return _nonce Result of calling ovmGETNONCE.
     */
    function safeGETNONCE(
        address _ovmExecutionManager
    )
        internal
        returns (
            uint256 _nonce
        )
    {
        bytes memory returndata = _safeExecutionManagerInteraction(
            _ovmExecutionManager,
            abi.encodeWithSignature(
                "ovmGETNONCE()"
            )
        );

        return abi.decode(returndata, (uint256));
    }

    /**
     * Performs a safe ovmSETNONCE call.
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     * @param _nonce New account nonce.
     */
    function safeSETNONCE(
        address _ovmExecutionManager,
        uint256 _nonce
    )
        internal
    {
        _safeExecutionManagerInteraction(
            _ovmExecutionManager,
            abi.encodeWithSignature(
                "ovmSETNONCE(uint256)",
                _nonce
            )
        );
    }


    /*********************
     * Private Functions *
     *********************/

    /**
     * Performs an ovm interaction and the necessary safety checks.
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     * @param _gasLimit Gas limit for the interaction.
     * @param _calldata Data to send to the OVM_ExecutionManager (encoded with sighash).
     * @return _returndata Data sent back by the OVM_ExecutionManager.
     */
    function _safeExecutionManagerInteraction(
        address _ovmExecutionManager,
        uint256 _gasLimit,
        bytes memory _calldata
    )
        private
        returns (
            bytes memory _returndata
        )
    {
        (
            bool success,
            bytes memory returndata
        ) = _ovmExecutionManager.call{gas: _gasLimit}(_calldata);

        if (success == false) {
            assembly {
                revert(add(returndata, 0x20), mload(returndata))
            }
        } else if (returndata.length == 1) {
            assembly {
                return(0, 1)
            }
        } else {
            return returndata;
        }
    }

    function _safeExecutionManagerInteraction(
        address _ovmExecutionManager,
        bytes memory _calldata
    )
        private
        returns (
            bytes memory _returndata
        )
    {
        return _safeExecutionManagerInteraction(
            _ovmExecutionManager,
            gasleft(),
            _calldata
        );
    }
}
