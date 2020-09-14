// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_DataTypes } from "../codec/iOVM_DataTypes.sol";

interface iOVM_StateManager {
    function putAccount(address _address, iOVM_DataTypes.OVMAccount memory _account) external;
    function getAccount(address _address) external returns (iOVM_DataTypes.OVMAccount memory _account);
    function hasAccount(address _address) external returns (bool _exists);
    function incrementAccountNonce(address _address) external;
    
    function initPendingAccount(address _address) external;
    function commitPendingAccount(address _address, address _ethAddress, bytes32 _codeHash) external;

    function putContractStorage(address _contract, bytes32 _key, bytes32 _value) external;
    function getContractStorage(address _contract, bytes32 _key) external returns (bytes32 _value);
    function hasContractStorage(address _contract, bytes32 _key) external returns (bool _exists);

    function testAndSetAccountLoaded(address _address) external returns (bool _wasAccountAlreadyLoaded);
    function testAndSetAccountChanged(address _address) external returns (bool _wasAccountAlreadyChanged);

    function testAndSetContractStorageLoaded(address _contract, bytes32 _key) external returns (bool _wasContractStorageAlreadyLoaded);
    function testAndSetContractStorageChanged(address _contract, bytes32 _key) external returns (bool _wasContractStorageAlreadyChanged);
}
