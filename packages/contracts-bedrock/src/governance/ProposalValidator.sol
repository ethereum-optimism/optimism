// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IOptimismGovernor } from "interfaces/governance/IOptimismGovernor.sol";
import { IGovernanceToken } from "interfaces/governance/IGovernanceToken.sol";
import { IProposalTypesConfigurator } from "interfaces/governance/IProposalTypesConfigurator.sol";
import { IEAS, Attestation } from "src/vendor/eas/IEAS.sol";
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

// Modules
import { ProposalSettings, ProposalOption, PassingCriteria } from "src/governance/ApprovalVotingModule.sol";

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

    /// @notice Thrown when an attestation schema is invalid.
    error ProposalValidator_InvalidAttestationSchema();

    /// @notice Thrown when the criteria value is invalid for council elections (must not exceed options length).
    error ProposalValidator_InvalidCriteriaValue();

    /*//////////////////////////////////////////////////////////////
                                 STRUCTS
    //////////////////////////////////////////////////////////////*/

    /// @notice Struct for storing proposal information.
    /// @param proposer The address that submitted the proposal.
    /// @param proposalType Type of the proposal from the ProposalType enum.
    /// @param inVoting Whether the proposal has been moved to the voting phase.
    /// @param delegateApprovals Mapping of delegate addresses to their approval status.
    /// @param approvalCount Number of approvals received so far.
    struct ProposalData {
        address proposer;
        ProposalType proposalType;
        bool inVoting;
        mapping(address => bool) delegateApprovals;
        uint256 approvalCount;
    }

    /// @notice Struct for storing explicit data for each proposal type.
    /// @param requiredApprovals The number of approvals each proposal type requires in order to be able to move for
    /// voting.
    /// @param proposalVotingModule The proposal type ID used to get the voting module from the configurator.
    struct ProposalTypeData {
        uint256 requiredApprovals;
        uint8 proposalVotingModule;
    }

    /// @notice Struct for storing voting cycle data.
    /// @param startingBlock The block number of the starting block of the voting cycle.
    /// @param duration The duration of the voting cycle.
    /// @param votingCycleDistributionLimit The max amount of tokens that can be distributed in a proposal.
    struct VotingCycleData {
        uint256 startingBlock;
        uint256 duration;
        uint256 votingCycleDistributionLimit;
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
                                 EVENTS
    //////////////////////////////////////////////////////////////*/

    /// @notice Emitted when a new proposal is submitted.
    /// @param proposalHash The hash of the submitted proposal.
    /// @param proposer The address that submitted the proposal.
    /// @param description Description of the proposal.
    /// @param proposalType Type of the proposal.
    event ProposalSubmitted(
        bytes32 indexed proposalHash, address indexed proposer, string description, ProposalType proposalType
    );

    /// @notice Emitted when a delegate approves a proposal.
    /// @param proposalHash The hash of the approved proposal.
    /// @param approver The address of the delegate who approved the proposal.
    event ProposalApproved(bytes32 indexed proposalHash, address indexed approver);

    /// @notice Emitted when a proposal is moved to the voting phase in the governor contract.
    /// @param proposalHash The hash of the proposal moved to vote.
    /// @param executor The address that executed the move to vote.
    event ProposalMovedToVote(bytes32 indexed proposalHash, address indexed executor);

    /// @notice Emitted when the voting cycle data is set.
    /// @param cycleNumber The number of the voting cycle.
    /// @param startBlock The block number of the starting block of the voting cycle.
    /// @param duration The duration of the voting cycle.
    /// @param votingCycleDistributionLimit The max amount of tokens that can be distributed during the voting cycle.
    event VotingCycleDataSet(
        uint256 cycleNumber, uint256 startBlock, uint256 duration, uint256 votingCycleDistributionLimit
    );

    /// @notice Emitted when the distribution threshold is set.
    /// @param newDistributionThreshold The new distribution threshold.
    event DistributionThresholdSet(uint256 newDistributionThreshold);

    /// @notice Emitted when the proposal type data is set.
    /// @param proposalType The type of proposal.
    /// @param requiredApprovals The required number of approvals.
    /// @param proposalVotingModule The proposal type ID.
    event ProposalTypeDataSet(ProposalType proposalType, uint256 requiredApprovals, uint8 proposalVotingModule);

    /// @notice Emitted with ProposalSubmitted event.
    /// @param proposalHash The hash of the submitted proposal.
    /// @param encodedVotingModuleData The encoded voting module data.
    event ProposalVotingModuleData(bytes32 indexed proposalHash, bytes encodedVotingModuleData);

    /// @notice The schema UID for attestations in the Ethereum Attestation Service for checking if the caller
    ///         is an approved proposer.
    /// @dev Schema format: { approvedProposer: address, proposalType: uint8 }
    bytes32 public immutable APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID;

    /// @notice The schema UID for attestations in the Ethereum Attestation Service for checking if the caller
    ///         is part of the top100 delegates.
    bytes32 public immutable TOP_DELEGATES_ATTESTATION_SCHEMA_UID;

    /// @notice The Optimism Governor contract that will handle the voting phase.
    IOptimismGovernor public immutable GOVERNOR;

    /// @notice The governance token contract.
    IGovernanceToken public immutable VOTING_TOKEN;

    /// @notice The proposal types configurator contract.
    IProposalTypesConfigurator public proposalTypesConfigurator;

    /// @notice The max amount of tokens that can be distributed in a proposal.
    uint256 public distributionThreshold;

    /// @notice Mapping of voting cycle numbers to their corresponding data.
    mapping(uint256 => VotingCycleData) public votingCycles;

    /// @notice Mapping of proposal types to their corresponding data.
    mapping(ProposalType => ProposalTypeData) public proposalTypesData;

    /// @notice Mapping of proposal hash to their corresponding proposal data.
    mapping(bytes32 => ProposalData) internal _proposals;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.1
    function version() public pure virtual returns (string memory) {
        return "1.0.0-beta.1";
    }

    /// @notice Constructs the ProposalValidator contract.
    /// @param _approvedProposerAttestationSchemaUid The schema UID for attestations in EAS for submitting proposals.
    /// @param _topDelegatesAttestationSchemaUid The schema UID for attestations in EAS for checking if the caller
    ///        is part of the top100 delegates.
    /// @param _governor The Optimism Governor contract address.
    /// @param _votingToken The token used to determine voting power.
    constructor(
        bytes32 _approvedProposerAttestationSchemaUid,
        bytes32 _topDelegatesAttestationSchemaUid,
        IOptimismGovernor _governor,
        IGovernanceToken _votingToken
    )
        ReinitializableBase(1)
    {
        APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID = _approvedProposerAttestationSchemaUid;
        TOP_DELEGATES_ATTESTATION_SCHEMA_UID = _topDelegatesAttestationSchemaUid;
        GOVERNOR = _governor;
        VOTING_TOKEN = _votingToken;
        _disableInitializers();
    }

    /// @notice Initializes the ProposalValidator contract.
    /// @param _owner The address that will own the contract.
    /// @param _proposalTypesConfigurator The proposal types configurator contract address.
    /// @param _cycleNumber The number of the current voting cycle.
    /// @param _startBlock The block number of the starting block of the voting cycle.
    /// @param _duration The duration of the voting cycle.
    /// @param _votingCycleDistributionLimit The max amount of tokens that can be distributed during the voting cycle.
    /// @param _distributionThreshold The max amount of tokens that can be distributed in a proposal.
    /// @param _proposalTypes Array of proposal types to set data for.
    /// @param _proposalTypesData Array of proposal type data corresponding to the proposal types.
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
    )
        external
        reinitializer(initVersion())
    {
        if (_proposalTypes.length != _proposalTypesData.length) {
            revert ProposalValidator_ProposalTypesDataLengthMismatch();
        }

        proposalTypesConfigurator = _proposalTypesConfigurator;
        _setVotingCycleData(_cycleNumber, _startBlock, _duration, _votingCycleDistributionLimit);
        _setDistributionThreshold(_distributionThreshold);

        for (uint256 i = 0; i < _proposalTypes.length; i++) {
            _setProposalTypeData(_proposalTypes[i], _proposalTypesData[i]);
        }

        __Ownable_init();
        transferOwnership(_owner);
    }

    /// @notice Submits a Council Member Elections proposal for approval and voting.
    /// @param _criteriaValue Since the passing criteria type is "TopChoices" this number represents the amount
    /// of top choices that can pass the voting.
    /// @param _optionDescriptions The strings of the different options that can be voted.
    /// @param _proposalDescription Description of the proposal.
    /// @param _attestationUid The UID of the attestation for the approved proposer.
    /// @return proposalHash_ The hash of the submitted proposal.
    function submitCouncilMemberElectionsProposal(
        uint128 _criteriaValue,
        string[] memory _optionDescriptions,
        string memory _proposalDescription,
        bytes32 _attestationUid
    )
        external
        returns (bytes32 proposalHash_)
    {
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

        ProposalOption[] memory options = new ProposalOption[](optionsLength);

        // Build proposal options without any execution calls (elections don't execute operations)
        for (uint256 i = 0; i < optionsLength; i++) {
            address[] memory targets = new address[](0);
            uint256[] memory values = new uint256[](0);
            bytes[] memory calldatas = new bytes[](0);

            options[i] = ProposalOption({
                budgetTokensSpent: 0, // No tokens spent for elections
                targets: targets,
                values: values,
                calldatas: calldatas,
                description: _optionDescriptions[i]
            });
        }

        // Configure approval voting settings with TopChoices criteria
        ProposalSettings memory settings = ProposalSettings({
            maxApprovals: uint8(optionsLength),
            criteria: uint8(PassingCriteria.TopChoices),
            budgetToken: address(0), // No budget token for elections
            criteriaValue: _criteriaValue,
            budgetAmount: 0 // No budget amount for elections
         });

        bytes memory proposalVotingModuleData = abi.encode(options, settings);

        // Get the module address from the configurator
        address votingModule = proposalTypesConfigurator.proposalTypes(
            proposalTypesData[ProposalType.CouncilMemberElections].proposalVotingModule
        ).module;

        // Generate unique proposal hash
        proposalHash_ =
            _hashProposalWithModule(votingModule, proposalVotingModuleData, keccak256(bytes(_proposalDescription)));

        ProposalData storage proposal = _proposals[proposalHash_];

        // Prevent duplicate proposals with same hash
        if (proposal.proposer != address(0)) {
            revert ProposalValidator_ProposalAlreadySubmitted();
        }

        // Check if proposal already exists in OptimismGovernor
        if (GOVERNOR.proposalSnapshot(uint256(proposalHash_)) != 0) {
            revert ProposalValidator_ProposalAlreadySubmitted();
        }

        // Store proposal metadata
        proposal.proposer = msg.sender;
        proposal.proposalType = ProposalType.CouncilMemberElections;

        emit ProposalSubmitted(proposalHash_, msg.sender, _proposalDescription, ProposalType.CouncilMemberElections);
        emit ProposalVotingModuleData(proposalHash_, proposalVotingModuleData);
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
    /// @return proposalHash_ The hash of the submitted proposal.
    function submitFundingProposal(
        uint128 _criteriaValue,
        string[] memory _optionsDescriptions,
        address[] memory _optionsRecipients,
        uint256[] memory _optionsAmounts,
        string memory _description,
        ProposalType _proposalType
    )
        external
        returns (bytes32 proposalHash_)
    {
        // Only funding proposal types can use this function
        if (_proposalType != ProposalType.GovernanceFund && _proposalType != ProposalType.CouncilBudget) {
            revert ProposalValidator_InvalidFundingProposalType();
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

        ProposalOption[] memory options = new ProposalOption[](optionsLength);
        uint256 totalBudget = 0;

        // Check amounts, build options, and calculate total budget in single loop
        for (uint256 i = 0; i < optionsLength; i++) {
            if (_optionsAmounts[i] > distributionThreshold) {
                revert ProposalValidator_ExceedsDistributionThreshold();
            }

            address[] memory targets = new address[](1);
            uint256[] memory values = new uint256[](1);
            bytes[] memory calldatas = new bytes[](1);

            targets[0] = Predeploys.GOVERNANCE_TOKEN;
            calldatas[0] = abi.encodeCall(IERC20.transfer, (_optionsRecipients[i], _optionsAmounts[i]));

            options[i] = ProposalOption({
                budgetTokensSpent: _optionsAmounts[i],
                targets: targets,
                values: values,
                calldatas: calldatas,
                description: _optionsDescriptions[i]
            });

            totalBudget += _optionsAmounts[i];
        }

        // Configure approval voting settings
        ProposalSettings memory settings = ProposalSettings({
            maxApprovals: uint8(optionsLength),
            criteria: uint8(PassingCriteria.Threshold),
            budgetToken: Predeploys.GOVERNANCE_TOKEN,
            criteriaValue: _criteriaValue,
            budgetAmount: uint128(totalBudget)
        });

        bytes memory proposalVotingModuleData = abi.encode(options, settings);

        // Get the module address from the configurator
        address votingModule =
            proposalTypesConfigurator.proposalTypes(proposalTypesData[_proposalType].proposalVotingModule).module;

        // Generate unique proposal hash
        proposalHash_ = _hashProposalWithModule(votingModule, proposalVotingModuleData, keccak256(bytes(_description)));

        ProposalData storage proposal = _proposals[proposalHash_];

        // Prevent duplicate proposals with same hash
        if (proposal.proposer != address(0)) {
            revert ProposalValidator_ProposalAlreadySubmitted();
        }

        // Check if proposal already exists in OptimismGovernor
        if (GOVERNOR.proposalSnapshot(uint256(proposalHash_)) != 0) {
            revert ProposalValidator_ProposalAlreadySubmitted();
        }

        // Store proposal metadata
        proposal.proposer = msg.sender;
        proposal.proposalType = _proposalType;

        emit ProposalSubmitted(proposalHash_, msg.sender, _description, _proposalType);
        emit ProposalVotingModuleData(proposalHash_, proposalVotingModuleData);
    }

    /// @notice Approves a proposal before being moved for voting.
    /// @dev This function should only be called by the top delegates.
    /// @param _proposalHash The hash of the proposal to approve
    /// @param _attestationUid The UID of the attestation for the delegate to approve the proposal
    function approveProposal(bytes32 _proposalHash, bytes32 _attestationUid) external {
        address _delegate = _msgSender();
        ProposalData storage proposal = _proposals[_proposalHash];
        // check if the proposal exists
        if (proposal.proposer == address(0)) {
            revert ProposalValidator_ProposalDoesNotExist();
        }

        // check if the caller has already approved the proposal
        if (proposal.delegateApprovals[_delegate]) {
            revert ProposalValidator_ProposalAlreadyApproved();
        }

        // validate the attestation
        _validateTopDelegateAttestation(_attestationUid, _msgSender());

        // store the approval
        proposal.delegateApprovals[_delegate] = true;
        proposal.approvalCount++;

        emit ProposalApproved(_proposalHash, _delegate);
    }

    /// @notice Move a proposal to voting phase after sufficient delegate approvals
    /// @param _targets Target addresses for proposal calls
    /// @param _values ETH values for proposal calls
    /// @param _calldatas Function data for proposal calls
    /// @param _description Description of the proposal
    /// @return governorProposalId_ The proposal ID in the governor contract
    function moveToVote(
        address[] memory _targets,
        uint256[] memory _values,
        bytes[] memory _calldatas,
        string memory _description
    )
        external
        returns (uint256 governorProposalId_)
    {
        // Verify that the provided data matches the proposalHash
        bytes32 _proposalHash = bytes32(0); // TODO: Implement hashProposalWithModule

        ProposalData storage proposal = _proposals[_proposalHash];

        if (proposal.proposer == address(0)) {
            revert ProposalValidator_ProposalDoesNotExist();
        }

        ProposalTypeData memory proposalTypeData = proposalTypesData[proposal.proposalType];
        if (proposal.approvalCount < proposalTypeData.requiredApprovals) {
            revert ProposalValidator_InsufficientApprovals();
        }

        if (proposal.inVoting) {
            revert ProposalValidator_ProposalAlreadySubmitted();
        }

        proposal.inVoting = true;

        governorProposalId_ =
            GOVERNOR.propose(_targets, _values, _calldatas, _description, uint8(proposal.proposalType));

        emit ProposalMovedToVote(_proposalHash, msg.sender);
    }

    /// @notice Checks if a delegate can approve a proposal.
    /// @dev Helper function for UI integration.
    /// @param _attestationUid The UID of the attestation to check.
    /// @return canApprove_ True if the delegate can approve the proposal, false otherwise.
    function canApproveProposal(bytes32 _attestationUid, address _delegate) external view returns (bool canApprove_) {
        canApprove_ = _validateTopDelegateAttestation(_attestationUid, _delegate);
    }

    /// @notice Sets the data of a voting cycle.
    /// @param _cycleNumber The number of the voting cycle to set.
    /// @param _startBlock The block number of the starting block of the voting cycle.
    /// @param _duration The duration of the voting cycle.
    /// @param _votingCycleDistributionLimit The max amount of tokens that can be distributed during the voting cycle.
    function setVotingCycleData(
        uint256 _cycleNumber,
        uint256 _startBlock,
        uint256 _duration,
        uint256 _votingCycleDistributionLimit
    )
        external
        onlyOwner
    {
        _setVotingCycleData(_cycleNumber, _startBlock, _duration, _votingCycleDistributionLimit);
    }

    /// @notice Sets the max amount of tokens that can be distributed in a proposal.
    /// @param _distributionThreshold The new distribution threshold.
    function setDistributionThreshold(uint256 _distributionThreshold) external onlyOwner {
        _setDistributionThreshold(_distributionThreshold);
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

        (address approvedDelegate, uint8 proposalType) = abi.decode(attestation.data, (address, uint8));

        if (
            attestation.attester != owner() || attestation.schema != APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID
                || approvedDelegate != msg.sender || proposalType != uint8(_expectedProposalType)
        ) {
            revert ProposalValidator_InvalidAttestation();
        }
    }

    /// @notice Validates the attestation data for a delegate that tries to approve a proposal.
    /// @dev Only acceptes attestations that does NOT include partial delegation.
    /// @param _attestationUid The UID of the attestation to validate.
    /// @param _delegate The delegate to validate the attestation for.
    /// @return canApprove_ True if the attestation is valid, false otherwise.
    function _validateTopDelegateAttestation(
        bytes32 _attestationUid,
        address _delegate
    )
        internal
        view
        returns (bool canApprove_)
    {
        Attestation memory attestation = IEAS(Predeploys.EAS).getAttestation(_attestationUid);
        (, bool _includePartialDelegation,) = abi.decode(attestation.data, (string, bool, string));

        // check if the schema is correct
        if (attestation.schema != TOP_DELEGATES_ATTESTATION_SCHEMA_UID) {
            revert ProposalValidator_InvalidAttestationSchema();
        }

        // check if the attestation is revoked
        if (attestation.revocationTime != 0) {
            revert ProposalValidator_AttestationRevoked();
        }

        // check if the attestation includes partial delegation or the recipient is not the caller
        if (_includePartialDelegation || attestation.recipient != _delegate) {
            revert ProposalValidator_InvalidAttestation();
        }

        canApprove_ = true;
    }

    /// @notice Calculate `proposalId` hashing similarly to `hashProposal` but based on `module` and `proposalData`.
    /// @param _module The address of the voting module to use for this proposal.
    /// @param _proposalData The proposal data to pass to the voting module.
    /// @param _descriptionHash The hash of the proposal description.
    /// @return The hash of the proposal.
    function _hashProposalWithModule(
        address _module,
        bytes memory _proposalData,
        bytes32 _descriptionHash
    )
        internal
        view
        returns (bytes32)
    {
        return keccak256(abi.encode(address(GOVERNOR), _module, _proposalData, _descriptionHash));
    }

    /// @notice Private function to set the voting cycle data and emit event.
    /// @param _cycleNumber The number of the voting cycle to set.
    /// @param _startBlock The block number of the starting block of the voting cycle.
    /// @param _duration The duration of the voting cycle.
    /// @param _votingCycleDistributionLimit The max amount of tokens that can be distributed during the voting cycle.
    function _setVotingCycleData(
        uint256 _cycleNumber,
        uint256 _startBlock,
        uint256 _duration,
        uint256 _votingCycleDistributionLimit
    )
        private
    {
        if (votingCycles[_cycleNumber].startingBlock != 0) {
            revert ProposalValidator_VotingCycleAlreadySet();
        }

        votingCycles[_cycleNumber] = VotingCycleData({
            startingBlock: _startBlock,
            duration: _duration,
            votingCycleDistributionLimit: _votingCycleDistributionLimit
        });
        emit VotingCycleDataSet(_cycleNumber, _startBlock, _duration, _votingCycleDistributionLimit);
    }

    /// @notice Private function to set the distribution threshold and emit event.
    /// @param _distributionThreshold The new distribution threshold.
    function _setDistributionThreshold(uint256 _distributionThreshold) private {
        distributionThreshold = _distributionThreshold;
        emit DistributionThresholdSet(_distributionThreshold);
    }

    /// @notice Private function to set a proposal's type data.
    /// @param _proposalType The type of proposal to set the data for.
    /// @param _proposalTypeData The data for the proposal type.
    function _setProposalTypeData(ProposalType _proposalType, ProposalTypeData memory _proposalTypeData) private {
        proposalTypesData[_proposalType] = _proposalTypeData;
        emit ProposalTypeDataSet(
            _proposalType, _proposalTypeData.requiredApprovals, _proposalTypeData.proposalVotingModule
        );
    }
}
