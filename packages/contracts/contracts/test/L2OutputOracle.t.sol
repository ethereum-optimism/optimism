//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { DSTest } from "../../lib/ds-test/src/test.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";

interface CheatCodes {
    function prank(address) external;
    function expectRevert(bytes calldata) external;
    function warp(uint256) external;
}

contract L2OutputOracle_Initializer is DSTest {
    // Utility variables
    CheatCodes cheats = CheatCodes(HEVM_ADDRESS);
    uint256 appendedTimestamp;

    // Test target
    L2OutputOracle oracle;

    // Constructor arguments
    address sequencer = 0x000000000000000000000000000000000000AbBa;
    uint256 submissionInterval = 1800;
    uint256 l2BlockTime = 2;
    bytes32 genesisL2Output = keccak256(abi.encode(0));
    uint256 historicalTotalBlocks = 100;

    // Cache of the initial L2 timestamp
    uint256 startingBlockTimestamp;

    // By default the first block has timestamp zero, which will cause underflows in the tests
    uint256 initTime = 1000;

    constructor() {
        // Move time forward so we have a non-zero starting timestamp
        cheats.warp(initTime);
        // Deploy the L2OutputOracle and transfer owernship to the sequencer
        oracle = new L2OutputOracle(
            submissionInterval,
            l2BlockTime,
            genesisL2Output,
            historicalTotalBlocks,
            sequencer
        );
        startingBlockTimestamp = block.timestamp;

    }
}

// Define this test in a standalone contract to ensure it runs immediately after the constructor.
contract L2OutputOracleTest_Constructor is L2OutputOracle_Initializer {
    function test_Constructor() external {
        assertEq(oracle.owner(), sequencer);
        assertEq(oracle.submissionInterval(), submissionInterval);
        assertEq(oracle.l2BlockTime(), l2BlockTime);
        assertEq(oracle.historicalTotalBlocks(), historicalTotalBlocks);
        assertEq(oracle.latestBlockTimestamp(), startingBlockTimestamp);
        assertEq(oracle.startingBlockTimestamp(), startingBlockTimestamp);
        assertEq(oracle.l2Outputs(startingBlockTimestamp), genesisL2Output);
    }
}

contract L2OutputOracleTest is L2OutputOracle_Initializer {
    bytes32 appendedOutput1 = keccak256(abi.encode(1));

    constructor() {
        appendedTimestamp = oracle.nextTimestamp();

        // Warp to after the timestamp we'll append
        cheats.warp(appendedTimestamp + 1);
        cheats.prank(sequencer);
        oracle.appendL2Output(
            appendedOutput1,
            appendedTimestamp
        );
    }

    function test_latestBlockTimestamp() external {
        assertEq(oracle.latestBlockTimestamp(), appendedTimestamp);
    }

    function test_getL2Outputs() external {
        assertEq(oracle.l2Outputs(appendedTimestamp), appendedOutput1);
    }

    function test_nextTimestamp() external {
        assertEq(
            oracle.nextTimestamp(),
            // The return value should match this arithmetic
            initTime + submissionInterval * 2
        );
    }

    function test_computesL2BlockNumber() external {
        // Test with an integer multiple of the l2BlockTime
        uint256 argTimestamp = startingBlockTimestamp + 20;
        uint256 expected = historicalTotalBlocks + 20/l2BlockTime;
        assertEq(
            oracle.computeL2BlockNumber(argTimestamp),
            expected
        );

        // Test with a remainder
        argTimestamp = startingBlockTimestamp + 33;
        expected = historicalTotalBlocks + 33/l2BlockTime;
        assertEq(
            oracle.computeL2BlockNumber(argTimestamp),
            expected
        );
    }

    function test_appendingAnotherOutput() external {
        bytes32 appendedOutput2 = keccak256(abi.encode(2));
        uint256 nextTimestamp = oracle.nextTimestamp();

        // Ensure the submissionInterval is enforced
        assertEq(nextTimestamp, appendedTimestamp + submissionInterval);

        cheats.warp(nextTimestamp + 1);
        cheats.prank(sequencer);
        oracle.appendL2Output(appendedOutput2, nextTimestamp);
    }

    function testCannot_appendOutputIfNotSequencer() external {
        bytes32 appendedOutput2 = keccak256(abi.encode(2));
        uint256 nextTimestamp = oracle.nextTimestamp();

        cheats.warp(nextTimestamp + 1);
        cheats.expectRevert("Ownable: caller is not the owner");
        oracle.appendL2Output(appendedOutput2, nextTimestamp);
    }

    function testCannot_appendEmptyOutput() external {
        bytes32 appendedOutput2 = bytes32(0);
        uint256 nextTimestamp = oracle.nextTimestamp();
        cheats.warp(nextTimestamp + 1);
        cheats.prank(sequencer);
        cheats.expectRevert("Cannot submit empty L2 output");
        oracle.appendL2Output(appendedOutput2, nextTimestamp);
    }

    function testCannot_appendUnexpectedTimestamp() external {
        bytes32 appendedOutput2 = bytes32(0);
        uint256 nextTimestamp = oracle.nextTimestamp();
        cheats.warp(nextTimestamp + 1);
        cheats.prank(sequencer);
        cheats.expectRevert("Timestamp not equal to next expected timestamp");
        oracle.appendL2Output(appendedOutput2, nextTimestamp - 1);
    }

    function testCannot_appendCurrentTimestamp() external {
        bytes32 appendedOutput2 = bytes32(0);
        uint256 nextTimestamp = oracle.nextTimestamp();
        cheats.warp(nextTimestamp + 1);
        cheats.prank(sequencer);
        cheats.expectRevert("Cannot append L2 output in future");
        oracle.appendL2Output(appendedOutput2, block.timestamp);
    }

    function testCannot_appendFutureTimestamp() external {
        bytes32 appendedOutput2 = bytes32(0);
        uint256 nextTimestamp = oracle.nextTimestamp();
        cheats.warp(nextTimestamp + 1);
        cheats.prank(sequencer);
        cheats.expectRevert("Cannot append L2 output in future");
        oracle.appendL2Output(appendedOutput2, block.timestamp);
    }

    function testCannot_computePreHistoricalL2BlockNumber() external {
        bytes memory expectedError = "Timestamp prior to startingBlockTimestamp";
        uint256 argTimestamp = startingBlockTimestamp - 1;
        cheats.expectRevert(expectedError);
        oracle.computeL2BlockNumber(argTimestamp);
    }
}
