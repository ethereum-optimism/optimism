pragma solidity ^0.5.0;

contract OriginGetter {
    function getTxOrigin() public view returns(address) {
        return tx.origin;
    }
}