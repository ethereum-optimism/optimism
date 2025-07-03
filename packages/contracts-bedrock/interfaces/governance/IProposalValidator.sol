// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import {IGovernanceToken} from './IGovernanceToken.sol';
import {IOptimismGovernor} from './IOptimismGovernor.sol';
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProposalTypesConfigurator } from './IProposalTypesConfigurator.sol';

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
    error ProposalValidator_InvalidAttestationSchema();
    error ProposalValidator_InvalidCriteriaValue();
    error ProposalValidator_InvalidAgainstThreshold();
    error ProposalValidator_InvalidUpgradeProposalType();
    error ProposalValidator_InvalidVotingCycle();
    error ProposalValidator_ProposalIdMismatch();
    error ProposalValidator_InvalidProposer();
    error ProposalValidator_InvalidProposal();

    event ProposalSubmitted(
        bytes32 indexed proposalHash,
        address indexed proposer,
        string description,
        ProposalType proposalType
    );

    event ProposalApproved(
        bytes32 indexed proposalHash,
        address indexed approver
    );

    event ProposalMovedToVote(
        bytes32 indexed proposalHash,
        address indexed executor
    );

    event VotingCycleDataSet(
        uint256 cycleNumber,
        uint256 startBlock,
        uint256 duration,
        uint256 votingCycleDistributionLimit
    );

    event DistributionThresholdSet(uint256 newDistributionThreshold);

    event ProposalTypeDataSet(
        ProposalType proposalType,
        uint256 requiredApprovals,
        uint8 proposalVotingModule
    );

    event ProposalVotingModuleData(
        bytes32 indexed proposalHash,
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
        uint8 proposalVotingModule;
    }

    struct VotingCycleData {
        uint256 startingBlock;
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
        uint256 _votingCycle
    ) external returns (bytes32 proposalHash_);

    function submitCouncilMemberElectionsProposal(
        uint128 _criteriaValue,
        string[] memory _optionDescriptions,
        string memory _proposalDescription,
        bytes32 _attestationUid,
        uint256 _votingCycle
    ) external returns (bytes32 proposalHash_);

    function submitFundingProposal(
        uint128 _criteriaValue,
        string[] memory _optionsDescriptions,
        address[] memory _optionsRecipients,
        uint256[] memory _optionsAmounts,
        string memory _description,
        ProposalType _proposalType,
        uint256 _votingCycle
    ) external returns (bytes32 proposalHash_);

    function approveProposal(bytes32 _proposalHash, bytes32 _attestationUid) external;

    function canApproveProposal(bytes32 _attestationUid, address _delegate) external view returns (bool canApprove_);

    function moveToVoteProtocolOrGovernorUpgradeProposal(
        uint248 _againstThreshold,
        string memory _proposalDescription
    ) external returns (bytes32 proposalHash_);

    function moveToVoteCouncilMemberElectionsProposal(
        uint128 _criteriaValue,
        string[] memory _optionsDescriptions,
        string memory _proposalDescription
    ) external returns (bytes32 proposalHash_);

    function moveToVoteFundingProposal(
        uint128 _criteriaValue,
        string[] memory _optionsDescriptions,
        address[] memory _optionsRecipients,
        uint256[] memory _optionsAmounts,
        string memory _description,
        ProposalType _proposalType
    ) external returns (bytes32 proposalHash_);

    function setVotingCycleData(
        uint256 _cycleNumber,
        uint256 _startBlock,
        uint256 _duration,
        uint256 _votingCycleDistributionLimit
    ) external;

    function setDistributionThreshold(uint256 _distributionThreshold) external;

    function setProposalTypeData(
        ProposalType _proposalType,
        ProposalTypeData memory _proposalTypeData
    ) external;

    function initialize(
        address _owner,
        IProposalTypesConfigurator _proposalTypesConfigurator,
        uint256 _cycleNumber,
        uint256 _startBlock,
        uint256 _duration,
        uint256 _votingCycleDistributionLimit,
        uint256 _distributionThreshold,
        ProposalType[] memory _proposalTypes,
        ProposalTypeData[] memory _proposalTypesData
    ) external;

    function renounceOwnership() external;

    function transferOwnership(address newOwner) external;

    function distributionThreshold() external view returns (uint256);

    function VOTING_TOKEN() external view returns (IGovernanceToken);

    function GOVERNOR() external view returns (IOptimismGovernor);

    function owner() external view returns (address);

    function initVersion() external view returns (uint8);

    function APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID() external view returns (bytes32);

    function TOP_DELEGATES_ATTESTATION_SCHEMA_UID() external view returns (bytes32);

    function OPTIMISTIC_MODULE_PERCENT_DIVISOR() external view returns (uint256);

    function proposalTypesConfigurator() external view returns (IProposalTypesConfigurator);

    function proposalTypesData(ProposalType) external view returns (uint256 requiredApprovals, uint8 proposalVotingModule);

    function votingCycles(uint256) external view returns (
        uint256 startingBlock,
        uint256 duration,
        uint256 votingCycleDistributionLimit,
        uint256 movedToVoteTokenCount
    );

    function __constructor__(
        bytes32 _approvedProposerAttestationSchemaUid,
        bytes32 _topDelegatesAttestationSchemaUid,
        IOptimismGovernor _governor,
        IGovernanceToken _votingToken
    ) external;
}
