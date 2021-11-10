// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

contract Helper_GasMeasurer {
    function measureCallGas(address _target, bytes memory _data) public returns (uint256) {
        uint256 gasBefore;
        uint256 gasAfter;

        uint256 calldataStart;
        uint256 calldataLength;
        assembly {
            calldataStart := add(_data, 0x20)
            calldataLength := mload(_data)
        }

        bool success;
        assembly {
            gasBefore := gas()
            success := call(gas(), _target, 0, calldataStart, calldataLength, 0, 0)
            gasAfter := gas()
        }
        require(success, "Call failed, but calls we want to measure gas for should succeed!");

        return gasBefore - gasAfter;
    }
}
