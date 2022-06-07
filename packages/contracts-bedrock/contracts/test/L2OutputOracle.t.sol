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
        assertEq(oracle.HISTORICAL_TOTAL_BLOCKS(), historicalTotalBlocks);
        assertEq(oracle.latestBlockNumber(), startingBlockNumber);
        assertEq(oracle.STARTING_BLOCK_NUMBER(), startingBlockNumber);

        L2OutputOracle.OutputProposal memory proposal = oracle.getL2Output(startingBlockNumber);
        assertEq(proposal.outputRoot, genesisL2Output);
        assertEq(proposal.timestamp, initTime);
    }

    /****************
     * Getter Tests *
     ****************/

    // Test: latestBlockNumber() should return the correct value
    function test_latestBlockNumber() external {
        uint256 appendedNumber = oracle.nextBlockNumber();

        // Roll to after the block number we'll append
        vm.roll(appendedNumber + 1);
        vm.prank(sequencer);
        oracle.appendL2Output(appendedOutput1, appendedNumber, 0, 0);
        assertEq(oracle.latestBlockNumber(), appendedNumber);
    }

    // Test: getL2Output() should return the correct value
    function test_getL2Output() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();

        vm.prank(sequencer);
        oracle.appendL2Output(appendedOutput1, nextBlockNumber, 0, 0);

        L2OutputOracle.OutputProposal memory proposal = oracle.getL2Output(nextBlockNumber);
        assertEq(proposal.outputRoot, appendedOutput1);
        assertEq(proposal.timestamp, block.timestamp);

        L2OutputOracle.OutputProposal memory proposal2 = oracle.getL2Output(0);
        assertEq(proposal2.outputRoot, bytes32(0));
        assertEq(proposal2.timestamp, 0);
    }

    // Test: nextBlockNumber() should return the correct value
    function test_nextBlockNumber() external {
        assertEq(
            oracle.nextBlockNumber(),
            // The return value should match this arithmetic
            oracle.latestBlockNumber() + oracle.SUBMISSION_INTERVAL()
        );
    }

    /*****************************
     * Append Tests - Happy Path *
     *****************************/

    // Test: appendL2Output succeeds when given valid input, and no block hash and number are
    // specified.
    function test_appendingAnotherOutput() public {
        bytes32 appendedOutput2 = keccak256(abi.encode(2));
        uint256 nextBlockNumber = oracle.nextBlockNumber();

        uint256 appendedNumber = oracle.latestBlockNumber();

        // Ensure the submissionInterval is enforced
        assertEq(nextBlockNumber, appendedNumber + submissionInterval);

        vm.roll(nextBlockNumber + 1);
        vm.prank(sequencer);
        oracle.appendL2Output(appendedOutput2, nextBlockNumber, 0, 0);
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

        uint256 nextBlockNumber = oracle.nextBlockNumber();
        vm.roll(nextBlockNumber + 1);
        vm.prank(sequencer);

        // Changing the l1BlockNumber argument should break this tests, however it does not
        // per the comment preceding this test.
        oracle.appendL2Output(nonZeroHash, nextBlockNumber, l1BlockHash, l1BlockNumber);
    }

    /***************************
     * Append Tests - Sad Path *
     ***************************/

    // Test: appendL2Output fails if called by a party that is not the sequencer.
    function testCannot_appendOutputIfNotSequencer() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();

        vm.prank(address(128));
        vm.roll(nextBlockNumber + 1);
        vm.expectRevert("Ownable: caller is not the owner");
        oracle.appendL2Output(nonZeroHash, nextBlockNumber, 0, 0);
    }

    // Test: appendL2Output fails given a zero blockhash.
    function testCannot_appendEmptyOutput() external {
        bytes32 outputToAppend = bytes32(0);
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        vm.roll(nextBlockNumber + 1);
        vm.prank(sequencer);
        vm.expectRevert("Cannot submit empty L2 output");
        oracle.appendL2Output(outputToAppend, nextBlockNumber, 0, 0);
    }

    // Test: appendL2Output fails if the block number doesn't match the next expected number.
    function testCannot_appendUnexpectedBlockNumber() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        vm.roll(nextBlockNumber + 1);
        vm.prank(sequencer);
        vm.expectRevert("Block number must be equal to next expected block number");
        oracle.appendL2Output(nonZeroHash, nextBlockNumber - 1, 0, 0);
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

        uint256 nextBlockNumber = oracle.nextBlockNumber();
        vm.roll(nextBlockNumber + 1);
        vm.prank(sequencer);

        // This will fail when foundry no longer returns zerod block hashes
        oracle.appendL2Output(nonZeroHash, nextBlockNumber, l1BlockHash, l1BlockNumber - 1);
    }

    /****************
     * Delete Tests *
     ****************/

    event l2OutputDeleted(
        bytes32 indexed _l2Output,
        uint256 indexed _l1Timestamp,
        uint256 indexed _l2BlockNumber
    );


    function test_deleteL2Output() external {
        test_appendingAnotherOutput();

        uint256 latestBlockNumber = oracle.latestBlockNumber();
        L2OutputOracle.OutputProposal memory proposalToDelete = oracle.getL2Output(latestBlockNumber);
        L2OutputOracle.OutputProposal memory newLatestOutput = oracle.getL2Output(latestBlockNumber - submissionInterval);

        vm.prank(sequencer);
        vm.expectEmit(true, true, false, false);
        emit l2OutputDeleted(
            proposalToDelete.outputRoot,
            proposalToDelete.timestamp,
            latestBlockNumber
        );
        oracle.deleteL2Output(proposalToDelete);

        // validate latestBlockNumber has been reduced
        uint256 latestBlockNumberAfter = oracle.latestBlockNumber();
        assertEq(
            latestBlockNumber - submissionInterval,
            latestBlockNumberAfter
        );

        L2OutputOracle.OutputProposal memory proposal = oracle.getL2Output(latestBlockNumberAfter);
        // validate that the new latest output is as expected.
        assertEq(newLatestOutput.outputRoot, proposal.outputRoot);
        assertEq(newLatestOutput.timestamp, proposal.timestamp);
    }

    function testCannot_deleteL2Output_ifNotSequencer() external {
        uint256 latestBlockNumber = oracle.latestBlockNumber();
        L2OutputOracle.OutputProposal memory proposal = oracle.getL2Output(latestBlockNumber);

        vm.expectRevert("Ownable: caller is not the owner");
        oracle.deleteL2Output(proposal);
    }

    function testCannot_deleteWrongL2Output() external {
        test_appendingAnotherOutput();

        uint256 previousBlockNumber = oracle.latestBlockNumber() - submissionInterval;
        L2OutputOracle.OutputProposal memory proposalToDelete = oracle.getL2Output(previousBlockNumber);

        vm.prank(sequencer);
        vm.expectRevert("Can only delete the most recent output.");
        oracle.deleteL2Output(proposalToDelete);
    }
}
