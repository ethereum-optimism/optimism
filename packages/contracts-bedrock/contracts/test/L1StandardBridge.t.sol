//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { Bridge_Initializer } from "./CommonTest.t.sol";
import { StandardBridge } from "../universal/StandardBridge.sol";
import { L2StandardBridge } from "../L2/L2StandardBridge.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";
import { PredeployAddresses } from "../libraries/PredeployAddresses.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { stdStorage, StdStorage } from "forge-std/Test.sol";

contract L1StandardBridge_Test is Bridge_Initializer {
    using stdStorage for StdStorage;

    function setUp() public override {
        super.setUp();
    }

    function test_initialize() external {
        assertEq(
            address(L1Bridge.messenger()),
            address(L1Messenger)
        );

        assertEq(
            address(L1Bridge.otherBridge()),
            PredeployAddresses.L2_STANDARD_BRIDGE
        );

        assertEq(
            address(L2Bridge),
            PredeployAddresses.L2_STANDARD_BRIDGE
        );
    }

    // receive
    // - can accept ETH
    function test_receive() external {
        assertEq(address(op).balance, 0);

        vm.expectEmit(true, true, true, true);
        emit ETHBridgeInitiated(alice, alice, 100, hex"");

        vm.expectCall(
            address(L1Messenger),
            abi.encodeWithSelector(
                CrossDomainMessenger.sendMessage.selector,
                address(L2Bridge),
                abi.encodeWithSelector(
                    StandardBridge.finalizeBridgeETH.selector,
                    alice,
                    alice,
                    100,
                    hex""
                ),
                200_000
            )
        );

        vm.prank(alice, alice);
        address(L1Bridge).call{ value: 100 }(hex"");
        assertEq(address(op).balance, 100);
    }

    // depositETH
    // - emits ETHDepositInitiated
    // - calls optimismPortal.depositTransaction
    // - only EOA
    // - ETH ends up in the optimismPortal
    function test_depositETH() external {
        assertEq(address(op).balance, 0);

        vm.expectEmit(true, true, true, true);
        emit ETHBridgeInitiated(alice, alice, 500, hex"ff");

        vm.expectCall(
            address(L1Messenger),
            abi.encodeWithSelector(
                CrossDomainMessenger.sendMessage.selector,
                address(L2Bridge),
                abi.encodeWithSelector(
                    StandardBridge.finalizeBridgeETH.selector,
                    alice,
                    alice,
                    500,
                    hex"ff"
                ),
                50000
            )
        );

        vm.prank(alice, alice);
        L1Bridge.depositETH{ value: 500 }(50000, hex"ff");
        assertEq(address(op).balance, 500);
    }

    function test_onlyEOADepositETH() external {
        // turn alice into a contract
        vm.etch(alice, address(L1Token).code);

        vm.expectRevert("Account not EOA");
        vm.prank(alice);
        L1Bridge.depositETH{ value: 1 }(300, hex"");
    }

    // depositETHTo
    // - emits ETHDepositInitiated
    // - calls optimismPortal.depositTransaction
    // - EOA or contract can call
    // - ETH ends up in the optimismPortal
    function test_depositETHTo() external {
        assertEq(address(op).balance, 0);

        vm.expectEmit(true, true, true, true);
        emit ETHDepositInitiated(alice, bob, 600, hex"dead");

        vm.expectEmit(true, true, true, true);
        emit ETHBridgeInitiated(alice, bob, 600, hex"dead");

        // depositETHTo on the L1 bridge should be called
        vm.expectCall(
            address(L1Bridge),
            abi.encodeWithSelector(
                L1Bridge.depositETHTo.selector,
                bob,
                1000,
                hex"dead"
            )
        );

        // the L1 bridge should call
        // L1CrossDomainMessenger.sendMessage
        vm.expectCall(
            address(L1Messenger),
            abi.encodeWithSelector(
                CrossDomainMessenger.sendMessage.selector,
                address(L2Bridge),
                abi.encodeWithSelector(
                    StandardBridge.finalizeBridgeETH.selector,
                    alice,
                    bob,
                    600,
                    hex"dead"
                ),
                1000
            )
        );

        // TODO: assert on OptimismPortal being called
        // and the event being emitted correctly

        // deposit eth to bob
        vm.prank(alice, alice);
        L1Bridge.depositETHTo{ value: 600 }(bob, 1000, hex"dead");
    }

    // depositERC20
    // - updates bridge.deposits
    // - emits ERC20DepositInitiated
    // - calls optimismPortal.depositTransaction
    // - only callable by EOA
    function test_depositERC20() external {
        vm.expectEmit(true, true, true, true);
        emit ERC20DepositInitiated(
            address(L1Token),
            address(L2Token),
            alice,
            alice,
            100,
            hex""
        );

        deal(address(L1Token), alice, 100000, true);

        vm.prank(alice);
        L1Token.approve(address(L1Bridge), type(uint256).max);

        // The L1Bridge should transfer alice's tokens
        // to itself
        vm.expectCall(
            address(L1Token),
            abi.encodeWithSelector(
                ERC20.transferFrom.selector,
                alice,
                address(L1Bridge),
                100
            )
        );

        // TODO: optimismPortal.depositTransaction call + event

        vm.prank(alice);
        L1Bridge.depositERC20(
            address(L1Token),
            address(L2Token),
            100,
            10000,
            hex""
        );

        assertEq(L1Bridge.deposits(address(L1Token), address(L2Token)), 100);
    }

    function test_onlyEOADepositERC20() external {
        // turn alice into a contract
        vm.etch(alice, hex"ffff");

        vm.expectRevert("Account not EOA");
        vm.prank(alice, alice);
        L1Bridge.depositERC20(
            address(0),
            address(0),
            100,
            100,
            hex""
        );
    }

    // depositERC20To
    // - updates bridge.deposits
    // - emits ERC20DepositInitiated
    // - calls optimismPortal.depositTransaction
    // - callable by a contract
    function test_depositERC20To() external {
        vm.expectEmit(true, true, true, true);
        emit ERC20DepositInitiated(
            address(L1Token),
            address(L2Token),
            alice,
            bob,
            1000,
            hex""
        );

        deal(address(L1Token), alice, 100000, true);

        vm.prank(alice);
        L1Token.approve(address(L1Bridge), type(uint256).max);

        vm.expectCall(
            address(L1Token),
            abi.encodeWithSelector(
                ERC20.transferFrom.selector,
                alice,
                address(L1Bridge),
                1000
            )
        );

        vm.prank(alice);
        L1Bridge.depositERC20To(
            address(L1Token),
            address(L2Token),
            bob,
            1000,
            10000,
            hex""
        );

        assertEq(L1Bridge.deposits(address(L1Token), address(L2Token)), 1000);
    }

    // finalizeETHWithdrawal
    // - emits ETHWithdrawalFinalized
    // - only callable by L2 bridge
    function test_finalizeETHWithdrawal() external {
        uint256 aliceBalance = alice.balance;

        vm.expectEmit(true, true, true, true);
        emit ETHWithdrawalFinalized(
            alice,
            alice,
            100,
            hex""
        );

        vm.expectCall(
            alice,
            hex""
        );

        vm.mockCall(
            address(L1Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L1Bridge.otherBridge()))
        );
        // ensure that the messenger has ETH to call with
        vm.deal(address(L1Bridge.messenger()), 100);
        vm.prank(address(L1Bridge.messenger()));
        L1Bridge.finalizeETHWithdrawal{ value: 100 }(
            alice,
            alice,
            100,
            hex""
        );

        assertEq(address(L1Bridge.messenger()).balance, 0);
        assertEq(aliceBalance + 100, alice.balance);
    }

    // finalizeERC20Withdrawal
    // - updates bridge.deposits
    // - emits ERC20WithdrawalFinalized
    // - only callable by L2 bridge
    function test_finalizeERC20Withdrawal() external {
        deal(address(L1Token), address(L1Bridge), 100, true);

        uint256 slot = stdstore
            .target(address(L1Bridge))
            .sig("deposits(address,address)")
            .with_key(address(L1Token))
            .with_key(address(L2Token))
            .find();

        // Give the L1 bridge some ERC20 tokens
        vm.store(address(L1Bridge), bytes32(slot), bytes32(uint256(100)));
        assertEq(L1Bridge.deposits(address(L1Token), address(L2Token)), 100);

        vm.expectEmit(true, true, true, true);
        emit ERC20WithdrawalFinalized(
            address(L1Token),
            address(L2Token),
            alice,
            alice,
            100,
            hex""
        );

        vm.expectCall(
            address(L1Token),
            abi.encodeWithSelector(
                ERC20.transfer.selector,
                alice,
                100
            )
        );

        vm.mockCall(
            address(L1Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L1Bridge.otherBridge()))
        );
        vm.prank(address(L1Bridge.messenger()));
        L1Bridge.finalizeERC20Withdrawal(
            address(L1Token),
            address(L2Token),
            alice,
            alice,
            100,
            hex""
        );

        assertEq(L1Token.balanceOf(address(L1Bridge)), 0);
        assertEq(L1Token.balanceOf(address(alice)), 100);
    }

    function test_onlyPortalFinalizeERC20Withdrawal() external {
        vm.mockCall(
            address(L1Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L1Bridge.otherBridge()))
        );
        vm.prank(address(28));
        vm.expectRevert("Could not authenticate bridge message.");
        L1Bridge.finalizeERC20Withdrawal(
            address(L1Token),
            address(L2Token),
            alice,
            alice,
            100,
            hex""
        );
    }

    function test_onlyL2BridgeFinalizeERC20Withdrawal() external {
        vm.mockCall(
            address(L1Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(address(0)))
        );
        vm.prank(address(L1Bridge.messenger()));
        vm.expectRevert("Could not authenticate bridge message.");
        L1Bridge.finalizeERC20Withdrawal(
            address(L1Token),
            address(L2Token),
            alice,
            alice,
            100,
            hex""
        );
    }
}
