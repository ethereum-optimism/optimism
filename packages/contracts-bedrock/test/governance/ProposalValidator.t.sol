// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IProposalValidator } from "interfaces/governance/IProposalValidator.sol";
import { IOptimismGovernor } from "interfaces/governance/IOptimismGovernor.sol";
import { IProposalTypesConfigurator } from "interfaces/governance/IProposalTypesConfigurator.sol";
import {
    IEAS,
    AttestationRequest,
    AttestationRequestData,
    RevocationRequest,
    RevocationRequestData
} from "src/vendor/eas/IEAS.sol";
import { ISchemaRegistry, ISchemaResolver } from "src/vendor/eas/ISchemaRegistry.sol";
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { IApprovalVotingModule } from "interfaces/governance/IApprovalVotingModule.sol";
import { IOptimisticModule } from "interfaces/governance/IOptimisticModule.sol";

// Testing utilities
import { stdStorage, StdStorage } from "forge-std/Test.sol";

// Contracts
import { ProposalValidator } from "src/governance/ProposalValidator.sol";

// Mocks
import { ProposalValidatorForTest } from "test/mocks/ProposalValidatorForTest.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Testing utilities
import { stdStorage, StdStorage } from "forge-std/Test.sol";
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title ProposalValidator_TestInit
/// @notice Setup contract for ProposalValidator tests
contract ProposalValidator_TestInit is CommonTest {
    using stdStorage for StdStorage;

    // voting cycle constants
    uint256 public constant CYCLE_NUMBER = 1;
    uint256 public constant START_TIMESTAMP = 1000000;
    uint256 public constant DURATION = 1 days;
    uint256 public constant VOTING_CYCLE_DISTRIBUTION_LIMIT = 20000 ether;

    // proposal data constants
    uint256 public constant PROPOSAL_DISTRIBUTION_THRESHOLD = 10000 ether;
    uint256 public constant PROPOSAL_REQUIRED_APPROVALS = 1;
    uint256 public constant OPTIMISTIC_MODULE_PERCENT_DIVISOR = 10_000;
    uint8 public constant APPROVAL_VOTING_MODULE_ID = 1;
    uint8 public constant OPTIMISTIC_VOTING_MODULE_ID = 2;
    uint64 public constant ATT_EXPIRATION_TIME = 10 days;
    uint248 public constant AGAINST_THRESHOLD = 5000; // 50%
    string public constant PROPOSAL_DESCRIPTION = "Test proposal";

    // attestation constants
    bytes32 public APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID;
    bytes32 public TOP_DELEGATES_ATTESTATION_SCHEMA_UID;

    address owner;
    address user;
    address topDelegate_A = makeAddr("topDelegate_A");
    bytes32 topDelegateAttestation_A;
    address approvedProposer = makeAddr("approvedProposer");
    address approvalVotingModule;
    address optimisticVotingModule;

    ProposalValidatorForTest public validator;
    IOptimismGovernor public governor;
    IProposalTypesConfigurator public proposalTypesConfigurator;

    event ProposalSubmitted(
        uint256 indexed proposalId,
        address indexed proposer,
        string description,
        ProposalValidator.ProposalType proposalType
    );
    event ProposalApproved(uint256 indexed proposalId, address indexed approver);
    event ProposalMovedToVote(uint256 indexed proposalId, address indexed executor);
    event MinimumVotingPowerSet(uint256 newMinimumVotingPower);
    event VotingCycleDataSet(
        uint256 cycleNumber, uint256 startingTimestamp, uint256 duration, uint256 votingCycleDistributionLimit
    );
    event ProposalDistributionThresholdSet(uint256 newProposalDistributionThreshold);
    event ProposalTypeDataSet(
        ProposalValidator.ProposalType proposalType, uint256 requiredApprovals, uint8 idInConfigurator
    );
    event ProposalVotingModuleData(uint256 indexed proposalId, bytes encodedVotingModuleData);

    /// @notice Helper function to setup a mock and expect a call to it.
    function _mockAndExpect(address _receiver, bytes memory _calldata, bytes memory _returned) internal {
        vm.mockCall(_receiver, _calldata, _returned);
        vm.expectCall(_receiver, _calldata);
    }

    /// @notice Helper function to set proposal type data using StdStorage.
    function _setProposalTypeData(
        ProposalValidator.ProposalType _proposalType,
        ProposalValidator.ProposalTypeData memory _data
    )
        internal
    {
        // Set requiredApprovals (depth 0)
        stdstore.target(address(validator)).sig("proposalTypesData(uint8)").with_key(uint256(_proposalType)).depth(0)
            .checked_write(_data.requiredApprovals);

        // Set idInConfigurator (depth 1)
        stdstore.target(address(validator)).sig("proposalTypesData(uint8)").with_key(uint256(_proposalType)).depth(1)
            .checked_write(_data.idInConfigurator);
    }

    /// @notice Helper function to set CouncilMemberElections proposal type data.
    function _setCouncilMemberElectionsProposalType() internal {
        _setProposalTypeData(
            ProposalValidator.ProposalType.CouncilMemberElections,
            ProposalValidator.ProposalTypeData({
                requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
                idInConfigurator: APPROVAL_VOTING_MODULE_ID
            })
        );
    }

    /// @notice Helper function to set GovernanceFund proposal type data.
    function _setGovernanceFundProposalType() internal {
        _setProposalTypeData(
            ProposalValidator.ProposalType.GovernanceFund,
            ProposalValidator.ProposalTypeData({
                requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
                idInConfigurator: APPROVAL_VOTING_MODULE_ID
            })
        );
    }

    /// @notice Helper function to set CouncilBudget proposal type data.
    function _setCouncilBudgetProposalType() internal {
        _setProposalTypeData(
            ProposalValidator.ProposalType.CouncilBudget,
            ProposalValidator.ProposalTypeData({
                requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
                idInConfigurator: APPROVAL_VOTING_MODULE_ID
            })
        );
    }

    /// @notice Helper function to set ProtocolOrGovernorUpgrade proposal type data.
    function _setProtocolOrGovernorUpgradeProposalType() internal {
        _setProposalTypeData(
            ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade,
            ProposalValidator.ProposalTypeData({
                requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
                idInConfigurator: OPTIMISTIC_VOTING_MODULE_ID
            })
        );
    }

    /// @notice Helper function to set MaintenanceUpgrade proposal type data.
    function _setMaintenanceUpgradeProposalType() internal {
        _setProposalTypeData(
            ProposalValidator.ProposalType.MaintenanceUpgrade,
            ProposalValidator.ProposalTypeData({
                requiredApprovals: 0, // MaintenanceUpgrade moves directly to voting
                idInConfigurator: OPTIMISTIC_VOTING_MODULE_ID
            })
        );
    }

    /// @notice Helper to create minimal valid arrays for funding proposal error tests

    function _createMinimalFundingArrays(uint256 _length)
        internal
        returns (string[] memory descriptions_, address[] memory recipients_, uint256[] memory amounts_)
    {
        descriptions_ = new string[](_length);
        recipients_ = new address[](_length);
        amounts_ = new uint256[](_length);
        for (uint256 i = 0; i < _length; i++) {
            descriptions_[i] = string.concat("Option ", vm.toString(i + 1));
            recipients_[i] = makeAddr(string.concat("recipient", vm.toString(i + 1)));
            amounts_[i] = 100 ether * (i + 1);
        }
    }

    function _getProposalTypesAndData()
        internal
        pure
        returns (ProposalValidator.ProposalType[] memory, ProposalValidator.ProposalTypeData[] memory)
    {
        ProposalValidator.ProposalType[] memory proposalTypes = new ProposalValidator.ProposalType[](5);
        proposalTypes[0] = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        proposalTypes[1] = ProposalValidator.ProposalType.MaintenanceUpgrade;
        proposalTypes[2] = ProposalValidator.ProposalType.CouncilMemberElections;
        proposalTypes[3] = ProposalValidator.ProposalType.GovernanceFund;
        proposalTypes[4] = ProposalValidator.ProposalType.CouncilBudget;

        ProposalValidator.ProposalTypeData[] memory proposalTypesData = new ProposalValidator.ProposalTypeData[](5);
        // ProtocolOrGovernorUpgrade
        proposalTypesData[0] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            idInConfigurator: OPTIMISTIC_VOTING_MODULE_ID
        });
        // MaintenanceUpgrade
        proposalTypesData[1] =
            ProposalValidator.ProposalTypeData({ requiredApprovals: 0, idInConfigurator: OPTIMISTIC_VOTING_MODULE_ID });
        // CouncilMemberElections
        proposalTypesData[2] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            idInConfigurator: APPROVAL_VOTING_MODULE_ID
        });
        // GovernanceFund
        proposalTypesData[3] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            idInConfigurator: APPROVAL_VOTING_MODULE_ID
        });
        // CouncilBudget
        proposalTypesData[4] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            idInConfigurator: APPROVAL_VOTING_MODULE_ID
        });

        return (proposalTypes, proposalTypesData);
    }

    function _constructFundingVotingModuleData(
        string[] memory _descriptions,
        address[] memory _recipients,
        uint256[] memory _amounts,
        uint128 _criteriaValue
    )
        internal
        pure
        returns (bytes memory)
    {
        // Construct ProposalOption array
        IApprovalVotingModule.ProposalOption[] memory options =
            new IApprovalVotingModule.ProposalOption[](_descriptions.length);

        for (uint256 i = 0; i < _descriptions.length; i++) {
            address[] memory targets = new address[](1);
            uint256[] memory values = new uint256[](1);
            bytes[] memory calldatas = new bytes[](1);

            targets[0] = Predeploys.GOVERNANCE_TOKEN;
            calldatas[0] = abi.encodeCall(IERC20.transfer, (_recipients[i], _amounts[i]));

            options[i] = IApprovalVotingModule.ProposalOption({
                budgetTokensSpent: _amounts[i],
                targets: targets,
                values: values,
                calldatas: calldatas,
                description: _descriptions[i]
            });
        }

        // Calculate total budget
        uint256 totalBudget = 0;
        for (uint256 i = 0; i < _amounts.length; i++) {
            totalBudget += _amounts[i];
        }

        // Construct ProposalSettings
        IApprovalVotingModule.ProposalSettings memory approvalSettings = IApprovalVotingModule.ProposalSettings({
            maxApprovals: uint8(_descriptions.length),
            criteria: uint8(IApprovalVotingModule.PassingCriteria.Threshold),
            budgetToken: Predeploys.GOVERNANCE_TOKEN,
            criteriaValue: _criteriaValue,
            budgetAmount: uint128(totalBudget)
        });

        return abi.encode(options, approvalSettings);
    }

    /// @notice Helper function to construct voting module data for council elections
    function _constructCouncilElectionVotingModuleData(
        string[] memory _descriptions,
        uint128 _criteriaValue
    )
        internal
        pure
        returns (bytes memory)
    {
        // Construct ProposalOption array for elections (no execution calls)
        IApprovalVotingModule.ProposalOption[] memory options =
            new IApprovalVotingModule.ProposalOption[](_descriptions.length);

        for (uint256 i = 0; i < _descriptions.length; i++) {
            address[] memory targets = new address[](0);
            uint256[] memory values = new uint256[](0);
            bytes[] memory calldatas = new bytes[](0);

            options[i] = IApprovalVotingModule.ProposalOption({
                budgetTokensSpent: 0,
                targets: targets,
                values: values,
                calldatas: calldatas,
                description: _descriptions[i]
            });
        }

        // Construct ProposalSettings with TopChoices criteria
        IApprovalVotingModule.ProposalSettings memory approvalSettings = IApprovalVotingModule.ProposalSettings({
            maxApprovals: uint8(_descriptions.length),
            criteria: uint8(IApprovalVotingModule.PassingCriteria.TopChoices),
            budgetToken: address(0),
            criteriaValue: _criteriaValue,
            budgetAmount: 0
        });

        return abi.encode(options, approvalSettings);
    }

    /// @notice Helper function to construct voting module data for upgrade proposals
    function _constructOptimisticVotingModuleData(uint248 _againstThreshold) internal pure returns (bytes memory) {
        IOptimisticModule.ProposalSettings memory optimisticSettings =
            IOptimisticModule.ProposalSettings({ againstThreshold: _againstThreshold, isRelativeToVotableSupply: true });

        return abi.encode(optimisticSettings);
    }

    /// @notice Helper function to create a proposal for move to vote
    function _createUpgradeProposalForMoveToVote(
        address _proposer,
        uint248 _againstThreshold,
        string memory _proposalDescription
    )
        internal
        returns (uint256 proposalId_, bytes memory votingModuleData_)
    {
        // Calculate expected proposal ID
        votingModuleData_ = _constructOptimisticVotingModuleData(_againstThreshold);
        proposalId_ = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData_, keccak256(bytes(_proposalDescription))
        );

        // 1 vote as default for being able to move to vote
        validator.setProposalData(
            proposalId_,
            _proposer,
            ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade,
            false,
            PROPOSAL_REQUIRED_APPROVALS,
            CYCLE_NUMBER
        );
    }

    /// @notice Helper function to create a proposal for move to vote for council elections
    function _createCouncilElectionProposalForMoveToVote(
        address _proposer,
        uint128 _criteriaValue,
        string[] memory _optionsDescriptions,
        string memory _proposalDescription
    )
        internal
        returns (uint256 proposalId_, bytes memory votingModuleData_)
    {
        votingModuleData_ = _constructCouncilElectionVotingModuleData(_optionsDescriptions, _criteriaValue);
        proposalId_ = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData_, keccak256(bytes(_proposalDescription))
        );

        validator.setProposalData(
            proposalId_,
            _proposer,
            ProposalValidator.ProposalType.CouncilMemberElections,
            false,
            PROPOSAL_REQUIRED_APPROVALS,
            CYCLE_NUMBER
        );
    }

    /// @notice Helper function to create a proposal for move to vote for a funding proposal type
    function _createFundingProposalForMoveToVote(
        address _proposer,
        uint128 _criteriaValue,
        string[] memory _optionsDescriptions,
        address[] memory _optionsRecipients,
        uint256[] memory _optionsAmounts,
        string memory _proposalDescription,
        ProposalValidator.ProposalType _proposalType
    )
        internal
        returns (uint256 proposalId_, bytes memory votingModuleData_)
    {
        votingModuleData_ =
            _constructFundingVotingModuleData(_optionsDescriptions, _optionsRecipients, _optionsAmounts, _criteriaValue);
        proposalId_ = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData_, keccak256(bytes(_proposalDescription))
        );

        validator.setProposalData(
            proposalId_, _proposer, _proposalType, false, PROPOSAL_REQUIRED_APPROVALS, CYCLE_NUMBER
        );
    }

    /// @notice Helper function to setup proposal types configurator mocks
    function _mockProposalTypesConfiguratorCall(uint8 _votingModuleId) internal {
        address moduleAddress;
        if (_votingModuleId == APPROVAL_VOTING_MODULE_ID) {
            moduleAddress = approvalVotingModule;
        } else if (_votingModuleId == OPTIMISTIC_VOTING_MODULE_ID) {
            moduleAddress = optimisticVotingModule;
        }

        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.PROPOSAL_TYPES_CONFIGURATOR, ()),
            abi.encode(proposalTypesConfigurator)
        );

        _mockAndExpect(
            address(proposalTypesConfigurator),
            abi.encodeCall(IProposalTypesConfigurator.proposalTypes, (_votingModuleId)),
            abi.encode(
                IProposalTypesConfigurator.ProposalType({
                    quorum: 100,
                    approvalThreshold: 100,
                    name: "Test Proposal Type",
                    description: "Test Description",
                    module: moduleAddress
                })
            )
        );
    }

    /// @notice Helper function to mock proposal types configurator call with changed module
    function _mockProposalTypesConfiguratorCallWithUninitializedModule(uint8 _votingModuleId) internal {
        address moduleAddress;
        if (_votingModuleId == APPROVAL_VOTING_MODULE_ID) {
            moduleAddress = approvalVotingModule;
        } else if (_votingModuleId == OPTIMISTIC_VOTING_MODULE_ID) {
            moduleAddress = optimisticVotingModule;
        }

        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.PROPOSAL_TYPES_CONFIGURATOR, ()),
            abi.encode(proposalTypesConfigurator)
        );

        _mockAndExpect(
            address(proposalTypesConfigurator),
            abi.encodeCall(IProposalTypesConfigurator.proposalTypes, (_votingModuleId)),
            abi.encode(
                IProposalTypesConfigurator.ProposalType({
                    quorum: 0,
                    approvalThreshold: 0,
                    name: "",
                    description: "",
                    module: address(0)
                })
            )
        );
    }

    /// @notice Initializes the validator
    function _initializeValidator() internal virtual {
        (
            ProposalValidator.ProposalType[] memory proposalTypes,
            ProposalValidator.ProposalTypeData[] memory proposalTypesData
        ) = _getProposalTypesAndData();

        // Create mock addresses
        proposalTypesConfigurator = IProposalTypesConfigurator(makeAddr("proposalTypesConfigurator"));

        validator = new ProposalValidatorForTest(
            owner, governor, APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID, TOP_DELEGATES_ATTESTATION_SCHEMA_UID
        );

        vm.startPrank(owner);
        validator.setProposalDistributionThreshold(PROPOSAL_DISTRIBUTION_THRESHOLD);
        for (uint256 i = 0; i < proposalTypes.length; i++) {
            validator.setProposalTypeData(proposalTypes[i], proposalTypesData[i]);
        }
        validator.setVotingCycleData(CYCLE_NUMBER, START_TIMESTAMP, DURATION, PROPOSAL_DISTRIBUTION_THRESHOLD);
        vm.stopPrank();
    }

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        owner = governanceToken.owner();
        user = makeAddr("user");
        governor = IOptimismGovernor(makeAddr("governor"));
        approvalVotingModule = makeAddr("approvalVotingModule");
        optimisticVotingModule = makeAddr("optimisticVotingModule");

        // Create schemas
        vm.prank(owner);
        APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID = ISchemaRegistry(Predeploys.SCHEMA_REGISTRY).register(
            "uint8 proposalType,string date", ISchemaResolver(address(0)), true
        );

        vm.prank(owner);
        TOP_DELEGATES_ATTESTATION_SCHEMA_UID = ISchemaRegistry(Predeploys.SCHEMA_REGISTRY).register(
            "string top100,bool includePartialDelegation,string date", ISchemaResolver(address(0)), true
        );

        _initializeValidator();

        // Create attestations for top delegates
        topDelegateAttestation_A = _createTopDelegateAttestation(topDelegate_A);
    }

    /// @notice Helper to create a valid attestation for an approved proposer
    function _createApprovedProposerAttestation(
        address _delegate,
        ProposalValidator.ProposalType _proposalType
    )
        internal
        returns (bytes32)
    {
        vm.prank(owner);
        return IEAS(Predeploys.EAS).attest(
            AttestationRequest({
                schema: APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID,
                data: AttestationRequestData({
                    recipient: _delegate,
                    expirationTime: uint64(block.timestamp + ATT_EXPIRATION_TIME),
                    revocable: true,
                    refUID: bytes32(0),
                    data: abi.encode(_proposalType, "2000-01-01"),
                    value: 0
                })
            })
        );
    }

    /// @notice Helper to create a valid attestation for a top delegate
    function _createTopDelegateAttestation(address _delegate) internal returns (bytes32) {
        vm.prank(owner);
        return IEAS(Predeploys.EAS).attest(
            AttestationRequest({
                schema: TOP_DELEGATES_ATTESTATION_SCHEMA_UID,
                data: AttestationRequestData({
                    recipient: _delegate,
                    expirationTime: 0,
                    revocable: true,
                    refUID: bytes32(0),
                    data: abi.encode("top100", false, "2000-01-01"),
                    value: 0
                })
            })
        );
    }
}

/// @title ProposalValidator_SubmitUpgradeProposal_Test
/// @notice Happy path tests for submitUpgradeProposal function
contract ProposalValidator_SubmitUpgradeProposal_Test is ProposalValidator_TestInit {
    function setUp() public override {
        super.setUp();

        _setProtocolOrGovernorUpgradeProposalType();
        _setMaintenanceUpgradeProposalType();
    }

    function testFuzz_submitUpgradeProposal_maintenanceUpgrade_succeeds(
        uint248 fuzzedAgainstThreshold,
        address fuzzedProposer
    )
        public
    {
        // Assume proposer is not zero address
        vm.assume(fuzzedProposer != address(0));

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.MaintenanceUpgrade;

        // Bound fuzzedAgainstThreshold to valid range (1 to 10000 basis points)
        fuzzedAgainstThreshold = uint248(bound(fuzzedAgainstThreshold, 1, OPTIMISTIC_MODULE_PERCENT_DIVISOR));

        // Create attestation for the proposal
        bytes32 fuzzedAttestationUid = _createApprovedProposerAttestation(fuzzedProposer, proposalType);

        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(fuzzedAgainstThreshold);
        uint256 expectedId = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(PROPOSAL_DESCRIPTION))
        );

        // Mock proposalSnapshot to return 0 (proposal doesn't exist in governor)
        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (optimisticVotingModule, votingModuleData, PROPOSAL_DESCRIPTION, OPTIMISTIC_VOTING_MODULE_ID)
            ),
            abi.encode(expectedId)
        );

        // For MaintenanceUpgrade, events are: ProposalSubmitted, ProposalVotingModuleData, ProposalMovedToVote
        vm.expectEmit(address(validator));
        emit ProposalSubmitted(expectedId, fuzzedProposer, PROPOSAL_DESCRIPTION, proposalType);

        vm.expectEmit(address(validator));
        emit ProposalVotingModuleData(expectedId, votingModuleData);

        vm.expectEmit(address(validator));
        emit ProposalMovedToVote(expectedId, fuzzedProposer);

        vm.prank(fuzzedProposer);
        uint256 proposalId = validator.submitUpgradeProposal(
            fuzzedAgainstThreshold, PROPOSAL_DESCRIPTION, fuzzedAttestationUid, proposalType, CYCLE_NUMBER
        );

        assertEq(proposalId, expectedId);

        // Verify proposal data was stored correctly
        (
            address storedProposer,
            ProposalValidator.ProposalType storedProposalType,
            bool movedToVote,
            uint256 approvalCount,
            uint256 votingCycle
        ) = validator.getProposalData(proposalId);

        assertEq(storedProposer, fuzzedProposer, "Proposer should match input");
        assertEq(uint8(storedProposalType), uint8(proposalType), "Proposal type should match input");
        assertTrue(movedToVote, "MaintenanceUpgrade should be in voting immediately");
        assertEq(approvalCount, 0, "Approval count should be 0");
        assertEq(votingCycle, CYCLE_NUMBER, "Voting cycle should match input");
    }

    function testFuzz_submitUpgradeProposal_protocolOrGovernorUpgrade_succeeds(
        uint248 fuzzedAgainstThreshold,
        address fuzzedProposer
    )
        public
    {
        // Assume proposer is not zero address
        vm.assume(fuzzedProposer != address(0));

        // Bound fuzzedAgainstThreshold to valid range (1 to 10000 basis points)
        fuzzedAgainstThreshold = uint248(bound(fuzzedAgainstThreshold, 1, OPTIMISTIC_MODULE_PERCENT_DIVISOR));

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;

        // Create attestation for the proposal
        bytes32 attestationUid = _createApprovedProposerAttestation(fuzzedProposer, proposalType);

        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(fuzzedAgainstThreshold);
        uint256 expectedId = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(PROPOSAL_DESCRIPTION))
        );

        // Mock proposalSnapshot to return 0 (proposal doesn't exist in governor)
        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // For ProtocolOrGovernorUpgrade, only ProposalSubmitted and ProposalVotingModuleData events
        vm.expectEmit(address(validator));
        emit ProposalSubmitted(expectedId, fuzzedProposer, PROPOSAL_DESCRIPTION, proposalType);

        vm.expectEmit(address(validator));
        emit ProposalVotingModuleData(expectedId, votingModuleData);

        vm.prank(fuzzedProposer);
        uint256 proposalId = validator.submitUpgradeProposal(
            fuzzedAgainstThreshold, PROPOSAL_DESCRIPTION, attestationUid, proposalType, CYCLE_NUMBER
        );

        assertEq(proposalId, expectedId);

        // Verify proposal data was stored correctly
        (
            address storedProposer,
            ProposalValidator.ProposalType storedProposalType,
            bool movedToVote,
            uint256 approvalCount,
            uint256 votingCycle
        ) = validator.getProposalData(proposalId);

        assertEq(storedProposer, fuzzedProposer, "Proposer should match input");
        assertEq(uint8(storedProposalType), uint8(proposalType), "Proposal type should match input");
        assertFalse(movedToVote, "ProtocolOrGovernorUpgrade should not be in voting yet");
        assertEq(approvalCount, 0, "Approval count should be 0");
        assertEq(votingCycle, CYCLE_NUMBER, "Voting cycle should match input");
    }

    function testFuzz_submitUpgradeProposal_invalidProposalType_reverts(uint8 fuzzedProposalTypeValue) public {
        // Valid upgrade proposal types are ProtocolOrGovernorUpgrade (0) and MaintenanceUpgrade (1)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 2, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidUpgradeProposalType.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitUpgradeProposal_invalidVotingCycle_reverts(
        uint8 fuzzedProposalTypeValue,
        uint256 fuzzedVotingCycle
    )
        public
    {
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 1));
        vm.assume(fuzzedVotingCycle != CYCLE_NUMBER);
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidVotingCycle.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION, attestationUid, proposalType, fuzzedVotingCycle
        );
    }

    function testFuzz_submitUpgradeProposal_invalidAttestation_reverts(bytes32 fuzzedAttestationUid) public {
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 validAttestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        vm.assume(fuzzedAttestationUid != validAttestationUid); // Ensure it's different from valid attestation

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION, fuzzedAttestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitUpgradeProposal_unattestedProposer_reverts(address fuzzedProposer) public {
        vm.assume(fuzzedProposer != topDelegate_A); // Ensure it's different from attested proposer

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // Try to submit with different address than attested
        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(fuzzedProposer); // Different from attested topDelegate_A
        validator.submitUpgradeProposal(
            AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitUpgradeProposal_attestationExpired_reverts(uint8 fuzzedProposalTypeValue) public {
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 1));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // warp the time to after the attestation expiration time
        vm.warp(block.timestamp + ATT_EXPIRATION_TIME + 1);
        vm.expectRevert(ProposalValidator.ProposalValidator_AttestationExpired.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function test_submitUpgradeProposal_zeroAgainstThreshold_reverts() public {
        uint248 zeroThreshold = 0;
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAgainstThreshold.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(zeroThreshold, PROPOSAL_DESCRIPTION, attestationUid, proposalType, CYCLE_NUMBER);
    }

    function testFuzz_submitUpgradeProposal_exceedsMaxAgainstThreshold_reverts(uint248 fuzzedExcessiveThreshold)
        public
    {
        // Bound excessive threshold to be greater than OPTIMISTIC_MODULE_PERCENT_DIVISOR
        fuzzedExcessiveThreshold =
            uint248(bound(fuzzedExcessiveThreshold, OPTIMISTIC_MODULE_PERCENT_DIVISOR + 1, type(uint248).max));

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAgainstThreshold.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            fuzzedExcessiveThreshold, PROPOSAL_DESCRIPTION, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function test_submitUpgradeProposal_invalidVotingModule_reverts() public {
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // Mock configurator to return uninitialized module
        _mockProposalTypesConfiguratorCallWithUninitializedModule(OPTIMISTIC_VOTING_MODULE_ID);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidVotingModule.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitUpgradeProposal_duplicateProposal_reverts(uint8 fuzzedProposalTypeValue) public {
        // Bound proposal type to only upgrade proposals (0 = ProtocolOrGovernorUpgrade, 1 = MaintenanceUpgrade)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 1));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(AGAINST_THRESHOLD);
        uint256 expectedId = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(PROPOSAL_DESCRIPTION))
        );

        // Mock proposalSnapshot to return 0 for first submission
        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // For MaintenanceUpgrade, mock the governor.proposeWithModule call
        if (proposalType == ProposalValidator.ProposalType.MaintenanceUpgrade) {
            _mockAndExpect(
                address(governor),
                abi.encodeCall(
                    IOptimismGovernor.proposeWithModule,
                    (optimisticVotingModule, votingModuleData, PROPOSAL_DESCRIPTION, OPTIMISTIC_VOTING_MODULE_ID)
                ),
                abi.encode(expectedId)
            );
        }

        // Submit first proposal
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION, attestationUid, proposalType, CYCLE_NUMBER
        );

        // Attempt to submit identical proposal should revert
        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalAlreadySubmitted.selector);

        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitUpgradeProposal_proposalExistsInGovernor_reverts(uint8 fuzzedProposalTypeValue) public {
        // Bound proposal type to only upgrade proposals (0 = ProtocolOrGovernorUpgrade, 1 = MaintenanceUpgrade)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 1));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(AGAINST_THRESHOLD);
        uint256 expectedId = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(PROPOSAL_DESCRIPTION))
        );

        // Mock proposalSnapshot to return non-zero (proposal already exists in governor)
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)),
            abi.encode(1000) // Non-zero indicates proposal exists
        );

        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalAlreadySubmitted.selector);

        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitUpgradeProposal_attestationNotFromOwner_reverts(address fuzzedAttester) public {
        vm.assume(fuzzedAttester != owner); // Ensure it's not the approved owner

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;

        // Create attestation but don't use proper owner as attester
        vm.prank(fuzzedAttester); // Not the owner
        bytes32 invalidAttestation = IEAS(Predeploys.EAS).attest(
            AttestationRequest({
                schema: APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID,
                data: AttestationRequestData({
                    recipient: topDelegate_A,
                    expirationTime: uint64(block.timestamp + ATT_EXPIRATION_TIME),
                    revocable: false,
                    refUID: bytes32(0),
                    data: abi.encode(proposalType, "2000-01-01"),
                    value: 0
                })
            })
        );

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION, invalidAttestation, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitUpgradeProposal_attestationRevoked_reverts(uint8 fuzzedProposalTypeValue) public {
        // Bound proposal type to only upgrade proposals (0 = ProtocolOrGovernorUpgrade, 1 = MaintenanceUpgrade)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 1));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // Create valid attestation first (make it revocable)
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // Revoke the attestation
        vm.prank(owner);
        IEAS(Predeploys.EAS).revoke(
            RevocationRequest({
                schema: APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID,
                data: RevocationRequestData({ uid: attestationUid, value: 0 })
            })
        );

        vm.expectRevert(ProposalValidator.ProposalValidator_AttestationRevoked.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function test_submitUpgradeProposal_proposalIdMismatch_reverts(uint256 fuzzedProposalId) public {
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.MaintenanceUpgrade;
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(AGAINST_THRESHOLD);
        uint256 expectedId = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(PROPOSAL_DESCRIPTION))
        );

        vm.assume(fuzzedProposalId != expectedId); // Ensure proposalId is different from expectedId

        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        // Mock the proposeWithModule call to return a different proposalId
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (optimisticVotingModule, votingModuleData, PROPOSAL_DESCRIPTION, OPTIMISTIC_VOTING_MODULE_ID)
            ),
            abi.encode(fuzzedProposalId)
        );

        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalIdMismatch.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION, attestationUid, proposalType, CYCLE_NUMBER
        );
    }
}

/// @title ProposalValidator_SubmitCouncilMemberElectionsProposal_Test
/// @notice Happy path tests for submitCouncilMemberElectionsProposal function
contract ProposalValidator_SubmitCouncilMemberElectionsProposal_Test is ProposalValidator_TestInit {
    uint128 criteriaValue;
    string[] optionDescriptions;
    bytes32 approvedProposerAttestationUid;

    function setUp() public override {
        super.setUp();

        _setCouncilMemberElectionsProposalType();

        criteriaValue = 2;
        optionDescriptions = new string[](3);
        optionDescriptions[0] = "Candidate A";
        optionDescriptions[1] = "Candidate B";
        optionDescriptions[2] = "Candidate C";

        approvedProposerAttestationUid =
            _createApprovedProposerAttestation(topDelegate_A, ProposalValidator.ProposalType.CouncilMemberElections);
    }

    function testFuzz_submitCouncilMemberElectionsProposal_succeeds(
        uint8 fuzzedOptionCount,
        uint128 fuzzedCriteriaValue
    )
        public
    {
        fuzzedOptionCount = uint8(bound(fuzzedOptionCount, 2, 5)); // Minimum 2 options to have valid criteria <
            // optionCount
        fuzzedCriteriaValue = uint128(bound(fuzzedCriteriaValue, 1, fuzzedOptionCount - 1)); // Must be less than
            // optionCount

        // Create dynamic array of option descriptions based on option count
        string[] memory fuzzedOptionDescriptions = new string[](fuzzedOptionCount);
        for (uint256 i = 0; i < fuzzedOptionCount; i++) {
            fuzzedOptionDescriptions[i] = string(abi.encodePacked("Candidate ", vm.toString(i)));
        }

        // Calculate expected proposal ID
        bytes memory votingModuleData =
            _constructCouncilElectionVotingModuleData(fuzzedOptionDescriptions, fuzzedCriteriaValue);
        uint256 expectedId = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData, keccak256(bytes(PROPOSAL_DESCRIPTION))
        );

        // Mock proposalSnapshot to return 0 (proposal doesn't exist in governor)
        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        // Expect ProposalSubmitted event
        vm.expectEmit(address(validator));
        emit ProposalSubmitted(
            expectedId, topDelegate_A, PROPOSAL_DESCRIPTION, ProposalValidator.ProposalType.CouncilMemberElections
        );

        // Expect ProposalVotingModuleData event
        vm.expectEmit(address(validator));
        emit ProposalVotingModuleData(expectedId, votingModuleData);

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.prank(topDelegate_A);
        uint256 proposalId = validator.submitCouncilMemberElectionsProposal(
            fuzzedCriteriaValue,
            fuzzedOptionDescriptions,
            PROPOSAL_DESCRIPTION,
            approvedProposerAttestationUid,
            CYCLE_NUMBER
        );

        assertEq(proposalId, expectedId);

        // Verify proposal data was stored correctly
        (
            address proposer,
            ProposalValidator.ProposalType proposalType,
            bool movedToVote,
            uint256 approvalCount,
            uint256 votingCycle
        ) = validator.getProposalData(proposalId);

        assertEq(proposer, topDelegate_A, "Proposer should be topDelegate_A");
        assertEq(
            uint8(proposalType),
            uint8(ProposalValidator.ProposalType.CouncilMemberElections),
            "Proposal type should be CouncilMemberElections"
        );
        assertFalse(movedToVote, "Proposal should not be in voting yet");
        assertEq(approvalCount, 0, "Approval count should be 0");
        assertEq(votingCycle, CYCLE_NUMBER, "Voting cycle should match input");
    }

    function testFuzz_submitCouncilMemberElectionsProposal_invalidVotingCycle_reverts(uint256 fuzzedVotingCycle)
        public
    {
        vm.assume(fuzzedVotingCycle != CYCLE_NUMBER);
        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidVotingCycle.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, PROPOSAL_DESCRIPTION, approvedProposerAttestationUid, fuzzedVotingCycle
        );
    }

    function testFuzz_submitCouncilMemberElectionsProposal_invalidAttestation_reverts(bytes32 fuzzedAttestationUid)
        public
    {
        vm.assume(fuzzedAttestationUid != approvedProposerAttestationUid); // Ensure it's different from valid
            // attestation

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, PROPOSAL_DESCRIPTION, fuzzedAttestationUid, CYCLE_NUMBER
        );
    }

    function testFuzz_submitCouncilMemberElectionsProposal_unattestedProposer_reverts(address fuzzedProposer) public {
        vm.assume(fuzzedProposer != topDelegate_A); // Ensure it's different from attested proposer

        // Try to submit with different address than attested
        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(fuzzedProposer); // Different from attested topDelegate_A
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, PROPOSAL_DESCRIPTION, approvedProposerAttestationUid, CYCLE_NUMBER
        );
    }

    function testFuzz_submitCouncilMemberElectionsProposal_attestationExpired_reverts() public {
        // warp the time to after the attestation expiration time
        vm.warp(block.timestamp + ATT_EXPIRATION_TIME + 1);
        vm.expectRevert(ProposalValidator.ProposalValidator_AttestationExpired.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, PROPOSAL_DESCRIPTION, approvedProposerAttestationUid, CYCLE_NUMBER
        );
    }

    function test_submitCouncilMemberElectionsProposal_zeroOptions_reverts() public {
        string[] memory emptyOptions = new string[](0);
        uint128 zeroCriteriaValue = 0; // 0 so it doesnt exceed the options length

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidOptionsLength.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            zeroCriteriaValue, emptyOptions, PROPOSAL_DESCRIPTION, approvedProposerAttestationUid, CYCLE_NUMBER
        );
    }

    function test_submitCouncilMemberElectionsProposal_duplicateProposal_reverts() public {
        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructCouncilElectionVotingModuleData(optionDescriptions, criteriaValue);
        uint256 expectedId = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData, keccak256(bytes(PROPOSAL_DESCRIPTION))
        );

        // Mock proposalSnapshot to return 0 for first submission
        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        // Submit first proposal
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, PROPOSAL_DESCRIPTION, approvedProposerAttestationUid, CYCLE_NUMBER
        );

        // Attempt to submit identical proposal should revert
        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalAlreadySubmitted.selector);

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, PROPOSAL_DESCRIPTION, approvedProposerAttestationUid, CYCLE_NUMBER
        );
    }

    function test_submitCouncilMemberElectionsProposal_proposalExistsInGovernor_reverts() public {
        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructCouncilElectionVotingModuleData(optionDescriptions, criteriaValue);
        uint256 expectedId = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData, keccak256(bytes(PROPOSAL_DESCRIPTION))
        );

        // Mock proposalSnapshot to return non-zero (proposal already exists in governor)
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)),
            abi.encode(1000) // Non-zero indicates proposal exists
        );

        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalAlreadySubmitted.selector);

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, PROPOSAL_DESCRIPTION, approvedProposerAttestationUid, CYCLE_NUMBER
        );
    }

    function testFuzz_submitCouncilMemberElectionsProposal_attestationNotFromOwner_reverts(address fuzzedAttester)
        public
    {
        vm.assume(fuzzedAttester != owner); // Ensure it's not the approved owner

        // Create attestation but don't use proper owner as attester
        vm.prank(fuzzedAttester); // Not the owner
        bytes32 invalidAttestation = IEAS(Predeploys.EAS).attest(
            AttestationRequest({
                schema: APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID,
                data: AttestationRequestData({
                    recipient: topDelegate_A,
                    expirationTime: uint64(block.timestamp + ATT_EXPIRATION_TIME),
                    revocable: false,
                    refUID: bytes32(0),
                    data: abi.encode(ProposalValidator.ProposalType.CouncilMemberElections, "2000-01-01"),
                    value: 0
                })
            })
        );

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, PROPOSAL_DESCRIPTION, invalidAttestation, CYCLE_NUMBER
        );
    }

    function testFuzz_submitCouncilMemberElectionsProposal_criteriaValueExceedsOptionsLength_reverts(
        uint128 fuzzedCriteriaValue
    )
        public
    {
        // Bound fuzzedCriteriaValue to be greater than options length
        fuzzedCriteriaValue = uint128(bound(fuzzedCriteriaValue, optionDescriptions.length + 1, type(uint128).max));

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidCriteriaValue.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            fuzzedCriteriaValue, optionDescriptions, PROPOSAL_DESCRIPTION, approvedProposerAttestationUid, CYCLE_NUMBER
        );
    }

    function test_submitCouncilMemberElectionsProposal_attestationRevoked_reverts() public {
        // Create valid attestation first (make it revocable)
        bytes32 revocableAttestationUid =
            _createApprovedProposerAttestation(topDelegate_A, ProposalValidator.ProposalType.CouncilMemberElections);

        // Revoke the attestation
        vm.prank(owner);
        IEAS(Predeploys.EAS).revoke(
            RevocationRequest({
                schema: APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID,
                data: RevocationRequestData({ uid: revocableAttestationUid, value: 0 })
            })
        );

        vm.expectRevert(ProposalValidator.ProposalValidator_AttestationRevoked.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, PROPOSAL_DESCRIPTION, revocableAttestationUid, CYCLE_NUMBER
        );
    }

    function test_submitCouncilMemberElectionsProposal_invalidVotingModule_reverts() public {
        // Mock configurator to return uninitialized module
        _mockProposalTypesConfiguratorCallWithUninitializedModule(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidVotingModule.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, PROPOSAL_DESCRIPTION, approvedProposerAttestationUid, CYCLE_NUMBER
        );
    }
}

/// @title ProposalValidator_SubmitFundingProposal_Test
/// @notice Happy path tests for submitFundingProposal function
contract ProposalValidator_SubmitFundingProposal_Test is ProposalValidator_TestInit {
    uint128 constant FUNDING_CRITERIA_VALUE = 1000;

    function setUp() public override {
        super.setUp();

        _setGovernanceFundProposalType();
        _setCouncilBudgetProposalType();
    }

    function testFuzz_submitFundingProposal_succeeds(
        uint8 fuzzedProposalTypeValue,
        uint8 fuzzedOptionCount,
        uint256 fuzzedAmount,
        address fuzzedProposer
    )
        public
    {
        // Assume proposer is not zero address
        vm.assume(fuzzedProposer != address(0));
        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // Bound option count between 1 and 5 for reasonable test execution
        fuzzedOptionCount = uint8(bound(fuzzedOptionCount, 1, 5));

        // Bound amount from 0 to PROPOSAL_DISTRIBUTION_THRESHOLD (inclusive)
        fuzzedAmount = bound(fuzzedAmount, 0, PROPOSAL_DISTRIBUTION_THRESHOLD);

        // Start with minimal arrays and extend based on option count
        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(fuzzedOptionCount);

        // fuzz the amounts
        for (uint256 i = 0; i < fuzzedOptionCount; i++) {
            amounts[i] = fuzzedAmount;
        }

        // Calculate expected proposal ID
        bytes memory votingModuleData =
            _constructFundingVotingModuleData(descriptions, recipients, amounts, FUNDING_CRITERIA_VALUE);
        uint256 expectedId = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData, keccak256(bytes(PROPOSAL_DESCRIPTION))
        );

        // Mock proposalSnapshot to return 0 (proposal doesn't exist in governor)
        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        // Expect ProposalSubmitted event
        vm.expectEmit(address(validator));
        emit ProposalSubmitted(expectedId, fuzzedProposer, PROPOSAL_DESCRIPTION, proposalType);

        // Expect ProposalVotingModuleData event
        vm.expectEmit(address(validator));
        emit ProposalVotingModuleData(expectedId, votingModuleData);

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.prank(fuzzedProposer);
        uint256 proposalId = validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, PROPOSAL_DESCRIPTION, proposalType, CYCLE_NUMBER
        );

        assertEq(proposalId, expectedId);

        // Verify proposal data was stored correctly
        (
            address storedProposer,
            ProposalValidator.ProposalType storedProposalType,
            bool movedToVote,
            uint256 approvalCount,
            uint256 votingCycle
        ) = validator.getProposalData(proposalId);

        assertEq(storedProposer, fuzzedProposer, "Proposer should match input");
        assertEq(uint8(storedProposalType), uint8(proposalType), "Proposal type should match input");
        assertFalse(movedToVote, "Proposal should not be in voting yet");
        assertEq(approvalCount, 0, "Approval count should be 0");
        assertEq(votingCycle, CYCLE_NUMBER, "Voting cycle should match input");
    }

    function testFuzz_submitFundingProposal_invalidProposalType_reverts(uint8 fuzzedProposalTypeValue) public {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 2));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(1);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidFundingProposalType.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, PROPOSAL_DESCRIPTION, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_invalidVotingCycle_reverts(
        uint8 fuzzedProposalTypeValue,
        uint256 fuzzedVotingCycle
    )
        public
    {
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);
        vm.assume(fuzzedVotingCycle != CYCLE_NUMBER);

        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(1);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidVotingCycle.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE,
            descriptions,
            recipients,
            amounts,
            PROPOSAL_DESCRIPTION,
            proposalType,
            fuzzedVotingCycle
        );
    }

    function testFuzz_submitFundingProposal_mismatchedDescriptionsLength_reverts(
        uint8 matchingLength,
        uint8 mismatchedLength,
        uint8 fuzzedProposalTypeValue
    )
        public
    {
        // Bound lengths to reasonable values (1-50) and ensure they're different
        matchingLength = uint8(bound(matchingLength, 1, 50));
        mismatchedLength = uint8(bound(mismatchedLength, 1, 50));
        vm.assume(matchingLength != mismatchedLength);

        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // Create arrays - recipients and amounts match, descriptions are different
        string[] memory mismatchedDescriptions = new string[](mismatchedLength);
        address[] memory matchingRecipients = new address[](matchingLength);
        uint256[] memory matchingAmounts = new uint256[](matchingLength);

        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalTypesDataLengthMismatch.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE,
            mismatchedDescriptions,
            matchingRecipients,
            matchingAmounts,
            PROPOSAL_DESCRIPTION,
            proposalType,
            CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_mismatchedRecipientsLength_reverts(
        uint8 matchingLength,
        uint8 mismatchedLength,
        uint8 fuzzedProposalTypeValue
    )
        public
    {
        // Bound lengths to reasonable values (1-50) and ensure they're different
        matchingLength = uint8(bound(matchingLength, 1, 50));
        mismatchedLength = uint8(bound(mismatchedLength, 1, 50));
        vm.assume(matchingLength != mismatchedLength);

        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // Create arrays - descriptions and amounts match, recipients are different
        string[] memory matchingDescriptions = new string[](matchingLength);
        address[] memory mismatchedRecipients = new address[](mismatchedLength);
        uint256[] memory matchingAmounts = new uint256[](matchingLength);

        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalTypesDataLengthMismatch.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE,
            matchingDescriptions,
            mismatchedRecipients,
            matchingAmounts,
            PROPOSAL_DESCRIPTION,
            proposalType,
            CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_mismatchedAmountsLength_reverts(
        uint8 matchingLength,
        uint8 mismatchedLength,
        uint8 fuzzedProposalTypeValue
    )
        public
    {
        // Bound lengths to reasonable values (1-50) and ensure they're different
        matchingLength = uint8(bound(matchingLength, 1, 50));
        mismatchedLength = uint8(bound(mismatchedLength, 1, 50));
        vm.assume(matchingLength != mismatchedLength);

        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // Create arrays - descriptions and recipients match, amounts are different
        string[] memory matchingDescriptions = new string[](matchingLength);
        address[] memory matchingRecipients = new address[](matchingLength);
        uint256[] memory mismatchedAmounts = new uint256[](mismatchedLength);

        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalTypesDataLengthMismatch.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE,
            matchingDescriptions,
            matchingRecipients,
            mismatchedAmounts,
            PROPOSAL_DESCRIPTION,
            proposalType,
            CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_exceedsProposalDistributionThreshold_reverts(
        uint256 fuzzedExcessAmount,
        uint8 fuzzedProposalTypeValue
    )
        public
    {
        // Bound excess amount to be greater than PROPOSAL_DISTRIBUTION_THRESHOLD
        fuzzedExcessAmount = bound(fuzzedExcessAmount, PROPOSAL_DISTRIBUTION_THRESHOLD + 1, type(uint128).max);

        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // Create arrays with excessive amount
        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(1);
        amounts[0] = fuzzedExcessAmount;

        vm.expectRevert(ProposalValidator.ProposalValidator_ExceedsDistributionThreshold.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, PROPOSAL_DESCRIPTION, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_duplicateProposal_reverts(uint8 fuzzedProposalTypeValue) public {
        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(1);

        // Calculate expected proposal ID
        bytes memory votingModuleData =
            _constructFundingVotingModuleData(descriptions, recipients, amounts, FUNDING_CRITERIA_VALUE);
        uint256 expectedId = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData, keccak256(bytes(PROPOSAL_DESCRIPTION))
        );

        // Mock proposalSnapshot to return 0 for first submission
        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        // Submit first proposal
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, PROPOSAL_DESCRIPTION, proposalType, CYCLE_NUMBER
        );

        // Attempt to submit identical proposal
        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalAlreadySubmitted.selector);

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, PROPOSAL_DESCRIPTION, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_proposalExistsInGovernor_reverts(uint8 fuzzedProposalTypeValue) public {
        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(1);

        // Calculate expected proposal ID
        bytes memory votingModuleData =
            _constructFundingVotingModuleData(descriptions, recipients, amounts, FUNDING_CRITERIA_VALUE);
        uint256 expectedId = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData, keccak256(bytes(PROPOSAL_DESCRIPTION))
        );

        // Mock proposalSnapshot to return non-zero (proposal already exists in governor)
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)),
            abi.encode(1000) // Non-zero indicates proposal exists
        );

        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalAlreadySubmitted.selector);

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, PROPOSAL_DESCRIPTION, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_zeroOptionsLength_reverts(uint8 fuzzedProposalTypeValue) public {
        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);
        string[] memory emptyDescriptions = new string[](0);
        address[] memory emptyRecipients = new address[](0);
        uint256[] memory emptyAmounts = new uint256[](0);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidOptionsLength.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE,
            emptyDescriptions,
            emptyRecipients,
            emptyAmounts,
            PROPOSAL_DESCRIPTION,
            proposalType,
            CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_exceedsMaxOptionsLength_reverts(
        uint256 fuzzedTooManyOptions,
        uint8 fuzzedProposalTypeValue
    )
        public
    {
        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);
        // Create arrays with more than 255 options (exceeds allowed uint8 max)
        fuzzedTooManyOptions = uint256(bound(fuzzedTooManyOptions, 256, 512));
        string[] memory tooManyDescriptions = new string[](fuzzedTooManyOptions);
        address[] memory tooManyRecipients = new address[](fuzzedTooManyOptions);
        uint256[] memory tooManyAmounts = new uint256[](fuzzedTooManyOptions);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidOptionsLength.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE,
            tooManyDescriptions,
            tooManyRecipients,
            tooManyAmounts,
            PROPOSAL_DESCRIPTION,
            proposalType,
            CYCLE_NUMBER
        );
    }

    function test_submitFundingProposal_invalidVotingModule_reverts() public {
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.GovernanceFund;
        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(1);

        // Mock configurator to return uninitialized module
        _mockProposalTypesConfiguratorCallWithUninitializedModule(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidVotingModule.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, PROPOSAL_DESCRIPTION, proposalType, CYCLE_NUMBER
        );
    }

    function test_submitFundingProposal_invalidTotalBudget_reverts(
        uint8 fuzzedProposalTypeValue,
        uint256 fuzzedAmount
    )
        public
    {
        fuzzedAmount = bound(fuzzedAmount, type(uint136).max, type(uint192).max);
        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(1);

        vm.prank(owner);
        validator.setProposalDistributionThreshold(type(uint256).max);

        amounts[0] = fuzzedAmount;
        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidTotalBudget.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, PROPOSAL_DESCRIPTION, proposalType, CYCLE_NUMBER
        );
    }
}

/// @title ProposalValidator_ApproveProposal_Test
/// @notice Happy path tests for approveProposal function
contract ProposalValidator_ApproveProposal_Test is ProposalValidator_TestInit {
    function setUp() public override {
        super.setUp();

        // create the previous voting cycle
        // cycle number decreased by 1 and start time CYCLE_DURATION before the current cycle
        vm.prank(owner);
        validator.setVotingCycleData(
            CYCLE_NUMBER - 1, START_TIMESTAMP - DURATION, DURATION, VOTING_CYCLE_DISTRIBUTION_LIMIT
        );
        // warp to the start of the previous cycle
        vm.warp(START_TIMESTAMP - DURATION);
    }

    function test_approveProposal_succeeds(uint256 fuzzedProposalId, uint8 fuzzedProposalTypeValue) public {
        // Ensure the proposal ID is not 0
        vm.assume(fuzzedProposalId != 0);

        // Bound the proposal type to valid enum values (0-4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // Set mock proposal data of a random proposal in the validator contract
        validator.setProposalData(fuzzedProposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);

        // Expect event to be emitted when approving
        vm.expectEmit(address(validator));
        emit ProposalApproved(fuzzedProposalId, topDelegate_A);

        // warp to the start of current cycle if the proposal is ProtocolOrGovernorUpgrade
        if (proposalType == ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade) {
            vm.warp(START_TIMESTAMP);
        }

        // Approve the proposal, use the attestation of the top delegate that was created in setUp
        vm.prank(topDelegate_A);
        validator.approveProposal(fuzzedProposalId, topDelegateAttestation_A);

        // Check that the proposal data has been updated
        assertTrue(validator.hasDelegateApproved(fuzzedProposalId, topDelegate_A));

        (,,, uint256 approvalCount,) = validator.getProposalData(fuzzedProposalId);
        assertEq(approvalCount, 1);
    }

    function test_approveProposal_proposalDoesNotExist_reverts(uint256 fuzzedProposalId) public {
        // There is no stored proposal data so this will revert
        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalDoesNotExist.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(fuzzedProposalId, topDelegateAttestation_A);
    }

    function test_approveProposal_proposalAlreadyApproved_reverts(
        uint256 fuzzedProposalId,
        uint8 fuzzedProposalTypeValue
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);
        // set proposal data so that the proposal exists
        validator.setProposalData(fuzzedProposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);

        // Mock the proposal as already approved by the top delegate
        validator.mockApproveProposal(fuzzedProposalId, topDelegate_A);
        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyApproved.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(fuzzedProposalId, topDelegateAttestation_A);
    }

    function test_approveProposal_proposalAlreadyMovedToVote_reverts(
        uint256 fuzzedProposalId,
        uint8 fuzzedProposalTypeValue
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);
        // set proposal data so that the proposal exists and set movedToVote to true
        validator.setProposalData(fuzzedProposalId, topDelegate_A, proposalType, true, 0, CYCLE_NUMBER);

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyMovedToVote.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(fuzzedProposalId, topDelegateAttestation_A);
    }

    function test_approveProposal_previousVotingCycleNotStarted_reverts(
        uint256 fuzzedProposalId,
        uint8 fuzzedProposalTypeValue
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);
        // set proposal data so that the proposal exists
        validator.setProposalData(fuzzedProposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);

        // warp before the start of the previous cycle so that it reverts
        vm.warp(START_TIMESTAMP - DURATION - 1);

        vm.expectRevert(IProposalValidator.ProposalValidator_PreviousVotingCycleNotStarted.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(fuzzedProposalId, topDelegateAttestation_A);
    }

    function test_approveProposal_invalidVotingCycle_reverts(
        uint256 fuzzedProposalId,
        uint8 fuzzedProposalTypeValue,
        uint256 fuzzedVotingCycle
    )
        public
    {
        vm.assume(fuzzedVotingCycle != CYCLE_NUMBER && fuzzedVotingCycle != 0);
        vm.assume(fuzzedVotingCycle != CYCLE_NUMBER + 1); // Avoid existing cycle

        // Bound the proposal type to valid enum values (0-4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // set proposal data so that the proposal exists
        validator.setProposalData(fuzzedProposalId, topDelegate_A, proposalType, false, 0, fuzzedVotingCycle);

        // warp to start of current voting cycle if the proposal is ProtocolOrGovernorUpgrade
        if (proposalType == ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade) {
            vm.warp(START_TIMESTAMP);
        }

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidVotingCycle.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(fuzzedProposalId, topDelegateAttestation_A);
    }

    function test_approveProposal_invalidSchema_reverts(
        uint256 fuzzedProposalId,
        uint8 fuzzedProposalTypeValue
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // create a new schema
        vm.prank(topDelegate_A);
        bytes32 _invalidSchemaUid = ISchemaRegistry(Predeploys.SCHEMA_REGISTRY).register(
            "string top100, string date", ISchemaResolver(address(0)), true
        );

        // create an attestation with the new schema
        vm.prank(topDelegate_A);
        bytes32 _invalidAttestationUid = IEAS(Predeploys.EAS).attest(
            AttestationRequest({
                schema: _invalidSchemaUid,
                data: AttestationRequestData({
                    recipient: topDelegate_A,
                    expirationTime: 0,
                    revocable: true,
                    refUID: bytes32(0),
                    data: abi.encode("top100", false, "2000-01-01"),
                    value: 0
                })
            })
        );

        // set proposal data so that the proposal exists
        validator.setProposalData(fuzzedProposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);
        // warp to start of current voting cycle if the proposal is ProtocolOrGovernorUpgrade
        if (proposalType == ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade) {
            vm.warp(START_TIMESTAMP);
        }

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestationSchema.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(fuzzedProposalId, _invalidAttestationUid);
    }

    function test_approveProposal_attestationRevoked_reverts(
        uint256 fuzzedProposalId,
        uint8 fuzzedProposalTypeValue
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);
        // set proposal data so that the proposal exists
        validator.setProposalData(fuzzedProposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);
        // warp to start of current voting cycle if the proposal is ProtocolOrGovernorUpgrade
        if (proposalType == ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade) {
            vm.warp(START_TIMESTAMP);
        }

        // revoke the attestation
        vm.prank(owner);
        IEAS(Predeploys.EAS).revoke(
            RevocationRequest({
                schema: TOP_DELEGATES_ATTESTATION_SCHEMA_UID,
                data: RevocationRequestData({ uid: topDelegateAttestation_A, value: 0 })
            })
        );

        vm.expectRevert(IProposalValidator.ProposalValidator_AttestationRevoked.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(fuzzedProposalId, topDelegateAttestation_A);
    }

    function test_approveProposal_attestationCreatedAfterPreviousVotingCycle_reverts(
        uint256 fuzzedProposalId,
        uint8 fuzzedProposalTypeValue
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // create a new delegate and attestation
        address _delegate = makeAddr("delegate");
        bytes32 _attestationUid;

        // create the attestation based on the proposal type
        if (proposalType == ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade) {
            // warp to after the start of the current voting cycle if the proposal is ProtocolOrGovernorUpgrade
            // because this proposal can be submitted and approved outside of a voting cycle
            vm.warp(START_TIMESTAMP + 1);
            _attestationUid = _createTopDelegateAttestation(_delegate);
        } else {
            // warp to after the start of the previous cycle for an other proposal type
            vm.warp(START_TIMESTAMP - DURATION + 1);
            _attestationUid = _createTopDelegateAttestation(_delegate);
        }

        // set proposal data so that the proposal exists
        validator.setProposalData(fuzzedProposalId, _delegate, proposalType, false, 0, CYCLE_NUMBER);

        // warp to after the start of the current voting cycle
        vm.warp(START_TIMESTAMP + 2);
        vm.expectRevert(IProposalValidator.ProposalValidator_AttestationCreatedAfterLastVotingCycle.selector);
        vm.prank(_delegate);
        validator.approveProposal(fuzzedProposalId, _attestationUid);
    }

    function test_approveProposal_invalidAttestationCaller_reverts(
        uint256 fuzzedProposalId,
        uint8 fuzzedProposalTypeValue,
        address fuzzedCaller
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // Ensure the caller is not a top delegate
        vm.assume(fuzzedCaller != topDelegate_A);

        // Set mock proposal data of a random proposal in the validator contract
        validator.setProposalData(fuzzedProposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);
        // warp to start of current voting cycle if the proposal is ProtocolOrGovernorUpgrade
        if (proposalType == ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade) {
            vm.warp(START_TIMESTAMP);
        }

        // Expect the invalid attestation error to be reverted
        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(fuzzedCaller);
        validator.approveProposal(fuzzedProposalId, topDelegateAttestation_A);
    }

    function test_approveProposal_invalidAttestationPartialDelegation_reverts(
        uint256 fuzzedProposalId,
        uint8 fuzzedProposalTypeValue
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // Set mock proposal data of a random proposal in the validator contract
        validator.setProposalData(fuzzedProposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);
        // warp to start of current voting cycle if the proposal is ProtocolOrGovernorUpgrade
        if (proposalType == ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade) {
            vm.warp(START_TIMESTAMP);
        }

        // create an attestation with partial delegation
        vm.prank(owner);
        bytes32 _attestationUidWithPartialDelegation = IEAS(Predeploys.EAS).attest(
            AttestationRequest({
                schema: TOP_DELEGATES_ATTESTATION_SCHEMA_UID,
                data: AttestationRequestData({
                    recipient: topDelegate_A,
                    expirationTime: 0,
                    revocable: true,
                    refUID: bytes32(0),
                    data: abi.encode("top100", true, "2000-01-01"),
                    value: 0
                })
            })
        );

        // Expect the invalid attestation error to be reverted
        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(fuzzedProposalId, _attestationUidWithPartialDelegation);
    }

    function test_approveProposal_nonExistentAttestation_reverts(
        uint256 fuzzedProposalId,
        uint8 fuzzedProposalTypeValue,
        bytes32 fuzzedNonExistentAttestationUid
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // Ensure the attestation uid is not one of the valid ones
        vm.assume(fuzzedNonExistentAttestationUid != topDelegateAttestation_A);

        // Set mock proposal data of a random proposal in the validator contract
        validator.setProposalData(fuzzedProposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);
        // warp to start of current voting cycle if the proposal is ProtocolOrGovernorUpgrade
        if (proposalType == ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade) {
            vm.warp(START_TIMESTAMP);
        }

        // Expect the invalid attestation error to be reverted when attestation doesn't exist
        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(fuzzedProposalId, fuzzedNonExistentAttestationUid);
    }
}

/// @title ProposalValidator_MoveToVoteProtocolOrGovernorUpgradeProposal_Test
/// @notice Happy path tests for moveToVoteProtocolOrGovernorUpgradeProposal function
contract ProposalValidator_MoveToVoteProtocolOrGovernorUpgradeProposal_Test is ProposalValidator_TestInit {
    ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
    bytes votingModuleData;
    uint256 expectedId;

    function setUp() public override {
        super.setUp();

        (expectedId, votingModuleData) =
            _createUpgradeProposalForMoveToVote(approvedProposer, AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION);
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_succeeds() public {
        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (optimisticVotingModule, votingModuleData, PROPOSAL_DESCRIPTION, OPTIMISTIC_VOTING_MODULE_ID)
            ),
            abi.encode(expectedId)
        );

        // Expect the ProposalMovedToVote event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalMovedToVote(expectedId, approvedProposer);

        // Move to vote
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION);

        // Check that the proposal is in voting
        (,, bool movedToVote,,) = validator.getProposalData(expectedId);
        assertTrue(movedToVote, "Proposal should be in voting");
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_invalidProposer_reverts(address fuzzedCaller) public {
        vm.assume(fuzzedCaller != approvedProposer);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidProposer.selector);
        vm.prank(fuzzedCaller);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION);
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_invalidProposal_reverts(uint248 fuzzedAgainstThreshold)
        public
    {
        // This will generate a different proposal ID which will make the proposal type wrong
        vm.assume(fuzzedAgainstThreshold != AGAINST_THRESHOLD);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidProposal.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(fuzzedAgainstThreshold, PROPOSAL_DESCRIPTION);
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_insufficientApprovals_reverts() public {
        // Set proposal data approved count to 0 since it is 1 by the approval on the setUp
        validator.setProposalData(expectedId, approvedProposer, proposalType, false, 0, CYCLE_NUMBER);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InsufficientApprovals.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION);
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_proposalAlreadyMovedToVote_reverts() public {
        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // Set proposal data movedToVote to true
        validator.setProposalData(expectedId, approvedProposer, proposalType, true, 1, CYCLE_NUMBER);

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyMovedToVote.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION);
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_proposalIdMismatch_reverts(uint256 fuzzedProposalId)
        public
    {
        vm.assume(fuzzedProposalId != expectedId);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (optimisticVotingModule, votingModuleData, PROPOSAL_DESCRIPTION, OPTIMISTIC_VOTING_MODULE_ID)
            ),
            abi.encode(uint256(fuzzedProposalId))
        );

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalIdMismatch.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(AGAINST_THRESHOLD, PROPOSAL_DESCRIPTION);
    }
}

contract ProposalValidator_MoveToVoteCouncilMemberElectionsProposal_Test is ProposalValidator_TestInit {
    ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.CouncilMemberElections;
    uint128 criteriaValue = 1;
    uint256 expectedId;
    bytes votingModuleData;
    string[] optionsDescriptions = new string[](2);

    function setUp() public override {
        super.setUp();

        // Create a proposal for move to vote with 1 top choice and 2 options
        optionsDescriptions[0] = "Option 1";
        optionsDescriptions[1] = "Option 2";
        (expectedId, votingModuleData) = _createCouncilElectionProposalForMoveToVote(
            approvedProposer, criteriaValue, optionsDescriptions, PROPOSAL_DESCRIPTION
        );
    }

    function test_moveToVoteCouncilMemberElectionsProposal_succeeds() public {
        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (approvalVotingModule, votingModuleData, PROPOSAL_DESCRIPTION, APPROVAL_VOTING_MODULE_ID)
            ),
            abi.encode(expectedId)
        );

        // Expect the ProposalMovedToVote event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalMovedToVote(expectedId, approvedProposer);

        // Move to vote
        vm.warp(START_TIMESTAMP + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, PROPOSAL_DESCRIPTION);

        // Check that the proposal is in voting
        (,, bool movedToVote,,) = validator.getProposalData(expectedId);
        assertTrue(movedToVote, "Proposal should be in voting");
    }

    function test_moveToVoteCouncilMemberElectionsProposal_invalidProposer_reverts(address fuzzedCaller) public {
        vm.assume(fuzzedCaller != approvedProposer);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidProposer.selector);
        vm.prank(fuzzedCaller);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, PROPOSAL_DESCRIPTION);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_invalidProposal_reverts() public {
        // This will generate a different proposal ID which will make the proposal type wrong
        uint128 _criteriaValue = 2; // we use 2 since it is the max based on the created proposal in setUp

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidProposal.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(_criteriaValue, optionsDescriptions, PROPOSAL_DESCRIPTION);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_invalidOptionsLength_reverts() public {
        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidOptionsLength.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, new string[](0), PROPOSAL_DESCRIPTION);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidOptionsLength.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, new string[](256), PROPOSAL_DESCRIPTION);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_insufficientApprovals_reverts() public {
        // Set proposal data approved count to 0 since it is 1 by the approval on the setUp
        validator.setProposalData(expectedId, approvedProposer, proposalType, false, 0, CYCLE_NUMBER);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InsufficientApprovals.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, PROPOSAL_DESCRIPTION);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_proposalAlreadyMovedToVote_reverts() public {
        // Set proposal data movedToVote to true
        validator.setProposalData(expectedId, approvedProposer, proposalType, true, 2, CYCLE_NUMBER);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyMovedToVote.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, PROPOSAL_DESCRIPTION);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_invalidVotingCycle_reverts() public {
        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidVotingCycle.selector);
        vm.warp(START_TIMESTAMP + DURATION + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, PROPOSAL_DESCRIPTION);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_proposalIdMismatch_reverts(uint256 fuzzedProposalId)
        public
    {
        vm.assume(fuzzedProposalId != expectedId);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (approvalVotingModule, votingModuleData, PROPOSAL_DESCRIPTION, APPROVAL_VOTING_MODULE_ID)
            ),
            abi.encode(uint256(fuzzedProposalId))
        );

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalIdMismatch.selector);
        vm.warp(START_TIMESTAMP + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, PROPOSAL_DESCRIPTION);
    }
}

contract ProposalValidator_MoveToVoteFundingProposal_Test is ProposalValidator_TestInit {
    ProposalValidator.ProposalType governanceFundProposalType = ProposalValidator.ProposalType.GovernanceFund;
    ProposalValidator.ProposalType councilBudgetProposalType = ProposalValidator.ProposalType.CouncilBudget;
    uint128 criteriaValue = 1;
    string governanceFundProposalDescription = "Test governance fund proposal";
    string councilBudgetProposalDescription = "Test council budget proposal";
    string[] optionsDescriptions = new string[](2);
    address[] optionsRecipients = new address[](2);
    uint256[] optionsAmounts = new uint256[](2);
    uint256 expectedGovernanceFundId;
    uint256 expectedCouncilBudgetId;
    bytes governanceFundVotingModuleData;
    bytes councilBudgetVotingModuleData;

    function setUp() public override {
        super.setUp();

        // Create option descriptions for the proposals
        optionsDescriptions[0] = "Option 1";
        optionsDescriptions[1] = "Option 2";

        // Create option recipients for the proposals
        optionsRecipients[0] = makeAddr("optionRecipient1");
        optionsRecipients[1] = makeAddr("optionRecipient2");

        // Create option amounts for the proposals
        optionsAmounts[0] = 100 ether;
        optionsAmounts[1] = 200 ether;

        // Create one proposal for each type
        (expectedGovernanceFundId, governanceFundVotingModuleData) = _createFundingProposalForMoveToVote(
            approvedProposer,
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            governanceFundProposalDescription,
            governanceFundProposalType
        );
        (expectedCouncilBudgetId, councilBudgetVotingModuleData) = _createFundingProposalForMoveToVote(
            approvedProposer,
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            councilBudgetProposalDescription,
            councilBudgetProposalType
        );
    }

    function test_moveToVoteFundingProposal_governanceFund_succeeds() public {
        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (
                    approvalVotingModule,
                    governanceFundVotingModuleData,
                    governanceFundProposalDescription,
                    APPROVAL_VOTING_MODULE_ID
                )
            ),
            abi.encode(uint256(expectedGovernanceFundId))
        );

        // Expect the ProposalMovedToVote event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalMovedToVote(expectedGovernanceFundId, approvedProposer);

        // Move to vote
        vm.warp(START_TIMESTAMP + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            governanceFundProposalDescription,
            governanceFundProposalType
        );

        // Check that the proposal is in voting
        (,, bool movedToVote,,) = validator.getProposalData(expectedGovernanceFundId);
        assertTrue(movedToVote, "Proposal should be in voting");
    }

    function test_moveToVoteFundingProposal_councilBudget_succeeds() public {
        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (
                    approvalVotingModule,
                    councilBudgetVotingModuleData,
                    councilBudgetProposalDescription,
                    APPROVAL_VOTING_MODULE_ID
                )
            ),
            abi.encode(uint256(expectedCouncilBudgetId))
        );

        // Expect the ProposalMovedToVote event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalMovedToVote(expectedCouncilBudgetId, approvedProposer);

        // Move to vote
        vm.warp(START_TIMESTAMP + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            councilBudgetProposalDescription,
            councilBudgetProposalType
        );
    }

    function test_moveToVoteFundingProposal_invalidFundingProposalType_reverts(
        uint8 fuzzedProposalTypeValue,
        string memory fuzzedProposalDescription
    )
        public
    {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 2));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidFundingProposalType.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            fuzzedProposalDescription,
            proposalType
        );
    }

    function test_moveToVoteFundingProposal_invalidProposal_reverts(
        uint8 fuzzedProposalTypeValue,
        uint128 fuzzedCriteriaValue
    )
        public
    {
        // Ensure the criteria value is not the same as the one in setUp so when calculating the proposal ID it will
        // not find the proposal
        vm.assume(fuzzedCriteriaValue != criteriaValue);

        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        string memory fundingProposalDescription;
        if (proposalType == governanceFundProposalType) {
            fundingProposalDescription = governanceFundProposalDescription;
        } else {
            fundingProposalDescription = councilBudgetProposalDescription;
        }

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidProposal.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            fuzzedCriteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            fundingProposalDescription,
            proposalType
        );
    }

    function test_moveToVoteFundingProposal_invalidProposalWrongProposalType_reverts(
        uint8 fuzzedWrongProposalTypeValue,
        uint8 fuzzedValidProposalTypeValue
    )
        public
    {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        fuzzedWrongProposalTypeValue = uint8(bound(fuzzedWrongProposalTypeValue, 0, 2));
        ProposalValidator.ProposalType wrongProposalType = ProposalValidator.ProposalType(fuzzedWrongProposalTypeValue);

        fuzzedValidProposalTypeValue = uint8(bound(fuzzedValidProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType validProposalType = ProposalValidator.ProposalType(fuzzedValidProposalTypeValue);

        string memory fundingProposalDescription;
        if (validProposalType == governanceFundProposalType) {
            // Set proposal data proposal type to a different value
            validator.setProposalData(
                expectedGovernanceFundId, approvedProposer, wrongProposalType, false, 0, CYCLE_NUMBER
            );
            fundingProposalDescription = governanceFundProposalDescription;
        } else {
            // Set proposal data proposal type to a different value
            validator.setProposalData(
                expectedCouncilBudgetId, approvedProposer, wrongProposalType, false, 0, CYCLE_NUMBER
            );
            fundingProposalDescription = councilBudgetProposalDescription;
        }

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidProposal.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            fundingProposalDescription,
            validProposalType
        );
    }

    function test_moveToVoteFundingProposal_invalidOptionsLength_reverts(uint8 fuzzedProposalTypeValue) public {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        string memory fundingProposalDescription;
        if (proposalType == governanceFundProposalType) {
            fundingProposalDescription = governanceFundProposalDescription;
        } else {
            fundingProposalDescription = councilBudgetProposalDescription;
        }

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidOptionsLength.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue, new string[](0), optionsRecipients, optionsAmounts, fundingProposalDescription, proposalType
        );

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidOptionsLength.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue,
            new string[](256),
            optionsRecipients,
            optionsAmounts,
            fundingProposalDescription,
            proposalType
        );
    }

    function test_moveToVoteFundingProposal_invalidTotalBudget_reverts(
        uint8 fuzzedProposalTypeValue,
        uint256 fuzzedAmount
    )
        public
    {
        fuzzedAmount = bound(fuzzedAmount, type(uint136).max, type(uint192).max);
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        string memory fundingProposalDescription;
        if (proposalType == governanceFundProposalType) {
            fundingProposalDescription = governanceFundProposalDescription;
        } else {
            fundingProposalDescription = councilBudgetProposalDescription;
        }

        vm.prank(owner);
        validator.setProposalDistributionThreshold(type(uint256).max);

        optionsAmounts[0] = fuzzedAmount;
        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidTotalBudget.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            fundingProposalDescription,
            proposalType
        );
    }

    function test_moveToVoteFundingProposal_insufficientApprovals_reverts(uint8 fuzzedProposalTypeValue) public {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        string memory fundingProposalDescription;
        if (proposalType == governanceFundProposalType) {
            // Set proposal data approved count to 0 since it is 1 by the approval on the setUp
            validator.setProposalData(expectedGovernanceFundId, approvedProposer, proposalType, false, 0, CYCLE_NUMBER);
            fundingProposalDescription = governanceFundProposalDescription;
        } else {
            // Set proposal data approved count to 0 since it is 1 by the approval on the setUp
            validator.setProposalData(expectedCouncilBudgetId, approvedProposer, proposalType, false, 0, CYCLE_NUMBER);
            fundingProposalDescription = councilBudgetProposalDescription;
        }

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InsufficientApprovals.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            fundingProposalDescription,
            proposalType
        );
    }

    function test_moveToVoteFundingProposal_proposalAlreadyMovedToVote_reverts(uint8 fuzzedProposalTypeValue) public {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        string memory fundingProposalDescription;
        if (proposalType == governanceFundProposalType) {
            // Set proposal data movedToVote to true
            validator.setProposalData(expectedGovernanceFundId, approvedProposer, proposalType, true, 1, CYCLE_NUMBER);
            fundingProposalDescription = governanceFundProposalDescription;
        } else {
            // Set proposal data movedToVote to true
            validator.setProposalData(expectedCouncilBudgetId, approvedProposer, proposalType, true, 1, CYCLE_NUMBER);
            fundingProposalDescription = councilBudgetProposalDescription;
        }

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyMovedToVote.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            fundingProposalDescription,
            proposalType
        );
    }

    function test_moveToVoteFundingProposal_invalidVotingCycle_reverts(uint8 fuzzedProposalTypeValue) public {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        string memory fundingProposalDescription;
        if (proposalType == governanceFundProposalType) {
            fundingProposalDescription = governanceFundProposalDescription;
        } else {
            fundingProposalDescription = councilBudgetProposalDescription;
        }

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidVotingCycle.selector);
        vm.warp(START_TIMESTAMP + DURATION + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            fundingProposalDescription,
            proposalType
        );
    }

    function test_moveToVoteFundingProposal_buildApprovalModuleOptionsExceedsProposalDistributionThreshold_reverts(
        uint8 fuzzedProposalTypeValue
    )
        public
    {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        string memory fundingProposalDescription;
        if (proposalType == governanceFundProposalType) {
            fundingProposalDescription = governanceFundProposalDescription;
        } else {
            fundingProposalDescription = councilBudgetProposalDescription;
        }

        // Set the first option amount to exceed the distribution threshold
        optionsAmounts[0] = PROPOSAL_DISTRIBUTION_THRESHOLD + 1;

        vm.expectRevert(IProposalValidator.ProposalValidator_ExceedsDistributionThreshold.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            fundingProposalDescription,
            proposalType
        );
    }

    function test_moveToVoteFundingProposal_exceedsDistributionThreshold_reverts(uint8 fuzzedProposalTypeValue)
        public
    {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        string memory fundingProposalDescription;
        if (proposalType == governanceFundProposalType) {
            fundingProposalDescription = governanceFundProposalDescription;
        } else {
            fundingProposalDescription = councilBudgetProposalDescription;
        }

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        string[] memory _optionsDescriptions = new string[](3);
        address[] memory _optionsRecipients = new address[](3);
        uint256[] memory _optionsAmounts = new uint256[](3);

        _optionsDescriptions[0] = "Option 1";
        _optionsDescriptions[1] = "Option 2";
        _optionsDescriptions[2] = "Option 3";

        _optionsRecipients[0] = makeAddr("optionRecipient1");
        _optionsRecipients[1] = makeAddr("optionRecipient2");
        _optionsRecipients[2] = makeAddr("optionRecipient3");

        _optionsAmounts[0] = PROPOSAL_DISTRIBUTION_THRESHOLD - 1;
        _optionsAmounts[1] = PROPOSAL_DISTRIBUTION_THRESHOLD - 1;
        _optionsAmounts[2] = PROPOSAL_DISTRIBUTION_THRESHOLD - 1;

        _createFundingProposalForMoveToVote(
            approvedProposer,
            criteriaValue,
            _optionsDescriptions,
            _optionsRecipients,
            _optionsAmounts,
            fundingProposalDescription,
            proposalType
        );

        vm.expectRevert(IProposalValidator.ProposalValidator_ExceedsDistributionThreshold.selector);
        vm.prank(approvedProposer);
        vm.warp(START_TIMESTAMP + 1);
        validator.moveToVoteFundingProposal(
            criteriaValue,
            _optionsDescriptions,
            _optionsRecipients,
            _optionsAmounts,
            fundingProposalDescription,
            proposalType
        );
    }

    function test_moveToVoteFundingProposal_proposalIdMismatch_reverts(
        uint8 fuzzedProposalTypeValue,
        uint256 fuzzedProposalId
    )
        public
    {
        vm.assume(fuzzedProposalId != expectedGovernanceFundId && fuzzedProposalId != expectedCouncilBudgetId);

        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        bytes memory votingModuleData;
        string memory fundingProposalDescription;
        if (proposalType == governanceFundProposalType) {
            votingModuleData = governanceFundVotingModuleData;
            fundingProposalDescription = governanceFundProposalDescription;
        } else {
            votingModuleData = councilBudgetVotingModuleData;
            fundingProposalDescription = councilBudgetProposalDescription;
        }

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (approvalVotingModule, votingModuleData, fundingProposalDescription, APPROVAL_VOTING_MODULE_ID)
            ),
            abi.encode(uint256(fuzzedProposalId))
        );

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalIdMismatch.selector);
        vm.warp(START_TIMESTAMP + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            fundingProposalDescription,
            proposalType
        );
    }
}

/// @title ProposalValidator_Setters_Test
/// @notice Tests for setter functions
contract ProposalValidator_SetVotingCycleData_Test is ProposalValidator_TestInit {
    function testFuzz_setVotingCycleData_succeeds(
        uint256 fuzzedCycleNumber,
        uint256 fuzzedStartingTimestamp,
        uint256 fuzzedDuration,
        uint256 fuzzedDistributionLimit
    )
        public
    {
        vm.assume(fuzzedCycleNumber != CYCLE_NUMBER); // Avoid existing cycle

        // Expect the VotingCycleDataSet event to be emitted
        vm.expectEmit(address(validator));
        emit VotingCycleDataSet(fuzzedCycleNumber, fuzzedStartingTimestamp, fuzzedDuration, fuzzedDistributionLimit);

        vm.prank(owner);
        validator.setVotingCycleData(
            fuzzedCycleNumber, fuzzedStartingTimestamp, fuzzedDuration, fuzzedDistributionLimit
        );

        (
            uint256 actualStartingTimestamp,
            uint256 actualDuration,
            uint256 actualDistributionLimit,
            uint256 actualMovedToVoteTokenCount
        ) = validator.votingCycles(fuzzedCycleNumber);

        assertEq(actualStartingTimestamp, fuzzedStartingTimestamp);
        assertEq(actualDuration, fuzzedDuration);
        assertEq(actualDistributionLimit, fuzzedDistributionLimit);
        assertEq(actualMovedToVoteTokenCount, 0);
    }

    function testFuzz_setVotingCycleData_notOwner_reverts(
        address fuzzedCaller,
        uint256 fuzzedCycleNumber,
        uint256 fuzzedStartingTimestamp,
        uint256 fuzzedDuration,
        uint256 fuzzedDistributionLimit
    )
        public
    {
        vm.assume(fuzzedCaller != owner);

        vm.prank(fuzzedCaller);
        vm.expectRevert("Ownable: caller is not the owner");
        validator.setVotingCycleData(
            fuzzedCycleNumber, fuzzedStartingTimestamp, fuzzedDuration, fuzzedDistributionLimit
        );
    }

    function testFuzz_setVotingCycleData_votingCycleAlreadySet_reverts(
        uint256 fuzzedStartingTimestamp,
        uint256 fuzzedDuration,
        uint256 fuzzedDistributionLimit
    )
        public
    {
        vm.expectRevert(ProposalValidator.ProposalValidator_VotingCycleAlreadySet.selector);
        vm.prank(owner);
        validator.setVotingCycleData(CYCLE_NUMBER, fuzzedStartingTimestamp, fuzzedDuration, fuzzedDistributionLimit);
    }
}

/// @title ProposalValidator_SetProposalDistributionThreshold_Test
/// @notice Tests for the setProposalDistributionThreshold function
contract ProposalValidator_SetProposalDistributionThreshold_Test is ProposalValidator_TestInit {
    function testFuzz_setProposalDistributionThreshold_succeeds(uint256 fuzzedNewProposalDistributionThreshold)
        public
    {
        // Expect the ProposalDistributionThresholdSet event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalDistributionThresholdSet(fuzzedNewProposalDistributionThreshold);

        vm.prank(owner);
        validator.setProposalDistributionThreshold(fuzzedNewProposalDistributionThreshold);

        assertEq(validator.proposalDistributionThreshold(), fuzzedNewProposalDistributionThreshold);
    }

    function testFuzz_setProposalDistributionThreshold_notOwner_reverts(
        address fuzzedCaller,
        uint256 fuzzedThreshold
    )
        public
    {
        vm.assume(fuzzedCaller != owner);

        vm.prank(fuzzedCaller);
        vm.expectRevert("Ownable: caller is not the owner");
        validator.setProposalDistributionThreshold(fuzzedThreshold);
    }
}

/// @title ProposalValidator_SetProposalTypeData_Test
/// @notice Tests for the setProposalTypeData function
contract ProposalValidator_SetProposalTypeData_Test is ProposalValidator_TestInit {
    function testFuzz_setProposalTypeData_succeeds(
        uint8 fuzzedProposalTypeValue,
        uint256 fuzzedNewRequiredApprovals,
        uint8 fuzzedNewProposalTypeId
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        fuzzedProposalTypeValue = uint8(bound(fuzzedProposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(fuzzedProposalTypeValue);

        ProposalValidator.ProposalTypeData memory newData = ProposalValidator.ProposalTypeData({
            requiredApprovals: fuzzedNewRequiredApprovals,
            idInConfigurator: fuzzedNewProposalTypeId
        });

        // Expect the ProposalTypeDataSet event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalTypeDataSet(proposalType, fuzzedNewRequiredApprovals, fuzzedNewProposalTypeId);

        vm.prank(owner);
        validator.setProposalTypeData(proposalType, newData);

        (uint256 requiredApprovals, uint8 idInConfigurator) = validator.proposalTypesData(proposalType);
        assertEq(requiredApprovals, fuzzedNewRequiredApprovals);
        assertEq(idInConfigurator, fuzzedNewProposalTypeId);
    }

    function testFuzz_setProposalTypeData_notOwner_reverts(address fuzzedCaller) public {
        vm.assume(fuzzedCaller != owner);

        ProposalValidator.ProposalTypeData memory newData =
            ProposalValidator.ProposalTypeData({ requiredApprovals: 4, idInConfigurator: 0 });

        vm.prank(fuzzedCaller);
        vm.expectRevert("Ownable: caller is not the owner");
        validator.setProposalTypeData(ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade, newData);
    }
}

/// @title ProposalValidator_Uncategorized_Test
/// @notice Tests for the `_hashProposalWithModule` function that is not part of the public interface
/// @dev This internal function is only exposed through the ProposalValidatorForTest contract
contract ProposalValidator_Uncategorized_Test is ProposalValidator_TestInit {
    function testFuzz_hashProposalWithModule_succeeds(
        address fuzzedModule,
        bytes memory fuzzedProposalData,
        bytes32 fuzzedDescriptionHash
    )
        public
        view
    {
        uint256 id = validator.hashProposalWithModule(fuzzedModule, fuzzedProposalData, fuzzedDescriptionHash);
        uint256 expectedId = uint256(
            keccak256(
                abi.encode(address(validator.GOVERNOR()), fuzzedModule, fuzzedProposalData, fuzzedDescriptionHash)
            )
        );

        assertEq(id, expectedId);
    }

    function test_hashProposalWithModule_differentInputs_succeeds() public {
        address module1 = makeAddr("module1");
        address module2 = makeAddr("module2");
        bytes memory data = abi.encode("data");
        bytes32 descHash = keccak256("desc");

        uint256 id1 = validator.hashProposalWithModule(module1, data, descHash);
        uint256 id2 = validator.hashProposalWithModule(module2, data, descHash);

        assertTrue(id1 != id2);
    }
}
