// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Target contract
import { L1Block } from "src/L2/L1Block.sol";

contract L1BlockTest is CommonTest {
    address depositor;

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();

        depositor = l1Block.DEPOSITOR_ACCOUNT();
        vm.prank(depositor);
        l1Block.setL1BlockValues({
            _number: uint64(1),
            _timestamp: uint64(2),
            _basefee: 3,
            _hash: keccak256(abi.encode(block.number)),
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
    )
        external
    {
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
    }

    /// @dev Tests that `number` returns the correct value.
    function test_number_succeeds() external {
        assertEq(l1Block.number(), uint64(1));
    }

    /// @dev Tests that `timestamp` returns the correct value.
    function test_timestamp_succeeds() external {
        assertEq(l1Block.timestamp(), uint64(2));
    }

    /// @dev Tests that `basefee` returns the correct value.
    function test_basefee_succeeds() external {
        assertEq(l1Block.basefee(), 3);
    }

    /// @dev Tests that `hash` returns the correct value.
    function test_hash_succeeds() external {
        assertEq(l1Block.hash(), keccak256(abi.encode(block.number)));
    }

    /// @dev Tests that `sequenceNumber` returns the correct value.
    function test_sequenceNumber_succeeds() external {
        assertEq(l1Block.sequenceNumber(), uint64(4));
    }

    /// @dev Tests that `setL1BlockValues` can set max values.
    function test_updateValues_succeeds() external {
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
    }
}
