// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { L1Block } from "../L2/L1Block.sol";

contract L1BlockTest is CommonTest {
    L1Block lb;
    address depositor;
    bytes32 immutable NON_ZERO_HASH = keccak256(abi.encode(1));

    function setUp() external {
        lb = new L1Block();
        depositor = lb.DEPOSITOR_ACCOUNT();
        vm.prank(depositor);
        lb.setL1BlockValues({
            _number: uint64(1),
            _timestamp: uint64(2),
            _basefee: 3,
            _hash: NON_ZERO_HASH,
            _sequenceNumber: uint64(4),
            _batcherHash: bytes32(0),
            _l1FeeOverhead: 2,
            _l1FeeScalar: 3
        });
    }

    function test_updatesValues(
        uint64 n,
        uint64 t,
        uint256 b,
        bytes32 h,
        uint64 s,
        bytes32 bt,
        uint256 fo,
        uint256 fs
    ) external {
        vm.prank(depositor);
        lb.setL1BlockValues(n, t, b, h, s, bt, fo, fs);
        assertEq(lb.number(), n);
        assertEq(lb.timestamp(), t);
        assertEq(lb.basefee(), b);
        assertEq(lb.hash(), h);
        assertEq(lb.sequenceNumber(), s);
        assertEq(lb.batcherHash(), bt);
        assertEq(lb.l1FeeOverhead(), fo);
        assertEq(lb.l1FeeScalar(), fs);
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

    function test_hash() external {
        assertEq(lb.hash(), NON_ZERO_HASH);
    }

    function test_sequenceNumber() external {
        assertEq(lb.sequenceNumber(), uint64(4));
    }

    function test_updateValues() external {
        vm.prank(depositor);
        lb.setL1BlockValues({
            _number: type(uint64).max,
            _timestamp: type(uint64).max,
            _basefee: type(uint256).max,
            _hash: keccak256(abi.encode(1)),
            _sequenceNumber: type(uint64).max,
            _batcherHash: bytes32(type(uint256).max),
            _l1FeeOverhead: type(uint256).max,
            _l1FeeScalar: type(uint256).max
        });
    }
}
