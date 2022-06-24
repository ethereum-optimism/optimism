//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { L2OutputOracle_Initializer, NextImpl } from "./CommonTest.t.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { Proxy } from "../universal/Proxy.sol";


contract L2OutputOracleTest is L2OutputOracle_Initializer {
    bytes32 appendedOutput1 = keccak256(abi.encode(1));

    function setUp() public override {
        super.setUp();
    }

    // Advance the evm's time to meet the L2OutputOracle's requirements for appendL2Output
    function warpToAppendTime(uint256 _nextBlockNumber) public {
        vm.warp(oracle.computeL2Timestamp(_nextBlockNumber) + 1);
    }

    function test_constructor() external {
        assertEq(oracle.owner(), owner);
        assertEq(oracle.SUBMISSION_INTERVAL(), submissionInterval);
        assertEq(oracle.HISTORICAL_TOTAL_BLOCKS(), historicalTotalBlocks);
        assertEq(oracle.latestBlockNumber(), startingBlockNumber);
        assertEq(oracle.STARTING_BLOCK_NUMBER(), startingBlockNumber);
        assertEq(oracle.STARTING_TIMESTAMP(), startingTimestamp);
        assertEq(oracle.sequencer(), sequencer);
        assertEq(oracle.owner(), owner);

        L2OutputOracle.OutputProposal memory proposal = oracle.getL2Output(startingBlockNumber);
        assertEq(proposal.outputRoot, genesisL2Output);
        assertEq(proposal.timestamp, initL1Time);
    }

    /****************
     * Getter Tests *
     ****************/

    // Test: latestBlockNumber() should return the correct value
    function test_latestBlockNumber() external {
        uint256 appendedNumber = oracle.nextBlockNumber();

        // Roll to after the block number we'll append
        warpToAppendTime(appendedNumber);
        vm.prank(sequencer);
        oracle.appendL2Output(appendedOutput1, appendedNumber, 0, 0);
        assertEq(oracle.latestBlockNumber(), appendedNumber);
    }

    // Test: getL2Output() should return the correct value
    function test_getL2Output() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToAppendTime(nextBlockNumber);
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

    function test_computeL2Timestamp() external {
        // reverts if timestamp is too low
        vm.expectRevert(
            "OutputOracle: Block number must be greater than or equal to the starting block number."
        );
        oracle.computeL2Timestamp(startingBlockNumber - 1);

        // returns the correct value...
        // ... for the very first block
        assertEq(oracle.computeL2Timestamp(startingBlockNumber), startingTimestamp);

        // ... for the first block after the starting block
        assertEq(
            oracle.computeL2Timestamp(startingBlockNumber + 1),
            startingTimestamp + submissionInterval
        );

        // ... for some other block number
        assertEq(
            oracle.computeL2Timestamp(startingBlockNumber + 96024),
            startingTimestamp + submissionInterval * 96024
        );
    }

    /*******************
     * Ownership tests *
     *******************/

    event SequencerChanged(address indexed previousSequencer, address indexed newSequencer);

    function test_changeSequencer() public {
        address newSequencer = address(20);
        vm.expectRevert("Ownable: caller is not the owner");
        oracle.changeSequencer(newSequencer);

        vm.startPrank(owner);
        vm.expectRevert("OutputOracle: new sequencer is the zero address");
        oracle.changeSequencer(address(0));

        vm.expectRevert("OutputOracle: sequencer cannot be same as the owner");
        oracle.changeSequencer(owner);

        // Double check sequencer has not changed.
        assertEq(sequencer, oracle.sequencer());

        vm.expectEmit(true, true, true, true);
        emit SequencerChanged(sequencer, newSequencer);
        oracle.changeSequencer(newSequencer);
        vm.stopPrank();
    }

    event OwnershipTransferred(address indexed, address indexed);

    function test_updateOwner() public {
        address newOwner = address(21);
        vm.expectRevert("Ownable: caller is not the owner");
        oracle.transferOwnership(newOwner);
        // Double check owner has not changed.
        assertEq(owner, oracle.owner());

        vm.startPrank(owner);
        vm.expectEmit(true, true, true, true);
        emit OwnershipTransferred(owner, newOwner);
        oracle.transferOwnership(newOwner);
        vm.stopPrank();
    }

    /*****************************
     * Append Tests - Happy Path *
     *****************************/

    // Test: appendL2Output succeeds when given valid input, and no block hash and number are
    // specified.
    function test_appendingAnotherOutput() public {
        bytes32 appendedOutput2 = keccak256(abi.encode(2));
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToAppendTime(nextBlockNumber);
        uint256 appendedNumber = oracle.latestBlockNumber();

        // Ensure the submissionInterval is enforced
        assertEq(nextBlockNumber, appendedNumber + submissionInterval);

        vm.roll(nextBlockNumber + 1);
        vm.prank(sequencer);
        oracle.appendL2Output(appendedOutput2, nextBlockNumber, 0, 0);
    }

    // Test: appendL2Output succeeds when given valid input, and when a block hash and number are
    // specified for reorg protection.
    function test_appendWithBlockhashAndHeight() external {
        // Get the number and hash of a previous block in the chain
        uint256 prevL1BlockNumber = block.number - 1;
        bytes32 prevL1BlockHash = blockhash(prevL1BlockNumber);

        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToAppendTime(nextBlockNumber);
        vm.prank(sequencer);
        oracle.appendL2Output(nonZeroHash, nextBlockNumber, prevL1BlockHash, prevL1BlockNumber);
    }

    /***************************
     * Append Tests - Sad Path *
     ***************************/

    // Test: appendL2Output fails if called by a party that is not the sequencer.
    function testCannot_appendOutputIfNotSequencer() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToAppendTime(nextBlockNumber);

        vm.prank(address(128));
        vm.expectRevert("OutputOracle: caller is not the sequencer");
        oracle.appendL2Output(nonZeroHash, nextBlockNumber, 0, 0);
    }

    // Test: appendL2Output fails given a zero blockhash.
    function testCannot_appendEmptyOutput() external {
        bytes32 outputToAppend = bytes32(0);
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToAppendTime(nextBlockNumber);
        vm.prank(sequencer);
        vm.expectRevert("OutputOracle: Cannot submit empty L2 output.");
        oracle.appendL2Output(outputToAppend, nextBlockNumber, 0, 0);
    }

    // Test: appendL2Output fails if the block number doesn't match the next expected number.
    function testCannot_appendUnexpectedBlockNumber() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToAppendTime(nextBlockNumber);
        vm.prank(sequencer);
        vm.expectRevert("OutputOracle: Block number must be equal to next expected block number.");
        oracle.appendL2Output(nonZeroHash, nextBlockNumber - 1, 0, 0);
    }

    // Test: appendL2Output fails if it would have a timestamp in the future.
    function testCannot_appendFutureTimetamp() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        uint256 nextTimestamp = oracle.computeL2Timestamp(nextBlockNumber);
        vm.warp(nextTimestamp);
        vm.prank(sequencer);
        vm.expectRevert("OutputOracle: Cannot append L2 output in future.");
        oracle.appendL2Output(nonZeroHash, nextBlockNumber, 0, 0);
    }

    // Test: appendL2Output fails if a non-existent L1 block hash and number are provided for reorg
    // protection.
    function testCannot_appendOnWrongFork() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToAppendTime(nextBlockNumber);
        vm.prank(sequencer);
        vm.expectRevert("OutputOracle: Blockhash does not match the hash at the expected height.");
        oracle.appendL2Output(
            nonZeroHash,
            nextBlockNumber,
            bytes32(uint256(0x01)),
            block.number - 1
        );
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
        warpToAppendTime(nextBlockNumber);
        vm.prank(sequencer);

        // This will fail when foundry no longer returns zerod block hashes
        oracle.appendL2Output(nonZeroHash, nextBlockNumber, l1BlockHash, l1BlockNumber - 1);
    }

    /*****************************
     * Delete Tests - Happy Path *
     *****************************/

    event L2OutputDeleted(
        bytes32 indexed _l2Output,
        uint256 indexed _l1Timestamp,
        uint256 indexed _l2BlockNumber
    );

    function test_deleteL2Output() external {
        test_appendingAnotherOutput();

        uint256 latestBlockNumber = oracle.latestBlockNumber();
        L2OutputOracle.OutputProposal memory proposalToDelete = oracle.getL2Output(
            latestBlockNumber
        );
        L2OutputOracle.OutputProposal memory newLatestOutput = oracle.getL2Output(
            latestBlockNumber - submissionInterval
        );

        vm.prank(owner);
        vm.expectEmit(true, true, false, false);
        emit L2OutputDeleted(
            proposalToDelete.outputRoot,
            proposalToDelete.timestamp,
            latestBlockNumber
        );
        oracle.deleteL2Output(proposalToDelete);

        // validate latestBlockNumber has been reduced
        uint256 latestBlockNumberAfter = oracle.latestBlockNumber();
        assertEq(latestBlockNumber - submissionInterval, latestBlockNumberAfter);

        L2OutputOracle.OutputProposal memory proposal = oracle.getL2Output(latestBlockNumberAfter);
        // validate that the new latest output is as expected.
        assertEq(newLatestOutput.outputRoot, proposal.outputRoot);
        assertEq(newLatestOutput.timestamp, proposal.timestamp);
    }

    /***************************
     * Delete Tests - Sad Path *
     ***************************/

    function testCannot_deleteL2Output_ifNotOwner() external {
        uint256 latestBlockNumber = oracle.latestBlockNumber();
        L2OutputOracle.OutputProposal memory proposal = oracle.getL2Output(latestBlockNumber);

        vm.expectRevert("Ownable: caller is not the owner");
        oracle.deleteL2Output(proposal);
    }

    function testCannot_deleteL2Output_withWrongRoot() external {
        test_appendingAnotherOutput();

        uint256 previousBlockNumber = oracle.latestBlockNumber() - submissionInterval;
        L2OutputOracle.OutputProposal memory proposalToDelete = oracle.getL2Output(
            previousBlockNumber
        );

        vm.prank(owner);
        vm.expectRevert(
            "OutputOracle: The output root to delete does not match the latest output proposal."
        );
        oracle.deleteL2Output(proposalToDelete);
    }

    function testCannot_deleteL2Output_withWrongTime() external {
        test_appendingAnotherOutput();

        uint256 latestBlockNumber = oracle.latestBlockNumber();
        L2OutputOracle.OutputProposal memory proposalToDelete = oracle.getL2Output(
            latestBlockNumber
        );

        // Modify the timestamp so that it does not match.
        proposalToDelete.timestamp -= 1;
        vm.prank(owner);
        vm.expectRevert(
            "OutputOracle: The timestamp to delete does not match the latest output proposal."
        );
        oracle.deleteL2Output(proposalToDelete);
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
        assertEq(historicalTotalBlocks, oracleImpl.HISTORICAL_TOTAL_BLOCKS());
        assertEq(startingBlockNumber, oracleImpl.STARTING_BLOCK_NUMBER());
        assertEq(startingTimestamp, oracleImpl.STARTING_TIMESTAMP());
        assertEq(l2BlockTime, oracleImpl.L2_BLOCK_TIME());

        L2OutputOracle.OutputProposal memory initOutput = oracleImpl.getL2Output(
            startingBlockNumber
        );
        assertEq(genesisL2Output, initOutput.outputRoot);
        assertEq(initL1Time, initOutput.timestamp);

        assertEq(sequencer, oracleImpl.sequencer());
        assertEq(owner, oracleImpl.owner());
    }

    function test_cannotInitProxy() external {
        vm.expectRevert("Initializable: contract is already initialized");
        address(proxy).call(abi.encodeWithSelector(L2OutputOracle.initialize.selector));
    }

    function test_cannotInitImpl() external {
        vm.expectRevert("Initializable: contract is already initialized");
        address(oracleImpl).call(abi.encodeWithSelector(L2OutputOracle.initialize.selector));
    }

    function test_upgrading() external {
        // Check an unused slot before upgrading.
        bytes32 slot21Before = vm.load(address(oracle), bytes32(uint256(21)));
        assertEq(bytes32(0), slot21Before);

        NextImpl nextImpl = new NextImpl();
        vm.startPrank(alice);
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
