// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { stdError } from "forge-std/Test.sol";
import { L2OutputOracle_Initializer, NextImpl } from "./CommonTest.t.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { Proxy } from "../universal/Proxy.sol";
import { Types } from "../libraries/Types.sol";

contract L2OutputOracleTest is L2OutputOracle_Initializer {
    bytes32 proposedOutput1 = keccak256(abi.encode(1));

    function setUp() public override {
        super.setUp();
    }

    function test_constructor() external {
        assertEq(oracle.PROPOSER(), proposer);
        assertEq(oracle.CHALLENGER(), owner);
        assertEq(oracle.SUBMISSION_INTERVAL(), submissionInterval);
        assertEq(oracle.latestBlockNumber(), startingBlockNumber);
        assertEq(oracle.startingBlockNumber(), startingBlockNumber);
        assertEq(oracle.startingTimestamp(), startingTimestamp);
    }

    function testCannot_constructWithBadTimestamp() external {
        vm.expectRevert("L2OutputOracle: starting L2 timestamp must be less than current time");

        new L2OutputOracle(
            submissionInterval,
            l2BlockTime,
            startingBlockNumber,
            // startingTimestamp is in the future
            block.timestamp + 1,
            proposer,
            owner
        );
    }

    /****************
     * Getter Tests *
     ****************/

    // Test: latestBlockNumber() should return the correct value
    function test_latestBlockNumber() external {
        uint256 proposedNumber = oracle.nextBlockNumber();

        // Roll to after the block number we'll propose
        warpToProposeTime(proposedNumber);
        vm.prank(proposer);
        oracle.proposeL2Output(proposedOutput1, proposedNumber, 0, 0);
        assertEq(oracle.latestBlockNumber(), proposedNumber);
    }

    // Test: getL2Output() should return the correct value
    function test_getL2Output() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        uint256 nextOutputIndex = oracle.nextOutputIndex();
        warpToProposeTime(nextBlockNumber);
        vm.prank(proposer);
        oracle.proposeL2Output(proposedOutput1, nextBlockNumber, 0, 0);

        Types.OutputProposal memory proposal = oracle.getL2Output(nextOutputIndex);
        assertEq(proposal.outputRoot, proposedOutput1);
        assertEq(proposal.timestamp, block.timestamp);

        // The block number is larger than the latest proposed output:
        vm.expectRevert(stdError.indexOOBError);
        oracle.getL2Output(nextOutputIndex + 1);
    }

    // Test: getL2OutputIndexAfter() returns correct value when input is exact block
    function test_getL2OutputIndexAfter_sameBlock_succeeds() external {
        bytes32 output1 = keccak256(abi.encode(1));
        uint256 nextBlockNumber1 = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber1);
        vm.prank(proposer);
        oracle.proposeL2Output(output1, nextBlockNumber1, 0, 0);

        // Querying with exact same block as proposed returns the proposal.
        uint256 index1 = oracle.getL2OutputIndexAfter(nextBlockNumber1);
        assertEq(index1, 0);
    }

    // Test: getL2OutputIndexAfter() returns correct value when input is previous block
    function test_getL2OutputIndexAfter_previousBlock_succeeds() external {
        bytes32 output1 = keccak256(abi.encode(1));
        uint256 nextBlockNumber1 = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber1);
        vm.prank(proposer);
        oracle.proposeL2Output(output1, nextBlockNumber1, 0, 0);

        // Querying with previous block returns the proposal too.
        uint256 index1 = oracle.getL2OutputIndexAfter(nextBlockNumber1 - 1);
        assertEq(index1, 0);
    }

    // Test: getL2OutputIndexAfter() returns correct value during binary search
    function test_getL2OutputIndexAfter_multipleOutputsExist() external {
        bytes32 output1 = keccak256(abi.encode(1));
        uint256 nextBlockNumber1 = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber1);
        vm.prank(proposer);
        oracle.proposeL2Output(output1, nextBlockNumber1, 0, 0);

        bytes32 output2 = keccak256(abi.encode(2));
        uint256 nextBlockNumber2 = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber2);
        vm.prank(proposer);
        oracle.proposeL2Output(output2, nextBlockNumber2, 0, 0);

        bytes32 output3 = keccak256(abi.encode(3));
        uint256 nextBlockNumber3 = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber3);
        vm.prank(proposer);
        oracle.proposeL2Output(output3, nextBlockNumber3, 0, 0);

        bytes32 output4 = keccak256(abi.encode(4));
        uint256 nextBlockNumber4 = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber4);
        vm.prank(proposer);
        oracle.proposeL2Output(output4, nextBlockNumber4, 0, 0);

        // Querying with a block number between the first and second proposal
        uint256 index1 = oracle.getL2OutputIndexAfter(nextBlockNumber1 + 1);
        assertEq(index1, 1);

        // Querying with a block number between the second and third proposal
        uint256 index2 = oracle.getL2OutputIndexAfter(nextBlockNumber2 + 1);
        assertEq(index2, 2);

        // Querying with a block number between the third and fourth proposal
        uint256 index3 = oracle.getL2OutputIndexAfter(nextBlockNumber3 + 1);
        assertEq(index3, 3);
    }

    // Test: getL2OutputIndexAfter() reverts when no output exists yet
    function test_getL2OutputIndexAfter_noOutputsExist() external {
        vm.expectRevert("L2OutputOracle: cannot get output as no outputs have been proposed yet");
        oracle.getL2OutputIndexAfter(0);
    }

    // Test: nextBlockNumber() should return the correct value
    function test_nextBlockNumber() external {
        assertEq(
            oracle.nextBlockNumber(),
            // The return value should match this arithmetic
            oracle.latestBlockNumber() + oracle.SUBMISSION_INTERVAL()
        );
    }

    function test_computeL2Timestamp() external {
        // reverts if timestamp is too low
        vm.expectRevert(stdError.arithmeticError);
        oracle.computeL2Timestamp(startingBlockNumber - 1);

        // returns the correct value...
        // ... for the very first block
        assertEq(oracle.computeL2Timestamp(startingBlockNumber), startingTimestamp);

        // ... for the first block after the starting block
        assertEq(
            oracle.computeL2Timestamp(startingBlockNumber + 1),
            startingTimestamp + l2BlockTime
        );

        // ... for some other block number
        assertEq(
            oracle.computeL2Timestamp(startingBlockNumber + 96024),
            startingTimestamp + l2BlockTime * 96024
        );
    }

    /*****************************
     * Propose Tests - Happy Path *
     *****************************/

    // Test: proposeL2Output succeeds when given valid input, and no block hash and number are
    // specified.
    function test_proposingAnotherOutput() public {
        bytes32 proposedOutput2 = keccak256(abi.encode(2));
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);
        uint256 proposedNumber = oracle.latestBlockNumber();

        // Ensure the submissionInterval is enforced
        assertEq(nextBlockNumber, proposedNumber + submissionInterval);

        vm.roll(nextBlockNumber + 1);
        vm.prank(proposer);
        oracle.proposeL2Output(proposedOutput2, nextBlockNumber, 0, 0);
    }

    // Test: proposeL2Output succeeds when given valid input, and when a block hash and number are
    // specified for reorg protection.
    function test_proposeWithBlockhashAndHeight() external {
        // Get the number and hash of a previous block in the chain
        uint256 prevL1BlockNumber = block.number - 1;
        bytes32 prevL1BlockHash = blockhash(prevL1BlockNumber);

        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);
        vm.prank(proposer);
        oracle.proposeL2Output(nonZeroHash, nextBlockNumber, prevL1BlockHash, prevL1BlockNumber);
    }

    /***************************
     * Propose Tests - Sad Path *
     ***************************/

    // Test: proposeL2Output fails if called by a party that is not the proposer.
    function testCannot_proposeL2Output_ifNotProposer() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);

        vm.prank(address(128));
        vm.expectRevert("L2OutputOracle: only the proposer address can propose new outputs");
        oracle.proposeL2Output(nonZeroHash, nextBlockNumber, 0, 0);
    }

    // Test: proposeL2Output fails given a zero blockhash.
    function testCannot_proposeEmptyOutput() external {
        bytes32 outputToPropose = bytes32(0);
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);
        vm.prank(proposer);
        vm.expectRevert("L2OutputOracle: L2 output proposal cannot be the zero hash");
        oracle.proposeL2Output(outputToPropose, nextBlockNumber, 0, 0);
    }

    // Test: proposeL2Output fails if the block number doesn't match the next expected number.
    function testCannot_proposeUnexpectedBlockNumber() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);
        vm.prank(proposer);
        vm.expectRevert("L2OutputOracle: block number must be equal to next expected block number");
        oracle.proposeL2Output(nonZeroHash, nextBlockNumber - 1, 0, 0);
    }

    // Test: proposeL2Output fails if it would have a timestamp in the future.
    function testCannot_proposeFutureTimetamp() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        uint256 nextTimestamp = oracle.computeL2Timestamp(nextBlockNumber);
        vm.warp(nextTimestamp);
        vm.prank(proposer);
        vm.expectRevert("L2OutputOracle: cannot propose L2 output in the future");
        oracle.proposeL2Output(nonZeroHash, nextBlockNumber, 0, 0);
    }

    // Test: proposeL2Output fails if a non-existent L1 block hash and number are provided for reorg
    // protection.
    function testCannot_proposeOnWrongFork() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);
        vm.prank(proposer);
        vm.expectRevert("L2OutputOracle: blockhash does not match the hash at the expected height");
        oracle.proposeL2Output(
            nonZeroHash,
            nextBlockNumber,
            bytes32(uint256(0x01)),
            block.number - 1
        );
    }

    // Test: proposeL2Output fails when given valid input, but the block hash and number do not
    // match.
    function testCannot_ProposeWithUnmatchedBlockhash() external {
        // Move ahead to block 100 so that we can reference historical blocks
        vm.roll(100);

        // Get the number and hash of a previous block in the chain
        uint256 l1BlockNumber = block.number - 1;
        bytes32 l1BlockHash = blockhash(l1BlockNumber);

        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);
        vm.prank(proposer);

        // This will fail when foundry no longer returns zerod block hashes
        vm.expectRevert("L2OutputOracle: blockhash does not match the hash at the expected height");
        oracle.proposeL2Output(nonZeroHash, nextBlockNumber, l1BlockHash, l1BlockNumber - 1);
    }

    /*****************************
     * Delete Tests - Happy Path *
     *****************************/

    event OutputsDeleted(uint256 indexed l2BlockNumber);

    function test_deleteOutputs_singleOutput() external {
        test_proposingAnotherOutput();
        test_proposingAnotherOutput();

        uint256 latestBlockNumber = oracle.latestBlockNumber();
        uint256 latestOutputIndex = oracle.latestOutputIndex();
        Types.OutputProposal memory newLatestOutput = oracle.getL2Output(latestOutputIndex - 1);

        vm.prank(owner);
        vm.expectEmit(true, true, false, false);
        emit OutputsDeleted(latestOutputIndex);
        oracle.deleteL2Outputs(latestOutputIndex);

        // validate latestBlockNumber has been reduced
        uint256 latestBlockNumberAfter = oracle.latestBlockNumber();
        uint256 latestOutputIndexAfter = oracle.latestOutputIndex();
        assertEq(latestBlockNumber - submissionInterval, latestBlockNumberAfter);

        // validate that the new latest output is as expected.
        Types.OutputProposal memory proposal = oracle.getL2Output(latestOutputIndexAfter);
        assertEq(newLatestOutput.outputRoot, proposal.outputRoot);
        assertEq(newLatestOutput.timestamp, proposal.timestamp);
    }

    function test_deleteOutputs_multipleOutputs() external {
        test_proposingAnotherOutput();
        test_proposingAnotherOutput();
        test_proposingAnotherOutput();
        test_proposingAnotherOutput();

        uint256 latestBlockNumber = oracle.latestBlockNumber();
        uint256 latestOutputIndex = oracle.latestOutputIndex();
        Types.OutputProposal memory newLatestOutput = oracle.getL2Output(latestOutputIndex - 3);

        vm.prank(owner);
        vm.expectEmit(true, true, false, false);
        emit OutputsDeleted(latestOutputIndex - 2);
        oracle.deleteL2Outputs(latestOutputIndex - 2);

        // validate latestBlockNumber has been reduced
        uint256 latestBlockNumberAfter = oracle.latestBlockNumber();
        uint256 latestOutputIndexAfter = oracle.latestOutputIndex();
        assertEq(latestBlockNumber - submissionInterval * 3, latestBlockNumberAfter);

        // validate that the new latest output is as expected.
        Types.OutputProposal memory proposal = oracle.getL2Output(latestOutputIndexAfter);
        assertEq(newLatestOutput.outputRoot, proposal.outputRoot);
        assertEq(newLatestOutput.timestamp, proposal.timestamp);
    }

    /***************************
     * Delete Tests - Sad Path *
     ***************************/

    function testCannot_deleteL2Outputs_ifNotChallenger() external {
        uint256 latestBlockNumber = oracle.latestBlockNumber();

        vm.expectRevert("L2OutputOracle: only the challenger address can delete outputs");
        oracle.deleteL2Outputs(latestBlockNumber);
    }

    function testCannot_deleteL2Outputs_nonExistent() external {
        test_proposingAnotherOutput();

        uint256 latestBlockNumber = oracle.latestBlockNumber();

        vm.prank(owner);
        vm.expectRevert("L2OutputOracle: cannot delete outputs after the latest output index");
        oracle.deleteL2Outputs(latestBlockNumber + 1);
    }

    function testCannot_deleteL2Outputs_afterLatest() external {
        // Start by proposing three outputs
        test_proposingAnotherOutput();
        test_proposingAnotherOutput();
        test_proposingAnotherOutput();

        // Delete the latest two outputs
        uint256 latestOutputIndex = oracle.latestOutputIndex();
        vm.prank(owner);
        oracle.deleteL2Outputs(latestOutputIndex - 2);

        // Now try to delete the same output again
        vm.prank(owner);
        vm.expectRevert("L2OutputOracle: cannot delete outputs after the latest output index");
        oracle.deleteL2Outputs(latestOutputIndex - 2);
    }
}

contract L2OutputOracleUpgradeable_Test is L2OutputOracle_Initializer {
    Proxy internal proxy;

    function setUp() public override {
        super.setUp();
        proxy = Proxy(payable(address(oracle)));
    }

    function test_initValuesOnProxy() external {
        assertEq(submissionInterval, oracleImpl.SUBMISSION_INTERVAL());
        assertEq(l2BlockTime, oracleImpl.L2_BLOCK_TIME());
        assertEq(startingBlockNumber, oracleImpl.startingBlockNumber());
        assertEq(startingTimestamp, oracleImpl.startingTimestamp());

        assertEq(proposer, oracleImpl.PROPOSER());
        assertEq(owner, oracleImpl.CHALLENGER());
    }

    function test_cannotInitProxy() external {
        vm.expectRevert("Initializable: contract is already initialized");
        L2OutputOracle(payable(proxy)).initialize(
            startingBlockNumber,
            startingTimestamp
        );
    }

    function test_cannotInitImpl() external {
        vm.expectRevert("Initializable: contract is already initialized");
        L2OutputOracle(oracleImpl).initialize(
            startingBlockNumber,
            startingTimestamp
        );
    }

    function test_upgrading() external {
        // Check an unused slot before upgrading.
        bytes32 slot21Before = vm.load(address(oracle), bytes32(uint256(21)));
        assertEq(bytes32(0), slot21Before);

        NextImpl nextImpl = new NextImpl();
        vm.startPrank(multisig);
        proxy.upgradeToAndCall(
            address(nextImpl),
            abi.encodeWithSelector(NextImpl.initialize.selector)
        );
        assertEq(proxy.implementation(), address(nextImpl));

        // Verify that the NextImpl contract initialized its values according as expected
        bytes32 slot21After = vm.load(address(oracle), bytes32(uint256(21)));
        bytes32 slot21Expected = NextImpl(address(oracle)).slot21Init();
        assertEq(slot21Expected, slot21After);
    }
}
