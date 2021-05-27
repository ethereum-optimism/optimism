// SPDX-License-Identifier: MIT
// @unsupported: evm
pragma solidity >=0.7.0;

import { Lib_ExecutionManagerWrapper } from "@eth-optimism/contracts/contracts/optimistic-ethereum/libraries/wrappers/Lib_ExecutionManagerWrapper.sol";

contract ValueCalls {

    // TODO: this is unneccessary without the compiler.
    // Once we have the compiler, we should add explicit `payable` and `receive` integration tests.
    // receive() external payable { }

    function getBalance(
        address _address
    ) external returns(uint256) {
        return Lib_ExecutionManagerWrapper.ovmBALANCE(_address);
    }

    function simpleSend(
        address _address,
        uint _value
    ) external returns (bool, bytes memory) {
        return sendWithData(_address, _value, hex"");
    }

    function sendWithData(
        address _address,
        uint _value,
        bytes memory _calldata
    ) public returns (bool, bytes memory) {
        (bool success, ) = Lib_ExecutionManagerWrapper.ovmCALL(gasleft(), _address, _value, _calldata);
    }

    function verifyCallValueAndRevert(
        uint256 _expectedValue
    ) external {
        bool correct = _checkCallValue(_expectedValue);
        // do the opposite of expected if the value is wrong.
        if (correct) {
            revert();
        } else {
            return;
        }
    }

    function verifyCallValueAndReturn(
        uint256 _expectedValue
    ) external {
        bool correct = _checkCallValue(_expectedValue);
        // do the opposite of expected if the value is wrong.
        if (correct) {
            return;
        } else {
            revert();
        }
    }

    function _checkCallValue(
        uint256 _expectedValue
    ) internal returns(bool) {
        uint256 callValue = Lib_ExecutionManagerWrapper.ovmCALLVALUE();
        return callValue == _expectedValue;
    }
}