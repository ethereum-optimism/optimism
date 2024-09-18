// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing utilities
import { Test } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contract
import {
    L2ToL2CrossDomainMessenger,
    NotEntered,
    MessageDestinationSameChain,
    RelayMessageCallerNotCrossL2Inbox,
    CrossL2InboxOriginNotL2ToL2CrossDomainMessenger,
    MessageDestinationNotRelayChain,
    MessageTargetCrossL2Inbox,
    MessageTargetL2ToL2CrossDomainMessenger,
    MessageAlreadyRelayed,
    ReentrantCall
} from "src/L2/L2ToL2CrossDomainMessenger.sol";
import { CrossL2Inbox } from "src/L2/CrossL2Inbox.sol";

/// @title L2ToL2CrossDomainMessengerWithModifiableTransientStorage
/// @dev L2ToL2CrossDomainMessenger contract with methods to modify the transient storage.
///      This is used to test the transient storage of L2ToL2CrossDomainMessenger.
contract L2ToL2CrossDomainMessengerWithModifiableTransientStorage is L2ToL2CrossDomainMessenger {
    /// @dev Returns the value of the entered slot in transient storage.
    /// @return Value of the entered slot.
    function entered() external view returns (bool) {
        return _entered();
    }

    /// @dev Sets the entered slot value in transient storage.
    /// @param _value Value to set.
    function setEntered(uint256 _value) external {
        assembly {
            tstore(ENTERED_SLOT, _value)
        }
    }

    /// @dev Sets the cross domain messenger sender in transient storage.
    /// @param _sender Sender address to set.
    function setCrossDomainMessageSender(address _sender) external {
        assembly {
            tstore(CROSS_DOMAIN_MESSAGE_SENDER_SLOT, _sender)
        }
    }

    /// @dev Sets the cross domain messenger source in transient storage.
    /// @param _source Source chain ID to set.
    function setCrossDomainMessageSource(uint256 _source) external {
        assembly {
            tstore(CROSS_DOMAIN_MESSAGE_SOURCE_SLOT, _source)
        }
    }
}

/// @title L2ToL2CrossDomainMessengerTest
/// @dev Contract for testing the L2ToL2CrossDomainMessenger contract.
contract L2ToL2CrossDomainMessengerTest is Test {
    /// @dev L2ToL2CrossDomainMessenger contract instance with modifiable transient storage.
    L2ToL2CrossDomainMessengerWithModifiableTransientStorage l2ToL2CrossDomainMessenger;

    /// @dev Sets up the test suite.
    function setUp() public {
        // Deploy the L2ToL2CrossDomainMessenger contract
        vm.etch(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            address(new L2ToL2CrossDomainMessengerWithModifiableTransientStorage()).code
        );
        l2ToL2CrossDomainMessenger =
            L2ToL2CrossDomainMessengerWithModifiableTransientStorage(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
    }

    /// @dev Tests that `sendMessage` succeeds and emits the correct event.
    function testFuzz_sendMessage_succeeds(uint256 _destination, address _target, bytes calldata _message) external {
        // Ensure the destination is not the same as the source, otherwise the function will revert
        vm.assume(_destination != block.chainid);

        // Ensure that the target contract is not CrossL2Inbox or L2ToL2CrossDomainMessenger
        vm.assume(_target != Predeploys.CROSS_L2_INBOX && _target != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        // Get the current message nonce
        uint256 messageNonce = l2ToL2CrossDomainMessenger.messageNonce();

        // Look for correct emitted event
        vm.recordLogs();

        // Call the sendMessage function
        l2ToL2CrossDomainMessenger.sendMessage({ _destination: _destination, _target: _target, _message: _message });

        // Check that the event was emitted with the correct parameters
        Vm.Log[] memory logs = vm.getRecordedLogs();
        assertEq(logs.length, 1);
        assertEq(
            logs[0].data,
            abi.encodeCall(
                L2ToL2CrossDomainMessenger.relayMessage,
                (_destination, block.chainid, messageNonce, address(this), _target, _message)
            )
        );

        // Check that the message nonce has been incremented
        assertEq(l2ToL2CrossDomainMessenger.messageNonce(), messageNonce + 1);
    }

    /// @dev Tests that the `sendMessage` function reverts when sending a ETH
    function testFuzz_sendMessage_nonPayable_reverts(
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

        // Ensure that _value is greater than 0
        vm.assume(_value > 0);

        // Add sufficient value to the contract to send the message with
        vm.deal(address(this), _value);

        // Call the sendMessage function with value to provoke revert
        (bool success,) = address(l2ToL2CrossDomainMessenger).call{ value: _value }(
            abi.encodeCall(l2ToL2CrossDomainMessenger.sendMessage, (_destination, _target, _message))
        );

        // Check that the function reverts
        assertFalse(success);
    }

    /// @dev Tests that the `sendMessage` function reverts when destination is the same as the source chain.
    function testFuzz_sendMessage_destinationSameChain_reverts(address _target, bytes calldata _message) external {
        // Expect a revert with the MessageDestinationSameChain selector
        vm.expectRevert(MessageDestinationSameChain.selector);

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
        // Ensure that the target contract is not a Forge contract.
        assumeNotForgeAddress(_target);

        // Ensure that the target contract is not CrossL2Inbox or L2ToL2CrossDomainMessenger
        vm.assume(_target != Predeploys.CROSS_L2_INBOX);
        vm.assume(_target != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

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

        // Check that successfulMessages mapping updates the message hash correctly
        assertEq(
            l2ToL2CrossDomainMessenger.successfulMessages(
                keccak256(abi.encode(block.chainid, _source, _nonce, _sender, _target, _message))
            ),
            true
        );

        // Check that entered slot is cleared after the function call
        assertEq(l2ToL2CrossDomainMessenger.entered(), false);

        // Check that metadata is cleared after the function call. We need to set the `entered` slot to non-zero value
        // to prevent NotEntered revert when calling the crossDomainMessageSender and crossDomainMessageSource functions
        l2ToL2CrossDomainMessenger.setEntered(1);
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSource(), 0);
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSender(), address(0));
    }

    /// @dev Mock target function that checks the source and sender of the message in transient storage.
    /// @param _source Source chain ID of the message.
    /// @param _sender Sender of the message.
    function mockTarget(uint256 _source, address _sender) external payable {
        // Ensure that the contract is entered
        assertEq(l2ToL2CrossDomainMessenger.entered(), true);

        // Ensure that the sender is correct
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSource(), _source);

        // Ensure that the source is correct
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSender(), _sender);
    }

    /// @dev Tests that the `relayMessage` function succeeds and stores the correct metadata in transient storage.
    function testFuzz_relayMessage_metadataStore_succeeds(
        uint256 _source,
        uint256 _nonce,
        address _sender,
        uint256 _value
    )
        external
    {
        // Since the target is this contract, we want to ensure the payment doesn't lead to overflow, since this
        // contract has a non-zero balance. Thus, we set this contract's balance to zero and we hoax afterwards.
        vm.deal(address(this), 0);

        // Mock the CrossL2Inbox origin to return the L2ToL2CrossDomainMessenger contract
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(CrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        // Set the target and message for the reentrant call
        address target = address(this);
        bytes memory message = abi.encodeWithSelector(this.mockTarget.selector, _source, _sender);

        // Look for correct emitted event
        vm.expectEmit(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        emit L2ToL2CrossDomainMessenger.RelayedMessage(
            keccak256(abi.encode(block.chainid, _source, _nonce, _sender, target, message))
        );

        // Ensure the target contract is called with the correct parameters
        vm.expectCall({ callee: target, msgValue: _value, data: message });

        // Ensure caller is CrossL2Inbox to prevent a revert from the caller check and that it has sufficient value
        hoax(Predeploys.CROSS_L2_INBOX, _value);

        // Call the relayMessage function
        l2ToL2CrossDomainMessenger.relayMessage{ value: _value }({
            _destination: block.chainid, // ensure the destination is the chain of L2ToL2CrossDomainMessenger
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: target,
            _message: message
        });

        // Check that successfulMessages mapping updates the message hash correctly
        assertEq(
            l2ToL2CrossDomainMessenger.successfulMessages(
                keccak256(abi.encode(block.chainid, _source, _nonce, _sender, target, message))
            ),
            true
        );

        // Check that entered slot is cleared after the function call
        assertEq(l2ToL2CrossDomainMessenger.entered(), false);

        // Check that metadata is cleared after the function call. We need to set the `entered` slot to non-zero value
        // to prevent NotEntered revert when calling the crossDomainMessageSender and crossDomainMessageSource functions
        l2ToL2CrossDomainMessenger.setEntered(1);
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSource(), 0);
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSender(), address(0));
    }

    /// @dev Mock reentrant function that calls the `relayMessage` function.
    /// @param _source Source chain ID of the message.
    /// @param _nonce Nonce of the message.
    /// @param _sender Sender of the message.
    function mockTargetReentrant(uint256 _source, uint256 _nonce, address _sender) external payable {
        // Ensure caller is CrossL2Inbox to prevent a revert from the caller check
        vm.prank(Predeploys.CROSS_L2_INBOX);

        // Ensure that the contract is entered
        assertEq(l2ToL2CrossDomainMessenger.entered(), true);

        vm.expectRevert(ReentrantCall.selector);

        l2ToL2CrossDomainMessenger.relayMessage({
            _destination: block.chainid,
            _source: _source,
            _nonce: _nonce,
            _sender: _sender,
            _target: address(0),
            _message: ""
        });

        // Ensure the function still reverts if `expectRevert` succeeds
        revert();
    }

    /// @dev Tests that the `relayMessage` function reverts when reentrancy is attempted.
    function testFuzz_relayMessage_reentrant_reverts(
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
        // contract has a non-zero balance. Thus, we set this contract's balance to zero and we hoax afterwards.
        vm.deal(address(this), 0);

        // Mock the CrossL2Inbox origin to return the L2ToL2CrossDomainMessenger contract
        vm.mockCall({
            callee: Predeploys.CROSS_L2_INBOX,
            data: abi.encodeWithSelector(CrossL2Inbox.origin.selector),
            returnData: abi.encode(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER)
        });

        // Set the target and message for the reentrant call
        address target = address(this);
        bytes memory message = abi.encodeWithSelector(this.mockTargetReentrant.selector, _source2, _nonce, _sender2);

        // Look for correct emitted event
        vm.expectEmit(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        emit L2ToL2CrossDomainMessenger.FailedRelayedMessage(
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

        // Check that entered slot is cleared after the function call
        assertEq(l2ToL2CrossDomainMessenger.entered(), false);

        // Check that metadata is cleared after the function call. We need to set the `entered` slot to non-zero value
        // to prevent NotEntered revert when calling the crossDomainMessageSender and crossDomainMessageSource functions
        l2ToL2CrossDomainMessenger.setEntered(1);
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSource(), 0);
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSender(), address(0));
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

        // Expect a revert with the RelayMessageCallerNotCrossL2Inbox selector
        vm.expectRevert(RelayMessageCallerNotCrossL2Inbox.selector);

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
        vm.expectRevert(CrossL2InboxOriginNotL2ToL2CrossDomainMessenger.selector);

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
        vm.expectRevert(MessageDestinationNotRelayChain.selector);

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
        vm.expectRevert(MessageAlreadyRelayed.selector);

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
        // Set `entered` to non-zero value to prevent NotEntered revert
        l2ToL2CrossDomainMessenger.setEntered(1);
        // Ensure that the contract is now entered
        assertEq(l2ToL2CrossDomainMessenger.entered(), true);
        // Set cross domain message sender in the transient storage
        l2ToL2CrossDomainMessenger.setCrossDomainMessageSender(_sender);
        // Check that the `crossDomainMessageSender` function returns the correct value
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSender(), _sender);
    }

    /// @dev Tests that the `crossDomainMessageSender` function reverts when not entered.
    function test_crossDomainMessageSender_notEntered_reverts() external {
        // Ensure that the contract is not entered
        assertEq(l2ToL2CrossDomainMessenger.entered(), false);

        // Expect a revert with the NotEntered selector
        vm.expectRevert(NotEntered.selector);

        // Call `crossDomainMessageSender` to provoke revert
        l2ToL2CrossDomainMessenger.crossDomainMessageSender();
    }

    /// @dev Tests that the `crossDomainMessageSource` function returns the correct value.
    function testFuzz_crossDomainMessageSource_succeeds(uint256 _source) external {
        // Set `entered` to non-zero value to prevent NotEntered revert
        l2ToL2CrossDomainMessenger.setEntered(1);
        // Ensure that the contract is now entered
        assertEq(l2ToL2CrossDomainMessenger.entered(), true);
        // Set cross domain message source in the transient storage
        l2ToL2CrossDomainMessenger.setCrossDomainMessageSource(_source);
        // Check that the `crossDomainMessageSource` function returns the correct value
        assertEq(l2ToL2CrossDomainMessenger.crossDomainMessageSource(), _source);
    }

    /// @dev Tests that the `crossDomainMessageSource` function reverts when not entered.
    function test_crossDomainMessageSource_notEntered_reverts() external {
        // Ensure that the contract is not entered
        assertEq(l2ToL2CrossDomainMessenger.entered(), false);

        // Expect a revert with the NotEntered selector
        vm.expectRevert(NotEntered.selector);

        // Call `crossDomainMessageSource` to provoke revert
        l2ToL2CrossDomainMessenger.crossDomainMessageSource();
    }
}
