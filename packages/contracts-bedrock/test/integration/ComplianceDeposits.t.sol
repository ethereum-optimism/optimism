// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { ComplianceIntegration_TestInit } from "test/setup/ComplianceIntegrationTestInit.sol";

// Contracts
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

// Libraries
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";

// Interfaces
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";

/// @title ComplianceDeposits_Integration_Test
/// @notice Tests deposits through the Portal, the bridge and the messenger while the
///         ComplianceModule is the Portal's hook and VisionPolicyV1 is its policy.
///
///         VisionPolicyV1 lets a deposit proceed only when its sender is on the allowlist, and
///         holds every other deposit. So alice's deposits are held unless a test allowlists her.
contract ComplianceDeposits_Integration_Test is ComplianceIntegration_TestInit {
    /// @notice Tests that a held ETH deposit completes as the same L2 transaction as a deposit
    ///         that proceeds, and that its ETH waits in the module in between.
    function test_ethDeposit_heldThenCompleted_succeeds() external {
        uint256 snapshot = vm.snapshotState();
        _allowlist(alice);
        _depositETH(alice, bob, AMOUNT);
        bytes32 direct = _publishedDeposit();
        vm.revertToState(snapshot);

        uint256 vault = _ethVault().balance;
        _depositETH(alice, bob, AMOUNT);
        (bytes32 id, BridgeHookTypes.Item memory item) = _heldDeposit();

        _assertHeldDeposit(id);
        assertEq(address(module).balance, AMOUNT);
        assertEq(_ethVault().balance, vault);

        _clear(id);
        assertEq(_completeDeposit(item), direct);
        _assertReleased(id);
        assertEq(address(module).balance, 0);
        assertEq(_ethVault().balance, vault + AMOUNT);
    }

    /// @notice Tests that a held deposit from a contract completes with the same aliased sender
    ///         as one that proceeds, while the policy screens the contract's own address.
    function test_contractDeposit_heldThenCompleted_succeeds() external {
        address depositor = makeAddr("depositorContract");
        vm.etch(depositor, hex"00");

        uint256 snapshot = vm.snapshotState();
        _allowlist(depositor);
        _depositETH(depositor, bob, AMOUNT);
        bytes32 direct = _publishedDeposit();
        vm.revertToState(snapshot);

        _depositETH(depositor, bob, AMOUNT);
        (bytes32 id, BridgeHookTypes.Item memory item) = _heldDeposit();

        _assertHeldDeposit(id);
        assertTrue(item.aliased);
        assertEq(item.from, AddressAliasHelper.applyL1ToL2Alias(depositor));
        assertEq(module.parties(item).sender, depositor);

        _clear(id);
        assertEq(_completeDeposit(item), direct);
        _assertReleased(id);
    }

    /// @notice Tests that a held token deposit completes as the same L2 transaction as a deposit
    ///         that proceeds, and that its tokens leave the bridge's escrow in between.
    function test_erc20Deposit_heldThenCompleted_succeeds() external {
        uint256 snapshot = vm.snapshotState();
        // The policy screens the depositor, not the messenger that calls the Portal.
        _allowlist(alice);
        _depositERC20(alice, bob, AMOUNT);
        bytes32 direct = _publishedDeposit();
        _assertTokenCustody({ _escrowed: AMOUNT, _held: 0 });
        vm.revertToState(snapshot);

        _depositERC20(alice, bob, AMOUNT);
        (bytes32 id, BridgeHookTypes.Item memory item) = _heldDeposit();

        _assertHeldDeposit(id);
        _assertTokenCustody({ _escrowed: 0, _held: AMOUNT });

        _clear(id);
        assertEq(_completeDeposit(item), direct);
        _assertReleased(id);
        _assertTokenCustody({ _escrowed: AMOUNT, _held: 0 });
        assertEq(L1Token.allowance(address(module), address(l1StandardBridge)), 0);
    }

    /// @notice Tests that a held message with no value completes as the same L2 transaction as a
    ///         message that proceeds.
    function test_messageDeposit_heldThenCompleted_succeeds() external {
        uint256 snapshot = vm.snapshotState();
        _allowlist(alice);
        _depositMessage(alice, bob, hex"1234");
        bytes32 direct = _publishedDeposit();
        vm.revertToState(snapshot);

        _depositMessage(alice, bob, hex"1234");
        (bytes32 id, BridgeHookTypes.Item memory item) = _heldDeposit();

        _assertHeldDeposit(id);
        assertEq(address(module).balance, 0);

        _clear(id);
        assertEq(_completeDeposit(item), direct);
        _assertReleased(id);
    }

    /// @notice Tests that only the message of a held mintable token deposit waits. The bridge
    ///         burns the tokens at submission, so the module never takes custody of them.
    function test_mintableTokenDeposit_heldThenCompleted_succeeds() external {
        address mintable = l1OptimismMintableERC20Factory.createStandardL2Token(address(NativeL2Token), "L1", "L1");
        _depositToken(mintable, address(NativeL2Token), alice, bob, AMOUNT);
        (bytes32 id, BridgeHookTypes.Item memory item) = _heldDeposit();

        _assertHeldDeposit(id);
        assertEq(ERC20(mintable).totalSupply(), 0);
        assertEq(module.heldTokensTotal(mintable), 0);

        _clear(id);
        _completeDeposit(item);
        _assertReleased(id);
        assertEq(ERC20(mintable).totalSupply(), 0);
    }

    /// @notice Tests that the verdict writer clears and completes held ETH and token deposits in
    ///         one transaction.
    function test_clearAndCompleteDeposit_batch_succeeds() external {
        uint256 vault = _ethVault().balance;
        bytes[] memory calls = new bytes[](2);
        bytes32[] memory ids = new bytes32[](2);
        BridgeHookTypes.Item memory item;

        _depositETH(alice, bob, AMOUNT);
        (ids[0], item) = _heldDeposit();
        calls[0] = abi.encodeCall(policy.clearAndCompleteDeposit, (item, _expiry()));
        _depositERC20(alice, bob, AMOUNT);
        (ids[1], item) = _heldDeposit();
        calls[1] = abi.encodeCall(policy.clearAndCompleteDeposit, (item, _expiry()));

        vm.prank(verdictWriter);
        policy.multicall(calls);

        assertEq(_publishedDepositCount(), 2);
        _assertReleased(ids[0]);
        _assertReleased(ids[1]);
        assertEq(address(module).balance, 0);
        assertEq(_ethVault().balance, vault + AMOUNT);
        _assertTokenCustody({ _escrowed: AMOUNT, _held: 0 });
    }

    /// @notice Tests that one deposit that cannot complete reverts the whole batch, and that the
    ///         other deposit completes when the writer sends the batch again without it.
    function test_clearAndCompleteDeposit_batchResentWithoutDenied_succeeds() external {
        address carol = makeAddr("carol");
        bytes[] memory calls = new bytes[](2);
        bytes32[] memory ids = new bytes32[](2);
        BridgeHookTypes.Item memory item;

        _depositETH(alice, bob, AMOUNT);
        (ids[0], item) = _heldDeposit();
        calls[0] = abi.encodeCall(policy.clearAndCompleteDeposit, (item, _expiry()));
        _depositETH(alice, carol, AMOUNT);
        (ids[1], item) = _heldDeposit();
        calls[1] = abi.encodeCall(policy.clearAndCompleteDeposit, (item, _expiry()));
        _deny(carol);

        vm.expectRevert(IComplianceModule.ComplianceModule_PolicyRefused.selector);
        vm.prank(verdictWriter);
        policy.multicall(calls);
        // Foundry records the logs of the reverted batch, so discard them.
        vm.getRecordedLogs();

        bytes[] memory resent = new bytes[](1);
        resent[0] = calls[0];
        vm.prank(verdictWriter);
        policy.multicall(resent);

        assertEq(_publishedDepositCount(), 1);
        _assertReleased(ids[0]);
        _assertHeldDeposit(ids[1]);
        assertEq(address(module).balance, AMOUNT);
    }

    /// @notice Tests that the deny list beats both the allowlist and a clear verdict. A refused
    ///         completion changes nothing, and the deposit completes once the entry is removed.
    function test_completeDeposit_afterRecipientUndenied_succeeds() external {
        _allowlist(alice);
        _deny(bob);
        _depositETH(alice, bob, AMOUNT);
        (bytes32 id, BridgeHookTypes.Item memory item) = _heldDeposit();
        _clear(id);
        uint64 expiry = _expiry();

        vm.expectRevert(IComplianceModule.ComplianceModule_PolicyRefused.selector);
        module.completeDeposit(item);

        _assertHeldDeposit(id);
        assertEq(module.items(id).expiresAt, expiry);
        assertEq(address(module).balance, AMOUNT);

        _undeny(bob);
        _completeDeposit(item);
        _assertReleased(id);
    }

    /// @notice Tests that a held deposit does not complete on a revoked or an expired verdict,
    ///         and completes on a new one.
    function test_completeDeposit_afterVerdictRenewed_succeeds() external {
        _depositETH(alice, bob, AMOUNT);
        (bytes32 id, BridgeHookTypes.Item memory item) = _heldDeposit();

        _clear(id);
        vm.prank(verdictWriter);
        policy.revoke(id);
        vm.expectRevert(IComplianceModule.ComplianceModule_ItemNotCleared.selector);
        module.completeDeposit(item);

        _clear(id);
        vm.warp(_expiry());
        vm.expectRevert(IComplianceModule.ComplianceModule_PolicyRefused.selector);
        module.completeDeposit(item);

        _assertHeldDeposit(id);
        assertEq(address(module).balance, AMOUNT);

        _clear(id);
        _completeDeposit(item);
        _assertReleased(id);
    }

    /// @notice Tests that two deposits with the same fields are separate items. Each completes
    ///         once, and completing one cannot spend the other's ETH.
    function test_completeDeposit_identicalDeposits_succeeds() external {
        _depositETH(alice, bob, AMOUNT);
        (bytes32 firstId, BridgeHookTypes.Item memory first) = _heldDeposit();
        _depositETH(alice, bob, AMOUNT);
        (bytes32 secondId, BridgeHookTypes.Item memory second) = _heldDeposit();
        assertNotEq(firstId, secondId);

        _clear(firstId);
        _completeDeposit(first);

        // A new verdict cannot run the completed deposit again.
        _clear(firstId);
        vm.expectRevert(IComplianceModule.ComplianceModule_ItemNotHeld.selector);
        module.completeDeposit(first);

        _assertHeldDeposit(secondId);
        assertEq(address(module).balance, AMOUNT);

        _clear(secondId);
        _completeDeposit(second);
        assertEq(address(module).balance, 0);
    }

    /// @notice Tests that completions count against the deposit gas limit of the block they are
    ///         published in. A completion that does not fit stays held and fits in the next block.
    function test_completeDeposit_retryInNextBlock_succeeds() external {
        // Two deposits fit in one block each, but not together.
        uint64 gasLimit = systemConfig.resourceConfig().maxResourceLimit / 2 + 1;
        bytes32[] memory ids = new bytes32[](2);
        BridgeHookTypes.Item[] memory items = new BridgeHookTypes.Item[](2);
        for (uint256 i; i < 2; i++) {
            vm.prank(alice, alice);
            optimismPortal2.depositTransaction(bob, 0, gasLimit, false, hex"");
            (ids[i], items[i]) = _heldDeposit();
            _clear(ids[i]);
            vm.roll(block.number + 1);
        }

        _completeDeposit(items[0]);
        vm.expectRevert(IResourceMetering.OutOfGas.selector);
        module.completeDeposit(items[1]);

        vm.roll(block.number + 1);
        _completeDeposit(items[1]);

        _assertReleased(ids[1]);
    }
}
