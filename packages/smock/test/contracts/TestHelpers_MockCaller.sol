// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;

contract TestHelpers_MockCaller {
    function callMock(address _target, bytes memory _data) public {
        _target.call(_data);
    }
}
