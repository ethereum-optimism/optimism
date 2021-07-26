pragma solidity ^0.5.16;

import "../../../contracts/Timelock.sol";

contract TimelockCertora is Timelock {
    constructor(address admin_, uint256 delay_) public Timelock(admin_, delay_) {}

    function grace() pure public returns(uint256) {
        return GRACE_PERIOD;
    }

    function queueTransactionStatic(address target, uint256 value, uint256 eta) public returns (bytes32) {
        return queueTransaction(target, value, "setCounter()", "", eta);
    }

    function cancelTransactionStatic(address target, uint256 value, uint256 eta) public {
        return cancelTransaction(target, value, "setCounter()", "", eta);
    }

    function executeTransactionStatic(address target, uint256 value, uint256 eta) public {
        executeTransaction(target, value, "setCounter()", "", eta); // NB: cannot return dynamic types (will hang solver)
    }
}
