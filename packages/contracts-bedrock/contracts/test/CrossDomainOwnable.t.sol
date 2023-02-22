// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest, Portal_Initializer } from "./CommonTest.t.sol";
import { CrossDomainOwnable } from "../L2/CrossDomainOwnable.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";
import { Vm, VmSafe } from "forge-std/Vm.sol";
import { Bytes32AddressLib } from "@rari-capital/solmate/src/utils/Bytes32AddressLib.sol";

contract XDomainSetter is CrossDomainOwnable {
    uint256 public value;

    function set(uint256 _value) external onlyOwner {
        value = _value;
    }
}

contract CrossDomainOwnable_Test is CommonTest {
    XDomainSetter setter;

    function setUp() public override {
        super.setUp();
        setter = new XDomainSetter();
    }

    // Check that the revert message is correct
    function test_onlyOwner_notOwner_reverts() external {
        vm.expectRevert("CrossDomainOwnable: caller is not the owner");
        setter.set(1);
    }

    // Check that making a call can set the value properly
    function test_onlyOwner_succeeds() external {
        assertEq(setter.value(), 0);

        vm.prank(AddressAliasHelper.applyL1ToL2Alias(setter.owner()));
        setter.set(1);
        assertEq(setter.value(), 1);
    }
}

contract CrossDomainOwnableThroughPortal_Test is Portal_Initializer {
    XDomainSetter setter;

    function setUp() public override {
        super.setUp();

        vm.prank(alice);
        setter = new XDomainSetter();
    }

    function test_depositTransaction_crossDomainOwner_succeeds() external {
        vm.recordLogs();

        vm.prank(alice);
        op.depositTransaction({
            _to: address(setter),
            _value: 0,
            _gasLimit: 21_000,
            _isCreation: false,
            _data: abi.encodeWithSelector(XDomainSetter.set.selector, 1)
        });

        // Simulate the operation of the `op-node` by parsing data
        // from logs
        VmSafe.Log[] memory logs = vm.getRecordedLogs();
        // Only 1 log emitted
        assertEq(logs.length, 1);

        VmSafe.Log memory log = logs[0];

        // It is the expected topic
        bytes32 topic = log.topics[0];
        assertEq(topic, keccak256("TransactionDeposited(address,address,uint256,bytes)"));

        // from is indexed and the first argument to the event.
        bytes32 _from = log.topics[1];
        address from = Bytes32AddressLib.fromLast20Bytes(_from);

        assertEq(AddressAliasHelper.undoL1ToL2Alias(from), alice);

        // Make a call from the "from" value received from the log.
        // In theory the opaque data could be parsed from the log
        // and passed to a low level call to "to", but calling set
        // directly on the setter is good enough.
        vm.prank(from);
        setter.set(1);
        assertEq(setter.value(), 1);
    }
}
