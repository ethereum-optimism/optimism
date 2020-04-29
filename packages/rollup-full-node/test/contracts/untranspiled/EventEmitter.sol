pragma solidity ^0.5.0;

contract EventEmitter {
    event DummyEvent();
    function emitEvent() public {
      emit DummyEvent();
    }
}
