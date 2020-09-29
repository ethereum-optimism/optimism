// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_EthMerkleTrie } from "../../optimistic-ethereum/libraries/trie/Lib_EthMerkleTrie.sol";
import { Lib_OVMCodec } from "../../optimistic-ethereum/libraries/codec/Lib_OVMCodec.sol";

/**
 * @title TestLib_EthMerkleTrie
 */
library TestLib_EthMerkleTrie {

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
        Lib_OVMCodec.ProofMatrix memory _proofMatrix,
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
            _proofMatrix,
            _stateTrieWitness,
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

    function proveAccountState(
        address _address,
        Lib_OVMCodec.Account memory _accountState,
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

    function proveAccountNonce(
        address _address,
        uint256 _nonce,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        view
        returns (bool)
    {
        return Lib_EthMerkleTrie.proveAccountNonce(
            _address,
            _nonce,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function proveAccountBalance(
        address _address,
        uint256 _balance,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        view
        returns (bool)
    {
        return Lib_EthMerkleTrie.proveAccountBalance(
            _address,
            _balance,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function proveAccountStorageRoot(
        address _address,
        bytes32 _storageRoot,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        view
        returns (bool)
    {
        return Lib_EthMerkleTrie.proveAccountStorageRoot(
            _address,
            _storageRoot,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function proveAccountCodeHash(
        address _address,
        bytes32 _codeHash,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        view
        returns (bool)
    {
        return Lib_EthMerkleTrie.proveAccountCodeHash(
            _address,
            _codeHash,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function updateAccountState(
        address _address,
        Lib_OVMCodec.EVMAccount memory _accountState,
        Lib_OVMCodec.ProofMatrix memory _proofMatrix,
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
            _proofMatrix,
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

    function updateAccountState(
        address _address,
        Lib_OVMCodec.Account memory _accountState,
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

    function updateAccountNonce(
        address _address,
        uint256 _nonce,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        view
        returns (bytes32)
    {
        return Lib_EthMerkleTrie.updateAccountNonce(
            _address,
            _nonce,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function updateAccountBalance(
        address _address,
        uint256 _balance,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        view
        returns (bytes32)
    {
        return Lib_EthMerkleTrie.updateAccountBalance(
            _address,
            _balance,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function updateAccountStorageRoot(
        address _address,
        bytes32 _storageRoot,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        view
        returns (bytes32)
    {
        return Lib_EthMerkleTrie.updateAccountStorageRoot(
            _address,
            _storageRoot,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function updateAccountCodeHash(
        address _address,
        bytes32 _codeHash,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        view
        returns (bytes32)
    {
        return Lib_EthMerkleTrie.updateAccountCodeHash(
            _address,
            _codeHash,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }
}
