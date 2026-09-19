// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IETHLiquidity } from "interfaces/L2/IETHLiquidity.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

// Target contract
import { ISafeETHBridge as Bridge } from "interfaces/periphery/interop/ISafeETHBridge.sol";

/// @notice Component tests mock only messenger context/send; the Go acceptance test relays real logs.
abstract contract SafeETHBridge_TestInit is CommonTest {
    Bridge internal bridge;
    address internal MESSENGER = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;
    uint256 internal constant REMOTE = 901;
    uint256 internal constant LOCAL = 902;
    uint256 internal constant AMOUNT = 1 ether;
    uint256 internal deadline;

    function setUp() public virtual override {
        super.setUp();
        vm.chainId(LOCAL);
        Bridge implementation = Bridge(vm.deployCode("SafeETHBridge.sol:SafeETHBridge", abi.encode(LOCAL, REMOTE)));
        vm.etch(Predeploys.SUPERCHAIN_ETH_BRIDGE, address(implementation).code);
        bridge = Bridge(Predeploys.SUPERCHAIN_ETH_BRIDGE);
        vm.etch(Predeploys.ETH_LIQUIDITY, vm.getDeployedCode("ETHLiquidity.sol:ETHLiquidity"));
        vm.deal(Predeploys.ETH_LIQUIDITY, 0);
        deadline = block.timestamp + 100;
        vm.deal(alice, 100 ether);
        vm.deal(bob, 0);
        vm.mockCall(
            MESSENGER, abi.encodePacked(IL2ToL2CrossDomainMessenger.sendMessage.selector), abi.encode(bytes32(0))
        );
        _context(address(bridge), REMOTE);
    }

    function _context(address _sender, uint256 _chain) internal {
        vm.mockCall(
            MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(_sender, _chain)
        );
    }

    function _initiate(uint256 _amount) internal returns (bytes32) {
        vm.prank(alice);
        return bridge.initiateTransfer{ value: _amount }(bob, deadline);
    }

    function _incoming(
        uint256 _amount,
        uint256 _nonce
    )
        internal
        view
        returns (bytes32 id_, Bridge.Transfer memory transfer_)
    {
        transfer_ = Bridge.Transfer(alice, bob, _amount, _nonce, deadline);
        id_ = keccak256(abi.encode(REMOTE, LOCAL, address(bridge), transfer_));
    }

    function _fund(uint256 _amount) internal {
        vm.deal(Predeploys.ETH_LIQUIDITY, _amount);
    }

    function _prepare(bytes32 _id, Bridge.Transfer memory _transfer) internal {
        vm.prank(MESSENGER);
        bridge.prepareDestination(_id, _transfer);
    }

    function _ack(bytes32 _id) internal {
        vm.prank(MESSENGER);
        bridge.acknowledgePrepare(_id);
    }

    function _commit(bytes32 _id) internal {
        vm.prank(MESSENGER);
        bridge.commitDestination(_id);
    }

    function _abort(bytes32 _id) internal {
        vm.prank(alice);
        bridge.abortTransfer(_id);
    }

    function _cancel(bytes32 _id, Bridge.Transfer memory _transfer) internal {
        vm.prank(MESSENGER);
        bridge.abortDestination(_id, _transfer);
    }
}

contract SafeETHBridge_InitiateTransfer_Test is SafeETHBridge_TestInit {
    function test_initiateTransfer_escrow_succeeds() external {
        bytes32 id = _initiate(AMOUNT);
        assertEq(uint256(bridge.source(id).status), uint256(Bridge.Status.PREPARED));
        assertEq(address(bridge).balance, AMOUNT);
        assertEq(bridge.reservedLiquidity(), 0);
        assertEq(alice.balance, 99 ether);
        assertEq(bob.balance, 0);
        assertEq(bridge.nonce(), 1);
        assertEq(id, keccak256(abi.encode(LOCAL, REMOTE, address(bridge), bridge.source(id).transfer)));
    }

    function testFuzz_initiateTransfer_invalidParameters_reverts(uint8 _case) external {
        uint256 choice = _case % 4;
        vm.expectRevert(Bridge.SafeETHBridge_InvalidTransfer.selector);
        vm.prank(alice);
        bridge.initiateTransfer{ value: choice == 0 ? 0 : AMOUNT }(
            choice == 1 ? address(0) : choice == 2 ? address(bridge) : bob, choice == 3 ? block.timestamp : deadline
        );
    }

    function test_initiateTransfer_sendFailureRollsBack_reverts() external {
        vm.mockCallRevert(MESSENGER, abi.encodePacked(IL2ToL2CrossDomainMessenger.sendMessage.selector), hex"12345678");
        vm.expectRevert(bytes4(0x12345678));
        _initiate(AMOUNT);
        assertEq(alice.balance, 100 ether);
        assertEq(bridge.nonce(), 0);
        assertEq(address(bridge).balance, 0);
    }
}

contract SafeETHBridge_AcknowledgePrepare_Test is SafeETHBridge_TestInit {
    function test_acknowledgePrepare_commitOnce_succeeds() external {
        bytes32 id = _initiate(AMOUNT);
        vm.expectCall(
            MESSENGER,
            abi.encodeCall(
                IL2ToL2CrossDomainMessenger.sendMessage,
                (REMOTE, address(bridge), abi.encodeCall(bridge.commitDestination, (id)))
            ),
            1
        );
        vm.expectCall(Predeploys.ETH_LIQUIDITY, AMOUNT, abi.encodeCall(IETHLiquidity.burn, ()), 1);
        _ack(id);
        _ack(id);
        assertEq(uint256(bridge.source(id).status), uint256(Bridge.Status.COMMITTED));
        assertEq(bridge.reservedLiquidity(), 0);
        assertEq(alice.balance, 99 ether);
        assertEq(address(bridge).balance, 0);
        assertEq(Predeploys.ETH_LIQUIDITY.balance, AMOUNT);
        vm.warp(deadline + 1);
        vm.expectRevert(Bridge.SafeETHBridge_InvalidState.selector);
        _abort(id);
    }

    function test_acknowledgePrepare_expired_succeeds() external {
        bytes32 id = _initiate(AMOUNT);
        vm.warp(deadline);
        _ack(id);
        assertEq(uint256(bridge.source(id).status), uint256(Bridge.Status.PREPARED));
        assertEq(bridge.reservedLiquidity(), 0);
        _abort(id);
        assertEq(alice.balance, 100 ether);
    }

    function test_acknowledgePrepare_unknown_reverts() external {
        vm.expectRevert(Bridge.SafeETHBridge_InvalidState.selector);
        _ack(bytes32(0));
    }

    function test_acknowledgePrepare_sendFailureRollsBack_reverts() external {
        bytes32 id = _initiate(AMOUNT);
        vm.mockCallRevert(MESSENGER, abi.encodePacked(IL2ToL2CrossDomainMessenger.sendMessage.selector), hex"12345678");
        vm.expectRevert(bytes4(0x12345678));
        _ack(id);
        assertEq(uint256(bridge.source(id).status), uint256(Bridge.Status.PREPARED));
        assertEq(bridge.reservedLiquidity(), 0);
        vm.clearMockedCalls();
        vm.mockCall(
            MESSENGER, abi.encodePacked(IL2ToL2CrossDomainMessenger.sendMessage.selector), abi.encode(bytes32(0))
        );
        vm.warp(deadline);
        _abort(id);
        assertEq(alice.balance, 100 ether);
    }
}

contract SafeETHBridge_PrepareDestination_Test is SafeETHBridge_TestInit {
    function test_prepareDestination_reserveAndAckOnce_succeeds() external {
        _fund(AMOUNT);
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        vm.expectCall(
            MESSENGER,
            abi.encodeCall(
                IL2ToL2CrossDomainMessenger.sendMessage,
                (REMOTE, address(bridge), abi.encodeCall(bridge.acknowledgePrepare, (id)))
            ),
            1
        );
        _prepare(id, transfer);
        _prepare(id, transfer);
        assertEq(uint256(bridge.destination(id).status), uint256(Bridge.Status.PREPARED));
        assertEq(bridge.reservedLiquidity(), AMOUNT);
        assertEq(address(bridge).balance, 0);
        assertEq(Predeploys.ETH_LIQUIDITY.balance, AMOUNT);
        assertEq(bob.balance, 0);
    }

    function test_prepareDestination_sendFailureRollsBack_reverts() external {
        _fund(AMOUNT);
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        vm.mockCallRevert(MESSENGER, abi.encodePacked(IL2ToL2CrossDomainMessenger.sendMessage.selector), hex"12345678");
        vm.expectRevert(bytes4(0x12345678));
        _prepare(id, transfer);
        assertEq(bridge.reservedLiquidity(), 0);
        assertEq(uint256(bridge.destination(id).status), uint256(Bridge.Status.NONE));
        assertEq(Predeploys.ETH_LIQUIDITY.balance, AMOUNT);
        assertEq(bob.balance, 0);
    }

    function test_prepareDestination_cannotUseSourceEscrow_reverts() external {
        _initiate(AMOUNT);
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        vm.expectRevert(Bridge.SafeETHBridge_InsufficientLiquidity.selector);
        _prepare(id, transfer);
        assertEq(uint256(bridge.destination(id).status), uint256(Bridge.Status.NONE));
    }

    function test_prepareDestination_cannotUseReservation_reverts() external {
        _fund(AMOUNT);
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        _prepare(id, transfer);
        (id, transfer) = _incoming(AMOUNT, 1);
        vm.expectRevert(Bridge.SafeETHBridge_InsufficientLiquidity.selector);
        _prepare(id, transfer);
    }

    function test_prepareDestination_fundAndRetry_succeeds() external {
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        vm.expectRevert(Bridge.SafeETHBridge_InsufficientLiquidity.selector);
        _prepare(id, transfer);
        _fund(AMOUNT);
        _prepare(id, transfer);
        _commit(id);
        assertEq(bob.balance, AMOUNT);
    }

    function testFuzz_prepareDestination_tampering_reverts(uint8 _field) external {
        _fund(AMOUNT);
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        uint256 field = _field % 6;
        if (field == 0) transfer.sender = bob;
        if (field == 1) transfer.recipient = alice;
        if (field == 2) transfer.amount++;
        if (field == 3) transfer.nonce++;
        if (field == 4) transfer.deadline++;
        if (field == 5) id = bytes32(uint256(id) ^ 1);
        vm.expectRevert(Bridge.SafeETHBridge_InvalidTransfer.selector);
        _prepare(id, transfer);
        vm.expectRevert(Bridge.SafeETHBridge_InvalidTransfer.selector);
        _cancel(id, transfer);
    }
}

contract SafeETHBridge_CommitDestination_Test is SafeETHBridge_TestInit {
    function test_commitDestination_mintFailureRetry_succeeds() external {
        _fund(AMOUNT);
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        _prepare(id, transfer);
        vm.mockCallRevert(Predeploys.ETH_LIQUIDITY, abi.encodeCall(IETHLiquidity.mint, (AMOUNT)), hex"12345678");
        vm.expectRevert(bytes4(0x12345678));
        _commit(id);
        assertEq(uint256(bridge.destination(id).status), uint256(Bridge.Status.PREPARED));
        assertEq(bridge.reservedLiquidity(), AMOUNT);
        assertEq(bob.balance, 0);
        assertEq(Predeploys.ETH_LIQUIDITY.balance, AMOUNT);
        vm.clearMockedCalls();
        _context(address(bridge), REMOTE);
        _commit(id);
        assertEq(uint256(bridge.destination(id).status), uint256(Bridge.Status.COMMITTED));
        assertEq(bob.balance, AMOUNT);
        assertEq(bridge.reservedLiquidity(), 0);
    }

    function test_commitDestination_payoutOnce_succeeds() external {
        _fund(AMOUNT);
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        _prepare(id, transfer);
        // A reservation never expires locally, even if COMMIT is delayed for years.
        vm.warp(deadline + 3650 days);
        vm.expectCall(Predeploys.ETH_LIQUIDITY, abi.encodeCall(IETHLiquidity.mint, (AMOUNT)), 1);
        _commit(id);
        _commit(id);
        _prepare(id, transfer);
        assertEq(bob.balance, AMOUNT);
        assertEq(address(bridge).balance, 0);
        assertEq(uint256(bridge.destination(id).status), uint256(Bridge.Status.COMMITTED));
        vm.expectRevert(Bridge.SafeETHBridge_InvalidState.selector);
        _cancel(id, transfer);
    }

    function test_commitDestination_withoutPrepare_reverts() external {
        vm.expectRevert(Bridge.SafeETHBridge_InvalidState.selector);
        _commit(bytes32(0));
    }

    function test_commitDestination_revertingRecipient_succeeds() external {
        // A fallback that always reverts must not be able to strand committed ETH.
        vm.etch(bob, hex"60006000fd");
        _fund(AMOUNT);
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        _prepare(id, transfer);
        _commit(id);
        assertEq(bob.balance, AMOUNT);
    }
}

contract SafeETHBridge_AbortTransfer_Test is SafeETHBridge_TestInit {
    function test_abortTransfer_refundAndLateAck_succeeds() external {
        bytes32 id = _initiate(AMOUNT);
        vm.warp(deadline);
        _abort(id);
        _abort(id);
        _ack(id);
        assertEq(alice.balance, 100 ether);
        assertEq(address(bridge).balance, 0);
        assertEq(bridge.reservedLiquidity(), 0);
        assertEq(uint256(bridge.source(id).status), uint256(Bridge.Status.ABORTED));
    }

    function test_abortTransfer_sendFailureRollsBack_reverts() external {
        bytes32 id = _initiate(AMOUNT);
        vm.warp(deadline);
        vm.mockCallRevert(MESSENGER, abi.encodePacked(IL2ToL2CrossDomainMessenger.sendMessage.selector), hex"12345678");
        vm.expectRevert(bytes4(0x12345678));
        _abort(id);
        assertEq(uint256(bridge.source(id).status), uint256(Bridge.Status.PREPARED));
        assertEq(address(bridge).balance, AMOUNT);
        assertEq(alice.balance, 99 ether);
    }

    function test_abortTransfer_early_reverts() external {
        bytes32 id = _initiate(AMOUNT);
        vm.warp(deadline - 1);
        vm.expectRevert(Bridge.SafeETHBridge_InvalidState.selector);
        _abort(id);
    }

    function test_abortTransfer_notSender_reverts() external {
        bytes32 id = _initiate(AMOUNT);
        vm.warp(deadline);
        vm.expectRevert(Bridge.SafeETHBridge_Unauthorized.selector);
        vm.prank(bob);
        bridge.abortTransfer(id);
    }

    function test_abortTransfer_revertingSender_succeeds() external {
        vm.etch(alice, hex"60006000fd");
        bytes32 id = _initiate(AMOUNT);
        vm.warp(deadline);
        _abort(id);
        assertEq(alice.balance, 100 ether);
    }
}

contract SafeETHBridge_AbortDestination_Test is SafeETHBridge_TestInit {
    function test_abortDestination_beforePrepare_succeeds() external {
        _fund(AMOUNT);
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        _cancel(id, transfer);
        _prepare(id, transfer);
        _cancel(id, transfer);
        _commit(id);
        assertEq(uint256(bridge.destination(id).status), uint256(Bridge.Status.ABORTED));
        assertEq(bridge.reservedLiquidity(), 0);
        assertEq(bob.balance, 0);
    }

    function test_abortDestination_afterPrepare_succeeds() external {
        _fund(AMOUNT);
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        _prepare(id, transfer);
        _cancel(id, transfer);
        _cancel(id, transfer);
        _prepare(id, transfer);
        _commit(id);
        assertEq(bridge.reservedLiquidity(), 0);
        assertEq(bob.balance, 0);
        assertEq(uint256(bridge.destination(id).status), uint256(Bridge.Status.ABORTED));
    }
}

contract SafeETHBridge_Uncategorized_Test is SafeETHBridge_TestInit {
    function testFuzz_callbacks_spoofing_reverts(uint8 _callback, uint8 _spoof) external {
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        bytes[4] memory calls = [
            abi.encodeCall(bridge.prepareDestination, (id, transfer)),
            abi.encodeCall(bridge.acknowledgePrepare, (id)),
            abi.encodeCall(bridge.commitDestination, (id)),
            abi.encodeCall(bridge.abortDestination, (id, transfer))
        ];
        uint256 spoof = _spoof % 3;
        if (spoof == 1) _context(alice, REMOTE);
        if (spoof == 2) _context(address(bridge), LOCAL);
        vm.prank(spoof == 0 ? alice : MESSENGER);
        (bool success, bytes memory data) = address(bridge).call(calls[_callback % 4]);
        assertFalse(success);
        assertEq(data, abi.encodePacked(Bridge.SafeETHBridge_Unauthorized.selector));
    }

    function testFuzz_reservations_concurrentMints_succeeds(uint96 _amount, bool _abortFirst) external {
        uint256 amount = bound(_amount, 1, 10 ether);
        _fund(2 * amount);
        (bytes32 first, Bridge.Transfer memory a) = _incoming(amount, 1);
        (bytes32 second, Bridge.Transfer memory b) = _incoming(amount, 2);
        _prepare(first, a);
        _prepare(second, b);
        assertEq(bridge.reservedLiquidity(), 2 * amount);
        assertEq(bob.balance, 0);
        assertEq(Predeploys.ETH_LIQUIDITY.balance, 2 * amount);
        if (_abortFirst) _cancel(first, a);
        else _commit(first);
        // Spending one reservation cannot invalidate the other one's acknowledged capacity.
        assertGe(Predeploys.ETH_LIQUIDITY.balance, bridge.reservedLiquidity());
        assertEq(bridge.reservedLiquidity(), amount);
        _commit(second);
        assertEq(bridge.reservedLiquidity(), 0);
        assertEq(bob.balance, _abortFirst ? amount : 2 * amount);
        assertEq(bob.balance + Predeploys.ETH_LIQUIDITY.balance, 2 * amount);
    }

    function test_constructor_wrongChain_reverts() external {
        vm.expectRevert(Bridge.SafeETHBridge_InvalidTransfer.selector);
        vm.deployCode("SafeETHBridge.sol:SafeETHBridge", abi.encode(1, 2));
        vm.expectRevert(Bridge.SafeETHBridge_InvalidTransfer.selector);
        vm.deployCode("SafeETHBridge.sol:SafeETHBridge", abi.encode(LOCAL, LOCAL));
    }

    function testFuzz_accounting_concurrentTransfers_succeeds(uint96 _amount, uint8 _decisions) external {
        uint256 amount = bound(_amount, 1, 10 ether);
        _fund(2 * amount);
        bytes32 first = _initiate(amount);
        bytes32 second = _initiate(amount);
        assertNotEq(first, second);
        (bytes32 incoming, Bridge.Transfer memory transfer) = _incoming(amount, 0);
        _prepare(incoming, transfer);
        assertEq(address(bridge).balance, 2 * amount);
        assertEq(bridge.reservedLiquidity(), amount);
        bool commitFirst = _decisions & 1 != 0;
        bool commitSecond = _decisions & 2 != 0;
        bool commitIncoming = _decisions & 4 != 0;
        if (commitFirst) _ack(first);
        if (commitSecond) _ack(second);
        vm.warp(deadline);
        if (!commitFirst) _abort(first);
        if (!commitSecond) _abort(second);
        if (commitIncoming) _commit(incoming);
        else _cancel(incoming, transfer);
        // No operation may change an already terminal decision or pay twice.
        _ack(first);
        _ack(second);
        _commit(incoming);
        uint256 committed = (commitFirst ? amount : 0) + (commitSecond ? amount : 0);
        uint256 payout = commitIncoming ? amount : 0;
        assertEq(alice.balance, 100 ether - committed);
        assertEq(bob.balance, payout);
        assertEq(address(bridge).balance, 0);
        assertEq(bridge.reservedLiquidity(), 0);
        assertEq(Predeploys.ETH_LIQUIDITY.balance, 2 * amount + committed - payout);
        assertEq(
            alice.balance + bob.balance + address(bridge).balance + Predeploys.ETH_LIQUIDITY.balance,
            100 ether + 2 * amount
        );
        assertEq(
            uint256(bridge.source(first).status), uint256(commitFirst ? Bridge.Status.COMMITTED : Bridge.Status.ABORTED)
        );
        assertEq(
            uint256(bridge.source(second).status),
            uint256(commitSecond ? Bridge.Status.COMMITTED : Bridge.Status.ABORTED)
        );
    }
}

/// @notice Ordinary bridging is opt-in independently of safe transfers, with reservation priority.
contract SafeETHBridge_SendETH_Test is SafeETHBridge_TestInit {
    function test_sendETH_preservesSafeEscrow_succeeds() external {
        bytes32 id = _initiate(AMOUNT);
        vm.expectCall(
            MESSENGER,
            abi.encodeCall(
                IL2ToL2CrossDomainMessenger.sendMessage,
                (REMOTE + 2, address(bridge), abi.encodeCall(bridge.relayETH, (alice, bob, AMOUNT)))
            )
        );
        vm.prank(alice);
        bridge.sendETH{ value: AMOUNT }(bob, REMOTE + 2);
        assertEq(address(bridge).balance, AMOUNT);
        assertEq(Predeploys.ETH_LIQUIDITY.balance, AMOUNT);
        vm.warp(deadline);
        _abort(id);
        assertEq(alice.balance, 99 ether);
        assertEq(address(bridge).balance, 0);
        assertEq(Predeploys.ETH_LIQUIDITY.balance, AMOUNT);
    }

    function test_sendETH_zeroRecipient_reverts() external {
        vm.expectRevert(Bridge.ZeroAddress.selector);
        bridge.sendETH(address(0), REMOTE);
    }
}

contract SafeETHBridge_RelayETH_Test is SafeETHBridge_TestInit {
    function _relay(uint256 _amount) internal {
        vm.prank(MESSENGER);
        bridge.relayETH(alice, bob, _amount);
    }

    function test_relayETH_reservedCapacityRetriesAfterSafeCommit_succeeds() external {
        _fund(AMOUNT);
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        _prepare(id, transfer);
        vm.expectRevert(Bridge.SafeETHBridge_InsufficientLiquidity.selector);
        _relay(AMOUNT);
        assertEq(bridge.reservedLiquidity(), AMOUNT);
        assertEq(Predeploys.ETH_LIQUIDITY.balance, AMOUNT);
        assertEq(bob.balance, 0);
        _commit(id);
        assertEq(bob.balance, AMOUNT);
        assertEq(bridge.reservedLiquidity(), 0);
        _fund(AMOUNT);
        _relay(AMOUNT);
        assertEq(bob.balance, 2 * AMOUNT);
        assertEq(Predeploys.ETH_LIQUIDITY.balance, 0);
    }

    function test_relayETH_abortReleasesCapacity_succeeds() external {
        _fund(AMOUNT);
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(AMOUNT, 0);
        _prepare(id, transfer);
        vm.expectRevert(Bridge.SafeETHBridge_InsufficientLiquidity.selector);
        _relay(AMOUNT);
        _cancel(id, transfer);
        _relay(AMOUNT);
        assertEq(bob.balance, AMOUNT);
        assertEq(bridge.reservedLiquidity(), 0);
        assertEq(uint256(bridge.destination(id).status), uint256(Bridge.Status.ABORTED));
    }

    function testFuzz_relayETH_onlyUnreservedCapacity_succeeds(uint96 _safe, uint96 _ordinary) external {
        uint256 safeAmount = bound(_safe, 1, type(uint96).max);
        uint256 ordinaryAmount = bound(_ordinary, 1, type(uint96).max);
        _fund(safeAmount + ordinaryAmount);
        (bytes32 id, Bridge.Transfer memory transfer) = _incoming(safeAmount, 0);
        _prepare(id, transfer);
        // Ordinary bridging keeps the original multi-chain authentication policy.
        _context(address(bridge), REMOTE + 2);
        _relay(ordinaryAmount);
        assertEq(bridge.reservedLiquidity(), safeAmount);
        assertEq(Predeploys.ETH_LIQUIDITY.balance, safeAmount);
        assertEq(bob.balance, ordinaryAmount);
        _context(address(bridge), REMOTE);
        _commit(id);
        assertEq(bob.balance, safeAmount + ordinaryAmount);
        assertEq(Predeploys.ETH_LIQUIDITY.balance, 0);
    }

    function test_relayETH_spoofedCaller_reverts() external {
        vm.expectRevert(Bridge.Unauthorized.selector);
        bridge.relayETH(alice, bob, AMOUNT);
    }

    function test_relayETH_spoofedSender_reverts() external {
        _context(alice, REMOTE);
        vm.expectRevert(Bridge.InvalidCrossDomainSender.selector);
        _relay(AMOUNT);
    }
}
