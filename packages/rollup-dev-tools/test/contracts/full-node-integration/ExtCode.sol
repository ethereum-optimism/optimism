pragma solidity ^0.5.0;

contract ExtCode {
    function getExtCodeSizeOf(address _addr) public returns(uint) {
        uint toReturn;
        assembly {
            toReturn:= extcodesize(_addr)
        }
        return toReturn;
    }

    function getExtCodeHashOf(address _addr) public returns(bytes32) {
        bytes32 toReturn;
        assembly {
            toReturn:= extcodehash(_addr)
        }
        return toReturn;
    }
}