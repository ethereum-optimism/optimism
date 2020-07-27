pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { MerkleTrie } from "./MerkleTrie.sol";
import { RLPWriter } from "./RLPWriter.sol";
import { RLPReader } from "./RLPReader.sol";
import { BytesLib } from "./BytesLib.sol";
import { DataTypes } from "./DataTypes.sol";

/**
 * @notice Convenience wrapper for ETH-related trie operations.
 */
library EthMerkleTrie {
    /*
     * Contract Constants
     */

    bytes32 constant private BYTES32_NULL = bytes32('');
    uint256 constant private UINT256_NULL = uint256(0);
    bytes constant private RLP_NULL_BYTES = hex'80';


    /*
     * Internal Functions
     */

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
        pure
        returns (bool)
    {
        // Retrieve the current storage root.
        DataTypes.AccountState memory accountState = getAccountState(
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );

        // Verify inclusion of the given k/v pair in the storage trie.
        return MerkleTrie.verifyInclusionProof(
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
        pure
        returns (bytes32)
    {
        // Retreive the old storage root.
        DataTypes.AccountState memory accountState = getAccountState(
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );

        // Generate a new storage root.
        accountState.storageRoot = MerkleTrie.update(
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
        DataTypes.AccountState memory _accountState,
        DataTypes.ProofMatrix memory _proofMatrix,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        pure
        returns (bool)
    {
        // Pull the current account state.
        DataTypes.AccountState memory accountState = getAccountState(
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
        pure
        returns (bool)
    {
        return proveAccountState(
            _address,
            DataTypes.AccountState({
                nonce: _nonce,
                balance: UINT256_NULL,
                storageRoot: BYTES32_NULL,
                codeHash: BYTES32_NULL
            }),
            DataTypes.ProofMatrix({
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
        pure
        returns (bool)
    {
        return proveAccountState(
            _address,
            DataTypes.AccountState({
                nonce: UINT256_NULL,
                balance: _balance,
                storageRoot: BYTES32_NULL,
                codeHash: BYTES32_NULL
            }),
            DataTypes.ProofMatrix({
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
        pure
        returns (bool)
    {
        return proveAccountState(
            _address,
            DataTypes.AccountState({
                nonce: UINT256_NULL,
                balance: UINT256_NULL,
                storageRoot: _storageRoot,
                codeHash: BYTES32_NULL
            }),
            DataTypes.ProofMatrix({
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
        pure
        returns (bool)
    {
        return proveAccountState(
            _address,
            DataTypes.AccountState({
                nonce: UINT256_NULL,
                balance: UINT256_NULL,
                storageRoot: BYTES32_NULL,
                codeHash: _codeHash
            }),
            DataTypes.ProofMatrix({
                checkNonce: false,
                checkBalance: false,
                checkStorageRoot: false,
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
     * @param _proofMatrix Matrix of fields to update or ignore.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return Root hash of the updated state trie.
     */
    function updateAccountState(
        address _address,
        DataTypes.AccountState memory _accountState,
        DataTypes.ProofMatrix memory _proofMatrix,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        internal
        pure
        returns (bytes32)
    {
        DataTypes.AccountState memory newAccountState = _accountState;

        // If the user has provided everything, don't bother pulling the
        // current account state.
        if (
            !_proofMatrix.checkNonce ||
            !_proofMatrix.checkBalance ||
            !_proofMatrix.checkStorageRoot ||
            !_proofMatrix.checkCodeHash
        ) {
            // Pull the old account state.
            DataTypes.AccountState memory oldAccountState = getAccountState(
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
        pure
        returns (bytes32)
    {
        return updateAccountState(
            _address,
            DataTypes.AccountState({
                nonce: _nonce,
                balance: UINT256_NULL,
                storageRoot: BYTES32_NULL,
                codeHash: BYTES32_NULL
            }),
            DataTypes.ProofMatrix({
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
        pure
        returns (bytes32)
    {
        return updateAccountState(
            _address,
            DataTypes.AccountState({
                nonce: UINT256_NULL,
                balance: _balance,
                storageRoot: BYTES32_NULL,
                codeHash: BYTES32_NULL
            }),
            DataTypes.ProofMatrix({
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
        pure
        returns (bytes32)
    {
        return updateAccountState(
            _address,
            DataTypes.AccountState({
                nonce: UINT256_NULL,
                balance: UINT256_NULL,
                storageRoot: _storageRoot,
                codeHash: BYTES32_NULL
            }),
            DataTypes.ProofMatrix({
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
        pure
        returns (bytes32)
    {
        return updateAccountState(
            _address,
            DataTypes.AccountState({
                nonce: UINT256_NULL,
                balance: UINT256_NULL,
                storageRoot: BYTES32_NULL,
                codeHash: _codeHash
            }),
            DataTypes.ProofMatrix({
                checkNonce: false,
                checkBalance: false,
                checkStorageRoot: false,
                checkCodeHash: true
            }),
            _stateTrieWitness,
            _stateTrieRoot
        );
    }


    /*
     * Private Functions
     */

    /**
     * @notice Decodes an RLP-encoded account state into a useful struct.
     * @param _encodedAccountState RLP-encoded account state.
     * @return Account state struct.
     */
    function decodeAccountState(
        bytes memory _encodedAccountState
    )
        private
        pure
        returns (DataTypes.AccountState memory)
    {
        RLPReader.RLPItem[] memory accountState = RLPReader.toList(RLPReader.toRlpItem(_encodedAccountState));

        return DataTypes.AccountState({
            nonce: RLPReader.toUint(accountState[0]),
            balance: RLPReader.toUint(accountState[1]),
            storageRoot: BytesLib.toBytes32(RLPReader.toBytes(accountState[2])),
            codeHash: BytesLib.toBytes32(RLPReader.toBytes(accountState[3]))
        });
    }

    /**
     * @notice RLP-encodes an account state struct.
     * @param _accountState Account state struct.
     * @return RLP-encoded account state.
     */
    function encodeAccountState(
        DataTypes.AccountState memory _accountState
    )
        private
        pure
        returns (bytes memory)
    {
        bytes[] memory raw = new bytes[](4);

        // Unfortunately we can't create this array outright because
        // RLPWriter.encodeList will reject fixed-size arrays. Assigning
        // index-by-index circumvents this issue.
        raw[0] = RLPWriter.encodeUint(_accountState.nonce);
        raw[1] = RLPWriter.encodeUint(_accountState.balance);
        raw[2] = _accountState.storageRoot == 0 ? RLP_NULL_BYTES : RLPWriter.encodeBytes(abi.encodePacked(_accountState.storageRoot));
        raw[3] = _accountState.codeHash == 0 ? RLP_NULL_BYTES : RLPWriter.encodeBytes(abi.encodePacked(_accountState.codeHash));

        return RLPWriter.encodeList(raw);
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
        pure
        returns (DataTypes.AccountState memory)
    {
        DataTypes.AccountState memory DEFAULT_ACCOUNT_STATE = DataTypes.AccountState({
            nonce: UINT256_NULL,
            balance: UINT256_NULL,
            storageRoot: keccak256(hex'80'),
            codeHash: keccak256(hex'')
        });

        (
            bool exists,
            bytes memory encodedAccountState
        ) = MerkleTrie.get(
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
        DataTypes.AccountState memory _accountState,
        address _address,
        bytes memory _stateTrieWitness,
        bytes32 _stateTrieRoot
    )
        private
        pure
        returns (bytes32)
    {
        bytes memory encodedAccountState = encodeAccountState(_accountState);

        return MerkleTrie.update(
            abi.encodePacked(_address),
            encodedAccountState,
            _stateTrieWitness,
            _stateTrieRoot
        );
    }
}