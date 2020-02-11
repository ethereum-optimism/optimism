pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;


contract DummyContract {
    bytes32 someVal;

    constructor() public {
        someVal = keccak256("derp");
    }

    function dummyFunction(uint testInt, bytes memory testBytes) public pure returns (bool success, bytes memory output) {
        success = testInt != 0;
        output = testBytes;
    }
}
