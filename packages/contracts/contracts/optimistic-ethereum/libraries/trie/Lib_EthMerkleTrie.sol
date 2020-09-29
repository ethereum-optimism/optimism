// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_SecureMerkleTrie } from "./Lib_SecureMerkleTrie.sol";
import { Lib_OVMCodec } from "../codec/Lib_OVMCodec.sol";
import { Lib_BytesUtils } from "../utils/Lib_BytesUtils.sol";
import { Lib_RLPWriter } from "../rlp/Lib_RLPWriter.sol";
import { Lib_RLPReader } from "../rlp/Lib_RLPReader.sol";

/**
 * @title Lib_EthMerkleTrie
 */
library Lib_EthMerkleTrie {

    /**********************
     * Contract Constants *
     **********************/

    bytes constant private RLP_NULL_BYTES = hex'80';
    bytes32 constant private BYTES32_NULL = bytes32('');
    uint256 constant private UINT256_NULL = uint256(0);


    /*************************************
     * Internal Functions: Storage Slots *
     *************************************/

    /**
     * @notice Verifies a proof for the value of an account storage slot.
     * @param _address Address of the contract account.
     * @param _key Key for the storage slot.
     * @param _value Value for the storage slot.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _storageTrieWitness Inclusion proof for the specific storage
     * slot associated with the given key.
     * @param _stateTrieRoot Known root of the state trie.
     * @return `true` if the k/v pair is included, `false` otherwise.
     */
    function proveAccountStorageSlotValue(
        address _address,
        bytes32 _key,
        bytes32 _value,
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bool)
    {
        // Retrieve the current storage root.
        Lib_OVMCodec.EVMAccount memory accountState = getAccountState(
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );

        // Verify inclusion of the given k/v pair in the storage trie.
        return Lib_SecureMerkleTrie.verifyInclusionProof(
            abi.encodePacked(_key),
            abi.encodePacked(_value),
            _storageTrieWitness,
            accountState.storageRoot
        );
    }

    /**
     * @notice Updates the value for a given account storage slot.
     * @param _address Address of the contract account.
     * @param _key Key for the storage slot.
     * @param _value New value for the storage slot.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _storageTrieWitness Inclusion proof for the specific storage
     * slot associated with the given key.
     * @param _stateTrieRoot Known root of the state trie.
     * @return Root hash of the updated state trie.
     */
    function updateAccountStorageSlotValue(
        address _address,
        bytes32 _key,
        bytes32 _value,
        bytes memory _stateTrieWitness,
        bytes memory _storageTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bytes32)
    {
        // Retreive the old storage root.
        Lib_OVMCodec.EVMAccount memory accountState = getAccountState(
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );

        // Generate a new storage root.
        accountState.storageRoot = Lib_SecureMerkleTrie.update(
            abi.encodePacked(_key),
            abi.encodePacked(_value),
            _storageTrieWitness,
            accountState.storageRoot
        );

        // Update the state trie with the new storage root.
        return setAccountState(
            accountState,
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }


    /**************************************
     * Internal Functions: Account Proofs *
     *************************************/

    /**
     * @notice Verifies a proof of the current state for a given account.
     * @param _address Address of the target account.
     * @param _accountState Account state object to verify.
     * @param _proofMatrix Matrix of fields to verify or ignore.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return `true` if the given account state is valid, `false` otherwise.
     */
    function proveAccountState(
        address _address,
        Lib_OVMCodec.EVMAccount memory _accountState,
        Lib_OVMCodec.ProofMatrix memory _proofMatrix,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bool)
    {
        // Pull the current account state.
        Lib_OVMCodec.EVMAccount memory accountState = getAccountState(
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );

        // Check each provided component conditionally.
        return (
            (!_proofMatrix.checkNonce || accountState.nonce == _accountState.nonce) &&
            (!_proofMatrix.checkBalance || accountState.balance == _accountState.balance) &&
            (!_proofMatrix.checkStorageRoot || accountState.storageRoot == _accountState.storageRoot) &&
            (!_proofMatrix.checkCodeHash || accountState.codeHash == _accountState.codeHash)
        );
    }

    /**
     * @notice Verifies a proof of the current state for a given account.
     * @param _address Address of the target account.
     * @param _accountState Account state object to verify.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return `true` if the given account state is valid, `false` otherwise.
     */
    function proveAccountState(
        address _address,
        Lib_OVMCodec.EVMAccount memory _accountState,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bool)
    {
        return proveAccountState(
            _address,
            _accountState,
            Lib_OVMCodec.ProofMatrix({
                checkNonce: true,
                checkBalance: true,
                checkStorageRoot: true,
                checkCodeHash: true
            }),
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    /**
     * @notice Verifies a proof of the current state for a given account.
     * @param _address Address of the target account.
     * @param _accountState Account state object to verify.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return `true` if the given account state is valid, `false` otherwise.
     */
    function proveAccountState(
        address _address,
        Lib_OVMCodec.Account memory _accountState,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bool)
    {
        return proveAccountState(
            _address,
            Lib_OVMCodec.EVMAccount({
                nonce: _accountState.nonce,
                balance: _accountState.balance,
                storageRoot: _accountState.storageRoot,
                codeHash: _accountState.codeHash
            }),
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    /**
     * @notice Verifies a proof of the account nonce.
     * @param _address Address of the target account.
     * @param _nonce Account transaction nonce.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return `true` if the given nonce is valid, `false` otherwise.
     */
    function proveAccountNonce(
        address _address,
        uint256 _nonce,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bool)
    {
        return proveAccountState(
            _address,
            Lib_OVMCodec.EVMAccount({
                nonce: _nonce,
                balance: UINT256_NULL,
                storageRoot: BYTES32_NULL,
                codeHash: BYTES32_NULL
            }),
            Lib_OVMCodec.ProofMatrix({
                checkNonce: true,
                checkBalance: false,
                checkStorageRoot: false,
                checkCodeHash: false
            }),
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    /**
     * @notice Verifies a proof of the account balance.
     * @param _address Address of the target account.
     * @param _balance Account balance in wei.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return `true` if the given balance is valid, `false` otherwise.
     */
    function proveAccountBalance(
        address _address,
        uint256 _balance,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bool)
    {
        return proveAccountState(
            _address,
            Lib_OVMCodec.EVMAccount({
                nonce: UINT256_NULL,
                balance: _balance,
                storageRoot: BYTES32_NULL,
                codeHash: BYTES32_NULL
            }),
            Lib_OVMCodec.ProofMatrix({
                checkNonce: false,
                checkBalance: true,
                checkStorageRoot: false,
                checkCodeHash: false
            }),
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    /**
     * @notice Verifies a proof of the account storage root.
     * @param _address Address of the target account.
     * @param _storageRoot Account storage root, empty if EOA.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return `true` if the given storage root is valid, `false` otherwise.
     */
    function proveAccountStorageRoot(
        address _address,
        bytes32 _storageRoot,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bool)
    {
        return proveAccountState(
            _address,
            Lib_OVMCodec.EVMAccount({
                nonce: UINT256_NULL,
                balance: UINT256_NULL,
                storageRoot: _storageRoot,
                codeHash: BYTES32_NULL
            }),
            Lib_OVMCodec.ProofMatrix({
                checkNonce: false,
                checkBalance: false,
                checkStorageRoot: true,
                checkCodeHash: false
            }),
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    /**
     * @notice Verifies a proof of the account code hash.
     * @param _address Address of the target account.
     * @param _codeHash Account code hash, empty if EOA.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return `true` if the given code hash is valid, `false` otherwise.
     */
    function proveAccountCodeHash(
        address _address,
        bytes32 _codeHash,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bool)
    {
        return proveAccountState(
            _address,
            Lib_OVMCodec.EVMAccount({
                nonce: UINT256_NULL,
                balance: UINT256_NULL,
                storageRoot: BYTES32_NULL,
                codeHash: _codeHash
            }),
            Lib_OVMCodec.ProofMatrix({
                checkNonce: false,
                checkBalance: false,
                checkStorageRoot: false,
                checkCodeHash: true
            }),
            _stateTrieWitness,
            _stateTrieRoot
        );
    }


    /***************************************
     * Internal Functions: Account Updates *
     ***************************************/

    /**
     * @notice Updates the current state for a given account.
     * @param _address Address of the target account.
     * @param _accountState Account state to insert.
     * @param _proofMatrix Matrix of fields to update or ignore.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return Root hash of the updated state trie.
     */
    function updateAccountState(
        address _address,
        Lib_OVMCodec.EVMAccount memory _accountState,
        Lib_OVMCodec.ProofMatrix memory _proofMatrix,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bytes32)
    {
        Lib_OVMCodec.EVMAccount memory newAccountState = _accountState;

        // If the user has provided everything, don't bother pulling the
        // current account state.
        if (
            !_proofMatrix.checkNonce ||
            !_proofMatrix.checkBalance ||
            !_proofMatrix.checkStorageRoot ||
            !_proofMatrix.checkCodeHash
        ) {
            // Pull the old account state.
            Lib_OVMCodec.EVMAccount memory oldAccountState = getAccountState(
                _address,
                _stateTrieWitness,
                _stateTrieRoot
            );

            // Conditionally update elements that haven't been provided with
            // elements from the old account state.

            if (!_proofMatrix.checkNonce) {
                newAccountState.nonce = oldAccountState.nonce;
            }

            if (!_proofMatrix.checkBalance) {
                newAccountState.balance = oldAccountState.balance;
            }

            if (!_proofMatrix.checkStorageRoot) {
                newAccountState.storageRoot = oldAccountState.storageRoot;
            }

            if (!_proofMatrix.checkCodeHash) {
                newAccountState.codeHash = oldAccountState.codeHash;
            }
        }

        // Update the account state.
        return setAccountState(
            newAccountState,
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    /**
     * @notice Updates the current state for a given account.
     * @param _address Address of the target account.
     * @param _accountState Account state to insert.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return Root hash of the updated state trie.
     */
    function updateAccountState(
        address _address,
        Lib_OVMCodec.EVMAccount memory _accountState,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bytes32)
    {
        return updateAccountState(
            _address,
            _accountState,
            Lib_OVMCodec.ProofMatrix({
                checkNonce: true,
                checkBalance: true,
                checkStorageRoot: true,
                checkCodeHash: true
            }),
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    /**
     * @notice Updates the current state for a given account.
     * @param _address Address of the target account.
     * @param _accountState Account state to insert.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return Root hash of the updated state trie.
     */
    function updateAccountState(
        address _address,
        Lib_OVMCodec.Account memory _accountState,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bytes32)
    {
        return updateAccountState(
            _address,
            Lib_OVMCodec.EVMAccount({
                nonce: _accountState.nonce,
                balance: _accountState.balance,
                storageRoot: _accountState.storageRoot,
                codeHash: _accountState.codeHash
            }),
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    /**
     * @notice Updates an account nonce.
     * @param _address Address of the target account.
     * @param _nonce New account transaction nonce.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return Root hash of the updated state trie.
     */
    function updateAccountNonce(
        address _address,
        uint256 _nonce,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bytes32)
    {
        return updateAccountState(
            _address,
            Lib_OVMCodec.EVMAccount({
                nonce: _nonce,
                balance: UINT256_NULL,
                storageRoot: BYTES32_NULL,
                codeHash: BYTES32_NULL
            }),
            Lib_OVMCodec.ProofMatrix({
                checkNonce: true,
                checkBalance: false,
                checkStorageRoot: false,
                checkCodeHash: false
            }),
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    /**
     * @notice Updates an account balance.
     * @param _address Address of the target account.
     * @param _balance New account balance in wei.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return Root hash of the updated state trie.
     */
    function updateAccountBalance(
        address _address,
        uint256 _balance,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bytes32)
    {
        return updateAccountState(
            _address,
            Lib_OVMCodec.EVMAccount({
                nonce: UINT256_NULL,
                balance: _balance,
                storageRoot: BYTES32_NULL,
                codeHash: BYTES32_NULL
            }),
            Lib_OVMCodec.ProofMatrix({
                checkNonce: false,
                checkBalance: true,
                checkStorageRoot: false,
                checkCodeHash: false
            }),
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    /**
     * @notice Updates an account storage root.
     * @param _address Address of the target account.
     * @param _storageRoot New account storage root, empty if EOA.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return Root hash of the updated state trie.
     */
    function updateAccountStorageRoot(
        address _address,
        bytes32 _storageRoot,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bytes32)
    {
        return updateAccountState(
            _address,
            Lib_OVMCodec.EVMAccount({
                nonce: UINT256_NULL,
                balance: UINT256_NULL,
                storageRoot: _storageRoot,
                codeHash: BYTES32_NULL
            }),
            Lib_OVMCodec.ProofMatrix({
                checkNonce: false,
                checkBalance: false,
                checkStorageRoot: true,
                checkCodeHash: false
            }),
            _stateTrieWitness,
            _stateTrieRoot
        );
    }

    /**
     * @notice Updates an account code hash.
     * @param _address Address of the target account.
     * @param _codeHash New account code hash, empty if EOA.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return Root hash of the updated state trie.
     */
    function updateAccountCodeHash(
        address _address,
        bytes32 _codeHash,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        view
        returns (bytes32)
    {
        return updateAccountState(
            _address,
            Lib_OVMCodec.EVMAccount({
                nonce: UINT256_NULL,
                balance: UINT256_NULL,
                storageRoot: BYTES32_NULL,
                codeHash: _codeHash
            }),
            Lib_OVMCodec.ProofMatrix({
                checkNonce: false,
                checkBalance: false,
                checkStorageRoot: false,
                checkCodeHash: true
            }),
            _stateTrieWitness,
            _stateTrieRoot
        );
    }


    /*********************
     * Private Functions *
     *********************/

    /**
     * @notice Decodes an RLP-encoded account state into a useful struct.
     * @param _encodedAccountState RLP-encoded account state.
     * @return Account state struct.
     */
    function decodeAccountState(
        bytes memory _encodedAccountState
    )
        private
        view
        returns (Lib_OVMCodec.EVMAccount memory)
    {
        Lib_RLPReader.RLPItem[] memory accountState = Lib_RLPReader.toList(Lib_RLPReader.toRlpItem(_encodedAccountState));

        return Lib_OVMCodec.EVMAccount({
            nonce: Lib_RLPReader.toUint(accountState[0]),
            balance: Lib_RLPReader.toUint(accountState[1]),
            storageRoot: Lib_BytesUtils.toBytes32(Lib_RLPReader.toBytes(accountState[2])),
            codeHash: Lib_BytesUtils.toBytes32(Lib_RLPReader.toBytes(accountState[3]))
        });
    }

    /**
     * @notice RLP-encodes an account state struct.
     * @param _accountState Account state struct.
     * @return RLP-encoded account state.
     */
    function encodeAccountState(
        Lib_OVMCodec.EVMAccount memory _accountState
    )
        private
        view
        returns (bytes memory)
    {
        bytes[] memory raw = new bytes[](4);

        // Unfortunately we can't create this array outright because
        // RLPWriter.encodeList will reject fixed-size arrays. Assigning
        // index-by-index circumvents this issue.
        raw[0] = Lib_RLPWriter.encodeUint(_accountState.nonce);
        raw[1] = Lib_RLPWriter.encodeUint(_accountState.balance);
        raw[2] = _accountState.storageRoot == 0 ? RLP_NULL_BYTES : Lib_RLPWriter.encodeBytes(abi.encodePacked(_accountState.storageRoot));
        raw[3] = _accountState.codeHash == 0 ? RLP_NULL_BYTES : Lib_RLPWriter.encodeBytes(abi.encodePacked(_accountState.codeHash));

        return Lib_RLPWriter.encodeList(raw);
    }

    /**
     * @notice Retrieves the current account state and converts into a struct.
     * @param _address Account address.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     */
    function getAccountState(
        address _address,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        private
        view
        returns (Lib_OVMCodec.EVMAccount memory)
    {
        Lib_OVMCodec.EVMAccount memory DEFAULT_ACCOUNT_STATE = Lib_OVMCodec.EVMAccount({
            nonce: UINT256_NULL,
            balance: UINT256_NULL,
            storageRoot: keccak256(hex'80'),
            codeHash: keccak256(hex'')
        });

        (
            bool exists,
            bytes memory encodedAccountState
        ) = Lib_SecureMerkleTrie.get(
            abi.encodePacked(_address),
            _stateTrieWitness,
            _stateTrieRoot
        );

        return exists ? decodeAccountState(encodedAccountState) : DEFAULT_ACCOUNT_STATE;
    }

    /**
     * @notice Updates the current account state for a given address.
     * @param _accountState New account state, as a struct.
     * @param _address Account address.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return Root hash of the updated state trie.
     */
    function setAccountState(
        Lib_OVMCodec.EVMAccount memory _accountState,
        address _address,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        private
        view
        returns (bytes32)
    {
        bytes memory encodedAccountState = encodeAccountState(_accountState);

        return Lib_SecureMerkleTrie.update(
            abi.encodePacked(_address),
            encodedAccountState,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }
}