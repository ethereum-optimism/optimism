// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { stdError } from "forge-std/Test.sol";
import { L2OutputOracleV2_Initializer, NextImpl } from "./CommonTest.t.sol";
import { L2OutputOracleV2 } from "../L1/L2OutputOracleV2.sol";
import { Proxy } from "../universal/Proxy.sol";
import { IBondManager } from "../universal/IBondManager.sol";
import { Types } from "../libraries/Types.sol";

contract L2OutputOracleV2Test is L2OutputOracleV2_Initializer {
    bytes32 proposedOutput1 = keccak256(abi.encode(1));

    function test_constructor_succeeds() external {
        assertEq(oracle.CHALLENGER(), owner);
        assertEq(oracle.startingBlockNumber(), startingBlockNumber);
        assertEq(oracle.startingTimestamp(), startingTimestamp);
    }

    function test_constructor_badTimestamp_reverts() external {
        vm.expectRevert("L2OutputOracleV2: starting L2 timestamp must be less than current time");
        new L2OutputOracleV2({
            _l2BlockTime: l2BlockTime,
            _startingBlockNumber: startingBlockNumber,
            _startingTimestamp: block.timestamp + 1,
            _challenger: owner,
            _finalizationPeriodSeconds: 7 days,
            _bondManager: IBondManager(bondManager)
        });
    }

    function test_constructor_l2BlockTimeZero_reverts() external {
        vm.expectRevert("L2OutputOracleV2: L2 block time must be greater than 0");
        new L2OutputOracleV2({
            _l2BlockTime: 0,
            _startingBlockNumber: startingBlockNumber,
            _startingTimestamp: block.timestamp,
            _challenger: owner,
            _finalizationPeriodSeconds: 7 days,
            _bondManager: IBondManager(bondManager)
        });
    }

    /****************
     * Getter Tests *
     ****************/

    // Test: getL2Output() should return the correct value
    function test_getL2Output_succeeds() external {
        uint256 highestL2BlockNumber = oracle.highestL2BlockNumber();
        if (highestL2BlockNumber < startingBlockNumber) {
            highestL2BlockNumber = startingBlockNumber;
        }
        warpToProposeTime(highestL2BlockNumber);

        oracle.proposeL2Output{ value: 1 ether }(proposedOutput1, highestL2BlockNumber, 0, 0);

        Types.OutputProposal memory proposal = oracle.getL2Output(highestL2BlockNumber);
        assertEq(proposal.outputRoot, proposedOutput1);
        assertEq(proposal.timestamp, block.timestamp);

        // The block number is larger than the latest proposed output:
        Types.OutputProposal memory emptyProposal = oracle.getL2Output(highestL2BlockNumber + 1);
        assertEq(emptyProposal.timestamp, 0);
    }

    function test_computeL2Timestamp_succeeds() external {
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
    function test_proposeL2Output_proposeAnotherOutput_succeeds() public {
        bytes32 proposedOutput2 = keccak256(abi.encode());
        uint256 highestL2BlockNumber = oracle.highestL2BlockNumber();
        uint256 startingBlockNumber = oracle.startingBlockNumber();
        if (highestL2BlockNumber < startingBlockNumber) {
            highestL2BlockNumber = startingBlockNumber;
        }
        uint256 nextL2BlockNumber = highestL2BlockNumber + 1;
        warpToProposeTime(nextL2BlockNumber);

        vm.roll(nextL2BlockNumber);

        vm.expectEmit(true, true, true, true);
        emit OutputProposed(proposedOutput2, nextL2BlockNumber, block.timestamp);

        oracle.proposeL2Output{ value: 1 ether }(proposedOutput2, nextL2BlockNumber, 0, 0);

        address proposer = oracle.getProposer(nextL2BlockNumber);
        assertEq(proposer, address(this));
    }

    // Test: proposeL2Output succeeds when given valid input, and when a block hash and number are
    // specified for reorg protection.
    function test_proposeWithBlockhashAndHeight_succeeds() external {
        // Get the number and hash of a previous block in the chain
        uint256 prevL1BlockNumber = block.number - 1;
        bytes32 prevL1BlockHash = blockhash(prevL1BlockNumber);

        uint256 highestL2BlockNumber = oracle.highestL2BlockNumber();
        if (highestL2BlockNumber < startingBlockNumber) {
            highestL2BlockNumber = startingBlockNumber;
        }
        warpToProposeTime(highestL2BlockNumber);
        oracle.proposeL2Output{ value: 1 ether }(
            nonZeroHash,
            highestL2BlockNumber,
            prevL1BlockHash,
            prevL1BlockNumber
        );
    }

    /***************************
     * Propose Tests - Sad Path *
     ***************************/

    // Test: proposeL2Output fails given a zero blockhash.
    function test_proposeL2Output_emptyOutput_reverts() external {
        bytes32 outputToPropose = bytes32(0);
        uint256 highestL2BlockNumber = oracle.highestL2BlockNumber();
        if (highestL2BlockNumber < startingBlockNumber) {
            highestL2BlockNumber = startingBlockNumber;
        }
        warpToProposeTime(highestL2BlockNumber);
        vm.expectRevert("L2OutputOracleV2: L2 output proposal cannot be the zero hash");
        oracle.proposeL2Output{ value: 1 ether }(outputToPropose, highestL2BlockNumber, 0, 0);
    }

    // Test: proposeL2Output fails if the output is already proposed.
    function test_proposeL2Output_alreadyProposed_reverts() external {
        uint256 highestL2BlockNumber = oracle.highestL2BlockNumber();
        if (highestL2BlockNumber < startingBlockNumber) {
            highestL2BlockNumber = startingBlockNumber;
        }
        warpToProposeTime(highestL2BlockNumber);
        vm.expectEmit(true, true, true, true);
        emit OutputProposed(nonZeroHash, highestL2BlockNumber, block.timestamp);
        oracle.proposeL2Output{ value: 1 ether }(nonZeroHash, highestL2BlockNumber, 0, 0);
        vm.expectRevert("L2OutputOracleV2: output already proposed");
        oracle.proposeL2Output{ value: 1 ether }(nonZeroHash, highestL2BlockNumber, 0, 0);
    }

    // Test: proposeL2Output fails if it would have a timestamp in the future.
    function test_proposeL2Output_futureTimetamp_reverts() external {
        uint256 highestL2BlockNumber = oracle.highestL2BlockNumber();
        if (highestL2BlockNumber < startingBlockNumber) {
            highestL2BlockNumber = startingBlockNumber;
        }
        uint256 nextTimestamp = oracle.computeL2Timestamp(highestL2BlockNumber);
        vm.warp(nextTimestamp);
        vm.expectRevert("L2OutputOracleV2: cannot propose L2 output in the future");
        oracle.proposeL2Output{ value: 1 ether }(nonZeroHash, highestL2BlockNumber, 0, 0);
    }

    // Test: proposeL2Output fails if a non-existent L1 block hash and number are provided for reorg
    // protection.
    function test_proposeL2Output_wrongFork_reverts() external {
        uint256 highestL2BlockNumber = oracle.highestL2BlockNumber();
        if (highestL2BlockNumber < startingBlockNumber) {
            highestL2BlockNumber = startingBlockNumber;
        }
        warpToProposeTime(highestL2BlockNumber);
        vm.expectRevert(
            "L2OutputOracleV2: block hash does not match the hash at the expected height"
        );
        oracle.proposeL2Output{ value: 1 ether }(
            nonZeroHash,
            highestL2BlockNumber,
            bytes32(uint256(0x01)),
            block.number - 1
        );
    }

    // Test: proposeL2Output fails when given valid input, but the block hash and number do not
    // match.
    function test_proposeL2Output_unmatchedBlockhash_reverts() external {
        // Move ahead to block 100 so that we can reference historical blocks
        vm.roll(100);

        // Get the number and hash of a previous block in the chain
        uint256 l1BlockNumber = block.number - 1;
        bytes32 l1BlockHash = blockhash(l1BlockNumber);

        uint256 highestL2BlockNumber = oracle.highestL2BlockNumber();
        if (highestL2BlockNumber < startingBlockNumber) {
            highestL2BlockNumber = startingBlockNumber;
        }
        warpToProposeTime(highestL2BlockNumber);

        // This will fail when foundry no longer returns zerod block hashes
        vm.expectRevert(
            "L2OutputOracleV2: block hash does not match the hash at the expected height"
        );
        oracle.proposeL2Output{ value: 1 ether }(
            nonZeroHash,
            highestL2BlockNumber,
            l1BlockHash,
            l1BlockNumber - 1
        );
    }

    /*****************************
     * Delete Tests - Happy Path *
     *****************************/

    function test_deleteOutputs_singleOutput_succeeds() external {
        test_proposeL2Output_proposeAnotherOutput_succeeds();
        test_proposeL2Output_proposeAnotherOutput_succeeds();

        uint256 highestL2BlockNumber = oracle.highestL2BlockNumber();
        Types.OutputProposal memory newLatestOutput = oracle.getL2Output(highestL2BlockNumber - 1);

        vm.prank(owner);
        vm.expectEmit(true, true, false, false);
        emit OutputsDeleted(0, highestL2BlockNumber);
        oracle.deleteL2Output(highestL2BlockNumber);

        address proposer = oracle.getProposer(highestL2BlockNumber);
        assertEq(proposer, address(0));

        // validate highestL2BlockNumber has been reduced
        uint256 nextHighestL2BlockNumber = oracle.highestL2BlockNumber();
        assertEq(highestL2BlockNumber - 1, nextHighestL2BlockNumber);

        // validate that the new latest output is as expected.
        Types.OutputProposal memory proposal = oracle.getL2Output(nextHighestL2BlockNumber);
        assertEq(newLatestOutput.outputRoot, proposal.outputRoot);
        assertEq(newLatestOutput.timestamp, proposal.timestamp);
    }

    function test_deleteOutputs_multipleOutputs_succeeds() external {
        test_proposeL2Output_proposeAnotherOutput_succeeds();
        test_proposeL2Output_proposeAnotherOutput_succeeds();
        test_proposeL2Output_proposeAnotherOutput_succeeds();
        test_proposeL2Output_proposeAnotherOutput_succeeds();

        uint256 highestL2BlockNumber = oracle.highestL2BlockNumber();
        Types.OutputProposal memory newLatestOutput = oracle.getL2Output(highestL2BlockNumber);

        vm.startPrank(owner);
        vm.expectEmit(true, true, false, false);
        emit OutputsDeleted(highestL2BlockNumber - 2, highestL2BlockNumber - 3);
        oracle.deleteL2Output(highestL2BlockNumber - 3);
        vm.expectEmit(true, true, false, false);
        emit OutputsDeleted(highestL2BlockNumber - 1, highestL2BlockNumber - 2);
        oracle.deleteL2Output(highestL2BlockNumber - 2);
        vm.expectEmit(true, true, false, false);
        emit OutputsDeleted(highestL2BlockNumber, highestL2BlockNumber - 1);
        oracle.deleteL2Output(highestL2BlockNumber - 1);

        uint256 nextHighestL2BlockNumber = oracle.highestL2BlockNumber();
        assertEq(highestL2BlockNumber, nextHighestL2BlockNumber);
        Types.OutputProposal memory proposal = oracle.getL2Output(nextHighestL2BlockNumber);
        assertEq(newLatestOutput.outputRoot, proposal.outputRoot);
        assertEq(newLatestOutput.timestamp, proposal.timestamp);

        // Now when we delete, the highest number should be updated
        vm.expectEmit(true, true, false, false);
        emit OutputsDeleted(0, highestL2BlockNumber);
        oracle.deleteL2Output(highestL2BlockNumber);
        assertEq(0, oracle.highestL2BlockNumber());
    }

    /***************************
     * Delete Tests - Sad Path *
     ***************************/

    function test_deleteL2Output_ifNotChallenger_reverts() external {
        uint256 highestL2BlockNumber = oracle.highestL2BlockNumber();
        if (highestL2BlockNumber < startingBlockNumber) {
            highestL2BlockNumber = startingBlockNumber;
        }

        vm.expectRevert("L2OutputOracleV2: only the challenger address can delete an output");
        oracle.deleteL2Output(highestL2BlockNumber);
    }

    function test_deleteL2Output_finalized_reverts() external {
        test_proposeL2Output_proposeAnotherOutput_succeeds();

        // Warp past the finalization period + 1 second
        vm.warp(block.timestamp + oracle.FINALIZATION_PERIOD_SECONDS() + 1);

        uint256 highestL2BlockNumber = oracle.highestL2BlockNumber();

        // Try to delete a finalized output
        vm.prank(owner);
        vm.expectRevert("L2OutputOracleV2: cannot delete outputs that have already been finalized");
        oracle.deleteL2Output(highestL2BlockNumber);
    }
}

contract L2OutputOracleV2Upgradeable_Test is L2OutputOracleV2_Initializer {
    Proxy internal proxy;

    function setUp() public override {
        super.setUp();
        proxy = Proxy(payable(address(oracle)));
    }

    function test_initValuesOnProxy_succeeds() external {
        assertEq(l2BlockTime, oracleImpl.L2_BLOCK_TIME());
        assertEq(startingBlockNumber, oracleImpl.startingBlockNumber());
        assertEq(startingTimestamp, oracleImpl.startingTimestamp());

        assertEq(owner, oracleImpl.CHALLENGER());
    }

    function test_initializeProxy_alreadyInitialized_reverts() external {
        vm.expectRevert("Initializable: contract is already initialized");
        L2OutputOracleV2(payable(proxy)).initialize(startingBlockNumber, startingTimestamp);
    }

    function test_initializeImpl_alreadyInitialized_reverts() external {
        vm.expectRevert("Initializable: contract is already initialized");
        L2OutputOracleV2(oracleImpl).initialize(startingBlockNumber, startingTimestamp);
    }

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
