// SPDX-License-ICrossL2Inbox.Identifier: MIT
pragma solidity 0.8.25;

// Testing utilities
import { Test } from "forge-std/Test.sol";
import { Reverter } from "test/mocks/Callers.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contracts
import { L1Block } from "src/L2/L1Block.sol";
import { CrossL2Inbox, NotEntered } from "src/L2/CrossL2Inbox.sol";
import { ICrossL2Inbox } from "src/L2/ICrossL2Inbox.sol";

contract CrossL2InboxTest is Test {
    /// @dev CrossL2Inbox contract instance.
    CrossL2Inbox crossL2Inbox;

    /// @dev Sample ICrossL2Inbox.Identifier.
    ICrossL2Inbox.Identifier sampleIdentifier = ICrossL2Inbox.Identifier({
        origin: address(0),
        blocknumber: 0,
        logIndex: 0,
        timestamp: block.timestamp,
        chainId: block.chainid
    });

    /// @dev Sets up the test suite.
    function setUp() public {
        crossL2Inbox = new CrossL2Inbox();
    }

    /// @dev Tests that `executeMessage` succeeds when called with valid parameters.
    function testFuzz_executeMessage_succeeds(
        bytes calldata _msg,
        ICrossL2Inbox.Identifier calldata _id,
        address _target,
        uint256 _value
    )
        external
        payable
    {
        vm.assume(_id.timestamp <= block.timestamp);
        vm.assume(_target.code.length == 0);

        // need to prevent call to L1Block.isInDependencySet from reverting
        vm.mockCall({
            callee: Predeploys.L1_BLOCK_ATTRIBUTES,
            data: abi.encodeWithSelector(L1Block.isInDependencySet.selector, _id.chainId),
            returnData: abi.encode(true)
        });

        vm.deal(tx.origin, _value);

        // executeMessage
        vm.prank(tx.origin);
        vm.expectCall(_target, _value, _msg);
        crossL2Inbox.executeMessage{ value: _value }({ _id: _id, _target: _target, _msg: _msg });

        assertEq(crossL2Inbox.origin(), _id.origin);
        assertEq(crossL2Inbox.blocknumber(), _id.blocknumber);
        assertEq(crossL2Inbox.logIndex(), _id.logIndex);
        assertEq(crossL2Inbox.timestamp(), _id.timestamp);
        assertEq(crossL2Inbox.chainId(), _id.chainId);
    }

    /// @dev Tests that `executeMessage` fails when called with an identifier with an invalid timestamp.
    function test_executeMessage_invalidTimestamp_fails() external {
        sampleIdentifier.timestamp = block.timestamp + 1;

        vm.prank(tx.origin);
        vm.expectRevert("CrossL2Inbox: invalid id timestamp");
        crossL2Inbox.executeMessage({ _id: sampleIdentifier, _target: address(0), _msg: hex"1234" });
    }

    /// @dev Tests that `executeMessage` fails when called with an identifier with an invalid chain ID.
    function test_executeMessage_invalidChainId_fails() external {
        vm.mockCall({
            callee: Predeploys.L1_BLOCK_ATTRIBUTES,
            data: abi.encodeWithSelector(L1Block.isInDependencySet.selector, sampleIdentifier.chainId),
            returnData: abi.encode(false)
        });

        vm.prank(tx.origin);
        vm.expectRevert("CrossL2Inbox: id chain not in dependency set");
        crossL2Inbox.executeMessage({ _id: sampleIdentifier, _target: address(0), _msg: hex"1234" });
    }

    /// @dev Tests that `executeMessage` succeeds when called with an identifier with the same chain ID as
    ///      the current chain.
    function test_executeMessage_sameChainId_succeeds() external {
        vm.mockCall({
            callee: Predeploys.L1_BLOCK_ATTRIBUTES,
            data: abi.encodeWithSelector(L1Block.isInDependencySet.selector, sampleIdentifier.chainId),
            returnData: abi.encode(true)
        });

        vm.prank(tx.origin);
        vm.expectCall(address(0), hex"1234");
        crossL2Inbox.executeMessage({ _id: sampleIdentifier, _target: address(0), _msg: hex"1234" });
    }

    /// @dev Tests that `executeMessage` fails when called by a non-EOA.
    function test_executeMessage_invalidSender_fails() external {
        vm.expectRevert("CrossL2Inbox: not EOA sender");
        crossL2Inbox.executeMessage({ _id: sampleIdentifier, _target: address(0), _msg: hex"1234" });
    }

    /// @dev Tests that `executeMessage` fails when the underlying target call reverts.
    function test_executeMessage_unsuccessfullSafeCall_fails() external {
        // need to make sure address leads to unsuccessfull SafeCall by executeMessage
        vm.etch(address(0), address(new Reverter()).code);

        vm.mockCall({
            callee: Predeploys.L1_BLOCK_ATTRIBUTES,
            data: abi.encodeWithSelector(L1Block.isInDependencySet.selector, sampleIdentifier.chainId),
            returnData: abi.encode(true)
        });

        vm.prank(tx.origin);
        vm.expectCall(address(0), hex"1234");
        vm.expectRevert("CrossL2Inbox: target call failed");
        crossL2Inbox.executeMessage({ _id: sampleIdentifier, _target: address(0), _msg: hex"1234" });
    }

    /// @dev Tests that `origin` reverts when not entered.
    function test_origin_notEntered_fails() external {
        vm.expectRevert(NotEntered.selector);
        crossL2Inbox.origin();
    }

    /// @dev Tests that `blocknumber` reverts when not entered.
    function test_blocknumber_notEntered_fails() external {
        vm.expectRevert(NotEntered.selector);
        crossL2Inbox.blocknumber();
    }

    /// @dev Tests that `logIndex` reverts when not entered.
    function test_logIndex_notEntered_fails() external {
        vm.expectRevert(NotEntered.selector);
        crossL2Inbox.logIndex();
    }

    /// @dev Tests that `timestamp` reverts when not entered.
    function test_timestamp_notEntered_fails() external {
        vm.expectRevert(NotEntered.selector);
        crossL2Inbox.timestamp();
    }

    /// @dev Tests that `chainId` reverts when not entered.
    function test_chainId_notEntered_fails() external {
        vm.expectRevert(NotEntered.selector);
        crossL2Inbox.chainId();
    }
}
