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
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { IApprovalVotingModule } from "interfaces/governance/IApprovalVotingModule.sol";
import { IOptimisticModule } from "interfaces/governance/IOptimisticModule.sol";

// Testing utilities
import { stdStorage, StdStorage } from "forge-std/Test.sol";

// Contracts
import { ProposalValidator } from "src/governance/ProposalValidator.sol";
import { Proxy } from "src/universal/Proxy.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Testing utilities
import { stdStorage, StdStorage } from "forge-std/Test.sol";
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title ProposalValidatorForTest
/// @notice A test contract that exposes the private _hashProposalWithModule function
contract ProposalValidatorForTest is ProposalValidator {
    constructor(IOptimismGovernor _governor) ProposalValidator(_governor) { }

    function hashProposalWithModule(
        address _module,
        bytes memory _proposalData,
        bytes32 _descriptionHash
    )
        public
        view
        returns (uint256)
    {
        return _hashProposalWithModule(_module, _proposalData, _descriptionHash);
    }

    /// @notice Exposes proposal data for testing
    function getProposalData(uint256 _proposalId)
        public
        view
        returns (
            address proposer_,
            ProposalType proposalType_,
            bool movedToVote_,
            uint256 approvalCount_,
            uint256 votingCycle_
        )
    {
        ProposalData storage proposal = _proposals[_proposalId];
        return (
            proposal.proposer, proposal.proposalType, proposal.movedToVote, proposal.approvalCount, proposal.votingCycle
        );
    }

    function setProposalData(
        uint256 _proposalId,
        address _proposer,
        ProposalType _proposalType,
        bool _movedToVote,
        uint256 _approvalCount,
        uint256 _votingCycle
    )
        public
    {
        _proposals[_proposalId].proposer = _proposer;
        _proposals[_proposalId].proposalType = _proposalType;
        _proposals[_proposalId].movedToVote = _movedToVote;
        _proposals[_proposalId].approvalCount = _approvalCount;
        _proposals[_proposalId].votingCycle = _votingCycle;
    }

    function mockApproveProposal(uint256 _proposalId, address _delegate) public {
        _proposals[_proposalId].delegateApprovals[_delegate] = true;
    }

    /// @notice Check if a delegate has approved a proposal
    function hasDelegateApproved(uint256 _proposalId, address _delegate) public view returns (bool hasApproved_) {
        return _proposals[_proposalId].delegateApprovals[_delegate];
    }
}

/// @title ProposalValidator_Init
/// @notice Setup contract for ProposalValidator tests
contract ProposalValidator_Init is CommonTest {
    using stdStorage for StdStorage;

    uint256 public constant CYCLE_NUMBER = 1;
    uint256 public constant START_TIMESTAMP = 1000000;
    uint256 public constant DURATION = 1 days;
    uint256 public constant DISTRIBUTION_LIMIT = 20000 ether;
    uint256 public constant DISTRIBUTION_THRESHOLD = 10000 ether;
    uint256 public constant PROPOSAL_REQUIRED_APPROVALS = 1;
    uint256 public constant MINIMUM_VOTING_POWER = 10000 ether;
    uint256 public constant OPTIMISTIC_MODULE_PERCENT_DIVISOR = 10_000;
    uint8 public constant APPROVAL_VOTING_MODULE_ID = 1;
    uint8 public constant OPTIMISTIC_VOTING_MODULE_ID = 2;
    uint64 public constant ATT_EXPIRATION_TIME = 10 days;
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
    ProposalValidatorForTest public impl;
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
        string[] memory descriptions,
        address[] memory recipients,
        uint256[] memory amounts,
        uint128 criteriaValue
    )
        internal
        pure
        returns (bytes memory)
    {
        // Construct ProposalOption array
        IApprovalVotingModule.ProposalOption[] memory options =
            new IApprovalVotingModule.ProposalOption[](descriptions.length);

        for (uint256 i = 0; i < descriptions.length; i++) {
            address[] memory targets = new address[](1);
            uint256[] memory values = new uint256[](1);
            bytes[] memory calldatas = new bytes[](1);

            targets[0] = Predeploys.GOVERNANCE_TOKEN;
            calldatas[0] = abi.encodeCall(IERC20.transfer, (recipients[i], amounts[i]));

            options[i] = IApprovalVotingModule.ProposalOption({
                budgetTokensSpent: amounts[i],
                targets: targets,
                values: values,
                calldatas: calldatas,
                description: descriptions[i]
            });
        }

        // Calculate total budget
        uint256 totalBudget = 0;
        for (uint256 i = 0; i < amounts.length; i++) {
            totalBudget += amounts[i];
        }

        // Construct ProposalSettings
        IApprovalVotingModule.ProposalSettings memory approvalSettings = IApprovalVotingModule.ProposalSettings({
            maxApprovals: uint8(descriptions.length),
            criteria: uint8(IApprovalVotingModule.PassingCriteria.Threshold),
            budgetToken: Predeploys.GOVERNANCE_TOKEN,
            criteriaValue: criteriaValue,
            budgetAmount: uint128(totalBudget)
        });

        return abi.encode(options, approvalSettings);
    }

    /// @notice Helper function to construct voting module data for council elections
    function _constructCouncilElectionVotingModuleData(
        string[] memory descriptions,
        uint128 criteriaValue
    )
        internal
        pure
        returns (bytes memory)
    {
        // Construct ProposalOption array for elections (no execution calls)
        IApprovalVotingModule.ProposalOption[] memory options =
            new IApprovalVotingModule.ProposalOption[](descriptions.length);

        for (uint256 i = 0; i < descriptions.length; i++) {
            address[] memory targets = new address[](0);
            uint256[] memory values = new uint256[](0);
            bytes[] memory calldatas = new bytes[](0);

            options[i] = IApprovalVotingModule.ProposalOption({
                budgetTokensSpent: 0,
                targets: targets,
                values: values,
                calldatas: calldatas,
                description: descriptions[i]
            });
        }

        // Construct ProposalSettings with TopChoices criteria
        IApprovalVotingModule.ProposalSettings memory approvalSettings = IApprovalVotingModule.ProposalSettings({
            maxApprovals: uint8(descriptions.length),
            criteria: uint8(IApprovalVotingModule.PassingCriteria.TopChoices),
            budgetToken: address(0),
            criteriaValue: criteriaValue,
            budgetAmount: 0
        });

        return abi.encode(options, approvalSettings);
    }

    /// @notice Helper function to construct voting module data for upgrade proposals
    function _constructOptimisticVotingModuleData(uint248 againstThreshold) internal pure returns (bytes memory) {
        IOptimisticModule.ProposalSettings memory optimisticSettings =
            IOptimisticModule.ProposalSettings({ againstThreshold: againstThreshold, isRelativeToVotableSupply: true });

        return abi.encode(optimisticSettings);
    }

    /// @notice Helper function to create a proposal for move to vote
    function _createUpgradeProposalForMoveToVote(
        address proposer,
        uint248 againstThreshold,
        string memory proposalDescription
    )
        internal
        returns (uint256 proposalId_, bytes memory votingModuleData_)
    {
        // Calculate expected proposal ID
        votingModuleData_ = _constructOptimisticVotingModuleData(againstThreshold);
        proposalId_ = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData_, keccak256(bytes(proposalDescription))
        );

        // 1 vote as default for being able to move to vote
        validator.setProposalData(
            proposalId_,
            proposer,
            ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade,
            false,
            PROPOSAL_REQUIRED_APPROVALS,
            CYCLE_NUMBER
        );
    }

    /// @notice Helper function to create a proposal for move to vote for council elections
    function _createCouncilElectionProposalForMoveToVote(
        address proposer,
        uint128 criteriaValue,
        string[] memory optionsDescriptions,
        string memory proposalDescription
    )
        internal
        returns (uint256 proposalId_, bytes memory votingModuleData_)
    {
        votingModuleData_ = _constructCouncilElectionVotingModuleData(optionsDescriptions, criteriaValue);
        proposalId_ = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData_, keccak256(bytes(proposalDescription))
        );

        validator.setProposalData(
            proposalId_,
            proposer,
            ProposalValidator.ProposalType.CouncilMemberElections,
            false,
            PROPOSAL_REQUIRED_APPROVALS,
            CYCLE_NUMBER
        );
    }

    /// @notice Helper function to create a proposal for move to vote for a funding proposal type
    function _createFundingProposalForMoveToVote(
        address proposer,
        uint128 criteriaValue,
        string[] memory optionsDescriptions,
        address[] memory optionsRecipients,
        uint256[] memory optionsAmounts,
        string memory proposalDescription,
        ProposalValidator.ProposalType proposalType
    )
        internal
        returns (uint256 proposalId_, bytes memory votingModuleData_)
    {
        votingModuleData_ =
            _constructFundingVotingModuleData(optionsDescriptions, optionsRecipients, optionsAmounts, criteriaValue);
        proposalId_ = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData_, keccak256(bytes(proposalDescription))
        );

        validator.setProposalData(proposalId_, proposer, proposalType, false, PROPOSAL_REQUIRED_APPROVALS, CYCLE_NUMBER);
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

        impl = new ProposalValidatorForTest(governor);
        validator = ProposalValidatorForTest(address(new Proxy(owner)));

        vm.prank(owner);
        IProxy(payable(address(validator))).upgradeToAndCall(
            address(impl),
            abi.encodeCall(
                impl.initialize,
                (
                    owner,
                    CYCLE_NUMBER,
                    START_TIMESTAMP,
                    DURATION,
                    DISTRIBUTION_LIMIT,
                    DISTRIBUTION_THRESHOLD,
                    APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID,
                    TOP_DELEGATES_ATTESTATION_SCHEMA_UID,
                    proposalTypes,
                    proposalTypesData
                )
            )
        );
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

/// @title ProposalValidator_Version_Test
/// @notice Tests for the version function
contract ProposalValidator_Version_Test is ProposalValidator_Init {
    function test_version_succeeds() public view {
        string memory versionString = validator.version();
        assertEq(versionString, "1.0.0");
    }
}

/// @title ProposalValidator_Initialize_Test
/// @notice Tests for the initialize function
contract ProposalValidator_Initialize_Test is ProposalValidator_Init {
    /// @dev Override to create validator proxy without initialization for testing
    function _initializeValidator() internal override {
        // Create mock addresses
        proposalTypesConfigurator = IProposalTypesConfigurator(makeAddr("proposalTypesConfigurator"));

        impl = new ProposalValidatorForTest(governor);
        validator = ProposalValidatorForTest(address(new Proxy(owner)));
    }

    function test_initialize_succeeds() public {
        (
            ProposalValidator.ProposalType[] memory proposalTypes,
            ProposalValidator.ProposalTypeData[] memory proposalTypesData
        ) = _getProposalTypesAndData();

        vm.prank(owner);
        IProxy(payable(address(validator))).upgradeToAndCall(
            address(impl),
            abi.encodeCall(
                impl.initialize,
                (
                    owner,
                    CYCLE_NUMBER,
                    START_TIMESTAMP,
                    DURATION,
                    DISTRIBUTION_LIMIT,
                    DISTRIBUTION_THRESHOLD,
                    APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID,
                    TOP_DELEGATES_ATTESTATION_SCHEMA_UID,
                    proposalTypes,
                    proposalTypesData
                )
            )
        );

        // Verify initialization was successful
        assertEq(validator.proposalDistributionThreshold(), DISTRIBUTION_THRESHOLD);
        assertEq(validator.owner(), owner);

        // Verify voting cycle data
        (uint256 startingTimestamp, uint256 duration, uint256 distributionLimit, uint256 movedToVoteTokenCount) =
            validator.votingCycles(CYCLE_NUMBER);
        assertEq(startingTimestamp, START_TIMESTAMP);
        assertEq(duration, DURATION);
        assertEq(distributionLimit, DISTRIBUTION_LIMIT);
        assertEq(movedToVoteTokenCount, 0);

        // Verify proposal type data
        for (uint256 i = 0; i < proposalTypes.length; i++) {
            (uint256 requiredApprovals, uint8 idInConfigurator) = validator.proposalTypesData(proposalTypes[i]);
            if (proposalTypes[i] == ProposalValidator.ProposalType.MaintenanceUpgrade) {
                assertEq(requiredApprovals, 0);
            } else {
                assertEq(requiredApprovals, PROPOSAL_REQUIRED_APPROVALS);
            }

            // GovernanceFund, CouncilBudget, and CouncilMemberElections use APPROVAL_VOTING_MODULE_ID
            if (
                proposalTypes[i] == ProposalValidator.ProposalType.GovernanceFund
                    || proposalTypes[i] == ProposalValidator.ProposalType.CouncilBudget
                    || proposalTypes[i] == ProposalValidator.ProposalType.CouncilMemberElections
            ) {
                assertEq(idInConfigurator, APPROVAL_VOTING_MODULE_ID);
            } else {
                // ProtocolOrGovernorUpgrade and MaintenanceUpgrade use OPTIMISTIC_VOTING_MODULE_ID
                assertEq(idInConfigurator, OPTIMISTIC_VOTING_MODULE_ID);
            }
        }
    }

    function test_initialize_mismatchedArrayLengths_reverts() public {
        ProposalValidator.ProposalType[] memory proposalTypes = new ProposalValidator.ProposalType[](3);
        proposalTypes[0] = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        proposalTypes[1] = ProposalValidator.ProposalType.MaintenanceUpgrade;
        proposalTypes[2] = ProposalValidator.ProposalType.CouncilMemberElections;

        // Create mismatched array with different length
        ProposalValidator.ProposalTypeData[] memory proposalTypesData = new ProposalValidator.ProposalTypeData[](2);
        proposalTypesData[0] =
            ProposalValidator.ProposalTypeData({ requiredApprovals: PROPOSAL_REQUIRED_APPROVALS, idInConfigurator: 0 });
        proposalTypesData[1] =
            ProposalValidator.ProposalTypeData({ requiredApprovals: PROPOSAL_REQUIRED_APPROVALS, idInConfigurator: 1 });

        vm.prank(owner);
        vm.expectRevert("Proxy: delegatecall to new implementation contract failed");
        IProxy(payable(address(validator))).upgradeToAndCall(
            address(impl),
            abi.encodeCall(
                impl.initialize,
                (
                    owner,
                    CYCLE_NUMBER,
                    START_TIMESTAMP,
                    DURATION,
                    DISTRIBUTION_LIMIT,
                    DISTRIBUTION_THRESHOLD,
                    APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID,
                    TOP_DELEGATES_ATTESTATION_SCHEMA_UID,
                    proposalTypes,
                    proposalTypesData
                )
            )
        );
    }
}

/// @title ProposalValidator_SubmitUpgradeProposal_Test
/// @notice Happy path tests for submitUpgradeProposal function
contract ProposalValidator_SubmitUpgradeProposal_Test is ProposalValidator_Init {
    string proposalDescription;

    function setUp() public override {
        super.setUp();

        _setProtocolOrGovernorUpgradeProposalType();
        _setMaintenanceUpgradeProposalType();

        proposalDescription = "Protocol Upgrade Proposal";
    }

    function testFuzz_submitUpgradeProposal_maintenanceUpgrade_succeeds(
        uint248 againstThreshold,
        address proposer
    )
        public
    {
        // Assume proposer is not zero address
        vm.assume(proposer != address(0));

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.MaintenanceUpgrade;

        // Bound againstThreshold to valid range (1 to 10000 basis points)
        againstThreshold = uint248(bound(againstThreshold, 1, OPTIMISTIC_MODULE_PERCENT_DIVISOR));

        // Create attestation for the proposal
        bytes32 attestationUid = _createApprovedProposerAttestation(proposer, proposalType);

        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(againstThreshold);
        uint256 expectedId = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
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
                (optimisticVotingModule, votingModuleData, proposalDescription, OPTIMISTIC_VOTING_MODULE_ID)
            ),
            abi.encode(expectedId)
        );

        // For MaintenanceUpgrade, events are: ProposalSubmitted, ProposalVotingModuleData, ProposalMovedToVote
        vm.expectEmit(address(validator));
        emit ProposalSubmitted(expectedId, proposer, proposalDescription, proposalType);

        vm.expectEmit(address(validator));
        emit ProposalVotingModuleData(expectedId, votingModuleData);

        vm.expectEmit(address(validator));
        emit ProposalMovedToVote(expectedId, proposer);

        vm.prank(proposer);
        uint256 proposalId = validator.submitUpgradeProposal(
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
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

        assertEq(storedProposer, proposer, "Proposer should match input");
        assertEq(uint8(storedProposalType), uint8(proposalType), "Proposal type should match input");
        assertTrue(movedToVote, "MaintenanceUpgrade should be in voting immediately");
        assertEq(approvalCount, 0, "Approval count should be 0");
        assertEq(votingCycle, CYCLE_NUMBER, "Voting cycle should match input");
    }

    function testFuzz_submitUpgradeProposal_protocolOrGovernorUpgrade_succeeds(
        uint248 againstThreshold,
        address proposer
    )
        public
    {
        // Assume proposer is not zero address
        vm.assume(proposer != address(0));

        // Bound againstThreshold to valid range (1 to 10000 basis points)
        againstThreshold = uint248(bound(againstThreshold, 1, OPTIMISTIC_MODULE_PERCENT_DIVISOR));

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;

        // Create attestation for the proposal
        bytes32 attestationUid = _createApprovedProposerAttestation(proposer, proposalType);

        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(againstThreshold);
        uint256 expectedId = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
        );

        // Mock proposalSnapshot to return 0 (proposal doesn't exist in governor)
        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // For ProtocolOrGovernorUpgrade, only ProposalSubmitted and ProposalVotingModuleData events
        vm.expectEmit(address(validator));
        emit ProposalSubmitted(expectedId, proposer, proposalDescription, proposalType);

        vm.expectEmit(address(validator));
        emit ProposalVotingModuleData(expectedId, votingModuleData);

        vm.prank(proposer);
        uint256 proposalId = validator.submitUpgradeProposal(
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
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

        assertEq(storedProposer, proposer, "Proposer should match input");
        assertEq(uint8(storedProposalType), uint8(proposalType), "Proposal type should match input");
        assertFalse(movedToVote, "ProtocolOrGovernorUpgrade should not be in voting yet");
        assertEq(approvalCount, 0, "Approval count should be 0");
        assertEq(votingCycle, CYCLE_NUMBER, "Voting cycle should match input");
    }
}

/// @title ProposalValidator_SubmitUpgradeProposal_TestFail
/// @notice Sad path tests for submitUpgradeProposal function
contract ProposalValidator_SubmitUpgradeProposal_TestFail is ProposalValidator_Init {
    string proposalDescription;
    uint248 againstThreshold = 5000; // 50%

    function setUp() public override {
        super.setUp();

        _setProtocolOrGovernorUpgradeProposalType();
        _setMaintenanceUpgradeProposalType();

        proposalDescription = "Test upgrade proposal";
    }

    function testFuzz_submitUpgradeProposal_invalidProposalType_reverts(uint8 proposalTypeValue) public {
        // Valid upgrade proposal types are ProtocolOrGovernorUpgrade (0) and MaintenanceUpgrade (1)
        proposalTypeValue = uint8(bound(proposalTypeValue, 2, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidUpgradeProposalType.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitUpgradeProposal_invalidVotingCycle_reverts(
        uint8 proposalTypeValue,
        uint256 votingCycle
    )
        public
    {
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 1));
        vm.assume(votingCycle != CYCLE_NUMBER);
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidVotingCycle.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            againstThreshold, proposalDescription, attestationUid, proposalType, votingCycle
        );
    }

    function testFuzz_submitUpgradeProposal_invalidAttestation_reverts(bytes32 fuzzedAttestationUid) public {
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 validAttestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        vm.assume(fuzzedAttestationUid != validAttestationUid); // Ensure it's different from valid attestation

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            againstThreshold, proposalDescription, fuzzedAttestationUid, proposalType, CYCLE_NUMBER
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
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitUpgradeProposal_attestationExpired_reverts(uint8 proposalTypeValue) public {
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 1));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // warp the time to after the attestation expiration time
        vm.warp(block.timestamp + ATT_EXPIRATION_TIME + 1);
        vm.expectRevert(ProposalValidator.ProposalValidator_AttestationExpired.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function test_submitUpgradeProposal_zeroAgainstThreshold_reverts() public {
        uint248 zeroThreshold = 0;
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAgainstThreshold.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(zeroThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER);
    }

    function testFuzz_submitUpgradeProposal_exceedsMaxAgainstThreshold_reverts(uint248 excessiveThreshold) public {
        // Bound excessive threshold to be greater than OPTIMISTIC_MODULE_PERCENT_DIVISOR
        excessiveThreshold =
            uint248(bound(excessiveThreshold, OPTIMISTIC_MODULE_PERCENT_DIVISOR + 1, type(uint248).max));

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAgainstThreshold.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            excessiveThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
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
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitUpgradeProposal_duplicateProposal_reverts(uint8 proposalTypeValue) public {
        // Bound proposal type to only upgrade proposals (0 = ProtocolOrGovernorUpgrade, 1 = MaintenanceUpgrade)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 1));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(againstThreshold);
        uint256 expectedId = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
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
                    (optimisticVotingModule, votingModuleData, proposalDescription, OPTIMISTIC_VOTING_MODULE_ID)
                ),
                abi.encode(expectedId)
            );
        }

        // Submit first proposal
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
        );

        // Attempt to submit identical proposal should revert
        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalAlreadySubmitted.selector);

        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitUpgradeProposal_proposalExistsInGovernor_reverts(uint8 proposalTypeValue) public {
        // Bound proposal type to only upgrade proposals (0 = ProtocolOrGovernorUpgrade, 1 = MaintenanceUpgrade)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 1));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(againstThreshold);
        uint256 expectedId = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
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
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
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
            againstThreshold, proposalDescription, invalidAttestation, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitUpgradeProposal_attestationRevoked_reverts(uint8 proposalTypeValue) public {
        // Bound proposal type to only upgrade proposals (0 = ProtocolOrGovernorUpgrade, 1 = MaintenanceUpgrade)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 1));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

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
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function test_submitUpgradeProposal_proposalIdMismatch_reverts(uint256 proposalId) public {
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.MaintenanceUpgrade;
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(againstThreshold);
        uint256 expectedId = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
        );

        vm.assume(proposalId != expectedId); // Ensure proposalId is different from expectedId

        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        // Mock the proposeWithModule call to return a different proposalId
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (optimisticVotingModule, votingModuleData, proposalDescription, OPTIMISTIC_VOTING_MODULE_ID)
            ),
            abi.encode(proposalId)
        );

        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalIdMismatch.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
        );
    }
}

/// @title ProposalValidator_SubmitCouncilMemberElectionsProposal_Test
/// @notice Happy path tests for submitCouncilMemberElectionsProposal function
contract ProposalValidator_SubmitCouncilMemberElectionsProposal_Test is ProposalValidator_Init {
    string proposalDescription;

    function setUp() public override {
        super.setUp();

        _setCouncilMemberElectionsProposalType();

        proposalDescription = "Council Member Elections Q4 2024";
    }

    function testFuzz_submitCouncilMemberElectionsProposal_succeeds(uint8 optionCount, uint128 criteriaValue) public {
        optionCount = uint8(bound(optionCount, 2, 5)); // Minimum 2 options to have valid criteria < optionCount
        criteriaValue = uint128(bound(criteriaValue, 1, optionCount - 1)); // Must be less than optionCount

        // Create dynamic array of option descriptions based on option count
        string[] memory optionDescriptions = new string[](optionCount);
        for (uint256 i = 0; i < optionCount; i++) {
            optionDescriptions[i] = string(abi.encodePacked("Candidate ", vm.toString(i)));
        }

        // Create attestation for the proposal
        bytes32 attestationUid =
            _createApprovedProposerAttestation(topDelegate_A, ProposalValidator.ProposalType.CouncilMemberElections);

        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructCouncilElectionVotingModuleData(optionDescriptions, criteriaValue);
        uint256 expectedId = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
        );

        // Mock proposalSnapshot to return 0 (proposal doesn't exist in governor)
        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        // Expect ProposalSubmitted event
        vm.expectEmit(address(validator));
        emit ProposalSubmitted(
            expectedId, topDelegate_A, proposalDescription, ProposalValidator.ProposalType.CouncilMemberElections
        );

        // Expect ProposalVotingModuleData event
        vm.expectEmit(address(validator));
        emit ProposalVotingModuleData(expectedId, votingModuleData);

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.prank(topDelegate_A);
        uint256 proposalId = validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, attestationUid, CYCLE_NUMBER
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
}

/// @title ProposalValidator_SubmitCouncilMemberElectionsProposal_TestFail
/// @notice Sad path tests for submitCouncilMemberElectionsProposal function
contract ProposalValidator_SubmitCouncilMemberElectionsProposal_TestFail is ProposalValidator_Init {
    uint128 criteriaValue;
    string[] optionDescriptions;
    string proposalDescription;
    bytes32 attestationUid;

    function setUp() public override {
        super.setUp();

        _setCouncilMemberElectionsProposalType();

        criteriaValue = 2;
        optionDescriptions = new string[](3);
        optionDescriptions[0] = "Candidate A";
        optionDescriptions[1] = "Candidate B";
        optionDescriptions[2] = "Candidate C";

        proposalDescription = "Test Council Elections";
        attestationUid =
            _createApprovedProposerAttestation(topDelegate_A, ProposalValidator.ProposalType.CouncilMemberElections);
    }

    function testFuzz_submitCouncilMemberElectionsProposal_invalidVotingCycle_reverts(uint256 votingCycle) public {
        vm.assume(votingCycle != CYCLE_NUMBER);
        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidVotingCycle.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, attestationUid, votingCycle
        );
    }

    function testFuzz_submitCouncilMemberElectionsProposal_invalidAttestation_reverts(bytes32 fuzzedAttestationUid)
        public
    {
        vm.assume(fuzzedAttestationUid != attestationUid); // Ensure it's different from valid attestation

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, fuzzedAttestationUid, CYCLE_NUMBER
        );
    }

    function testFuzz_submitCouncilMemberElectionsProposal_unattestedProposer_reverts(address fuzzedProposer) public {
        vm.assume(fuzzedProposer != topDelegate_A); // Ensure it's different from attested proposer

        // Try to submit with different address than attested
        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(fuzzedProposer); // Different from attested topDelegate_A
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, attestationUid, CYCLE_NUMBER
        );
    }

    function testFuzz_submitCouncilMemberElectionsProposal_attestationExpired_reverts() public {
        // warp the time to after the attestation expiration time
        vm.warp(block.timestamp + ATT_EXPIRATION_TIME + 1);
        vm.expectRevert(ProposalValidator.ProposalValidator_AttestationExpired.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, attestationUid, CYCLE_NUMBER
        );
    }

    function test_submitCouncilMemberElectionsProposal_zeroOptions_reverts() public {
        string[] memory emptyOptions = new string[](0);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidOptionsLength.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, emptyOptions, proposalDescription, attestationUid, CYCLE_NUMBER
        );
    }

    function test_submitCouncilMemberElectionsProposal_duplicateProposal_reverts() public {
        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructCouncilElectionVotingModuleData(optionDescriptions, criteriaValue);
        uint256 expectedId = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
        );

        // Mock proposalSnapshot to return 0 for first submission
        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        // Submit first proposal
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, attestationUid, CYCLE_NUMBER
        );

        // Attempt to submit identical proposal should revert
        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalAlreadySubmitted.selector);

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, attestationUid, CYCLE_NUMBER
        );
    }

    function test_submitCouncilMemberElectionsProposal_proposalExistsInGovernor_reverts() public {
        // Calculate expected proposal ID
        bytes memory votingModuleData = _constructCouncilElectionVotingModuleData(optionDescriptions, criteriaValue);
        uint256 expectedId = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
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
            criteriaValue, optionDescriptions, proposalDescription, attestationUid, CYCLE_NUMBER
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
            criteriaValue, optionDescriptions, proposalDescription, invalidAttestation, CYCLE_NUMBER
        );
    }

    function testFuzz_submitCouncilMemberElectionsProposal_criteriaValueExceedsOptionsLength_reverts(
        uint128 invalidCriteriaValue
    )
        public
    {
        // Bound invalidCriteriaValue to be greater than options length
        invalidCriteriaValue = uint128(bound(invalidCriteriaValue, optionDescriptions.length + 1, type(uint128).max));

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidCriteriaValue.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            invalidCriteriaValue, optionDescriptions, proposalDescription, attestationUid, CYCLE_NUMBER
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
            criteriaValue, optionDescriptions, proposalDescription, revocableAttestationUid, CYCLE_NUMBER
        );
    }

    function test_submitCouncilMemberElectionsProposal_invalidVotingModule_reverts() public {
        attestationUid =
            _createApprovedProposerAttestation(topDelegate_A, ProposalValidator.ProposalType.CouncilMemberElections);

        // Mock configurator to return uninitialized module
        _mockProposalTypesConfiguratorCallWithUninitializedModule(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidVotingModule.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, attestationUid, CYCLE_NUMBER
        );
    }
}

/// @title ProposalValidator_SubmitFundingProposal_Test
/// @notice Happy path tests for submitFundingProposal function
contract ProposalValidator_SubmitFundingProposal_Test is ProposalValidator_Init {
    uint128 criteriaValue = 1000 ether;
    string description = "Test funding proposal";

    function setUp() public override {
        super.setUp();

        _setGovernanceFundProposalType();
        _setCouncilBudgetProposalType();
    }

    function testFuzz_submitFundingProposal_succeeds(
        uint8 proposalTypeValue,
        uint8 optionCount,
        uint256 amount,
        address proposer
    )
        public
    {
        // Assume proposer is not zero address
        vm.assume(proposer != address(0));
        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        // Bound option count between 1 and 5 for reasonable test execution
        optionCount = uint8(bound(optionCount, 1, 5));

        // Bound amount from 0 to DISTRIBUTION_THRESHOLD (inclusive)
        amount = bound(amount, 0, DISTRIBUTION_THRESHOLD);

        // Start with minimal arrays and extend based on option count
        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(optionCount);

        // fuzz the amounts
        for (uint256 i = 0; i < optionCount; i++) {
            amounts[i] = amount;
        }

        // Calculate expected proposal ID
        bytes memory votingModuleData =
            _constructFundingVotingModuleData(descriptions, recipients, amounts, criteriaValue);
        uint256 expectedId =
            validator.hashProposalWithModule(approvalVotingModule, votingModuleData, keccak256(bytes(description)));

        // Mock proposalSnapshot to return 0 (proposal doesn't exist in governor)
        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        // Expect ProposalSubmitted event
        vm.expectEmit(address(validator));
        emit ProposalSubmitted(expectedId, proposer, description, proposalType);

        // Expect ProposalVotingModuleData event
        vm.expectEmit(address(validator));
        emit ProposalVotingModuleData(expectedId, votingModuleData);

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.prank(proposer);
        uint256 proposalId = validator.submitFundingProposal(
            criteriaValue, descriptions, recipients, amounts, description, proposalType, CYCLE_NUMBER
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

        assertEq(storedProposer, proposer, "Proposer should match input");
        assertEq(uint8(storedProposalType), uint8(proposalType), "Proposal type should match input");
        assertFalse(movedToVote, "Proposal should not be in voting yet");
        assertEq(approvalCount, 0, "Approval count should be 0");
        assertEq(votingCycle, CYCLE_NUMBER, "Voting cycle should match input");
    }
}

/// @title ProposalValidator_SubmitFundingProposal_TestFail
/// @notice Sad path tests for submitFundingProposal function
contract ProposalValidator_SubmitFundingProposal_TestFail is ProposalValidator_Init {
    uint128 public constant FUNDING_CRITERIA_VALUE = 50;
    string description = "Test funding proposal";

    function setUp() public override {
        super.setUp();
        // Set both funding proposal types to use the approval voting module
        _setGovernanceFundProposalType();
        _setCouncilBudgetProposalType();
    }

    function testFuzz_submitFundingProposal_invalidProposalType_reverts(uint8 proposalTypeValue) public {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 2));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(1);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidFundingProposalType.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, description, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_invalidVotingCycle_reverts(
        uint8 proposalTypeValue,
        uint256 votingCycle
    )
        public
    {
        proposalTypeValue = uint8(bound(proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);
        vm.assume(votingCycle != CYCLE_NUMBER);

        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(1);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidVotingCycle.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, description, proposalType, votingCycle
        );
    }

    function testFuzz_submitFundingProposal_mismatchedDescriptionsLength_reverts(
        uint8 matchingLength,
        uint8 mismatchedLength,
        uint8 proposalTypeValue
    )
        public
    {
        // Bound lengths to reasonable values (1-50) and ensure they're different
        matchingLength = uint8(bound(matchingLength, 1, 50));
        mismatchedLength = uint8(bound(mismatchedLength, 1, 50));
        vm.assume(matchingLength != mismatchedLength);

        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

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
            description,
            proposalType,
            CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_mismatchedRecipientsLength_reverts(
        uint8 matchingLength,
        uint8 mismatchedLength,
        uint8 proposalTypeValue
    )
        public
    {
        // Bound lengths to reasonable values (1-50) and ensure they're different
        matchingLength = uint8(bound(matchingLength, 1, 50));
        mismatchedLength = uint8(bound(mismatchedLength, 1, 50));
        vm.assume(matchingLength != mismatchedLength);

        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

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
            description,
            proposalType,
            CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_mismatchedAmountsLength_reverts(
        uint8 matchingLength,
        uint8 mismatchedLength,
        uint8 proposalTypeValue
    )
        public
    {
        // Bound lengths to reasonable values (1-50) and ensure they're different
        matchingLength = uint8(bound(matchingLength, 1, 50));
        mismatchedLength = uint8(bound(mismatchedLength, 1, 50));
        vm.assume(matchingLength != mismatchedLength);

        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

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
            description,
            proposalType,
            CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_exceedsProposalDistributionThreshold_reverts(
        uint256 excessAmount,
        uint8 proposalTypeValue
    )
        public
    {
        // Bound excess amount to be greater than DISTRIBUTION_THRESHOLD
        excessAmount = bound(excessAmount, DISTRIBUTION_THRESHOLD + 1, type(uint128).max);

        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        // Create arrays with excessive amount
        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(1);
        amounts[0] = excessAmount;

        vm.expectRevert(ProposalValidator.ProposalValidator_ExceedsDistributionThreshold.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, description, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_duplicateProposal_reverts(uint8 proposalTypeValue) public {
        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(1);

        // Calculate expected proposal ID
        bytes memory votingModuleData =
            _constructFundingVotingModuleData(descriptions, recipients, amounts, FUNDING_CRITERIA_VALUE);
        uint256 expectedId =
            validator.hashProposalWithModule(approvalVotingModule, votingModuleData, keccak256(bytes(description)));

        // Mock proposalSnapshot to return 0 for first submission
        _mockAndExpect(
            address(governor), abi.encodeCall(IOptimismGovernor.proposalSnapshot, (expectedId)), abi.encode(0)
        );

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        // Submit first proposal
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, description, proposalType, CYCLE_NUMBER
        );

        // Attempt to submit identical proposal
        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalAlreadySubmitted.selector);

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, description, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_proposalExistsInGovernor_reverts(uint8 proposalTypeValue) public {
        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays(1);

        // Calculate expected proposal ID
        bytes memory votingModuleData =
            _constructFundingVotingModuleData(descriptions, recipients, amounts, FUNDING_CRITERIA_VALUE);
        uint256 expectedId =
            validator.hashProposalWithModule(approvalVotingModule, votingModuleData, keccak256(bytes(description)));

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
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, description, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_zeroOptionsLength_reverts(uint8 proposalTypeValue) public {
        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);
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
            description,
            proposalType,
            CYCLE_NUMBER
        );
    }

    function testFuzz_submitFundingProposal_exceedsMaxOptionsLength_reverts(
        uint256 tooManyOptions,
        uint8 proposalTypeValue
    )
        public
    {
        // Bound proposal type to only GovernanceFund (3) or CouncilBudget (4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);
        // Create arrays with more than 255 options (exceeds allowed uint8 max)
        tooManyOptions = uint256(bound(tooManyOptions, 256, 512));
        string[] memory tooManyDescriptions = new string[](tooManyOptions);
        address[] memory tooManyRecipients = new address[](tooManyOptions);
        uint256[] memory tooManyAmounts = new uint256[](tooManyOptions);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidOptionsLength.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE,
            tooManyDescriptions,
            tooManyRecipients,
            tooManyAmounts,
            description,
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
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, description, proposalType, CYCLE_NUMBER
        );
    }
}

/// @title ProposalValidator_ApproveProposal_Test
/// @notice Happy path tests for approveProposal function
contract ProposalValidator_ApproveProposal_Test is ProposalValidator_Init {
    function setUp() public override {
        super.setUp();

        // create a new voting cycle
        // cycle number decreased by 1 and start time CYCLE_DURATION before the current cycle
        vm.prank(owner);
        validator.setVotingCycleData(CYCLE_NUMBER - 1, START_TIMESTAMP - DURATION, DURATION, DISTRIBUTION_THRESHOLD);
    }

    function test_approveProposal_succeeds(uint256 _proposalId, uint8 proposalTypeValue) public {
        // Ensure the proposal ID is not 0
        vm.assume(_proposalId != 0);

        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        // Set mock proposal data of a random proposal in the validator contract
        validator.setProposalData(_proposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);

        // Expect event to be emitted when approving
        vm.expectEmit(address(validator));
        emit ProposalApproved(_proposalId, topDelegate_A);

        // Approve the proposal, use the attestation of the top delegate that was created in setUp
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalId, topDelegateAttestation_A);

        // Check that the proposal data has been updated
        assertTrue(validator.hasDelegateApproved(_proposalId, topDelegate_A));

        (,,, uint256 approvalCount,) = validator.getProposalData(_proposalId);
        assertEq(approvalCount, 1);
    }
}

/// @title ProposalValidator_ApproveProposal_TestFail
/// @notice Sad path tests for approveProposal function
contract ProposalValidator_ApproveProposal_TestFail is ProposalValidator_Init {
    function setUp() public override {
        super.setUp();
    }

    function test_approveProposal_proposalDoesNotExist_reverts(uint256 _proposalId) public {
        // There is no stored proposal data so this will revert
        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalDoesNotExist.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalId, topDelegateAttestation_A);
    }

    function test_approveProposal_proposalAlreadyApproved_reverts(
        uint256 _proposalId,
        uint8 proposalTypeValue
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);
        // set proposal data so that the proposal exists
        validator.setProposalData(_proposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);

        // Mock the proposal as already approved by the top delegate
        validator.mockApproveProposal(_proposalId, topDelegate_A);
        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyApproved.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalId, topDelegateAttestation_A);
    }

    function test_approveProposal_proposalAlreadyMovedToVote_reverts(
        uint256 _proposalId,
        uint8 proposalTypeValue
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);
        // set proposal data so that the proposal exists and set movedToVote to true
        validator.setProposalData(_proposalId, topDelegate_A, proposalType, true, 0, CYCLE_NUMBER);

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyMovedToVote.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalId, topDelegateAttestation_A);
    }

    function test_approveProposal_invalidVotingCycle_reverts(
        uint256 _proposalId,
        uint8 proposalTypeValue,
        uint256 votingCycle
    )
        public
    {
        vm.assume(votingCycle != CYCLE_NUMBER && votingCycle != 0);
        vm.assume(votingCycle != CYCLE_NUMBER + 1); // Avoid existing cycle

        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        // set proposal data so that the proposal exists
        validator.setProposalData(_proposalId, topDelegate_A, proposalType, false, 0, votingCycle);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidVotingCycle.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalId, topDelegateAttestation_A);
    }

    function test_approveProposal_invalidSchema_reverts(uint256 _proposalId, uint8 proposalTypeValue) public {
        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

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
        validator.setProposalData(_proposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);
        // set the voting cycle data of the previous cycle
        vm.prank(owner);
        validator.setVotingCycleData(CYCLE_NUMBER - 1, START_TIMESTAMP - DURATION, DURATION, DISTRIBUTION_THRESHOLD);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestationSchema.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalId, _invalidAttestationUid);
    }

    function test_approveProposal_attestationRevoked_reverts(uint256 _proposalId, uint8 proposalTypeValue) public {
        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);
        // set proposal data so that the proposal exists
        validator.setProposalData(_proposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);
        // set the voting cycle data of the previous cycle
        vm.prank(owner);
        validator.setVotingCycleData(CYCLE_NUMBER - 1, START_TIMESTAMP - DURATION, DURATION, DISTRIBUTION_THRESHOLD);

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
        validator.approveProposal(_proposalId, topDelegateAttestation_A);
    }

    function test_approveProposal_invalidAttestationCaller_reverts(
        uint256 _proposalId,
        uint8 proposalTypeValue,
        address _caller
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        // Ensure the caller is not a top delegate
        vm.assume(_caller != topDelegate_A);

        // Set mock proposal data of a random proposal in the validator contract
        validator.setProposalData(_proposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);
        // set the voting cycle data of the previous cycle
        vm.prank(owner);
        validator.setVotingCycleData(CYCLE_NUMBER - 1, START_TIMESTAMP - DURATION, DURATION, DISTRIBUTION_THRESHOLD);

        // Expect the invalid attestation error to be reverted
        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(_caller);
        validator.approveProposal(_proposalId, topDelegateAttestation_A);
    }

    function test_approveProposal_invalidAttestationPartialDelegation_reverts(
        uint256 _proposalId,
        uint8 proposalTypeValue
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        // Set mock proposal data of a random proposal in the validator contract
        validator.setProposalData(_proposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);
        // set the voting cycle data of the previous cycle
        vm.prank(owner);
        validator.setVotingCycleData(CYCLE_NUMBER - 1, START_TIMESTAMP - DURATION, DURATION, DISTRIBUTION_THRESHOLD);

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
        validator.approveProposal(_proposalId, _attestationUidWithPartialDelegation);
    }

    function test_approveProposal_nonExistentAttestation_reverts(
        uint256 _proposalId,
        uint8 proposalTypeValue,
        bytes32 _nonExistentAttestationUid
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        // Ensure the attestation uid is not one of the valid ones
        vm.assume(_nonExistentAttestationUid != topDelegateAttestation_A);

        // Set mock proposal data of a random proposal in the validator contract
        validator.setProposalData(_proposalId, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);
        // set the voting cycle data of the previous cycle
        vm.prank(owner);
        validator.setVotingCycleData(CYCLE_NUMBER - 1, START_TIMESTAMP - DURATION, DURATION, DISTRIBUTION_THRESHOLD);

        // Expect the invalid attestation error to be reverted when attestation doesn't exist
        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalId, _nonExistentAttestationUid);
    }
}

/// @title ProposalValidator_MoveToVoteProtocolOrGovernorUpgradeProposal_Test
/// @notice Happy path tests for moveToVoteProtocolOrGovernorUpgradeProposal function
contract ProposalValidator_MoveToVoteProtocolOrGovernorUpgradeProposal_Test is ProposalValidator_Init {
    uint248 againstThreshold = 5000; // 50%
    string proposalDescription = "Test proposal";
    ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
    bytes votingModuleData;
    uint256 expectedId;

    function setUp() public override {
        super.setUp();

        (expectedId, votingModuleData) =
            _createUpgradeProposalForMoveToVote(approvedProposer, againstThreshold, proposalDescription);
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_succeeds() public {
        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (optimisticVotingModule, votingModuleData, proposalDescription, OPTIMISTIC_VOTING_MODULE_ID)
            ),
            abi.encode(expectedId)
        );

        // Expect the ProposalMovedToVote event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalMovedToVote(expectedId, approvedProposer);

        // Move to vote
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(againstThreshold, proposalDescription);

        // Check that the proposal is in voting
        (,, bool movedToVote,,) = validator.getProposalData(expectedId);
        assertTrue(movedToVote, "Proposal should be in voting");
    }
}

contract ProposalValidator_MoveToVoteProtocolOrGovernorUpgradeProposal_TestFail is ProposalValidator_Init {
    uint248 againstThreshold = 5000; // 50%
    string proposalDescription = "Test proposal";
    ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
    bytes votingModuleData;
    uint256 expectedId;

    function setUp() public override {
        super.setUp();

        (expectedId, votingModuleData) =
            _createUpgradeProposalForMoveToVote(approvedProposer, againstThreshold, proposalDescription);
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_invalidProposer_reverts(address _caller) public {
        vm.assume(_caller != approvedProposer);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidProposer.selector);
        vm.prank(_caller);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(againstThreshold, proposalDescription);
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_invalidProposal_reverts(uint248 _againstThreshold)
        public
    {
        // This will generate a different proposal ID which will make the proposal type wrong
        vm.assume(_againstThreshold != againstThreshold);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidProposal.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(_againstThreshold, proposalDescription);
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_insufficientApprovals_reverts() public {
        // Set proposal data approved count to 0 since it is 1 by the approval on the setUp
        validator.setProposalData(expectedId, approvedProposer, proposalType, false, 0, CYCLE_NUMBER);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InsufficientApprovals.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(againstThreshold, proposalDescription);
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_proposalAlreadyMovedToVote_reverts() public {
        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // Set proposal data movedToVote to true
        validator.setProposalData(expectedId, approvedProposer, proposalType, true, 1, CYCLE_NUMBER);

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyMovedToVote.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(againstThreshold, proposalDescription);
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_proposalIdMismatch_reverts(uint256 _randomId) public {
        vm.assume(_randomId != expectedId);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (optimisticVotingModule, votingModuleData, proposalDescription, OPTIMISTIC_VOTING_MODULE_ID)
            ),
            abi.encode(uint256(_randomId))
        );

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalIdMismatch.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(againstThreshold, proposalDescription);
    }
}

contract ProposalValidator_MoveToVoteCouncilMemberElectionsProposal_Test is ProposalValidator_Init {
    ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.CouncilMemberElections;
    uint128 criteriaValue = 1;
    uint256 expectedId;
    bytes votingModuleData;
    string proposalDescription = "Test proposal";
    string[] optionsDescriptions = new string[](2);

    function setUp() public override {
        super.setUp();

        // Create a proposal for move to vote with 1 top choice and 2 options
        optionsDescriptions[0] = "Option 1";
        optionsDescriptions[1] = "Option 2";
        (expectedId, votingModuleData) = _createCouncilElectionProposalForMoveToVote(
            approvedProposer, criteriaValue, optionsDescriptions, proposalDescription
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
                (approvalVotingModule, votingModuleData, proposalDescription, APPROVAL_VOTING_MODULE_ID)
            ),
            abi.encode(expectedId)
        );

        // Expect the ProposalMovedToVote event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalMovedToVote(expectedId, approvedProposer);

        // Move to vote
        vm.warp(START_TIMESTAMP + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, proposalDescription);

        // Check that the proposal is in voting
        (,, bool movedToVote,,) = validator.getProposalData(expectedId);
        assertTrue(movedToVote, "Proposal should be in voting");
    }
}

contract ProposalValidator_MoveToVoteCouncilMemberElectionsProposal_TestFail is ProposalValidator_Init {
    ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.CouncilMemberElections;
    uint128 criteriaValue = 1;
    string proposalDescription = "Test proposal";
    string[] optionsDescriptions = new string[](2);
    uint256 expectedId;
    bytes votingModuleData;

    function setUp() public override {
        super.setUp();

        // Create a proposal for move to vote with 1 top choice and 2 options
        optionsDescriptions[0] = "Option 1";
        optionsDescriptions[1] = "Option 2";
        (expectedId, votingModuleData) = _createCouncilElectionProposalForMoveToVote(
            approvedProposer, criteriaValue, optionsDescriptions, proposalDescription
        );
    }

    function test_moveToVoteCouncilMemberElectionsProposal_invalidProposer_reverts(address _caller) public {
        vm.assume(_caller != approvedProposer);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidProposer.selector);
        vm.prank(_caller);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, proposalDescription);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_invalidProposal_reverts() public {
        // This will generate a different proposal ID which will make the proposal type wrong
        uint128 _criteriaValue = 2; // we use 2 since it is the max based on the created proposal in setUp

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidProposal.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(_criteriaValue, optionsDescriptions, proposalDescription);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_insufficientApprovals_reverts() public {
        // Set proposal data approved count to 0 since it is 1 by the approval on the setUp
        validator.setProposalData(expectedId, approvedProposer, proposalType, false, 0, CYCLE_NUMBER);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InsufficientApprovals.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, proposalDescription);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_proposalAlreadyMovedToVote_reverts() public {
        // Set proposal data movedToVote to true
        validator.setProposalData(expectedId, approvedProposer, proposalType, true, 2, CYCLE_NUMBER);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyMovedToVote.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, proposalDescription);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_invalidVotingCycle_reverts() public {
        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidVotingCycle.selector);
        vm.warp(START_TIMESTAMP + DURATION + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, proposalDescription);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_proposalIdMismatch_reverts(uint256 _randomId) public {
        vm.assume(_randomId != expectedId);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (approvalVotingModule, votingModuleData, proposalDescription, APPROVAL_VOTING_MODULE_ID)
            ),
            abi.encode(uint256(_randomId))
        );

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalIdMismatch.selector);
        vm.warp(START_TIMESTAMP + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, proposalDescription);
    }
}

contract ProposalValidator_MoveToVoteFundingProposal_Test is ProposalValidator_Init {
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
}

contract ProposalValidator_MoveToVoteFundingProposal_TestFail is ProposalValidator_Init {
    ProposalValidator.ProposalType governanceFundProposalType = ProposalValidator.ProposalType.GovernanceFund;
    ProposalValidator.ProposalType councilBudgetProposalType = ProposalValidator.ProposalType.CouncilBudget;
    uint128 criteriaValue = 1;
    string governanceFundProposalDescription = "Test governance fund proposal";
    string councilBudgetProposalDescription = "Test council budget proposal";
    string[] optionsDescriptions;
    address[] optionsRecipients;
    uint256[] optionsAmounts;
    uint256 governanceFundExpectedId;
    uint256 councilBudgetExpectedId;
    bytes governanceFundVotingModuleData;
    bytes councilBudgetVotingModuleData;

    function setUp() public override {
        super.setUp();

        (optionsDescriptions, optionsRecipients, optionsAmounts) = _createMinimalFundingArrays(1);
        (governanceFundExpectedId, governanceFundVotingModuleData) = _createFundingProposalForMoveToVote(
            approvedProposer,
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            governanceFundProposalDescription,
            governanceFundProposalType
        );
        (councilBudgetExpectedId, councilBudgetVotingModuleData) = _createFundingProposalForMoveToVote(
            approvedProposer,
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            councilBudgetProposalDescription,
            councilBudgetProposalType
        );
    }

    function test_moveToVoteFundingProposal_invalidFundingProposalType_reverts(
        uint8 _proposalTypeValue,
        string memory _proposalDescription
    )
        public
    {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        _proposalTypeValue = uint8(bound(_proposalTypeValue, 0, 2));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(_proposalTypeValue);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidFundingProposalType.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue, optionsDescriptions, optionsRecipients, optionsAmounts, _proposalDescription, proposalType
        );
    }

    function test_moveToVoteFundingProposal_invalidProposal_reverts(
        uint8 _proposalTypeValue,
        uint128 _criteriaValue
    )
        public
    {
        // Ensure the criteria value is not the same as the one in setUp so when calculating the proposal ID it will
        // not find the proposal
        vm.assume(_criteriaValue != criteriaValue);

        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        _proposalTypeValue = uint8(bound(_proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(_proposalTypeValue);

        string memory proposalDescription;
        if (proposalType == governanceFundProposalType) {
            proposalDescription = governanceFundProposalDescription;
        } else {
            proposalDescription = councilBudgetProposalDescription;
        }

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidProposal.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            _criteriaValue, optionsDescriptions, optionsRecipients, optionsAmounts, proposalDescription, proposalType
        );
    }

    function test_moveToVoteFundingProposal_invalidProposalWrongProposalType_reverts(
        uint8 _wrongProposalTypeValue,
        uint8 _validProposalTypeValue
    )
        public
    {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        _wrongProposalTypeValue = uint8(bound(_wrongProposalTypeValue, 0, 2));
        ProposalValidator.ProposalType wrongProposalType = ProposalValidator.ProposalType(_wrongProposalTypeValue);

        _validProposalTypeValue = uint8(bound(_validProposalTypeValue, 3, 4));
        ProposalValidator.ProposalType validProposalType = ProposalValidator.ProposalType(_validProposalTypeValue);

        string memory proposalDescription;
        if (validProposalType == governanceFundProposalType) {
            // Set proposal data proposal type to a different value
            validator.setProposalData(
                governanceFundExpectedId, approvedProposer, wrongProposalType, false, 0, CYCLE_NUMBER
            );
            proposalDescription = governanceFundProposalDescription;
        } else {
            // Set proposal data proposal type to a different value
            validator.setProposalData(
                councilBudgetExpectedId, approvedProposer, wrongProposalType, false, 0, CYCLE_NUMBER
            );
            proposalDescription = councilBudgetProposalDescription;
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
            proposalDescription,
            validProposalType
        );
    }

    function test_moveToVoteFundingProposal_insufficientApprovals_reverts(uint8 _proposalTypeValue) public {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        _proposalTypeValue = uint8(bound(_proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(_proposalTypeValue);

        string memory proposalDescription;
        if (proposalType == governanceFundProposalType) {
            // Set proposal data approved count to 0 since it is 1 by the approval on the setUp
            validator.setProposalData(governanceFundExpectedId, approvedProposer, proposalType, false, 0, CYCLE_NUMBER);
            proposalDescription = governanceFundProposalDescription;
        } else {
            // Set proposal data approved count to 0 since it is 1 by the approval on the setUp
            validator.setProposalData(councilBudgetExpectedId, approvedProposer, proposalType, false, 0, CYCLE_NUMBER);
            proposalDescription = councilBudgetProposalDescription;
        }

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InsufficientApprovals.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue, optionsDescriptions, optionsRecipients, optionsAmounts, proposalDescription, proposalType
        );
    }

    function test_moveToVoteFundingProposal_proposalAlreadyMovedToVote_reverts(uint8 _proposalTypeValue) public {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        _proposalTypeValue = uint8(bound(_proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(_proposalTypeValue);

        string memory proposalDescription;
        if (proposalType == governanceFundProposalType) {
            // Set proposal data movedToVote to true
            validator.setProposalData(governanceFundExpectedId, approvedProposer, proposalType, true, 1, CYCLE_NUMBER);
            proposalDescription = governanceFundProposalDescription;
        } else {
            // Set proposal data movedToVote to true
            validator.setProposalData(councilBudgetExpectedId, approvedProposer, proposalType, true, 1, CYCLE_NUMBER);
            proposalDescription = councilBudgetProposalDescription;
        }

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyMovedToVote.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue, optionsDescriptions, optionsRecipients, optionsAmounts, proposalDescription, proposalType
        );
    }

    function test_moveToVoteFundingProposal_invalidVotingCycle_reverts(uint8 _proposalTypeValue) public {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        _proposalTypeValue = uint8(bound(_proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(_proposalTypeValue);

        string memory proposalDescription;
        if (proposalType == governanceFundProposalType) {
            proposalDescription = governanceFundProposalDescription;
        } else {
            proposalDescription = councilBudgetProposalDescription;
        }

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidVotingCycle.selector);
        vm.warp(START_TIMESTAMP + DURATION + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue, optionsDescriptions, optionsRecipients, optionsAmounts, proposalDescription, proposalType
        );
    }

    function test_moveToVoteFundingProposal_buildApprovalModuleOptionsExceedsProposalDistributionThreshold_reverts(
        uint8 _proposalTypeValue
    )
        public
    {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        _proposalTypeValue = uint8(bound(_proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(_proposalTypeValue);

        string memory proposalDescription;
        if (proposalType == governanceFundProposalType) {
            proposalDescription = governanceFundProposalDescription;
        } else {
            proposalDescription = councilBudgetProposalDescription;
        }

        // Set the first option amount to exceed the distribution threshold
        optionsAmounts[0] = DISTRIBUTION_THRESHOLD + 1;

        vm.expectRevert(IProposalValidator.ProposalValidator_ExceedsDistributionThreshold.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue, optionsDescriptions, optionsRecipients, optionsAmounts, proposalDescription, proposalType
        );
    }

    function test_moveToVoteFundingProposal_exceedsDistributionThreshold_reverts(uint8 _proposalTypeValue) public {
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        _proposalTypeValue = uint8(bound(_proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(_proposalTypeValue);

        string memory proposalDescription;
        if (proposalType == governanceFundProposalType) {
            proposalDescription = governanceFundProposalDescription;
        } else {
            proposalDescription = councilBudgetProposalDescription;
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

        _optionsAmounts[0] = DISTRIBUTION_THRESHOLD - 1;
        _optionsAmounts[1] = DISTRIBUTION_THRESHOLD - 1;
        _optionsAmounts[2] = DISTRIBUTION_THRESHOLD - 1;

        _createFundingProposalForMoveToVote(
            approvedProposer,
            criteriaValue,
            _optionsDescriptions,
            _optionsRecipients,
            _optionsAmounts,
            proposalDescription,
            proposalType
        );

        vm.expectRevert(IProposalValidator.ProposalValidator_ExceedsDistributionThreshold.selector);
        vm.prank(approvedProposer);
        vm.warp(START_TIMESTAMP + 1);
        validator.moveToVoteFundingProposal(
            criteriaValue, _optionsDescriptions, _optionsRecipients, _optionsAmounts, proposalDescription, proposalType
        );
    }

    function test_moveToVoteFundingProposal_proposalIdMismatch_reverts(
        uint8 _proposalTypeValue,
        uint256 _randomId
    )
        public
    {
        vm.assume(_randomId != governanceFundExpectedId && _randomId != councilBudgetExpectedId);

        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        _proposalTypeValue = uint8(bound(_proposalTypeValue, 3, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(_proposalTypeValue);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        bytes memory votingModuleData;
        string memory proposalDescription;
        if (proposalType == governanceFundProposalType) {
            votingModuleData = governanceFundVotingModuleData;
            proposalDescription = governanceFundProposalDescription;
        } else {
            votingModuleData = councilBudgetVotingModuleData;
            proposalDescription = councilBudgetProposalDescription;
        }

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (approvalVotingModule, votingModuleData, proposalDescription, APPROVAL_VOTING_MODULE_ID)
            ),
            abi.encode(uint256(_randomId))
        );

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalIdMismatch.selector);
        vm.warp(START_TIMESTAMP + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue, optionsDescriptions, optionsRecipients, optionsAmounts, proposalDescription, proposalType
        );
    }
}

/// @title ProposalValidator_Setters_Test
/// @notice Tests for setter functions
contract ProposalValidator_Setters_Test is ProposalValidator_Init {
    function testFuzz_setVotingCycleData_succeeds(
        uint256 cycleNumber,
        uint256 startingTimestamp,
        uint256 duration,
        uint256 distributionLimit
    )
        public
    {
        vm.assume(cycleNumber != CYCLE_NUMBER); // Avoid existing cycle

        // Expect the VotingCycleDataSet event to be emitted
        vm.expectEmit(address(validator));
        emit VotingCycleDataSet(cycleNumber, startingTimestamp, duration, distributionLimit);

        vm.prank(owner);
        validator.setVotingCycleData(cycleNumber, startingTimestamp, duration, distributionLimit);

        (
            uint256 actualStartingTimestamp,
            uint256 actualDuration,
            uint256 actualDistributionLimit,
            uint256 actualMovedToVoteTokenCount
        ) = validator.votingCycles(cycleNumber);

        assertEq(actualStartingTimestamp, startingTimestamp);
        assertEq(actualDuration, duration);
        assertEq(actualDistributionLimit, distributionLimit);
        assertEq(actualMovedToVoteTokenCount, 0);
    }

    function testFuzz_setVotingCycleData_notOwner_reverts(
        address caller,
        uint256 cycleNumber,
        uint256 startingTimestamp,
        uint256 duration,
        uint256 distributionLimit
    )
        public
    {
        vm.assume(caller != owner);

        vm.prank(caller);
        vm.expectRevert("Ownable: caller is not the owner");
        validator.setVotingCycleData(cycleNumber, startingTimestamp, duration, distributionLimit);
    }

    function testFuzz_setVotingCycleData_votingCycleAlreadySet_reverts(
        uint256 startingTimestamp,
        uint256 duration,
        uint256 distributionLimit
    )
        public
    {
        vm.expectRevert(ProposalValidator.ProposalValidator_VotingCycleAlreadySet.selector);
        vm.prank(owner);
        validator.setVotingCycleData(CYCLE_NUMBER, startingTimestamp, duration, distributionLimit);
    }

    function testFuzz_setProposalDistributionThreshold_succeeds(uint256 newProposalDistributionThreshold) public {
        // Expect the ProposalDistributionThresholdSet event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalDistributionThresholdSet(newProposalDistributionThreshold);

        vm.prank(owner);
        validator.setProposalDistributionThreshold(newProposalDistributionThreshold);

        assertEq(validator.proposalDistributionThreshold(), newProposalDistributionThreshold);
    }

    function testFuzz_setProposalDistributionThreshold_notOwner_reverts(address caller, uint256 threshold) public {
        vm.assume(caller != owner);

        vm.prank(caller);
        vm.expectRevert("Ownable: caller is not the owner");
        validator.setProposalDistributionThreshold(threshold);
    }

    function testFuzz_setProposalTypeData_succeeds(
        uint8 proposalTypeValue,
        uint256 newRequiredApprovals,
        uint8 newProposalTypeId
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        ProposalValidator.ProposalTypeData memory newData = ProposalValidator.ProposalTypeData({
            requiredApprovals: newRequiredApprovals,
            idInConfigurator: newProposalTypeId
        });

        // Expect the ProposalTypeDataSet event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalTypeDataSet(proposalType, newRequiredApprovals, newProposalTypeId);

        vm.prank(owner);
        validator.setProposalTypeData(proposalType, newData);

        (uint256 requiredApprovals, uint8 idInConfigurator) = validator.proposalTypesData(proposalType);
        assertEq(requiredApprovals, newRequiredApprovals);
        assertEq(idInConfigurator, newProposalTypeId);
    }

    function testFuzz_setProposalTypeData_notOwner_reverts(address caller) public {
        vm.assume(caller != owner);

        ProposalValidator.ProposalTypeData memory newData =
            ProposalValidator.ProposalTypeData({ requiredApprovals: 4, idInConfigurator: 0 });

        vm.prank(caller);
        vm.expectRevert("Ownable: caller is not the owner");
        validator.setProposalTypeData(ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade, newData);
    }
}

/// @title ProposalValidator_HashProposalWithModule_Test
/// @notice Tests for the hashProposalWithModule function
contract ProposalValidator_HashProposalWithModule_Test is ProposalValidator_Init {
    function testFuzz_hashProposalWithModule_succeeds(
        address module,
        bytes memory proposalData,
        bytes32 descriptionHash
    )
        public
        view
    {
        uint256 id = validator.hashProposalWithModule(module, proposalData, descriptionHash);
        uint256 expectedId =
            uint256(keccak256(abi.encode(address(validator.GOVERNOR()), module, proposalData, descriptionHash)));

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
