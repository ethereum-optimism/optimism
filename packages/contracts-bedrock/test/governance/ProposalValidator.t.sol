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
import { ProposalSettings, ProposalOption, PassingCriteria } from "src/governance/ApprovalVotingModule.sol";

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
        returns (address proposer_, ProposalType proposalType_, bool inVoting_, uint256 approvalCount_)
    {
        ProposalData storage proposal = _proposals[_proposalHash];
        return (proposal.proposer, proposal.proposalType, proposal.inVoting, proposal.approvalCount);
    }

    function setProposalData(
        bytes32 _proposalHash,
        address _proposer,
        ProposalType _proposalType,
        bool _inVoting,
        uint256 _approvalCount
    )
        public
    {
        _proposals[_proposalHash].proposer = _proposer;
        _proposals[_proposalHash].proposalType = _proposalType;
        _proposals[_proposalHash].inVoting = _inVoting;
        _proposals[_proposalHash].approvalCount = _approvalCount;
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
    uint256 public constant DISTRIBUTION_LIMIT = 10000 ether;
    uint256 public constant DISTRIBUTION_THRESHOLD = 10000 ether;
    uint256 public constant PROPOSAL_REQUIRED_APPROVALS = 4;
    uint8 public constant APPROVAL_VOTING_MODULE_ID = 3;

    address owner;
    address user;
    address topDelegate_A = makeAddr("topDelegate_A");
    address topDelegate_B = makeAddr("topDelegate_B");
    address topDelegate_C = makeAddr("topDelegate_C");
    address topDelegate_D = makeAddr("topDelegate_D");
    bytes32 topDelegateAttestation_A;
    bytes32 topDelegateAttestation_B;
    bytes32 topDelegateAttestation_C;
    bytes32 topDelegateAttestation_D;
    address approvalVotingModule;

    ProposalValidatorForTest public validator;
    ProposalValidatorForTest public impl;
    IOptimismGovernor public governor;
    IProposalTypesConfigurator public proposalTypesConfigurator;
    bytes32 public APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID;
    bytes32 public TOP_DELEGATES_ATTESTATION_SCHEMA_UID;
    bytes32 public proposalHash;

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

    /// @notice Helper function to set both funding proposal types.
    function _setFundingProposalTypes() internal {
        _setGovernanceFundProposalType();
        _setCouncilBudgetProposalType();
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
        proposalTypesData[0] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            proposalVotingModule: 0
        });
        proposalTypesData[1] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            proposalVotingModule: 1
        });
        proposalTypesData[2] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            proposalVotingModule: 2
        });
        proposalTypesData[3] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            proposalVotingModule: APPROVAL_VOTING_MODULE_ID
        });
        proposalTypesData[4] = ProposalValidator.ProposalTypeData({
            requiredApprovals: PROPOSAL_REQUIRED_APPROVALS,
            proposalVotingModule: APPROVAL_VOTING_MODULE_ID
        });

        return (proposalTypes, proposalTypesData);
    }

    function _constructVotingModuleData(
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
        ProposalSettings memory settings = ProposalSettings({
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
        ProposalSettings memory settings = ProposalSettings({
            maxApprovals: uint8(descriptions.length),
            criteria: uint8(PassingCriteria.TopChoices),
            budgetToken: address(0),
            criteriaValue: criteriaValue,
            budgetAmount: 0
        });

        return abi.encode(options, settings);
    }

    /// @notice Helper function to setup proposal types configurator mocks
    function _setupProposalTypesConfiguratorMocks() internal {
        // Mock calls for different proposal type IDs
        for (uint8 i = 0; i < 5; i++) {
            address moduleAddress = (i == 2 || i == APPROVAL_VOTING_MODULE_ID) ? approvalVotingModule : address(0);

            vm.mockCall(
                address(proposalTypesConfigurator),
                abi.encodeCall(IProposalTypesConfigurator.proposalTypes, (i)),
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
    }

    /// @notice Initializes the validator
    function _initializeValidator() internal virtual {
        (
            ProposalValidator.ProposalType[] memory proposalTypes,
            ProposalValidator.ProposalTypeData[] memory proposalTypesData
        ) = _getProposalTypesAndData();

        // Create mock addresses
        proposalTypesConfigurator = IProposalTypesConfigurator(makeAddr("proposalTypesConfigurator"));

        // Setup mocks
        _setupProposalTypesConfiguratorMocks();

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

        // Create schemas
        vm.prank(owner);
        APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID = ISchemaRegistry(Predeploys.SCHEMA_REGISTRY).register(
            "address approvedAddress,uint8 proposalType", ISchemaResolver(address(0)), false
        );

        vm.prank(owner);
        TOP_DELEGATES_ATTESTATION_SCHEMA_UID = ISchemaRegistry(Predeploys.SCHEMA_REGISTRY).register(
            "string top100,bool includePartialDelegation,string date", ISchemaResolver(address(0)), true
        );

        _initializeValidator();

        // Create attestations for top delegates
        topDelegateAttestation_A = _createTopDelegateAttestation(topDelegate_A);
        topDelegateAttestation_B = _createTopDelegateAttestation(topDelegate_B);
        topDelegateAttestation_C = _createTopDelegateAttestation(topDelegate_C);
        topDelegateAttestation_D = _createTopDelegateAttestation(topDelegate_D);
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
                    revocable: false,
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

    /// @notice Helper to create a standard proposal setup
    function _createProposalSetup()
        internal
        view
        returns (
            address[] memory targets_,
            uint256[] memory values_,
            bytes[] memory calldatas_,
            string memory description_
        )
    {
        targets_ = new address[](1);
        targets_[0] = address(0);
        values_ = new uint256[](1);
        values_[0] = 0;
        calldatas_ = new bytes[](1);
        calldatas_[0] = bytes("");
        description_ = "Test proposal";
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
        validator.setProposalData(_proposalHash, topDelegate_A, proposalType, false, 0);

        // Expect event to be emitted when approving
        vm.expectEmit(address(validator));
        emit ProposalApproved(_proposalHash, topDelegate_A);

        // Approve the proposal, use the attestation of the top delegate that was created in setUp
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalHash, topDelegateAttestation_A);

        // Check that the proposal data has been updated
        assertTrue(validator.hasDelegateApproved(_proposalHash, topDelegate_A));

        (,,, uint256 approvalCount) = validator.getProposalData(_proposalHash);
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
        validator.setProposalData(_proposalHash, topDelegate_A, proposalType, false, 0);

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
        validator.setProposalData(_proposalHash, topDelegate_A, proposalType, false, 0);

        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestationSchema.selector);
        vm.prank(topDelegate_A);
        validator.approveProposal(_proposalHash, _invalidAttestationUid);
    }

    function test_approveProposal_attestationRevoked_reverts(bytes32 _proposalHash, uint8 proposalTypeValue) public {
        // Bound the proposal type to valid enum values (0-4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 4));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);
        // set proposal data so that the proposal exists
        validator.setProposalData(_proposalHash, topDelegate_A, proposalType, false, 0);

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
        vm.assume(
            _caller != topDelegate_A && _caller != topDelegate_B && _caller != topDelegate_C && _caller != topDelegate_D
        );

        // Set mock proposal data of a random proposal in the validator contract
        validator.setProposalData(_proposalHash, topDelegate_A, proposalType, false, 0);

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
        validator.setProposalData(_proposalHash, topDelegate_A, proposalType, false, 0);

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
}

// /// @title ProposalValidator_MoveToVote_Test
// /// @notice Happy path tests for moveToVote function
// contract ProposalValidator_MoveToVote_Test is ProposalValidator_Init {
//     address[] targets;
//     uint256[] values;
//     bytes[] calldatas;
//     string description;
//     ProposalValidator.ProposalType proposalType;
//     uint8 proposalVotingModule;

//     function setUp() public override {
//         super.setUp();

//         (targets, values, calldatas, description) = _createProposalSetup();

//         proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
//         proposalVotingModule = 0;
//         bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

//         /* vm.prank(topDelegate_A); */
//         proposalHash = bytes32(0); // TODO: Implement after submitFundingProposal is implemented

//         _approveProposal(topDelegate_A, proposalHash);
//         _approveProposal(topDelegate_B, proposalHash);
//         _approveProposal(topDelegate_C, proposalHash);
//         _approveProposal(topDelegate_D, proposalHash);
//     }

//     function test_moveToVote_succeeds() public {
//         _mockAndExpect(
//             address(governor),
//             abi.encodeCall(IOptimismGovernor.propose, (targets, values, calldatas, description,
// proposalVotingModule)),
//             abi.encode(1)
//         );

//         // Expect the ProposalMovedToVote event to be emitted
//         vm.expectEmit(address(validator));
//         emit ProposalMovedToVote(proposalHash, owner);

//         vm.prank(owner);
//         uint256 governorProposalId = validator.moveToVote(targets, values, calldatas, description);

//         assertEq(governorProposalId, 1);
//     }
// }

// /// @title ProposalValidator_MoveToVote_TestFail
// /// @notice Sad path tests for moveToVote function
// contract ProposalValidator_MoveToVote_TestFail is ProposalValidator_Init {
//     address[] targets;
//     uint256[] values;
//     bytes[] calldatas;
//     string description;
//     ProposalValidator.ProposalType proposalType;
//     uint8 proposalVotingModule;

//     function setUp() public override {
//         super.setUp();

//         (targets, values, calldatas, description) = _createProposalSetup();

//         proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
//         proposalVotingModule = 0;
//         bytes32 attestationUid = _createApprovedProposerAttestation(topDelegate_A, proposalType);

//         /* vm.prank(topDelegate_A); */
//         proposalHash = bytes32(0); // TODO: Implement after submitFundingProposal is implemented
//     }

//     function test_moveToVote_insufficientApprovals_reverts() public {
//         // Only approve with 3 delegates (need 4)
//         _approveProposal(topDelegate_A, proposalHash);
//         _approveProposal(topDelegate_B, proposalHash);
//         _approveProposal(topDelegate_C, proposalHash);

//         vm.expectRevert(IProposalValidator.ProposalValidator_InsufficientApprovals.selector);
//         vm.prank(owner);
//         validator.moveToVote(targets, values, calldatas, description);
//     }

//     function test_moveToVote_alreadyProposed_reverts() public {
//         // Approve with all 4 delegates
//         _approveProposal(topDelegate_A, proposalHash);
//         _approveProposal(topDelegate_B, proposalHash);
//         _approveProposal(topDelegate_C, proposalHash);
//         _approveProposal(topDelegate_D, proposalHash);

//         _mockAndExpect(
//             address(governor),
//             abi.encodeCall(IOptimismGovernor.propose, (targets, values, calldatas, description,
// proposalVotingModule)),
//             abi.encode(1)
//         );

//         vm.prank(owner);
//         validator.moveToVote(targets, values, calldatas, description);

//         vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadySubmitted.selector);
//         vm.prank(owner);
//         validator.moveToVote(targets, values, calldatas, description);
//     }
// }

/// @title ProposalValidator_CanApproveProposal_Test
/// @notice Tests for the canApproveProposal function
contract ProposalValidator_CanApproveProposal_Test is ProposalValidator_Init {
    function test_canApproveProposal_ReturnsTrue_succeeds() public {
        // Attestation already created in setUp
        bool canApprove = validator.canApproveProposal(topDelegateAttestation_A, topDelegate_A);
        assertTrue(canApprove);
    }

    function test_canApproveProposal_ReturnsFalse_succeeds(bytes32 _attestationUid, address _delegate) public {
        // Ensure the attestation uid is not one of the top delegates
        vm.assume(
            _attestationUid != topDelegateAttestation_A && _attestationUid != topDelegateAttestation_B
                && _attestationUid != topDelegateAttestation_C && _attestationUid != topDelegateAttestation_D
        );

        bool canApprove;
        // Expect the invalid attestation error to be reverted
        vm.expectRevert();
        try validator.canApproveProposal(_attestationUid, _delegate) returns (bool result) {
            canApprove = result;
        } catch {
            canApprove = false;
        }

        assertEq(canApprove, false);
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

        (uint256 actualStartBlock, uint256 actualDuration, uint256 actualDistributionLimit) =
            validator.votingCycles(cycleNumber);

        assertEq(actualStartBlock, startBlock);
        assertEq(actualDuration, duration);
        assertEq(actualDistributionLimit, distributionLimit);
    }

    function test_setVotingCycleData_notOwner_reverts() public {
        vm.prank(user);
        vm.expectRevert("Ownable: caller is not the owner");
        validator.setVotingCycleData(2, block.number, 100, 10000 ether);
    }

    function test_setVotingCycleData_votingCycleAlreadySet_reverts() public {
        vm.prank(owner);
        validator.setVotingCycleData(2, block.number, 100, 10000 ether);

        vm.expectRevert(ProposalValidator.ProposalValidator_VotingCycleAlreadySet.selector);
        vm.prank(owner);
        validator.setVotingCycleData(2, block.number, 100, 10000 ether);
    }

    function testFuzz_setDistributionThreshold_succeeds(uint256 newDistributionThreshold) public {
        // Expect the DistributionThresholdSet event to be emitted
        vm.expectEmit(address(validator));
        emit DistributionThresholdSet(newDistributionThreshold);

        vm.prank(owner);
        validator.setDistributionThreshold(newDistributionThreshold);

        assertEq(validator.distributionThreshold(), newDistributionThreshold);
    }

    function test_setDistributionThreshold_notOwner_reverts() public {
        vm.prank(user);
        vm.expectRevert("Ownable: caller is not the owner");
        validator.setDistributionThreshold(10000 ether);
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

    function test_setProposalTypeData_notOwner_reverts() public {
        ProposalValidator.ProposalTypeData memory newData =
            ProposalValidator.ProposalTypeData({ requiredApprovals: 4, proposalVotingModule: 0 });

        vm.prank(user);
        vm.expectRevert("Ownable: caller is not the owner");
        validator.setProposalTypeData(ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade, newData);
    }
}

/// @title ProposalValidator_HashProposalWithModule_Test
/// @notice Tests for the hashProposalWithModule function
contract ProposalValidator_HashProposalWithModule_Test is ProposalValidator_Init {
    function test_hashProposalWithModule_succeeds() public {
        address testModule = makeAddr("testModule");
        bytes memory testProposalData = abi.encode("test", "proposal", "data");
        bytes32 testDescriptionHash = keccak256("test description");

        bytes32 hash = validator.hashProposalWithModule(testModule, testProposalData, testDescriptionHash);
        assertTrue(hash != bytes32(0));
    }

    function test_hashProposalWithModule_consistentHash_succeeds() public {
        address testModule = makeAddr("testModule");
        bytes memory testProposalData = abi.encode("test data");
        bytes32 testDescriptionHash = keccak256("description");

        bytes32 hash1 = validator.hashProposalWithModule(testModule, testProposalData, testDescriptionHash);
        bytes32 hash2 = validator.hashProposalWithModule(testModule, testProposalData, testDescriptionHash);

        assertEq(hash1, hash2);
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

/// @title ProposalValidator_SubmitFundingProposal_Test
/// @notice Happy path tests for submitFundingProposal function
contract ProposalValidator_SubmitFundingProposal_Test is ProposalValidator_Init {
    uint128 criteriaValue;
    string[] optionsDescriptions;
    address[] optionsRecipients;
    uint256[] optionsAmounts;
    string description;

    event ProposalVotingModuleData(bytes32 indexed proposalHash, bytes encodedVotingModuleData);

    function setUp() public override {
        super.setUp();

        _setFundingProposalTypes();

        criteriaValue = 1000 ether;
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

        // Bound option count between 1 and 50 for reasonable test execution
        optionCount = uint8(bound(optionCount, 1, 50));

        // Bound amount from 0 to DISTRIBUTION_THRESHOLD (inclusive)
        amount = bound(amount, 0, DISTRIBUTION_THRESHOLD);

        // Create arrays based on option count
        string[] memory descriptions = new string[](optionCount);
        address[] memory recipients = new address[](optionCount);
        uint256[] memory amounts = new uint256[](optionCount);

        for (uint256 i = 0; i < optionCount; i++) {
            descriptions[i] = string(abi.encodePacked("Option ", vm.toString(i)));
            recipients[i] = makeAddr(string(abi.encodePacked("recipient", vm.toString(i))));
            amounts[i] = amount; // Use the same bounded amount for all options
        }

        // Calculate expected proposal hash
        bytes memory votingModuleData = _constructVotingModuleData(descriptions, recipients, amounts, criteriaValue);
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

        vm.prank(proposer);
        bytes32 proposalHash =
            validator.submitFundingProposal(criteriaValue, descriptions, recipients, amounts, description, proposalType);

        assertEq(proposalHash, expectedHash);

        // Verify proposal data was stored correctly
        (
            address storedProposer,
            ProposalValidator.ProposalType storedProposalType,
            bool inVoting,
            uint256 approvalCount
        ) = validator.getProposalData(proposalHash);

        assertEq(storedProposer, proposer, "Proposer should match input");
        assertEq(uint8(storedProposalType), uint8(proposalType), "Proposal type should match input");
        assertFalse(inVoting, "Proposal should not be in voting yet");
        assertEq(approvalCount, 0, "Approval count should be 0");
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
        _setFundingProposalTypes();
    }

    function testFuzz_submitFundingProposal_invalidProposalType_reverts(uint8 proposalTypeValue) public {
        // Bound to proposal types that are NOT funding proposals (0, 1, 2)
        // Valid funding proposal types are GovernanceFund (3) and CouncilBudget (4)
        proposalTypeValue = uint8(bound(proposalTypeValue, 0, 2));
        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType(proposalTypeValue);

        (string[] memory descriptions, address[] memory recipients, uint256[] memory amounts) =
            _createMinimalFundingArrays();

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidFundingProposalType.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, description, proposalType
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
            proposalType
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
            proposalType
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
            proposalType
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
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, description, proposalType
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
            _constructVotingModuleData(descriptions, recipients, amounts, FUNDING_CRITERIA_VALUE);
        bytes32 expectedHash =
            validator.hashProposalWithModule(approvalVotingModule, votingModuleData, keccak256(bytes(description)));

        // Mock proposalSnapshot to return 0 for first submission
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (uint256(expectedHash))),
            abi.encode(0)
        );

        // Submit first proposal
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, description, proposalType
        );

        // Attempt to submit identical proposal
        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalAlreadySubmitted.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, description, proposalType
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
            _constructVotingModuleData(descriptions, recipients, amounts, FUNDING_CRITERIA_VALUE);
        bytes32 expectedHash =
            validator.hashProposalWithModule(approvalVotingModule, votingModuleData, keccak256(bytes(description)));

        // Mock proposalSnapshot to return non-zero (proposal already exists in governor)
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.proposalSnapshot, (uint256(expectedHash))),
            abi.encode(1000) // Non-zero indicates proposal exists
        );

        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalAlreadySubmitted.selector);
        vm.prank(user);
        validator.submitFundingProposal(
            FUNDING_CRITERIA_VALUE, descriptions, recipients, amounts, description, proposalType
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
            FUNDING_CRITERIA_VALUE, emptyDescriptions, emptyRecipients, emptyAmounts, description, proposalType
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
            FUNDING_CRITERIA_VALUE, tooManyDescriptions, tooManyRecipients, tooManyAmounts, description, proposalType
        );
    }
}

/// @title ProposalValidator_Initialize_Test
/// @notice Tests for the initialize function
contract ProposalValidator_Initialize_Test is ProposalValidator_Init {
    /// @dev Override to create validator proxy without initialization for testing
    function _initializeValidator() internal override {
        // Create mock addresses
        proposalTypesConfigurator = IProposalTypesConfigurator(makeAddr("proposalTypesConfigurator"));

        // Setup mocks
        _setupProposalTypesConfiguratorMocks();

        impl = new ProposalValidatorForTest(
            APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID, TOP_DELEGATES_ATTESTATION_SCHEMA_UID, governor, governanceToken
        );
        validator = ProposalValidatorForTest(address(new Proxy(owner)));
        // Initialize will be tested manually
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
        (uint256 startBlock, uint256 duration, uint256 distributionLimit) = validator.votingCycles(CYCLE_NUMBER);
        assertEq(startBlock, START_BLOCK);
        assertEq(duration, DURATION);
        assertEq(distributionLimit, DISTRIBUTION_LIMIT);

        // Verify proposal type data
        for (uint256 i = 0; i < proposalTypes.length; i++) {
            (uint256 requiredApprovals, uint8 proposalVotingModule) = validator.proposalTypesData(proposalTypes[i]);
            assertEq(requiredApprovals, PROPOSAL_REQUIRED_APPROVALS);

            // Both GovernanceFund and CouncilBudget use APPROVAL_VOTING_MODULE_ID
            if (
                proposalTypes[i] == ProposalValidator.ProposalType.GovernanceFund
                    || proposalTypes[i] == ProposalValidator.ProposalType.CouncilBudget
            ) {
                assertEq(proposalVotingModule, APPROVAL_VOTING_MODULE_ID);
            } else {
                assertEq(proposalVotingModule, uint8(i));
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

/// @title ProposalValidator_SubmitCouncilMemberElectionsProposal_Test
/// @notice Happy path tests for submitCouncilMemberElectionsProposal function
contract ProposalValidator_SubmitCouncilMemberElectionsProposal_Test is ProposalValidator_Init {
    string proposalDescription;

    event ProposalVotingModuleData(bytes32 indexed proposalHash, bytes encodedVotingModuleData);

    function setUp() public override {
        super.setUp();

        _setCouncilMemberElectionsProposalType();

        proposalDescription = "Council Member Elections Q4 2024";
    }

    function testFuzz_submitCouncilMemberElectionsProposal_succeeds(uint8 optionCount, uint128 criteriaValue) public {
        optionCount = uint8(bound(optionCount, 2, type(uint8).max)); // Minimum 2 options to have valid criteria <
            // optionCount
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

        vm.prank(topDelegate_A);
        bytes32 proposalHash = validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, attestationUid
        );

        assertEq(proposalHash, expectedHash);

        // Verify proposal data was stored correctly
        (address proposer, ProposalValidator.ProposalType proposalType, bool inVoting, uint256 approvalCount) =
            validator.getProposalData(proposalHash);

        assertEq(proposer, topDelegate_A, "Proposer should be topDelegate_A");
        assertEq(
            uint8(proposalType),
            uint8(ProposalValidator.ProposalType.CouncilMemberElections),
            "Proposal type should be CouncilMemberElections"
        );
        assertFalse(inVoting, "Proposal should not be in voting yet");
        assertEq(approvalCount, 0, "Approval count should be 0");
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
            criteriaValue, optionDescriptions, proposalDescription, fuzzedAttestationUid
        );
    }

    function testFuzz_submitCouncilMemberElectionsProposal_unattestedProposer_reverts(address fuzzedProposer) public {
        vm.assume(fuzzedProposer != topDelegate_A); // Ensure it's different from attested proposer

        // Try to submit with different address than attested
        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidAttestation.selector);
        vm.prank(fuzzedProposer); // Different from attested topDelegate_A
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, attestationUid
        );
    }

    function test_submitCouncilMemberElectionsProposal_zeroOptions_reverts() public {
        string[] memory emptyOptions = new string[](0);

        vm.expectRevert(ProposalValidator.ProposalValidator_InvalidOptionsLength.selector);
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(criteriaValue, emptyOptions, proposalDescription, attestationUid);
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

        // Submit first proposal
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, attestationUid
        );

        // Create new attestation for second attempt
        bytes32 secondAttestation =
            _createApprovedProposerAttestation(topDelegate_B, ProposalValidator.ProposalType.CouncilMemberElections);

        // Attempt to submit identical proposal should revert
        vm.expectRevert(ProposalValidator.ProposalValidator_ProposalAlreadySubmitted.selector);
        vm.prank(topDelegate_B);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, secondAttestation
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
        vm.prank(topDelegate_A);
        validator.submitCouncilMemberElectionsProposal(
            criteriaValue, optionDescriptions, proposalDescription, attestationUid
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
            criteriaValue, optionDescriptions, proposalDescription, invalidAttestation
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
            invalidCriteriaValue, optionDescriptions, proposalDescription, attestationUid
        );
    }
}
