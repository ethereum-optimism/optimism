// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IApprovalVotingModule
/// @notice Interface for the Approval Voting Module containing only the essential types
///         needed by the ProposalValidator contract.
interface IApprovalVotingModule {
    struct ProposalOption {
        uint256 budgetTokensSpent;
        address[] targets;
        uint256[] values;
        bytes[] calldatas;
        string description;
    }

    struct ProposalSettings {
        uint8 maxApprovals;
        uint8 criteria;
        address budgetToken;
        uint128 criteriaValue;
        uint128 budgetAmount;
    }

    enum PassingCriteria {
        Threshold,
        TopChoices
    }
}
