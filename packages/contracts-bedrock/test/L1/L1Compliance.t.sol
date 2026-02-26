// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { Vm } from "forge-std/Vm.sol";

// Contracts
import { L1Compliance } from "src/L1/L1Compliance.sol";
import { Proxy } from "src/universal/Proxy.sol";

// Mocks
import { MockRuleApprove, MockRulePending, MockRuleReject } from "test/mocks/MockRule.sol";

// Libraries
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { Features } from "src/libraries/Features.sol";

// Interfaces
import { ICompliance } from "interfaces/universal/ICompliance.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";

/// @title L1Compliance_TestInit
/// @notice Shared setup for L1 compliance integration tests.
abstract contract L1Compliance_TestInit is CommonTest {
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
    // Note: These don't conflict with Events base class which has no Approved/Refunded events.
    event Approved(bytes32 indexed id);
    event Refunded(bytes32 indexed id);

    L1Compliance l1Compliance;
    address complianceOwner;

    MockRuleApprove ruleApprove;
    MockRulePending rulePending;
    MockRuleReject ruleReject;

    function setUp() public virtual override {
        super.setUp();

        complianceOwner = makeAddr("complianceOwner");

        // Deploy L1Compliance behind a proxy
        L1Compliance impl = new L1Compliance();
        Proxy proxy = new Proxy(address(this));
        proxy.upgradeTo(address(impl));

        l1Compliance = L1Compliance(payable(address(proxy)));
        l1Compliance.initialize(address(optimismPortal2), complianceOwner);

        // Set compliance on the portal using stdstore
        // Compliance is at storage slot 64 in OptimismPortal2 (per forge inspect).
        vm.store(address(optimismPortal2), bytes32(uint256(64)), bytes32(uint256(uint160(address(l1Compliance)))));

        // Deploy mock rules
        ruleApprove = new MockRuleApprove();
        rulePending = new MockRulePending();
        ruleReject = new MockRuleReject();
    }

    /// @dev Helper to get the minimum gas limit for depositTransaction.
    function _minGasLimit(bytes memory _data) internal view returns (uint64) {
        return optimismPortal2.minimumGasLimit(uint64(_data.length));
    }
}

// ============================================================
// DepositTransaction with Compliance
// ============================================================

contract L1Compliance_DepositTransaction_Test is L1Compliance_TestInit {
    function test_depositTransaction_approved_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        vm.prank(complianceOwner);
        l1Compliance.addRule(address(ruleApprove));

        uint256 mint = 1 ether;
        address target = makeAddr("target");
        bytes memory txData = hex"ff";
        uint64 gasLimit = _minGasLimit(txData);

        vm.deal(alice, mint);

        // Expect TransactionDeposited event from portal
        vm.expectEmit(address(optimismPortal2));
        emit TransactionDeposited(alice, target, 0, abi.encodePacked(mint, uint256(0), gasLimit, false, txData));

        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: mint }({
            _to: target,
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: txData
        });
    }

    function test_depositTransaction_flaggedPending_noDeposit_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        vm.prank(complianceOwner);
        l1Compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        address target = makeAddr("target");
        bytes memory txData = hex"ff";
        uint64 gasLimit = _minGasLimit(txData);

        vm.deal(alice, mint);

        // Expect Pending event from compliance, NOT TransactionDeposited
        bytes32 expectedId = keccak256(abi.encode(alice, target, uint256(0), mint, gasLimit, false, txData, uint256(0)));
        vm.expectEmit(address(l1Compliance));
        emit Pending(expectedId, alice, target, 0, mint, gasLimit, 0, txData);

        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: mint }({
            _to: target,
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: txData
        });

        // ETH held by compliance
        assertEq(address(l1Compliance).balance, mint);
    }

    function test_depositTransaction_flaggedRejected_noDeposit_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        vm.prank(complianceOwner);
        l1Compliance.addRule(address(ruleReject));

        uint256 mint = 1 ether;
        address target = makeAddr("target");
        bytes memory txData = hex"ff";
        uint64 gasLimit = _minGasLimit(txData);

        vm.deal(alice, mint);

        bytes32 expectedId = keccak256(abi.encode(alice, target, uint256(0), mint, gasLimit, false, txData, uint256(0)));
        vm.expectEmit(address(l1Compliance));
        emit Rejected(expectedId, alice, target, 0, mint, gasLimit, 0, txData);

        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: mint }({
            _to: target,
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: txData
        });

        assertEq(address(l1Compliance).balance, mint);
    }

    function test_depositTransaction_noCompliance_legacyBehavior_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        // Disable compliance
        vm.store(address(optimismPortal2), bytes32(uint256(64)), bytes32(uint256(0)));

        uint256 mint = 1 ether;
        address target = makeAddr("target");
        bytes memory txData = hex"ff";
        uint64 gasLimit = _minGasLimit(txData);

        vm.deal(alice, mint);

        vm.expectEmit(address(optimismPortal2));
        emit TransactionDeposited(alice, target, 0, abi.encodePacked(mint, uint256(0), gasLimit, false, txData));

        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: mint }({
            _to: target,
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: txData
        });
    }

    function testFuzz_depositTransaction_approved_correctOpaqueData_succeeds(
        address _to,
        uint256 _value,
        uint256 _mint,
        bytes memory _data
    )
        external
    {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        // No rules = auto-approve
        _mint = bound(_mint, 0, type(uint128).max);
        if (_to == address(0)) _to = address(1); // avoid contract creation
        uint64 gasLimit = _minGasLimit(_data);
        gasLimit = uint64(bound(gasLimit, gasLimit, systemConfig.resourceConfig().maxResourceLimit));

        vm.deal(alice, _mint);

        vm.expectEmit(address(optimismPortal2));
        emit TransactionDeposited(alice, _to, 0, abi.encodePacked(_mint, _value, gasLimit, false, _data));

        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: _mint }({
            _to: _to,
            _value: _value,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: _data
        });
    }
}

// ============================================================
// Portal approved() callback
// ============================================================

contract L1Compliance_PortalApproved_Test is L1Compliance_TestInit {
    function test_approved_notCompliance_reverts() external {
        vm.expectRevert(IOptimismPortal.OptimismPortal_OnlyCompliance.selector);
        optimismPortal2.approved(alice, bob, 0, 0, 100_000, false, hex"");
    }

    function test_approved_emitsTransactionDeposited_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        vm.prank(complianceOwner);
        l1Compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        address target = makeAddr("target");
        bytes memory txData = hex"ff";
        uint64 gasLimit = _minGasLimit(txData);

        vm.deal(alice, mint);
        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: mint }({
            _to: target,
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: txData
        });

        bytes32 id = keccak256(abi.encode(alice, target, uint256(0), mint, gasLimit, false, txData, uint256(0)));

        vm.prank(complianceOwner);
        l1Compliance.approve(id);

        // Expect TransactionDeposited when settle is called
        vm.expectEmit(address(optimismPortal2));
        emit TransactionDeposited(alice, target, 0, abi.encodePacked(mint, uint256(0), gasLimit, false, txData));

        l1Compliance.settle(alice, target, 0, mint, gasLimit, false, txData, 0);
    }

    function test_approved_contractSender_aliased_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        vm.prank(complianceOwner);
        l1Compliance.addRule(address(rulePending));

        // Use a contract address as _from (has code)
        address contractSender = address(l1Compliance); // any contract
        uint256 mint = 1 ether;
        address target = makeAddr("target");
        bytes memory txData = hex"ff";
        uint64 gasLimit = _minGasLimit(txData);

        // Fund the portal and do check through the bridge
        vm.deal(address(optimismPortal2), mint);
        vm.prank(address(optimismPortal2));
        l1Compliance.check{ value: mint }(contractSender, target, 0, gasLimit, false, txData, 0);

        bytes32 id =
            keccak256(abi.encode(contractSender, target, uint256(0), mint, gasLimit, false, txData, uint256(0)));

        vm.prank(complianceOwner);
        l1Compliance.approve(id);

        // Expect aliased address in event
        address aliased = AddressAliasHelper.applyL1ToL2Alias(contractSender);
        vm.expectEmit(address(optimismPortal2));
        emit TransactionDeposited(aliased, target, 0, abi.encodePacked(mint, uint256(0), gasLimit, false, txData));

        l1Compliance.settle(contractSender, target, 0, mint, gasLimit, false, txData, 0);
    }

    function test_approved_eoaSender_notAliased_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        vm.prank(complianceOwner);
        l1Compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        address target = makeAddr("target");
        bytes memory txData = hex"ff";
        uint64 gasLimit = _minGasLimit(txData);

        vm.deal(alice, mint);
        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: mint }({
            _to: target,
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: txData
        });

        bytes32 id = keccak256(abi.encode(alice, target, uint256(0), mint, gasLimit, false, txData, uint256(0)));

        vm.prank(complianceOwner);
        l1Compliance.approve(id);

        // alice is EOA, no aliasing
        vm.expectEmit(address(optimismPortal2));
        emit TransactionDeposited(alice, target, 0, abi.encodePacked(mint, uint256(0), gasLimit, false, txData));

        l1Compliance.settle(alice, target, 0, mint, gasLimit, false, txData, 0);
    }

    function test_approved_withLockbox_locksETH_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);
        if (!systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX) || address(optimismPortal2.ethLockbox()) == address(0))
        {
            vm.skip(true);
        }

        vm.prank(complianceOwner);
        l1Compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        address target = makeAddr("target");
        bytes memory txData = hex"ff";
        uint64 gasLimit = _minGasLimit(txData);

        vm.deal(alice, mint);
        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: mint }({
            _to: target,
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: txData
        });

        bytes32 id = keccak256(abi.encode(alice, target, uint256(0), mint, gasLimit, false, txData, uint256(0)));

        vm.prank(complianceOwner);
        l1Compliance.approve(id);

        uint256 lockboxBefore = address(optimismPortal2.ethLockbox()).balance;

        l1Compliance.settle(alice, target, 0, mint, gasLimit, false, txData, 0);

        assertEq(address(optimismPortal2.ethLockbox()).balance, lockboxBefore + mint);
    }
}

// ============================================================
// DonateETH Tests
// ============================================================

contract L1Compliance_DonateETH_Test is L1Compliance_TestInit {
    function test_donateETH_acceptsETH_succeeds() external {
        uint256 amount = 1 ether;
        vm.deal(address(this), amount);

        uint256 balBefore = address(optimismPortal2).balance;
        optimismPortal2.donateETH{ value: amount }();
        assertEq(address(optimismPortal2).balance, balBefore + amount);
    }

    function test_donateETH_noSideEffects_succeeds() external {
        uint256 amount = 1 ether;
        vm.deal(address(this), amount);

        // donateETH should not emit TransactionDeposited
        vm.recordLogs();
        optimismPortal2.donateETH{ value: amount }();

        // Check no TransactionDeposited events were emitted
        Vm.Log[] memory logs = vm.getRecordedLogs();
        bytes32 depositedTopic = keccak256("TransactionDeposited(address,address,uint256,bytes)");
        for (uint256 i = 0; i < logs.length; i++) {
            assertTrue(logs[i].topics[0] != depositedTopic, "donateETH should not emit TransactionDeposited");
        }
    }
}

// ============================================================
// End-to-End Tests
// ============================================================

contract L1Compliance_EndToEnd_Test is L1Compliance_TestInit {
    function test_e2e_flagThenApproveThenSettle_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        vm.prank(complianceOwner);
        l1Compliance.addRule(address(rulePending));

        uint256 mint = 2 ether;
        address target = makeAddr("target");
        bytes memory txData = hex"1234";
        uint64 gasLimit = _minGasLimit(txData);

        // Step 1: Deposit is flagged
        vm.deal(alice, mint);
        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: mint }({
            _to: target,
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: txData
        });

        assertEq(address(l1Compliance).balance, mint);

        bytes32 id = keccak256(abi.encode(alice, target, uint256(0), mint, gasLimit, false, txData, uint256(0)));
        (bool isFinal, ICompliance.Status s) = l1Compliance.status(id);
        assertFalse(isFinal);
        assertEq(uint256(s), uint256(ICompliance.Status.Pending));

        // Step 2: Owner approves
        vm.prank(complianceOwner);
        l1Compliance.approve(id);

        // Step 3: Anyone settles
        vm.expectEmit(address(optimismPortal2));
        emit TransactionDeposited(alice, target, 0, abi.encodePacked(mint, uint256(0), gasLimit, false, txData));

        l1Compliance.settle(alice, target, 0, mint, gasLimit, false, txData, 0);

        assertEq(address(l1Compliance).balance, 0);
    }

    function test_e2e_flagThenRejectThenSettle_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        vm.prank(complianceOwner);
        l1Compliance.addRule(address(rulePending));

        uint256 mint = 2 ether;
        address target = makeAddr("target");
        bytes memory txData = hex"1234";
        uint64 gasLimit = _minGasLimit(txData);

        // Step 1: Deposit is flagged
        vm.deal(alice, mint);
        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: mint }({
            _to: target,
            _value: 0,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: txData
        });

        bytes32 id = keccak256(abi.encode(alice, target, uint256(0), mint, gasLimit, false, txData, uint256(0)));

        // Step 2: Owner rejects
        vm.prank(complianceOwner);
        l1Compliance.reject(id);

        // Step 3: Anyone settles → ETH refunded to alice
        uint256 aliceBefore = alice.balance;

        vm.expectEmit(address(l1Compliance));
        emit Refunded(id);

        l1Compliance.settle(alice, target, 0, mint, gasLimit, false, txData, 0);

        assertEq(alice.balance, aliceBefore + mint);
        assertEq(address(l1Compliance).balance, 0);
    }

    function test_e2e_hashConsistency_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);

        vm.prank(complianceOwner);
        l1Compliance.addRule(address(rulePending));

        uint256 mint = 1 ether;
        address target = makeAddr("target");
        uint256 txValue = 0.5 ether;
        bytes memory txData = hex"cafe";
        uint64 gasLimit = _minGasLimit(txData);

        // check() hashes: (msg.sender, _to, _value, msg.value, _gasLimit, _isCreation, _data, _nonce)
        // For deposits: msg.sender=alice, _value=txValue, msg.value=mint, _nonce=0
        bytes32 expectedId = keccak256(abi.encode(alice, target, txValue, mint, gasLimit, false, txData, uint256(0)));

        vm.deal(alice, mint);
        vm.prank(alice, alice);
        optimismPortal2.depositTransaction{ value: mint }({
            _to: target,
            _value: txValue,
            _gasLimit: gasLimit,
            _isCreation: false,
            _data: txData
        });

        // Verify the compliance has this id stored
        (, ICompliance.Status s) = l1Compliance.status(expectedId);
        assertEq(uint256(s), uint256(ICompliance.Status.Pending));

        // settle() hashes: (_from, _to, _value, _mint, _gasLimit, _isCreation, _data, _nonce)
        // _mint is the 4th abi.encode field, matching msg.value from check()
        vm.prank(complianceOwner);
        l1Compliance.approve(expectedId);

        // This should succeed because settle hash == check hash
        l1Compliance.settle(alice, target, txValue, mint, gasLimit, false, txData, 0);
    }
}
