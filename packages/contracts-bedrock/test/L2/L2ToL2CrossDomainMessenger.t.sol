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
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contract dependencies
import { L2ToL1MessagePasser } from "src/L2/L2ToL1MessagePasser.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { ICrossL2Inbox } from "src/L2/ICrossL2Inbox.sol";

contract L2ToL2CrossDomainMessengerTest is Bridge_Initializer {
    function testFuzz_sendMessage_succeeds(
        uint256 _destination,
        address _target,
        bytes memory _message,
        uint256 _value
    )
        external
    {
        vm.assume(_destination != block.chainid);

        vm.deal(address(this), _value);

        l2ToL2CrossDomainMessenger.sendMessage{ value: _value }({
            _destination: _destination,
            _target: _target,
            _message: _message
        });
    }

    function test_sendMessage_toSelf_fails() external {
        vm.expectRevert("L2ToL2CrossDomainMessenger: cannot send message to self");
        l2ToL2CrossDomainMessenger.sendMessage({
            _destination: block.chainid,
            _target: address(0x1234),
            _message: hex"1234"
        });
    }

    function testFuzz_relayMessage_succeeds(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes memory _message,
        uint256 _value
    )
        external
    {
        vm.assume(_target.code.length == 0);

        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(ICrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        vm.expectEmit(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        emit RelayedMessage(keccak256(abi.encode(block.chainid, _source, _nonce, _sender, _target, _message)));

        vm.deal(Predeploys.CROSS_L2_INBOX, _value);

        vm.prank(Predeploys.CROSS_L2_INBOX);
        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: block.chainid,
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: _target,
            _message: _message
        });

        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSender(), _sender);
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSource(), _source);
    }

    function test_relayMessage_senderNotCrossL2Inbox_fails() external {
        vm.expectRevert("L2ToL2CrossDomainMessenger: sender not CrossL2Inbox");
        l2ToL2CrossDomainMessenger.relayMessage({
            _destination: block.chainid,
            _source: block.chainid,
            _nonce: 0,
            _sender: address(0x1234),
            _target: address(0),
            _message: hex"1234"
        });
    }

    function test_relayMessage_crossL2InboxOriginNotThisContract_fails() external {
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(ICrossL2Inbox.origin.selector),
            returnData: abi.encode(address(0))
        });

        vm.prank(Predeploys.CROSS_L2_INBOX);
        vm.expectRevert("L2ToL2CrossDomainMessenger: CrossL2Inbox origin not this contract");
        l2ToL2CrossDomainMessenger.relayMessage({
            _destination: block.chainid,
            _source: block.chainid,
            _nonce: 0,
            _sender: address(0x1234),
            _target: address(0xabcd),
            _message: hex"1234"
        });
    }

    function test_relayMessage_destinationNotThisChain_fails() external {
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(ICrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        vm.prank(Predeploys.CROSS_L2_INBOX);
        vm.expectRevert("L2ToL2CrossDomainMessenger: destination not this chain");
        l2ToL2CrossDomainMessenger.relayMessage(0, block.chainid, 0, address(0x1234), address(0xabcd), hex"1234");
    }

    function test_relayMessage_crossL2InboxCannotCallItself_fails() external {
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(ICrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        vm.prank(Predeploys.CROSS_L2_INBOX);
        vm.expectRevert("L2ToL2CrossDomainMessenger: CrossL2Inbox cannot call itself");
        l2ToL2CrossDomainMessenger.relayMessage(
            block.chainid, block.chainid, 0, address(0x1234), Predeploys.CROSS_L2_INBOX, hex"1234"
        );
    }

    function test_relayMessage_messageAlreadyRelayed_fails() external {
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(ICrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        // First call should succeed
        vm.prank(Predeploys.CROSS_L2_INBOX);
        l2ToL2CrossDomainMessenger.relayMessage({
            _destination: block.chainid,
            _source: block.chainid,
            _nonce: 0,
            _sender: address(0x1234),
            _target: address(0xabcd),
            _message: hex"1234"
        });

        // Second call should fail
        vm.prank(Predeploys.CROSS_L2_INBOX);
        vm.expectRevert("L2ToL2CrossDomainMessenger: message already relayed");
        l2ToL2CrossDomainMessenger.relayMessage({
            _destination: block.chainid,
            _source: block.chainid,
            _nonce: 0,
            _sender: address(0x1234),
            _target: address(0xabcd),
            _message: hex"1234"
        });
    }

    function test_relayMessage_targetCallFails() external {
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(ICrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        // Target call should fail, so we etch a Reverter() to the target contract
        vm.etch(address(0xabcd), address(new Reverter()).code);

        vm.prank(Predeploys.CROSS_L2_INBOX);
        vm.expectEmit(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        emit FailedRelayedMessage(
            keccak256(abi.encode(block.chainid, block.chainid, 0, address(0x1234), address(0xabcd), hex"1234"))
        );
        l2ToL2CrossDomainMessenger.relayMessage({
            _destination: block.chainid,
            _source: block.chainid,
            _nonce: 0,
            _sender: address(0x1234),
            _target: address(0xabcd),
            _message: hex"1234"
        });
    }
}
