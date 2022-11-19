// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

contract ValueContext {
    function getSelfBalance() external payable returns(uint256) {
        uint selfBalance;
        assembly {
            selfBalance := selfbalance()
        }
        return selfBalance;
    }

    function getAddressThisBalance() external view returns(uint256) {
        return address(this).balance;
    }

    function getBalance(
        address _address
    ) external payable returns(uint256) {
        return _address.balance;
    }

    function getCallValue() public payable returns(uint256) {
        return msg.value;
    }

    function getCaller() external view returns (address){
        return msg.sender;
    }
}

contract ValueCalls is ValueContext {
    receive() external payable {}

    function nonPayable() external {}

    function simpleSend(
        address _address,
        uint _value
    ) external payable returns (bool, bytes memory) {
        return sendWithData(_address, _value, hex"");
    }

    function sendWithDataAndGas(
        address _address,
        uint _value,
        uint _gasLimit,
        bytes memory _calldata
    ) public returns (bool, bytes memory) {
        return _address.call{value: _value, gas: _gasLimit}(_calldata);
    }

    function sendWithData(
        address _address,
        uint _value,
        bytes memory _calldata
    ) public returns (bool, bytes memory) {
        return _address.call{value: _value}(_calldata);
    }

    function verifyCallValueAndRevert(
        uint256 _expectedValue
    ) external payable {
        bool correct = _checkCallValue(_expectedValue);
        // do the opposite of expected if the value is wrong.
        if (correct) {
            revert("expected revert");
        } else {
            return;
        }
    }

    function verifyCallValueAndReturn(
        uint256 _expectedValue
    ) external payable {
        bool correct = _checkCallValue(_expectedValue);
        // do the opposite of expected if the value is wrong.
        if (correct) {
            return;
        } else {
            revert("unexpected revert");
        }
    }

    function delegateCallToCallValue(
        address _valueContext
    ) public payable returns(bool, bytes memory) {
        bytes memory data = abi.encodeWithSelector(ValueContext.getCallValue.selector);
        return _valueContext.delegatecall(data);
    }

    function delegateCallToAddressThisBalance(
        address _valueContext
    ) public payable returns(bool, bytes memory) {
        bytes memory data = abi.encodeWithSelector(ValueContext.getAddressThisBalance.selector);
        return _valueContext.delegatecall(data);
    }

    function _checkCallValue(
        uint256 _expectedValue
    ) internal returns(bool) {
        return getCallValue() == _expectedValue;
    }
}

contract ValueGasMeasurer {
    function measureGasOfTransferingEthViaCall(
        address target,
        uint256 value,
        uint256 gasLimit
    ) public returns(uint256) {
        uint256 gasBefore = gasleft();
        assembly {
            pop(call(gasLimit, target, value, 0, 0, 0, 0))
        }
        return gasBefore - gasleft();
    }
}

contract PayableConstant {
    function returnValue() external payable returns(uint256) {
        return 42;
    }
}

contract SendETHAwayAndDelegateCall {
    function emptySelfAndDelegateCall(
        address _delegateTo,
        bytes memory _data
    ) public payable returns (bool, bytes memory) {
        address(0).call{value: address(this).balance}(_data);

        return _delegateTo.delegatecall(_data);
    }
}
