// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { Vm } from "forge-std/Vm.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Contracts
import { L2Compliance } from "src/L2/L2Compliance.sol";
import { Proxy } from "src/universal/Proxy.sol";

// Mocks
import { MockRuleApprove, MockRulePending, MockRuleReject } from "test/mocks/MockRule.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Features } from "src/libraries/Features.sol";
// Interfaces
import { ICompliance } from "interfaces/universal/ICompliance.sol";
import { IL2ToL1MessagePasser } from "interfaces/L2/IL2ToL1MessagePasser.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

/// @title L2Compliance_TestInit
/// @notice Shared setup for L2 compliance integration tests.
abstract contract L2Compliance_TestInit is CommonTest {
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
    event Rejected(bytes32 indexed id);
    event Approved(bytes32 indexed id);
    event Refunded(bytes32 indexed id);

    L2Compliance l2Compliance;
    address complianceOwner;

    MockRuleApprove ruleApprove;
    MockRulePending rulePending;
    MockRuleReject ruleReject;

    // The proxy admin for L2 predeploys.
    address l2ProxyAdmin;
    address l2ProxyAdminOwner;

    function setUp() public virtual override {
        super.setUp();
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        complianceOwner = makeAddr("complianceOwner");

        // Deploy L2Compliance behind a proxy
        L2Compliance impl = new L2Compliance();
        Proxy proxy = new Proxy(address(this));
        proxy.upgradeTo(address(impl));

        l2Compliance = L2Compliance(payable(address(proxy)));
        l2Compliance.initialize(address(l2ToL1MessagePasser), complianceOwner);

        // Get the proxy admin for L2 predeploys
        l2ProxyAdmin = EIP1967Helper.getAdmin(address(l2ToL1MessagePasser));
        l2ProxyAdminOwner = IProxyAdmin(l2ProxyAdmin).owner();

        // Set compliance on the message passer via setCompliance
        vm.prank(l2ProxyAdminOwner);
        l2ToL1MessagePasser.setCompliance(address(l2Compliance));

        // Deploy mock rules
        ruleApprove = new MockRuleApprove();
        rulePending = new MockRulePending();
        ruleReject = new MockRuleReject();
    }

    /// @dev Returns the versioned nonce for a given raw nonce.
    function _versionedNonce(uint240 _nonce) internal pure returns (uint256) {
        return Encoding.encodeVersionedNonce(_nonce, 1); // MESSAGE_VERSION = 1
    }
}

// ============================================================
// InitiateWithdrawal with Compliance
// ============================================================

contract L2Compliance_check_Test is L2Compliance_TestInit {
    function test_initiateWithdrawal_approved_succeeds() external {
        vm.prank(complianceOwner);
        l2Compliance.addRule(address(ruleApprove));

        uint256 wdValue = 1 ether;
        address target = makeAddr("target");
        uint256 wdGasLimit = 100_000;
        bytes memory wdData = hex"ff";

        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: alice,
                target: target,
                value: wdValue,
                gasLimit: wdGasLimit,
                data: wdData
            })
        );

        vm.deal(alice, wdValue);

        vm.expectEmit(address(l2ToL1MessagePasser));
        emit MessagePassed(nonce, alice, target, wdValue, wdGasLimit, wdData, withdrawalHash);

        vm.prank(alice);
        l2ToL1MessagePasser.initiateWithdrawal{ value: wdValue }({
            _target: target,
            _gasLimit: wdGasLimit,
            _data: wdData
        });

        assertTrue(l2ToL1MessagePasser.sentMessages(withdrawalHash));
        assertEq(l2ToL1MessagePasser.messageNonce(), nonce + 1);
    }

    function test_initiateWithdrawal_flaggedReservesNonce_succeeds() external {
        vm.prank(complianceOwner);
        l2Compliance.addRule(address(rulePending));

        uint256 wdValue = 1 ether;
        address target = makeAddr("target");
        uint256 wdGasLimit = 100_000;
        bytes memory wdData = hex"ff";

        uint256 nonceBefore = l2ToL1MessagePasser.messageNonce();

        vm.deal(alice, wdValue);
        vm.prank(alice);
        l2ToL1MessagePasser.initiateWithdrawal{ value: wdValue }({
            _target: target,
            _gasLimit: wdGasLimit,
            _data: wdData
        });

        // Nonce should be incremented (reserved)
        assertEq(l2ToL1MessagePasser.messageNonce(), nonceBefore + 1);

        // But sentMessages should NOT be set
        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonceBefore,
                sender: alice,
                target: target,
                value: wdValue,
                gasLimit: wdGasLimit,
                data: wdData
            })
        );
        assertFalse(l2ToL1MessagePasser.sentMessages(withdrawalHash));

        // ETH held by compliance
        assertEq(address(l2Compliance).balance, wdValue);
    }

    function test_initiateWithdrawal_flaggedNoMessagePassed_succeeds() external {
        vm.prank(complianceOwner);
        l2Compliance.addRule(address(rulePending));

        uint256 wdValue = 1 ether;
        address target = makeAddr("target");
        uint256 wdGasLimit = 100_000;
        bytes memory wdData = hex"ff";

        vm.deal(alice, wdValue);

        // Record logs and verify no MessagePassed event
        vm.recordLogs();
        vm.prank(alice);
        l2ToL1MessagePasser.initiateWithdrawal{ value: wdValue }({
            _target: target,
            _gasLimit: wdGasLimit,
            _data: wdData
        });

        Vm.Log[] memory logs = vm.getRecordedLogs();
        bytes32 messagePassedTopic = keccak256("MessagePassed(uint256,address,address,uint256,uint256,bytes,bytes32)");
        for (uint256 i = 0; i < logs.length; i++) {
            if (logs[i].emitter == address(l2ToL1MessagePasser)) {
                assertTrue(logs[i].topics[0] != messagePassedTopic, "MessagePassed should not be emitted when flagged");
            }
        }
    }

    function test_initiateWithdrawal_noComplianceLegacyBehavior_succeeds() external {
        // Disable compliance
        vm.prank(l2ProxyAdminOwner);
        l2ToL1MessagePasser.setCompliance(address(0));

        uint256 wdValue = 1 ether;
        address target = makeAddr("target");
        uint256 wdGasLimit = 100_000;
        bytes memory wdData = hex"ff";

        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: alice,
                target: target,
                value: wdValue,
                gasLimit: wdGasLimit,
                data: wdData
            })
        );

        vm.deal(alice, wdValue);

        vm.expectEmit(address(l2ToL1MessagePasser));
        emit MessagePassed(nonce, alice, target, wdValue, wdGasLimit, wdData, withdrawalHash);

        vm.prank(alice);
        l2ToL1MessagePasser.initiateWithdrawal{ value: wdValue }({
            _target: target,
            _gasLimit: wdGasLimit,
            _data: wdData
        });

        assertTrue(l2ToL1MessagePasser.sentMessages(withdrawalHash));
    }

    function test_initiateWithdrawal_hashIncludesVersionedNonce_succeeds() external {
        vm.prank(complianceOwner);
        l2Compliance.addRule(address(rulePending));

        uint256 wdValue = 1 ether;
        address target = makeAddr("target");
        uint256 wdGasLimit = 100_000;
        bytes memory wdData = hex"ff";

        uint256 nonce = l2ToL1MessagePasser.messageNonce();
        // nonce should include MESSAGE_VERSION=1 in the upper bytes
        assertTrue(nonce >> 240 == 1, "Nonce should have version 1");

        vm.deal(alice, wdValue);

        // The compliance hash includes the versioned nonce
        bytes32 complianceId =
            keccak256(abi.encode(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce));

        vm.expectEmit(address(l2Compliance));
        emit Pending(complianceId, alice, target, wdValue, wdValue, uint64(wdGasLimit), nonce, wdData);

        vm.prank(alice);
        l2ToL1MessagePasser.initiateWithdrawal{ value: wdValue }({
            _target: target,
            _gasLimit: wdGasLimit,
            _data: wdData
        });
    }
}

// ============================================================
// MessagePasser approved() callback
// ============================================================

contract L2Compliance_settle_Test is L2Compliance_TestInit {
    function test_approved_notCompliance_reverts() external {
        vm.expectRevert(IL2ToL1MessagePasser.L2ToL1MessagePasser_OnlyCompliance.selector);
        l2ToL1MessagePasser.approved(alice, bob, 1 ether, 100_000, hex"", 0);
    }

    function test_approved_storesWithdrawalHash_succeeds() external {
        vm.prank(complianceOwner);
        l2Compliance.addRule(address(rulePending));

        uint256 wdValue = 1 ether;
        address target = makeAddr("target");
        uint256 wdGasLimit = 100_000;
        bytes memory wdData = hex"ff";

        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        vm.deal(alice, wdValue);
        vm.prank(alice);
        l2ToL1MessagePasser.initiateWithdrawal{ value: wdValue }({
            _target: target,
            _gasLimit: wdGasLimit,
            _data: wdData
        });

        bytes32 complianceId =
            keccak256(abi.encode(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce));

        vm.prank(complianceOwner);
        l2Compliance.approve(complianceId);

        // Settle triggers approved() on the message passer
        l2Compliance.settle(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce);

        // Withdrawal hash should now be stored
        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: alice,
                target: target,
                value: wdValue,
                gasLimit: wdGasLimit,
                data: wdData
            })
        );
        assertTrue(l2ToL1MessagePasser.sentMessages(withdrawalHash));
    }

    function test_approved_emitsMessagePassed_succeeds() external {
        vm.prank(complianceOwner);
        l2Compliance.addRule(address(rulePending));

        uint256 wdValue = 1 ether;
        address target = makeAddr("target");
        uint256 wdGasLimit = 100_000;
        bytes memory wdData = hex"ff";

        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        vm.deal(alice, wdValue);
        vm.prank(alice);
        l2ToL1MessagePasser.initiateWithdrawal{ value: wdValue }({
            _target: target,
            _gasLimit: wdGasLimit,
            _data: wdData
        });

        bytes32 complianceId =
            keccak256(abi.encode(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce));

        vm.prank(complianceOwner);
        l2Compliance.approve(complianceId);

        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: alice,
                target: target,
                value: wdValue,
                gasLimit: wdGasLimit,
                data: wdData
            })
        );

        vm.expectEmit(address(l2ToL1MessagePasser));
        emit MessagePassed(nonce, alice, target, wdValue, wdGasLimit, wdData, withdrawalHash);

        l2Compliance.settle(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce);
    }

    function test_approved_withdrawalHashMatchesHashing_succeeds() external {
        vm.prank(complianceOwner);
        l2Compliance.addRule(address(rulePending));

        uint256 wdValue = 1 ether;
        address target = makeAddr("target");
        uint256 wdGasLimit = 100_000;
        bytes memory wdData = hex"ff";

        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        vm.deal(alice, wdValue);
        vm.prank(alice);
        l2ToL1MessagePasser.initiateWithdrawal{ value: wdValue }({
            _target: target,
            _gasLimit: wdGasLimit,
            _data: wdData
        });

        bytes32 complianceId =
            keccak256(abi.encode(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce));

        vm.prank(complianceOwner);
        l2Compliance.approve(complianceId);

        l2Compliance.settle(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce);

        // Verify via Hashing library that the stored hash matches
        bytes32 expectedHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: alice,
                target: target,
                value: wdValue,
                gasLimit: wdGasLimit,
                data: wdData
            })
        );

        assertTrue(l2ToL1MessagePasser.sentMessages(expectedHash));
    }
}

// ============================================================
// SetCompliance Tests
// ============================================================

contract L2Compliance_initialize_Test is L2Compliance_TestInit {
    function test_setCompliance_fromProxyAdmin_succeeds() external {
        vm.prank(l2ProxyAdmin);
        l2ToL1MessagePasser.setCompliance(address(0xdead));
        assertEq(l2ToL1MessagePasser.compliance(), address(0xdead));
    }

    function test_setCompliance_fromProxyAdminOwner_succeeds() external {
        vm.prank(l2ProxyAdminOwner);
        l2ToL1MessagePasser.setCompliance(address(0xbeef));
        assertEq(l2ToL1MessagePasser.compliance(), address(0xbeef));
    }

    function test_setCompliance_unauthorized_reverts() external {
        vm.prank(makeAddr("random"));
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);
        l2ToL1MessagePasser.setCompliance(address(0xdead));
    }

    function test_setCompliance_toZeroDisables_succeeds() external {
        vm.prank(l2ProxyAdminOwner);
        l2ToL1MessagePasser.setCompliance(address(0));
        assertEq(l2ToL1MessagePasser.compliance(), address(0));

        // Subsequent withdrawal should proceed without compliance
        uint256 wdValue = 1 ether;
        address target = makeAddr("target");
        uint256 wdGasLimit = 100_000;
        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: alice,
                target: target,
                value: wdValue,
                gasLimit: wdGasLimit,
                data: hex""
            })
        );

        vm.deal(alice, wdValue);
        vm.prank(alice);
        l2ToL1MessagePasser.initiateWithdrawal{ value: wdValue }({ _target: target, _gasLimit: wdGasLimit, _data: hex"" });

        assertTrue(l2ToL1MessagePasser.sentMessages(withdrawalHash));
    }
}

// ============================================================
// DonateETH Tests
// ============================================================

contract L2Compliance_Uncategorized_Test is L2Compliance_TestInit {
    function test_donateETH_acceptsETH_succeeds() external {
        uint256 amount = 1 ether;
        vm.deal(address(this), amount);

        uint256 balBefore = address(l2ToL1MessagePasser).balance;
        l2ToL1MessagePasser.donateETH{ value: amount }();
        assertEq(address(l2ToL1MessagePasser).balance, balBefore + amount);
    }

    function test_donateETH_noMessagePassed_succeeds() external {
        uint256 amount = 1 ether;
        vm.deal(address(this), amount);

        vm.recordLogs();
        l2ToL1MessagePasser.donateETH{ value: amount }();

        Vm.Log[] memory logs = vm.getRecordedLogs();
        bytes32 messagePassedTopic = keccak256("MessagePassed(uint256,address,address,uint256,uint256,bytes,bytes32)");
        for (uint256 i = 0; i < logs.length; i++) {
            assertTrue(logs[i].topics[0] != messagePassedTopic, "donateETH should not emit MessagePassed");
        }

        // sentMessages should not have any new entries (can't easily verify all,
        // but we verify the hash for a dummy withdrawal is not set)
        bytes32 dummyHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: 0,
                sender: address(this),
                target: address(this),
                value: amount,
                gasLimit: 0,
                data: hex""
            })
        );
        assertFalse(l2ToL1MessagePasser.sentMessages(dummyHash));
    }
}

// ============================================================
// End-to-End Tests
// ============================================================

contract L2Compliance_approve_Test is L2Compliance_TestInit {
    function test_e2e_flagThenApproveThenSettle_succeeds() external {
        vm.prank(complianceOwner);
        l2Compliance.addRule(address(rulePending));

        uint256 wdValue = 2 ether;
        address target = makeAddr("target");
        uint256 wdGasLimit = 100_000;
        bytes memory wdData = hex"1234";

        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        // Step 1: Initiate (flagged)
        vm.deal(alice, wdValue);
        vm.prank(alice);
        l2ToL1MessagePasser.initiateWithdrawal{ value: wdValue }({
            _target: target,
            _gasLimit: wdGasLimit,
            _data: wdData
        });

        assertEq(address(l2Compliance).balance, wdValue);

        bytes32 complianceId =
            keccak256(abi.encode(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce));
        (, ICompliance.Status s) = l2Compliance.status(complianceId);
        assertEq(uint256(s), uint256(ICompliance.Status.Pending));

        // Step 2: Owner approves
        vm.prank(complianceOwner);
        l2Compliance.approve(complianceId);

        // Step 3: Settle
        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: alice,
                target: target,
                value: wdValue,
                gasLimit: wdGasLimit,
                data: wdData
            })
        );

        vm.expectEmit(address(l2ToL1MessagePasser));
        emit MessagePassed(nonce, alice, target, wdValue, wdGasLimit, wdData, withdrawalHash);

        l2Compliance.settle(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce);

        assertTrue(l2ToL1MessagePasser.sentMessages(withdrawalHash));
        assertEq(address(l2Compliance).balance, 0);
    }

    function test_e2e_flagThenRejectThenSettle_succeeds() external {
        vm.prank(complianceOwner);
        l2Compliance.addRule(address(rulePending));

        uint256 wdValue = 2 ether;
        address target = makeAddr("target");
        uint256 wdGasLimit = 100_000;
        bytes memory wdData = hex"1234";

        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        // Step 1: Initiate (flagged)
        vm.deal(alice, wdValue);
        vm.prank(alice);
        l2ToL1MessagePasser.initiateWithdrawal{ value: wdValue }({
            _target: target,
            _gasLimit: wdGasLimit,
            _data: wdData
        });

        bytes32 complianceId =
            keccak256(abi.encode(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce));

        // Step 2: Owner rejects
        vm.prank(complianceOwner);
        l2Compliance.reject(complianceId);

        // Step 3: Settle → ETH refunded to alice
        uint256 aliceBefore = alice.balance;

        vm.expectEmit(address(l2Compliance));
        emit Refunded(complianceId);

        l2Compliance.settle(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce);

        assertEq(alice.balance, aliceBefore + wdValue);
        assertEq(address(l2Compliance).balance, 0);
    }

    function test_e2e_noncePreservation_succeeds() external {
        vm.prank(complianceOwner);
        l2Compliance.addRule(address(rulePending));

        uint256 wdValue = 1 ether;
        address target = makeAddr("target");
        uint256 wdGasLimit = 100_000;
        bytes memory wdData = hex"ff";

        // Record the nonce before the withdrawal
        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        vm.deal(alice, wdValue);
        vm.prank(alice);
        l2ToL1MessagePasser.initiateWithdrawal{ value: wdValue }({
            _target: target,
            _gasLimit: wdGasLimit,
            _data: wdData
        });

        // Nonce should have advanced
        assertEq(l2ToL1MessagePasser.messageNonce(), nonce + 1);

        bytes32 complianceId =
            keccak256(abi.encode(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce));

        vm.prank(complianceOwner);
        l2Compliance.approve(complianceId);

        l2Compliance.settle(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce);

        // The withdrawal hash should use the reserved nonce
        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: alice,
                target: target,
                value: wdValue,
                gasLimit: wdGasLimit,
                data: wdData
            })
        );

        assertTrue(l2ToL1MessagePasser.sentMessages(withdrawalHash));
    }

    function test_e2e_l2HashComputation_succeeds() external {
        vm.prank(complianceOwner);
        l2Compliance.addRule(address(rulePending));

        uint256 wdValue = 1 ether;
        address target = makeAddr("target");
        uint256 wdGasLimit = 100_000;
        bytes memory wdData = hex"cafe";

        uint256 nonce = l2ToL1MessagePasser.messageNonce();

        vm.deal(alice, wdValue);
        vm.prank(alice);
        l2ToL1MessagePasser.initiateWithdrawal{ value: wdValue }({
            _target: target,
            _gasLimit: wdGasLimit,
            _data: wdData
        });

        // For L2 path:
        // check(msg.sender, _target, msg.value, uint64(_gasLimit), false, _data, nonce) with msg.value
        // So in the hash: _value == msg.value == _mint
        // isCreation = false, nonce = versioned nonce
        bytes32 complianceId = keccak256(
            abi.encode(
                alice, // _from
                target, // _to
                wdValue, // _value = msg.value
                wdValue, // _mint = msg.value (both are msg.value for L2)
                uint64(wdGasLimit),
                false, // _isCreation always false for withdrawals
                wdData,
                nonce // versioned nonce
            )
        );

        (, ICompliance.Status s) = l2Compliance.status(complianceId);
        assertEq(uint256(s), uint256(ICompliance.Status.Pending));

        vm.prank(complianceOwner);
        l2Compliance.approve(complianceId);

        // settle() uses the same hash: (_from, _to, _value, _mint, _gasLimit, _isCreation, _data, _nonce)
        // For L2: _value == _mint == wdValue
        l2Compliance.settle(alice, target, wdValue, wdValue, uint64(wdGasLimit), false, wdData, nonce);

        // Verify the withdrawal was properly stored
        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: alice,
                target: target,
                value: wdValue,
                gasLimit: wdGasLimit,
                data: wdData
            })
        );

        assertTrue(l2ToL1MessagePasser.sentMessages(withdrawalHash));
    }
}
