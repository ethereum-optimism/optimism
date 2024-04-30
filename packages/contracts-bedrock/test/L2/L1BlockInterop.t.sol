// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { L1BlockInterop, DependencySetSizeMismatch, NotDepositor } from "src/L2/L1BlockInterop.sol";

contract L1BlockInterop_Test is Test {
    L1BlockInterop l1Block;
    address depositor;

    function setUp() public {
        l1Block = new L1BlockInterop();
        depositor = l1Block.DEPOSITOR_ACCOUNT();
    }

    /// @dev Tests that setL1BlockValuesInterop updates the values appropriately.
    function testFuzz_setL1BlockValuesInterop_succeeds(
        uint32 baseFeeScalar,
        uint32 blobBaseFeeScalar,
        uint64 sequenceNumber,
        uint64 timestamp,
        uint64 number,
        uint256 baseFee,
        uint256 blobBaseFee,
        bytes32 hash,
        bytes32 batcherHash,
        uint256[] calldata dependencySet
    )
        external
    {
        vm.assume(dependencySet.length <= type(uint8).max);
        vm.assume(uint160(uint256(batcherHash)) == uint256(batcherHash));

        bytes memory functionCallDataPacked = Encoding.encodeSetL1BlockValuesInterop({
            _baseFeeScalar: baseFeeScalar,
            _blobBaseFeeScalar: blobBaseFeeScalar,
            _sequenceNumber: sequenceNumber,
            _timestamp: timestamp,
            _number: number,
            _baseFee: baseFee,
            _blobBaseFee: blobBaseFee,
            _hash: hash,
            _batcherHash: batcherHash,
            _dependencySet: dependencySet
        });

        vm.prank(depositor);
        (bool success,) = address(l1Block).call(functionCallDataPacked);
        assertTrue(success, "Function call failed");

        assertEq(l1Block.baseFeeScalar(), baseFeeScalar);
        assertEq(l1Block.blobBaseFeeScalar(), blobBaseFeeScalar);
        assertEq(l1Block.sequenceNumber(), sequenceNumber);
        assertEq(l1Block.timestamp(), timestamp);
        assertEq(l1Block.number(), number);
        assertEq(l1Block.basefee(), baseFee);
        assertEq(l1Block.blobBaseFee(), blobBaseFee);
        assertEq(l1Block.hash(), hash);
        assertEq(l1Block.batcherHash(), batcherHash);
        assertEq(l1Block.dependencySetSize(), dependencySet.length);
        for (uint256 i = 0; i < dependencySet.length; i++) {
            assertEq(l1Block.dependencySet(i), dependencySet[i]);
            assertTrue(l1Block.isInDependencySet(dependencySet[i]));
        }

        // ensure we didn't accidentally pollute the 128 bits of the sequencenum+scalars slot that
        // should be empty
        bytes32 scalarsSlot = vm.load(address(l1Block), bytes32(uint256(3)));
        bytes32 mask128 = hex"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF00000000000000000000000000000000";

        assertEq(0, scalarsSlot & mask128);

        // ensure we didn't accidentally pollute the 128 bits of the number & timestamp slot that
        // should be empty
        bytes32 numberTimestampSlot = vm.load(address(l1Block), bytes32(uint256(0)));
        assertEq(0, numberTimestampSlot & mask128);
    }

    /// @dev Tests that `setL1BlockValuesInterop` succeeds if sender address is the depositor
    function test_setL1BlockValuesInterop_isDepositor_succeeds() external {
        bytes memory functionCallDataPacked = Encoding.encodeSetL1BlockValuesInterop({
            _baseFeeScalar: type(uint32).max,
            _blobBaseFeeScalar: type(uint32).max,
            _sequenceNumber: type(uint64).max,
            _timestamp: type(uint64).max,
            _number: type(uint64).max,
            _baseFee: type(uint256).max,
            _blobBaseFee: type(uint256).max,
            _hash: bytes32(type(uint256).max),
            _batcherHash: bytes32(0),
            _dependencySet: new uint256[](0)
        });

        vm.prank(depositor);
        (bool success,) = address(l1Block).call(functionCallDataPacked);
        assertTrue(success, "function call failed");
    }

    /// @dev Tests that `setL1BlockValuesInterop` reverts if sender address is not the depositor
    function test_setL1BlockValuesInterop_isDepositor_reverts() external {
        bytes memory functionCallDataPacked = Encoding.encodeSetL1BlockValuesInterop({
            _baseFeeScalar: type(uint32).max,
            _blobBaseFeeScalar: type(uint32).max,
            _sequenceNumber: type(uint64).max,
            _timestamp: type(uint64).max,
            _number: type(uint64).max,
            _baseFee: type(uint256).max,
            _blobBaseFee: type(uint256).max,
            _hash: bytes32(type(uint256).max),
            _batcherHash: bytes32(0),
            _dependencySet: new uint256[](0)
        });

        (bool success, bytes memory data) = address(l1Block).call(functionCallDataPacked);
        assertTrue(!success, "function call should have failed");
        // make sure return value is the expected function selector for "NotDepositor()"
        assertEq(bytes4(data), NotDepositor.selector);
    }

    /// @dev Tests that `setL1BlockValuesInterop` succeeds if _dependencySetSize is the same as
    ///      the length of _dependencySet. (happy path)
    function testFuzz_setL1BlockValuesInterop_dependencySetSizeMatch_succeeds(uint256[] calldata dependencySet)
        external
    {
        vm.assume(dependencySet.length <= type(uint8).max);

        bytes memory functionCallDataPacked = Encoding.encodeSetL1BlockValuesInterop({
            _baseFeeScalar: type(uint32).max,
            _blobBaseFeeScalar: type(uint32).max,
            _sequenceNumber: type(uint64).max,
            _timestamp: type(uint64).max,
            _number: type(uint64).max,
            _baseFee: type(uint256).max,
            _blobBaseFee: type(uint256).max,
            _hash: bytes32(type(uint256).max),
            _batcherHash: bytes32(0),
            _dependencySet: dependencySet
        });

        vm.prank(depositor);
        (bool success,) = address(l1Block).call(functionCallDataPacked);
        assertTrue(success, "function call failed");
    }

    /// @dev Tests that `setL1BlockValuesInterop` reverts if _dependencySetSize is not the same as
    ///      the length of _dependencySet. (bad path)
    function testFuzz_setL1BlockValuesInterop_dependencySetSizeMatch_reverts(
        uint8 notDependencySetSize,
        uint256[] calldata dependencySet
    )
        external
    {
        vm.assume(dependencySet.length <= type(uint8).max);
        vm.assume(notDependencySetSize != dependencySet.length);

        bytes memory functionCallDataPacked = abi.encodePacked(
            bytes4(keccak256("setL1BlockValuesInterop()")),
            type(uint32).max,
            type(uint32).max,
            type(uint64).max,
            type(uint64).max,
            type(uint64).max,
            type(uint256).max,
            type(uint256).max,
            bytes32(type(uint256).max),
            bytes32(type(uint256).max),
            notDependencySetSize,
            dependencySet
        );

        vm.prank(depositor);
        (bool success, bytes memory data) = address(l1Block).call(functionCallDataPacked);
        assertTrue(!success, "function call should have failed");
        // make sure return value is the expected function selector for "DependencySetSizeMismatch()"
        assertEq(bytes4(data), DependencySetSizeMismatch.selector);
    }

    /// @dev Tests that an arbitrary dependency set can be set and that ìsInDependencySet returns
    ///      the expected results.
    function testFuzz_isInDependencySet_succeeds(
        uint32 baseFeeScalar,
        uint32 blobBaseFeeScalar,
        uint64 sequenceNumber,
        uint64 timestamp,
        uint64 number,
        uint256 baseFee,
        uint256 blobBaseFee,
        bytes32 hash,
        bytes32 batcherHash,
        uint256[] calldata dependencySet
    )
        external
    {
        vm.assume(dependencySet.length <= type(uint8).max);
        vm.assume(uint160(uint256(batcherHash)) == uint256(batcherHash));

        bytes memory functionCallDataPacked = Encoding.encodeSetL1BlockValuesInterop({
            _baseFeeScalar: baseFeeScalar,
            _blobBaseFeeScalar: blobBaseFeeScalar,
            _sequenceNumber: sequenceNumber,
            _timestamp: timestamp,
            _number: number,
            _baseFee: baseFee,
            _blobBaseFee: blobBaseFee,
            _hash: hash,
            _batcherHash: batcherHash,
            _dependencySet: dependencySet
        });

        vm.prank(depositor);
        (bool success,) = address(l1Block).call(functionCallDataPacked);
        assertTrue(success, "Function call failed");

        assertEq(l1Block.dependencySetSize(), dependencySet.length);

        for (uint256 i = 0; i < dependencySet.length; i++) {
            assertTrue(l1Block.isInDependencySet(dependencySet[i]));
        }
    }

    /// @dev Tests that `isInDependencySet` returns true when the current chain ID is passed as the input
    function test_isInDependencySet_isChainId_succeeds() external view {
        assertTrue(l1Block.isInDependencySet(block.chainid));
    }

    /// @dev Tests that `isInDependencySet` reverts when the input chain ID is not in the dependency set
    function testFuzz_isInDependencySet_reverts(uint256 chainId) external {
        vm.assume(chainId != 1);

        uint256[] memory dependencySet = new uint256[](1);
        dependencySet[0] = 1;

        bytes memory functionCallDataPacked = Encoding.encodeSetL1BlockValuesInterop({
            _baseFeeScalar: 0,
            _blobBaseFeeScalar: 0,
            _sequenceNumber: 0,
            _timestamp: 0,
            _number: 0,
            _baseFee: 0,
            _blobBaseFee: 0,
            _hash: bytes32(0),
            _batcherHash: bytes32(0),
            _dependencySet: dependencySet
        });

        vm.prank(depositor);
        (bool success,) = address(l1Block).call(functionCallDataPacked);
        assertTrue(success, "Function call failed");

        assertFalse(l1Block.isInDependencySet(chainId));
    }

    /// @dev Tests that `isInDependencySet` returns false when the dependency set is empty
    function testFuzz_isInDependencySet_dependencySetEmpty_succeeds(uint256 chainId) external view {
        assertTrue(l1Block.dependencySetSize() == 0);
        assertFalse(l1Block.isInDependencySet(chainId));
    }
}
