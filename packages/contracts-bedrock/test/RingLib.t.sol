// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { RingLib, RING_SIZE, RING_BITS } from "src/libraries/RingLib.sol";

contract RingLibTest is Test {
    using RingLib for bytes32[RING_SIZE];

    bytes32[128] internal paddingBefore;
    bytes32[RING_SIZE] internal ring;
    bytes32[128] internal paddingAfter;

    function setUp() public {
        for (uint64 i; i < paddingBefore.length; i++) {
            paddingBefore[i] = keccak256(abi.encode("before", i));
        }

        for (uint64 i; i < paddingAfter.length; i++) {
            paddingAfter[i] = keccak256(abi.encode("after", i));
        }
    }

    /// @dev Checks the ring lib constants.
    function test_constants_succeeds() public {
        assertEq(RING_SIZE, 2 ** RING_BITS);
    }

    /// @dev Tests that `get` returns correct value.
    function test_get_succeeds() public {
        bytes32 val = keccak256(abi.encode("test"));
        ring.set(100, val);
        bytes32 _hash = ring.get(100);
        assertEq(_hash, val);
    }

    /// @dev Tests that `get` returns correct values.
    function testFuzz_get_succeeds(uint64 _index) public {
        bytes32 _hash = ring.get(_index);
        assertEq(_hash, bytes32(0));
    }

    /// @dev Tests that `set` sets the values correctly.
    function testFuzz_set_succeeds(uint64 _index, bytes32 _hash) public {
        ring.set(_index, _hash);
        assertEq(ring.get(_index), _hash);
    }

    /// @dev  Tests that `get` returns correct values when the buffer is filled.
    function testFuzz_setConsec_succeeds(uint64 _startIndex, uint256 _n, bytes32 _seed) public {
        _n = bound(_n, 0, uint256(RING_SIZE) - 1);

        bytes32[] memory values = new bytes32[](_n);

        unchecked {
            for (uint64 i = 0; i < _n; i++) {
                _seed = keccak256(abi.encode(_seed));
                values[i] = _seed;
                ring.set(_startIndex + i, values[i]);
            }

            for (uint64 i = 0; i < _n; i++) {
                assertEq(ring.get(_startIndex + i), values[i]);
            }
        }
    }
}
