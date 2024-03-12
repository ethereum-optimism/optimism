// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Bridge_Initializer } from "test/setup/Bridge_Initializer.sol";
import { Reverter, ConfigurableCaller } from "test/mocks/Callers.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { ICrossL2Inbox } from "src/L2/ICrossL2Inbox.sol";

// Libraries
import { Hashing } from "src/libraries/Hashing.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Types } from "src/libraries/Types.sol";

// Target contract dependencies
import { L2ToL1MessagePasser } from "src/L2/L2ToL1MessagePasser.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";

contract L2ToL2CrossDomainMessengerTest is Bridge_Initializer {
    address origin = address(l2ToL2CrossDomainMessenger);
    uint256 destination = block.chainid;
    uint256 source = block.chainid;
    uint256 nonce = 0;
    address sender = address(0x1234);
    address target = address(0xabcd);
    bytes message = hex"1234";

    function testFuzz_sendMessage_succeeds(uint256 _destination, address _target, bytes memory _message) external {
        vm.assume(_destination != block.chainid);
        l2ToL2CrossDomainMessenger.sendMessage(_destination, _target, _message);
    }

    function test_sendMessage_toSelf_fails() external {
        destination = block.chainid;

        vm.expectRevert("L2ToL2CrossDomainMessenger: cannot send message to self");
        l2ToL2CrossDomainMessenger.sendMessage(destination, target, message);
    }

    function testFuzz_relayMessage_succeeds(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        bytes memory _message
    )
        external
    {
        target = address(0);

        ICrossL2Inbox.Identifier memory id = ICrossL2Inbox.Identifier({
            origin: origin,
            blocknumber: 0,
            logIndex: 0,
            timestamp: block.timestamp,
            chainId: block.chainid
        });

        vm.prank(tx.origin);
        crossL2Inbox.executeMessage(id, target, _message);

        vm.expectEmit(origin);
        emit RelayedMessage(keccak256(abi.encode(destination, _source, _nonce, _sender, target, _message)));

        vm.prank(address(crossL2Inbox));
        l2ToL2CrossDomainMessenger.relayMessage(destination, _source, _nonce, _sender, target, _message);

        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSender(), _sender);
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSource(), _source);
    }

    function test_relayMessage_senderNotCrossL2Inbox_fails() external {
        vm.expectRevert("L2ToL2CrossDomainMessenger: sender not CrossL2Inbox");
        l2ToL2CrossDomainMessenger.relayMessage(destination, source, nonce, sender, target, message);
    }

    function test_relayMessage_crossL2InboxOriginNotThisContract_fails() external {
        origin = address(0);
        ICrossL2Inbox.Identifier memory id = ICrossL2Inbox.Identifier({
            origin: origin,
            blocknumber: 0,
            logIndex: 0,
            timestamp: block.timestamp,
            chainId: block.chainid
        });

        vm.prank(tx.origin);
        crossL2Inbox.executeMessage(id, target, message);

        vm.prank(address(crossL2Inbox));

        vm.expectRevert("L2ToL2CrossDomainMessenger: CrossL2Inbox origin not this contract");
        l2ToL2CrossDomainMessenger.relayMessage(destination, source, nonce, sender, target, message);
    }

    function test_relayMessage_destinationNotThisChain_fails() external {
        ICrossL2Inbox.Identifier memory id = ICrossL2Inbox.Identifier({
            origin: origin,
            blocknumber: 0,
            logIndex: 0,
            timestamp: block.timestamp,
            chainId: block.chainid
        });

        vm.prank(tx.origin);
        crossL2Inbox.executeMessage(id, target, message);

        destination = 0;

        vm.prank(address(crossL2Inbox));

        vm.expectRevert("L2ToL2CrossDomainMessenger: destination not this chain");
        l2ToL2CrossDomainMessenger.relayMessage(destination, source, nonce, sender, target, message);
    }

    function test_relayMessage_crossL2InboxCannotCallItself_fails() external {
        ICrossL2Inbox.Identifier memory id = ICrossL2Inbox.Identifier({
            origin: origin,
            blocknumber: 0,
            logIndex: 0,
            timestamp: block.timestamp,
            chainId: block.chainid
        });

        vm.prank(tx.origin);
        crossL2Inbox.executeMessage(id, target, message);

        target = address(crossL2Inbox);

        vm.prank(address(crossL2Inbox));

        vm.expectRevert("L2ToL2CrossDomainMessenger: CrossL2Inbox cannot call itself");
        l2ToL2CrossDomainMessenger.relayMessage(destination, source, nonce, sender, target, message);
    }

    function test_relayMessage_messageAlreadyRelayed_fails() external {
        ICrossL2Inbox.Identifier memory id = ICrossL2Inbox.Identifier({
            origin: origin,
            blocknumber: 0,
            logIndex: 0,
            timestamp: block.timestamp,
            chainId: block.chainid
        });

        vm.prank(tx.origin);
        crossL2Inbox.executeMessage(id, target, message);

        vm.prank(address(crossL2Inbox));
        l2ToL2CrossDomainMessenger.relayMessage(destination, source, nonce, sender, target, message);

        vm.prank(address(crossL2Inbox));
        vm.expectRevert("L2ToL2CrossDomainMessenger: message already relayed");
        l2ToL2CrossDomainMessenger.relayMessage(destination, source, nonce, sender, target, message);
    }

    function test_relayMessage_targetCallFails() external {
        ICrossL2Inbox.Identifier memory id = ICrossL2Inbox.Identifier({
            origin: origin,
            blocknumber: 0,
            logIndex: 0,
            timestamp: block.timestamp,
            chainId: block.chainid
        });

        vm.prank(tx.origin);
        crossL2Inbox.executeMessage(id, target, message);

        vm.etch(target, address(new Reverter()).code);

        vm.expectEmit(origin);
        emit FailedRelayedMessage(keccak256(abi.encode(destination, source, nonce, sender, target, message)));

        vm.prank(address(crossL2Inbox));
        l2ToL2CrossDomainMessenger.relayMessage(destination, source, nonce, sender, target, message);
    }
}
