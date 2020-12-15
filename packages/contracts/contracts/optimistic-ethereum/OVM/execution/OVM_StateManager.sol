// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

/* Interface Imports */
import { iOVM_StateManager } from "../../iOVM/execution/iOVM_StateManager.sol";

/**
 * @title OVM_StateManager
 */
contract OVM_StateManager is iOVM_StateManager {

    /**********************
     * Contract Constants *
     **********************/

    bytes32 constant internal EMPTY_ACCOUNT_STORAGE_ROOT = 0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421;
    bytes32 constant internal EMPTY_ACCOUNT_CODE_HASH =    0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470;
    bytes32 constant internal STORAGE_XOR_VALUE =          0xFEEDFACECAFEBEEFFEEDFACECAFEBEEFFEEDFACECAFEBEEFFEEDFACECAFEBEEF;


    /*******************************************
     * Contract Variables: Contract References *
     *******************************************/

    address override public owner;
    address override public ovmExecutionManager;


    /****************************************
     * Contract Variables: Internal Storage *
     ****************************************/

    mapping (address => Lib_OVMCodec.Account) internal accounts;
    mapping (address => mapping (bytes32 => bytes32)) internal contractStorage;
    mapping (address => mapping (bytes32 => bool)) internal verifiedContractStorage;
    mapping (bytes32 => ItemState) internal itemStates;
    uint256 internal totalUncommittedAccounts;
    uint256 internal totalUncommittedContractStorage;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _owner Address of the owner of this contract.
     */
    constructor(
        address _owner
    ) {
        owner = _owner;
    }


    /**********************
     * Function Modifiers *
     **********************/

    /**
     * Simple authentication, this contract should only be accessible to the owner or to the
     * OVM_ExecutionManager during the transaction execution process.
     */
    modifier authenticated() {
        require(
            msg.sender == owner || msg.sender == ovmExecutionManager,
            "Function can only be called by authenticated addresses"
        );
        _;
    }

    /***************************
     * Public Functions: Misc *
     ***************************/


    function isAuthenticated(
        address _address
    )
        override
        public
        view
        returns (bool)
    {
        return (_address == owner || _address == ovmExecutionManager);
    }

    /***************************
     * Public Functions: Setup *
     ***************************/

    /**
     * Sets the address of the OVM_ExecutionManager.
     * @param _ovmExecutionManager Address of the OVM_ExecutionManager.
     */
    function setExecutionManager(
        address _ovmExecutionManager
    )
        override
        public
        authenticated
    {
        ovmExecutionManager = _ovmExecutionManager;
    }


    /************************************
     * Public Functions: Account Access *
     ************************************/

    /**
     * Inserts an account into the state.
     * @param _address Address of the account to insert.
     * @param _account Account to insert for the given address.
     */
    function putAccount(
        address _address,
        Lib_OVMCodec.Account memory _account
    )
        override
        public
        authenticated
    {
        accounts[_address] = _account;
    }

    /**
     * Marks an account as empty.
     * @param _address Address of the account to mark.
     */
    function putEmptyAccount(
        address _address
    )
        override
        public
        authenticated
    {
        Lib_OVMCodec.Account storage account = accounts[_address];
        account.storageRoot = EMPTY_ACCOUNT_STORAGE_ROOT;
        account.codeHash = EMPTY_ACCOUNT_CODE_HASH;
    }

    /**
     * Retrieves an account from the state.
     * @param _address Address of the account to retrieve.
     * @return _account Account for the given address.
     */
    function getAccount(address _address)
        override
        public
        view
        returns (
            Lib_OVMCodec.Account memory _account
        )
    {
        return accounts[_address];
    }

    /**
     * Checks whether the state has a given account.
     * @param _address Address of the account to check.
     * @return _exists Whether or not the state has the account.
     */
    function hasAccount(
        address _address
    )
        override
        public
        view
        returns (
            bool _exists
        )
    {
        return accounts[_address].codeHash != bytes32(0);
    }

    /**
     * Checks whether the state has a given known empty account.
     * @param _address Address of the account to check.
     * @return _exists Whether or not the state has the empty account.
     */
    function hasEmptyAccount(
        address _address
    )
        override
        public
        view
        returns (
            bool _exists
        )
    {
        return (
            accounts[_address].codeHash == EMPTY_ACCOUNT_CODE_HASH
            && accounts[_address].nonce == 0
        );
    }

    /**
     * Sets the nonce of an account.
     * @param _address Address of the account to modify.
     * @param _nonce New account nonce.
     */
    function setAccountNonce(
        address _address,
        uint256 _nonce
    )
        override
        public
        authenticated
    {
        accounts[_address].nonce = _nonce;
    }

    /**
     * Gets the nonce of an account.
     * @param _address Address of the account to access.
     * @return _nonce Nonce of the account.
     */
    function getAccountNonce(
        address _address
    )
        override
        public
        view
        returns (
            uint256 _nonce
        )
    {
        return accounts[_address].nonce;
    }

    /**
     * Retrieves the Ethereum address of an account.
     * @param _address Address of the account to access.
     * @return _ethAddress Corresponding Ethereum address.
     */
    function getAccountEthAddress(
        address _address
    )
        override
        public
        view
        returns (
            address _ethAddress
        )
    {
        return accounts[_address].ethAddress;
    }

    /**
     * Retrieves the storage root of an account.
     * @param _address Address of the account to access.
     * @return _storageRoot Corresponding storage root.
     */
    function getAccountStorageRoot(
        address _address
    )
        override
        public
        view
        returns (
            bytes32 _storageRoot
        )
    {
        return accounts[_address].storageRoot;
    }

    /**
     * Initializes a pending account (during CREATE or CREATE2) with the default values.
     * @param _address Address of the account to initialize.
     */
    function initPendingAccount(
        address _address
    )
        override
        public
        authenticated
    {
        Lib_OVMCodec.Account storage account = accounts[_address];
        account.nonce = 1;
        account.storageRoot = EMPTY_ACCOUNT_STORAGE_ROOT;
        account.codeHash = EMPTY_ACCOUNT_CODE_HASH;
        account.isFresh = true;
    }

    /**
     * Finalizes the creation of a pending account (during CREATE or CREATE2).
     * @param _address Address of the account to finalize.
     * @param _ethAddress Address of the account's associated contract on Ethereum.
     * @param _codeHash Hash of the account's code.
     */
    function commitPendingAccount(
        address _address,
        address _ethAddress,
        bytes32 _codeHash
    )
        override
        public
        authenticated
    {
        Lib_OVMCodec.Account storage account = accounts[_address];
        account.ethAddress = _ethAddress;
        account.codeHash = _codeHash;
    }

    /**
     * Checks whether an account has already been retrieved, and marks it as retrieved if not.
     * @param _address Address of the account to check.
     * @return _wasAccountAlreadyLoaded Whether or not the account was already loaded.
     */
    function testAndSetAccountLoaded(
        address _address
    )
        override
        public
        authenticated
        returns (
            bool _wasAccountAlreadyLoaded
        )
    {
        return _testAndSetItemState(
            _getItemHash(_address),
            ItemState.ITEM_LOADED
        );
    }

    /**
     * Checks whether an account has already been modified, and marks it as modified if not.
     * @param _address Address of the account to check.
     * @return _wasAccountAlreadyChanged Whether or not the account was already modified.
     */
    function testAndSetAccountChanged(
        address _address
    )
        override
        public
        authenticated
        returns (
            bool _wasAccountAlreadyChanged
        )
    {
        return _testAndSetItemState(
            _getItemHash(_address),
            ItemState.ITEM_CHANGED
        );
    }

    /**
     * Attempts to mark an account as committed.
     * @param _address Address of the account to commit.
     * @return _wasAccountCommitted Whether or not the account was committed.
     */
    function commitAccount(
        address _address
    )
        override
        public
        authenticated
        returns (
            bool _wasAccountCommitted
        )
    {
        bytes32 item = _getItemHash(_address);
        if (itemStates[item] != ItemState.ITEM_CHANGED) {
            return false;
        }

        itemStates[item] = ItemState.ITEM_COMMITTED;
        totalUncommittedAccounts -= 1;

        return true;
    }

    /**
     * Increments the total number of uncommitted accounts.
     */
    function incrementTotalUncommittedAccounts()
        override
        public
        authenticated
    {
        totalUncommittedAccounts += 1;
    }

    /**
     * Gets the total number of uncommitted accounts.
     * @return _total Total uncommitted accounts.
     */
    function getTotalUncommittedAccounts()
        override
        public
        view
        returns (
            uint256 _total
        )
    {
        return totalUncommittedAccounts;
    }

    /**
     * Checks whether a given account was changed during execution.
     * @param _address Address to check.
     * @return Whether or not the account was changed.
     */
    function wasAccountChanged(
        address _address
    )
        override
        public
        view
        returns (
            bool
        )
    {
        bytes32 item = _getItemHash(_address);
        return itemStates[item] >= ItemState.ITEM_CHANGED;
    }

    /**
     * Checks whether a given account was committed after execution.
     * @param _address Address to check.
     * @return Whether or not the account was committed.
     */
    function wasAccountCommitted(
        address _address
    )
        override
        public
        view
        returns (
            bool
        )
    {
        bytes32 item = _getItemHash(_address);
        return itemStates[item] >= ItemState.ITEM_COMMITTED;
    }


    /************************************
     * Public Functions: Storage Access *
     ************************************/

    /**
     * Changes a contract storage slot value.
     * @param _contract Address of the contract to modify.
     * @param _key 32 byte storage slot key.
     * @param _value 32 byte storage slot value.
     */
    function putContractStorage(
        address _contract,
        bytes32 _key,
        bytes32 _value
    )
        override
        public
        authenticated
    {
        // A hilarious optimization. `SSTORE`ing a value of `bytes32(0)` is common enough that it's
        // worth populating this with a non-zero value in advance (during the fraud proof
        // initialization phase) to cut the execution-time cost down to 5000 gas.
        contractStorage[_contract][_key] = _value ^ STORAGE_XOR_VALUE;

        // Only used when initially populating the contract storage. OVM_ExecutionManager will
        // perform a `hasContractStorage` INVALID_STATE_ACCESS check before putting any contract
        // storage because writing to zero when the actual value is nonzero causes a gas
        // discrepancy. Could be moved into a new `putVerifiedContractStorage` function, or
        // something along those lines.
        if (verifiedContractStorage[_contract][_key] == false) {
            verifiedContractStorage[_contract][_key] = true;
        }
    }

    /**
     * Retrieves a contract storage slot value.
     * @param _contract Address of the contract to access.
     * @param _key 32 byte storage slot key.
     * @return _value 32 byte storage slot value.
     */
    function getContractStorage(
        address _contract,
        bytes32 _key
    )
        override
        public
        view
        returns (
            bytes32 _value
        )
    {
        // Storage XOR system doesn't work for newly created contracts that haven't set this
        // storage slot value yet.
        if (
            verifiedContractStorage[_contract][_key] == false
            && accounts[_contract].isFresh
        ) {
            return bytes32(0);
        }

        // See `putContractStorage` for more information about the XOR here.
        return contractStorage[_contract][_key] ^ STORAGE_XOR_VALUE;
    }

    /**
     * Checks whether a contract storage slot exists in the state.
     * @param _contract Address of the contract to access.
     * @param _key 32 byte storage slot key.
     * @return _exists Whether or not the key was set in the state.
     */
    function hasContractStorage(
        address _contract,
        bytes32 _key
    )
        override
        public
        view
        returns (
            bool _exists
        )
    {
        return verifiedContractStorage[_contract][_key] || accounts[_contract].isFresh;
    }

    /**
     * Checks whether a storage slot has already been retrieved, and marks it as retrieved if not.
     * @param _contract Address of the contract to check.
     * @param _key 32 byte storage slot key.
     * @return _wasContractStorageAlreadyLoaded Whether or not the slot was already loaded.
     */
    function testAndSetContractStorageLoaded(
        address _contract,
        bytes32 _key
    )
        override
        public
        authenticated
        returns (
            bool _wasContractStorageAlreadyLoaded
        )
    {
        return _testAndSetItemState(
            _getItemHash(_contract, _key),
            ItemState.ITEM_LOADED
        );
    }

    /**
     * Checks whether a storage slot has already been modified, and marks it as modified if not.
     * @param _contract Address of the contract to check.
     * @param _key 32 byte storage slot key.
     * @return _wasContractStorageAlreadyChanged Whether or not the slot was already modified.
     */
    function testAndSetContractStorageChanged(
        address _contract,
        bytes32 _key
    )
        override
        public
        authenticated
        returns (
            bool _wasContractStorageAlreadyChanged
        )
    {
        return _testAndSetItemState(
            _getItemHash(_contract, _key),
            ItemState.ITEM_CHANGED
        );
    }

    /**
     * Attempts to mark a storage slot as committed.
     * @param _contract Address of the account to commit.
     * @param _key 32 byte slot key to commit.
     * @return _wasContractStorageCommitted Whether or not the slot was committed.
     */
    function commitContractStorage(
        address _contract,
        bytes32 _key
    )
        override
        public
        authenticated
        returns (
            bool _wasContractStorageCommitted
        )
    {
        bytes32 item = _getItemHash(_contract, _key);
        if (itemStates[item] != ItemState.ITEM_CHANGED) {
            return false;
        }

        itemStates[item] = ItemState.ITEM_COMMITTED;
        totalUncommittedContractStorage -= 1;

        return true;
    }

    /**
     * Increments the total number of uncommitted storage slots.
     */
    function incrementTotalUncommittedContractStorage()
        override
        public
        authenticated
    {
        totalUncommittedContractStorage += 1;
    }

    /**
     * Gets the total number of uncommitted storage slots.
     * @return _total Total uncommitted storage slots.
     */
    function getTotalUncommittedContractStorage()
        override
        public
        view
        returns (
            uint256 _total
        )
    {
        return totalUncommittedContractStorage;
    }

    /**
     * Checks whether a given storage slot was changed during execution.
     * @param _contract Address to check.
     * @param _key Key of the storage slot to check.
     * @return Whether or not the storage slot was changed.
     */
    function wasContractStorageChanged(
        address _contract,
        bytes32 _key
    )
        override
        public
        view
        returns (
            bool
        )
    {
        bytes32 item = _getItemHash(_contract, _key);
        return itemStates[item] >= ItemState.ITEM_CHANGED;
    }

    /**
     * Checks whether a given storage slot was committed after execution.
     * @param _contract Address to check.
     * @param _key Key of the storage slot to check.
     * @return Whether or not the storage slot was committed.
     */
    function wasContractStorageCommitted(
        address _contract,
        bytes32 _key
    )
        override
        public
        view
        returns (
            bool
        )
    {
        bytes32 item = _getItemHash(_contract, _key);
        return itemStates[item] >= ItemState.ITEM_COMMITTED;
    }


    /**********************
     * Internal Functions *
     **********************/

    /**
     * Generates a unique hash for an address.
     * @param _address Address to generate a hash for.
     * @return Unique hash for the given address.
     */
    function _getItemHash(
        address _address
    )
        internal
        pure
        returns (
            bytes32
        )
    {
        return keccak256(abi.encodePacked(_address));
    }

    /**
     * Generates a unique hash for an address/key pair.
     * @param _contract Address to generate a hash for.
     * @param _key Key to generate a hash for.
     * @return Unique hash for the given pair.
     */
    function _getItemHash(
        address _contract,
        bytes32 _key
    )
        internal
        pure
        returns (
            bytes32
        )
    {
        return keccak256(abi.encodePacked(
            _contract,
            _key
        ));
    }

    /**
     * Checks whether an item is in a particular state (ITEM_LOADED or ITEM_CHANGED) and sets the
     * item to the provided state if not.
     * @param _item 32 byte item ID to check.
     * @param _minItemState Minimum state that must be satisfied by the item.
     * @return _wasItemState Whether or not the item was already in the state.
     */
    function _testAndSetItemState(
        bytes32 _item,
        ItemState _minItemState
    )
        internal
        returns (
            bool _wasItemState
        )
    {
        bool wasItemState = itemStates[_item] >= _minItemState;

        if (wasItemState == false) {
            itemStates[_item] = _minItemState;
        }

        return wasItemState;
    }
}
