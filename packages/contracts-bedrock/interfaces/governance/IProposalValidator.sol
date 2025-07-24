// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import {IOptimismGovernor} from './IOptimismGovernor.sol';
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title IProposalValidator
/// @notice Interface for the ProposalValidator contract.
interface IProposalValidator is ISemver {
    error ReinitializableBase_ZeroInitVersion();
    error ProposalValidator_InsufficientApprovals();
    error ProposalValidator_ProposalAlreadyApproved();
    error ProposalValidator_ProposalAlreadySubmitted();
    error ProposalValidator_ProposalAlreadyMovedToVote();
    error ProposalValidator_InvalidAttestation();
    error ProposalValidator_VotingCycleAlreadySet();
    error ProposalValidator_ProposalDoesNotExist();
    error ProposalValidator_ProposalTypesDataLengthMismatch();
    error ProposalValidator_InvalidFundingProposalType();
    error ProposalValidator_ExceedsDistributionThreshold();
    error ProposalValidator_InvalidOptionsLength();
    error ProposalValidator_AttestationRevoked();
    error ProposalValidator_AttestationExpired();
    error ProposalValidator_InvalidAttestationSchema();
    error ProposalValidator_InvalidCriteriaValue();
    error ProposalValidator_InvalidAgainstThreshold();
    error ProposalValidator_InvalidUpgradeProposalType();
    error ProposalValidator_InvalidVotingCycle();
    error ProposalValidator_ProposalIdMismatch();
    error ProposalValidator_InvalidProposer();
    error ProposalValidator_InvalidProposal();
    error ProposalValidator_InvalidVotingModule();
    error ProposalValidator_AttestationCreatedAfterLastVotingCycle();

    event ProposalSubmitted(
        uint256 indexed proposalId,
        address indexed proposer,
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

    event VotingCycleDataSet(
        uint256 cycleNumber,
        uint256 startingTimestamp,
        uint256 duration,
        uint256 votingCycleDistributionLimit
    );

    event ProposalDistributionThresholdSet(uint256 newProposalDistributionThreshold);

    event ProposalTypeDataSet(
        ProposalType proposalType,
        uint256 requiredApprovals,
        uint8 idInConfigurator
    );

    event ProposalVotingModuleData(
        uint256 indexed proposalId,
        bytes encodedVotingModuleData
    );

    event Initialized(uint8 version);

    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    struct ProposalData {
        address proposer;
        ProposalType proposalType;
        bool movedToVote;
        mapping(address => bool) delegateApprovals;
        uint256 approvalCount;
        uint256 votingCycle;
    }

    struct ProposalTypeData {
        uint256 requiredApprovals;
        uint8 idInConfigurator;
    }

    struct VotingCycleData {
        uint256 startingTimestamp;
        uint256 duration;
        uint256 votingCycleDistributionLimit;
        uint256 movedToVoteTokenCount;
    }

    enum ProposalType {
        ProtocolOrGovernorUpgrade,
        MaintenanceUpgrade,
        CouncilMemberElections,
        GovernanceFund,
        CouncilBudget
    }

    function submitUpgradeProposal(
        uint248 _againstThreshold,
        string memory _proposalDescription,
        bytes32 _attestationUid,
        ProposalType _proposalType,
        uint256 _latestVotingCycle
    ) external returns (uint256 proposalId_);

    function submitCouncilMemberElectionsProposal(
        uint128 _criteriaValue,
        string[] memory _optionDescriptions,
        string memory _proposalDescription,
        bytes32 _attestationUid,
        uint256 _votingCycle
    ) external returns (uint256 proposalId_);

    function submitFundingProposal(
        uint128 _criteriaValue,
        string[] memory _optionsDescriptions,
        address[] memory _optionsRecipients,
        uint256[] memory _optionsAmounts,
        string memory _description,
        ProposalType _proposalType,
        uint256 _votingCycle
    ) external returns (uint256 proposalId_);

    function approveProposal(uint256 _proposalId, bytes32 _attestationUid) external;

    function moveToVoteProtocolOrGovernorUpgradeProposal(
        uint248 _againstThreshold,
        string memory _proposalDescription
    ) external returns (uint256 proposalId_);

    function moveToVoteCouncilMemberElectionsProposal(
        uint128 _criteriaValue,
        string[] memory _optionsDescriptions,
        string memory _proposalDescription
    ) external returns (uint256 proposalId_);

    function moveToVoteFundingProposal(
        uint128 _criteriaValue,
        string[] memory _optionsDescriptions,
        address[] memory _optionsRecipients,
        uint256[] memory _optionsAmounts,
        string memory _description,
        ProposalType _proposalType
    ) external returns (uint256 proposalId_);

    function setVotingCycleData(
        uint256 _cycleNumber,
        uint256 _startingTimestamp,
        uint256 _duration,
        uint256 _votingCycleDistributionLimit
    ) external;

    function setProposalDistributionThreshold(uint256 _proposalDistributionThreshold) external;

    function setProposalTypeData(
        ProposalType _proposalType,
        ProposalTypeData memory _proposalTypeData
    ) external;

    function initialize(
        address _owner,
        uint256 _cycleNumber,
        uint256 _startingTimestamp,
        uint256 _duration,
        uint256 _votingCycleDistributionLimit,
        uint256 _proposalDistributionThreshold,
        bytes32 _approvedProposerAttestationSchemaUid,
        bytes32 _topDelegatesAttestationSchemaUid,
        ProposalType[] memory _proposalTypes,
        ProposalTypeData[] memory _proposalTypesData
    ) external;

    function renounceOwnership() external;

    function transferOwnership(address newOwner) external;

    function proposalDistributionThreshold() external view returns (uint256);

    function GOVERNOR() external view returns (IOptimismGovernor);

    function owner() external view returns (address);

    function initVersion() external view returns (uint8);

    function approvedProposerAttestationSchemaUid() external view returns (bytes32);

    function topDelegatesAttestationSchemaUid() external view returns (bytes32);

    function OPTIMISTIC_MODULE_PERCENT_DIVISOR() external view returns (uint256);

    function proposalTypesData(ProposalType) external view returns (uint256 requiredApprovals, uint8 idInConfigurator);

    function votingCycles(uint256) external view returns (
        uint256 startingTimestamp,
        uint256 duration,
        uint256 votingCycleDistributionLimit,
        uint256 movedToVoteTokenCount
    );

    function __constructor__(
        IOptimismGovernor _governor
    ) external;
}
