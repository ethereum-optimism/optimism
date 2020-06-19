pragma solidity ^0.5.0;

import './MerkleTrie.sol';
import './RLPWriter.sol';
import './RLPReader.sol';
import './BytesLib.sol';

/**
 * @notice Convenience wrapper for ETH-related trie operations.
 */
contract EthMerkleTrie is MerkleTrie {
    bytes32 constant BYTES32_NULL = bytes32('');
    uint256 constant UINT256_NULL = uint256(0);

    struct AccountState {
        uint256 nonce;
        uint256 balance;
        bytes32 storageRoot;
        bytes32 codeHash;
    }


    /*
     * Public Functions
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
    ) public pure returns (bool) {
        // Retrieve the current storage root.
        AccountState memory accountState = getAccountState(
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );

        // Verify inclusion of the given k/v pair in the storage trie.
        return verifyInclusionProof(
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
    ) public pure returns (bytes32) {
        // Retreive the old storage root.
        AccountState memory accountState = getAccountState(
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );

        // Generate a new storage root.
        accountState.storageRoot = update(
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
     * @param _nonce Account transaction nonce.
     * @param _balance Account balance in wei.
     * @param _storageRoot Account storage root, empty if EOA.
     * @param _codeHash Account code hash, empty if EOA.
     * @param _proveNonce Whether or not to prove the nonce.
     * @param _proveBalance Whether or not to prove the balance.
     * @param _proveStorageRoot Whether or not to prove the storage root.
     * @param _proveCodeHash Whether or not to prove the code hash.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return `true` if the given account state is valid, `false` otherwise.
     */
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
        // Pull the current account state.
        AccountState memory accountState = getAccountState(
            _address,
            _stateTrieWitness,
            _stateTrieRoot
        );

        // Check each provided component conditionally.
        return (
            (!_proveNonce || accountState.nonce == _nonce) &&
            (!_proveBalance || accountState.balance == _balance) &&
            (!_proveStorageRoot || accountState.storageRoot == _storageRoot) &&
            (!_proveCodeHash || accountState.codeHash == _codeHash)
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

    /**
     * @notice Updates the current state for a given account.
     * @param _address Address of the target account.
     * @param _nonce New account transaction nonce.
     * @param _balance New account balance in wei.
     * @param _storageRoot New account storage root, empty if EOA.
     * @param _codeHash New account code hash, empty if EOA.
     * @param _updateNonce Whether or not to update the nonce.
     * @param _updateBalance Whether or not to update the balance.
     * @param _updateStorageRoot Whether or not to update the storage root.
     * @param _updateCodeHash Whether or not to update the code hash.
     * @param _stateTrieWitness Inclusion proof for the account state within
     * the state trie.
     * @param _stateTrieRoot Known root of the state trie.
     * @return Root hash of the updated state trie.
     */
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
        // Create a struct for the new account state.
        AccountState memory newAccountState = AccountState({
            nonce: _nonce,
            balance: _balance,
            storageRoot: _storageRoot,
            codeHash: _codeHash
        });

        // If the user has provided everything, don't bother pulling the
        // current account state.
        if (
            !_updateNonce ||
            !_updateBalance ||
            !_updateStorageRoot ||
            !_updateCodeHash
        ) {
            // Pull the old account state.
            AccountState memory oldAccountState = getAccountState(
                _address,
                _stateTrieWitness,
                _stateTrieRoot
            );

            // Conditionally update elements that haven't been provided with
            // elements from the old account state.

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

    /**
     * @notice Decodes an RLP-encoded account state into a useful struct.
     * @param _encodedAccountState RLP-encoded account state.
     * @return Account state struct.
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

    /**
     * @notice RLP-encodes an account state struct.
     * @param _accountState Account state struct.
     * @return RLP-encoded account state.
     */
    function encodeAccountState(
        AccountState memory _accountState
    ) internal pure returns (bytes memory) {
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
    ) internal pure returns (AccountState memory) {
        bytes memory encodedAccountState = get(
            abi.encodePacked(_address),
            _stateTrieWitness,
            _stateTrieRoot
        );

        return decodeAccountState(encodedAccountState);
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
}