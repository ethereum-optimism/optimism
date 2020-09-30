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
        // Pull the current account state.
        Lib_OVMCodec.EVMAccount memory accountState = getAccountState(
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );

        return (
            accountState.nonce == _accountState.nonce
            && accountState.balance == _accountState.balance
            && accountState.storageRoot == _accountState.storageRoot
            && accountState.codeHash == _accountState.codeHash
        );
    }


    /***************************************
     * Internal Functions: Account Updates *
     ***************************************/

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
        return setAccountState(
            _accountState,
            _address,
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

        // TODO: Needs "single node root hash" logic.
        (
            bool exists,
            bytes memory encodedAccountState
        ) = Lib_SecureMerkleTrie.get(
            abi.encodePacked(_address),
            _stateTrieWitness,
            _stateTrieRoot
        );

        // TODO: Must fix this logic, in the next PR.
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