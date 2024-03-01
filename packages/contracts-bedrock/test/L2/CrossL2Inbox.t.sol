// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { Reverter, ConfigurableCaller } from "test/mocks/Callers.sol";

// Libraries
import { ICrossL2Inbox } from "src/libraries/Predeploys.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";

import { L1Block } from "src/L2/L1Block.sol";

contract CrossL2InboxTest is CommonTest {
    address depositor;

    function setUp() public virtual override {
        super.setUp();
        depositor = l1Block.DEPOSITOR_ACCOUNT();
    }

    /// @dev Tests that the implementation is constructed correctly.
    function test_constructor_succeeds() external {
        assertEq(crossL2Inbox.l1Block(), address(l1Block));
    }

    function testFuzz_executeMessage_succeeds(
        bytes calldata _msg,
        ICrossL2Inbox.Identifier calldata _id,
        address _target
    )
        external
        payable
    {
        // need to make sure address leads to successfull SafeCall by executeMessage
        // TODO: how to make sure gasLeft coincides wtih executeMessage?
        bool success = SafeCall.call({ _target: _target, _gas: gasleft(), _value: msg.value, _calldata: _msg });
        vm.assume(success);

        // timestamp invariant
        vm.assume(_id.timestamp <= block.timestamp);

        // chainId invariant
        uint256[] memory chainIds = new uint256[](1);
        chainIds[0] = _id.chainId;
        vm.prank(depositor);
        l1Block.setL1BlockValues(0, 0, 0, bytes32(0), 0, bytes32(0), 0, 0, 1, chainIds);
        vm.assume(L1Block(crossL2Inbox.l1Block()).isInDependencySet(_id.chainId));

        // only EOA invariant
        vm.prank(tx.origin);

        vm.expectCall(_target, _msg);
        crossL2Inbox.executeMessage{ value: msg.value }(_msg, _id, _target);

        assertEq(crossL2Inbox.origin(), _id.origin);
        assertEq(crossL2Inbox.blocknumber(), _id.blocknumber);
        assertEq(crossL2Inbox.logIndex(), _id.logIndex);
        assertEq(crossL2Inbox.timestamp(), _id.timestamp);
        assertEq(crossL2Inbox.chainId(), _id.chainId);
    }

    function test_executeMessage_invalidTimestamp_fails() external {
        uint256[] memory chainIds = new uint256[](1);
        chainIds[0] = 1;
        vm.prank(depositor);
        l1Block.setL1BlockValues(0, 0, 0, bytes32(0), 0, bytes32(0), 0, 0, 1, chainIds);

        ICrossL2Inbox.Identifier memory id = ICrossL2Inbox.Identifier({
            origin: address(0),
            blocknumber: 0,
            logIndex: 0,
            timestamp: block.timestamp + 1,
            chainId: chainIds[0]
        });

        vm.prank(tx.origin);

        bytes memory msg_ = abi.encode("");

        vm.expectRevert("CrossL2Inbox: invalid id timestamp");
        crossL2Inbox.executeMessage(msg_, id, address(0));
    }

    function test_executeMessage_invalidChainId_fails() external {
        ICrossL2Inbox.Identifier memory id = ICrossL2Inbox.Identifier({
            origin: address(0),
            blocknumber: 0,
            logIndex: 0,
            timestamp: block.timestamp,
            chainId: 1
        });

        vm.prank(tx.origin);

        bytes memory msg_ = abi.encode("");

        vm.expectRevert("CrossL2Inbox: invalid id chainId");
        crossL2Inbox.executeMessage(msg_, id, address(0));
    }

    function test_executeMessage_sameChainId_succeeds() external {
        ICrossL2Inbox.Identifier memory id = ICrossL2Inbox.Identifier({
            origin: address(0),
            blocknumber: 0,
            logIndex: 0,
            timestamp: block.timestamp,
            chainId: block.chainid
        });

        vm.prank(tx.origin);

        bytes memory msg_ = abi.encode("");

        crossL2Inbox.executeMessage(msg_, id, address(0));
    }

    function test_executeMessage_invalidSender_fails() external {
        uint256[] memory chainIds = new uint256[](1);
        chainIds[0] = 1;
        vm.prank(depositor);
        l1Block.setL1BlockValues(0, 0, 0, bytes32(0), 0, bytes32(0), 0, 0, 1, chainIds);

        ICrossL2Inbox.Identifier memory id = ICrossL2Inbox.Identifier({
            origin: address(0),
            blocknumber: 0,
            logIndex: 0,
            timestamp: block.timestamp,
            chainId: chainIds[0]
        });

        bytes memory msg_ = abi.encode("");

        vm.expectRevert("CrossL2Inbox: Not EOA sender");
        crossL2Inbox.executeMessage(msg_, id, address(0));
    }

    function test_executeMessage_unsuccessfullSafeCall_fails() external {
        uint256[] memory chainIds = new uint256[](1);
        chainIds[0] = block.chainid;
        vm.prank(depositor);
        l1Block.setL1BlockValues(0, 0, 0, bytes32(0), 0, bytes32(0), 0, 0, 1, chainIds);

        ICrossL2Inbox.Identifier memory id = ICrossL2Inbox.Identifier({
            origin: address(0),
            blocknumber: 0,
            logIndex: 0,
            timestamp: block.timestamp,
            chainId: chainIds[0]
        });

        vm.etch(address(0), address(new Reverter()).code);

        vm.prank(tx.origin);
        vm.expectRevert("CrossL2Inbox: call failed");
        crossL2Inbox.executeMessage(hex"1111", id, address(0));
    }
}
