pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

contract RevertTest {
    uint a;
    uint b;

    constructor() public {
        a = 0;
        b = 0;
    }

    function entryPoint() public {
        bytes32 callBothMethodId = keccak256("callBoth()");
        bool success;
        bytes memory callBytes;
        address addr = address(this);
        assembly {
            callBytes := mload(0x40)
            mstore(callBytes, callBothMethodId)

            success := call(gas, addr, 0, callBytes, 4, 0, 0)
        }

        require(!success, "Sub-call should have reverted");
    }

    function callBoth() public {
        changeA();
        revertInHere();
    }

    function changeA() internal {
        a = 5;
    }

    function revertInHere() internal pure {
        revert("Trying to revert A's state change.");
    }

    function getA() public view returns (uint) {
        return a;
    }
}
