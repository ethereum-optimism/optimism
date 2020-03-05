pragma solidity ^0.5.0;

contract SelfAware {
    function getMyAddress() public view returns(address) {
        return address(this);
    }
}