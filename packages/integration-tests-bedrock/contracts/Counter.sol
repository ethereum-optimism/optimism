pragma solidity ^0.8.10;

contract Counter {
    uint256 public value = 0;

    constructor() {}

    function getValue() public view returns (uint256) {
        return value;
    }

    function incValue() public {
        value++;
    }
}