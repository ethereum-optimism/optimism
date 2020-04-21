pragma solidity ^0.5.0;

contract EventEmitter {
    event Event();
    function emitEvent(address exeMgrAddr) public {
      emit Event();
    }
}
