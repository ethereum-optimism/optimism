pragma solidity ^0.5.0;

import {ExecutionManager} from "../execution/ExecutionManager.sol";


contract L1MessageSender {
    ExecutionManager executionManager;

    constructor(address _executionManagerAddress) public {
        executionManager = ExecutionManager(_executionManagerAddress);
    }

    function getL1MessageSender() public returns(address) {
        return executionManager.getL1MessageSender();
    }
}
