//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { CommonTest } from "./CommonTest.t.sol";
import { L1Block } from "../L2/L1Block.sol";
import { Lib_FeeSmoothing } from "../libraries/Lib_FeeSmoothing.sol";

contract L1BlockTest is CommonTest {
    L1Block lb;
    address depositor;
    bytes32 immutable NON_ZERO_HASH = keccak256(abi.encode(1));

    uint256 averageBasefee;

    function setUp() external {
        lb = new L1Block();
        depositor = lb.DEPOSITOR_ACCOUNT();
        vm.prank(depositor);
        lb.setL1BlockValues(uint64(1), uint64(2), 3, NON_ZERO_HASH, uint64(4));
    }

    function test_updatesValues(uint64 n, uint64 t, uint256 b, bytes32 h, uint64 s) external {
        averageBasefee = Lib_FeeSmoothing.rollingAverage(averageBasefee, b);

        vm.prank(depositor);
        lb.setL1BlockValues(n, t, b, h, s);
        assertEq(lb.number(), n);
        assertEq(lb.timestamp(), t);
        assertEq(lb.basefee(), b);
        assertEq(lb.averageBasefee(), averageBasefee);
        assertEq(lb.hash(), h);
        assertEq(lb.sequenceNumber(), s);
    }

    function test_number() external {
        assertEq(lb.number(), uint64(1));
    }

    function test_timestamp() external {
        assertEq(lb.timestamp(), uint64(2));
    }

    function test_basefee() external {
        assertEq(lb.basefee(), 3);
    }

    function test_averageBasefee() external {
        uint256 expect = Lib_FeeSmoothing.rollingAverage(0, 3);
        assertEq(lb.averageBasefee(), expect);
    }

    function test_hash() external {
        assertEq(lb.hash(), NON_ZERO_HASH);
    }

    function test_sequenceNumber() external {
        assertEq(lb.sequenceNumber(), uint64(4));
    }

    function test_updateValues() external {
        vm.prank(depositor);
        lb.setL1BlockValues(type(uint64).max, type(uint64).max, type(uint256).max, keccak256(abi.encode(1)), type(uint64).max);
    }
}
