//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { DSTest } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";
import { L1Block } from "../L2/L1Block.sol";

contract L1BLockTest is DSTest {
    Vm vm = Vm(HEVM_ADDRESS);
    L1Block lb;
    address depositor;
    bytes32 immutable NON_ZERO_HASH = keccak256(abi.encode(1));

    function setUp() external {
        lb = new L1Block();
        depositor = lb.DEPOSITOR_ACCOUNT();
        vm.prank(depositor);
        lb.setL1BlockValues(1, 2, 3, NON_ZERO_HASH);
    }

    function test_number() external {
        assertEq(lb.number(), 1);
    }

    function test_timestamp() external {
        assertEq(lb.timestamp(), 2);
    }

    function test_basefee() external {
        assertEq(lb.basefee(), 3);
    }

    function test_hash() external {
        assertEq(lb.hash(), NON_ZERO_HASH);
    }
}
