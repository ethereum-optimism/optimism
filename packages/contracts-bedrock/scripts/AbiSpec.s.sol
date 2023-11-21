// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";
import { stdJson } from "forge-std/StdJson.sol";

/// @title AbiSpec
/// @notice Describes the ABI of L1 contracts.
///         This can be used to directly interfact with L1 contracts via ABI rather than solidity interfaces.
contract AbiSpec is Script {
    string internal _json;

    /// @notice constructs an AbiSpec given from a JSON specification
    constructor(string memory _path) {
        _json = vm.readFile(_path);
    }

    /// @notice returns the selector for a method identifier
    ///         Asserts that the selector is present in the ABI spec
    function method(string memory _contractName, string memory _identifier) public view returns (bytes4 sel_) {
        sel_ = bytes4(keccak256(bytes(_identifier)));
        string memory key = string(abi.encodePacked("$.", _contractName, ".methodIdentifiers.", toHex(sel_)));
        string memory value = stdJson.readString(_json, key);
        require(bytes(value).length > 0, "AbiSpec: method not found");
    }

    /// @notice returns the slot and offset for a storage identifier
    ///         Asserts that the storage identifier is present in the ABI spec
    function slot(
        string memory _contractName,
        string memory _identifier
    )
        public
        view
        returns (uint256 slot_, uint8 offset_)
    {
        string memory slotKey = string(abi.encodePacked("$.", _contractName, ".storageLayout.", _identifier, ".slot"));
        string memory offsetKey =
            string(abi.encodePacked("$.", _contractName, ".storageLayout.", _identifier, ".offset"));
        slot_ = stdJson.readUint(_json, slotKey);
        offset_ = uint8(stdJson.readUint(_json, offsetKey));
    }

    function toHex(bytes4 _b) internal pure returns (string memory hex_) {
        bytes memory c = new bytes(8);
        bytes memory _base = "0123456789abcdef";
        for (uint256 i = 0; i < 4; i++) {
            c[i * 2] = _base[uint8(_b[i]) / _base.length];
            c[i * 2 + 1] = _base[uint8(_b[i]) % _base.length];
        }
        hex_ = string(abi.encodePacked(c));
    }
}
