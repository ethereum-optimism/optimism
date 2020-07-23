pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import { DataTypes } from "../../utils/libraries/DataTypes.sol";

/**
 * @title IStateTransitioner
 */
contract IStateTransitioner {
    bytes32 public preStateRoot;
    bytes32 public stateRoot;

    function proveContractInclusion(
        address _ovmContractAddress,
        address _codeContractAddress,
        uint256 _nonce,
        bytes memory _stateTrieWitness
    ) public;

    function proveStorageSlotInclusion(
        address _ovmContractAddress,
        bytes32 _slot,
        bytes32 _value,
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness
    ) public;

    function applyTransaction(
        DataTypes.OVMTransactionData memory _transactionData
    ) public;

    function proveUpdatedStorageSlot(
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness
    ) public;

    function proveUpdatedContract(
        bytes memory _stateTrieWitness
    ) public;

    function completeTransition() public;

    function isComplete() public view returns (bool);
}
