//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { L2OutputOracle_Initializer } from "./CommonTest.t.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";

contract L2OutputOracleTest is L2OutputOracle_Initializer {
    bytes32 appendedOutput1 = keccak256(abi.encode(1));

    function setUp() public override {
        super.setUp();
    }

    function test_constructor() external {
        assertEq(oracle.owner(), sequencer);
        assertEq(oracle.SUBMISSION_INTERVAL(), submissionInterval);
        assertEq(oracle.L2_BLOCK_TIME(), l2BlockTime);
        assertEq(oracle.HISTORICAL_TOTAL_BLOCKS(), historicalTotalBlocks);
        assertEq(oracle.latestBlockTimestamp(), startingBlockTimestamp);
        assertEq(oracle.STARTING_BLOCK_TIMESTAMP(), startingBlockTimestamp);

        L2OutputOracle.OutputProposal memory proposal = oracle.getL2Output(startingBlockTimestamp);
        assertEq(proposal.outputRoot, genesisL2Output);
        assertEq(proposal.timestamp, initTime);
    }

    /****************
     * Getter Tests *
     ****************/

    // Test: latestBlockTimestamp() should return the correct value
    function test_latestBlockTimestamp() external {
        uint256 appendedTimestamp = oracle.nextTimestamp();

        // Warp to after the timestamp we'll append
        vm.warp(appendedTimestamp + 1);
        vm.prank(sequencer);
        oracle.appendL2Output(appendedOutput1, appendedTimestamp, 0, 0);
        assertEq(oracle.latestBlockTimestamp(), appendedTimestamp);
    }

    // Test: getL2Output() should return the correct value
    function test_getL2Output() external {
        uint256 nextTimestamp = oracle.nextTimestamp();

        vm.warp(nextTimestamp + 1);
        vm.prank(sequencer);
        oracle.appendL2Output(appendedOutput1, nextTimestamp, 0, 0);

        L2OutputOracle.OutputProposal memory proposal = oracle.getL2Output(nextTimestamp);
        assertEq(proposal.outputRoot, appendedOutput1);
        assertEq(proposal.timestamp, nextTimestamp + 1);

        L2OutputOracle.OutputProposal memory proposal2 = oracle.getL2Output(0);
        assertEq(proposal2.outputRoot, bytes32(0));
        assertEq(proposal2.timestamp, 0);

    }

    // Test: nextTimestamp() should return the correct value
    function test_nextTimestamp() external {
        assertEq(
            oracle.nextTimestamp(),
            // The return value should match this arithmetic
            oracle.latestBlockTimestamp() + oracle.SUBMISSION_INTERVAL()
        );
    }

    // Test: computeL2BlockNumber() should return the correct value
    function test_computeL2BlockNumber() external {
        // Test with the timestamp of the very first appended block
        uint256 argTimestamp = startingBlockTimestamp;
        uint256 expected = historicalTotalBlocks;
        assertEq(oracle.computeL2BlockNumber(argTimestamp), expected);

        // Test with an integer multiple of the l2BlockTime
        argTimestamp = startingBlockTimestamp + 20;
        expected = historicalTotalBlocks + (20 / l2BlockTime);
        assertEq(oracle.computeL2BlockNumber(argTimestamp), expected);

        // Test with a remainder
        argTimestamp = startingBlockTimestamp + 33;
        expected = historicalTotalBlocks + (33 / l2BlockTime);
        assertEq(oracle.computeL2BlockNumber(argTimestamp), expected);
    }

    // Test: computeL2BlockNumber() fails with a blockNumber from before the startingBlockTimestamp
    function testCannot_computePreHistoricalL2BlockNumber() external {
        bytes memory expectedError = "Timestamp prior to startingBlockTimestamp";
        uint256 argTimestamp = startingBlockTimestamp - 1;
        vm.expectRevert(expectedError);
        oracle.computeL2BlockNumber(argTimestamp);
    }

    /*****************************
     * Append Tests - Happy Path *
     *****************************/

    // Test: appendL2Output succeeds when given valid input, and no block hash and number are
    // specified.
    function test_appendingAnotherOutput() public {
        bytes32 appendedOutput2 = keccak256(abi.encode(2));
        uint256 nextTimestamp = oracle.nextTimestamp();

        uint256 appendedTimestamp = oracle.latestBlockTimestamp();

        // Ensure the submissionInterval is enforced
        assertEq(nextTimestamp, appendedTimestamp + submissionInterval);

        vm.warp(nextTimestamp + 1);
        vm.prank(sequencer);
        oracle.appendL2Output(appendedOutput2, nextTimestamp, 0, 0);
    }

    // Test: appendL2Output succeeds when given valid input, and when a block hash and number are
    // specified for reorg protection.
    // This tests is disabled (w/ skip_ prefix) because all blocks in Foundry currently have a
    // blockhash of zero.
    function skip_test_appendWithBlockhashAndHeight() external {
        // Move ahead to block 100 so that we can reference historical blocks
        vm.roll(100);

        // Get the number and hash of a previous block in the chain
        uint256 l1BlockNumber = block.number - 1;
        bytes32 l1BlockHash = blockhash(l1BlockNumber);

        uint256 nextTimestamp = oracle.nextTimestamp();
        vm.warp(nextTimestamp + 1);
        vm.prank(sequencer);

        // Changing the l1BlockNumber argument should break this tests, however it does not
        // per the comment preceding this test.
        oracle.appendL2Output(nonZeroHash, nextTimestamp, l1BlockHash, l1BlockNumber);
    }

    /***************************
     * Append Tests - Sad Path *
     ***************************/

    // Test: appendL2Output fails if called by a party that is not the sequencer.
    function testCannot_appendOutputIfNotSequencer() external {
        uint256 nextTimestamp = oracle.nextTimestamp();

        vm.prank(address(128));
        vm.warp(nextTimestamp + 1);
        vm.expectRevert("Ownable: caller is not the owner");
        oracle.appendL2Output(nonZeroHash, nextTimestamp, 0, 0);
    }

    // Test: appendL2Output fails given a zero blockhash.
    function testCannot_appendEmptyOutput() external {
        bytes32 outputToAppend = bytes32(0);
        uint256 nextTimestamp = oracle.nextTimestamp();
        vm.warp(nextTimestamp + 1);
        vm.prank(sequencer);
        vm.expectRevert("Cannot submit empty L2 output");
        oracle.appendL2Output(outputToAppend, nextTimestamp, 0, 0);
    }

    // Test: appendL2Output fails if the timestamp doesn't match the next expected timestamp.
    function testCannot_appendUnexpectedTimestamp() external {
        uint256 nextTimestamp = oracle.nextTimestamp();
        vm.warp(nextTimestamp + 1);
        vm.prank(sequencer);
        vm.expectRevert("Timestamp not equal to next expected timestamp");
        oracle.appendL2Output(nonZeroHash, nextTimestamp - 1, 0, 0);
    }

    // Test: appendL2Output fails if the timestamp is equal to the current L1 timestamp.
    function testCannot_appendCurrentTimestamp() external {
        uint256 nextTimestamp = oracle.nextTimestamp();
        vm.warp(nextTimestamp + 1);
        vm.prank(sequencer);
        vm.expectRevert("Cannot append L2 output in future");
        oracle.appendL2Output(nonZeroHash, block.timestamp, 0, 0);
    }

    // Test: appendL2Output fails if the timestamp is in the future.
    function testCannot_appendFutureTimestamp() external {
        uint256 nextTimestamp = oracle.nextTimestamp();
        vm.warp(nextTimestamp + 1);
        vm.prank(sequencer);
        vm.expectRevert("Cannot append L2 output in future");
        oracle.appendL2Output(nonZeroHash, block.timestamp + 1, 0, 0);
    }

    // Test: appendL2Output fails when given valid input, but the block hash and number do not
    // match.
    // This tests is disabled (w/ skip_ prefix) because all blocks in Foundry currently have a
    // blockhash of zero.
    function skip_testCannot_AppendWithUnmatchedBlockhash() external {
        // Move ahead to block 100 so that we can reference historical blocks
        vm.roll(100);

        // Get the number and hash of a previous block in the chain
        uint256 l1BlockNumber = block.number - 1;
        bytes32 l1BlockHash = blockhash(l1BlockNumber);

        uint256 nextTimestamp = oracle.nextTimestamp();
        vm.warp(nextTimestamp + 1);
        vm.prank(sequencer);

        // This will fail when foundry no longer returns zerod block hashes
        oracle.appendL2Output(nonZeroHash, nextTimestamp, l1BlockHash, l1BlockNumber - 1);
    }

    /****************
     * Delete Tests *
     ****************/

    event l2OutputDeleted(
        bytes32 indexed _l2Output,
        uint256 indexed _l1Timestamp,
        uint256 indexed _l2timestamp
    );

    function test_deleteL2Output() external {
        test_appendingAnotherOutput();

        uint256 latestBlockTimestamp = oracle.latestBlockTimestamp();
        L2OutputOracle.OutputProposal memory proposalToDelete = oracle.getL2Output(latestBlockTimestamp);
        L2OutputOracle.OutputProposal memory newLatestOutput = oracle.getL2Output(latestBlockTimestamp - submissionInterval);

        vm.prank(sequencer);
        vm.expectEmit(true, true, false, false);
        emit l2OutputDeleted(
            proposalToDelete.outputRoot,
            proposalToDelete.timestamp,
            latestBlockTimestamp
        );
        oracle.deleteL2Output(proposalToDelete);

        // validate latestBlockTimestamp has been reduced
        uint256 latestBlockTimestampAfter = oracle.latestBlockTimestamp();
        assertEq(
            latestBlockTimestamp - submissionInterval,
            latestBlockTimestampAfter
        );

        L2OutputOracle.OutputProposal memory proposal = oracle.getL2Output(latestBlockTimestampAfter);
        // validate that the new latest output is as expected.
        assertEq(newLatestOutput.outputRoot, proposal.outputRoot);
        assertEq(newLatestOutput.timestamp, proposal.timestamp);
    }

    function testCannot_deleteL2Output_ifNotSequencer() external {
        uint256 latestBlockTimestamp = oracle.latestBlockTimestamp();
        L2OutputOracle.OutputProposal memory proposal = oracle.getL2Output(latestBlockTimestamp);

        vm.expectRevert("Ownable: caller is not the owner");
        oracle.deleteL2Output(proposal);
    }

    function testCannot_deleteWrongL2Output() external {
        test_appendingAnotherOutput();

        uint256 previousBlockTimestamp = oracle.latestBlockTimestamp() - submissionInterval;
        L2OutputOracle.OutputProposal memory proposalToDelete = oracle.getL2Output(previousBlockTimestamp);

        vm.prank(sequencer);
        vm.expectRevert("Can only delete the most recent output.");
        oracle.deleteL2Output(proposalToDelete);
    }
}
