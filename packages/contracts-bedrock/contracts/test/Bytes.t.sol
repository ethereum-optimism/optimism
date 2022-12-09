pragma solidity 0.8.15;

import { Bytes } from "../libraries/Bytes.sol";
import { Test } from "forge-std/Test.sol";

contract Bytes_Test is Test {
    /// @dev Test that the `toBytes` function works as expected.
    function testFuzz_toNibbles_isEquivalent_success(bytes memory _bytes) public {
        // Get nibbles of `_bytes` via the new optimized function
        bytes memory _nibbles = Bytes.toNibbles(_bytes);

        // Get nibbles of `_bytes` via the old function's method.
        uint256 bytesLength = _bytes.length;
        bytes memory nibbles = new bytes(bytesLength * 2);
        bytes1 b;

        for (uint256 i = 0; i < bytesLength; ) {
            b = _bytes[i];
            nibbles[i * 2] = b >> 4;
            nibbles[i * 2 + 1] = b & 0x0f;
            ++i;
        }

        // Ensure that the two implementations are equivalent
        assertEq(_nibbles, nibbles);
    }

    /// @dev Test that the `equals` function works as expected.
    function testFuzz_equals_success(bytes memory _a) public {
        assertTrue(Bytes.equal(_a, _a));
        assertEq(_a, _a);
    }

    /// @dev Test that the `equals` function returns false when passed
    /// non-equal byte arrays.
    function testFuzz_equals_failure(bytes memory _a, bytes memory _b) public {
        vm.assume(keccak256(_a) != keccak256(_b));

        assertFalse(Bytes.equal(_a, _b));
    }
}
