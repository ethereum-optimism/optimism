pragma solidity ^0.5.0;
import "./SubEventEmitter.sol";

contract MasterEventEmitter {
    event Taco(address);
    SubEventEmitter public sub;
    constructor (address _sub) public {
        sub = SubEventEmitter(_sub);
    }
    function callSubEmitter() public {
        emit Taco(0x0000000000000000000000000000000000000000);
        sub.doEmit();
    }
}