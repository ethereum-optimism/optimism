// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract SimpleStorage {
    mapping(bytes32 => bytes32) public db;

    function set(bytes32 _key, bytes32 _value) public payable {
        db[_key] = _value;
    }

    function get(bytes32 _key) public view returns (bytes32) {
        return db[_key];
    }
}
