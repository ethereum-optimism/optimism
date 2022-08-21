// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

contract Coersion {
    function toUint256(bytes32 _val) external pure returns (uint256) {
        return uint256(_val);
    }

    function toBytes32(uint256 _val) external pure returns (bytes32) {
        return bytes32(_val);
    }
}
