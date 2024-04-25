// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { TransientContext } from "src/libraries/TransientContext.sol";

// Target contracts
import {
    CrossL2Inbox,
    NotEntered,
    InvalidIdTimestamp,
    ChainNotInDependencySet,
    TargetCallFailed
} from "src/L2/CrossL2Inbox.sol";
import { ICrossL2Inbox } from "src/L2/ICrossL2Inbox.sol";

/// @title CrossL2InboxWithIncrement
/// @dev CrossL2Inbox contract with a method that allows incrementing the transient call depth.
///      This is used to test the transient storage of the CrossL2Inbox contract.
contract CrossL2InboxWithIncrement is CrossL2Inbox {
    /// @dev Increments the call depth.
    function increment() external {
        TransientContext.increment();
    }
}

/// @title CrossL2InboxTest
/// @dev Contract for testing the CrossL2Inbox contract.
contract CrossL2InboxTest is Test {
    /// @dev Selector for the `isInDependencySet` method of the L1Block contract.
    bytes4 constant L1BlockIsInDependencySetSelector = bytes4(keccak256("isInDependencySet(uint256)"));

    /// @dev CrossL2Inbox contract instance.
    CrossL2Inbox crossL2Inbox;

    /// @dev Sets up the test suite.
    function setUp() public {
        // Deploy the L2ToL2CrossDomainMessenger contract
        vm.etch(Predeploys.CROSS_L2_INBOX, address(new CrossL2InboxWithIncrement()).code);
        crossL2Inbox = CrossL2Inbox(Predeploys.CROSS_L2_INBOX);
    }

    /// @dev Tests that the `executeMessage` function  succeeds.
    function testFuzz_executeMessage_succeeds(
        ICrossL2Inbox.Identifier memory _id,
        address _target,
        bytes calldata _message,
        uint256 _value
    )
        external
        payable
    {
        // Ensure that the id's timestamp is valid (less than or equal to the current block timestamp)
        _id.timestamp = bound(_id.timestamp, 0, block.timestamp);

        // Ensure that the target call does not revert
        vm.mockCall({ callee: _target, msgValue: _value, data: _message, returnData: "" });

        // Ensure that the chain ID is in the dependency set
        vm.mockCall({
            callee: Predeploys.L1_BLOCK_ATTRIBUTES,
            data: abi.encodeWithSelector(L1BlockIsInDependencySetSelector, _id.chainId),
            returnData: abi.encode(true)
        });

        // Ensure that the contract has enough balance to send with value
        vm.deal(address(this), _value);

        // Look for the call to the target contract
        vm.expectCall(_target, _value, _message);

        // Call the executeMessage function
        crossL2Inbox.executeMessage{ value: _value }({ _id: _id, _target: _target, _message: _message });

        // Check that the Identifier was stored correctly, but first we have to increment the call depth to the one
        // where the Identifier is stored in transient storage
        CrossL2InboxWithIncrement(Predeploys.CROSS_L2_INBOX).increment();
        assertEq(crossL2Inbox.origin(), _id.origin);
        assertEq(crossL2Inbox.blockNumber(), _id.blockNumber);
        assertEq(crossL2Inbox.logIndex(), _id.logIndex);
        assertEq(crossL2Inbox.timestamp(), _id.timestamp);
        assertEq(crossL2Inbox.chainId(), _id.chainId);
    }

    /// @dev Mock reentrant function that calls the `executeMessage` function.
    /// @param _id Identifier to pass to the `executeMessage` function.
    function mockReentrant(ICrossL2Inbox.Identifier calldata _id) external payable {
        crossL2Inbox.executeMessage({ _id: _id, _target: address(0), _message: "" });
    }

    /// @dev Tests that the `executeMessage` function successfully handles reentrant calls.
    function testFuzz_executeMessage_reentrant_succeeds(
        ICrossL2Inbox.Identifier memory _id1, // identifier passed to `executeMessage` by the initial call.
        ICrossL2Inbox.Identifier memory _id2, // identifier passed to `executeMessage` by the reentrant call.
        uint256 _value
    )
        external
        payable
    {
        // Ensure that the ids' timestamp are valid (less than or equal to the current block timestamp)
        _id1.timestamp = bound(_id1.timestamp, 0, block.timestamp);
        _id2.timestamp = bound(_id2.timestamp, 0, block.timestamp);

        // Ensure that id1's chain ID is in the dependency set
        vm.mockCall({
            callee: Predeploys.L1_BLOCK_ATTRIBUTES,
            data: abi.encodeWithSelector(L1BlockIsInDependencySetSelector, _id1.chainId),
            returnData: abi.encode(true)
        });

        // Ensure that id2's chain ID is in the dependency set
        vm.mockCall({
            callee: Predeploys.L1_BLOCK_ATTRIBUTES,
            data: abi.encodeWithSelector(L1BlockIsInDependencySetSelector, _id2.chainId),
            returnData: abi.encode(true)
        });

        // Set the target and message for the reentrant call
        address target = address(this);
        bytes memory message = abi.encodeWithSelector(this.mockReentrant.selector, _id2);

        // Ensure that the contract has enough balance to send with value
        vm.deal(address(this), _value);

        // Look for the call to the target contract
        vm.expectCall(target, _value, message);

        // Call the executeMessage function
        crossL2Inbox.executeMessage{ value: _value }({ _id: _id1, _target: target, _message: message });

        // Check that the reentrant function didn't update Identifier in transient storage at first call's call depth
        CrossL2InboxWithIncrement(Predeploys.CROSS_L2_INBOX).increment();
        assertEq(crossL2Inbox.origin(), _id1.origin);
        assertEq(crossL2Inbox.blockNumber(), _id1.blockNumber);
        assertEq(crossL2Inbox.logIndex(), _id1.logIndex);
        assertEq(crossL2Inbox.timestamp(), _id1.timestamp);
        assertEq(crossL2Inbox.chainId(), _id1.chainId);

        // Check that the reentrant function updated the Identifier at deeper call depth
        CrossL2InboxWithIncrement(Predeploys.CROSS_L2_INBOX).increment();
        assertEq(crossL2Inbox.origin(), _id2.origin);
        assertEq(crossL2Inbox.blockNumber(), _id2.blockNumber);
        assertEq(crossL2Inbox.logIndex(), _id2.logIndex);
        assertEq(crossL2Inbox.timestamp(), _id2.timestamp);
        assertEq(crossL2Inbox.chainId(), _id2.chainId);
    }

    /// @dev Tests that the `executeMessage` function  reverts when called with an identifier with an invalid timestamp.
    function testFuzz_executeMessage_invalidIdTimestamp_reverts(
        ICrossL2Inbox.Identifier calldata _id,
        address _target,
        bytes calldata _message,
        uint256 _value
    )
        external
    {
        // Ensure that the id's timestamp is invalid (greater than the current block timestamp)
        vm.assume(_id.timestamp > block.timestamp);

        // Ensure that the contract has enough balance to send with value
        vm.deal(address(this), _value);

        // Expect a revert with the InvalidIdTimestamp selector
        vm.expectRevert(abi.encodeWithSelector(InvalidIdTimestamp.selector, _id.timestamp, block.timestamp));

        // Call the executeMessage function
        crossL2Inbox.executeMessage{ value: _value }({ _id: _id, _target: _target, _message: _message });
    }

    /// @dev Tests that the `executeMessage` function  reverts when called with an identifier with a chain ID not in
    /// dependency set.
    function testFuzz_executeMessage_chainNotInDependencySet_reverts(
        ICrossL2Inbox.Identifier memory _id,
        address _target,
        bytes calldata _message,
        uint256 _value
    )
        external
    {
        // Ensure that the id's timestamp is valid (less than or equal to the current block timestamp)
        _id.timestamp = bound(_id.timestamp, 0, block.timestamp);

        // Ensure that the chain ID is NOT in the dependency set
        vm.mockCall({
            callee: Predeploys.L1_BLOCK_ATTRIBUTES,
            data: abi.encodeWithSelector(L1BlockIsInDependencySetSelector, _id.chainId),
            returnData: abi.encode(false)
        });

        // Ensure that the contract has enough balance to send with value
        vm.deal(address(this), _value);

        // Expect a revert with the ChainNotInDependencySet selector
        vm.expectRevert(abi.encodeWithSelector(ChainNotInDependencySet.selector, _id.chainId));

        // Call the executeMessage function
        crossL2Inbox.executeMessage{ value: _value }({ _id: _id, _target: _target, _message: _message });
    }

    /// @dev Tests that the `executeMessage` function  reverts when the target call fails.
    function testFuzz_executeMessage_targetCallFailed_reverts(
        ICrossL2Inbox.Identifier memory _id,
        address _target,
        bytes calldata _message,
        uint256 _value
    )
        external
    {
        // Ensure that the id's timestamp is valid (less than or equal to the current block timestamp)
        _id.timestamp = bound(_id.timestamp, 0, block.timestamp);

        // Ensure that the target call reverts
        vm.mockCallRevert({ callee: _target, msgValue: _value, data: _message, revertData: "" });

        // Ensure that the chain ID is in the dependency set
        vm.mockCall({
            callee: Predeploys.L1_BLOCK_ATTRIBUTES,
            data: abi.encodeWithSelector(L1BlockIsInDependencySetSelector, _id.chainId),
            returnData: abi.encode(true)
        });

        // Ensure that the contract has enough balance to send with value
        vm.deal(address(this), _value);

        // Look for the call to the target contract
        vm.expectCall(_target, _value, _message);

        // Expect a revert with the TargetCallFailed selector
        vm.expectRevert(abi.encodeWithSelector(TargetCallFailed.selector, _target, _message));

        // Call the executeMessage function
        crossL2Inbox.executeMessage{ value: _value }({ _id: _id, _target: _target, _message: _message });
    }

    /// @dev Tests that `origin` reverts when not entered.
    function test_origin_notEntered_reverts() external {
        vm.expectRevert(NotEntered.selector);
        crossL2Inbox.origin();
    }

    /// @dev Tests that `blockNumber` reverts when not entered.
    function test_blockNumber_notEntered_reverts() external {
        vm.expectRevert(NotEntered.selector);
        crossL2Inbox.blockNumber();
    }

    /// @dev Tests that `logIndex` reverts when not entered.
    function test_logIndex_notEntered_reverts() external {
        vm.expectRevert(NotEntered.selector);
        crossL2Inbox.logIndex();
    }

    /// @dev Tests that `timestamp` reverts when not entered.
    function test_timestamp_notEntered_reverts() external {
        vm.expectRevert(NotEntered.selector);
        crossL2Inbox.timestamp();
    }

    /// @dev Tests that `chainId` reverts when not entered.
    function test_chainId_notEntered_reverts() external {
        vm.expectRevert(NotEntered.selector);
        crossL2Inbox.chainId();
    }
}
