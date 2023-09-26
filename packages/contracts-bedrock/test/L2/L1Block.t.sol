// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Target contract
import { L1Block } from "src/L2/L1Block.sol";

import { RING_SIZE } from "src/libraries/RingLib.sol";

contract L1BlockTest is CommonTest {
    address depositor;
    bytes32 immutable NON_ZERO_HASH = keccak256(abi.encode(1));

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        depositor = l1Block.DEPOSITOR_ACCOUNT();
    }

    /// @dev Tests that `setL1BlockValues` sets the values correctly.
    function testFuzz_setValues_succeeds(
        uint64 n,
        uint64 t,
        uint256 b,
        bytes32 h,
        uint64 s,
        bytes32 bt,
        uint256 fo,
        uint256 fs
    )
        external
    {
        vm.assume(h != bytes32(0));

        vm.prank(depositor);
        l1Block.setL1BlockValues({
            _number: n,
            _timestamp: t,
            _basefee: b,
            _hash: h,
            _sequenceNumber: s,
            _batcherHash: bt,
            _l1FeeOverhead: fo,
            _l1FeeScalar: fs
        });
        assertEq(l1Block.number(), n);
        assertEq(l1Block.timestamp(), t);
        assertEq(l1Block.basefee(), b);
        assertEq(l1Block.hash(), h);
        assertEq(l1Block.sequenceNumber(), s);
        assertEq(l1Block.batcherHash(), bt);
        assertEq(l1Block.l1FeeOverhead(), fo);
        assertEq(l1Block.l1FeeScalar(), fs);
        assertEq(l1Block.getL1BlockHash(n), h);
    }

    /// @dev Tests that `setL1BlockValues` updates the values correctly.
    function testFuzz_updatesValues_succeeds(
        uint64 n,
        uint64 t,
        uint256 b,
        bytes32 h,
        uint64 s,
        bytes32 bt,
        uint256 fo,
        uint256 fs
    )
        external
    {
        vm.assume(n > 1 && h != bytes32(0));

        vm.prank(depositor);
        l1Block.setL1BlockValues({
            _number: uint64(1),
            _timestamp: uint64(2),
            _basefee: 3,
            _hash: NON_ZERO_HASH,
            _sequenceNumber: uint64(4),
            _batcherHash: bytes32(0),
            _l1FeeOverhead: 2,
            _l1FeeScalar: 3
        });

        assertEq(l1Block.number(), uint64(1));
        assertEq(l1Block.timestamp(), uint64(2));
        assertEq(l1Block.basefee(), 3);
        assertEq(l1Block.hash(), NON_ZERO_HASH);
        assertEq(l1Block.sequenceNumber(), uint64(4));
        assertEq(l1Block.batcherHash(), bytes32(0));
        assertEq(l1Block.l1FeeOverhead(), 2);
        assertEq(l1Block.l1FeeScalar(), 3);
        assertEq(l1Block.getL1BlockHash(1), NON_ZERO_HASH);

        vm.prank(depositor);
        l1Block.setL1BlockValues(n, t, b, h, s, bt, fo, fs);

        assertEq(l1Block.number(), n);
        assertEq(l1Block.timestamp(), t);
        assertEq(l1Block.basefee(), b);
        assertEq(l1Block.hash(), h);
        assertEq(l1Block.sequenceNumber(), s);
        assertEq(l1Block.batcherHash(), bt);
        assertEq(l1Block.l1FeeOverhead(), fo);
        assertEq(l1Block.l1FeeScalar(), fs);
        assertEq(l1Block.getL1BlockHash(n), h);
    }

    /// @dev Tests that `setL1BlockValues` can set max values.
    function test_updateValues_maxValues_succeeds() external {
        vm.prank(depositor);
        l1Block.setL1BlockValues({
            _number: type(uint64).max,
            _timestamp: type(uint64).max,
            _basefee: type(uint256).max,
            _hash: keccak256(abi.encode(1)),
            _sequenceNumber: type(uint64).max,
            _batcherHash: bytes32(type(uint256).max),
            _l1FeeOverhead: type(uint256).max,
            _l1FeeScalar: type(uint256).max
        });

        assertEq(l1Block.number(), type(uint64).max);
        assertEq(l1Block.timestamp(), type(uint64).max);
        assertEq(l1Block.basefee(), type(uint256).max);
        assertEq(l1Block.hash(), keccak256(abi.encode(1)));
        assertEq(l1Block.sequenceNumber(), type(uint64).max);
        assertEq(l1Block.batcherHash(), bytes32(type(uint256).max));
        assertEq(l1Block.l1FeeOverhead(), type(uint256).max);
        assertEq(l1Block.l1FeeScalar(), type(uint256).max);
        assertEq(l1Block.getL1BlockHash(type(uint64).max), keccak256(abi.encode(1)));
    }

    /// @dev Tests that `getL1BlockHash` reverts when query
    ///      block hash not being set for lower bound.
    function testFuzz_getL1BlockHash_lowerBound_reverts(uint64 m, uint64 n) external {
        vm.assume(m < n && n < type(uint64).max - RING_SIZE);

        vm.prank(depositor);
        l1Block.setL1BlockValues({
            _number: n,
            _timestamp: uint64(2),
            _basefee: 3,
            _hash: keccak256(abi.encode(m)),
            _sequenceNumber: uint64(4),
            _batcherHash: bytes32(0),
            _l1FeeOverhead: 2,
            _l1FeeScalar: 3
        });

        vm.expectRevert("L1Block: hash number out of bounds");
        l1Block.getL1BlockHash(m);
    }

    /// @dev Tests that `getL1BlockHash` reverts when query
    ///      block hash not being set for upper bound.
    function testFuzz_getL1BlockHash_upperBound_reverts(uint64 n) external {
        vm.assume(n > 0);

        vm.prank(depositor);
        l1Block.setL1BlockValues({
            _number: uint64(0),
            _timestamp: uint64(2),
            _basefee: 3,
            _hash: keccak256(abi.encode(1)),
            _sequenceNumber: uint64(4),
            _batcherHash: bytes32(0),
            _l1FeeOverhead: 2,
            _l1FeeScalar: 3
        });

        vm.expectRevert("L1Block: hash number out of bounds");
        l1Block.getL1BlockHash(n);
    }

    /// @dev Tests that `getL1BlockHash` when overwrite block hash.
    function testFuzz_getL1BlockHash_overwrite_succeeds(uint64 n) external {
        vm.startPrank(depositor);
        l1Block.setL1BlockValues({
            _number: n,
            _timestamp: uint64(2),
            _basefee: 3,
            _hash: keccak256(abi.encode(n)),
            _sequenceNumber: uint64(4),
            _batcherHash: bytes32(0),
            _l1FeeOverhead: 2,
            _l1FeeScalar: 3
        });

        l1Block.setL1BlockValues({
            _number: n,
            _timestamp: uint64(2),
            _basefee: 3,
            _hash: keccak256(abi.encode(n)),
            _sequenceNumber: uint64(4),
            _batcherHash: bytes32(0),
            _l1FeeOverhead: 2,
            _l1FeeScalar: 3
        });
        vm.stopPrank();

        assertEq(l1Block.getL1BlockHash(n), keccak256(abi.encode(n)));
    }

    /// @dev Tests that `setL1BlockValues` reverts when use non depositor address.
    function testFuzz_setL1BlockValues_nonDepositor_reverts(address nonDepositor) external {
        vm.assume(nonDepositor != depositor);

        vm.prank(nonDepositor);
        vm.expectRevert("L1Block: only the depositor account can set L1 block values");
        l1Block.setL1BlockValues({
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

    /// @dev Tests that `getL1BlockHash` returns correct values when overwrite the ring buffer.
    function test_getL1BlockHash_overwrittenRingBuffer_succeeds() external {
        vm.startPrank(depositor);

        uint64 lowerBound = 0;
        uint8 m = 3;

        for (uint64 i = 1; i <= uint256(m) * RING_SIZE + (m - 1); i++) {
            l1Block.setL1BlockValues({
                _number: i,
                _timestamp: uint64(0),
                _basefee: 0,
                _hash: keccak256(abi.encodePacked(i)),
                _sequenceNumber: uint64(0),
                _batcherHash: bytes32(0),
                _l1FeeOverhead: 0,
                _l1FeeScalar: 0
            });

            if (i % RING_SIZE == 0) {
                lowerBound = i - RING_SIZE + 1;

                vm.expectRevert("L1Block: hash number out of bounds");
                l1Block.getL1BlockHash(lowerBound - 1);

                for (uint64 k = lowerBound; k < i + 1; k++) {
                    assertEq(l1Block.getL1BlockHash(k), keccak256(abi.encodePacked(k)));
                }

                vm.expectRevert("L1Block: hash number out of bounds");
                l1Block.getL1BlockHash(i + 1);
            }
        }

        vm.stopPrank();
    }
}
