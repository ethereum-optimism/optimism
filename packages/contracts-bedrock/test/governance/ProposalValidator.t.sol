// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IProposalValidator } from "interfaces/governance/IProposalValidator.sol";
import { IOptimismGovernor } from "interfaces/governance/IOptimismGovernor.sol";
import { IEAS, AttestationRequest, AttestationRequestData } from "src/vendor/eas/IEAS.sol";
import { ISchemaRegistry, ISchemaResolver } from "src/vendor/eas/ISchemaRegistry.sol";

// Contracts
import { ProposalValidator } from "src/governance/ProposalValidator.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title ProposalValidator_Init
/// @notice Setup contract for ProposalValidator tests
contract ProposalValidator_Init is CommonTest {
    uint256 public constant TOP_DELEGATE_VOTING_POWER = 10000 ether; // 10k OP
    uint256 public constant VOTING_CYCLE_BLOCK = 100;
    uint256 public constant DISTRIBUTION_THRESHOLD = 10000 ether;
    uint256 public constant PROPOSAL_REQUIRED_APPROVALS = 4;
    uint256 public constant MINIMUM_VOTING_POWER = 10000 ether;

    address owner;
    address rando;
    address topDelegate_A;
    address topDelegate_B;
    address topDelegate_C;
    address topDelegate_D;

    ProposalValidator public validator;
    IOptimismGovernor public governor;
    bytes32 public ATTESTATION_SCHEMA_UID;

    /// @notice Helper function to setup a mock and expect a call to it.
    function _mockAndExpect(address _receiver, bytes memory _calldata, bytes memory _returned) internal {
        vm.mockCall(_receiver, _calldata, _returned);
        vm.expectCall(_receiver, _calldata);
    }

    /// @notice Helper function to make a top delegate.
    function _makeTopDelegate(string memory _name) internal returns (address) {
        address delegate = makeAddr(_name);
        deal(address(governanceToken), delegate, TOP_DELEGATE_VOTING_POWER);
        vm.prank(delegate);
        governanceToken.delegate(delegate);
        return delegate;
    }

    /// @notice Helper function to make a (top) delegate approve a proposal.
    function _approveProposal(address _delegate, uint256 _proposalId) internal {
        vm.prank(_delegate);
        validator.approveProposal(_proposalId);
    }

    function _getProposalTypesRequiredApprovalsAndImmutableData()
        internal
        pure
        returns (
            ProposalValidator.ProposalType[] memory,
            uint256[] memory,
            ProposalValidator.ImmutableProposalTypeData[] memory
        )
    {
        ProposalValidator.ProposalType[] memory proposalTypes = new ProposalValidator.ProposalType[](5);
        proposalTypes[0] = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        proposalTypes[1] = ProposalValidator.ProposalType.MaintenanceUpgrade;
        proposalTypes[2] = ProposalValidator.ProposalType.CouncilMemberElections;
        proposalTypes[3] = ProposalValidator.ProposalType.GovernanceFund;
        proposalTypes[4] = ProposalValidator.ProposalType.CouncilBudget;

        uint256[] memory requiredApprovals = new uint256[](5);
        requiredApprovals[0] = PROPOSAL_REQUIRED_APPROVALS;
        requiredApprovals[1] = PROPOSAL_REQUIRED_APPROVALS;
        requiredApprovals[2] = PROPOSAL_REQUIRED_APPROVALS;
        requiredApprovals[3] = PROPOSAL_REQUIRED_APPROVALS;
        requiredApprovals[4] = PROPOSAL_REQUIRED_APPROVALS;

        ProposalValidator.ImmutableProposalTypeData[] memory immutableProposalTypeData =
            new ProposalValidator.ImmutableProposalTypeData[](5);
        immutableProposalTypeData[0] = ProposalValidator.ImmutableProposalTypeData({
            targets: new address[](1),
            values: new uint256[](1),
            signatures: new string[](1)
        });

        return (proposalTypes, requiredApprovals, immutableProposalTypeData);
    }

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        owner = governanceToken.owner();
        rando = makeAddr("rando");
        governor = IOptimismGovernor(makeAddr("governor"));

        vm.prank(owner);
        ATTESTATION_SCHEMA_UID = ISchemaRegistry(Predeploys.SCHEMA_REGISTRY).register(
            "address approvedAddress,uint8 proposalType", ISchemaResolver(address(0)), false
        );

        (
            ProposalValidator.ProposalType[] memory proposalTypes,
            uint256[] memory requiredApprovals,
            ProposalValidator.ImmutableProposalTypeData[] memory immutableProposalTypeData
        ) = _getProposalTypesRequiredApprovalsAndImmutableData();

        validator = new ProposalValidator(
            owner,
            governor,
            governanceToken,
            ATTESTATION_SCHEMA_UID,
            MINIMUM_VOTING_POWER,
            VOTING_CYCLE_BLOCK,
            DISTRIBUTION_THRESHOLD,
            proposalTypes,
            requiredApprovals,
            immutableProposalTypeData
        );

        topDelegate_A = _makeTopDelegate("topDelegate_A");
        topDelegate_B = _makeTopDelegate("topDelegate_B");
        topDelegate_C = _makeTopDelegate("topDelegate_C");
        topDelegate_D = _makeTopDelegate("topDelegate_D");
    }

    /// @notice Helper to create a valid attestation for a proposal
    function _createAttestation(
        address _delegate,
        ProposalValidator.ProposalType _proposalType
    )
        internal
        returns (bytes32)
    {
        vm.prank(owner);
        return IEAS(Predeploys.EAS).attest(
            AttestationRequest({
                schema: ATTESTATION_SCHEMA_UID,
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

/// @title ProposalValidator_SubmitProposal_Test
/// @notice Happy path tests for submitProposal function
contract ProposalValidator_SubmitProposal_Test is ProposalValidator_Init {
    function test_submitProposal_succeeds() public {
        (address[] memory _targets, uint256[] memory _values, bytes[] memory _calldatas, string memory _description) =
            _createProposalSetup();

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 attestationUid = _createAttestation(topDelegate_A, proposalType);

        vm.prank(topDelegate_A);
        uint256 proposalId =
            validator.submitProposal(_targets, _values, _calldatas, _description, proposalType, attestationUid);

        assertEq(proposalId, 1);
    }
}

/// @title ProposalValidator_SubmitProposal_TestFail
/// @notice Sad path tests for submitProposal function
contract ProposalValidator_SubmitProposal_TestFail is ProposalValidator_Init {
    function test_submitProposal_invalidAttestation_reverts() public {
        (address[] memory targets, uint256[] memory values, bytes[] memory calldatas, string memory description) =
            _createProposalSetup();

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 invalidAttestationUid = bytes32(uint256(1)); // Invalid attestation UID

        vm.prank(topDelegate_A);
        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestation.selector);
        validator.submitProposal(targets, values, calldatas, description, proposalType, invalidAttestationUid);
    }

    function test_submitProposal_wrongAttester_reverts() public {
        (address[] memory targets, uint256[] memory values, bytes[] memory calldatas, string memory description) =
            _createProposalSetup();

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;

        // Create attestation with wrong delegate
        bytes32 attestationUid = _createAttestation(topDelegate_B, proposalType);

        vm.prank(topDelegate_A);
        vm.expectRevert(IProposalValidator.ProposalValidator_InvalidAttestation.selector);
        validator.submitProposal(targets, values, calldatas, description, proposalType, attestationUid);
    }
}

/// @title ProposalValidator_ApproveProposal_Test
/// @notice Happy path tests for approveProposal function
contract ProposalValidator_ApproveProposal_Test is ProposalValidator_Init {
    uint256 proposalId;

    function setUp() public override {
        super.setUp();

        (address[] memory targets, uint256[] memory values, bytes[] memory calldatas, string memory description) =
            _createProposalSetup();

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 attestationUid = _createAttestation(topDelegate_A, proposalType);

        vm.prank(topDelegate_A);
        proposalId = validator.submitProposal(targets, values, calldatas, description, proposalType, attestationUid);
    }

    function test_approveProposal_succeeds() public {
        _approveProposal(topDelegate_A, proposalId);
        _approveProposal(topDelegate_B, proposalId);
        _approveProposal(topDelegate_C, proposalId);
        _approveProposal(topDelegate_D, proposalId);
    }
}

/// @title ProposalValidator_ApproveProposal_TestFail
/// @notice Sad path tests for approveProposal function
contract ProposalValidator_ApproveProposal_TestFail is ProposalValidator_Init {
    uint256 proposalId;

    function setUp() public override {
        super.setUp();

        (address[] memory targets, uint256[] memory values, bytes[] memory calldatas, string memory description) =
            _createProposalSetup();

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 attestationUid = _createAttestation(topDelegate_A, proposalType);

        vm.prank(topDelegate_A);
        proposalId = validator.submitProposal(targets, values, calldatas, description, proposalType, attestationUid);
    }

    function test_approveProposal_insufficientVotingPower_reverts() public {
        vm.expectRevert(IProposalValidator.ProposalValidator_InsufficientVotingPower.selector);
        _approveProposal(rando, proposalId);
    }

    function test_approveProposal_alreadyApproved_reverts() public {
        _approveProposal(topDelegate_A, proposalId);

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyApproved.selector);
        _approveProposal(topDelegate_A, proposalId);
    }
}

/// @title ProposalValidator_MoveToVote_Test
/// @notice Happy path tests for moveToVote function
contract ProposalValidator_MoveToVote_Test is ProposalValidator_Init {
    uint256 proposalId;
    address[] targets;
    uint256[] values;
    bytes[] calldatas;
    string description;
    ProposalValidator.ProposalType proposalType;

    function setUp() public override {
        super.setUp();

        (targets, values, calldatas, description) = _createProposalSetup();

        proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 attestationUid = _createAttestation(topDelegate_A, proposalType);

        vm.prank(topDelegate_A);
        proposalId = validator.submitProposal(targets, values, calldatas, description, proposalType, attestationUid);

        _approveProposal(topDelegate_A, proposalId);
        _approveProposal(topDelegate_B, proposalId);
        _approveProposal(topDelegate_C, proposalId);
        _approveProposal(topDelegate_D, proposalId);
    }

    function test_moveToVote_succeeds() public {
        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.propose, (targets, values, calldatas, description, uint8(proposalType))),
            abi.encode(1)
        );

        vm.prank(owner);
        uint256 governorProposalId = validator.moveToVote(proposalId);

        assertEq(governorProposalId, 1);
    }
}

/// @title ProposalValidator_MoveToVote_TestFail
/// @notice Sad path tests for moveToVote function
contract ProposalValidator_MoveToVote_TestFail is ProposalValidator_Init {
    uint256 proposalId;
    address[] targets;
    uint256[] values;
    bytes[] calldatas;
    string description;
    ProposalValidator.ProposalType proposalType;

    function setUp() public override {
        super.setUp();

        (targets, values, calldatas, description) = _createProposalSetup();

        proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 attestationUid = _createAttestation(topDelegate_A, proposalType);

        vm.prank(topDelegate_A);
        proposalId = validator.submitProposal(targets, values, calldatas, description, proposalType, attestationUid);
    }

    function test_moveToVote_insufficientApprovals_reverts() public {
        // Only approve with 3 delegates (need 4)
        _approveProposal(topDelegate_A, proposalId);
        _approveProposal(topDelegate_B, proposalId);
        _approveProposal(topDelegate_C, proposalId);

        vm.expectRevert(IProposalValidator.ProposalValidator_InsufficientApprovals.selector);
        vm.prank(owner);
        validator.moveToVote(proposalId);
    }

    function test_moveToVote_alreadyProposed_reverts() public {
        // Approve with all 4 delegates
        _approveProposal(topDelegate_A, proposalId);
        _approveProposal(topDelegate_B, proposalId);
        _approveProposal(topDelegate_C, proposalId);
        _approveProposal(topDelegate_D, proposalId);

        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.propose, (targets, values, calldatas, description, uint8(proposalType))),
            abi.encode(1)
        );

        vm.prank(owner);
        validator.moveToVote(proposalId);

        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyInVoting.selector);
        vm.prank(owner);
        validator.moveToVote(proposalId);
    }
}

/// @title ProposalValidator_Getters_Test
/// @notice Tests for getter functions
contract ProposalValidator_Getters_Test is ProposalValidator_Init {
    function test_canSignOff_succeeds() public {
        bool canSignOff = validator.canSignOff(topDelegate_A);
        assertTrue(canSignOff);

        bool cannotSignOff = validator.canSignOff(rando);
        assertFalse(cannotSignOff);
    }
}

/// @title ProposalValidator_Setters_Test
/// @notice Tests for setter functions
contract ProposalValidator_Setters_Test is ProposalValidator_Init {
// TODO: Implement tests for setters
}

/// @title ProposalValidator_Integration_Test
/// @notice Integration tests for the full proposal flow
contract ProposalValidator_Integration_Test is ProposalValidator_Init {
    function test_proposalFullFlow_succeeds() public {
        // Create a proposal
        (address[] memory targets, uint256[] memory values, bytes[] memory calldatas, string memory description) =
            _createProposalSetup();

        ProposalValidator.ProposalType proposalType = ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade;
        bytes32 attestationUid = _createAttestation(topDelegate_A, proposalType);

        vm.prank(topDelegate_A);
        uint256 proposalId =
            validator.submitProposal(targets, values, calldatas, description, proposalType, attestationUid);

        assertEq(proposalId, 1);

        // It reverts when caller is not a top delegate
        vm.expectRevert(IProposalValidator.ProposalValidator_InsufficientVotingPower.selector);
        _approveProposal(rando, proposalId);

        _approveProposal(topDelegate_A, proposalId);
        _approveProposal(topDelegate_B, proposalId);
        _approveProposal(topDelegate_C, proposalId);

        // It reverts when proposal hasn't reached the required approvals
        vm.expectRevert(IProposalValidator.ProposalValidator_InsufficientApprovals.selector);
        vm.prank(owner);
        validator.moveToVote(proposalId);

        _approveProposal(topDelegate_D, proposalId);

        _mockAndExpect(
            address(governor),
            abi.encodeCall(IOptimismGovernor.propose, (targets, values, calldatas, description, uint8(proposalType))),
            abi.encode(1)
        );

        vm.prank(owner);
        validator.moveToVote(proposalId);

        // It reverts when proposal is already in voting phase
        vm.expectRevert(IProposalValidator.ProposalValidator_ProposalAlreadyInVoting.selector);
        vm.prank(owner);
        validator.moveToVote(proposalId);
    }
}
