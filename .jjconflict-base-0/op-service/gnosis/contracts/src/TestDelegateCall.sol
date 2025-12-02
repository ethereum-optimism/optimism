// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract TestDelegateCall {
    function setTestValue(uint256 _value) external {
        bytes32 slot = keccak256("TestDelegateCall.testValue");
        assembly {
            sstore(slot, _value)
        }
    }

    function getTestValue() external view returns (uint256) {
        bytes32 slot = keccak256("TestDelegateCall.testValue");
        uint256 value;
        assembly {
            value := sload(slot)
        }
        return value;
    }
}