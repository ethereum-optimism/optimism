pragma solidity ^0.5.0;

contract SubEventEmitter {
    event Burger(address);

    function doEmit() public {
        emit Burger(0x4206900000000000000000000000000000000000);
    }
}