// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_StateManager } from "../../iOVM/execution/iOVM_StateManager.sol";
import { iOVM_DataTypes } from "../../iOVM/codec/iOVM_DataTypes.sol";

contract OVM_StateManager is iOVM_StateManager {
    enum ItemState {
        ITEM_UNTOUCHED,
        ITEM_LOADED,
        ITEM_CHANGED
    }

    mapping (address => iOVM_DataTypes.OVMAccount) public accounts;
    mapping (address => iOVM_DataTypes.OVMAccount) public pendingAccounts;
    mapping (address => mapping (bytes32 => bytes32)) public contractStorage;
    mapping (address => mapping (bytes32 => bool)) public verifiedContractStorage;
    mapping (bytes32 => ItemState) public itemStates;

    function putAccount(
        address _address,
        iOVM_DataTypes.OVMAccount memory _account
    )
        override
        public
    {
        accounts[_address] = _account;
    }

    function getAccount(address _address)
        override
        public
        returns (
            iOVM_DataTypes.OVMAccount memory _account
        )
    {
        return accounts[_address];
    }

    function hasAccount(
        address _address
    )
        override
        public
        returns (
            bool _exists
        )
    {
        return getAccount(_address).codeHash != bytes32(0);
    }

    function incrementAccountNonce(
        address _address
    )
        override
        public
    {
        accounts[_address].nonce += 1;
    }

    function initPendingAccount(
        address _address
    )
        override
        public
    {
        iOVM_DataTypes.OVMAccount storage account = accounts[_address];
        account.nonce = 1;
        account.codeHash = keccak256(hex'80');
    }

    function commitPendingAccount(
        address _address,
        address _ethAddress,
        bytes32 _codeHash
    )
        override
        public
    {
        iOVM_DataTypes.OVMAccount storage account = accounts[_address];
        account.ethAddress = _ethAddress;
        account.codeHash = _codeHash;
    }

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


    /*
     * Internal Functions
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
