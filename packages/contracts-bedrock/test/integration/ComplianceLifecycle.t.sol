// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { ComplianceIntegration_TestInit } from "test/setup/ComplianceIntegrationTestInit.sol";

// Libraries
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { Types } from "src/libraries/Types.sol";

// Interfaces
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IVisionPolicyV1 } from "interfaces/compliance/IVisionPolicyV1.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";

/// @title ComplianceLifecycle_Integration_Test
/// @notice Tests how the hook is deployed, disabled, removed, set again and upgraded, and what
///         happens to held items across each change.
contract ComplianceLifecycle_Integration_Test is ComplianceIntegration_TestInit {
    /// @notice One held and cleared item of each kind that moves value.
    struct HeldItems {
        BridgeHookTypes.Item ethDeposit;
        BridgeHookTypes.Item erc20Deposit;
        BridgeHookTypes.Item ethWithdrawal;
        bytes32[3] ids;
        uint64 expiry;
    }

    /// @notice Tests that the deployment wires the module, the policy and the Portal to each other
    ///         and gives each role to its own address.
    function test_setUp_wiring_succeeds() external {
        assertEq(address(optimismPortal2.bridgeHook()), address(module));

        assertEq(address(module.portal()), address(optimismPortal2));
        assertEq(address(module.l1CrossDomainMessenger()), systemConfig.l1CrossDomainMessenger());
        assertEq(address(module.bridge()), systemConfig.l1StandardBridge());
        assertEq(address(module.policy()), address(policy));
        assertEq(module.owner(), moduleOwner);
        assertFalse(module.disabled());
        assertEq(moduleProxyAdmin.getProxyAdmin(payable(address(module))), address(moduleProxyAdmin));
        assertEq(moduleProxyAdmin.owner(), custodian);

        assertEq(address(policy.module()), address(module));
        assertEq(policy.owner(), policyOwner);
        assertEq(policy.verdictWriter(), verdictWriter);
        assertEq(policy.maxVerdictValidity(), MAX_VERDICT_VALIDITY);
        assertEq(policy.holdDelay(), HOLD_DELAY);

        // Neither the proxy nor its implementation can be initialized again.
        address impl = moduleProxyAdmin.getProxyImplementation(address(module));
        vm.expectRevert(IComplianceModule.InvalidInitialization.selector);
        vm.prank(address(moduleProxyAdmin));
        module.initialize(anyone, IPolicy(anyone), optimismPortal2, l1CrossDomainMessenger, l1StandardBridge);
        vm.expectRevert(IComplianceModule.InvalidInitialization.selector);
        IComplianceModule(impl)
            .initialize(anyone, IPolicy(anyone), optimismPortal2, l1CrossDomainMessenger, l1StandardBridge);
    }

    /// @notice Tests that removing the hook strands held items and stops screening, and that
    ///         setting the same module again lets the items held before complete.
    function test_removeHook_hookSetAgain_succeeds() external {
        Types.WithdrawalTransaction memory unscreened = _withdrawal(1, alice, bob, AMOUNT);
        _prove(unscreened);
        HeldItems memory held = _holdAndClearItems();

        vm.prank(optimismPortal2.proxyAdminOwner());
        optimismPortal2.removeHook();

        vm.expectRevert(IOptimismPortal2.OptimismPortal_NotBridgeHook.selector);
        module.completeDeposit(held.ethDeposit);
        vm.expectRevert(IOptimismPortal2.OptimismPortal_NotBridgeHook.selector);
        module.completeWithdrawal(held.ethWithdrawal);
        _assertItemsHeld(held);

        // Without a hook, a deposit and a withdrawal that the policy would hold proceed as on a
        // default chain.
        uint256 bobBalance = bob.balance;
        _depositETH(alice, bob, AMOUNT);
        _publishedDeposit();
        optimismPortal2.finalizeWithdrawalTransaction(unscreened);
        assertEq(bob.balance, bobBalance + AMOUNT);

        _upgradeChain(address(module));

        _assertItemsHeld(held);
        _completeItems(held);
    }

    /// @notice Tests that a Portal upgrade through OPCM keeps the hook, the held items, their
    ///         verdicts and their value, and that the items complete afterward.
    function test_upgradeChain_heldItems_succeeds() external {
        HeldItems memory held = _holdAndClearItems();

        _upgradeChain(address(0));

        assertEq(address(optimismPortal2.bridgeHook()), address(module));
        _assertItemsHeld(held);
        _completeItems(held);
    }

    /// @notice Tests that a module upgrade keeps the held items, their verdicts and their value,
    ///         and that the items complete afterward.
    function test_upgradeModule_heldItems_succeeds() external {
        HeldItems memory held = _holdAndClearItems();

        address newImpl = _deployModuleImpl();
        vm.prank(custodian);
        moduleProxyAdmin.upgrade(payable(address(module)), newImpl);

        assertEq(moduleProxyAdmin.getProxyImplementation(address(module)), newImpl);
        assertEq(module.owner(), moduleOwner);
        assertEq(address(module.policy()), address(policy));
        _assertItemsHeld(held);
        _completeItems(held);
    }

    /// @notice Tests that replacing the policy keeps held items and the verdicts the old policy
    ///         recorded, and that the old policy can no longer record verdicts.
    function test_setPolicy_heldItems_succeeds() external {
        HeldItems memory held = _holdAndClearItems();
        IVisionPolicyV1 newPolicy = _deployPolicy(policyOwner, verdictWriter);

        vm.prank(moduleOwner);
        module.setPolicy(IPolicy(address(newPolicy)));

        vm.expectRevert(IComplianceModule.ComplianceModule_CallerNotPolicy.selector);
        vm.prank(verdictWriter);
        policy.clear(held.ids[0], held.expiry);

        _assertItemsHeld(held);
        _completeItems(held);
    }

    /// @notice Tests that disabling the module lets new deposits through unscreened but leaves a
    ///         held deposit waiting for a verdict, and that enabling it restores screening.
    function test_disable_deposits_succeeds() external {
        _depositETH(alice, bob, AMOUNT);
        (bytes32 id, BridgeHookTypes.Item memory item) = _heldDeposit();

        vm.prank(moduleOwner);
        module.disable(0);

        _depositETH(alice, bob, AMOUNT);
        _publishedDeposit();

        vm.expectRevert(IComplianceModule.ComplianceModule_ItemNotCleared.selector);
        module.completeDeposit(item);
        _assertHeldDeposit(id);

        vm.prank(moduleOwner);
        module.enable();

        _depositETH(alice, bob, AMOUNT);
        _heldDeposit();

        _clear(id);
        module.completeDeposit(item);
        _publishedDeposit();
        _assertReleased(id);
    }

    /// @notice Tests that disabling the module pays withdrawals initiated at or after the cutoff
    ///         with no verdict, while a withdrawal initiated before it is still screened.
    function test_disable_withdrawals_succeeds() external {
        Types.WithdrawalTransaction memory beforeCutoff = _withdrawal(5, alice, bob, AMOUNT);
        Types.WithdrawalTransaction memory atCutoff = _withdrawal(6, alice, bob, AMOUNT);
        (bytes32 id,) = _itemOf(beforeCutoff);
        _prove(beforeCutoff);
        uint256 holdableAt = _warpToFinalizable(_prove(atCutoff)) + HOLD_DELAY;
        uint256 bobBalance = bob.balance;

        vm.prank(moduleOwner);
        module.disable(atCutoff.nonce);

        optimismPortal2.finalizeWithdrawalTransaction(atCutoff);
        assertEq(bob.balance, bobBalance + AMOUNT);

        vm.expectRevert(abi.encodeWithSelector(IVisionPolicyV1.VisionPolicyV1_NotYetHoldable.selector, holdableAt));
        optimismPortal2.finalizeWithdrawalTransaction(beforeCutoff);

        _clear(id);
        optimismPortal2.finalizeWithdrawalTransaction(beforeCutoff);
        assertEq(bob.balance, bobBalance + 2 * AMOUNT);
    }

    /// @notice Holds an ETH deposit, a token deposit and an ETH withdrawal to bob, and records a
    ///         clear verdict for each.
    function _holdAndClearItems() internal returns (HeldItems memory held_) {
        Types.WithdrawalTransaction memory withdrawal = _withdrawal(0, alice, bob, AMOUNT);
        _proveAndWarpToHoldable(withdrawal);
        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);
        (held_.ids[2], held_.ethWithdrawal) = _itemOf(withdrawal);

        _depositETH(alice, bob, AMOUNT);
        (held_.ids[0], held_.ethDeposit) = _heldDeposit();
        _depositERC20(alice, bob, AMOUNT);
        (held_.ids[1], held_.erc20Deposit) = _heldDeposit();

        held_.expiry = _expiry();
        for (uint256 i; i < held_.ids.length; i++) {
            _clear(held_.ids[i]);
        }
    }

    /// @notice Asserts that every item is still held with its verdict and its value.
    function _assertItemsHeld(HeldItems memory _held) internal view {
        _assertHeldDeposit(_held.ids[0]);
        _assertHeldDeposit(_held.ids[1]);
        _assertHeldWithdrawal(_held.ids[2]);
        for (uint256 i; i < _held.ids.length; i++) {
            assertEq(module.items(_held.ids[i]).expiresAt, _held.expiry, "verdict");
        }
        assertEq(address(module).balance, 2 * AMOUNT, "held ETH");
        _assertTokenCustody({ _escrowed: 0, _held: AMOUNT });
    }

    /// @notice Completes every item and asserts that its value reached its destination.
    function _completeItems(HeldItems memory _held) internal {
        uint256 vault = _ethVault().balance;
        uint256 bobBalance = bob.balance;

        vm.startPrank(anyone);
        module.completeDeposit(_held.ethDeposit);
        _publishedDeposit();
        module.completeDeposit(_held.erc20Deposit);
        _publishedDeposit();
        module.completeWithdrawal(_held.ethWithdrawal);
        vm.stopPrank();

        for (uint256 i; i < _held.ids.length; i++) {
            _assertReleased(_held.ids[i]);
        }
        assertEq(address(module).balance, 0, "held ETH");
        assertEq(_ethVault().balance, vault + AMOUNT, "locked ETH");
        assertEq(bob.balance, bobBalance + AMOUNT, "withdrawn ETH");
        _assertTokenCustody({ _escrowed: AMOUNT, _held: 0 });
    }
}
