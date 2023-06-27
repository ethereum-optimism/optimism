// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "./CommonTest.t.sol";

// Target contract
import { L1Block } from "../L2/L1Block.sol";

contract L1BlockTest is CommonTest {
    L1Block lb;
    address depositor;
    bytes32 immutable NON_ZERO_HASH = keccak256(abi.encode(1));

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
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

    /// @dev Tests that `number` returns the correct value.
    function test_number_succeeds() external {
        assertEq(lb.number(), uint64(1));
    }

    /// @dev Tests that `timestamp` returns the correct value.
    function test_timestamp_succeeds() external {
        assertEq(lb.timestamp(), uint64(2));
    }

    /// @dev Tests that `basefee` returns the correct value.
    function test_basefee_succeeds() external {
        assertEq(lb.basefee(), 3);
    }

    /// @dev Tests that `hash` returns the correct value.
    function test_hash_succeeds() external {
        assertEq(lb.hash(), NON_ZERO_HASH);
    }

    /// @dev Tests that `sequenceNumber` returns the correct value.
    function test_sequenceNumber_succeeds() external {
        assertEq(lb.sequenceNumber(), uint64(4));
    }

    /// @dev Tests that `setL1BlockValues` can set max values.
    function test_updateValues_succeeds() external {
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
