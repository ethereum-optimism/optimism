// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import {IGovernanceToken} from './IGovernanceToken.sol';
import {IOptimismGovernor} from './IOptimismGovernor.sol';
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title IProposalValidator
/// @notice Interface for the ProposalValidator contract.
interface IProposalValidator is ISemver {
    error ProposalValidator_InsufficientApprovals();
    error ProposalValidator_ProposalAlreadyApproved();
    error ProposalValidator_ProposalAlreadySubmitted();
    error ProposalValidator_InsufficientVotingPower();
    error ProposalValidator_InvalidAttestation();
    error ProposalValidator_ProposalDoesNotExist();
    error ProposalValidator_VotingCycleAlreadySet();
    error ProposalValidator_ProposalTypesDataLengthMismatch();
    error ReinitializableBase_ZeroInitVersion();

    struct ProposalData {
        address proposer;
        ProposalType proposalType;
        bool inVoting;
        mapping(address => bool) delegateApprovals;
        uint256 approvalCount;
    }

    struct ProposalTypeData {
        uint256 requiredApprovals;
        uint8 proposalVotingModule;
    }
    
    enum ProposalType {
        ProtocolOrGovernorUpgrade,
        MaintenanceUpgrade,
        CouncilMemberElections,
        GovernanceFund,
        CouncilBudget
    }

    event ProposalSubmitted(
        bytes32 indexed proposalHash,
        address indexed proposer,
        address[] targets,
        uint256[] values,
        bytes[] calldatas,
        string description,
        ProposalType proposalType,
        uint8 proposalVotingModule
    );

    event ProposalApproved(
        bytes32 indexed proposalHash,
        address indexed approver
    );

    event ProposalMovedToVote(
        bytes32 indexed proposalHash,
        address indexed executor
    );

    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    event MinimumVotingPowerSet(uint256 newMinimumVotingPower);

    event DistributionThresholdSet(uint256 newDistributionThreshold);

    event ProposalTypeDataSet(ProposalType proposalType, uint256 requiredApprovals, uint8 proposalVotingModule);
    
    event VotingCycleDataSet(
        uint256 cycleNumber, 
        uint256 startBlock, 
        uint256 duration, 
        uint256 votingCycleDistributionLimit
    );
    
    event Initialized(uint8 version);

    function submitProposal(
        address[] memory _targets,
        uint256[] memory _values,
        bytes[] memory _calldatas,
        string memory _description,
        ProposalType _proposalType,
        bytes32 _attestationUid
    ) external returns (bytes32 proposalHash_);

    function approveProposal(bytes32 _proposalHash) external;

    function moveToVote(
        address[] memory _targets,
        uint256[] memory _values,
        bytes[] memory _calldatas,
        string memory _description
    ) external returns (uint256 governorProposalId_);
    
    function setMinimumVotingPower(uint256 _minimumVotingPower) external;

    function setDistributionThreshold(uint256 _distributionThreshold) external;
    
    function setProposalTypeData(
        ProposalType _proposalType,
        ProposalTypeData memory _proposalTypeData
    ) external;
    
    function setVotingCycleData(
        uint256 _cycleNumber,
        uint256 _startBlock,
        uint256 _duration,
        uint256 _votingCycleDistributionLimit
    ) external;
    
    function initialize(
        address _owner,
        uint256 _minimumVotingPower,
        uint256 _cycleNumber,
        uint256 _startBlock,
        uint256 _duration,
        uint256 _votingCycleDistributionLimit,
        uint256 _distributionThreshold,
        ProposalType[] memory _proposalTypes,
        ProposalTypeData[] memory _proposalTypesData
    ) external;
    
    function renounceOwnership() external;
    
    function canSignOff(address _delegate) external view returns (bool canSignOff_);
    
    function transferOwnership(address newOwner) external;

    function minimumVotingPower() external view returns (uint256);

    function distributionThreshold() external view returns (uint256);

    function VOTING_TOKEN() external view returns (IGovernanceToken);

    function GOVERNOR() external view returns (IOptimismGovernor);

    function owner() external view returns (address);

    function initVersion() external view returns (uint8);

    function ATTESTATION_SCHEMA_UID() external view returns (bytes32);
    
    function proposalTypesData(ProposalType) external view returns (uint256 requiredApprovals, uint8 proposalVotingModule);
    
    function votingCycles(uint256) external view returns (
        uint256 startingBlock, 
        uint256 duration, 
        uint256 votingCycleDistributionLimit
    );

    function __constructor__(
        bytes32 _attestationSchemaUid,
        IOptimismGovernor _governor,
        IGovernanceToken _votingToken
    ) external;
}
