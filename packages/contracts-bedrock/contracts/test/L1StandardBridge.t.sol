// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Bridge_Initializer } from "./CommonTest.t.sol";
import { StandardBridge } from "../universal/StandardBridge.sol";
import { L2StandardBridge } from "../L2/L2StandardBridge.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";
import { Predeploys } from "../libraries/Predeploys.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { stdStorage, StdStorage } from "forge-std/Test.sol";

contract L1StandardBridge_Getter_Test is Bridge_Initializer {
    function test_getters_success() external {
        assert(L1Bridge.l2TokenBridge() == address(L2Bridge));
        assert(L1Bridge.OTHER_BRIDGE() == L2Bridge);
        assert(L1Bridge.messenger() == L1Messenger);
        assert(L1Bridge.MESSENGER() == L1Messenger);
        assertEq(L1Bridge.version(), "0.0.2");
    }
}

contract L1StandardBridge_Initialize_Test is Bridge_Initializer {
    function test_initialize_success() external {
        assertEq(address(L1Bridge.messenger()), address(L1Messenger));

        assertEq(address(L1Bridge.OTHER_BRIDGE()), Predeploys.L2_STANDARD_BRIDGE);

        assertEq(address(L2Bridge), Predeploys.L2_STANDARD_BRIDGE);
    }
}

contract L1StandardBridge_Initialize_TestFail is Bridge_Initializer {}

contract L1StandardBridge_Receive_Test is Bridge_Initializer {
    // receive
    // - can accept ETH
    function test_receive_success() external {
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
        (bool success, ) = address(L1Bridge).call{ value: 100 }(hex"");
        assertEq(success, true);
        assertEq(address(op).balance, 100);
    }
}

contract L1StandardBridge_Receive_TestFail {}

contract L1StandardBridge_DepositETH_Test is Bridge_Initializer {
    // depositETH
    // - emits ETHDepositInitiated
    // - calls optimismPortal.depositTransaction
    // - only EOA
    // - ETH ends up in the optimismPortal
    function test_depositETH_success() external {
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
}

contract L1StandardBridge_DepositETH_TestFail is Bridge_Initializer {
    function test_DepositETH_revert_notEoa() external {
        // turn alice into a contract
        vm.etch(alice, address(L1Token).code);

        vm.expectRevert("StandardBridge: function can only be called from an EOA");
        vm.prank(alice);
        L1Bridge.depositETH{ value: 1 }(300, hex"");
    }
}

contract L1StandardBridge_DepositETHTo_Test is Bridge_Initializer {
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
            abi.encodeWithSelector(L1Bridge.depositETHTo.selector, bob, 1000, hex"dead")
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
}

contract L1StandardBridge_DepositETHTo_TestFail is Bridge_Initializer {}

contract L1StandardBridge_DepositERC20_Test is Bridge_Initializer {
    using stdStorage for StdStorage;

    // depositERC20
    // - updates bridge.deposits
    // - emits ERC20DepositInitiated
    // - calls optimismPortal.depositTransaction
    // - only callable by EOA
    function test_depositERC20_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit ERC20DepositInitiated(address(L1Token), address(L2Token), alice, alice, 100, hex"");

        deal(address(L1Token), alice, 100000, true);

        vm.prank(alice);
        L1Token.approve(address(L1Bridge), type(uint256).max);

        // The L1Bridge should transfer alice's tokens
        // to itself
        vm.expectCall(
            address(L1Token),
            abi.encodeWithSelector(ERC20.transferFrom.selector, alice, address(L1Bridge), 100)
        );

        // TODO: optimismPortal.depositTransaction call + event

        vm.prank(alice);
        L1Bridge.depositERC20(address(L1Token), address(L2Token), 100, 10000, hex"");

        assertEq(L1Bridge.deposits(address(L1Token), address(L2Token)), 100);
    }
}

contract L1StandardBridge_DepositERC20_TestFail is Bridge_Initializer {
    function test_depositERC20_revert_notEoa() external {
        // turn alice into a contract
        vm.etch(alice, hex"ffff");

        vm.expectRevert("StandardBridge: function can only be called from an EOA");
        vm.prank(alice, alice);
        L1Bridge.depositERC20(address(0), address(0), 100, 100, hex"");
    }
}

contract L1StandardBridge_DepositERC20To_Test is Bridge_Initializer {
    // depositERC20To
    // - updates bridge.deposits
    // - emits ERC20DepositInitiated
    // - calls optimismPortal.depositTransaction
    // - callable by a contract
    function test_depositERC20To_success() external {
        vm.expectEmit(true, true, true, true);
        emit ERC20DepositInitiated(address(L1Token), address(L2Token), alice, bob, 1000, hex"");

        deal(address(L1Token), alice, 100000, true);

        vm.prank(alice);
        L1Token.approve(address(L1Bridge), type(uint256).max);

        vm.expectCall(
            address(L1Token),
            abi.encodeWithSelector(ERC20.transferFrom.selector, alice, address(L1Bridge), 1000)
        );

        vm.prank(alice);
        L1Bridge.depositERC20To(address(L1Token), address(L2Token), bob, 1000, 10000, hex"");

        assertEq(L1Bridge.deposits(address(L1Token), address(L2Token)), 1000);
    }
}

contract L1StandardBridge_FinalizeETHWithdrawal_Test is Bridge_Initializer {
    using stdStorage for StdStorage;

    // finalizeETHWithdrawal
    // - emits ETHWithdrawalFinalized
    // - only callable by L2 bridge
    function test_finalizeETHWithdrawal() external {
        uint256 aliceBalance = alice.balance;

        vm.expectEmit(true, true, true, true);
        emit ETHWithdrawalFinalized(alice, alice, 100, hex"");

        vm.expectCall(alice, hex"");

        vm.mockCall(
            address(L1Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L1Bridge.OTHER_BRIDGE()))
        );
        // ensure that the messenger has ETH to call with
        vm.deal(address(L1Bridge.messenger()), 100);
        vm.prank(address(L1Bridge.messenger()));
        L1Bridge.finalizeETHWithdrawal{ value: 100 }(alice, alice, 100, hex"");

        assertEq(address(L1Bridge.messenger()).balance, 0);
        assertEq(aliceBalance + 100, alice.balance);
    }
}

contract L1StandardBridge_FinalizeETHWithdrawal_TestFail is Bridge_Initializer {}

contract L1StandardBridge_FinalizeERC20Withdrawal_Test is Bridge_Initializer {
    using stdStorage for StdStorage;

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
        emit ERC20WithdrawalFinalized(address(L1Token), address(L2Token), alice, alice, 100, hex"");

        vm.expectCall(
            address(L1Token),
            abi.encodeWithSelector(ERC20.transfer.selector, alice, 100)
        );

        vm.mockCall(
            address(L1Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L1Bridge.OTHER_BRIDGE()))
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
}

contract L1StandardBridge_FinalizeERC20Withdrawal_TestFail is Bridge_Initializer {
    function test_finalizeERC20Withdrawal_revert_notMessenger() external {
        vm.mockCall(
            address(L1Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L1Bridge.OTHER_BRIDGE()))
        );
        vm.prank(address(28));
        vm.expectRevert("StandardBridge: function can only be called from the other bridge");
        L1Bridge.finalizeERC20Withdrawal(
            address(L1Token),
            address(L2Token),
            alice,
            alice,
            100,
            hex""
        );
    }

    function test_finalizeERC20Withdrawal_revert_notOtherBridge() external {
        vm.mockCall(
            address(L1Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(address(0)))
        );
        vm.prank(address(L1Bridge.messenger()));
        vm.expectRevert("StandardBridge: function can only be called from the other bridge");
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

// Todo: move these next two contracts into a test file specific to the direction agnostic
// StandardBridge interface
contract L1StandardBridge_FinalizeBridgeETH_Test is Bridge_Initializer {

}

contract L1StandardBridge_FinalizeBridgeETH_TestFail is Bridge_Initializer {
    function test_finalizeBridgeETH_revert_incorrectValue() external {
        address messenger = address(L1Bridge.messenger());
        vm.mockCall(
            messenger,
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L1Bridge.OTHER_BRIDGE()))
        );
        vm.deal(messenger, 100);
        vm.prank(messenger);
        vm.expectRevert("StandardBridge: amount sent does not match amount required");
        L1Bridge.finalizeBridgeETH{ value: 50 }(alice, alice, 100, hex"");
    }

    function test_finalizeBridgeETH_revert_sendToSelf() external {
        address messenger = address(L1Bridge.messenger());
        vm.mockCall(
            messenger,
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L1Bridge.OTHER_BRIDGE()))
        );
        vm.deal(messenger, 100);
        vm.prank(messenger);
        vm.expectRevert("StandardBridge: cannot send to self");
        L1Bridge.finalizeBridgeETH{ value: 100 }(alice, address(L1Bridge), 100, hex"");
    }

    function test_finalizeBridgeETH_revert_sendToMessenger() external {
        address messenger = address(L1Bridge.messenger());
        vm.mockCall(
            messenger,
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L1Bridge.OTHER_BRIDGE()))
        );
        vm.deal(messenger, 100);
        vm.prank(messenger);
        vm.expectRevert("StandardBridge: cannot send to messenger");
        L1Bridge.finalizeBridgeETH{ value: 100 }(alice, messenger, 100, hex"");
    }
}
