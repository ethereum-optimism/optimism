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
     * Makes an ovmCALL and performs all the necessary safety checks.
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     * @param _gasLimit Gas limit for the call.
     * @param _target Address to call.
     * @param _calldata Data to send to the call.
     * @return _success Whether or not the call reverted.
     * @return _returndata Data returned by the call.
     */
    function safeDELEGATECALL(
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
                "ovmDELEGATECALL(uint256,address,bytes)",
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
     * Performs an ovmEXTCODESIZE and the necessary safety checks.
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     * @param _contract Address of the contract to query the size of.
     * @return _EXTCODESIZE Size of the requested contract in bytes.
     */
    function safeEXTCODESIZE(
        address _ovmExecutionManager,
        address _contract
    )
        internal
        returns (
            uint256 _EXTCODESIZE
        )
    {
        bytes memory returndata = _safeExecutionManagerInteraction(
            _ovmExecutionManager,
            abi.encodeWithSignature(
                "ovmEXTCODESIZE(address)",
                _contract
            )
        );

        return abi.decode(returndata, (uint256));
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
     * Performs a safe ovmCALLER call.
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     * @return _CALLER Result of calling ovmCALLER.
     */
    function safeCALLER(
        address _ovmExecutionManager
    )
        internal
        returns (
            address _CALLER
        )
    {
        bytes memory returndata = _safeExecutionManagerInteraction(
            _ovmExecutionManager,
            abi.encodeWithSignature(
                "ovmCALLER()"
            )
        );

        return abi.decode(returndata, (address));
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

    /**
     * Performs a safe ovmCREATEEOA call.
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     * @param _messageHash Message hash which was signed by EOA
     * @param _v v value of signature (0 or 1)
     * @param _r r value of signature
     * @param _s s value of signature
     */
    function safeCREATEEOA(
        address _ovmExecutionManager,
        bytes32 _messageHash,
        uint8 _v,
        bytes32 _r,
        bytes32 _s
    )
        internal
    {
        _safeExecutionManagerInteraction(
            _ovmExecutionManager,
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
     * Performs a safe REVERT.
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     * @param _reason String revert reason to pass along with the REVERT.
     */
    function safeREVERT(
        address _ovmExecutionManager,
        string memory _reason
    )
        internal
    {
        _safeExecutionManagerInteraction(
            _ovmExecutionManager,
            abi.encodeWithSignature(
                "ovmREVERT(bytes)",
                bytes(_reason)
            )
        );
    }

    /**
     * Performs a safe "require".
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     * @param _condition Boolean condition that must be true or will revert.
     * @param _reason String revert reason to pass along with the REVERT.
     */
    function safeREQUIRE(
        address _ovmExecutionManager,
        bool _condition,
        string memory _reason
    )
        internal
    {
        if (!_condition) {
            safeREVERT(
                _ovmExecutionManager,
                _reason
            );
        }
    }

    /**
     * Performs a safe ovmSLOAD call.
     */
    function safeSLOAD(
        address _ovmExecutionManager,
        bytes32 _key
    )
        internal
        returns (
            bytes32
        )
    {
        bytes memory returndata = _safeExecutionManagerInteraction(
            _ovmExecutionManager,
            abi.encodeWithSignature(
                "ovmSLOAD(bytes32)",
                _key
            )
        );

        return abi.decode(returndata, (bytes32));
    }

    /**
     * Performs a safe ovmSSTORE call.
     */
    function safeSSTORE(
        address _ovmExecutionManager,
        bytes32 _key,
        bytes32 _value
    )
        internal
    {
        _safeExecutionManagerInteraction(
            _ovmExecutionManager,
            abi.encodeWithSignature(
                "ovmSSTORE(bytes32,bytes32)",
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
