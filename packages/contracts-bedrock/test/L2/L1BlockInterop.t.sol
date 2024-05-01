// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { L1BlockInterop, DependencySetSizeMismatch, NotDepositor } from "src/L2/L1BlockInterop.sol";

contract L1BlockInteropTest is Test {
    L1BlockInterop l1Block;
    address depositor;

    function setUp() public {
        l1Block = new L1BlockInterop();
        depositor = l1Block.DEPOSITOR_ACCOUNT();
    }

    /// @dev Tests that setL1BlockValuesInterop updates the values appropriately.
    function testFuzz_setL1BlockValuesInterop_succeeds(
        uint32 _baseFeeScalar,
        uint32 _blobBaseFeeScalar,
        uint64 _sequenceNumber,
        uint64 _timestamp,
        uint64 _number,
        uint256 _baseFee,
        uint256 _blobBaseFee,
        bytes32 _hash,
        bytes32 _batcherHash,
        uint256[] calldata _dependencySet
    )
        external
    {
        vm.assume(_dependencySet.length <= type(uint8).max);
        vm.assume(uint160(uint256(_batcherHash)) == uint256(_batcherHash));

        bytes memory functionCallDataPacked = Encoding.encodeSetL1BlockValuesInterop({
            _baseFeeScalar: _baseFeeScalar,
            _blobBaseFeeScalar: _blobBaseFeeScalar,
            _sequenceNumber: _sequenceNumber,
            _timestamp: _timestamp,
            _number: _number,
            _baseFee: _baseFee,
            _blobBaseFee: _blobBaseFee,
            _hash: _hash,
            _batcherHash: _batcherHash,
            _dependencySet: _dependencySet
        });

        vm.prank(depositor);
        (bool success,) = address(l1Block).call(functionCallDataPacked);
        assertTrue(success, "Function call failed");

        assertEq(l1Block.baseFeeScalar(), _baseFeeScalar);
        assertEq(l1Block.blobBaseFeeScalar(), _blobBaseFeeScalar);
        assertEq(l1Block.sequenceNumber(), _sequenceNumber);
        assertEq(l1Block.timestamp(), _timestamp);
        assertEq(l1Block.number(), _number);
        assertEq(l1Block.basefee(), _baseFee);
        assertEq(l1Block.blobBaseFee(), _blobBaseFee);
        assertEq(l1Block.hash(), _hash);
        assertEq(l1Block.batcherHash(), _batcherHash);
        assertEq(l1Block.dependencySetSize(), _dependencySet.length);
        for (uint256 i = 0; i < _dependencySet.length; i++) {
            assertEq(l1Block.dependencySet(i), _dependencySet[i]);
            assertTrue(l1Block.isInDependencySet(_dependencySet[i]));
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

    /// @dev Tests that `setL1BlockValuesInterop` reverts if _dependencySetSize is not the same as
    ///      the length of _dependencySet. (bad path)
    function testFuzz_setL1BlockValuesInterop_dependencySetSizeMatch_reverts(
        uint8 _notDependencySetSize,
        uint256[] calldata _dependencySet
    )
        external
    {
        vm.assume(_dependencySet.length <= type(uint8).max);
        vm.assume(_notDependencySetSize != _dependencySet.length);

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
            _notDependencySetSize,
            _dependencySet
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
        uint32 _baseFeeScalar,
        uint32 _blobBaseFeeScalar,
        uint64 _sequenceNumber,
        uint64 _timestamp,
        uint64 _number,
        uint256 _baseFee,
        uint256 _blobBaseFee,
        bytes32 _hash,
        bytes32 _batcherHash,
        uint256[] calldata _dependencySet
    )
        external
    {
        vm.assume(_dependencySet.length <= type(uint8).max);
        vm.assume(uint160(uint256(_batcherHash)) == uint256(_batcherHash));

        bytes memory functionCallDataPacked = Encoding.encodeSetL1BlockValuesInterop({
            _baseFeeScalar: _baseFeeScalar,
            _blobBaseFeeScalar: _blobBaseFeeScalar,
            _sequenceNumber: _sequenceNumber,
            _timestamp: _timestamp,
            _number: _number,
            _baseFee: _baseFee,
            _blobBaseFee: _blobBaseFee,
            _hash: _hash,
            _batcherHash: _batcherHash,
            _dependencySet: _dependencySet
        });

        vm.prank(depositor);
        (bool success,) = address(l1Block).call(functionCallDataPacked);
        assertTrue(success, "Function call failed");

        assertEq(l1Block.dependencySetSize(), _dependencySet.length);

        for (uint256 i = 0; i < _dependencySet.length; i++) {
            assertTrue(l1Block.isInDependencySet(_dependencySet[i]));
        }
    }

    /// @dev Tests that `isInDependencySet` returns true when the current chain ID is passed as the input
    function test_isInDependencySet_isChainId_succeeds() external view {
        assertTrue(l1Block.isInDependencySet(block.chainid));
    }

    /// @dev Tests that `isInDependencySet` reverts when the input chain ID is not in the dependency set
    function testFuzz_isInDependencySet_reverts(uint256 _chainId) external {
        vm.assume(_chainId != 1);

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

        assertFalse(l1Block.isInDependencySet(_chainId));
    }

    /// @dev Tests that `isInDependencySet` returns false when the dependency set is empty
    function testFuzz_isInDependencySet_dependencySetEmpty_succeeds(uint256 _chainId) external view {
        assertTrue(l1Block.dependencySetSize() == 0);
        assertFalse(l1Block.isInDependencySet(_chainId));
    }
}
