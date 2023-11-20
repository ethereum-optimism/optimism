// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { stdJson } from "forge-std/StdJson.sol";
import { VmSafe } from "forge-std/Vm.sol";

/// @title LibStateDiff
/// @author refcell
/// @notice Library to write StateDiff output to json.
library LibStateDiff {
    VmSafe private constant vm = VmSafe(address(uint160(uint256(keccak256("hevm cheat code")))));

    using stdJson for string;

    /// @notice Accepts an array of AccountAccess structs from the Vm and encodes them as a json string.
    /// @param _accountAccesses Array of AccountAccess structs.
    /// @return serialized_ string
    function encodeAccountAccesses(VmSafe.AccountAccess[] memory _accountAccesses)
        internal
        returns (string memory serialized_)
    {
        string[] memory accountAccesses = new string[](_accountAccesses.length);
        for (uint256 i = 0; i < _accountAccesses.length; i++) {
            accountAccesses[i] = serializeAccountAccess(_accountAccesses[i]);
        }
        serialized_ = "";
        serialized_.serialize("accountAccesses", accountAccesses);
    }

    /// @notice Turns an AccountAccess into a json serialized string
    /// @param _accountAccess The AccountAccess to serialize
    /// @return serialized_ The json serialized string
    function serializeAccountAccess(VmSafe.AccountAccess memory _accountAccess)
        internal
        returns (string memory serialized_)
    {
        string memory json = "access";

        json.serialize("account", _accountAccess.account);
        json.serialize("accessor", _accountAccess.accessor);
        json.serialize("initialized", _accountAccess.initialized);
        json.serialize("oldBalance", _accountAccess.oldBalance);
        json.serialize("newBalance", _accountAccess.newBalance);
        json.serialize("deployedCode", _accountAccess.deployedCode);
        json.serialize("value", _accountAccess.value);
        json.serialize("data", _accountAccess.data);
        json.serialize("reverted", _accountAccess.reverted);

        string memory chainInfo = serializeChainInfo(_accountAccess.chainInfo);
        string memory accountAccessKind = serializeAccountAccessKind(_accountAccess.kind);
        string memory storageAccesses = serializeStorageAccesses(_accountAccess.storageAccesses);

        json = vm.serializeString(json, "chainInfo", chainInfo);
        json = vm.serializeString(json, "kind", accountAccessKind);

        json = vm.serializeString(json, "storageAccesses", storageAccesses);
        serialized_ = json;
    }

    /// @notice Accepts a VmSafe.ChainInfo struct and encodes it as a json string.
    /// @param _chainInfo The ChainInfo struct to serialize
    /// @return serialized_ string
    function serializeChainInfo(VmSafe.ChainInfo memory _chainInfo) internal returns (string memory serialized_) {
        string memory json = "chainInfo";
        json.serialize("forkId", _chainInfo.forkId);
        json.serialize("chainId", _chainInfo.chainId);
        serialized_ = json;
    }

    /// @notice Turns an AccountAccessKind into a string.
    /// @param _kind The AccountAccessKind to serialize
    /// @return serialized_ The string representation of the AccountAccessKind
    function serializeAccountAccessKind(VmSafe.AccountAccessKind _kind)
        internal
        pure
        returns (string memory serialized_)
    {
        if (_kind == VmSafe.AccountAccessKind.Call) {
            serialized_ = "Call";
        } else if (_kind == VmSafe.AccountAccessKind.DelegateCall) {
            serialized_ = "DelegateCall";
        } else if (_kind == VmSafe.AccountAccessKind.CallCode) {
            serialized_ = "CallCode";
        } else if (_kind == VmSafe.AccountAccessKind.StaticCall) {
            serialized_ = "StaticCall";
        } else if (_kind == VmSafe.AccountAccessKind.Create) {
            serialized_ = "Create";
        } else if (_kind == VmSafe.AccountAccessKind.SelfDestruct) {
            serialized_ = "SelfDestruct";
        } else {
            serialized_ = "Resume";
        }
    }

    /// @notice Accepts an array of StorageAccess structs from the Vm and encodes each as a json string.
    /// @param _storageAccesses Array of StorageAccess structs.
    /// @return serialized_ json serialized StorageAccess structs.
    function serializeStorageAccesses(VmSafe.StorageAccess[] memory _storageAccesses)
        internal
        returns (string memory serialized_)
    {
        string[] memory storageAccesses = new string[](_storageAccesses.length);
        for (uint256 i = 0; i < _storageAccesses.length; i++) {
            storageAccesses[i] = serializeStorageAccess(_storageAccesses[i]);
        }
        serialized_ = "storageAccesses";
        serialized_.serialize("storageAccesses", storageAccesses);
    }

    /// @notice Turns a StorageAccess into a json serialized string
    /// @param _storageAccess The StorageAccess to serialize
    /// @return serialized_ The json serialized string
    function serializeStorageAccess(VmSafe.StorageAccess memory _storageAccess)
        internal
        returns (string memory serialized_)
    {
        string memory json = "storageAccess";
        json.serialize("account", _storageAccess.account);
        json.serialize("slot", _storageAccess.slot);
        json.serialize("isWrite", _storageAccess.isWrite);
        json.serialize("previousValue", _storageAccess.previousValue);
        json.serialize("newValue", _storageAccess.newValue);
        json.serialize("reverted", _storageAccess.reverted);
        serialized_ = json;
    }
}
