// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { stdError } from "forge-std/Test.sol";
import { L2OutputOracle_Initializer, NextImpl } from "./CommonTest.t.sol";

// Libraries
import { Types } from "../libraries/Types.sol";

// Target contract dependencies
import { Proxy } from "../universal/Proxy.sol";

// Target contract
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";

contract L2OutputOracle_constructor_Test is L2OutputOracle_Initializer {
    /// @dev Tests that constructor sets the initial values correctly.
    function test_constructor_succeeds() external {
        assertEq(oracle.PROPOSER(), proposer);
        assertEq(oracle.CHALLENGER(), owner);
        assertEq(oracle.SUBMISSION_INTERVAL(), submissionInterval);
        assertEq(oracle.latestBlockNumber(), startingBlockNumber);
        assertEq(oracle.startingBlockNumber(), startingBlockNumber);
        assertEq(oracle.startingTimestamp(), startingTimestamp);
    }

    /// @dev Tests that the constructor reverts if the starting timestamp is invalid.
    function test_constructor_badTimestamp_reverts() external {
        vm.expectRevert("L2OutputOracle: starting L2 timestamp must be less than current time");

        // startingTimestamp is in the future
        new L2OutputOracle({
            _submissionInterval: submissionInterval,
            _l2BlockTime: l2BlockTime,
            _startingBlockNumber: startingBlockNumber,
            _startingTimestamp: block.timestamp + 1,
            _proposer: proposer,
            _challenger: owner,
            _finalizationPeriodSeconds: 7 days
        });
    }

    /// @dev Tests that the constructor reverts if the l2BlockTime is invalid.
    function test_constructor_l2BlockTimeZero_reverts() external {
        vm.expectRevert("L2OutputOracle: L2 block time must be greater than 0");
        new L2OutputOracle({
            _submissionInterval: submissionInterval,
            _l2BlockTime: 0,
            _startingBlockNumber: startingBlockNumber,
            _startingTimestamp: block.timestamp,
            _proposer: proposer,
            _challenger: owner,
            _finalizationPeriodSeconds: 7 days
        });
    }

    /// @dev Tests that the constructor reverts if the submissionInterval is zero.
    function test_constructor_submissionInterval_reverts() external {
        vm.expectRevert("L2OutputOracle: submission interval must be greater than 0");
        new L2OutputOracle({
            _submissionInterval: 0,
            _l2BlockTime: l2BlockTime,
            _startingBlockNumber: startingBlockNumber,
            _startingTimestamp: block.timestamp,
            _proposer: proposer,
            _challenger: owner,
            _finalizationPeriodSeconds: 7 days
        });
    }
}

contract L2OutputOracle_getter_Test is L2OutputOracle_Initializer {
    bytes32 proposedOutput1 = keccak256(abi.encode(1));

    /// @dev Tests that `latestBlockNumber` returns the correct value.
    function test_latestBlockNumber_succeeds() external {
        uint256 proposedNumber = oracle.nextBlockNumber();

        // Roll to after the block number we'll propose
        warpToProposeTime(proposedNumber);
        vm.prank(proposer);
        oracle.proposeL2Output(proposedOutput1, proposedNumber, 0, 0);
        assertEq(oracle.latestBlockNumber(), proposedNumber);
    }

    /// @dev Tests that `getL2Output` returns the correct value.
    function test_getL2Output_succeeds() external {
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

    /// @dev Tests that `getL2OutputIndexAfter` returns the correct value
    ///      when the input is the exact block number of the proposal.
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

    /// @dev Tests that `getL2OutputIndexAfter` returns the correct value
    ///      when the input is the previous block number of the proposal.
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

    /// @dev Tests that `getL2OutputIndexAfter` returns the correct value.
    function test_getL2OutputIndexAfter_multipleOutputsExist_succeeds() external {
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

    /// @dev Tests that `getL2OutputIndexAfter` reverts when no output exists.
    function test_getL2OutputIndexAfter_noOutputsExis_reverts() external {
        vm.expectRevert("L2OutputOracle: cannot get output as no outputs have been proposed yet");
        oracle.getL2OutputIndexAfter(0);
    }

    /// @dev Tests that `nextBlockNumber` returns the correct value.
    function test_nextBlockNumber_succeeds() external {
        assertEq(
            oracle.nextBlockNumber(),
            // The return value should match this arithmetic
            oracle.latestBlockNumber() + oracle.SUBMISSION_INTERVAL()
        );
    }

    /// @dev Tests that `computeL2Timestamp` returns the correct value.
    function test_computeL2Timestamp_succeeds() external {
        // reverts if timestamp is too low
        vm.expectRevert(stdError.arithmeticError);
        oracle.computeL2Timestamp(startingBlockNumber - 1);

        // check timestamp for the very first block
        assertEq(oracle.computeL2Timestamp(startingBlockNumber), startingTimestamp);

        // check timestamp for the first block after the starting block
        assertEq(
            oracle.computeL2Timestamp(startingBlockNumber + 1),
            startingTimestamp + l2BlockTime
        );

        // check timestamp for some other block number
        assertEq(
            oracle.computeL2Timestamp(startingBlockNumber + 96024),
            startingTimestamp + l2BlockTime * 96024
        );
    }
}

contract L2OutputOracle_proposeL2Output_Test is L2OutputOracle_Initializer {
    /// @dev Test that `proposeL2Output` succeeds for a valid input
    ///      and when a block hash and number are not specified.
    function test_proposeL2Output_proposeAnotherOutput_succeeds() public {
        proposeAnotherOutput();
    }

    /// @dev Tests that `proposeL2Output` succeeds when given valid input and
    ///      when a block hash and number are specified for reorg protection.
    function test_proposeWithBlockhashAndHeight_succeeds() external {
        // Get the number and hash of a previous block in the chain
        uint256 prevL1BlockNumber = block.number - 1;
        bytes32 prevL1BlockHash = blockhash(prevL1BlockNumber);

        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);
        vm.prank(proposer);
        oracle.proposeL2Output(nonZeroHash, nextBlockNumber, prevL1BlockHash, prevL1BlockNumber);
    }

    /// @dev Tests that `proposeL2Output` reverts when called by a party
    ///      that is not the proposer.
    function test_proposeL2Output_notProposer_reverts() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);

        vm.prank(address(128));
        vm.expectRevert("L2OutputOracle: only the proposer address can propose new outputs");
        oracle.proposeL2Output(nonZeroHash, nextBlockNumber, 0, 0);
    }

    /// @dev Tests that `proposeL2Output` reverts when given a zero blockhash.
    function test_proposeL2Output_emptyOutput_reverts() external {
        bytes32 outputToPropose = bytes32(0);
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);
        vm.prank(proposer);
        vm.expectRevert("L2OutputOracle: L2 output proposal cannot be the zero hash");
        oracle.proposeL2Output(outputToPropose, nextBlockNumber, 0, 0);
    }

    /// @dev Tests that `proposeL2Output` reverts when given a block number
    ///      that does not match the next expected block number.
    function test_proposeL2Output_unexpectedBlockNumber_reverts() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);
        vm.prank(proposer);
        vm.expectRevert("L2OutputOracle: block number must be equal to next expected block number");
        oracle.proposeL2Output(nonZeroHash, nextBlockNumber - 1, 0, 0);
    }

    /// @dev Tests that `proposeL2Output` reverts when given a block number
    ///      that has a timestamp in the future.
    function test_proposeL2Output_futureTimetamp_reverts() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        uint256 nextTimestamp = oracle.computeL2Timestamp(nextBlockNumber);
        vm.warp(nextTimestamp);
        vm.prank(proposer);
        vm.expectRevert("L2OutputOracle: cannot propose L2 output in the future");
        oracle.proposeL2Output(nonZeroHash, nextBlockNumber, 0, 0);
    }

    /// @dev Tests that `proposeL2Output` reverts when given a block number
    ///      whose hash does not match the given block hash.
    function test_proposeL2Output_wrongFork_reverts() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);
        vm.prank(proposer);
        vm.expectRevert(
            "L2OutputOracle: block hash does not match the hash at the expected height"
        );
        oracle.proposeL2Output(
            nonZeroHash,
            nextBlockNumber,
            bytes32(uint256(0x01)),
            block.number - 1
        );
    }

    /// @dev Tests that `proposeL2Output` reverts when given a block number
    ///      whose block hash does not match the given block hash.
    function test_proposeL2Output_unmatchedBlockhash_reverts() external {
        // Move ahead to block 100 so that we can reference historical blocks
        vm.roll(100);

        // Get the number and hash of a previous block in the chain
        uint256 l1BlockNumber = block.number - 1;
        bytes32 l1BlockHash = blockhash(l1BlockNumber);

        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);
        vm.prank(proposer);

        // This will fail when foundry no longer returns zerod block hashes
        vm.expectRevert(
            "L2OutputOracle: block hash does not match the hash at the expected height"
        );
        oracle.proposeL2Output(nonZeroHash, nextBlockNumber, l1BlockHash, l1BlockNumber - 1);
    }
}

contract L2OutputOracle_deleteOutputs_Test is L2OutputOracle_Initializer {
    /// @dev Tests that `deleteL2Outputs` succeeds for a single output.
    function test_deleteOutputs_singleOutput_succeeds() external {
        proposeAnotherOutput();
        proposeAnotherOutput();

        uint256 latestBlockNumber = oracle.latestBlockNumber();
        uint256 latestOutputIndex = oracle.latestOutputIndex();
        Types.OutputProposal memory newLatestOutput = oracle.getL2Output(latestOutputIndex - 1);

        vm.prank(owner);
        vm.expectEmit(true, true, false, false);
        emit OutputsDeleted(latestOutputIndex + 1, latestOutputIndex);
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

    /// @dev Tests that `deleteL2Outputs` succeeds for multiple outputs.
    function test_deleteOutputs_multipleOutputs_succeeds() external {
        proposeAnotherOutput();
        proposeAnotherOutput();
        proposeAnotherOutput();
        proposeAnotherOutput();

        uint256 latestBlockNumber = oracle.latestBlockNumber();
        uint256 latestOutputIndex = oracle.latestOutputIndex();
        Types.OutputProposal memory newLatestOutput = oracle.getL2Output(latestOutputIndex - 3);

        vm.prank(owner);
        vm.expectEmit(true, true, false, false);
        emit OutputsDeleted(latestOutputIndex + 1, latestOutputIndex - 2);
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

    /// @dev Tests that `deleteL2Outputs` reverts when not called by the challenger.
    function test_deleteL2Outputs_ifNotChallenger_reverts() external {
        uint256 latestBlockNumber = oracle.latestBlockNumber();

        vm.expectRevert("L2OutputOracle: only the challenger address can delete outputs");
        oracle.deleteL2Outputs(latestBlockNumber);
    }

    /// @dev Tests that `deleteL2Outputs` reverts for a non-existant output index.
    function test_deleteL2Outputs_nonExistent_reverts() external {
        proposeAnotherOutput();

        uint256 latestBlockNumber = oracle.latestBlockNumber();

        vm.prank(owner);
        vm.expectRevert("L2OutputOracle: cannot delete outputs after the latest output index");
        oracle.deleteL2Outputs(latestBlockNumber + 1);
    }

    /// @dev Tests that `deleteL2Outputs` reverts when trying to delete outputs
    ///      after the latest output index.
    function test_deleteL2Outputs_afterLatest_reverts() external {
        proposeAnotherOutput();
        proposeAnotherOutput();
        proposeAnotherOutput();

        // Delete the latest two outputs
        uint256 latestOutputIndex = oracle.latestOutputIndex();
        vm.prank(owner);
        oracle.deleteL2Outputs(latestOutputIndex - 2);

        // Now try to delete the same output again
        vm.prank(owner);
        vm.expectRevert("L2OutputOracle: cannot delete outputs after the latest output index");
        oracle.deleteL2Outputs(latestOutputIndex - 2);
    }

    /// @dev Tests that `deleteL2Outputs` reverts for finalized outputs.
    function test_deleteL2Outputs_finalized_reverts() external {
        proposeAnotherOutput();

        // Warp past the finalization period + 1 second
        vm.warp(block.timestamp + oracle.FINALIZATION_PERIOD_SECONDS() + 1);

        uint256 latestOutputIndex = oracle.latestOutputIndex();

        // Try to delete a finalized output
        vm.prank(owner);
        vm.expectRevert("L2OutputOracle: cannot delete outputs that have already been finalized");
        oracle.deleteL2Outputs(latestOutputIndex);
    }
}

contract L2OutputOracleUpgradeable_Test is L2OutputOracle_Initializer {
    Proxy internal proxy;

    function setUp() public override {
        super.setUp();
        proxy = Proxy(payable(address(oracle)));
    }

    /// @dev Tests that the proxy is initialized with the correct values.
    function test_initValuesOnProxy_succeeds() external {
        assertEq(submissionInterval, oracleImpl.SUBMISSION_INTERVAL());
        assertEq(l2BlockTime, oracleImpl.L2_BLOCK_TIME());
        assertEq(startingBlockNumber, oracleImpl.startingBlockNumber());
        assertEq(startingTimestamp, oracleImpl.startingTimestamp());

        assertEq(proposer, oracleImpl.PROPOSER());
        assertEq(owner, oracleImpl.CHALLENGER());
    }

    /// @dev Tests that the proxy cannot be initialized twice.
    function test_initializeProxy_alreadyInitialized_reverts() external {
        vm.expectRevert("Initializable: contract is already initialized");
        L2OutputOracle(payable(proxy)).initialize(startingBlockNumber, startingTimestamp);
    }

    /// @dev Tests that the implementation contract cannot be initialized twice.
    function test_initializeImpl_alreadyInitialized_reverts() external {
        vm.expectRevert("Initializable: contract is already initialized");
        L2OutputOracle(oracleImpl).initialize(startingBlockNumber, startingTimestamp);
    }

    /// @dev Tests that the proxy can be successfully upgraded.
    function test_upgrading_succeeds() external {
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
