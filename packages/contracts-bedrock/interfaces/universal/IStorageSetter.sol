// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IStorageSetter {
    struct Slot {
        bytes32 key;
        bytes32 value;
    }

    function version() external view returns (string memory);
    function setBytes32(bytes32 _slot, bytes32 _value) external;
    function setBytes32(Slot[] calldata _slots) external;
    function getBytes32(bytes32 _slot) external view returns (bytes32 value_);
    function setUint(bytes32 _slot, uint256 _value) external;
    function getUint(bytes32 _slot) external view returns (uint256 value_);
    function setAddress(bytes32 _slot, address _address) external;
    function getAddress(bytes32 _slot) external view returns (address addr_);
    function setBool(bytes32 _slot, bool _value) external;
    function getBool(bytes32 _slot) external view returns (bool value_);

    function __constructor__() external;
}
