// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {VotingModule} from "src/governance/VotingModule.sol";

interface IOptimismGovernor {
    function propose(
        address[] memory targets,
        uint256[] memory values,
        bytes[] memory calldatas,
        string memory description,
        uint8 proposalType
    ) external returns (uint256 proposalId);

    function proposeWithModule(
        VotingModule module,
        bytes memory proposalData,
        string memory description,
        uint8 proposalType
    ) external returns (uint256 proposalId);

    function timelock() external view returns (address);

    /// @notice Returns the snapshot block number for a proposal, 0 if proposal doesn't exist
    /// @param proposalId The ID of the proposal
    /// @return The snapshot block number, or 0 if proposal doesn't exist
    function proposalSnapshot(uint256 proposalId) external view returns (uint256);
}