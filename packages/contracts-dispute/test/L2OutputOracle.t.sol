// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { L2OutputOracle_Initializer } from "./CommonTest.t.sol";

import { GameStatus } from "src/types/Types.sol";

import { Types } from "contracts-bedrock/libraries/Types.sol";
import { L2OutputOracle } from "contracts-bedrock/L1/L2OutputOracle.sol";

/// @title L2OutputOracle Tests
contract L2OutputOracle_Test is L2OutputOracle_Initializer {
    ////////////////////////////////////////////////////////////////
    //           DELETE OUTPUT TESTS - HAPPY PATH                 //
    ////////////////////////////////////////////////////////////////

    // function test_deleteOutputs_singleOutput_succeeds() external {
    //     test_proposeL2Output_proposeAnotherOutput_succeeds();
    //     test_proposeL2Output_proposeAnotherOutput_succeeds();
    //
    //     uint256 highestL2BlockNumber = oracle.latestBlockNumber() + 1;
    //     Types.OutputProposal memory newLatestOutput = oracle.getL2Output(highestL2BlockNumber - 1);
    //
    //     vm.prank(owner);
    //     vm.expectEmit(true, true, false, false);
    //     emit OutputsDeleted(0, highestL2BlockNumber);
    //     oracle.deleteL2Output(highestL2BlockNumber);
    //
    //     // validate that the new latest output is as expected.
    //     Types.OutputProposal memory proposal = oracle.getL2Output(highestL2BlockNumber);
    //     assertEq(newLatestOutput.outputRoot, proposal.outputRoot);
    //     assertEq(newLatestOutput.timestamp, proposal.timestamp);
    // }

    /// @notice If the timestamp is past the finalization period, the deletion should revert.
    function test_deleteL2Outputs_succeeds() external {
        uint256 _l2BlockNumber = oracle.startingBlockNumber();
        uint256 proposalTimestamp = block.timestamp + oracle.FINALIZATION_PERIOD_SECONDS() + 1;
        vm.warp(proposalTimestamp);
        oracle.proposeL2Output{ value: minimumProposalCost }(
            bytes32("0x1234"),
            _l2BlockNumber,
            0,
            0
        );

        vm.warp(block.timestamp + oracle.FINALIZATION_PERIOD_SECONDS() - 1);
        vm.prank(address(disputeGameProxy));
        oracle.deleteL2Outputs(_l2BlockNumber);

        // validate that the new latest output is as expected.
        Types.OutputProposal memory proposal = oracle.getL2Output(0);
        assertEq(bytes32(""), proposal.outputRoot);
        assertEq(0, proposal.timestamp);
    }

    ////////////////////////////////////////////////////////////////
    //              DELETE OUTPUT TESTS - SAD PATH                //
    ////////////////////////////////////////////////////////////////

    function testFuzz_deleteL2Outputs_nonDisputeGame_reverts(address game) external {
        uint256 highestL2BlockNumber = oracle.startingBlockNumber();

        vm.prank(game);
        vm.expectRevert();
        oracle.deleteL2Outputs(highestL2BlockNumber);
    }

    /// @notice Calling from the implementation should revert since it's unauthorized.
    function test_deleteL2Outputs_unauthorized_reverts() external {
        uint256 highestL2BlockNumber = oracle.startingBlockNumber();
        vm.prank(address(disputeGameImplementation));
        vm.expectRevert("L2OutputOracle: Unauthorized output deletion.");
        oracle.deleteL2Outputs(highestL2BlockNumber);
    }

    /// @notice Games that are not complete should revert.
    function test_deleteL2Outputs_gameIncomplete_reverts() external {
        GameStatus gs = GameStatus.IN_PROGRESS;
        uint256 highestL2BlockNumber = oracle.startingBlockNumber();
        disputeGameProxy.setGameStatus(gs);
        vm.prank(address(disputeGameProxy));
        vm.expectRevert("L2OutputOracle: Game incomplete.");
        oracle.deleteL2Outputs(highestL2BlockNumber);
    }

    /// @notice Deleting an unknown output should revert.
    function test_deleteL2Outputs_unknown_reverts() external {
        uint256 highestL2BlockNumber = oracle.startingBlockNumber();
        vm.prank(address(disputeGameProxy));
        vm.expectRevert("L2OutputOracle: No output exists for the given L2 block number");
        oracle.deleteL2Outputs(highestL2BlockNumber);
    }

    /// @notice If the timestamp is past the finalization period, the deletion should revert.
    function test_deleteL2Outputs_finalized_reverts() external {
        uint256 _l2BlockNumber = oracle.startingBlockNumber();
        vm.warp(block.timestamp + oracle.FINALIZATION_PERIOD_SECONDS() + 1);
        oracle.proposeL2Output{ value: minimumProposalCost }(
            bytes32("0x1234"),
            _l2BlockNumber,
            0,
            0
        );

        vm.warp(block.timestamp + oracle.FINALIZATION_PERIOD_SECONDS() + 1);
        vm.prank(address(disputeGameProxy));
        vm.expectRevert("L2OutputOracle: cannot delete outputs that have already been finalized");
        oracle.deleteL2Outputs(_l2BlockNumber);
    }
}
