pragma solidity 0.8.15;

import { L2OutputOracle } from "../../L1/L2OutputOracle.sol";
import { L2ToL1MessagePasser } from "../../L2/L2ToL1MessagePasser.sol";
import { Predeploys } from "../../libraries/Predeploys.sol";
import { Test } from "forge-std/Test.sol";

contract L2OutputOracle_Invariants is Test {
    // Global
    address multisig = address(512);

    // Test target
    L2OutputOracle oracle;

    L2ToL1MessagePasser messagePasser =
        L2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER));

    // Constructor arguments
    address internal proposer = 0x000000000000000000000000000000000000AbBa;
    address internal owner = 0x000000000000000000000000000000000000ACDC;
    uint256 internal submissionInterval = 1800;
    uint256 internal l2BlockTime = 2;
    uint256 internal startingBlockNumber = 200;
    uint256 internal startingTimestamp = 1000;

    // Test data
    uint256 initL1Time;

    // Advance the evm's time to meet the L2OutputOracle's requirements for proposeL2Output
    function warpToProposeTime(uint256 _nextBlockNumber) public {
        vm.warp(oracle.computeL2Timestamp(_nextBlockNumber) + 1);
    }

    function setUp() public virtual {
        // By default the first block has timestamp and number zero, which will cause underflows
        // in the tests, so we'll move forward to these block values.
        initL1Time = startingTimestamp + 1;
        vm.warp(initL1Time);
        vm.roll(startingBlockNumber);

        // Deploy the L2OutputOracle and transfer owernship to the proposer
        oracle = new L2OutputOracle(
            submissionInterval,
            l2BlockTime,
            startingBlockNumber,
            startingTimestamp,
            proposer,
            owner
        );
    }

    /**
     * INVARIANT: The block number of the output root proposals should monotonically increase.
     *
     * When a new output is submitted, it should never be allowed to correspond to a block number
     * that is less than the current output.
     */
    function invariant_monotonicBlockNumIncrease() external {
        // Current state of the L2OutputOracle
        uint256 nextNumber = oracle.nextBlockNumber();
        uint256 previousNumber = oracle.latestBlockNumber();

        // Warp to the time that we can propose the next L2 output
        warpToProposeTime(nextNumber);

        // Propose a mock output
        vm.prank(proposer);
        oracle.proposeL2Output(bytes32(uint256(1)), nextNumber, 0, 0);

        // Ensure that the output was proposed with the correct block number
        assertEq(oracle.latestBlockNumber(), nextNumber);
        // Ensure that the latest block number always monotonically increases
        assertTrue(nextNumber >= previousNumber);
    }

    /**
     * INVARIANT: The block number of the output root proposals should monotonically increase.
     *
     * When a new output is submitted, it should never be allowed to correspond to a block number
     * that is less than the current output.
     *
     * This is a stripped version of `invariant_monotonicBlockNumIncrease` that gives foundry's
     * invariant fuzzer less context.
     */
    function invariant_monotonicBlockNumIncrease_stripped() external {
        // Assert that the block number of proposals must monotonically increase.
        assertTrue(oracle.nextBlockNumber() >= oracle.latestBlockNumber());
    }
}
