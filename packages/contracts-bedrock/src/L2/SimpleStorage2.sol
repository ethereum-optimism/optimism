// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

contract SimpleStorage2 {
    uint256 private _storedData;

    uint256 public constant SPECIAL_NUMBER = 1234;

    // Event to emit when value is changed
    event ValueChanged(uint256 newValue);

    // Function to store a new value
    function set(uint256 x) public {
        _storedData = x;
        emit ValueChanged(x);
    }

    // Function to get the stored value
    function get() public view returns (uint256) {
        return _storedData;
    }

    function getSpecialNumber() public pure returns (uint256) {
        return SPECIAL_NUMBER;
    }
}
