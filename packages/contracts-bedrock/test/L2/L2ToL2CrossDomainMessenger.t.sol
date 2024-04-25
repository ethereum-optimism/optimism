// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contract
import {
    L2ToL2CrossDomainMessenger,
    NotEntered,
    MessageDestinationSameChain,
    RelayCallerNotCrossL2Inbox,
    CrossL2InboxOriginNotL2ToL2CrossDomainMessenger,
    MessageDestinationNotRelayChain,
    MessageTargetCrossL2Inbox,
    MessageTargetL2ToL2CrossDomainMessenger,
    MessageAlreadyRelayed
} from "src/L2/L2ToL2CrossDomainMessenger.sol";
import { CrossL2Inbox } from "src/L2/CrossL2Inbox.sol";

/// @title L2ToL2CrossDomainMessengerTest
/// @dev The L2ToL2CrossDomainMessengerTest contract tests the L2ToL2CrossDomainMessenger contract.
contract L2ToL2CrossDomainMessengerTest is Test {
    /// @dev L2ToL2CrossDomainMessenger contract instance.
    L2ToL2CrossDomainMessenger l2ToL2CrossDomainMessenger;

    /// @dev Sets up the test suite.
    function setUp() public {
        // Deploy the L2ToL2CrossDomainMessenger contract
        vm.etch(Predeploys.CROSS_L2_INBOX, address(new CrossL2Inbox()).code);
        // Deploy the L2ToL2CrossDomainMessenger contract
        vm.etch(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER, address(new L2ToL2CrossDomainMessenger()).code);
        l2ToL2CrossDomainMessenger = L2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
    }

    /// @dev Tests that `sendMessage` succeeds and emits the correct SentMessage event.
    function testFuzz_sendMessage_succeeds(
        uint256 _destination,
        address _target,
        bytes memory _message,
        uint256 _value
    )
        external
    {
        // Ensure the destination is not the same as the source, otherwise the function will revert
        vm.assume(_destination != block.chainid);

        // Get the current message nonce
        uint256 messageNonce = l2ToL2CrossDomainMessenger.messageNonce();

        // Add sufficient value to the contract to send the message with
        vm.deal(address(this), _value);

        // Look for correct emitted event
        vm.expectEmit(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        emit L2ToL2CrossDomainMessenger.SentMessage(
            abi.encodeCall(
                L2ToL2CrossDomainMessenger.relayMessage,
                (_destination, block.chainid, messageNonce, address(this), _target, _message)
            )
        );

        l2ToL2CrossDomainMessenger.sendMessage{ value: _value }({
            _destination: _destination,
            _target: _target,
            _message: _message
        });

        // Ensure the message nonce has been incremented
        assertEq(l2ToL2CrossDomainMessenger.messageNonce(), messageNonce + 1);
    }

    /// @dev Tests that the `sendMessage` function reverts when destination is the same as the source chain.
    function testFuzz_sendMessage_destinationSameChain_reverts(address _target, bytes memory _message) external {
        vm.expectRevert(abi.encodeWithSelector(MessageDestinationSameChain.selector, block.chainid));
        l2ToL2CrossDomainMessenger.sendMessage({ _destination: block.chainid, _target: _target, _message: _message });
    }

    /// @dev Tests that the `sendMessage` function reverts when the target is CrossL2Inbox.
    function testFuzz_sendMessage_targetCrossL2Inbox_reverts(uint256 _destination, bytes memory _message) external {
        // Ensure the destination is not the same as the source, otherwise the function will revert regardless of target
        vm.assume(_destination != block.chainid);

        vm.expectRevert(MessageTargetCrossL2Inbox.selector);
        l2ToL2CrossDomainMessenger.sendMessage({
            _destination: _destination,
            _target: Predeploys.CROSS_L2_INBOX,
            _message: _message
        });
    }

    /// @dev Tests that the `sendMessage` function reverts when the target is L2ToL2CrossDomainMessenger.
    function testFuzz_sendMessage_targetL2ToL2CrossDomainMessenger_reverts(
        uint256 _destination,
        bytes memory _message
    )
        external
    {
        // Ensure the destination is not the same as the source, otherwise the function will revert regardless of target
        vm.assume(_destination != block.chainid);

        vm.expectRevert(MessageTargetL2ToL2CrossDomainMessenger.selector);
        l2ToL2CrossDomainMessenger.sendMessage({
            _destination: _destination,
            _target: Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            _message: _message
        });
    }

    /// @dev Tests that the `relayMessage` function succeeds and emits the correct RelayedMessage event.
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
        // Ensure that the target contract is not CrossL2Inbox or L2ToL2CrossDomainMessenger
        vm.assume(_target != Predeploys.CROSS_L2_INBOX && _target != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        // Ensure that the target contract does not revert
        vm.mockCall({ callee: _target, msgValue: _value, data: _message, returnData: "" });

        // Mock the CrossL2Inbox origin to return the L2ToL2CrossDomainMessenger contract
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(CrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        // Look for correct emitted event
        vm.expectEmit(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        emit L2ToL2CrossDomainMessenger.RelayedMessage(
            keccak256(abi.encode(block.chainid, _source, _nonce, _sender, _target, _message))
        );

        // Ensure the target contract is called with the correct parameters
        vm.expectCall({ callee: _target, msgValue: _value, data: _message });

        // Ensure caller is CrossL2Inbox to prevent a revert from the caller check and that it has sufficient value
        hoax(Predeploys.CROSS_L2_INBOX, _value);

        // Call the relayMessage function
        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: block.chainid, // ensure the destination is the chain of L2ToL2CrossDomainMessenger
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: _target,
            _message: _message
        });

        // Check that the crossDomainMessageSender and crossDomainMessageSource update correctly
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSender(), _sender);
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSource(), _source);
    }

    /// @dev Tests that the `relayMessage` function reverts when the caller is not the CrossL2Inbox contract.
    function testFuzz_relayMessage_callerNotCrossL2Inbox_reverts(
        uint256 _destination,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes memory _message,
        uint256 _value
    )
        external
    {
        // Add sufficient value to the contract to relay the message with
        vm.deal(address(this), _value);

        vm.expectRevert(abi.encodeWithSelector(RelayCallerNotCrossL2Inbox.selector, address(this)));
        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: _destination,
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: _target,
            _message: _message
        });
    }

    /// @dev Tests that the `relayMessage` function reverts when CrossL2Inbox's origin is not
    ///      L2ToL2CrossDomainMessenger.
    function testFuzz_relayMessage_crossL2InboxOriginNotL2ToL2CrossDomainMessenger_reverts(
        uint256 _destination,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes memory _message,
        uint256 _value
    )
        external
    {
        // Set address(0) as the origin of the CrossL2Inbox contract, which is not the L2ToL2CrossDomainMessenger
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(CrossL2Inbox.origin.selector),
            returnData: abi.encode(address(0))
        });

        // Ensure caller is CrossL2Inbox to prevent a revert from the caller check and that it has sufficient value
        hoax(Predeploys.CROSS_L2_INBOX, _value);

        vm.expectRevert(abi.encodeWithSelector(CrossL2InboxOriginNotL2ToL2CrossDomainMessenger.selector, address(0)));
        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: _destination,
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: _target,
            _message: _message
        });
    }

    /// @dev Tests that the `relayMessage` function reverts when the destination is not the relay chain.
    function testFuzz_relayMessage_destinationNotRelayChain_reverts(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes memory _message,
        uint256 _value
    )
        external
    {
        // Mock the CrossL2Inbox origin to return the L2ToL2CrossDomainMessenger contract
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(CrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        // Ensure caller is CrossL2Inbox to prevent a revert from the caller check and that it has sufficient value
        hoax(Predeploys.CROSS_L2_INBOX, _value);

        vm.expectRevert(abi.encodeWithSelector(MessageDestinationNotRelayChain.selector, 0));
        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: 0,
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: _target,
            _message: _message
        });
    }

    /// @dev Tests that the `relayMessage` function reverts when the message target is CrossL2Inbox.
    function testFuzz_relayMessage_targetCrossL2Inbox_reverts(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        bytes memory _message,
        uint256 _value
    )
        external
    {
        // Mock the CrossL2Inbox origin to return the L2ToL2CrossDomainMessenger contract
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(CrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        // Ensure caller is CrossL2Inbox to prevent a revert from the caller check and that it has sufficient value
        hoax(Predeploys.CROSS_L2_INBOX, _value);

        vm.expectRevert(MessageTargetCrossL2Inbox.selector);
        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: block.chainid,
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: Predeploys.CROSS_L2_INBOX,
            _message: _message
        });
    }

    /// @dev Tests that the `relayMessage` function reverts when the message target is L2ToL2CrossDomainMessenger.
    function testFuzz_relayMessage_targetL2ToL2CrossDomainMessenger_reverts(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        bytes memory _message,
        uint256 _value
    )
        external
    {
        // Mock the CrossL2Inbox origin to return the L2ToL2CrossDomainMessenger contract
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(CrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        // Ensure caller is CrossL2Inbox to prevent a revert from the caller check and that it has sufficient value
        hoax(Predeploys.CROSS_L2_INBOX, _value);

        vm.expectRevert(MessageTargetL2ToL2CrossDomainMessenger.selector);
        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: block.chainid,
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            _message: _message
        });
    }

    /// @dev Tests that the `relayMessage` function reverts when the message has already been relayed.
    function testFuzz_relayMessage_alreadyRelayed_reverts(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes memory _message,
        uint256 _value
    )
        external
    {
        // Ensure that payment doesn't overflow since we send value to L2ToL2CrossDomainMessenger twice
        vm.assume(_value < type(uint256).max / 2);

        // Ensure that the target contract is not CrossL2Inbox or L2ToL2CrossDomainMessenger
        vm.assume(_target != Predeploys.CROSS_L2_INBOX && _target != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        // Ensure that the target contract does not revert
        vm.mockCall({ callee: _target, msgValue: _value, data: _message, returnData: "" });

        // Mock the CrossL2Inbox origin to return the L2ToL2CrossDomainMessenger contract
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(CrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        // Ensure caller is CrossL2Inbox to prevent a revert from the caller check and that it has sufficient value
        hoax(Predeploys.CROSS_L2_INBOX, _value);

        // Look for correct emitted event for first call.
        vm.expectEmit(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        emit L2ToL2CrossDomainMessenger.RelayedMessage(
            keccak256(abi.encode(block.chainid, _source, _nonce, _sender, _target, _message))
        );

        // First call should succeed
        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: block.chainid,
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: _target,
            _message: _message
        });

        // Ensure caller is CrossL2Inbox to prevent a revert from the caller check and that it has sufficient value
        hoax(Predeploys.CROSS_L2_INBOX, _value);

        // Second call should fail
        vm.expectRevert(
            abi.encodeWithSelector(
                MessageAlreadyRelayed.selector,
                keccak256(abi.encode(block.chainid, _source, _nonce, _sender, _target, _message))
            )
        );

        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: block.chainid,
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: _target,
            _message: _message
        });
    }

    /// @dev Tests that the `relayMessage` function reverst when the target call fails.
    function testFuzz_relayMessage_targetCallFails_reverts(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes memory _message,
        uint256 _value
    )
        external
    {
        // Ensure that the target contract is not CrossL2Inbox or L2ToL2CrossDomainMessenger
        vm.assume(_target != Predeploys.CROSS_L2_INBOX && _target != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        // Ensure that the target contract reverts
        vm.mockCallRevert({ callee: _target, msgValue: _value, data: _message, revertData: "" });

        // Mock the CrossL2Inbox origin to return the L2ToL2CrossDomainMessenger contract
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(CrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        // Ensure caller is CrossL2Inbox to prevent a revert from the caller check and that it has sufficient value
        hoax(Predeploys.CROSS_L2_INBOX, _value);

        // Look for correct emitted event
        vm.expectEmit(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        emit L2ToL2CrossDomainMessenger.FailedRelayedMessage(
            keccak256(abi.encode(block.chainid, _source, _nonce, _sender, _target, _message))
        );

        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: block.chainid,
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: _target,
            _message: _message
        });
    }

    /// @dev Tests that `crossDomainMessageSender` reverts when not entered.
    function test_crossDomainMessageSender_notEntered_reverts() external {
        vm.expectRevert(NotEntered.selector);
        l2ToL2CrossDomainMessenger.crossDomainMessageSender();
    }

    /// @dev Tests that `crossDomainMessageSource` reverts when not entered.
    function test_crossDomainMessageSource_notEntered_reverts() external {
        vm.expectRevert(NotEntered.selector);
        l2ToL2CrossDomainMessenger.crossDomainMessageSource();
    }
}
