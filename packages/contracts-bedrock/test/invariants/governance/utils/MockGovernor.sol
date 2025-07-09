// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IOptimismGovernor } from "interfaces/governance/IOptimismGovernor.sol";
import { IVotesUpgradeable } from "@openzeppelin/contracts-upgradeable/governance/utils/IVotesUpgradeable.sol";

contract MockGovernor is IOptimismGovernor {
    function propose(
        address[] memory targets,
        uint256[] memory values,
        bytes[] memory calldatas,
        string memory description,
        uint8 proposalType
    )
        external
        returns (uint256 proposalId)
    {
        return 0;
    }

    function proposeWithModule(
        address module,
        bytes memory proposalData,
        string memory description,
        uint8 proposalType
    )
        external
        returns (uint256 proposalId)
    {
        return 0;
    }

    function timelock() external view returns (address) {
        return address(0);
    }

    function PROPOSAL_TYPES_CONFIGURATOR() external view returns (address) {
        return address(0);
    }

    function token() external view returns (IVotesUpgradeable) {
        return IVotesUpgradeable(address(0));
    }

    function getProposalType(uint256 proposalId) external view returns (uint8) {
        return 0;
    }

    function proposalVotes(uint256 proposalId)
        external
        view
        returns (uint256 againstVotes, uint256 forVotes, uint256 abstainVotes)
    {
        return (0, 0, 0);
    }

    function proposalSnapshot(uint256 proposalId) external view returns (uint256) {
        return 0;
    }
}
