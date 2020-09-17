// SPDX-License-Identifier: UNLICENSED
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

    /****************************************
     * Contract Variables: Internal Storage *
     ****************************************/

    mapping (address => Lib_OVMCodec.Account) internal accounts;
    mapping (address => mapping (bytes32 => bytes32)) internal contractStorage;
    mapping (address => mapping (bytes32 => bool)) internal verifiedContractStorage;
    mapping (bytes32 => ItemState) internal itemStates;


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
    {
        accounts[_address] = _account;
    }

    /**
     * Retrieves an account from the state.
     * @param _address Address of the account to retrieve.
     * @return _account Account for the given address.
     */
    function getAccount(address _address)
        override
        public
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
        returns (
            bool _exists
        )
    {
        return accounts[_address].codeHash != bytes32(0);
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
        returns (
            address _ethAddress
        )
    {
        return accounts[_address].ethAddress;
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
    {
        Lib_OVMCodec.Account storage account = accounts[_address];
        account.nonce = 1;
        account.codeHash = keccak256(hex'80');
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
        returns (
            bool _wasAccountAlreadyLoaded
        )
    {
        return _testItemState(
            keccak256(abi.encodePacked(_address)),
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
        returns (
            bool _wasAccountAlreadyChanged
        )
    {
        return _testItemState(
            keccak256(abi.encodePacked(_address)),
            ItemState.ITEM_CHANGED
        );
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
    {
        contractStorage[_contract][_key] = _value;
        verifiedContractStorage[_contract][_key] = true;
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
        returns (
            bytes32 _value
        )
    {
        return contractStorage[_contract][_key];
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
        returns (
            bool _exists
        )
    {
        return verifiedContractStorage[_contract][_key];
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
        returns (
            bool _wasContractStorageAlreadyLoaded
        )
    {
        return _testItemState(
            keccak256(abi.encodePacked(_contract, _key)),
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
        returns (
            bool _wasContractStorageAlreadyChanged
        )
    {
        return _testItemState(
            keccak256(abi.encodePacked(_contract, _key)),
            ItemState.ITEM_CHANGED
        );
    }


    /**********************
     * Internal Functions *
     **********************/

    /**
     * Checks whether an item is in a particular state (ITEM_LOADED or ITEM_CHANGED) and sets the
     * item to the provided state if not.
     * @param _item 32 byte item ID to check.
     * @param _minItemState Minumum state that must be satisfied by the item.
     * @return _wasItemState Whether or not the item was already in the state.
     */
    function _testItemState(
        bytes32 _item,
        ItemState _minItemState
    )
        internal
        returns (
            bool _wasItemState
        )
    {
        ItemState itemState = itemStates[_item];
        bool wasItemState = itemState >= _minItemState;

        if (wasItemState == false) {
            itemStates[_item] = _minItemState;
        }

        return wasItemState;
    }
}
