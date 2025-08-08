// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IOptimismGovernor } from "interfaces/governance/IOptimismGovernor.sol";
import { IProposalTypesConfigurator } from "interfaces/governance/IProposalTypesConfigurator.sol";
import { IEAS, Attestation } from "src/vendor/eas/IEAS.sol";
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IApprovalVotingModule } from "interfaces/governance/IApprovalVotingModule.sol";
import { IOptimisticModule } from "interfaces/governance/IOptimisticModule.sol";

/// @custom:proxied true
/// @title ProposalValidator
/// @notice The ProposalValidator contract is responsible for validating proposals and moving
///         them to the vote phase on the Optimism Governor.
contract ProposalValidator is OwnableUpgradeable, ReinitializableBase, ISemver {
    /*//////////////////////////////////////////////////////////////
                                 ERRORS
    //////////////////////////////////////////////////////////////*/

    /// @notice Thrown when a proposal doesn't have enough delegate approvals to move to vote.
    error ProposalValidator_InsufficientApprovals();

    /// @notice Thrown when a delegate attempts to approve a proposal they've already approved.
    error ProposalValidator_ProposalAlreadyApproved();

    /// @notice Thrown when attempting to move a proposal to vote that is already in voting.
    error ProposalValidator_ProposalAlreadySubmitted();

    /// @notice Thrown when attempting to move a proposal to vote that is already in voting.
    error ProposalValidator_ProposalAlreadyMovedToVote();

    /// @notice Thrown when an invalid attestation is provided for a proposal.
    error ProposalValidator_InvalidAttestation();

    /// @notice Thrown when a voting cycle is already set.
    error ProposalValidator_VotingCycleAlreadySet();

    /// @notice Thrown when a proposal does not exist.
    error ProposalValidator_ProposalDoesNotExist();

    /// @notice Thrown when the length of the proposal types and proposal types data arrays do not match.
    error ProposalValidator_ProposalTypesDataLengthMismatch();

    /// @notice Thrown when the proposal type is not valid for funding proposals.
    error ProposalValidator_InvalidFundingProposalType();

    /// @notice Thrown when the requested amount exceeds the distribution threshold.
    error ProposalValidator_ExceedsDistributionThreshold();

    /// @notice Thrown when the options length is invalid (zero or exceeds uint8 max).
    error ProposalValidator_InvalidOptionsLength();

    /// @notice Thrown when an attestation is revoked.
    error ProposalValidator_AttestationRevoked();

    /// @notice Thrown when the attestation is expired.
    error ProposalValidator_AttestationExpired();

    /// @notice Thrown when an attestation schema is invalid.
    error ProposalValidator_InvalidAttestationSchema();

    /// @notice Thrown when the criteria value is invalid for council elections (must not exceed options length).
    error ProposalValidator_InvalidCriteriaValue();

    /// @notice Thrown when the against threshold is invalid (must be > 0 and <= 10000 basis points).
    error ProposalValidator_InvalidAgainstThreshold();

    /// @notice Thrown when an invalid proposal type is provided for upgrade proposals.
    error ProposalValidator_InvalidUpgradeProposalType();

    /// @notice Thrown when the trying to move a proposal to vote outside of the accepted voting cycle.
    error ProposalValidator_InvalidVotingCycle();

    /// @notice Thrown when the proposalId returned by the Governor does not match the expected proposalId.
    error ProposalValidator_ProposalIdMismatch();

    /// @notice Thrown when the caller is not the proposer.
    error ProposalValidator_InvalidProposer();

    /// @notice Thrown when the proposal is invalid trying to move to vote.
    error ProposalValidator_InvalidProposal();

    /// @notice Thrown when the voting module address is invalid.
    error ProposalValidator_InvalidVotingModule();

    /// @notice Thrown when the total budget is invalid (must be > 0 and <= uint128 max).
    error ProposalValidator_InvalidTotalBudget();

    /// @notice Thrown when the attestation was created after the last voting cycle.
    error ProposalValidator_AttestationCreatedAfterLastVotingCycle();

    /// @notice Thrown when trying to approve and the previous voting cycle has not started.
    error ProposalValidator_PreviousVotingCycleNotStarted();

    /*//////////////////////////////////////////////////////////////
                                 EVENTS
    //////////////////////////////////////////////////////////////*/

    /// @notice Emitted when a new proposal is submitted.
    /// @param proposalId The ID of the submitted proposal.
    /// @param proposer The address that submitted the proposal.
    /// @param description Description of the proposal.
    /// @param proposalType Type of the proposal.
    event ProposalSubmitted(
        uint256 indexed proposalId, address indexed proposer, string description, ProposalType proposalType
    );

    /// @notice Emitted when a delegate approves a proposal.
    /// @param proposalId The ID of the approved proposal.
    /// @param approver The address of the delegate who approved the proposal.
    event ProposalApproved(uint256 indexed proposalId, address indexed approver);

    /// @notice Emitted when a proposal is moved to the voting phase in the governor contract.
    /// @param proposalId The ID of the proposal moved to vote.
    /// @param executor The address that executed the move to vote.
    event ProposalMovedToVote(uint256 indexed proposalId, address indexed executor);

    /// @notice Emitted when the voting cycle data is set.
    /// @param cycleNumber The number of the voting cycle.
    /// @param startingTimestamp The starting timestamp of the voting cycle.
    /// @param duration The duration of the voting cycle.
    /// @param votingCycleDistributionLimit The max amount of tokens that can be distributed during the voting cycle.
    event VotingCycleDataSet(
        uint256 cycleNumber, uint256 startingTimestamp, uint256 duration, uint256 votingCycleDistributionLimit
    );

    /// @notice Emitted when the proposal distribution limit is set.
    /// @param newProposalDistributionThreshold The new proposal distribution threshold.
    event ProposalDistributionThresholdSet(uint256 newProposalDistributionThreshold);

    /// @notice Emitted when the proposal type data is set.
    /// @param proposalType The type of proposal.
    /// @param requiredApprovals The required number of approvals.
    /// @param idInConfigurator The proposal type ID in the ProposalTypesConfigurator contract.
    event ProposalTypeDataSet(ProposalType proposalType, uint256 requiredApprovals, uint8 idInConfigurator);

    /// @notice Emitted with ProposalSubmitted event.
    /// @param proposalId The ID of the submitted proposal.
    /// @param encodedVotingModuleData The encoded voting module data.
    event ProposalVotingModuleData(uint256 indexed proposalId, bytes encodedVotingModuleData);

    /*//////////////////////////////////////////////////////////////
                                 STRUCTS
    //////////////////////////////////////////////////////////////*/

    /// @notice Struct for storing proposal information.
    /// @param proposer The address that submitted the proposal.
    /// @param proposalType Type of the proposal from the ProposalType enum.
    /// @param movedToVote Whether the proposal has been proposed to the Governor for voting.
    /// @param delegateApprovals Mapping of delegate addresses to their approval status.
    /// @param approvalCount Number of approvals received so far.
    /// @param votingCycle The voting cycle number the proposal is targetted for.
    struct ProposalData {
        address proposer;
        ProposalType proposalType;
        bool movedToVote;
        mapping(address => bool) delegateApprovals;
        uint256 approvalCount;
        uint256 votingCycle;
    }

    /// @notice Struct for storing explicit data for each proposal type.
    /// @param requiredApprovals The number of approvals each proposal type requires in order to be able to move for
    /// voting.
    /// @param idInConfigurator The proposal type ID used to get the voting module from the configurator.
    /// @dev Based on the spec document, funding and council member elections proposals are
    /// configured for the ApprovalVotingModule, while the upgrade proposals are configured for the
    /// OptimisticVotingModule.
    /// Any change on the module used for proposals would require the Validator to be upgraded.
    struct ProposalTypeData {
        uint256 requiredApprovals;
        uint8 idInConfigurator;
    }

    /// @notice Struct for storing voting cycle data.
    /// @param startingTimestamp The starting timestamp of the voting cycle.
    /// @param duration The duration of the voting cycle. Should be 1 day which is the end of Week 2 and start of Week 3
    /// of the voting cycle.
    /// @param votingCycleDistributionLimit The max amount of tokens that can be distributed in a proposal.
    /// @param movedToVoteTokenCount The total amount of tokens to possibly be distributed in the voting cycle.
    struct VotingCycleData {
        uint256 startingTimestamp;
        uint256 duration;
        uint256 votingCycleDistributionLimit;
        uint256 movedToVoteTokenCount;
    }

    /*//////////////////////////////////////////////////////////////
                                 ENUMS
    //////////////////////////////////////////////////////////////*/

    /// @notice Types of proposals that can be submitted.
    /// @param ProtocolOrGovernorUpgrade Proposals for upgrading the protocol or governor.
    /// @param MaintenanceUpgrade Proposals for maintenance upgrades.
    /// @param CouncilMemberElections Proposals for council member elections.
    /// @param GovernanceFund Proposals related to the governance fund.
    /// @param CouncilBudget Proposals related to the council budget.
    enum ProposalType {
        ProtocolOrGovernorUpgrade,
        MaintenanceUpgrade,
        CouncilMemberElections,
        GovernanceFund,
        CouncilBudget
    }

    /*//////////////////////////////////////////////////////////////
                               CONSTANTS
    //////////////////////////////////////////////////////////////*/

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The divisor used for percentage calculations in optimistic voting modules.
    /// @dev Represents 100% in basis points (10,000 = 100%).
    uint256 public constant OPTIMISTIC_MODULE_PERCENT_DIVISOR = 10_000;

    /*//////////////////////////////////////////////////////////////
                                 STATE VARIABLES
    //////////////////////////////////////////////////////////////*/

    /// @notice The Optimism Governor contract that will handle the voting phase.
    IOptimismGovernor public immutable GOVERNOR;

    /// @notice The schema UID for attestations in the Ethereum Attestation Service for checking if the caller
    ///         is an approved proposer.
    /// @dev Schema format: { proposalType: uint8, date: string }
    bytes32 public immutable APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID;

    /// @notice The schema UID for attestations in the Ethereum Attestation Service for checking if the caller
    ///         is part of the top100 delegates.
    bytes32 public immutable TOP_DELEGATES_ATTESTATION_SCHEMA_UID;

    /// @notice The max amount of tokens that can be distributed in a proposal.
    uint256 public proposalDistributionThreshold;

    /// @notice Mapping of voting cycle numbers to their corresponding data.
    mapping(uint256 => VotingCycleData) public votingCycles;

    /// @notice Mapping of proposal types to their corresponding data.
    mapping(ProposalType => ProposalTypeData) public proposalTypesData;

    /// @notice Mapping of proposal ID to their corresponding proposal data.
    mapping(uint256 => ProposalData) internal _proposals;

    /// @notice Constructs the ProposalValidator contract.
    /// @param _governor The Optimism Governor contract address.
    /// @param _approvedProposerAttestationSchemaUid The schema UID for attestations in the Ethereum Attestation Service
    /// for checking if the caller
    ///         is an approved proposer.
    /// @param _topDelegatesAttestationSchemaUid The schema UID for attestations in the Ethereum Attestation Service for
    /// checking if the caller
    ///         is part of the top100 delegates.
    constructor(
        IOptimismGovernor _governor,
        bytes32 _approvedProposerAttestationSchemaUid,
        bytes32 _topDelegatesAttestationSchemaUid
    )
        ReinitializableBase(1)
    {
        GOVERNOR = _governor;
        APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID = _approvedProposerAttestationSchemaUid;
        TOP_DELEGATES_ATTESTATION_SCHEMA_UID = _topDelegatesAttestationSchemaUid;
        _disableInitializers();
    }

    /// @notice Initializes the ProposalValidator contract.
    /// @param _owner The address that will own the contract.
    /// @param _cycleNumber The number of the current voting cycle.
    /// @param _startingTimestamp The starting timestamp of the voting cycle.
    /// @param _duration The duration of the voting cycle.
    /// @param _votingCycleDistributionLimit The max amount of tokens that can be distributed during the voting cycle.
    /// @param _proposalDistributionThreshold The max amount of tokens that can be distributed in a proposal.
    /// @param _proposalTypes Array of proposal types to set data for.
    /// @param _proposalTypesData Array of proposal type data corresponding to the proposal types.
    function initialize(
        address _owner,
        uint256 _cycleNumber,
        uint256 _startingTimestamp,
        uint256 _duration,
        uint256 _votingCycleDistributionLimit,
        uint256 _proposalDistributionThreshold,
        ProposalType[] memory _proposalTypes,
        ProposalTypeData[] memory _proposalTypesData
    )
        external
        reinitializer(initVersion())
    {
        if (_proposalTypes.length != _proposalTypesData.length) {
            revert ProposalValidator_ProposalTypesDataLengthMismatch();
        }

        _setVotingCycleData(_cycleNumber, _startingTimestamp, _duration, _votingCycleDistributionLimit);
        _setProposalDistributionThreshold(_proposalDistributionThreshold);

        for (uint256 i = 0; i < _proposalTypes.length; i++) {
            _setProposalTypeData(_proposalTypes[i], _proposalTypesData[i]);
        }

        __Ownable_init();
        transferOwnership(_owner);
    }

    /// @notice Submits a Protocol/Governor Upgrade or Maintenance Upgrade proposal.
    /// @param _againstThreshold The percentage that will be used to calculate the fraction of the votable supply
    /// that the proposal will need in votes against it to fail.
    /// @param _proposalDescription Description of the proposal.
    /// @param _attestationUid The UID of the attestation for the approved proposer.
    /// @param _proposalType The type of proposal (ProtocolOrGovernorUpgrade or MaintenanceUpgrade).
    /// @param _latestVotingCycle The latest voting cycle number. Even though the upgrade proposal can be submitted
    /// outside of a voting cycle, we still need the latest voting cycle number to validate top delegates attestations.
    /// @return proposalId_ The ID of the submitted proposal.
    function submitUpgradeProposal(
        uint248 _againstThreshold,
        string memory _proposalDescription,
        bytes32 _attestationUid,
        ProposalType _proposalType,
        uint256 _latestVotingCycle
    )
        external
        returns (uint256 proposalId_)
    {
        // Validate proposal type is valid for upgrade proposals
        if (_proposalType != ProposalType.ProtocolOrGovernorUpgrade && _proposalType != ProposalType.MaintenanceUpgrade)
        {
            revert ProposalValidator_InvalidUpgradeProposalType();
        }

        // Validate voting cycle exists
        if (votingCycles[_latestVotingCycle].startingTimestamp == 0) {
            revert ProposalValidator_InvalidVotingCycle();
        }

        // Validate EAS attestation - must be called by owner-approved address
        _validateApprovedProposerAttestation(_attestationUid, _proposalType);

        // Validate againstThreshold is non-zero and within bounds for percentage-based thresholds
        if (_againstThreshold == 0 || _againstThreshold > OPTIMISTIC_MODULE_PERCENT_DIVISOR) {
            revert ProposalValidator_InvalidAgainstThreshold();
        }

        // Optimistic proposals are signal-only, no execution targets/calldatas needed
        bytes memory proposalVotingModuleData = abi.encode(
            IOptimisticModule.ProposalSettings({
                againstThreshold: _againstThreshold,
                isRelativeToVotableSupply: true // MUST always be true
             })
        );

        // Retrieve the ID to use in the proposal type configurator
        uint8 idInConfigurator = proposalTypesData[_proposalType].idInConfigurator;

        // Get the optimistic module address from configurator
        address votingModule;
        {
            IProposalTypesConfigurator.ProposalType memory proposalTypeConfig =
                IProposalTypesConfigurator(GOVERNOR.PROPOSAL_TYPES_CONFIGURATOR()).proposalTypes(idInConfigurator);

            // Validate voting module exists
            if (bytes(proposalTypeConfig.name).length == 0) {
                revert ProposalValidator_InvalidVotingModule();
            }

            votingModule = proposalTypeConfig.module;
        }

        // Generate unique proposal ID
        proposalId_ =
            _hashProposalWithModule(votingModule, proposalVotingModuleData, keccak256(bytes(_proposalDescription)));

        ProposalData storage proposal = _proposals[proposalId_];

        // Prevent duplicate proposals
        if (proposal.proposer != address(0)) {
            revert ProposalValidator_ProposalAlreadySubmitted();
        }

        // Check if proposal already exists in OptimismGovernor
        if (GOVERNOR.proposalSnapshot(proposalId_) != 0) {
            revert ProposalValidator_ProposalAlreadySubmitted();
        }

        // Store proposal metadata
        proposal.proposer = _msgSender();
        proposal.proposalType = _proposalType;
        proposal.votingCycle = _latestVotingCycle;

        emit ProposalSubmitted(proposalId_, _msgSender(), _proposalDescription, _proposalType);
        emit ProposalVotingModuleData(proposalId_, proposalVotingModuleData);

        // MaintenanceUpgrade proposals move directly to voting (atomic operation)
        if (_proposalType == ProposalType.MaintenanceUpgrade) {
            proposal.movedToVote = true;

            uint256 proposalId = GOVERNOR.proposeWithModule(
                votingModule, proposalVotingModuleData, _proposalDescription, idInConfigurator
            );

            // Make sure the proposalId matches
            if (proposalId != proposalId_) {
                revert ProposalValidator_ProposalIdMismatch();
            }

            emit ProposalMovedToVote(proposalId_, _msgSender());
        }
    }

    /// @notice Submits a Council Member Elections proposal for approval and voting.
    /// @param _criteriaValue Since the passing criteria type is "TopChoices" this number represents the amount
    /// of top choices that can pass the voting.
    /// @param _optionDescriptions The strings of the different options that can be voted.
    /// @param _proposalDescription Description of the proposal.
    /// @param _attestationUid The UID of the attestation for the approved proposer.
    /// @param _votingCycle The voting cycle number the proposal is targetted for.
    /// @return proposalId_ The ID of the submitted proposal.
    function submitCouncilMemberElectionsProposal(
        uint128 _criteriaValue,
        string[] memory _optionDescriptions,
        string memory _proposalDescription,
        bytes32 _attestationUid,
        uint256 _votingCycle
    )
        external
        returns (uint256 proposalId_)
    {
        // Validate voting cycle exists and is not in the past
        VotingCycleData memory votingCycleData = votingCycles[_votingCycle];
        if (votingCycleData.startingTimestamp == 0 || votingCycleData.startingTimestamp < block.timestamp) {
            revert ProposalValidator_InvalidVotingCycle();
        }

        // Validate EAS attestation - must be called by owner-approved address
        _validateApprovedProposerAttestation(_attestationUid, ProposalType.CouncilMemberElections);

        // Validate options length bounds
        uint256 optionsLength = _optionDescriptions.length;
        if (optionsLength == 0 || optionsLength > type(uint8).max) {
            revert ProposalValidator_InvalidOptionsLength();
        }

        // Validate criteria value doesn't exceed options length for TopChoices
        if (_criteriaValue > optionsLength) {
            revert ProposalValidator_InvalidCriteriaValue();
        }

        // Build proposal options (elections don't execute operations)
        (IApprovalVotingModule.ProposalOption[] memory options,) =
            _buildApprovalModuleOptions(_optionDescriptions, new address[](0), new uint256[](0));

        // Configure approval voting settings with TopChoices criteria
        IApprovalVotingModule.ProposalSettings memory approvalSettings = IApprovalVotingModule.ProposalSettings({
            maxApprovals: uint8(optionsLength),
            criteria: uint8(IApprovalVotingModule.PassingCriteria.TopChoices),
            budgetToken: address(0), // No budget token for elections
            criteriaValue: _criteriaValue,
            budgetAmount: 0 // No budget amount for elections
         });

        bytes memory proposalVotingModuleData = abi.encode(options, approvalSettings);

        // Retrieve the ID to use in the proposal type configurator
        uint8 idInConfigurator = proposalTypesData[ProposalType.CouncilMemberElections].idInConfigurator;

        // Get the module address from the configurator
        IProposalTypesConfigurator.ProposalType memory proposalTypeConfig =
            IProposalTypesConfigurator(GOVERNOR.PROPOSAL_TYPES_CONFIGURATOR()).proposalTypes(idInConfigurator);

        // Validate voting module exists
        if (bytes(proposalTypeConfig.name).length == 0) {
            revert ProposalValidator_InvalidVotingModule();
        }

        address votingModule = proposalTypeConfig.module;

        // Generate unique proposal ID
        proposalId_ =
            _hashProposalWithModule(votingModule, proposalVotingModuleData, keccak256(bytes(_proposalDescription)));

        ProposalData storage proposal = _proposals[proposalId_];

        // Prevent duplicate proposals with same ID
        if (proposal.proposer != address(0)) {
            revert ProposalValidator_ProposalAlreadySubmitted();
        }

        // Check if proposal already exists in OptimismGovernor
        if (GOVERNOR.proposalSnapshot(proposalId_) != 0) {
            revert ProposalValidator_ProposalAlreadySubmitted();
        }

        // Store proposal metadata
        proposal.proposer = _msgSender();
        proposal.proposalType = ProposalType.CouncilMemberElections;
        proposal.votingCycle = _votingCycle;

        emit ProposalSubmitted(proposalId_, _msgSender(), _proposalDescription, ProposalType.CouncilMemberElections);
        emit ProposalVotingModuleData(proposalId_, proposalVotingModuleData);
    }

    /// @notice Submits a GovernanceFund or CouncilBudget proposal type that transfers OP tokens for approval and
    /// voting.
    /// @dev For UI integration: Frontend interfaces should present this as a percentage input to users (e.g., "25%"),
    /// then convert to the absolute vote count by calculating: (percentage / 100) * total_votable_supply.
    /// Direct contract callers must provide the absolute number of votes required for passage.
    /// @param _criteriaValue The absolute number of votes required for the proposal to pass. This represents the
    /// threshold that must be met or exceeded for any option to be considered successful.
    /// @param _optionsDescriptions The strings of the different options that can be voted.
    /// @param _optionsRecipients An address for each option to transfer funds to in case the option passes the voting.
    /// @param _optionsAmounts The amount to transfer for each option in case the option passes the voting.
    /// @param _description Description of the proposal.
    /// @param _proposalType The type of proposal (must be GovernanceFund or CouncilBudget).
    /// @param _votingCycle The voting cycle number the proposal is targetted for.
    /// @return proposalId_ The ID of the submitted proposal.
    function submitFundingProposal(
        uint128 _criteriaValue,
        string[] memory _optionsDescriptions,
        address[] memory _optionsRecipients,
        uint256[] memory _optionsAmounts,
        string memory _description,
        ProposalType _proposalType,
        uint256 _votingCycle
    )
        external
        returns (uint256 proposalId_)
    {
        // Only funding proposal types can use this function
        if (_proposalType != ProposalType.GovernanceFund && _proposalType != ProposalType.CouncilBudget) {
            revert ProposalValidator_InvalidFundingProposalType();
        }

        // Validate voting cycle exists and is not in the past
        VotingCycleData memory votingCycleData = votingCycles[_votingCycle];
        if (votingCycleData.startingTimestamp == 0 || votingCycleData.startingTimestamp < block.timestamp) {
            revert ProposalValidator_InvalidVotingCycle();
        }

        // Validate input arrays have matching lengths
        uint256 optionsLength = _optionsDescriptions.length;
        if (optionsLength != _optionsRecipients.length || optionsLength != _optionsAmounts.length) {
            revert ProposalValidator_ProposalTypesDataLengthMismatch();
        }

        // Validate options length bounds
        if (optionsLength == 0 || optionsLength > type(uint8).max) {
            revert ProposalValidator_InvalidOptionsLength();
        }

        // Build proposal options with funding execution data
        (IApprovalVotingModule.ProposalOption[] memory options, uint256 totalBudget) =
            _buildApprovalModuleOptions(_optionsDescriptions, _optionsRecipients, _optionsAmounts);

        // Configure approval voting settings
        IApprovalVotingModule.ProposalSettings memory approvalSettings = IApprovalVotingModule.ProposalSettings({
            maxApprovals: uint8(optionsLength),
            criteria: uint8(IApprovalVotingModule.PassingCriteria.Threshold),
            budgetToken: Predeploys.GOVERNANCE_TOKEN,
            criteriaValue: _criteriaValue,
            budgetAmount: uint128(totalBudget)
        });

        bytes memory proposalVotingModuleData = abi.encode(options, approvalSettings);

        // Retrieve the ID to use in the proposal type configurator
        uint8 idInConfigurator = proposalTypesData[_proposalType].idInConfigurator;

        // Get the module address from the configurator
        address votingModule;
        {
            IProposalTypesConfigurator.ProposalType memory proposalTypeConfig =
                IProposalTypesConfigurator(GOVERNOR.PROPOSAL_TYPES_CONFIGURATOR()).proposalTypes(idInConfigurator);

            // Validate voting module exists
            if (bytes(proposalTypeConfig.name).length == 0) {
                revert ProposalValidator_InvalidVotingModule();
            }

            votingModule = proposalTypeConfig.module;
        }

        // Generate unique proposal ID
        proposalId_ = _hashProposalWithModule(votingModule, proposalVotingModuleData, keccak256(bytes(_description)));

        ProposalData storage proposal = _proposals[proposalId_];

        // Prevent duplicate proposals with same ID
        if (proposal.proposer != address(0)) {
            revert ProposalValidator_ProposalAlreadySubmitted();
        }

        // Check if proposal already exists in OptimismGovernor
        if (GOVERNOR.proposalSnapshot(proposalId_) != 0) {
            revert ProposalValidator_ProposalAlreadySubmitted();
        }

        // Store proposal metadata
        proposal.proposer = _msgSender();
        proposal.proposalType = _proposalType;
        proposal.votingCycle = _votingCycle;

        emit ProposalSubmitted(proposalId_, _msgSender(), _description, _proposalType);
        emit ProposalVotingModuleData(proposalId_, proposalVotingModuleData);
    }

    /// @notice Approves a proposal before being moved for voting.
    /// @dev This function should only be called by the top delegates.
    /// @param _proposalId The ID of the proposal to approve
    /// @param _attestationUid The UID of the attestation for the delegate to approve the proposal
    function approveProposal(uint256 _proposalId, bytes32 _attestationUid) external {
        address _delegate = _msgSender();
        ProposalData storage proposal = _proposals[_proposalId];
        // check if the proposal exists
        // proposal.votingCycle should never be 0, voting cycles already exist before the ProposalValidator is deployed
        // and should be set by the OP Foundation
        if (proposal.proposer == address(0) || proposal.votingCycle == 0) {
            revert ProposalValidator_ProposalDoesNotExist();
        }

        // check if the caller has already approved the proposal
        if (proposal.delegateApprovals[_delegate]) {
            revert ProposalValidator_ProposalAlreadyApproved();
        }

        // check if proposal has already moved to vote
        if (proposal.movedToVote) {
            revert ProposalValidator_ProposalAlreadyMovedToVote();
        }

        // The previous voting cycle of a proposal should be the one before the
        // proposal's targetted voting cycle.
        uint256 previousVotingCycle = proposal.votingCycle - 1;
        // Proposal or Governor Upgrade proposals are submitted with the latest voting cycle number,
        // because they can be submitted outside of a voting cycle.
        if (proposal.proposalType == ProposalType.ProtocolOrGovernorUpgrade) {
            previousVotingCycle = proposal.votingCycle;
        }

        // revert if the previous voting cycle has not started, we should only  allow delegates
        // to approve relative close to the proposals voting cycle
        if (votingCycles[previousVotingCycle].startingTimestamp > block.timestamp) {
            revert ProposalValidator_PreviousVotingCycleNotStarted();
        }

        // validate the attestation
        _validateTopDelegateAttestation(_attestationUid, previousVotingCycle);

        // store the approval
        proposal.delegateApprovals[_delegate] = true;
        proposal.approvalCount++;

        emit ProposalApproved(_proposalId, _delegate);
    }

    /// @notice Moves a Protocol or Governor Upgrade proposal to vote by proposing it on the Governor.
    /// @param _againstThreshold The threshold for the proposal to be against the total supply.
    /// @param _proposalDescription Description of the proposal.
    /// @return proposalId_ The ID of the submitted proposal.
    function moveToVoteProtocolOrGovernorUpgradeProposal(
        uint248 _againstThreshold,
        string memory _proposalDescription
    )
        external
        returns (uint256 proposalId_)
    {
        // Configure optimistic proposal settings
        IOptimisticModule.ProposalSettings memory optimisticSettings =
            IOptimisticModule.ProposalSettings({ againstThreshold: _againstThreshold, isRelativeToVotableSupply: true });

        bytes memory proposalVotingModuleData = abi.encode(optimisticSettings);

        // Retrieve the ID to use in the proposal type configurator
        uint8 idInConfigurator = proposalTypesData[ProposalType.ProtocolOrGovernorUpgrade].idInConfigurator;

        // Get the module address from the configurator
        ProposalType proposalType = ProposalType.ProtocolOrGovernorUpgrade;
        address votingModule =
            IProposalTypesConfigurator(GOVERNOR.PROPOSAL_TYPES_CONFIGURATOR()).proposalTypes(idInConfigurator).module;

        // Generate unique proposal ID
        proposalId_ =
            _hashProposalWithModule(votingModule, proposalVotingModuleData, keccak256(bytes(_proposalDescription)));

        ProposalData storage proposal = _proposals[proposalId_];

        // Proposal must exist and be valid
        if (proposal.proposer == address(0) || proposal.proposalType != proposalType) {
            revert ProposalValidator_InvalidProposal();
        }

        // Check if the caller is the proposer
        if (proposal.proposer != _msgSender()) {
            revert ProposalValidator_InvalidProposer();
        }

        // Check if proposal has enough approvals
        if (proposal.approvalCount < proposalTypesData[proposalType].requiredApprovals) {
            revert ProposalValidator_InsufficientApprovals();
        }

        // Check if proposal is already in voting
        if (proposal.movedToVote) {
            revert ProposalValidator_ProposalAlreadyMovedToVote();
        }

        proposal.movedToVote = true;

        // Propose with module on the Governor
        uint256 proposalId =
            GOVERNOR.proposeWithModule(votingModule, proposalVotingModuleData, _proposalDescription, idInConfigurator);

        // Make sure the proposalId matches
        if (proposalId != proposalId_) {
            revert ProposalValidator_ProposalIdMismatch();
        }

        emit ProposalMovedToVote(proposalId_, _msgSender());
    }

    /// @notice Moves a council member elections proposal to vote by proposing it on the Governor.
    /// @param _criteriaValue The number of top choices that can pass the voting.
    /// @param _optionsDescriptions The strings of the different options that can be voted.
    /// @param _proposalDescription Description of the proposal.
    /// @return proposalId_ The ID of the submitted proposal.
    function moveToVoteCouncilMemberElectionsProposal(
        uint128 _criteriaValue,
        string[] memory _optionsDescriptions,
        string memory _proposalDescription
    )
        external
        returns (uint256 proposalId_)
    {
        uint256 optionsLength = _optionsDescriptions.length;
        // Validate options length bounds
        if (optionsLength == 0 || optionsLength > type(uint8).max) {
            revert ProposalValidator_InvalidOptionsLength();
        }
        // Configure approval module options
        (IApprovalVotingModule.ProposalOption[] memory options,) =
            _buildApprovalModuleOptions(_optionsDescriptions, new address[](0), new uint256[](0));

        // Configure approval module settings
        IApprovalVotingModule.ProposalSettings memory approvalSettings = IApprovalVotingModule.ProposalSettings({
            maxApprovals: uint8(optionsLength),
            criteria: uint8(IApprovalVotingModule.PassingCriteria.TopChoices),
            budgetToken: address(0),
            criteriaValue: _criteriaValue,
            budgetAmount: 0
        });

        bytes memory proposalVotingModuleData = abi.encode(options, approvalSettings);

        ProposalType _proposalType = ProposalType.CouncilMemberElections;

        // Retrieve the ID to use in the proposal type configurator
        uint8 idInConfigurator = proposalTypesData[_proposalType].idInConfigurator;

        // Get the module address from the configurator
        address votingModule =
            IProposalTypesConfigurator(GOVERNOR.PROPOSAL_TYPES_CONFIGURATOR()).proposalTypes(idInConfigurator).module;

        // Generate unique proposal ID
        proposalId_ =
            _hashProposalWithModule(votingModule, proposalVotingModuleData, keccak256(bytes(_proposalDescription)));

        ProposalData storage proposal = _proposals[proposalId_];

        // Proposal must exist and be valid
        if (proposal.proposer == address(0) || proposal.proposalType != _proposalType) {
            revert ProposalValidator_InvalidProposal();
        }

        // Check if the caller is the proposer
        if (proposal.proposer != _msgSender()) {
            revert ProposalValidator_InvalidProposer();
        }

        // Check if proposal has enough approvals
        if (proposal.approvalCount < proposalTypesData[_proposalType].requiredApprovals) {
            revert ProposalValidator_InsufficientApprovals();
        }

        // Check if proposal is already in voting
        if (proposal.movedToVote) {
            revert ProposalValidator_ProposalAlreadyMovedToVote();
        }

        // Check if the voting cycle is valid
        VotingCycleData memory votingCycleData = votingCycles[proposal.votingCycle];
        if (
            votingCycleData.startingTimestamp > block.timestamp
                || votingCycleData.startingTimestamp + votingCycleData.duration < block.timestamp
        ) {
            revert ProposalValidator_InvalidVotingCycle();
        }

        proposal.movedToVote = true;

        // Propose with module on the Governor
        uint256 proposalId =
            GOVERNOR.proposeWithModule(votingModule, proposalVotingModuleData, _proposalDescription, idInConfigurator);

        // Make sure the proposalId matches
        if (proposalId != proposalId_) {
            revert ProposalValidator_ProposalIdMismatch();
        }

        emit ProposalMovedToVote(proposalId_, _msgSender());
    }

    /// @notice Moves a funding proposal to vote by proposing it on the Governor.
    /// @dev For UI integration: Frontend interfaces should present this as a percentage input to users (e.g., "25%"),
    /// then convert to the absolute vote count by calculating: (percentage / 100) * total_votable_supply.
    /// Direct contract callers must provide the absolute number of votes required for passage.
    /// @param _criteriaValue The absolute number of votes required for the proposal to pass. This represents the
    /// threshold that must be met or exceeded for any option to be considered successful.
    /// @param _optionsDescriptions The strings of the different options that can be voted.
    /// @param _optionsRecipients An address for each option to transfer funds to in case the option passes the voting.
    /// @param _optionsAmounts The amount to transfer for each option in case the option passes the voting.
    /// @param _description Description of the proposal.
    /// @param _proposalType The type of proposal (must be GovernanceFund or CouncilBudget).
    /// @return proposalId_ The ID of the submitted proposal.
    function moveToVoteFundingProposal(
        uint128 _criteriaValue,
        string[] memory _optionsDescriptions,
        address[] memory _optionsRecipients,
        uint256[] memory _optionsAmounts,
        string memory _description,
        ProposalType _proposalType
    )
        external
        returns (uint256 proposalId_)
    {
        // Only funding proposal types can use this function
        if (_proposalType != ProposalType.GovernanceFund && _proposalType != ProposalType.CouncilBudget) {
            revert ProposalValidator_InvalidFundingProposalType();
        }

        uint256 optionsLength = _optionsDescriptions.length;
        // Validate options length bounds
        if (optionsLength == 0 || optionsLength > type(uint8).max) {
            revert ProposalValidator_InvalidOptionsLength();
        }

        // Configure approval module options
        (IApprovalVotingModule.ProposalOption[] memory options, uint256 totalBudget) =
            _buildApprovalModuleOptions(_optionsDescriptions, _optionsRecipients, _optionsAmounts);

        // Configure approval module settings
        IApprovalVotingModule.ProposalSettings memory approvalSettings = IApprovalVotingModule.ProposalSettings({
            maxApprovals: uint8(_optionsDescriptions.length),
            criteria: uint8(IApprovalVotingModule.PassingCriteria.Threshold),
            budgetToken: Predeploys.GOVERNANCE_TOKEN,
            criteriaValue: _criteriaValue,
            budgetAmount: uint128(totalBudget)
        });

        bytes memory proposalVotingModuleData = abi.encode(options, approvalSettings);

        // Retrieve the ID to use in the proposal type configurator
        uint8 idInConfigurator = proposalTypesData[_proposalType].idInConfigurator;

        // Get the module address from the configurator
        address votingModule =
            IProposalTypesConfigurator(GOVERNOR.PROPOSAL_TYPES_CONFIGURATOR()).proposalTypes(idInConfigurator).module;

        // Generate unique proposal ID
        proposalId_ = _hashProposalWithModule(votingModule, proposalVotingModuleData, keccak256(bytes(_description)));

        ProposalData storage proposal = _proposals[proposalId_];

        // Proposal must exist
        if (proposal.proposer == address(0) || proposal.proposalType != _proposalType) {
            revert ProposalValidator_InvalidProposal();
        }

        // Check if proposal has enough approvals
        if (proposal.approvalCount < proposalTypesData[_proposalType].requiredApprovals) {
            revert ProposalValidator_InsufficientApprovals();
        }

        // Check if proposal is already in voting
        if (proposal.movedToVote) {
            revert ProposalValidator_ProposalAlreadyMovedToVote();
        }

        {
            // Check if proposal can be moved to vote
            VotingCycleData memory votingCycleData = votingCycles[proposal.votingCycle];
            if (
                votingCycleData.startingTimestamp > block.timestamp
                    || votingCycleData.startingTimestamp + votingCycleData.duration < block.timestamp
            ) {
                revert ProposalValidator_InvalidVotingCycle();
            }

            // Check if total budget is within the voting cycle distribution limit
            if (votingCycleData.movedToVoteTokenCount + totalBudget > votingCycleData.votingCycleDistributionLimit) {
                revert ProposalValidator_ExceedsDistributionThreshold();
            }
        }

        // Move proposal to vote
        proposal.movedToVote = true;
        votingCycles[proposal.votingCycle].movedToVoteTokenCount += totalBudget;

        // Propose with module on the Governor
        uint256 proposalId =
            GOVERNOR.proposeWithModule(votingModule, proposalVotingModuleData, _description, idInConfigurator);

        // Make sure the proposalId matches
        if (proposalId != proposalId_) {
            revert ProposalValidator_ProposalIdMismatch();
        }

        emit ProposalMovedToVote(proposalId_, _msgSender());
    }

    /// @notice Sets the data of a voting cycle.
    /// @param _cycleNumber The number of the voting cycle to set.
    /// @param _startingTimestamp The starting timestamp of the voting cycle.
    /// @param _duration The duration of the voting cycle. Should be 1 day which is the end of Week 2 and start of Week
    /// 3
    /// of the voting cycle.
    /// @param _votingCycleDistributionLimit The max amount of tokens that can be distributed during the voting cycle.
    function setVotingCycleData(
        uint256 _cycleNumber,
        uint256 _startingTimestamp,
        uint256 _duration,
        uint256 _votingCycleDistributionLimit
    )
        external
        onlyOwner
    {
        _setVotingCycleData(_cycleNumber, _startingTimestamp, _duration, _votingCycleDistributionLimit);
    }

    /// @notice Sets the max amount of tokens that can be distributed in a proposal.
    /// @param _proposalDistributionThreshold The new proposal distribution threshold.
    function setProposalDistributionThreshold(uint256 _proposalDistributionThreshold) external onlyOwner {
        _setProposalDistributionThreshold(_proposalDistributionThreshold);
    }

    /// @notice Sets the data for a proposal type.
    /// @param _proposalType The type of proposal to set the data for.
    /// @param _proposalTypeData The data for the proposal type.
    function setProposalTypeData(
        ProposalType _proposalType,
        ProposalTypeData memory _proposalTypeData
    )
        external
        onlyOwner
    {
        _setProposalTypeData(_proposalType, _proposalTypeData);
    }

    /// @notice Validates the attestation data for a proposal.
    /// @dev Checks that the attester is the owner, the schema is correct,
    ///      the sender is the approved delegate, and that the proposal type is correct.
    ///      Reverts with ProposalValidator_InvalidAttestation if validation fails.
    /// @param _attestationUid The UID of the attestation to validate.
    /// @param _expectedProposalType The expected proposal type from the attestation.
    function _validateApprovedProposerAttestation(
        bytes32 _attestationUid,
        ProposalType _expectedProposalType
    )
        internal
        view
    {
        Attestation memory attestation = IEAS(Predeploys.EAS).getAttestation(_attestationUid);

        // Check if attestation exists, equivalent to calling EAS.isAttestationValid(_attestationUid)
        if (attestation.uid == bytes32(0)) {
            revert ProposalValidator_InvalidAttestation();
        }

        // check if the attestation is revoked
        if (attestation.revocationTime != 0) {
            revert ProposalValidator_AttestationRevoked();
        }

        // check if the attestation is expired
        if (attestation.expirationTime != 0 && attestation.expirationTime < block.timestamp) {
            revert ProposalValidator_AttestationExpired();
        }

        (uint8 proposalType,) = abi.decode(attestation.data, (uint8, string));

        if (
            attestation.attester != owner() || attestation.schema != APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID
                || attestation.recipient != _msgSender() || proposalType != uint8(_expectedProposalType)
        ) {
            revert ProposalValidator_InvalidAttestation();
        }
    }

    /// @notice Validates the attestation data for a delegate that tries to approve a proposal.
    /// @dev Only accepts attestations that do NOT include partial delegation.
    /// @param _attestationUid The UID of the attestation to validate.
    /// @param _lastVotingCycle The last voting cycle to validate against.
    function _validateTopDelegateAttestation(bytes32 _attestationUid, uint256 _lastVotingCycle) internal view {
        Attestation memory attestation = IEAS(Predeploys.EAS).getAttestation(_attestationUid);
        VotingCycleData memory previousVotingCycleData = votingCycles[_lastVotingCycle];
        if (previousVotingCycleData.startingTimestamp == 0) {
            revert ProposalValidator_InvalidVotingCycle();
        }

        // Check if attestation exists, equivalent to calling EAS.isAttestationValid(_attestationUid)
        if (attestation.uid == bytes32(0)) {
            revert ProposalValidator_InvalidAttestation();
        }

        // check if the schema is correct
        if (attestation.schema != TOP_DELEGATES_ATTESTATION_SCHEMA_UID) {
            revert ProposalValidator_InvalidAttestationSchema();
        }

        // check if the attestation is revoked
        if (attestation.revocationTime != 0) {
            revert ProposalValidator_AttestationRevoked();
        }

        // since the attestations are updated daily we should only allow attestations
        // created before the last voting cycle of the proposal
        if (attestation.time > previousVotingCycleData.startingTimestamp) {
            revert ProposalValidator_AttestationCreatedAfterLastVotingCycle();
        }

        (, bool _includePartialDelegation,) = abi.decode(attestation.data, (string, bool, string));

        // check if the attestation includes partial delegation or the recipient is not the caller
        if (_includePartialDelegation || attestation.recipient != _msgSender()) {
            revert ProposalValidator_InvalidAttestation();
        }
    }

    /// @notice Internal function to build proposal options with optional execution data.
    /// @param _optionDescriptions The strings of the different options that can be voted.
    /// @param _recipients An address for each option to transfer funds to (empty for non-funding proposals).
    /// @param _amounts The amount to transfer for each option (empty for non-funding proposals).
    /// @return options_ The built proposal options.
    /// @return totalBudget_ The total budget amount (sum of all amounts, 0 for non-funding proposals).
    function _buildApprovalModuleOptions(
        string[] memory _optionDescriptions,
        address[] memory _recipients,
        uint256[] memory _amounts
    )
        internal
        view
        returns (IApprovalVotingModule.ProposalOption[] memory options_, uint256 totalBudget_)
    {
        uint256 optionsLength = _optionDescriptions.length;
        options_ = new IApprovalVotingModule.ProposalOption[](optionsLength);

        for (uint256 i = 0; i < optionsLength; i++) {
            address[] memory targets;
            uint256[] memory values;
            bytes[] memory calldatas;
            uint256 budgetTokensSpent;

            // Check if this is a funding proposal (has recipients and amounts)
            if (_recipients.length > 0 && _amounts.length > 0) {
                // Validate amount doesn't exceed distribution threshold
                if (_amounts[i] > proposalDistributionThreshold) {
                    revert ProposalValidator_ExceedsDistributionThreshold();
                }
                targets = new address[](1);
                values = new uint256[](1);
                calldatas = new bytes[](1);

                targets[0] = Predeploys.GOVERNANCE_TOKEN;
                calldatas[0] = abi.encodeCall(IERC20.transfer, (_recipients[i], _amounts[i]));
                budgetTokensSpent = _amounts[i];
                totalBudget_ += _amounts[i];
            } else {
                // Non-funding proposals have no execution data
                targets = new address[](0);
                values = new uint256[](0);
                calldatas = new bytes[](0);
                budgetTokensSpent = 0;
            }

            options_[i] = IApprovalVotingModule.ProposalOption({
                budgetTokensSpent: budgetTokensSpent,
                targets: targets,
                values: values,
                calldatas: calldatas,
                description: _optionDescriptions[i]
            });
        }

        if (totalBudget_ > type(uint128).max) {
            revert ProposalValidator_InvalidTotalBudget();
        }
    }

    /// @notice Calculate `proposalId` based on `module`, `proposalData` and `descriptionHash`.
    /// @param _module The address of the voting module to use for this proposal.
    /// @param _proposalData The proposal data to pass to the voting module.
    /// @param _descriptionHash The hash of the proposal description.
    /// @return The proposal ID as uint256.
    function _hashProposalWithModule(
        address _module,
        bytes memory _proposalData,
        bytes32 _descriptionHash
    )
        internal
        view
        returns (uint256)
    {
        return uint256(keccak256(abi.encode(address(GOVERNOR), _module, _proposalData, _descriptionHash)));
    }

    /// @notice Private function to set the voting cycle data and emit event.
    /// @param _cycleNumber The number of the voting cycle to set.
    /// @param _startingTimestamp The starting timestamp of the voting cycle.
    /// @param _duration The duration of the voting cycle. Should be 1 day which is the end of Week 2 and start of Week
    /// 3
    /// of the voting cycle.
    /// @param _votingCycleDistributionLimit The max amount of tokens that can be distributed during the voting cycle.
    function _setVotingCycleData(
        uint256 _cycleNumber,
        uint256 _startingTimestamp,
        uint256 _duration,
        uint256 _votingCycleDistributionLimit
    )
        private
    {
        if (votingCycles[_cycleNumber].startingTimestamp != 0) {
            revert ProposalValidator_VotingCycleAlreadySet();
        }

        votingCycles[_cycleNumber] = VotingCycleData({
            startingTimestamp: _startingTimestamp,
            duration: _duration,
            votingCycleDistributionLimit: _votingCycleDistributionLimit,
            movedToVoteTokenCount: 0
        });
        emit VotingCycleDataSet(_cycleNumber, _startingTimestamp, _duration, _votingCycleDistributionLimit);
    }

    /// @notice Private function to set the proposal distribution threshold and emit event.
    /// @param _proposalDistributionThreshold The new proposal distribution threshold.
    function _setProposalDistributionThreshold(uint256 _proposalDistributionThreshold) private {
        proposalDistributionThreshold = _proposalDistributionThreshold;
        emit ProposalDistributionThresholdSet(_proposalDistributionThreshold);
    }

    /// @notice Private function to set a proposal's type data.
    /// @param _proposalType The type of proposal to set the data for.
    /// @param _proposalTypeData The data for the proposal type.
    function _setProposalTypeData(ProposalType _proposalType, ProposalTypeData memory _proposalTypeData) private {
        proposalTypesData[_proposalType] = _proposalTypeData;
        emit ProposalTypeDataSet(_proposalType, _proposalTypeData.requiredApprovals, _proposalTypeData.idInConfigurator);
    }
}
