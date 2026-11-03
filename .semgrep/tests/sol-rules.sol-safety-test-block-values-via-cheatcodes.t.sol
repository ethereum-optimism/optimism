// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

contract SemgrepTest__sol_safety_test_block_values_via_cheatcodes {
    uint256 internal sentAt;

    function test() internal {
        // ruleid: sol-safety-test-block-values-via-cheatcodes
        uint256 timestamp = block.timestamp;

        // ruleid: sol-safety-test-block-values-via-cheatcodes
        uint256 number = block.number;

        // ruleid: sol-safety-test-block-values-via-cheatcodes
        uint256 chainId = block.chainid;

        // ruleid: sol-safety-test-block-values-via-cheatcodes
        uint64 timestamp64 = uint64(block.timestamp);

        // ruleid: sol-safety-test-block-values-via-cheatcodes
        uint256 baseFee = block.basefee;

        // ruleid: sol-safety-test-block-values-via-cheatcodes
        address origin = tx.origin;

        uint256 reassigned;
        // ruleid: sol-safety-test-block-values-via-cheatcodes
        reassigned = block.number;

        // ruleid: sol-safety-test-block-values-via-cheatcodes
        (uint256 tupleTimestamp, uint256 tupleNumber) = (block.timestamp, block.number);

        // ok: sol-safety-test-block-values-via-cheatcodes
        uint256 viaCheatcode = vm.getBlockTimestamp();

        // ok: sol-safety-test-block-values-via-cheatcodes
        uint256 deadline = block.timestamp + 100;

        // ok: sol-safety-test-block-values-via-cheatcodes
        uint256[] memory infos = fetch(block.chainid);

        // ok: sol-safety-test-block-values-via-cheatcodes
        vm.warp(block.timestamp + 1);

        // ok: sol-safety-test-block-values-via-cheatcodes
        sentAt = block.timestamp;
    }
}
