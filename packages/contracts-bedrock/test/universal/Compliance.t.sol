// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Contracts
import { L1Compliance } from "src/L1/L1Compliance.sol";
import { Proxy } from "src/universal/Proxy.sol";

// Mocks
import { MockBridge } from "test/mocks/MockBridge.sol";
import {
    MockRuleApprove,
    MockRulePending,
    MockRuleReject,
    MockRuleRefunded,
    MockRuleConfigurable,
    ReentrantSettler
} from "test/mocks/MockRule.sol";

// Interfaces
import { ICompliance } from "interfaces/universal/ICompliance.sol";

/// @title Compliance_TestInit
/// @notice Shared setup for core Compliance state machine tests.
///         Deploys L1Compliance behind a proxy with a MockBridge.
abstract contract Compliance_TestInit is Test {
    // Re-declare compliance events for Solidity 0.8.15 compatibility.
    event Pending(
        bytes32 indexed id,
        address indexed from,
        address indexed to,
        uint256 value,
        uint256 mint,
        uint64 gasLimit,
        uint256 nonce,
        bytes data
    );
    event Rejected(
        bytes32 indexed id,
        address indexed from,
        address indexed to,
        uint256 value,
        uint256 mint,
        uint64 gasLimit,
        uint256 nonce,
        bytes data
    );
    event Approved(bytes32 indexed id);
    event Refunded(bytes32 indexed id);

    MockBridge mockBridge;
    L1Compliance compliance;
    address owner;

    MockRuleApprove ruleApprove;
    MockRulePending rulePending;
    MockRuleReject ruleReject;
    MockRuleRefunded ruleRefunded;

    // Default check params
    address from;
    address to;
    uint256 value_;
    uint64 gasLimit;
    bool isCreation;
    bytes data;
    uint256 nonce;

    function setUp() public virtual {
        owner = makeAddr("owner");
        from = makeAddr("from");
        to = makeAddr("to");
        value_ = 1 ether;
        gasLimit = 100_000;
        isCreation = false;
        data = hex"abcd";
        nonce = 0;

        // Deploy mock bridge
        mockBridge = new MockBridge();

        // Deploy L1Compliance behind a proxy
        L1Compliance impl = new L1Compliance();
        Proxy proxy = new Proxy(address(this));
        proxy.upgradeTo(address(impl));

        compliance = L1Compliance(payable(address(proxy)));
        compliance.initialize(address(mockBridge), owner);

        mockBridge.setCompliance(address(compliance));

        // Deploy mock rules
        ruleApprove = new MockRuleApprove();
        rulePending = new MockRulePending();
        ruleReject = new MockRuleReject();
        ruleRefunded = new MockRuleRefunded();
    }

    /// @dev Calls check() through the mock bridge and returns (allowed, id).
    function _doCheck(
        address _from,
        address _to,
        uint256 _value,
        uint256 _mint,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data,
        uint256 _nonce
    )
        internal
        returns (bool allowed_, bytes32 id_)
    {
        (allowed_, id_) =
            mockBridge.callCheck{ value: _mint }(_from, _to, _value, _gasLimit, _isCreation, _data, _nonce);
    }

    /// @dev Computes the expected id for a given set of params.
    function _computeId(
        address _from,
        address _to,
        uint256 _value,
        uint256 _mint,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data,
        uint256 _nonce
    )
        internal
        pure
        returns (bytes32)
    {
        return keccak256(abi.encode(_from, _to, _value, _mint, _gasLimit, _isCreation, _data, _nonce));
    }
}

// ============================================================
// Initialization Tests
// ============================================================

contract Compliance_Initialize_Test is Compliance_TestInit {
    function test_initialize_succeeds() external view {
        assertEq(compliance.bridge(), address(mockBridge));
        assertEq(compliance.owner(), owner);
    }

    function test_initialize_cannotReinitialize_reverts() external {
        vm.expectRevert("Initializable: contract is already initialized");
        compliance.initialize(address(1), address(2));
    }

    function test_version_succeeds() external view {
        assertEq(compliance.version(), "1.0.0");
    }
}

// ============================================================
// Rule Management Tests
// ============================================================

contract Compliance_RuleManagement_Test is Compliance_TestInit {
    function test_addRule_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(ruleApprove));
        assertTrue(compliance.hasRule(address(ruleApprove)));
        assertEq(compliance.rules().length, 1);
        assertEq(compliance.rules()[0], address(ruleApprove));
    }

    function test_addRule_duplicate_reverts() external {
        vm.startPrank(owner);
        compliance.addRule(address(ruleApprove));
        vm.expectRevert(ICompliance.Compliance_DuplicateRule.selector);
        compliance.addRule(address(ruleApprove));
        vm.stopPrank();
    }

    function test_addRule_notOwner_reverts() external {
        vm.prank(makeAddr("notOwner"));
        vm.expectRevert(bytes4(keccak256("Unauthorized()")));
        compliance.addRule(address(ruleApprove));
    }

    function test_removeRule_succeeds() external {
        vm.startPrank(owner);
        compliance.addRule(address(ruleApprove));
        compliance.removeRule(address(ruleApprove));
        vm.stopPrank();

        assertFalse(compliance.hasRule(address(ruleApprove)));
        assertEq(compliance.rules().length, 0);
    }

    function test_removeRule_notFound_reverts() external {
        vm.prank(owner);
        vm.expectRevert(ICompliance.Compliance_RuleNotFound.selector);
        compliance.removeRule(address(ruleApprove));
    }

    function test_removeRule_notOwner_reverts() external {
        vm.prank(owner);
        compliance.addRule(address(ruleApprove));

        vm.prank(makeAddr("notOwner"));
        vm.expectRevert(bytes4(keccak256("Unauthorized()")));
        compliance.removeRule(address(ruleApprove));
    }

    function test_rules_returnsAll_succeeds() external {
        MockRuleApprove rule2 = new MockRuleApprove();
        MockRuleApprove rule3 = new MockRuleApprove();

        vm.startPrank(owner);
        compliance.addRule(address(ruleApprove));
        compliance.addRule(address(rule2));
        compliance.addRule(address(rule3));
        vm.stopPrank();

        assertEq(compliance.rules().length, 3);
    }

    function test_hasRule_unknownAddress_succeeds() external view {
        assertFalse(compliance.hasRule(address(0xdead)));
    }
}

// ============================================================
// Check Tests
// ============================================================

contract Compliance_Check_Test is Compliance_TestInit {
    function test_check_notBridge_reverts() external {
        vm.expectRevert(ICompliance.Compliance_OnlyBridge.selector);
        compliance.check(from, to, value_, gasLimit, isCreation, data, nonce);
    }

    function test_check_zeroRules_approves_succeeds() external {
        uint256 mint = 1 ether;
        vm.deal(address(this), mint);

        bytes32 expectedId = _computeId(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.expectEmit(address(compliance));
        emit Approved(expectedId);

        (bool allowed,) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);
        assertTrue(allowed);
        // ETH should be donated back to bridge
        assertEq(address(compliance).balance, 0);
        assertEq(address(mockBridge).balance, mint);
    }

    function test_check_ruleApproves_returnsTrue_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(ruleApprove));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);

        (bool allowed,) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);
        assertTrue(allowed);
        assertEq(address(compliance).balance, 0);
        assertEq(address(mockBridge).balance, mint);
    }

    function test_check_rulePending_returnsFalse_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);

        bytes32 expectedId = _computeId(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.expectEmit(address(compliance));
        emit Pending(expectedId, from, to, value_, mint, gasLimit, nonce, data);

        (bool allowed, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);
        assertFalse(allowed);
        assertEq(id, expectedId);
        // ETH held by compliance
        assertEq(address(compliance).balance, mint);

        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertFalse(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Pending));
    }

    function test_check_ruleRejects_returnsFalse_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(ruleReject));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);

        bytes32 expectedId = _computeId(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.expectEmit(address(compliance));
        emit Rejected(expectedId, from, to, value_, mint, gasLimit, nonce, data);

        (bool allowed, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);
        assertFalse(allowed);
        assertEq(id, expectedId);

        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertFalse(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Rejected));
    }

    function test_check_ruleRefunded_reverts() external {
        vm.prank(owner);
        compliance.addRule(address(ruleRefunded));

        vm.deal(address(this), 1 ether);
        vm.expectRevert(ICompliance.Compliance_InvalidRuleStatus.selector);
        _doCheck(from, to, value_, 1 ether, gasLimit, isCreation, data, nonce);
    }

    function test_check_strictest_pendingOverApprove_succeeds() external {
        vm.startPrank(owner);
        compliance.addRule(address(ruleApprove));
        compliance.addRule(address(rulePending));
        vm.stopPrank();

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);

        (bool allowed, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);
        assertFalse(allowed);
        (, ICompliance.Status s) = compliance.status(id);
        assertEq(uint256(s), uint256(ICompliance.Status.Pending));
    }

    function test_check_strictest_rejectedOverPending_succeeds() external {
        vm.startPrank(owner);
        compliance.addRule(address(rulePending));
        compliance.addRule(address(ruleReject));
        vm.stopPrank();

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);

        (bool allowed, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);
        assertFalse(allowed);
        (, ICompliance.Status s) = compliance.status(id);
        assertEq(uint256(s), uint256(ICompliance.Status.Rejected));
    }

    function test_check_strictest_rejectedOverApprove_succeeds() external {
        vm.startPrank(owner);
        compliance.addRule(address(ruleApprove));
        compliance.addRule(address(ruleReject));
        vm.stopPrank();

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);

        (bool allowed, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);
        assertFalse(allowed);
        (, ICompliance.Status s) = compliance.status(id);
        assertEq(uint256(s), uint256(ICompliance.Status.Rejected));
    }

    function test_check_zeroValue_approves_succeeds() external {
        // No rules, msg.value=0 → approve without donateETH call
        (bool allowed,) = _doCheck(from, to, value_, 0, gasLimit, isCreation, data, nonce);
        assertTrue(allowed);
        assertEq(address(compliance).balance, 0);
    }

    function testFuzz_check_hashComputation_succeeds(
        address _from,
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bytes calldata _data,
        uint256 _nonce
    )
        external
    {
        // Add pending rule so status is stored
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 0; // Use 0 mint to avoid dealing ETH
        vm.deal(address(this), 0);

        (, bytes32 id) = _doCheck(_from, _to, _value, mint, _gasLimit, false, _data, _nonce);
        bytes32 expectedId = _computeId(_from, _to, _value, mint, _gasLimit, false, _data, _nonce);
        assertEq(id, expectedId);

        (, ICompliance.Status s) = compliance.status(expectedId);
        assertEq(uint256(s), uint256(ICompliance.Status.Pending));
    }
}

// ============================================================
// Approve Tests
// ============================================================

contract Compliance_Approve_Test is Compliance_TestInit {
    function test_approve_pending_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.expectEmit(address(compliance));
        emit Approved(id);

        vm.prank(owner);
        compliance.approve(id);

        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertTrue(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Approved));
    }

    function test_approve_notPending_reverts() external {
        bytes32 unknownId = keccak256("unknown");
        vm.prank(owner);
        vm.expectRevert(ICompliance.Compliance_NotPending.selector);
        compliance.approve(unknownId);
    }

    function test_approve_rejected_reverts() external {
        vm.prank(owner);
        compliance.addRule(address(ruleReject));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        vm.expectRevert(ICompliance.Compliance_NotPending.selector);
        compliance.approve(id);
    }

    function test_approve_refunded_reverts() external {
        // Create a pending tx, reject it, settle it (refund), then try to approve
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.reject(id);

        // Settle to get Refunded status
        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        vm.expectRevert(ICompliance.Compliance_NotPending.selector);
        compliance.approve(id);
    }

    function test_approve_alreadyOverrideApproved_reverts() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.approve(id);

        // Already override-approved → status is Approved, not Pending
        vm.prank(owner);
        vm.expectRevert(ICompliance.Compliance_NotPending.selector);
        compliance.approve(id);
    }

    function test_approve_notOwner_reverts() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(makeAddr("notOwner"));
        vm.expectRevert(bytes4(keccak256("Unauthorized()")));
        compliance.approve(id);
    }

    function test_approve_setsOverrideBit_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.approve(id);

        (bool isFinal,) = compliance.status(id);
        assertTrue(isFinal);
    }
}

// ============================================================
// Reject Tests
// ============================================================

contract Compliance_Reject_Test is Compliance_TestInit {
    function test_reject_pending_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.reject(id);

        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertTrue(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Rejected));
    }

    function test_reject_approved_succeeds() external {
        // Owner approves first, then owner can reject (Approved -> Rejected)
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.approve(id);

        vm.prank(owner);
        compliance.reject(id);

        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertTrue(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Rejected));
    }

    function test_reject_alreadyRejected_reverts() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.reject(id);

        // Already override-rejected
        vm.prank(owner);
        vm.expectRevert(ICompliance.Compliance_NotPending.selector);
        compliance.reject(id);
    }

    function test_reject_refunded_reverts() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.reject(id);

        // Settle to get Refunded
        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        vm.expectRevert(ICompliance.Compliance_NotPending.selector);
        compliance.reject(id);
    }

    function test_reject_notOwner_reverts() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(makeAddr("notOwner"));
        vm.expectRevert(bytes4(keccak256("Unauthorized()")));
        compliance.reject(id);
    }

    function test_reject_unknownId_succeeds() external {
        // Edge case: unknown id has raw status 0. status() decodes this as
        // (false, Approved) because Status(0) == Approved. The reject() function
        // accepts Approved status, so this sets Override+Rejected on a never-checked id.
        bytes32 unknownId = keccak256("never-checked");

        vm.prank(owner);
        compliance.reject(unknownId);

        (bool isFinal, ICompliance.Status s) = compliance.status(unknownId);
        assertTrue(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Rejected));
    }
}

// ============================================================
// Status Tests
// ============================================================

contract Compliance_Status_Test is Compliance_TestInit {
    function test_status_unknown_returnsNotFinalApproved_succeeds() external view {
        bytes32 unknownId = keccak256("unknown");
        (bool isFinal, ICompliance.Status s) = compliance.status(unknownId);
        assertFalse(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Approved));
    }

    function test_status_pending_notFinal_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertFalse(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Pending));
    }

    function test_status_rejected_notFinal_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(ruleReject));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertFalse(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Rejected));
    }

    function test_status_overrideApproved_isFinal_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.approve(id);

        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertTrue(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Approved));
    }

    function test_status_overrideRejected_isFinal_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.reject(id);

        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertTrue(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Rejected));
    }

    function test_status_refunded_isFinal_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.reject(id);

        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertTrue(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Refunded));
    }
}

// ============================================================
// Settle Tests
// ============================================================

contract Compliance_Settle_Test is Compliance_TestInit {
    function test_settle_unknownId_reverts() external {
        vm.expectRevert(ICompliance.Compliance_NotPending.selector);
        compliance.settle(from, to, value_, 1 ether, gasLimit, isCreation, data, nonce);
    }

    function test_settle_wrongPreimage_reverts() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        // Different params → different hash → _status[id]==0
        vm.expectRevert(ICompliance.Compliance_NotPending.selector);
        compliance.settle(from, to, value_ + 1, mint, gasLimit, isCreation, data, nonce);
    }

    function test_settle_ownerApproved_executes_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.approve(id);

        uint256 bridgeBalBefore = address(mockBridge).balance;

        vm.expectEmit(address(compliance));
        emit Approved(id);

        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        // Status should be deleted (reads as Approved, not final)
        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertFalse(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Approved));

        // Bridge should have received the ETH
        assertEq(address(mockBridge).balance, bridgeBalBefore + mint);
        assertTrue(mockBridge.approvedCalled());
        assertEq(mockBridge.lastFrom(), from);
        assertEq(mockBridge.lastTo(), to);
        assertEq(mockBridge.lastValue(), value_);
        assertEq(mockBridge.lastMint(), mint);
    }

    function test_settle_ownerRejected_refunds_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.reject(id);

        uint256 fromBalBefore = from.balance;

        vm.expectEmit(address(compliance));
        emit Refunded(id);

        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertTrue(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Refunded));
        assertEq(from.balance, fromBalBefore + mint);
    }

    function test_settle_ownerRejected_zeroMint_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 0;
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.reject(id);

        vm.expectEmit(address(compliance));
        emit Refunded(id);

        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        (, ICompliance.Status s) = compliance.status(id);
        assertEq(uint256(s), uint256(ICompliance.Status.Refunded));
    }

    function test_settle_pending_noOp_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        // Settle without override, rules still return Pending → no-op
        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertFalse(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Pending));
        // ETH still held
        assertEq(address(compliance).balance, mint);
    }

    function test_settle_reEval_rulesNowApprove_succeeds() external {
        MockRuleConfigurable configRule = new MockRuleConfigurable(ICompliance.Status.Pending);

        vm.prank(owner);
        compliance.addRule(address(configRule));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        // Change rule to approve
        configRule.setStatus(ICompliance.Status.Approved);

        vm.expectEmit(address(compliance));
        emit Approved(id);

        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        // Status deleted (approved settlement)
        (bool isFinal, ICompliance.Status s) = compliance.status(id);
        assertFalse(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Approved));
        assertTrue(mockBridge.approvedCalled());
    }

    function test_settle_reEval_rulesNowReject_succeeds() external {
        MockRuleConfigurable configRule = new MockRuleConfigurable(ICompliance.Status.Pending);

        vm.prank(owner);
        compliance.addRule(address(configRule));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        // Change rule to reject
        configRule.setStatus(ICompliance.Status.Rejected);

        uint256 fromBalBefore = from.balance;

        vm.expectEmit(address(compliance));
        emit Refunded(id);

        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        (, ICompliance.Status s) = compliance.status(id);
        assertEq(uint256(s), uint256(ICompliance.Status.Refunded));
        assertEq(from.balance, fromBalBefore + mint);
    }

    function test_settle_reEval_rulesStillPending_noOp_succeeds() external {
        MockRuleConfigurable configRule = new MockRuleConfigurable(ICompliance.Status.Pending);

        vm.prank(owner);
        compliance.addRule(address(configRule));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        // Rules still pending → no-op
        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        (, ICompliance.Status s) = compliance.status(id);
        assertEq(uint256(s), uint256(ICompliance.Status.Pending));
        assertEq(address(compliance).balance, mint);
    }

    function test_settle_finalOverridesRuleChange_succeeds() external {
        MockRuleConfigurable configRule = new MockRuleConfigurable(ICompliance.Status.Pending);

        vm.prank(owner);
        compliance.addRule(address(configRule));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        // Owner approves (final)
        vm.prank(owner);
        compliance.approve(id);

        // Rule changed to reject
        configRule.setStatus(ICompliance.Status.Rejected);

        // Settle should still execute (finality wins)
        vm.expectEmit(address(compliance));
        emit Approved(id);

        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        assertTrue(mockBridge.approvedCalled());
    }

    function test_settle_doubleSettleApproved_reverts() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        bytes32 id = _computeId(from, to, value_, mint, gasLimit, isCreation, data, nonce);
        vm.prank(owner);
        compliance.approve(id);

        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        // Second settle: _status[id]==0 → reverts
        vm.expectRevert(ICompliance.Compliance_NotPending.selector);
        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);
    }

    function test_settle_doubleSettleRefunded_reverts() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        bytes32 id = _computeId(from, to, value_, mint, gasLimit, isCreation, data, nonce);
        vm.prank(owner);
        compliance.reject(id);

        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        // Second settle: status==Refunded → reverts
        vm.expectRevert(ICompliance.Compliance_NotPending.selector);
        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);
    }

    function test_settle_refundFails_reverts() external {
        // _from is a contract that rejects ETH
        address payable noReceive = payable(address(new NoReceive()));

        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(noReceive, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.reject(id);

        vm.expectRevert(ICompliance.Compliance_TransferFailed.selector);
        compliance.settle(noReceive, to, value_, mint, gasLimit, isCreation, data, nonce);
    }

    function test_settle_reentrancy_reverts() external {
        ReentrantSettler reentrant = new ReentrantSettler();

        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(address(reentrant), to, value_, mint, gasLimit, isCreation, data, nonce);

        reentrant.setup(address(compliance), to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.reject(id);

        // settle() will try to refund reentrant, which will try to call settle() again
        vm.expectRevert(ICompliance.Compliance_TransferFailed.selector);
        compliance.settle(address(reentrant), to, value_, mint, gasLimit, isCreation, data, nonce);
    }

    function test_settle_anyoneCanCall_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.approve(id);

        // Random address can call settle
        address random = makeAddr("random");
        vm.prank(random);
        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        assertTrue(mockBridge.approvedCalled());
    }

    function testFuzz_settle_hashMatchesCheck_succeeds(
        address _from,
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bytes calldata _data
    )
        external
    {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 0; // Use 0 mint to avoid ETH dealing complexity

        (, bytes32 checkId) = _doCheck(_from, _to, _value, mint, _gasLimit, false, _data, 0);

        vm.prank(owner);
        compliance.approve(checkId);

        // settle with same params should succeed (hash matches)
        compliance.settle(_from, _to, _value, mint, _gasLimit, false, _data, 0);

        assertTrue(mockBridge.approvedCalled());
    }
}

// ============================================================
// ETH Accounting Tests
// ============================================================

contract Compliance_ETHAccounting_Test is Compliance_TestInit {
    function test_eth_approvedCheck_returnedViaDonate_succeeds() external {
        uint256 mint = 2 ether;
        vm.deal(address(this), mint);

        uint256 bridgeBefore = address(mockBridge).balance;
        uint256 complianceBefore = address(compliance).balance;

        _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        assertEq(address(compliance).balance, complianceBefore);
        assertEq(address(mockBridge).balance, bridgeBefore + mint);
    }

    function test_eth_flaggedCheck_heldByCompliance_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 2 ether;
        vm.deal(address(this), mint);

        uint256 complianceBefore = address(compliance).balance;

        _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        assertEq(address(compliance).balance, complianceBefore + mint);
    }

    function test_eth_settleApproved_forwardedToBridge_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 2 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.approve(id);

        uint256 bridgeBefore = address(mockBridge).balance;
        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        assertEq(address(compliance).balance, 0);
        assertEq(address(mockBridge).balance, bridgeBefore + mint);
    }

    function test_eth_settleRejected_refundedToFrom_succeeds() external {
        vm.prank(owner);
        compliance.addRule(address(rulePending));

        uint256 mint = 2 ether;
        vm.deal(address(this), mint);
        (, bytes32 id) = _doCheck(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        vm.prank(owner);
        compliance.reject(id);

        uint256 fromBefore = from.balance;
        compliance.settle(from, to, value_, mint, gasLimit, isCreation, data, nonce);

        assertEq(address(compliance).balance, 0);
        assertEq(from.balance, fromBefore + mint);
    }
}

// ============================================================
// Helper contracts
// ============================================================

/// @notice Contract that rejects ETH transfers.
contract NoReceive {
// No receive or fallback → ETH transfers revert
}
