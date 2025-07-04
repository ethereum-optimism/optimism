// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IProposalValidator } from "interfaces/governance/IProposalValidator.sol";
import { IOptimismGovernor } from "interfaces/governance/IOptimismGovernor.sol";
import { IGovernanceToken } from "interfaces/governance/IGovernanceToken.sol";
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

// Testing utilities
import { stdStorage, StdStorage } from "forge-std/Test.sol";

// Contracts
import { ProposalValidator } from "src/governance/ProposalValidator.sol";
import { Proxy } from "src/universal/Proxy.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Modules
import {
    ProposalSettings as ApprovalProposalSettings,
    ProposalOption,
    PassingCriteria
} from "src/governance/ApprovalVotingModule.sol";
import { ProposalSettings as OptimisticProposalSettings } from "src/governance/OptimisticModule.sol";
import { VotingModule } from "src/governance/VotingModule.sol";

// Testing utilities
import { stdStorage, StdStorage } from "forge-std/Test.sol";
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title ProposalValidatorForTest
/// @notice A test contract that exposes the private _hashProposal function
contract ProposalValidatorForTest is ProposalValidator {
    constructor(
        bytes32 _approvedProposerAttestationSchemaUid,
        bytes32 _topDelegatesAttestationSchemaUid,
        IOptimismGovernor _governor,
        IGovernanceToken _governanceToken
    )
        ProposalValidator(
            _approvedProposerAttestationSchemaUid,
            _topDelegatesAttestationSchemaUid,
            _governor,
            _governanceToken
        )
    { }

    function hashProposalWithModule(
        address _module,
        bytes memory _proposalData,
        bytes32 _descriptionHash
    )
        public
        view
        returns (bytes32)
    {
        return _hashProposalWithModule(_module, _proposalData, _descriptionHash);
    }

    /// @notice Exposes proposal data for testing
    function getProposalData(bytes32 _proposalHash)
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
        ProposalData storage proposal = _proposals[_proposalHash];
        return (
            proposal.proposer, proposal.proposalType, proposal.movedToVote, proposal.approvalCount, proposal.votingCycle
        );
    }

    function setProposalData(
        bytes32 _proposalHash,
        address _proposer,
        ProposalType _proposalType,
        bool _movedToVote,
        uint256 _approvalCount,
        uint256 _votingCycle
    )
        public
    {
        _proposals[_proposalHash].proposer = _proposer;
        _proposals[_proposalHash].proposalType = _proposalType;
        _proposals[_proposalHash].movedToVote = _movedToVote;
        _proposals[_proposalHash].approvalCount = _approvalCount;
        _proposals[_proposalHash].votingCycle = _votingCycle;
    }

    function mockApproveProposal(bytes32 _proposalHash, address _delegate) public {
        _proposals[_proposalHash].delegateApprovals[_delegate] = true;
    }

    /// @notice Check if a delegate has approved a proposal
    function hasDelegateApproved(bytes32 _proposalHash, address _delegate) public view returns (bool hasApproved_) {
        return _proposals[_proposalHash].delegateApprovals[_delegate];
    }
}

/// @title ProposalValidator_Init
/// @notice Setup contract for ProposalValidator tests
contract ProposalValidator_Init is CommonTest {
    using stdStorage for StdStorage;

    uint256 public constant CYCLE_NUMBER = 1;
    uint256 public constant START_BLOCK = 1000000;
    uint256 public constant DURATION = 100;
    uint256 public constant DISTRIBUTION_LIMIT = 20000 ether;
    uint256 public constant DISTRIBUTION_THRESHOLD = 10000 ether;
    uint256 public constant PROPOSAL_REQUIRED_APPROVALS = 1;
    uint256 public constant MINIMUM_VOTING_POWER = 10000 ether;
    uint256 public constant OPTIMISTIC_MODULE_PERCENT_DIVISOR = 10_000;
    uint8 public constant APPROVAL_VOTING_MODULE_ID = 1;
    uint8 public constant OPTIMISTIC_VOTING_MODULE_ID = 2;

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
    bytes32 public APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID;
    bytes32 public TOP_DELEGATES_ATTESTATION_SCHEMA_UID;

    event ProposalSubmitted(
        bytes32 indexed proposalHash,
        address indexed proposer,
        string description,
        ProposalValidator.ProposalType proposalType
    );
    event ProposalApproved(bytes32 indexed proposalHash, address indexed approver);
    event ProposalMovedToVote(bytes32 indexed proposalHash, address indexed executor);
    event MinimumVotingPowerSet(uint256 newMinimumVotingPower);
    event VotingCycleDataSet(
        uint256 cycleNumber, uint256 startBlock, uint256 duration, uint256 votingCycleDistributionLimit
    );
    event DistributionThresholdSet(uint256 newDistributionThreshold);
    event ProposalTypeDataSet(
        ProposalValidator.ProposalType proposalType, uint256 requiredApprovals, uint8 proposalVotingModule
    );
    event ProposalVotingModuleData(bytes32 indexed proposalHash, bytes encodedVotingModuleData);

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

        // Set proposalVotingModule (depth 1)
        stdstore.target(address(validator)).sig("proposalTypesData(uint8)").with_key(uint256(_proposalType)).depth(1)
            .checked_write(_data.proposalVotingModule);
    }

    /// @notice Helper function to set CouncilMemberElections proposal type data.
    function _setCouncilMemberElectionsProposalType() internal {
        _setProposalTypeData(
            ProposalValidator.ProposalType.CouncilMemberElections,
            ProposalValidator.ProposalTypeData({
                requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
                proposalVotingModule: APPROVAL_VOTING_MODULE_ID
            })
        );
    }

    /// @notice Helper function to set GovernanceFund proposal type data.
    function _setGovernanceFundProposalType() internal {
        _setProposalTypeData(
            ProposalValidator.ProposalType.GovernanceFund,
            ProposalValidator.ProposalTypeData({
                requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
                proposalVotingModule: APPROVAL_VOTING_MODULE_ID
            })
        );
    }

    /// @notice Helper function to set CouncilBudget proposal type data.
    function _setCouncilBudgetProposalType() internal {
        _setProposalTypeData(
            ProposalValidator.ProposalType.CouncilBudget,
            ProposalValidator.ProposalTypeData({
                requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
                proposalVotingModule: APPROVAL_VOTING_MODULE_ID
            })
        );
    }

    /// @notice Helper function to set ProtocolOrGovernorUpgrade proposal type data.
    function _setProtocolOrGovernorUpgradeProposalType() internal {
        _setProposalTypeData(
            ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade,
            ProposalValidator.ProposalTypeData({
                requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
                proposalVotingModule: OPTIMISTIC_VOTING_MODULE_ID
            })
        );
    }

    /// @notice Helper function to set MaintenanceUpgrade proposal type data.
    function _setMaintenanceUpgradeProposalType() internal {
        _setProposalTypeData(
            ProposalValidator.ProposalType.MaintenanceUpgrade,
            ProposalValidator.ProposalTypeData({
                requiredApprovals: 0, // MaintenanceUpgrade moves directly to voting
                proposalVotingModule: OPTIMISTIC_VOTING_MODULE_ID
            })
        );
    }

    /// @notice Helper to create minimal valid arrays for funding proposal error tests

    function _createMinimalFundingArrays()
        internal
        pure
        returns (string[] memory descriptions_, address[] memory recipients_, uint256[] memory amounts_)
    {
        descriptions_ = new string[](1);
        descriptions_[0] = "Option A";

        recipients_ = new address[](1);
        recipients_[0] = address(0x1);

        amounts_ = new uint256[](1);
        amounts_[0] = 100 ether;
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
            proposalVotingModule: OPTIMISTIC_VOTING_MODULE_ID
        });
        // MaintenanceUpgrade
        proposalTypesData[1] = ProposalValidator.ProposalTypeData({
            requiredApprovals: 0,
            proposalVotingModule: OPTIMISTIC_VOTING_MODULE_ID
        });
        // CouncilMemberElections
        proposalTypesData[2] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            proposalVotingModule: APPROVAL_VOTING_MODULE_ID
        });
        // GovernanceFund
        proposalTypesData[3] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            proposalVotingModule: APPROVAL_VOTING_MODULE_ID
        });
        // CouncilBudget
        proposalTypesData[4] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            proposalVotingModule: APPROVAL_VOTING_MODULE_ID
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
        ProposalOption[] memory options = new ProposalOption[](descriptions.length);

        for (uint256 i = 0; i < descriptions.length; i++) {
            address[] memory targets = new address[](1);
            uint256[] memory values = new uint256[](1);
            bytes[] memory calldatas = new bytes[](1);

            targets[0] = Predeploys.GOVERNANCE_TOKEN;
            calldatas[0] = abi.encodeCall(IERC20.transfer, (recipients[i], amounts[i]));

            options[i] = ProposalOption({
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
        ApprovalProposalSettings memory settings = ApprovalProposalSettings({
            maxApprovals: uint8(descriptions.length),
            criteria: uint8(PassingCriteria.Threshold),
            budgetToken: Predeploys.GOVERNANCE_TOKEN,
            criteriaValue: criteriaValue,
            budgetAmount: uint128(totalBudget)
        });

        return abi.encode(options, settings);
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
        ProposalOption[] memory options = new ProposalOption[](descriptions.length);

        for (uint256 i = 0; i < descriptions.length; i++) {
            address[] memory targets = new address[](0);
            uint256[] memory values = new uint256[](0);
            bytes[] memory calldatas = new bytes[](0);

            options[i] = ProposalOption({
                budgetTokensSpent: 0,
                targets: targets,
                values: values,
                calldatas: calldatas,
                description: descriptions[i]
            });
        }

        // Construct ProposalSettings with TopChoices criteria
        ApprovalProposalSettings memory settings = ApprovalProposalSettings({
            maxApprovals: uint8(descriptions.length),
            criteria: uint8(PassingCriteria.TopChoices),
            budgetToken: address(0),
            criteriaValue: criteriaValue,
            budgetAmount: 0
        });

        return abi.encode(options, settings);
    }

    /// @notice Helper function to construct voting module data for upgrade proposals
    function _constructOptimisticVotingModuleData(uint248 againstThreshold) internal pure returns (bytes memory) {
        OptimisticProposalSettings memory settings =
            OptimisticProposalSettings({ againstThreshold: againstThreshold, isRelativeToVotableSupply: true });

        return abi.encode(settings);
    }

    /// @notice Helper function to create a proposal for move to vote
    function _createUpgradeProposalForMoveToVote(
        address proposer,
        uint248 againstThreshold,
        string memory proposalDescription
    )
        internal
        returns (bytes32 proposalHash_, bytes memory votingModuleData_)
    {
        // Calculate expected proposal hash
        votingModuleData_ = _constructOptimisticVotingModuleData(againstThreshold);
        proposalHash_ = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData_, keccak256(bytes(proposalDescription))
        );

        // 1 vote as default for being able to move to vote
        validator.setProposalData(
            proposalHash_,
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
        returns (bytes32 proposalHash_, bytes memory votingModuleData_)
    {
        votingModuleData_ = _constructCouncilElectionVotingModuleData(optionsDescriptions, criteriaValue);
        proposalHash_ = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData_, keccak256(bytes(proposalDescription))
        );

        validator.setProposalData(
            proposalHash_,
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
        returns (bytes32 proposalHash_, bytes memory votingModuleData_)
    {
        votingModuleData_ =
            _constructFundingVotingModuleData(optionsDescriptions, optionsRecipients, optionsAmounts, criteriaValue);
        proposalHash_ = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData_, keccak256(bytes(proposalDescription))
        );

        validator.setProposalData(
            proposalHash_, proposer, proposalType, false, PROPOSAL_REQUIRED_APPROVALS, CYCLE_NUMBER
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

    /// @notice Initializes the validator
    function _initializeValidator() internal virtual {
        (
            ProposalValidator.ProposalType[] memory proposalTypes,
            ProposalValidator.ProposalTypeData[] memory proposalTypesData
        ) = _getProposalTypesAndData();

        // Create mock addresses
        proposalTypesConfigurator = IProposalTypesConfigurator(makeAddr("proposalTypesConfigurator"));

        impl = new ProposalValidatorForTest(
            APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID, TOP_DELEGATES_ATTESTATION_SCHEMA_UID, governor, governanceToken
        );
        validator = ProposalValidatorForTest(address(new Proxy(owner)));

        vm.prank(owner);
        IProxy(payable(address(validator))).upgradeToAndCall(
            address(impl),
            abi.encodeCall(
                impl.initialize,
                (
                    owner,
                    proposalTypesConfigurator,
                    CYCLE_NUMBER,
                    START_BLOCK,
                    DURATION,
                    DISTRIBUTION_LIMIT,
                    DISTRIBUTION_THRESHOLD,
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
            "address approvedAddress,uint8 proposalType", ISchemaResolver(address(0)), true
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
                    recipient: address(0),
                    expirationTime: 0,
                    revocable: true,
                    refUID: bytes32(0),
                    data: abi.encode(_delegate, _proposalType),
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
    function test_version_succeeds() public {
        string memory versionString = validator.version();
        assertEq(versionString, "1.0.0-beta.1");
    }
}

/// @title ProposalValidator_Initialize_Test
/// @notice Tests for the initialize function
contract ProposalValidator_Initialize_Test is ProposalValidator_Init {
    /// @dev Override to create validator proxy without initialization for testing
    function _initializeValidator() internal override {
        // Create mock addresses
        proposalTypesConfigurator = IProposalTypesConfigurator(makeAddr("proposalTypesConfigurator"));

        impl = new ProposalValidatorForTest(
            APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID, TOP_DELEGATES_ATTESTATION_SCHEMA_UID, governor, governanceToken
        );
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
                    proposalTypesConfigurator,
                    CYCLE_NUMBER,
                    START_BLOCK,
                    DURATION,
                    DISTRIBUTION_LIMIT,
                    DISTRIBUTION_THRESHOLD,
                    proposalTypes,
                    proposalTypesData
                )
            )
        );

        // Verify initialization was successful
        assertEq(validator.distributionThreshold(), DISTRIBUTION_THRESHOLD);
        assertEq(validator.owner(), owner);

        // Verify voting cycle data
        (uint256 startBlock, uint256 duration, uint256 distributionLimit, uint256 movedToVoteTokenCount) =
            validator.votingCycles(CYCLE_NUMBER);
        assertEq(startBlock, START_BLOCK);
        assertEq(duration, DURATION);
        assertEq(distributionLimit, DISTRIBUTION_LIMIT);
        assertEq(movedToVoteTokenCount, 0);

        // Verify proposal type data
        for (uint256 i = 0; i < proposalTypes.length; i++) {
            (uint256 requiredApprovals, uint8 proposalVotingModule) = validator.proposalTypesData(proposalTypes[i]);
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
                assertEq(proposalVotingModule, APPROVAL_VOTING_MODULE_ID);
            } else {
                // ProtocolOrGovernorUpgrade and MaintenanceUpgrade use OPTIMISTIC_VOTING_MODULE_ID
                assertEq(proposalVotingModule, OPTIMISTIC_VOTING_MODULE_ID);
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
        proposalTypesData[0] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            proposalVotingModule: 0
        });
        proposalTypesData[1] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            proposalVotingModule: 1
        });

        vm.prank(owner);
        vm.expectRevert("Proxy: delegatecall to new implementation contract failed");
        IProxy(payable(address(validator))).upgradeToAndCall(
            address(impl),
            abi.encodeCall(
                impl.initialize,
                (
                    owner,
                    proposalTypesConfigurator,
                    CYCLE_NUMBER,
                    START_BLOCK,
                    DURATION,
                    DISTRIBUTION_LIMIT,
                    DISTRIBUTION_THRESHOLD,
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

        // Calculate expected proposal hash
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(againstThreshold);
        bytes32 expectedHash = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
        );

        // Mock proposalSnapshot to return 0 (proposal doesn't exist in governor)
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (uint256(expectedHash))),
            abi.encode(0)
        );

        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (VotingModule(optimisticVotingModule), votingModuleData, proposalDescription, uint8(proposalType))
            ),
            abi.encode(uint256(expectedHash))
        );

        // For MaintenanceUpgrade, events are: ProposalSubmitted, ProposalVotingModuleData, ProposalMovedToVote
        vm.expectEmit(address(validator));
        emit ProposalSubmitted(expectedHash, proposer, proposalDescription, proposalType);

        vm.expectEmit(address(validator));
        emit ProposalVotingModuleData(expectedHash, votingModuleData);

        vm.expectEmit(address(validator));
        emit ProposalMovedToVote(expectedHash, proposer);

        vm.prank(proposer);
        bytes32 proposalHash = validator.submitUpgradeProposal(
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
        );

        assertEq(proposalHash, expectedHash);

        // Verify proposal data was stored correctly
        (
            address storedProposer,
            ProposalValidator.ProposalType storedProposalType,
            bool movedToVote,
            uint256 approvalCount,
            uint256 votingCycle
        ) = validator.getProposalData(proposalHash);

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

        // Calculate expected proposal hash
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(againstThreshold);
        bytes32 expectedHash = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
        );

        // Mock proposalSnapshot to return 0 (proposal doesn't exist in governor)
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (uint256(expectedHash))),
            abi.encode(0)
        );

        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // For ProtocolOrGovernorUpgrade, only ProposalSubmitted and ProposalVotingModuleData events
        vm.expectEmit(address(validator));
        emit ProposalSubmitted(expectedHash, proposer, proposalDescription, proposalType);

        vm.expectEmit(address(validator));
        emit ProposalVotingModuleData(expectedHash, votingModuleData);

        vm.prank(proposer);
        bytes32 proposalHash = validator.submitUpgradeProposal(
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
        );

        assertEq(proposalHash, expectedHash);

        // Verify proposal data was stored correctly
        (
            address storedProposer,
            ProposalValidator.ProposalType storedProposalType,
            bool movedToVote,
            uint256 approvalCount,
            uint256 votingCycle
        ) = validator.getProposalData(proposalHash);

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

        uint248 againstThreshold = 5000; // 50%
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidUpgradeProposalType.selector);
        vm.prank(topDelegate_A);
        validator.submitUpgradeProposal(
            againstThreshold, proposalDescription, attestationUid, proposalType, CYCLE_NUMBER
        );
    }

    function testFuzz_submitUpgradeProposal_invalidAttestation_reverts(bytes32 fuzzedAttestationUid) public {
        uint248 againstThreshold = 5000;
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

        uint248 againstThreshold = 5000;
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // Try to submit with different address than attested
        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(fuzzedProposer); // Different from attested topDelegate_A
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

    function testFuzz_submitUpgradeProposal_duplicateProposal_reverts(uint8 proposalTypeValue) public {
        // Bound proposal type to only upgrade proposals (0 = ProtocolOrGovernorUpgrade, 1 = MaintenanceUpgrade)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 1));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        uint248 againstThreshold = 5000;
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // Calculate expected proposal hash
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(againstThreshold);
        bytes32 expectedHash = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
        );

        // Mock proposalSnapshot to return 0 for first submission
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (uint256(expectedHash))),
            abi.encode(0)
        );

        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // For MaintenanceUpgrade, mock the governor.proposeWithModule call
        if (proposalType == ProposalValidator.ProposalType.MaintenanceUpgrade) {
            _mockAndExpect(
                address(governor),
                abi.encodeCall(
                    IOptimismGovernor.proposeWithModule,
                    (VotingModule(optimisticVotingModule), votingModuleData, proposalDescription, uint8(proposalType))
                ),
                abi.encode(uint256(expectedHash))
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

        uint248 againstThreshold = 5000;
        bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

        // Calculate expected proposal hash
        bytes memory votingModuleData = _constructOptimisticVotingModuleData(againstThreshold);
        bytes32 expectedHash = validator.hashProposalWithModule(
            optimisticVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
        );

        // Mock proposalSnapshot to return non-zero (proposal already exists in governor)
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (uint256(expectedHash))),
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

        uint248 againstThreshold = 5000;
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;

        // Create attestation but don't use proper owner as attester
        vm.prank(fuzzedAttester); // Not the owner
        bytes32 invalidAttestation = IEAS(Predeploys.EAS).attest(
            AttestationRequest({
                schema: APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID,
                data: AttestationRequestData({
                    recipient: address(0),
                    expirationTime: 0,
                    revocable: false,
                    refUID: bytes32(0),
                    data: abi.encode(topDelegate_A, proposalType),
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

        uint248 againstThreshold = 5000;

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

        // Calculate expected proposal hash
        bytes memory votingModuleData = _constructCouncilElectionVotingModuleData(optionDescriptions, criteriaValue);
        bytes32 expectedHash = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
        );

        // Mock proposalSnapshot to return 0 (proposal doesn't exist in governor)
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (uint256(expectedHash))),
            abi.encode(0)
        );

        // Expect ProposalSubmitted event
        vm.expectEmit(address(validator));
        emit ProposalSubmitted(
            expectedHash, topDelegate_A, proposalDescription, ProposalValidator.ProposalType.CouncilMemberElections
        );

        // Expect ProposalVotingModuleData event
        vm.expectEmit(address(validator));
        emit ProposalVotingModuleData(expectedHash, votingModuleData);

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.prank(topDelegate_A);
        bytes32 proposalHash = validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, attestationUid, CYCLE_NUMBER
        );

        assertEq(proposalHash, expectedHash);

        // Verify proposal data was stored correctly
        (
            address proposer,
            ProposalValidator.ProposalType proposalType,
            bool movedToVote,
            uint256 approvalCount,
            uint256 votingCycle
        ) = validator.getProposalData(proposalHash);

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

    function test_submitCouncilMemberElectionsProposal_zeroOptions_reverts() public {
        string[] memory emptyOptions = new string[](0);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidOptionsLength.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, emptyOptions, proposalDescription, attestationUid, CYCLE_NUMBER
        );
    }

    function test_submitCouncilMemberElectionsProposal_duplicateProposal_reverts() public {
        // Calculate expected proposal hash
        bytes memory votingModuleData = _constructCouncilElectionVotingModuleData(optionDescriptions, criteriaValue);
        bytes32 expectedHash = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
        );

        // Mock proposalSnapshot to return 0 for first submission
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (uint256(expectedHash))),
            abi.encode(0)
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
        // Calculate expected proposal hash
        bytes memory votingModuleData = _constructCouncilElectionVotingModuleData(optionDescriptions, criteriaValue);
        bytes32 expectedHash = validator.hashProposalWithModule(
            approvalVotingModule, votingModuleData, keccak256(bytes(proposalDescription))
        );

        // Mock proposalSnapshot to return non-zero (proposal already exists in governor)
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (uint256(expectedHash))),
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
                    recipient: address(0),
                    expirationTime: 0,
                    revocable: false,
                    refUID: bytes32(0),
                    data: abi.encode(topDelegate_A, ProposalValidator.ProposalType.CouncilMemberElections),
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
            _createMinimalFundingArrays();

        for (uint256 i = 0; i < optionCount; i++) {
            descriptions[i] = descriptions[0];
            recipients[i] = recipients[0];
            amounts[i] = amount;
        }

        // Calculate expected proposal hash
        bytes memory votingModuleData =
            _constructFundingVotingModuleData(descriptions, recipients, amounts, criteriaValue);
        bytes32 expectedHash =
            validator.hashProposalWithModule(approvalVotingModule, votingModuleData, keccak256(bytes(description)));

        // Mock proposalSnapshot to return 0 (proposal doesn't exist in governor)
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (uint256(expectedHash))),
            abi.encode(0)
        );

        // Expect ProposalSubmitted event
        vm.expectEmit(address(validator));
        emit ProposalSubmitted(expectedHash, proposer, description, proposalType);

        // Expect ProposalVotingModuleData event
        vm.expectEmit(address(validator));
        emit ProposalVotingModuleData(expectedHash, votingModuleData);

        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.prank(proposer);
        bytes32 proposalHash = validator.submitFundingProposal(
            criteriaValue, descriptions, recipients, amounts, description, proposalType, CYCLE_NUMBER
        );

        assertEq(proposalHash, expectedHash);

        // Verify proposal data was stored correctly
        (
            address storedProposer,
            ProposalValidator.ProposalType storedProposalType,
            bool movedToVote,
            uint256 approvalCount,
            uint256 votingCycle
        ) = validator.getProposalData(proposalHash);

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
            _createMinimalFundingArrays();

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidFundingProposalType.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, description, proposalType, CYCLE_NUMBER
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

    function testFuzz_submitFundingProposal_exceedsDistributionThreshold_reverts(
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
            _createMinimalFundingArrays();
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
            _createMinimalFundingArrays();

        // Calculate expected proposal hash
        bytes memory votingModuleData =
            _constructFundingVotingModuleData(descriptions, recipients, amounts, FUNDING_CRITERIA_VALUE);
        bytes32 expectedHash =
            validator.hashProposalWithModule(approvalVotingModule, votingModuleData, keccak256(bytes(description)));

        // Mock proposalSnapshot to return 0 for first submission
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (uint256(expectedHash))),
            abi.encode(0)
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
            _createMinimalFundingArrays();

        // Calculate expected proposal hash
        bytes memory votingModuleData =
            _constructFundingVotingModuleData(descriptions, recipients, amounts, FUNDING_CRITERIA_VALUE);
        bytes32 expectedHash =
            validator.hashProposalWithModule(approvalVotingModule, votingModuleData, keccak256(bytes(description)));

        // Mock proposalSnapshot to return non-zero (proposal already exists in governor)
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (uint256(expectedHash))),
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
}

/// @title ProposalValidator_ApproveProposal_Test
/// @notice Happy path tests for approveProposal function
contract ProposalValidator_ApproveProposal_Test is ProposalValidator_Init {
    function test_approveProposal_succeeds(bytes32 _proposalHash, uint8 proposalTypeValue) public {
        // Ensure the proposal hash is not 0
        vm.assume(_proposalHash != bytes32(0));

        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        // Set mock proposal data of a random proposal in the validator contract
        validator.setProposalData(_proposalHash, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);

        // Expect event to be emitted when approving
        vm.expectEmit(address(validator));
        emit ProposalApproved(_proposalHash, topDelegate_A);

        // Approve the proposal, use the attestation of the top delegate that was created in setUp
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalHash, topDelegateAttestation_A);

        // Check that the proposal data has been updated
        assertTrue(validator.hasDelegateApproved(_proposalHash, topDelegate_A));

        (,,, uint256 approvalCount,) = validator.getProposalData(_proposalHash);
        assertEq(approvalCount, 1);
    }
}

/// @title ProposalValidator_ApproveProposal_TestFail
/// @notice Sad path tests for approveProposal function
contract ProposalValidator_ApproveProposal_TestFail is ProposalValidator_Init {
    function setUp() public override {
        super.setUp();
    }

    function test_approveProposal_proposalDoesNotExist_reverts(bytes32 _proposalHash) public {
        // There is no stored proposal data so this will revert
        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalDoesNotExist.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalHash, topDelegateAttestation_A);
    }

    function test_approveProposal_proposalAlreadyApproved_reverts(
        bytes32 _proposalHash,
        uint8 proposalTypeValue
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);
        // set proposal data so that the proposal exists
        validator.setProposalData(_proposalHash, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);

        // Mock the proposal as already approved by the top delegate
        validator.mockApproveProposal(_proposalHash, topDelegate_A);
        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyApproved.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalHash, topDelegateAttestation_A);
    }

    function test_approveProposal_invalidSchema_reverts(bytes32 _proposalHash, uint8 proposalTypeValue) public {
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
        validator.setProposalData(_proposalHash, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestationSchema.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalHash, _invalidAttestationUid);
    }

    function test_approveProposal_attestationRevoked_reverts(bytes32 _proposalHash, uint8 proposalTypeValue) public {
        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);
        // set proposal data so that the proposal exists
        validator.setProposalData(_proposalHash, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);

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
        validator.approveProposal(_proposalHash, topDelegateAttestation_A);
    }

    function test_approveProposal_invalidAttestationCaller_reverts(
        bytes32 _proposalHash,
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
        validator.setProposalData(_proposalHash, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);

        // Expect the invalid attestation error to be reverted
        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(_caller);
        validator.approveProposal(_proposalHash, topDelegateAttestation_A);
    }

    function test_approveProposal_invalidAttestationPartialDelegation_reverts(
        bytes32 _proposalHash,
        uint8 proposalTypeValue
    )
        public
    {
        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        // Set mock proposal data of a random proposal in the validator contract
        validator.setProposalData(_proposalHash, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);

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
        validator.approveProposal(_proposalHash, _attestationUidWithPartialDelegation);
    }

    function test_approveProposal_nonExistentAttestation_reverts(
        bytes32 _proposalHash,
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
        validator.setProposalData(_proposalHash, topDelegate_A, proposalType, false, 0, CYCLE_NUMBER);

        // Expect the invalid attestation error to be reverted when attestation doesn't exist
        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalHash, _nonExistentAttestationUid);
    }
}

/// @title ProposalValidator_CanApproveProposal_Test
/// @notice Tests for the canApproveProposal function
contract ProposalValidator_CanApproveProposal_Test is ProposalValidator_Init {
    function test_canApproveProposal_returnTrue_succeeds() public {
        // Attestation already created in setUp
        bool canApprove = validator.canApproveProposal(topDelegateAttestation_A, topDelegate_A);
        assertTrue(canApprove);
    }

    function test_canApproveProposal_returnFalse_succeeds(bytes32 attestationUid, address delegate) public {
        // Ensure the attestation uid is not one of the top delegates
        vm.assume(attestationUid != topDelegateAttestation_A);

        bool canApprove;
        // Expect the invalid attestation error to be reverted
        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestation.selector);
        try validator.canApproveProposal(attestationUid, delegate) returns (bool result_) {
            canApprove = result_;
        } catch {
            canApprove = false;
        }

        assertEq(canApprove, false);
    }
}

/// @title ProposalValidator_MoveToVoteProtocolOrGovernorUpgradeProposal_Test
/// @notice Happy path tests for moveToVoteProtocolOrGovernorUpgradeProposal function
contract ProposalValidator_MoveToVoteProtocolOrGovernorUpgradeProposal_Test is ProposalValidator_Init {
    uint248 againstThreshold = 5000; // 50%
    string proposalDescription = "Test proposal";
    ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
    bytes votingModuleData;
    bytes32 expectedHash;

    function setUp() public override {
        super.setUp();

        (expectedHash, votingModuleData) =
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
                (VotingModule(optimisticVotingModule), votingModuleData, proposalDescription, uint8(proposalType))
            ),
            abi.encode(uint256(expectedHash))
        );

        // Expect the ProposalMovedToVote event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalMovedToVote(expectedHash, approvedProposer);

        // Move to vote
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(againstThreshold, proposalDescription);

        // Check that the proposal is in voting
        (,, bool movedToVote,,) = validator.getProposalData(expectedHash);
        assertTrue(movedToVote, "Proposal should be in voting");
    }
}

contract ProposalValidator_MoveToVoteProtocolOrGovernorUpgradeProposal_TestFail is ProposalValidator_Init {
    uint248 againstThreshold = 5000; // 50%
    string proposalDescription = "Test proposal";
    ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
    bytes votingModuleData;
    bytes32 expectedHash;

    function setUp() public override {
        super.setUp();

        (expectedHash, votingModuleData) =
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
        // This will generate a different proposal hash which will make the proposal type wrong
        vm.assume(_againstThreshold != againstThreshold);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidProposal.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(_againstThreshold, proposalDescription);
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_insufficientApprovals_reverts() public {
        // Set proposal data approved count to 0 since it is 1 by the approval on the setUp
        validator.setProposalData(expectedHash, approvedProposer, proposalType, false, 0, CYCLE_NUMBER);

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
        validator.setProposalData(expectedHash, approvedProposer, proposalType, true, 1, CYCLE_NUMBER);

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyMovedToVote.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(againstThreshold, proposalDescription);
    }

    function test_moveToVoteProtocolOrGovernorUpgradeProposal_proposalIdMismatch_reverts(bytes32 _randomHash) public {
        vm.assume(_randomHash != expectedHash);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(OPTIMISTIC_VOTING_MODULE_ID);

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (VotingModule(optimisticVotingModule), votingModuleData, proposalDescription, uint8(proposalType))
            ),
            abi.encode(uint256(_randomHash))
        );

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalIdMismatch.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteProtocolOrGovernorUpgradeProposal(againstThreshold, proposalDescription);
    }
}

contract ProposalValidator_MoveToVoteCouncilMemberElectionsProposal_Test is ProposalValidator_Init {
    ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.CouncilMemberElections;
    uint128 criteriaValue = 1;
    bytes32 expectedHash;
    bytes votingModuleData;
    string proposalDescription = "Test proposal";
    string[] optionsDescriptions = new string[](2);

    function setUp() public override {
        super.setUp();

        // Create a proposal for move to vote with 1 top choice and 2 options
        optionsDescriptions[0] = "Option 1";
        optionsDescriptions[1] = "Option 2";
        (expectedHash, votingModuleData) = _createCouncilElectionProposalForMoveToVote(
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
                (VotingModule(approvalVotingModule), votingModuleData, proposalDescription, uint8(proposalType))
            ),
            abi.encode(uint256(expectedHash))
        );

        // Expect the ProposalMovedToVote event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalMovedToVote(expectedHash, approvedProposer);

        // Move to vote
        vm.roll(START_BLOCK + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, proposalDescription);

        // Check that the proposal is in voting
        (,, bool movedToVote,,) = validator.getProposalData(expectedHash);
        assertTrue(movedToVote, "Proposal should be in voting");
    }
}

contract ProposalValidator_MoveToVoteCouncilMemberElectionsProposal_TestFail is ProposalValidator_Init {
    ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.CouncilMemberElections;
    uint128 criteriaValue = 1;
    string proposalDescription = "Test proposal";
    string[] optionsDescriptions = new string[](2);
    bytes32 expectedHash;
    bytes votingModuleData;

    function setUp() public override {
        super.setUp();

        // Create a proposal for move to vote with 1 top choice and 2 options
        optionsDescriptions[0] = "Option 1";
        optionsDescriptions[1] = "Option 2";
        (expectedHash, votingModuleData) = _createCouncilElectionProposalForMoveToVote(
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
        // This will generate a different proposal hash which will make the proposal type wrong
        uint128 _criteriaValue = 2; // we use 2 since it is the max based on the created proposal in setUp

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidProposal.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(_criteriaValue, optionsDescriptions, proposalDescription);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_insufficientApprovals_reverts() public {
        // Set proposal data approved count to 0 since it is 1 by the approval on the setUp
        validator.setProposalData(expectedHash, approvedProposer, proposalType, false, 0, CYCLE_NUMBER);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        vm.expectRevert(IProposalValidator.ProposalValidator_InsufficientApprovals.selector);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, proposalDescription);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_proposalAlreadyMovedToVote_reverts() public {
        // Set proposal data movedToVote to true
        validator.setProposalData(expectedHash, approvedProposer, proposalType, true, 2, CYCLE_NUMBER);

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
        vm.roll(block.number + DURATION + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteCouncilMemberElectionsProposal(criteriaValue, optionsDescriptions, proposalDescription);
    }

    function test_moveToVoteCouncilMemberElectionsProposal_proposalIdMismatch_reverts(bytes32 _randomHash) public {
        vm.assume(_randomHash != expectedHash);

        // Mock the proposal types configurator call
        _mockProposalTypesConfiguratorCall(APPROVAL_VOTING_MODULE_ID);

        // Mock the governor.proposeWithModule call
        _mockAndExpect(
            address(governor),
            abi.encodeCall(
                IOptimismGovernor.proposeWithModule,
                (VotingModule(approvalVotingModule), votingModuleData, proposalDescription, uint8(proposalType))
            ),
            abi.encode(uint256(_randomHash))
        );

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalIdMismatch.selector);
        vm.roll(START_BLOCK + 1);
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
    bytes32 expectedGovernanceFundHash;
    bytes32 expectedCouncilBudgetHash;
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
        (expectedGovernanceFundHash, governanceFundVotingModuleData) = _createFundingProposalForMoveToVote(
            approvedProposer,
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            governanceFundProposalDescription,
            governanceFundProposalType
        );
        (expectedCouncilBudgetHash, councilBudgetVotingModuleData) = _createFundingProposalForMoveToVote(
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
                    VotingModule(approvalVotingModule),
                    governanceFundVotingModuleData,
                    governanceFundProposalDescription,
                    uint8(governanceFundProposalType)
                )
            ),
            abi.encode(uint256(expectedGovernanceFundHash))
        );

        // Expect the ProposalMovedToVote event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalMovedToVote(expectedGovernanceFundHash, approvedProposer);

        // Move to vote
        vm.roll(START_BLOCK + 1);
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
        (,, bool movedToVote,,) = validator.getProposalData(expectedGovernanceFundHash);
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
                    VotingModule(approvalVotingModule),
                    councilBudgetVotingModuleData,
                    councilBudgetProposalDescription,
                    uint8(councilBudgetProposalType)
                )
            ),
            abi.encode(uint256(expectedCouncilBudgetHash))
        );

        // Expect the ProposalMovedToVote event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalMovedToVote(expectedCouncilBudgetHash, approvedProposer);

        // Move to vote
        vm.roll(START_BLOCK + 1);
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
    bytes32 governanceFundExpectedHash;
    bytes32 councilBudgetExpectedHash;
    bytes governanceFundVotingModuleData;
    bytes councilBudgetVotingModuleData;

    function setUp() public override {
        super.setUp();

        (optionsDescriptions, optionsRecipients, optionsAmounts) = _createMinimalFundingArrays();
        (governanceFundExpectedHash, governanceFundVotingModuleData) = _createFundingProposalForMoveToVote(
            approvedProposer,
            criteriaValue,
            optionsDescriptions,
            optionsRecipients,
            optionsAmounts,
            governanceFundProposalDescription,
            governanceFundProposalType
        );
        (councilBudgetExpectedHash, councilBudgetVotingModuleData) = _createFundingProposalForMoveToVote(
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
        // Ensure the criteria value is not the same as the one in setUp so when calculating the proposal hash it will
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
                governanceFundExpectedHash, approvedProposer, wrongProposalType, false, 0, CYCLE_NUMBER
            );
            proposalDescription = governanceFundProposalDescription;
        } else {
            // Set proposal data proposal type to a different value
            validator.setProposalData(
                councilBudgetExpectedHash, approvedProposer, wrongProposalType, false, 0, CYCLE_NUMBER
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
            validator.setProposalData(
                governanceFundExpectedHash, approvedProposer, proposalType, false, 0, CYCLE_NUMBER
            );
            proposalDescription = governanceFundProposalDescription;
        } else {
            // Set proposal data approved count to 0 since it is 1 by the approval on the setUp
            validator.setProposalData(councilBudgetExpectedHash, approvedProposer, proposalType, false, 0, CYCLE_NUMBER);
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
            validator.setProposalData(governanceFundExpectedHash, approvedProposer, proposalType, true, 1, CYCLE_NUMBER);
            proposalDescription = governanceFundProposalDescription;
        } else {
            // Set proposal data movedToVote to true
            validator.setProposalData(councilBudgetExpectedHash, approvedProposer, proposalType, true, 1, CYCLE_NUMBER);
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
        vm.roll(START_BLOCK + DURATION + 1);
        vm.prank(approvedProposer);
        validator.moveToVoteFundingProposal(
            criteriaValue, optionsDescriptions, optionsRecipients, optionsAmounts, proposalDescription, proposalType
        );
    }

    function test_moveToVoteFundingProposal_buildApprovalModuleOptionsExceedsDistributionThreshold_reverts(
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
        vm.roll(START_BLOCK + 1);
        validator.moveToVoteFundingProposal(
            criteriaValue, _optionsDescriptions, _optionsRecipients, _optionsAmounts, proposalDescription, proposalType
        );
    }

    function test_moveToVoteFundingProposal_proposalIdMismatch_reverts(
        uint8 _proposalTypeValue,
        bytes32 _randomHash
    )
        public
    {
        vm.assume(_randomHash != governanceFundExpectedHash && _randomHash != councilBudgetExpectedHash);

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
                (VotingModule(approvalVotingModule), votingModuleData, proposalDescription, uint8(proposalType))
            ),
            abi.encode(uint256(_randomHash))
        );

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalIdMismatch.selector);
        vm.roll(START_BLOCK + 1);
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
        uint256 startBlock,
        uint256 duration,
        uint256 distributionLimit
    )
        public
    {
        vm.assume(cycleNumber != CYCLE_NUMBER); // Avoid existing cycle

        // Expect the VotingCycleDataSet event to be emitted
        vm.expectEmit(address(validator));
        emit VotingCycleDataSet(cycleNumber, startBlock, duration, distributionLimit);

        vm.prank(owner);
        validator.setVotingCycleData(cycleNumber, startBlock, duration, distributionLimit);

        (
            uint256 actualStartBlock,
            uint256 actualDuration,
            uint256 actualDistributionLimit,
            uint256 actualMovedToVoteTokenCount
        ) = validator.votingCycles(cycleNumber);

        assertEq(actualStartBlock, startBlock);
        assertEq(actualDuration, duration);
        assertEq(actualDistributionLimit, distributionLimit);
        assertEq(actualMovedToVoteTokenCount, 0);
    }

    function testFuzz_setVotingCycleData_notOwner_reverts(
        address caller,
        uint256 cycleNumber,
        uint256 startBlock,
        uint256 duration,
        uint256 distributionLimit
    )
        public
    {
        vm.assume(caller != owner);

        vm.prank(caller);
        vm.expectRevert("Ownable: caller is not the owner");
        validator.setVotingCycleData(cycleNumber, startBlock, duration, distributionLimit);
    }

    function testFuzz_setVotingCycleData_votingCycleAlreadySet_reverts(
        uint256 startBlock,
        uint256 duration,
        uint256 distributionLimit
    )
        public
    {
        vm.expectRevert(ProposalValidator.ProposalValidator_VotingCycleAlreadySet.selector);
        vm.prank(owner);
        validator.setVotingCycleData(CYCLE_NUMBER, startBlock, duration, distributionLimit);
    }

    function testFuzz_setDistributionThreshold_succeeds(uint256 newDistributionThreshold) public {
        // Expect the DistributionThresholdSet event to be emitted
        vm.expectEmit(address(validator));
        emit DistributionThresholdSet(newDistributionThreshold);

        vm.prank(owner);
        validator.setDistributionThreshold(newDistributionThreshold);

        assertEq(validator.distributionThreshold(), newDistributionThreshold);
    }

    function testFuzz_setDistributionThreshold_notOwner_reverts(address caller, uint256 threshold) public {
        vm.assume(caller != owner);

        vm.prank(caller);
        vm.expectRevert("Ownable: caller is not the owner");
        validator.setDistributionThreshold(threshold);
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
            proposalVotingModule: newProposalTypeId
        });

        // Expect the ProposalTypeDataSet event to be emitted
        vm.expectEmit(address(validator));
        emit ProposalTypeDataSet(proposalType, newRequiredApprovals, newProposalTypeId);

        vm.prank(owner);
        validator.setProposalTypeData(proposalType, newData);

        (uint256 requiredApprovals, uint8 proposalVotingModule) = validator.proposalTypesData(proposalType);
        assertEq(requiredApprovals, newRequiredApprovals);
        assertEq(proposalVotingModule, newProposalTypeId);
    }

    function testFuzz_setProposalTypeData_notOwner_reverts(address caller) public {
        vm.assume(caller != owner);

        ProposalValidator.ProposalTypeData memory newData =
            ProposalValidator.ProposalTypeData({ requiredApprovals: 4, proposalVotingModule: 0 });

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
    {
        bytes32 hash = validator.hashProposalWithModule(module, proposalData, descriptionHash);
        bytes32 expectedHash =
            keccak256(abi.encode(address(validator.GOVERNOR()), module, proposalData, descriptionHash));

        assertEq(hash, expectedHash);
    }

    function test_hashProposalWithModule_differentInputs_succeeds() public {
        address module1 = makeAddr("module1");
        address module2 = makeAddr("module2");
        bytes memory data = abi.encode("data");
        bytes32 descHash = keccak256("desc");

        bytes32 hash1 = validator.hashProposalWithModule(module1, data, descHash);
        bytes32 hash2 = validator.hashProposalWithModule(module2, data, descHash);

        assertTrue(hash1 != hash2);
    }
}
