pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import {ExecutionManager} from "./ExecutionManager.sol";

/**
 * @title CreatorContract
 * @notice This contract simply deploys whatever data it is sent in the transaction calling it.
 *         It comes in handy for serving as an initial contract in rollup chains which can
 *         deploy any initial contracts.
 */
contract CreatorContract {
    ExecutionManager executionManager;

    constructor(address _executionManager) public {
        // TODO: Remove explicit reference to execution manager
        executionManager = ExecutionManager(_executionManager);
    }

    /**
     * @notice Fallback function which simply CREATEs a contract with whatever tx data it receives.
     */
    function() external {
        executionManager.ovmCREATE(msg.data);
    }
}
