pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import { ExecutionManager } from "../ExecutionManager.sol";

contract L1MessageSender {
    ExecutionManager executionManager;

    constructor(
        address _executionManagerAddress
    ) public {
        executionManager = ExecutionManager(_executionManagerAddress);
    }

    function getL1MessageSender() public returns (address) {
        return executionManager.getL1MessageSender();
    }
}