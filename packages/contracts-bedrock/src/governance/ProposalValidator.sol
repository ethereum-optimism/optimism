// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IOptimismGovernor } from "interfaces/governance/IOptimismGovernor.sol";
import { IGovernanceToken } from "interfaces/governance/IGovernanceToken.sol";
import { IEAS, Attestation } from "src/vendor/eas/IEAS.sol";

/// @title ProposalValidator
/// @notice The ProposalValidator contract is responsible for validating proposals and moving
///         them to the vote phase on the Optimism Governor.
contract ProposalValidator is Ownable {
    /*//////////////////////////////////////////////////////////////
                                 ERRORS
    //////////////////////////////////////////////////////////////*/

    /// @notice Thrown when a proposal doesn't have enough delegate approvals to move to vote.
    error ProposalValidator_InsufficientApprovals();

    /// @notice Thrown when a delegate attempts to approve a proposal they've already approved.
    error ProposalValidator_ProposalAlreadyApproved();

    /// @notice Thrown when attempting to move a proposal to vote that is already in voting.
    error ProposalValidator_ProposalAlreadySubmitted();

    /// @notice Thrown when a delegate has insufficient voting power to approve a proposal.
    error ProposalValidator_InsufficientVotingPower();

    /// @notice Thrown when an invalid attestation is provided for a proposal.
    error ProposalValidator_InvalidAttestation();

    /// @notice Thrown when a proposal does not exist.
    error ProposalValidator_ProposalDoesNotExist();

    /*//////////////////////////////////////////////////////////////
                                 STRUCTS
    //////////////////////////////////////////////////////////////*/

    /// @notice Data structure for storing proposal information.
    /// @param proposer The address that submitted the proposal.
    /// @param proposalType Type of the proposal from the ProposalType enum.
    /// @param proposalTypeConfigurator Configuration value specific to the proposal type.
    /// @param inVoting Whether the proposal has been moved to the voting phase.
    /// @param delegateApprovals Mapping of delegate addresses to their approval status.
    /// @param remainingApprovalsRequired Number of approvals still needed before voting.
    struct ProposalData {
        address proposer;
        ProposalType proposalType;
        uint8 proposalTypeConfigurator;
        bool inVoting;
        mapping(address => bool) delegateApprovals;
        uint256 remainingApprovalsRequired;
    }

    /// @notice Data structure for storing immutable proposal type data.
    /// @param targets Target addresses for proposal calls.
    /// @param values ETH values for proposal calls.
    /// @param signatures Function signatures for proposal calls.
    struct ImmutableProposalTypeData {
        address[] targets;
        uint256[] values;
        string[] signatures;
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

    /// @notice Emitted when a new proposal is submitted to the validator contract.
    /// @param proposalHash The hash of the submitted proposal.
    /// @param proposer The address that submitted the proposal.
    /// @param targets Target addresses for proposal calls.
    /// @param values ETH values for proposal calls.
    /// @param calldatas Function data for proposal calls.
    /// @param description Description of the proposal.
    /// @param proposalType Type of the proposal.
    /// @param proposalTypeConfigurator Configuration value specific to the proposal type.
    event ProposalSubmitted(
        bytes32 indexed proposalHash,
        address indexed proposer,
        address[] targets,
        uint256[] values,
        bytes[] calldatas,
        string description,
        ProposalType proposalType,
        uint8 proposalTypeConfigurator
    );

    /// @notice Emitted when a delegate approves a proposal.
    /// @param proposalHash The hash of the approved proposal.
    /// @param approver The address of the delegate who approved the proposal.
    event ProposalApproved(bytes32 indexed proposalHash, address indexed approver);

    /// @notice Emitted when a proposal is moved to the voting phase in the governor contract.
    /// @param proposalHash The hash of the proposal moved to vote.
    /// @param executor The address that executed the move to vote.
    event ProposalMovedToVote(bytes32 indexed proposalHash, address indexed executor);

    /// @notice Emitted when the minimum voting power is set.
    /// @param newMinimumVotingPower The new minimum voting power.
    event MinimumVotingPowerSet(uint256 newMinimumVotingPower);

    /// @notice Emitted when the voting cycle block is set.
    /// @param newVotingCycleBlock The new voting cycle block.
    event VotingCycleBlockSet(uint256 newVotingCycleBlock);

    /// @notice Emitted when the distribution threshold is set.
    /// @param newDistributionThreshold The new distribution threshold.
    event DistributionThresholdSet(uint256 newDistributionThreshold);

    /// @notice Emitted when the number of approvals required for a proposal type is set.
    /// @param proposalType The type of proposal.
    /// @param newApprovalThreshold The new approval threshold.
    event ProposalApprovalThresholdSet(ProposalType proposalType, uint256 newApprovalThreshold);

    /// @notice The schema UID for attestations in the Ethereum Attestation Service.
    /// @dev Schema format: { approvedProposer: address, proposalType: uint8 }
    bytes32 public immutable ATTESTATION_SCHEMA_UID;

    /// @notice The Optimism Governor contract that will handle the voting phase.
    IOptimismGovernor public immutable GOVERNOR;

    /// @notice The token used to determine voting power.
    IGovernanceToken public immutable VOTING_TOKEN;

    /// @notice The minimum voting power required for a delegate to approve proposals.
    uint256 public minimumVotingPower;

    /// @notice The block number of the current voting cycle.
    uint256 public votingCycleBlock;

    /// @notice The max amount of tokens that can be distributed in a proposal.
    uint256 public distributionThreshold;

    /// @notice The number of approvals required for each proposal type.
    mapping(ProposalType => uint256) private _proposalRequiredApprovals;

    /// @notice The immutable data for each proposal type.
    mapping(ProposalType => ImmutableProposalTypeData) private _proposalTypeData;

    /// @notice Mapping of proposal hash to their corresponding proposal data.
    mapping(bytes32 => ProposalData) private _proposals;

    /// @notice Initializes the ProposalValidator contract.
    /// @param _owner The address that will own the contract.
    /// @param _governor The Optimism Governor contract address.
    /// @param _votingToken The token used to determine voting power.
    /// @param _attestationSchemaUid The schema UID for attestations in EAS.
    /// @param _minimumVotingPower The minimum voting power required for a delegate to approve proposals.
    /// @param _votingCycleBlock The block number of the current voting cycle.
    /// @param _distributionThreshold The max amount of tokens that can be distributed in a proposal.
    /// @param _proposalTypes Array of proposal types to set approval thresholds for.
    /// @param _requiredApprovals Array of approval thresholds corresponding to the proposal types.
    /// @param _immutableProposalTypeDatas Array of immutable proposal type data corresponding to the proposal types.
    constructor(
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
    ) {
        transferOwnership(_owner);
        GOVERNOR = _governor;
        VOTING_TOKEN = _votingToken;
        ATTESTATION_SCHEMA_UID = _attestationSchemaUid;

        _setMinimumVotingPower(_minimumVotingPower);
        _setVotingCycleBlock(_votingCycleBlock);
        _setDistributionThreshold(_distributionThreshold);

        for (uint256 i = 0; i < _proposalTypes.length; i++) {
            _setProposalRequiredApprovals(_proposalTypes[i], _requiredApprovals[i]);
            _proposalTypeData[_proposalTypes[i]] = _immutableProposalTypeDatas[i];
        }
    }

    /// @notice Submit a proposal for delegate approval
    /// @param _targets Target addresses for proposal calls
    /// @param _values ETH values for proposal calls
    /// @param _calldatas Function data for proposal calls
    /// @param _description Description of the proposal
    /// @param _proposalType Type of the proposal
    /// @return proposalHash_ The hash of the submitted proposal
    function submitProposal(
        address[] memory _targets,
        uint256[] memory _values,
        bytes[] memory _calldatas,
        string memory _description,
        ProposalType _proposalType,
        uint8 _proposalTypeConfigurator,
        bytes32 _attestationUid
    )
        external
        returns (bytes32 proposalHash_)
    {
        _validateProposal(_targets, _values, _calldatas, _proposalType, _attestationUid);

        proposalHash_ = _hashProposal(_targets, _values, _calldatas, _description);
        ProposalData storage proposal = _proposals[proposalHash_];

        if (proposal.proposer != address(0)) {
            revert ProposalValidator_ProposalAlreadySubmitted();
        }

        proposal.proposer = msg.sender;
        proposal.proposalType = _proposalType;
        proposal.proposalTypeConfigurator = _proposalTypeConfigurator;
        proposal.inVoting = false;
        proposal.remainingApprovalsRequired = 4; // Hardcoded for now, will change with proposalTypes

        emit ProposalSubmitted(
            proposalHash_,
            msg.sender,
            _targets,
            _values,
            _calldatas,
            _description,
            _proposalType,
            _proposalTypeConfigurator
        );
    }

    /// @notice Approve a proposal (only callable by delegates with sufficient voting power)
    /// @param _proposalHash The hash of the proposal to approve
    function approveProposal(bytes32 _proposalHash) external {
        if (!canSignOff(msg.sender)) {
            revert ProposalValidator_InsufficientVotingPower();
        }

        ProposalData storage proposal = _proposals[_proposalHash];

        if (proposal.delegateApprovals[msg.sender]) {
            revert ProposalValidator_ProposalAlreadyApproved();
        }

        proposal.delegateApprovals[msg.sender] = true;
        proposal.remainingApprovalsRequired--; // Expected overflow when all approvals are granted

        emit ProposalApproved(_proposalHash, msg.sender);
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
        bytes32 _proposalHash = _hashProposal(_targets, _values, _calldatas, _description);

        ProposalData storage proposal = _proposals[_proposalHash];

        if (proposal.proposer == address(0)) {
            revert ProposalValidator_ProposalDoesNotExist();
        }

        if (proposal.remainingApprovalsRequired > 0) {
            revert ProposalValidator_InsufficientApprovals();
        }

        if (proposal.inVoting) {
            revert ProposalValidator_ProposalAlreadySubmitted();
        }

        proposal.inVoting = true;

        governorProposalId_ =
            GOVERNOR.propose(_targets, _values, _calldatas, _description, proposal.proposalTypeConfigurator);

        emit ProposalMovedToVote(_proposalHash, msg.sender);
    }

    /// @notice Returns whether a delegate has enough voting power to approve a proposal.
    /// @param _delegate The address of the delegate to check.
    /// @return canSignOff_ True if the delegate has sufficient voting power, false otherwise.
    function canSignOff(address _delegate) public view returns (bool canSignOff_) {
        canSignOff_ = VOTING_TOKEN.balanceOf(_delegate) >= minimumVotingPower;
    }

    /// @notice Sets the minimum voting power required for a delegate to approve proposals.
    /// @param _minimumVotingPower The new minimum voting power threshold.
    function setMinimumVotingPower(uint256 _minimumVotingPower) external onlyOwner {
        _setMinimumVotingPower(_minimumVotingPower);
    }

    /// @notice Sets the block number of the current voting cycle.
    /// @param _votingCycleBlock The new voting cycle block number.
    function setVotingCycleBlock(uint256 _votingCycleBlock) external onlyOwner {
        _setVotingCycleBlock(_votingCycleBlock);
    }

    /// @notice Sets the max amount of tokens that can be distributed in a proposal.
    /// @param _distributionThreshold The new distribution threshold.
    function setDistributionThreshold(uint256 _distributionThreshold) external onlyOwner {
        _setDistributionThreshold(_distributionThreshold);
    }

    /// @notice Sets the number of approvals required for each proposal type.
    /// @param _proposalType The type of proposal to set the required approvals for.
    /// @param _requiredApprovals The new required approvals.
    function setProposalRequiredApprovals(ProposalType _proposalType, uint256 _requiredApprovals) external onlyOwner {
        _setProposalRequiredApprovals(_proposalType, _requiredApprovals);
    }

    /// @notice Validates a proposal before submission.
    /// @dev Checks if the proposal requires approval and validates the attestation.
    /// @param _targets Target addresses for proposal calls.
    /// @param _values ETH values for proposal calls.
    /// @param _calldatas Function data for proposal calls.
    /// @param _proposalType Type of the proposal.
    /// @param _attestationUid The UID of the attestation proving eligibility.
    function _validateProposal(
        address[] memory _targets,
        uint256[] memory _values,
        bytes[] memory _calldatas,
        ProposalType _proposalType,
        bytes32 _attestationUid
    )
        private
        view
    {
        if (_requiresAttestation(_proposalType)) {
            Attestation memory attestation = IEAS(Predeploys.EAS).getAttestation(_attestationUid);
            if (
                attestation.attester != owner() || attestation.schema != ATTESTATION_SCHEMA_UID
                    || !_isValidAttestationData(attestation.data, _proposalType)
            ) {
                revert ProposalValidator_InvalidAttestation();
            }
        }
    }

    /// @notice Determines if a proposal type requires approval via attestation.
    /// @param _proposalType The type of proposal to check.
    /// @return requiresAttestation_ True if the proposal type requires approval, false otherwise.
    function _requiresAttestation(ProposalType _proposalType) private pure returns (bool requiresAttestation_) {
        return _proposalType == ProposalType.ProtocolOrGovernorUpgrade
            || _proposalType == ProposalType.MaintenanceUpgrade || _proposalType == ProposalType.CouncilMemberElections;
    }

    /// @notice Validates the attestation data for a proposal.
    /// @dev Checks that the sender is the approved delegate and that the proposal type is correct.
    /// @param _data The attestation data to validate.
    /// @param _expectedProposalType The expected proposal type from the attestation.
    /// @return isValid_ True if the attestation data is valid, false otherwise.
    function _isValidAttestationData(
        bytes memory _data,
        ProposalType _expectedProposalType
    )
        private
        view
        returns (bool isValid_)
    {
        (address approvedDelegate, uint8 proposalType) = abi.decode(_data, (address, uint8));
        isValid_ = approvedDelegate == msg.sender && proposalType == uint8(_expectedProposalType);
    }

    function _hashProposal(
        address[] memory _targets,
        uint256[] memory _values,
        bytes[] memory _calldatas,
        string memory _description
    )
        internal
        pure
        returns (bytes32 proposalHash_)
    {
        return keccak256(abi.encode(_targets, _values, _calldatas, _description));
    }

    /// @notice Private function to set the minimum voting power and emit event.
    /// @param _minimumVotingPower The new minimum voting power threshold.
    function _setMinimumVotingPower(uint256 _minimumVotingPower) private {
        minimumVotingPower = _minimumVotingPower;
        emit MinimumVotingPowerSet(_minimumVotingPower);
    }

    /// @notice Private function to set the voting cycle block and emit event.
    /// @param _votingCycleBlock The new voting cycle block number.
    function _setVotingCycleBlock(uint256 _votingCycleBlock) private {
        votingCycleBlock = _votingCycleBlock;
        emit VotingCycleBlockSet(_votingCycleBlock);
    }

    /// @notice Private function to set the distribution threshold and emit event.
    /// @param _distributionThreshold The new distribution threshold.
    function _setDistributionThreshold(uint256 _distributionThreshold) private {
        distributionThreshold = _distributionThreshold;
        emit DistributionThresholdSet(_distributionThreshold);
    }

    /// @notice Private function to set a proposal's type required approvals and emit event.
    /// @param _proposalType The type of proposal to set the required approvals for.
    /// @param _requiredApprovals The new required approvals.
    function _setProposalRequiredApprovals(ProposalType _proposalType, uint256 _requiredApprovals) private {
        _proposalRequiredApprovals[_proposalType] = _requiredApprovals;
        emit ProposalApprovalThresholdSet(_proposalType, _requiredApprovals);
    }
}
