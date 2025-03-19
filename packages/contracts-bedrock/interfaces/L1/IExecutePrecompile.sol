// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IExecutePrecompile {
    /**
     * @notice Interface for the execute precompile 
     * @param payload       Contains the trace required to execute the block
     * @param preStateRoot  The state root of the L2 chain before executing the block
     * @param postStateRoot The state root of the L2 chain after executing the block
     * @param gasUsed       The total gas used during execution of the block
     * @return success
     */
    function execute(
        bytes memory payload,
        bytes32 preStateRoot,
        bytes32 postStateRoot,
        uint256 gasUsed
    ) 
        external
        returns (bool success);
}