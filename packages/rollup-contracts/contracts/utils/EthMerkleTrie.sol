pragma solidity ^0.5.0;

import './MerkleTrie.sol';
import './RLPWriter.sol';
import './RLPReader.sol';
import './BytesLib.sol';

contract EthMerkleTrie is MerkleTrie {
    bytes32 constant BYTES32_NULL = bytes32('');
    uint256 constant UINT256_NULL = uint256(0);

    struct AccountState {
        uint256 nonce;
        uint256 balance;
        bytes32 storageRoot;
        bytes32 codeHash;
    }

    function proveAccountStorageSlotValue(
        address _address,
        bytes32 _key,
        bytes32 _value,
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness,
        bytes32 _stateTrieRoot
    ) public pure returns (bool) {
        bytes32 storageRoot = getStorageRoot(
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );

        return verifyInclusionProof(
            abi.encodePacked(_key),
            abi.encodePacked(_value),
            _storageTrieWitness,
            storageRoot
        );
    }

    function updateAccountStorageSlotValue(
        address _address,
        bytes32 _key,
        bytes32 _value,
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness,
        bytes32 _stateTrieRoot
    ) public pure returns (bytes32) {
        bytes32 storageRoot = getStorageRoot(
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );

        return update(
            abi.encodePacked(_key),
            abi.encodePacked(_value),
            _storageTrieWitness,
            storageRoot
        );
    }

    function proveAccountState(
        address _address,
        uint256 _nonce,
        uint256 _balance,
        bytes32 _storageRoot,
        bytes32 _codeHash,
        bool _proveNonce,
        bool _proveBalance,
        bool _proveStorageRoot,
        bool _proveCodeHash,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    ) public pure returns (bool) {
        AccountState memory accountState = getAccountState(
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );

        return (
            (!_proveNonce || accountState.nonce == _nonce) &&
            (!_proveBalance || accountState.balance == _balance) &&
            (!_proveStorageRoot || accountState.storageRoot == _storageRoot) &&
            (!_proveCodeHash || accountState.codeHash == _codeHash)
        );
    }

    function proveAccountNonce(
        address _address,
        uint256 _nonce,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    ) public pure returns (bool) {
        return proveAccountState(
            _address,
            _nonce,
            UINT256_NULL,
            BYTES32_NULL,
            BYTES32_NULL,
            true,
            false,
            false,
            false,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function proveAccountBalance(
        address _address,
        uint256 _balance,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    ) public pure returns (bool) {
        return proveAccountState(
            _address,
            UINT256_NULL,
            _balance,
            BYTES32_NULL,
            BYTES32_NULL,
            false,
            true,
            false,
            false,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function proveAccountStorageRoot(
        address _address,
        bytes32 _storageRoot,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    ) public pure returns (bool) {
        return proveAccountState(
            _address,
            UINT256_NULL,
            UINT256_NULL,
            _storageRoot,
            BYTES32_NULL,
            false,
            false,
            true,
            false,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function proveAccountCodeHash(
        address _address,
        bytes32 _codeHash,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    ) public pure returns (bool) {
        return proveAccountState(
            _address,
            UINT256_NULL,
            UINT256_NULL,
            BYTES32_NULL,
            _codeHash,
            false,
            false,
            false,
            true,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function updateAccountState(
        address _address,
        uint256 _nonce,
        uint256 _balance,
        bytes32 _storageRoot,
        bytes32 _codeHash,
        bool _updateNonce,
        bool _updateBalance,
        bool _updateStorageRoot,
        bool _updateCodeHash,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    ) public pure returns (bytes32) {
        AccountState memory newAccountState = AccountState({
            nonce: _nonce,
            balance: _balance,
            storageRoot: _storageRoot,
            codeHash: _codeHash
        });

        if (
            !_updateNonce ||
            !_updateBalance ||
            !_updateStorageRoot ||
            !_updateCodeHash
        ) {
            AccountState memory oldAccountState = getAccountState(
                _address,
                _stateTrieWitness,
                _stateTrieRoot
            );

            if (!_updateNonce) {
                newAccountState.nonce = oldAccountState.nonce;
            }

            if (!_updateBalance) {
                newAccountState.balance = oldAccountState.balance;
            }

            if (!_updateStorageRoot) {
                newAccountState.storageRoot = oldAccountState.storageRoot;
            }

            if (!_updateCodeHash) {
                newAccountState.codeHash = oldAccountState.codeHash;
            }
        }

        return setAccountState(
            newAccountState,
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function updateAccountNonce(
        address _address,
        uint256 _nonce,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    ) public pure returns (bytes32) {
        return updateAccountState(
            _address,
            _nonce,
            UINT256_NULL,
            BYTES32_NULL,
            BYTES32_NULL,
            true,
            false,
            false,
            false,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function updateAccountBalance(
        address _address,
        uint256 _balance,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    ) public pure returns (bytes32) {
        return updateAccountState(
            _address,
            UINT256_NULL,
            _balance,
            BYTES32_NULL,
            BYTES32_NULL,
            false,
            true,
            false,
            false,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function updateAccountStorageRoot(
        address _address,
        bytes32 _storageRoot,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    ) public pure returns (bytes32) {
        return updateAccountState(
            _address,
            UINT256_NULL,
            UINT256_NULL,
            _storageRoot,
            BYTES32_NULL,
            false,
            false,
            true,
            false,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function updateAccountCodeHash(
        address _address,
        bytes32 _codeHash,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    ) public pure returns (bytes32) {
        return updateAccountState(
            _address,
            UINT256_NULL,
            UINT256_NULL,
            BYTES32_NULL,
            _codeHash,
            false,
            false,
            false,
            true,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }


    /*
     * Internal Functions
     */

    function decodeAccountState(
        bytes memory _encodedAccountState
    ) internal pure returns (AccountState memory) {
        RLPReader.RLPItem[] memory accountState = RLPReader.toList(RLPReader.toRlpItem(_encodedAccountState));

        return AccountState({
            nonce: RLPReader.toUint(accountState[0]),
            balance: RLPReader.toUint(accountState[1]),
            storageRoot: BytesLib.toBytes32(RLPReader.toBytes(accountState[2])),
            codeHash: BytesLib.toBytes32(RLPReader.toBytes(accountState[3]))
        });
    }

    function encodeAccountState(
        AccountState memory _accountState
    ) internal pure returns (bytes memory) {
        bytes[] memory raw = new bytes[](4);

        raw[0] = RLPWriter.encodeUint(_accountState.nonce);
        raw[1] = RLPWriter.encodeUint(_accountState.balance);
        raw[2] = RLPWriter.encodeBytes(abi.encodePacked(_accountState.storageRoot));
        raw[3] = RLPWriter.encodeBytes(abi.encodePacked(_accountState.codeHash));

        return RLPWriter.encodeList(raw);
    }

    function getAccountState(
        address _address,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    ) internal pure returns (AccountState memory) {
        bytes memory encodedAccountState = get(
            abi.encodePacked(_address),
            _stateTrieWitness,
            _stateTrieRoot
        );
        return decodeAccountState(encodedAccountState);
    }

    function setAccountState(
        AccountState memory _accountState,
        address _address,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    ) internal pure returns (bytes32) {
        bytes memory encodedAccountState = encodeAccountState(_accountState);

        return update(
            abi.encodePacked(_address),
            encodedAccountState,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    function getStorageRoot(
        address _address,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    ) internal pure returns (bytes32) {
        AccountState memory accountState = getAccountState(
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );

        return accountState.storageRoot;
    }
}