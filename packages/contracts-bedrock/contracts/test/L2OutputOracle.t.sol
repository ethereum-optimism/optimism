// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

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
        assertEq(oracle.owner(), owner);
        assertEq(oracle.SUBMISSION_INTERVAL(), submissionInterval);
        assertEq(oracle.HISTORICAL_TOTAL_BLOCKS(), historicalTotalBlocks);
        assertEq(oracle.latestBlockNumber(), startingBlockNumber);
        assertEq(oracle.STARTING_BLOCK_NUMBER(), startingBlockNumber);
        assertEq(oracle.STARTING_TIMESTAMP(), startingTimestamp);
        assertEq(oracle.proposer(), proposer);
        assertEq(oracle.owner(), owner);

        Types.OutputProposal memory proposal = oracle.getL2Output(startingBlockNumber);
        assertEq(proposal.outputRoot, genesisL2Output);
        assertEq(proposal.timestamp, initL1Time);
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
        warpToProposeTime(nextBlockNumber);
        vm.prank(proposer);
        oracle.proposeL2Output(proposedOutput1, nextBlockNumber, 0, 0);

        Types.OutputProposal memory proposal = oracle.getL2Output(nextBlockNumber);
        assertEq(proposal.outputRoot, proposedOutput1);
        assertEq(proposal.timestamp, block.timestamp);

        // Handles a block number that is between checkpoints:
        proposal = oracle.getL2Output(nextBlockNumber - 1);
        assertEq(proposal.outputRoot, proposedOutput1);
        assertEq(proposal.timestamp, block.timestamp);

        // The block number is too low:
        vm.expectRevert("L2OutputOracle: block number cannot be less than the starting block number.");
        oracle.getL2Output(0);

        // The block number is larger than the latest proposed output:
        vm.expectRevert("L2OutputOracle: No output found for that block number.");
        oracle.getL2Output(nextBlockNumber + 1);
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
            "L2OutputOracle: block number must be greater than or equal to starting block number"
        );
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

    /*******************
     * Ownership tests *
     *******************/

    event ProposerChanged(address indexed previousProposer, address indexed newProposer);

    function test_changeProposer() public {
        address newProposer = address(20);
        vm.expectRevert("Ownable: caller is not the owner");
        oracle.changeProposer(newProposer);

        vm.startPrank(owner);
        vm.expectRevert("L2OutputOracle: new proposer cannot be the zero address");
        oracle.changeProposer(address(0));

        vm.expectRevert("L2OutputOracle: proposer cannot be the same as the owner");
        oracle.changeProposer(owner);

        // Double check proposer has not changed.
        assertEq(proposer, oracle.proposer());

        vm.expectEmit(true, true, true, true);
        emit ProposerChanged(proposer, newProposer);
        oracle.changeProposer(newProposer);
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
    function testCannot_proposeL2OutputIfNotProposer() external {
        uint256 nextBlockNumber = oracle.nextBlockNumber();
        warpToProposeTime(nextBlockNumber);

        vm.prank(address(128));
        vm.expectRevert("L2OutputOracle: function can only be called by proposer");
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

    event OutputDeleted(
        bytes32 indexed l2Output,
        uint256 indexed l1Timestamp,
        uint256 indexed l2BlockNumber
    );

    function test_deleteOutput() external {
        test_proposingAnotherOutput();

        uint256 latestBlockNumber = oracle.latestBlockNumber();
        Types.OutputProposal memory proposalToDelete = oracle.getL2Output(
            latestBlockNumber
        );
        Types.OutputProposal memory newLatestOutput = oracle.getL2Output(
            latestBlockNumber - submissionInterval
        );

        vm.prank(owner);
        vm.expectEmit(true, true, false, false);
        emit OutputDeleted(
            proposalToDelete.outputRoot,
            proposalToDelete.timestamp,
            latestBlockNumber
        );
        oracle.deleteL2Output(proposalToDelete);

        // validate latestBlockNumber has been reduced
        uint256 latestBlockNumberAfter = oracle.latestBlockNumber();
        assertEq(latestBlockNumber - submissionInterval, latestBlockNumberAfter);

        Types.OutputProposal memory proposal = oracle.getL2Output(latestBlockNumberAfter);
        // validate that the new latest output is as expected.
        assertEq(newLatestOutput.outputRoot, proposal.outputRoot);
        assertEq(newLatestOutput.timestamp, proposal.timestamp);
    }

    /***************************
     * Delete Tests - Sad Path *
     ***************************/

    function testCannot_deleteL2Output_ifNotOwner() external {
        uint256 latestBlockNumber = oracle.latestBlockNumber();
        Types.OutputProposal memory proposal = oracle.getL2Output(latestBlockNumber);

        vm.expectRevert("Ownable: caller is not the owner");
        oracle.deleteL2Output(proposal);
    }

    function testCannot_deleteL2Output_withWrongRoot() external {
        test_proposingAnotherOutput();

        uint256 previousBlockNumber = oracle.latestBlockNumber() - submissionInterval;
        Types.OutputProposal memory proposalToDelete = oracle.getL2Output(
            previousBlockNumber
        );

        vm.prank(owner);
        vm.expectRevert(
            "L2OutputOracle: output root to delete does not match the latest output proposal"
        );
        oracle.deleteL2Output(proposalToDelete);
    }

    function testCannot_deleteL2Output_withWrongTime() external {
        test_proposingAnotherOutput();

        uint256 latestBlockNumber = oracle.latestBlockNumber();
        Types.OutputProposal memory proposalToDelete = oracle.getL2Output(
            latestBlockNumber
        );

        // Modify the timestamp so that it does not match.
        proposalToDelete.timestamp -= 1;
        vm.prank(owner);
        vm.expectRevert(
            "L2OutputOracle: timestamp to delete does not match the latest output proposal"
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

        Types.OutputProposal memory initOutput = oracleImpl.getL2Output(
            startingBlockNumber
        );
        assertEq(genesisL2Output, initOutput.outputRoot);
        assertEq(initL1Time, initOutput.timestamp);

        assertEq(proposer, oracleImpl.proposer());
        assertEq(owner, oracleImpl.owner());
    }

    function test_cannotInitProxy() external {
        vm.expectRevert("Initializable: contract is already initialized");
        L2OutputOracle(payable(proxy)).initialize(
            genesisL2Output,
            startingBlockNumber,
            proposer,
            owner
        );
    }

    function test_cannotInitImpl() external {
        vm.expectRevert("Initializable: contract is already initialized");
        L2OutputOracle(oracleImpl).initialize(
            genesisL2Output,
            startingBlockNumber,
            proposer,
            owner
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
