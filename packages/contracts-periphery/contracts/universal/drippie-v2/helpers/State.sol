// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

contract State {
    mapping (address => mapping (bytes32 => bytes32)) public db;

    function set(bytes32 _key, bytes32 _val) external {
        db[msg.sender][_key] = _val;
    }

    function get(bytes32 _key) external view returns (bytes32) {
        return db[msg.sender][_key];
    }
}
