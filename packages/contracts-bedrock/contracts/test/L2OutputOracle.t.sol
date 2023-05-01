// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { stdError } from "forge-std/Test.sol";
import { L2OutputOracle_Initializer, NextImpl } from "./CommonTest.t.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { Proxy } from "../universal/Proxy.sol";
import { Types } from "../libraries/Types.sol";

import { BondManager } from "@dispute/BondManager.sol";
import { IBondManager } from "@dispute/interfaces/IBondManager.sol";
import { DisputeGameFactory } from "@dispute/DisputeGameFactory.sol";
import { IDisputeGameFactory } from "@dispute/interfaces/IDisputeGameFactory.sol";

contract L2OutputOracleTest is L2OutputOracle_Initializer {
    bytes32 proposedOutput1 = keccak256(abi.encode(1));

    function test_constructor_succeeds() external {
        assertEq(oracle.startingBlockNumber(), startingBlockNumber);
        assertEq(oracle.startingTimestamp(), startingTimestamp);
        assertEq(address(oracle.BOND_MANAGER()), address(bondManager));
        assertEq(address(oracle.DISPUTE_GAME_FACTORY()), address(disputeGameFactory));
        assertEq(oracle.OUTPUT_BOND_COST(), minimumProposalCost);
        assertEq(oracle.FINALIZATION_PERIOD_SECONDS(), finalizationPeriodSeconds);
    }

    function test_constructor_badTimestamp_reverts() external {
        vm.expectRevert("L2OutputOracle: starting L2 timestamp must be less than current time");
        new L2OutputOracle({
            _l2BlockTime: l2BlockTime,
            _startingBlockNumber: startingBlockNumber,
            _startingTimestamp: block.timestamp + 1,
            _finalizationPeriodSeconds: finalizationPeriodSeconds,
            _bondManager: IBondManager(address(bondManager)),
            _disputeGameFactory: IDisputeGameFactory(address(disputeGameFactory))
        });
    }

    function test_constructor_l2BlockTimeZero_reverts() external {
        vm.expectRevert("L2OutputOracle: L2 block time must be greater than 0");
        new L2OutputOracle({
            _l2BlockTime: 0,
            _startingBlockNumber: startingBlockNumber,
            _startingTimestamp: block.timestamp,
            _finalizationPeriodSeconds: finalizationPeriodSeconds,
            _bondManager: IBondManager(address(bondManager)),
            _disputeGameFactory: IDisputeGameFactory(address(disputeGameFactory))
        });
    }

    /****************
     * Getter Tests *
     ****************/

    // Test: getL2Output() should return the correct value
    function test_getL2Output_succeeds() external {
        warpToProposeTime(startingBlockNumber);
        uint256 nextOutputIndex = oracle.nextOutputIndex();

        oracle.proposeL2Output{ value: 1 ether }(proposedOutput1, startingBlockNumber, 0, 0);

        Types.OutputProposal memory proposal = oracle.getL2Output(nextOutputIndex);
        assertEq(proposal.outputRoot, proposedOutput1);
        assertEq(proposal.timestamp, block.timestamp);

        // The block number is larger than the latest proposed output:
        vm.expectRevert(stdError.indexOOBError);
        oracle.getL2Output(nextOutputIndex + 1);
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
        uint256 nextOutputIndex = oracle.nextOutputIndex();
        uint256 nextL2BlockNumber = oracle.latestBlockNumber() + 1;
        warpToProposeTime(nextL2BlockNumber);

        vm.roll(nextL2BlockNumber);

        vm.expectEmit(true, true, true, true);
        emit OutputProposed(proposedOutput2, nextOutputIndex, nextL2BlockNumber, block.timestamp);

        oracle.proposeL2Output{ value: 1 ether }(proposedOutput2, nextL2BlockNumber, 0, 0);
    }

    // Test: proposeL2Output succeeds when given valid input, and when a block hash and number are
    // specified for reorg protection.
    function test_proposeWithBlockhashAndHeight_succeeds() external {
        // Get the number and hash of a previous block in the chain
        uint256 prevL1BlockNumber = block.number - 1;
        bytes32 prevL1BlockHash = blockhash(prevL1BlockNumber);

        uint256 startingBlockNumber = oracle.startingBlockNumber();
        warpToProposeTime(startingBlockNumber);
        oracle.proposeL2Output{ value: 1 ether }(
            nonZeroHash,
            startingBlockNumber,
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
        uint256 startingBlockNumber = oracle.startingBlockNumber();
        warpToProposeTime(startingBlockNumber);
        vm.expectRevert("L2OutputOracle: L2 output proposal cannot be the zero hash");
        oracle.proposeL2Output{ value: 1 ether }(outputToPropose, startingBlockNumber, 0, 0);
    }

    // Test: proposeL2Output fails if the output is already proposed.
    function test_proposeL2Output_alreadyProposed_reverts() external {
        uint256 startingBlockNumber = oracle.startingBlockNumber();
        uint256 nextOutputIndex = oracle.nextOutputIndex();
        warpToProposeTime(startingBlockNumber);
        vm.expectEmit(true, true, true, true);
        emit OutputProposed(nonZeroHash, nextOutputIndex, startingBlockNumber, block.timestamp);
        oracle.proposeL2Output{ value: 1 ether }(nonZeroHash, startingBlockNumber, 0, 0);
        vm.expectRevert("L2OutputOracle: Output already exists at the given block number");
        oracle.proposeL2Output{ value: 1 ether }(nonZeroHash, startingBlockNumber, 0, 0);
    }

    // Test: proposeL2Output fails if it would have a timestamp in the future.
    function test_proposeL2Output_futureTimetamp_reverts() external {
        uint256 startingBlockNumber = oracle.startingBlockNumber();
        uint256 nextTimestamp = oracle.computeL2Timestamp(startingBlockNumber);
        vm.warp(nextTimestamp);
        vm.expectRevert("L2OutputOracle: cannot propose L2 output in the future");
        oracle.proposeL2Output{ value: 1 ether }(nonZeroHash, startingBlockNumber, 0, 0);
    }

    // Test: proposeL2Output fails if a non-existent L1 block hash and number are provided for reorg
    // protection.
    function test_proposeL2Output_wrongFork_reverts() external {
        uint256 startingBlockNumber = oracle.startingBlockNumber();
        warpToProposeTime(startingBlockNumber);
        vm.expectRevert(
            "L2OutputOracle: block hash does not match the hash at the expected height"
        );
        oracle.proposeL2Output{ value: 1 ether }(
            nonZeroHash,
            startingBlockNumber,
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

        uint256 startingBlockNumber = oracle.startingBlockNumber();
        warpToProposeTime(startingBlockNumber);

        // This will fail when foundry no longer returns zerod block hashes
        vm.expectRevert(
            "L2OutputOracle: block hash does not match the hash at the expected height"
        );
        oracle.proposeL2Output{ value: 1 ether }(
            nonZeroHash,
            startingBlockNumber,
            l1BlockHash,
            l1BlockNumber - 1
        );
    }
}

contract L2OutputOracleUpgradeable_Test is L2OutputOracle_Initializer {
    Proxy internal proxy;

    function setUp() public override {
        super.setUp();
        proxy = Proxy(payable(address(oracle)));
    }

    function test_initValuesOnProxy_succeeds() external {
        assertEq(l2BlockTime, oracleImpl.L2_BLOCK_TIME());
        assertEq(startingBlockNumber, oracleImpl.startingBlockNumber());
        assertEq(startingTimestamp, oracleImpl.startingTimestamp());
    }

    function test_initializeProxy_alreadyInitialized_reverts() external {
        vm.expectRevert("Initializable: contract is already initialized");
        L2OutputOracle(payable(proxy)).initialize(startingBlockNumber, startingTimestamp);
    }

    function test_initializeImpl_alreadyInitialized_reverts() external {
        vm.expectRevert("Initializable: contract is already initialized");
        L2OutputOracle(oracleImpl).initialize(startingBlockNumber, startingTimestamp);
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
