// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { BlockOracle } from "src/dispute/BlockOracle.sol";
import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";

contract BlockOracle_Test is Test {
    BlockOracle oracle;

    /// @notice Emitted when a block is checkpointed.
    event Checkpoint(uint256 indexed blockNumber, Hash indexed blockHash, Timestamp indexed childTimestamp);

    function setUp() public {
        oracle = new BlockOracle();
        // Roll the chain forward 1 block.
        vm.roll(block.number + 1);
        vm.warp(block.timestamp + 13);
    }

    /// @notice Tests that checkpointing a block and loading its information succeeds.
    function test_checkpointAndLoad_succeeds() public {
        vm.expectEmit(true, true, true, false);
        emit Checkpoint(
            block.number - 1, Hash.wrap(blockhash(block.number - 1)), Timestamp.wrap(uint64(block.timestamp))
        );
        oracle.checkpoint();
        uint256 blockNumber = block.number - 1;
        BlockOracle.BlockInfo memory res = oracle.load(blockNumber);

        assertEq(Hash.unwrap(res.hash), blockhash(blockNumber));
        assertEq(Timestamp.unwrap(res.childTimestamp), block.timestamp);
    }

    /// @notice Tests that the `load` function reverts if the block hash for the given block
    ///         number has not been stored.
    function test_load_noBlockHash_reverts() public {
        vm.expectRevert(BlockHashNotPresent.selector);
        oracle.load(0);
    }
}
