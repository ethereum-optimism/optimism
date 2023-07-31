// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { BlockHashOracle } from "src/dispute/BlockHashOracle.sol";
import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";

contract BlockHashOracle_Test is Test {
    BlockHashOracle oracle;

    function setUp() public {
        oracle = new BlockHashOracle();
        vm.roll(block.number + 255);
    }

    /// @notice Tests that loading a block hash for a block number within the range of the
    ///         `BLOCKHASH` opcode succeeds.
    function testFuzz_store_succeeds(uint256 _blockNumber) public {
        _blockNumber = bound(_blockNumber, 0, 255);
        oracle.store(_blockNumber);
        assertEq(Hash.unwrap(oracle.load(_blockNumber)), blockhash(_blockNumber));
    }

    /// @notice Tests that loading a block hash for a block number outside the range of the
    ///         `BLOCKHASH` opcode fails.
    function testFuzz_store_oob_reverts(uint256 _blockNumber) public {
        // Fast forward another 256 blocks.
        vm.roll(block.number + 256);
        // Bound the block number to the set { 0, ..., 255 } ∪ { 512, ..., type(uint256).max }
        _blockNumber = _blockNumber % 2 == 0
            ? bound(_blockNumber, 0, 255)
            : bound(_blockNumber, 512, type(uint256).max);

        // Attempt to load the block hash, which should fail.
        vm.expectRevert(BlockNumberOOB.selector);
        oracle.store(_blockNumber);
    }

    /// @notice Tests that the `load` function reverts if the block hash for the given block
    ///         number has not been stored.
    function test_load_noBlockHash_reverts() public {
        vm.expectRevert(BlockHashNotPresent.selector);
        oracle.load(0);
    }
}
