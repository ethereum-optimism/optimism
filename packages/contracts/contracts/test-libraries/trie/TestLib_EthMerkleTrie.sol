// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_EthMerkleTrie } from "../../optimistic-ethereum/libraries/trie/Lib_EthMerkleTrie.sol";
import { Lib_OVMCodec } from "../../optimistic-ethereum/libraries/codec/Lib_OVMCodec.sol";

/**
 * @title TestLib_EthMerkleTrie
 */
contract TestLib_EthMerkleTrie {

    function proveAccountStorageSlotValue(
        address _address,
        bytes32 _key,
        bytes32 _value,
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        view
        returns (bool)
    {
        return Lib_EthMerkleTrie.proveAccountStorageSlotValue(
            _address,
            _key,
            _value,
            _stateTrieWitness,
            _storageTrieWitness,
            _stateTrieRoot
        );
    }

    function updateAccountStorageSlotValue(
        address _address,
        bytes32 _key,
        bytes32 _value,
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        view
        returns (bytes32)
    {
        return Lib_EthMerkleTrie.updateAccountStorageSlotValue(
            _address,
            _key,
            _value,
            _stateTrieWitness,
            _storageTrieWitness,
            _stateTrieRoot
        );
    }

    function proveAccountState(
        address _address,
        Lib_OVMCodec.EVMAccount memory _accountState,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        view
        returns (bool)
    {
        return Lib_EthMerkleTrie.proveAccountState(
            _address,
            _accountState,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function updateAccountState(
        address _address,
        Lib_OVMCodec.EVMAccount memory _accountState,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        view
        returns (bytes32)
    {
        return Lib_EthMerkleTrie.updateAccountState(
            _address,
            _accountState,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }
}
