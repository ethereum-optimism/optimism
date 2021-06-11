// SPDX-License-Identifier: MIT
pragma solidity >=0.7.0;

contract GasMeasurer {
    function measureGasCostOfCall(
        address _target,
        bytes memory _data
    ) public returns(uint) {
        uint256 gasBefore = gasleft();
        _target.call{gas: gasleft()}(_data);
        return gasBefore - gasleft();
    }
}
