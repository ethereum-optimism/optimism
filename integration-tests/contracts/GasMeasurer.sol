// SPDX-License-Identifier: MIT
pragma solidity >=0.7.0;

contract GasMeasurer {
    function measureGasCostOfCall(
        address _target,
        bytes memory _data
    ) public returns(uint) {
        uint256 gasBefore = gasleft();
        (bool success, bytes memory returndata) = _target.call{gas: gasleft()}(_data);
        require(success, string(abi.encodePacked("Attempted to measure gas of unsuccessfull call.  error is: ", returndata)));
        require(gasBefore > gasleft(), "Overflow: did you get a big refund back?");
        return gasBefore - gasleft();
    }
}
