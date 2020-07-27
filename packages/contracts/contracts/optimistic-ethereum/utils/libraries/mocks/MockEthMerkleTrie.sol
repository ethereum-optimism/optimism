pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { DataTypes } from "../DataTypes.sol";
import { EthMerkleTrie } from "../EthMerkleTrie.sol";

contract MockEthMerkleTrie {
    function proveAccountStorageSlotValue(
        address _address,
        bytes32 _key,
        bytes32 _value,
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        pure
        returns (bool)
    {
        return EthMerkleTrie.proveAccountStorageSlotValue(
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
        pure
        returns (bytes32)
    {
        return EthMerkleTrie.updateAccountStorageSlotValue(
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
        DataTypes.AccountState memory _accountState,
        DataTypes.ProofMatrix memory _proofMatrix,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        pure
        returns (bool)
    {
        return EthMerkleTrie.proveAccountState(
            _address,
            _accountState,
            _proofMatrix,
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
        pure
        returns (bool)
    {
        return EthMerkleTrie.proveAccountNonce(
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
        pure
        returns (bool)
    {
        return EthMerkleTrie.proveAccountBalance(
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
        pure
        returns (bool)
    {
        return EthMerkleTrie.proveAccountStorageRoot(
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
        pure
        returns (bool)
    {
        return EthMerkleTrie.proveAccountCodeHash(
            _address,
            _codeHash,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function updateAccountState(
        address _address,
        DataTypes.AccountState memory _accountState,
        DataTypes.ProofMatrix memory _proofMatrix,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        public
        pure
        returns (bytes32)
    {
        return EthMerkleTrie.updateAccountState(
            _address,
            _accountState,
            _proofMatrix,
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
        pure
        returns (bytes32)
    {
        return EthMerkleTrie.updateAccountNonce(
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
        pure
        returns (bytes32)
    {
        return EthMerkleTrie.updateAccountBalance(
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
        pure
        returns (bytes32)
    {
        return EthMerkleTrie.updateAccountStorageRoot(
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
        pure
        returns (bytes32)
    {
        return EthMerkleTrie.updateAccountCodeHash(
            _address,
            _codeHash,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }
}