// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { TransientContext } from "src/libraries/TransientContext.sol";

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

/// @title L2ToL2CrossDomainMessengerWithModifiableTransientStorage
/// @dev L2ToL2CrossDomainMessenger contract with methods to modify the transient storage.
///      This is used to test the transient storage of L2ToL2CrossDomainMessenger.
contract L2ToL2CrossDomainMessengerWithModifiableTransientStorage is L2ToL2CrossDomainMessenger {
    /// @dev Increments the call depth.
    function increment() external {
        TransientContext.increment();
    }

    /// @dev Sets the cross domain messenger sender in transient storage.
    /// @param _sender Sender address to set.
    function setCrossDomainMessageSender(address _sender) external {
        TransientContext.set(CROSS_DOMAIN_MESSAGE_SENDER_SLOT, uint160(_sender));
    }

    /// @dev Sets the cross domain messenger source in transient storage.
    /// @param _source Source chain ID to set.
    function setCrossDomainMessageSource(uint256 _source) external {
        TransientContext.set(CROSS_DOMAIN_MESSAGE_SOURCE_SLOT, _source);
    }
}

/// @title L2ToL2CrossDomainMessengerTest
/// @dev Contract for testing the L2ToL2CrossDomainMessenger contract.
contract L2ToL2CrossDomainMessengerTest is Test {
    /// @dev L2ToL2CrossDomainMessenger contract instance.
    L2ToL2CrossDomainMessenger l2ToL2CrossDomainMessenger;

    /// @dev Sets up the test suite.
    function setUp() public {
        // Deploy the L2ToL2CrossDomainMessenger contract
        vm.etch(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            address(new L2ToL2CrossDomainMessengerWithModifiableTransientStorage()).code
        );
        l2ToL2CrossDomainMessenger = L2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
    }

    /// @dev Tests that `sendMessage` succeeds and emits the correct SentMessage event.
    function testFuzz_sendMessage_succeeds(
        uint256 _destination,
        address _target,
        bytes calldata _message,
        uint256 _value
    )
        external
    {
        // Ensure the destination is not the same as the source, otherwise the function will revert
        vm.assume(_destination != block.chainid);

        // Ensure that the target contract is not CrossL2Inbox or L2ToL2CrossDomainMessenger
        vm.assume(_target != Predeploys.CROSS_L2_INBOX && _target != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

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

        // Check that the message nonce has been incremented
        assertEq(l2ToL2CrossDomainMessenger.messageNonce(), messageNonce + 1);
    }

    /// @dev Tests that the `sendMessage` function reverts when destination is the same as the source chain.
    function testFuzz_sendMessage_destinationSameChain_reverts(address _target, bytes calldata _message) external {
        // Expect a revert with the MessageDestinationSameChain selector
        vm.expectRevert(abi.encodeWithSelector(MessageDestinationSameChain.selector, block.chainid));

        // Call `sendMessage` with the current chain as the destination to prevent revert due to invalid destination
        l2ToL2CrossDomainMessenger.sendMessage({ _destination: block.chainid, _target: _target, _message: _message });
    }

    /// @dev Tests that the `sendMessage` function reverts when the target is CrossL2Inbox.
    function testFuzz_sendMessage_targetCrossL2Inbox_reverts(uint256 _destination, bytes calldata _message) external {
        // Ensure the destination is not the same as the source, otherwise the function will revert regardless of target
        vm.assume(_destination != block.chainid);

        // Expect a revert with the MessageTargetCrossL2Inbox selector
        vm.expectRevert(MessageTargetCrossL2Inbox.selector);

        // Call `senderMessage` with the CrossL2Inbox as the target to provoke revert
        l2ToL2CrossDomainMessenger.sendMessage({
            _destination: _destination,
            _target: Predeploys.CROSS_L2_INBOX,
            _message: _message
        });
    }

    /// @dev Tests that the `sendMessage` function reverts when the target is L2ToL2CrossDomainMessenger.
    function testFuzz_sendMessage_targetL2ToL2CrossDomainMessenger_reverts(
        uint256 _destination,
        bytes calldata _message
    )
        external
    {
        // Ensure the destination is not the same as the source, otherwise the function will revert regardless of target
        vm.assume(_destination != block.chainid);

        // Expect a revert with the MessageTargetL2ToL2CrossDomainMessenger selector
        vm.expectRevert(MessageTargetL2ToL2CrossDomainMessenger.selector);

        // Call `senderMessage` with the L2ToL2CrossDomainMessenger as the target to provoke revert
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
        bytes calldata _message,
        uint256 _value
    )
        external
    {
        // Ensure that the target contract is not CrossL2Inbox or L2ToL2CrossDomainMessenger
        vm.assume(_target != Predeploys.CROSS_L2_INBOX && _target != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        // Ensure that the target call is payable if value is sent
        if (_value > 0) assumePayable(_target);

        // Ensure that the target contract does not revert
        vm.mockCall({ callee: _target, msgValue: _value, data: _message, returnData: abi.encode(true) });

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

        // Check that the crossDomainMessageSender and crossDomainMessageSource update correctly, but first we have to
        // increment the call depth to the one where the data is stored in transient storage
        L2ToL2CrossDomainMessengerWithModifiableTransientStorage(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).increment();
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSender(), _sender);
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSource(), _source);
    }

    /// @dev Mock reentrant function that calls the `relayMessage` function.
    /// @param _source Source chain ID of the message.
    /// @param _sender Sender of the message.
    function mockReentrant(uint256 _source, uint256 _nonce, address _sender) external payable {
        // Ensure caller is CrossL2Inbox to prevent a revert from the caller check
        vm.prank(Predeploys.CROSS_L2_INBOX);

        l2ToL2CrossDomainMessenger.relayMessage({
            _destination: block.chainid,
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: address(0),
            _message: ""
        });
    }

    /// @dev Tests that the `relayMessage` function  successfully handles reentrant calls.
    function testFuzz_relayMessage_reentrant_succeeds(
        uint256 _source1, // source passed to `relayMessage` by the initial call.
        address _sender1, // sender passed to `relayMessage` by the initial call.
        uint256 _source2, // sender passed to `relayMessage` by the reentrant call.
        address _sender2, // sender passed to `relayMessage` by the reentrant call.
        uint256 _nonce,
        uint256 _value
    )
        external
    {
        // Since the target is this contract, we want to ensure the payment doesn't lead to overflow, since this
        // contract has a non-zero balance. Thus, we set this contract's balance to zero.
        vm.deal(address(this), 0);

        // Mock the CrossL2Inbox origin to return the L2ToL2CrossDomainMessenger contract
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(CrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        // Set the target and message for the reentrant call
        address target = address(this);
        bytes memory message = abi.encodeWithSelector(this.mockReentrant.selector, _source2, _nonce, _sender2);

        // Look for correct emitted event
        vm.expectEmit(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        emit L2ToL2CrossDomainMessenger.RelayedMessage(
            keccak256(abi.encode(block.chainid, _source1, _nonce, _sender1, target, message))
        );

        // Ensure the target contract is called with the correct parameters
        vm.expectCall({ callee: target, msgValue: _value, data: message });

        // Ensure caller is CrossL2Inbox to prevent a revert from the caller check and that it has sufficient value
        hoax(Predeploys.CROSS_L2_INBOX, _value);

        // Call the relayMessage function
        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: block.chainid, // ensure the destination is the chain of L2ToL2CrossDomainMessenger
            _source: _source1,
            _nonce: _nonce,
            _sender: _sender1,
            _target: target,
            _message: message
        });

        // Check that the balance of the target contract is correct (i.e., the value was transferred successfully)
        assertEq(target.balance, _value);

        // Check that the crossDomainMessageSender and crossDomainMessageSource update correctly, but first we have to
        // increment the call depth to the one where the data is stored in transient storage
        L2ToL2CrossDomainMessengerWithModifiableTransientStorage(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).increment();
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSender(), _sender1);
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSource(), _source1);

        // Check that the reentrant function correctly updated the slots at deeper call depth
        L2ToL2CrossDomainMessengerWithModifiableTransientStorage(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).increment();
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSender(), _sender2);
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSource(), _source2);
    }

    /// @dev Tests that the `relayMessage` function reverts when the caller is not the CrossL2Inbox contract.
    function testFuzz_relayMessage_callerNotCrossL2Inbox_reverts(
        uint256 _destination,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message,
        uint256 _value
    )
        external
    {
        // Add sufficient value to the contract to relay the message with
        vm.deal(address(this), _value);

        // Expect a revert with the RelayCallerNotCrossL2Inbox selector
        vm.expectRevert(abi.encodeWithSelector(RelayCallerNotCrossL2Inbox.selector, address(this)));

        // Call `relayMessage` with the current contract as the caller to provoke revert
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
        bytes calldata _message,
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

        // Expect a revert with the CrossL2InboxOriginNotL2ToL2CrossDomainMessenger selector
        vm.expectRevert(abi.encodeWithSelector(CrossL2InboxOriginNotL2ToL2CrossDomainMessenger.selector, address(0)));

        // Call `relayMessage` with invalid CrossL2Inbox origin to provoke revert
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
        uint256 _destination,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message,
        uint256 _value
    )
        external
    {
        // Ensure the destination is not this chain
        vm.assume(_destination != block.chainid);

        // Mock the CrossL2Inbox origin to return the L2ToL2CrossDomainMessenger contract
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(CrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        // Ensure caller is CrossL2Inbox to prevent a revert from the caller check and that it has sufficient value
        hoax(Predeploys.CROSS_L2_INBOX, _value);

        // Expect a revert with the MessageDestinationNotRelayChain selector
        vm.expectRevert(abi.encodeWithSelector(MessageDestinationNotRelayChain.selector, _destination, block.chainid));

        // Call `relayMessage`
        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: _destination,
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
        bytes calldata _message,
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

        // Expect a revert with the MessageTargetCrossL2Inbox selector
        vm.expectRevert(MessageTargetCrossL2Inbox.selector);

        // Call `relayMessage` with CrossL2Inbox as the target to provoke revert. The current chain is the destination
        // to prevent revert due to invalid destination
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
        bytes calldata _message,
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

        // Expect a revert with the MessageTargetL2ToL2CrossDomainMessenger selector
        vm.expectRevert(MessageTargetL2ToL2CrossDomainMessenger.selector);

        // Call `relayMessage` with L2ToL2CrossDomainMessenger as the target to provoke revert. The current chain is the
        // destination to prevent revert due to invalid destination
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
        bytes calldata _message,
        uint256 _value
    )
        external
    {
        // Ensure that payment doesn't overflow since we send value to L2ToL2CrossDomainMessenger twice
        _value = bound(_value, 0, type(uint256).max / 2);

        // Ensure that the target call is payable if value is sent
        if (_value > 0) assumePayable(_target);

        // Ensure that the target contract is not CrossL2Inbox or L2ToL2CrossDomainMessenger
        vm.assume(_target != Predeploys.CROSS_L2_INBOX && _target != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        // Ensure that the target contract does not revert
        vm.mockCall({ callee: _target, msgValue: _value, data: _message, returnData: abi.encode(true) });

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

        // First call to `relayMessage` should succeed. The current chain is the destination to prevent revert due to
        // invalid destination
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

        // Second call should fail with MessageAlreadyRelayed selector
        vm.expectRevert(
            abi.encodeWithSelector(
                MessageAlreadyRelayed.selector,
                keccak256(abi.encode(block.chainid, _source, _nonce, _sender, _target, _message))
            )
        );

        // Call `relayMessage` again. The current chain is the destination to prevent revert due to invalid destination
        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: block.chainid,
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: _target,
            _message: _message
        });
    }

    /// @dev Tests that the `relayMessage` function reverts when the target call fails.
    function testFuzz_relayMessage_targetCallFails_reverts(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message,
        uint256 _value
    )
        external
    {
        // Ensure that the target contract is not CrossL2Inbox or L2ToL2CrossDomainMessenger
        vm.assume(_target != Predeploys.CROSS_L2_INBOX && _target != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        // Ensure that the target call is payable if value is sent
        if (_value > 0) assumePayable(_target);

        // Ensure that the target contract reverts
        vm.mockCallRevert({ callee: _target, msgValue: _value, data: _message, revertData: abi.encode(false) });

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

    /// @dev Tests that the `crossDomainMessageSender` function returns the correct value.
    function testFuzz_crossDomainMessageSender_succeeds(address _sender) external {
        // Increment the call depth to prevent NotEntered revert
        L2ToL2CrossDomainMessengerWithModifiableTransientStorage(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).increment();
        // Set cross domain message sender in the transient storage
        L2ToL2CrossDomainMessengerWithModifiableTransientStorage(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
            .setCrossDomainMessageSender(_sender);
        // Check that the `crossDomainMessageSender` function returns the correct value
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSender(), _sender);
    }

    /// @dev Tests that the `crossDomainMessageSender` function reverts when not entered.
    function test_crossDomainMessageSender_notEntered_reverts() external {
        // Expect a revert with the NotEntered selector
        vm.expectRevert(NotEntered.selector);

        // Call `crossDomainMessageSender` to provoke revert
        l2ToL2CrossDomainMessenger.crossDomainMessageSender();
    }

    /// @dev Tests that the `crossDomainMessageSource` function returns the correct value.
    function testFuzz_crossDomainMessageSource_succeeds(uint256 _source) external {
        // Increment the call depth to prevent NotEntered revert
        L2ToL2CrossDomainMessengerWithModifiableTransientStorage(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).increment();
        // Set cross domain message source in the transient storage
        L2ToL2CrossDomainMessengerWithModifiableTransientStorage(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
            .setCrossDomainMessageSource(_source);
        // Check that the `crossDomainMessageSource` function returns the correct value
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSource(), _source);
    }

    /// @dev Tests that the `crossDomainMessageSource` function reverts when not entered.
    function test_crossDomainMessageSource_notEntered_reverts() external {
        // Expect a revert with the NotEntered selector
        vm.expectRevert(NotEntered.selector);

        // Call `crossDomainMessageSource` to provoke revert
        l2ToL2CrossDomainMessenger.crossDomainMessageSource();
    }
}
