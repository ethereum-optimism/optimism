// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { Reverter, ConfigurableCaller } from "test/mocks/Callers.sol";
import { ICrossL2Inbox } from "src/L2/ICrossL2Inbox.sol";

// Libraries
import { SafeCall } from "src/libraries/SafeCall.sol";

import { L1Block } from "src/L2/L1Block.sol";

contract CrossL2InboxTest is CommonTest {
    ICrossL2Inbox.Identifier sampleId = ICrossL2Inbox.Identifier({
        origin: address(0),
        blocknumber: 0,
        logIndex: 0,
        timestamp: block.timestamp,
        chainId: block.chainid
    });

    address depositor;

    function setUp() public virtual override {
        super.setUp();
        depositor = l1Block.DEPOSITOR_ACCOUNT();
    }

    function testFuzz_executeMessage_succeeds(
        bytes calldata _msg,
        ICrossL2Inbox.Identifier calldata _id,
        address _target
    )
        external
        payable
    {
        vm.assume(_id.timestamp <= block.timestamp);

        // need to prevent call to L1Block.isInDependencySet from reverting
        vm.mockCall(
            address(l1Block), abi.encodeWithSelector(L1Block.isInDependencySet.selector, _id.chainId), abi.encode(true)
        );

        // need to prevent underlying SafeCall to target from reverting
        vm.etch(_target, address(0).code);

        // executeMessage
        vm.prank(tx.origin);
        vm.expectCall(_target, _msg);
        crossL2Inbox.executeMessage{ value: msg.value }(_id, _target, _msg);

        assertEq(crossL2Inbox.origin(), _id.origin);
        assertEq(crossL2Inbox.blocknumber(), _id.blocknumber);
        assertEq(crossL2Inbox.logIndex(), _id.logIndex);
        assertEq(crossL2Inbox.timestamp(), _id.timestamp);
        assertEq(crossL2Inbox.chainId(), _id.chainId);
    }

    function test_executeMessage_invalidTimestamp_fails() external {
        ICrossL2Inbox.Identifier memory id = sampleId;
        id.timestamp = block.timestamp + 1;

        vm.prank(tx.origin);
        vm.expectRevert("CrossL2Inbox: invalid id timestamp");
        crossL2Inbox.executeMessage(id, address(0), hex"1234");
    }

    function test_executeMessage_invalidChainId_fails() external {
        ICrossL2Inbox.Identifier memory id = sampleId;
        id.chainId = 1;

        vm.prank(tx.origin);
        vm.expectRevert("CrossL2Inbox: id chain not in dependency set");
        crossL2Inbox.executeMessage(id, address(0), hex"1234");
    }

    function test_executeMessage_sameChainId_succeeds() external {
        vm.prank(tx.origin);
        crossL2Inbox.executeMessage(sampleId, address(0), hex"1234");
    }

    function test_executeMessage_invalidSender_fails() external {
        vm.expectRevert("CrossL2Inbox: not EOA sender");
        crossL2Inbox.executeMessage(sampleId, address(0), hex"1234");
    }

    function test_executeMessage_unsuccessfullSafeCall_fails() external {
        // need to make sure address leads to unsuccessfull SafeCall by executeMessage
        vm.etch(address(0), address(new Reverter()).code);

        vm.prank(tx.origin);
        vm.expectRevert("CrossL2Inbox: target call failed");
        crossL2Inbox.executeMessage(sampleId, address(0), hex"1234");
    }
}
