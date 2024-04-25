// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contracts
import {
    CrossL2Inbox,
    NotEntered,
    InvalidIdTimestamp,
    ChainNotInDependencySet,
    TargetCallFailed
} from "src/L2/CrossL2Inbox.sol";
import { ICrossL2Inbox } from "src/L2/ICrossL2Inbox.sol";

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
        vm.etch(Predeploys.CROSS_L2_INBOX, address(new CrossL2Inbox()).code);
        crossL2Inbox = CrossL2Inbox(Predeploys.CROSS_L2_INBOX);
    }

    /// @dev Tests that the `executeMessage` function  succeeds.
    function testFuzz_executeMessage_succeeds(
        ICrossL2Inbox.Identifier calldata _id,
        address _target,
        bytes calldata _message,
        uint256 _value
    )
        external
        payable
    {
        // Ensure that the id's timestamp is valid (less than or equal to the current block timestamp)
        vm.assume(_id.timestamp <= block.timestamp);

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

        // Check that the Identifier was stored correctly
        assertEq(crossL2Inbox.origin(), _id.origin);
        assertEq(crossL2Inbox.blocknumber(), _id.blocknumber);
        assertEq(crossL2Inbox.logIndex(), _id.logIndex);
        assertEq(crossL2Inbox.timestamp(), _id.timestamp);
        assertEq(crossL2Inbox.chainId(), _id.chainId);
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
        ICrossL2Inbox.Identifier calldata _id,
        address _target,
        bytes calldata _message,
        uint256 _value
    )
        external
    {
        // Ensure that the id's timestamp is valid (less than or equal to the current block timestamp)
        vm.assume(_id.timestamp <= block.timestamp);

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
        ICrossL2Inbox.Identifier calldata _id,
        address _target,
        bytes calldata _message,
        uint256 _value
    )
        external
    {
        // Ensure that the id's timestamp is valid (less than or equal to the current block timestamp)
        vm.assume(_id.timestamp <= block.timestamp);

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

    /// @dev Tests that `blocknumber` reverts when not entered.
    function test_blocknumber_notEntered_reverts() external {
        vm.expectRevert(NotEntered.selector);
        crossL2Inbox.blocknumber();
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
