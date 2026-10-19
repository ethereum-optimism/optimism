// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";

// Interfaces
import { IBridgeHook } from "interfaces/L1/IBridgeHook.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";

/// @title OptimismPortal2_CompleteDepositTransaction_Test
/// @notice Tests completing deposits held by the bridge hook.
contract OptimismPortal2_CompleteDepositTransaction_Test is CommonTest {
    /// @notice Gas limit of each deposit. Two fit under the default 20m cap, three do not.
    uint64 internal constant GAS_LIMIT = 8_000_000;

    address internal hook = makeAddr("hook");

    /// @notice Nonce the Portal assigns to the next screened deposit.
    uint256 internal nextNonce;

    function setUp() public override {
        super.setUp();
        vm.etch(hook, hex"00");
        vm.prank(proxyAdminOwner);
        optimismPortal2.setBridgeHook(IBridgeHook(hook));
    }

    /// @notice Sends a deposit from alice and returns the item the Portal builds for it.
    function _deposit() internal returns (BridgeHookTypes.Item memory item_) {
        item_.direction = BridgeHookTypes.Direction.Deposit;
        item_.asset = BridgeHookTypes.Asset.ETH;
        item_.from = alice;
        item_.to = bob;
        item_.gasLimit = GAS_LIMIT;
        item_.nonce = nextNonce++;

        vm.prank(alice, alice);
        optimismPortal2.depositTransaction(bob, 0, GAS_LIMIT, false, hex"");
    }

    /// @notice The resource metering RFC example. With a 20m cap, deposits A and B of 8m are held
    ///         in two blocks, and in a third block deposit C of 8m proceeds and A completes. B would
    ///         take the gas published in that block to 24m, so its completion reverts and B stays
    ///         held. It completes in the next block.
    function test_completeDepositTransaction_overCap_reverts() external {
        vm.mockCall(hook, abi.encodeWithSelector(IBridgeHook.screenDeposit.selector), abi.encode(false));
        BridgeHookTypes.Item memory a = _deposit();
        vm.roll(block.number + 1);
        BridgeHookTypes.Item memory b = _deposit();

        vm.roll(block.number + 1);
        vm.mockCall(hook, abi.encodeWithSelector(IBridgeHook.screenDeposit.selector), abi.encode(true));
        _deposit();
        vm.prank(hook);
        optimismPortal2.completeDepositTransaction(a);

        vm.expectRevert(IOptimismPortal2.OutOfGas.selector);
        vm.prank(hook);
        optimismPortal2.completeDepositTransaction(b);
        assertTrue(optimismPortal2.heldDeposits(BridgeHookHashing.hashItem(b)));

        vm.roll(block.number + 1);
        vm.prank(hook);
        optimismPortal2.completeDepositTransaction(b);
        assertFalse(optimismPortal2.heldDeposits(BridgeHookHashing.hashItem(b)));
    }
}
