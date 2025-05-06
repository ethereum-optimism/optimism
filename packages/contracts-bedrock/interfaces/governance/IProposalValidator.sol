// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {IGovernanceToken} from "./IGovernanceToken.sol";
import {IOptimismGovernor} from "./IOptimismGovernor.sol";

/// @title IProposalValidator
/// @notice Interface for the ProposalValidator contract.
interface IProposalValidator {
    error ProposalValidator_InsufficientApprovals();
    error ProposalValidator_ProposalAlreadyApproved();
    error ProposalValidator_ProposalAlreadyInVoting();
    error ProposalValidator_InsufficientVotingPower();
    error ProposalValidator_InvalidAttestation();

    struct ProposalData {
        address proposer;
        address[] targets;
        uint256[] values;
        bytes[] calldatas;
        string description;
        ProposalType proposalType;
        bool inVoting;
        mapping(address => bool) delegateApprovals;
        uint256 remainingApprovalsRequired;
    }

    struct ImmutableProposalTypeData {
        address[] targets;
        uint256[] values;
        string[] signatures;
    }

    enum ProposalType {
        ProtocolOrGovernorUpgrade,
        MaintenanceUpgrade,
        CouncilMemberElections,
        GovernanceFund,
        CouncilBudget
    }

    event ProposalSubmitted(
        uint256 indexed proposalId,
        address indexed proposer,
        address[] targets,
        uint256[] values,
        bytes[] calldatas,
        string description,
        ProposalType proposalType
    );

    event ProposalApproved(
        uint256 indexed proposalId,
        address indexed approver
    );

    event ProposalMovedToVote(
        uint256 indexed proposalId,
        address indexed executor
    );

    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    event MinimumVotingPowerSet(uint256 newMinimumVotingPower);

    event VotingCycleBlockSet(uint256 newVotingCycleBlock);

    event DistributionThresholdSet(uint256 newDistributionThreshold);

    event ProposalApprovalThresholdSet(ProposalType proposalType, uint256 newApprovalThreshold);

    function submitProposal(
        address[] memory _targets,
        uint256[] memory _values,
        bytes[] memory _calldatas,
        string memory _description,
        ProposalType _proposalType,
        bytes32 _attestationUid
    ) external returns (uint256 proposalId_);

    function approveProposal(uint256 _proposalId) external;

    function moveToVote(uint256 _proposalId) external returns (uint256 governorProposalId_);
    
    function setMinimumVotingPower(uint256 _minimumVotingPower) external;

    function setVotingCycleBlock(uint256 _votingCycleBlock) external;

    function setDistributionThreshold(uint256 _distributionThreshold) external;

    function setProposalRequiredApprovals(ProposalType _proposalType, uint256 _requiredApprovals) external;
    
    function renounceOwnership() external;
    
    function canSignOff(address _delegate) external view returns (bool canSignOff_);
    
    function transferOwnership(address newOwner) external;

    function minimumVotingPower() external view returns (uint256);

    function votingCycleBlock() external view returns (uint256);

    function distributionThreshold() external view returns (uint256);

    function VOTING_TOKEN() external view returns (IGovernanceToken);

    function GOVERNOR() external view returns (IOptimismGovernor);

    function owner() external view returns (address);

    function ATTESTATION_SCHEMA_UID() external view returns (bytes32);
    
    function __constructor__(
        address _owner,
        IOptimismGovernor _governor,
        IGovernanceToken _votingToken,
        bytes32 _attestationSchemaUid,
        uint256 _minimumVotingPower,
        uint256 _votingCycleBlock,
        uint256 _distributionThreshold,
        ProposalType[] memory _proposalTypes,
        uint256[] memory _requiredApprovals,
        ImmutableProposalTypeData[] memory _immutableProposalTypeDatas
    ) external;
}
