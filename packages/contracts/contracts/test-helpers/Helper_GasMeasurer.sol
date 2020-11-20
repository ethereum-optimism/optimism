// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

contract Helper_GasMeasurer {
    function measureCallGas(
        address _target,
        bytes memory _data
    )
        public
        returns ( uint )
    {
        uint gasBefore;
        uint gasAfter;

        uint calldataStart;
        uint calldataLength;
        assembly {
            calldataStart := add(_data,0x20)
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
