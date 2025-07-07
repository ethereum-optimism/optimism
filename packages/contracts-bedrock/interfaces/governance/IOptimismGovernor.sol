// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {IVotesUpgradeable} from "@openzeppelin/contracts-upgradeable/governance/utils/IVotesUpgradeable.sol";

interface IOptimismGovernor {
    function propose(
        address[] memory targets,
        uint256[] memory values,
        bytes[] memory calldatas,
        string memory description,
        uint8 proposalType
    ) external returns (uint256 proposalId);

    function proposeWithModule(
        address module,
        bytes memory proposalData,
        string memory description,
        uint8 proposalType
    ) external returns (uint256 proposalId);

    function timelock() external view returns (address);

    function PROPOSAL_TYPES_CONFIGURATOR() external view returns (address);

    function token() external view returns (IVotesUpgradeable);

    function getProposalType(uint256 proposalId) external view returns (uint8);

    function proposalVotes(uint256 proposalId)
        external
        view
        returns (uint256 againstVotes, uint256 forVotes, uint256 abstainVotes);

    /// @notice Returns the snapshot block number for a proposal, 0 if proposal doesn't exist
    /// @param proposalId The ID of the proposal
    /// @return The snapshot block number, or 0 if proposal doesn't exist
    function proposalSnapshot(uint256 proposalId) external view returns (uint256);
}