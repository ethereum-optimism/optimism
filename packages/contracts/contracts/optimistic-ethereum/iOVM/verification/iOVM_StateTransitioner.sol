// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

/**
 * @title iOVM_StateTransitioner
 */
interface iOVM_StateTransitioner {

    /**********************************
     * Public Functions: State Access *
     **********************************/

    function getPreStateRoot() external view returns (bytes32 _preStateRoot);
    function getPostStateRoot() external view returns (bytes32 _postStateRoot);
    function isComplete() external view returns (bool _complete);

    
    /***********************************
     * Public Functions: Pre-Execution *
     ***********************************/

    function proveContractState(
        address _ovmContractAddress,
        address _ethContractAddress,
        Lib_OVMCodec.EVMAccount calldata _account,
        bytes calldata _stateTrieWitness
    ) external;

    function proveEmptyContractState(
        address _ovmContractAddress,
        bytes calldata _stateTrieWitness
    ) external;

    function proveStorageSlot(
        address _ovmContractAddress,
        bytes32 _key,
        bytes32 _value,
        bytes calldata _storageTrieWitness
    ) external;


    /*******************************
     * Public Functions: Execution *
     *******************************/

    function applyTransaction(
        Lib_OVMCodec.Transaction calldata _transaction
    ) external;


    /************************************
     * Public Functions: Post-Execution *
     ************************************/

    function commitContractState(
        address _ovmContractAddress,
        bytes calldata _stateTrieWitness
    ) external;

    function commitStorageSlot(
        address _ovmContractAddress,
        bytes32 _key,
        bytes calldata _stateTrieWitness,
        bytes calldata _storageTrieWitness
    ) external;


    /**********************************
     * Public Functions: Finalization *
     **********************************/

    function completeTransition() external;
}
