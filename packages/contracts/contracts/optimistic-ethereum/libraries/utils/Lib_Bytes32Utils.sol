// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;

/**
 * @title Lib_Byte32Utils
 */
library Lib_Bytes32Utils {

    /**********************
     * Internal Functions *
     **********************/

    function toBool(
        bytes32 _in
    )
        internal
        pure
        returns (
            bool _out
        )
    {
        return _in != 0;
    }

    function fromBool(
        bool _in
    )
        internal
        pure
        returns (
            bytes32 _out
        )
    {
        return bytes32(uint256(_in ? 1 : 0));
    }

    function toAddress(
        bytes32 _in
    )
        internal
        pure
        returns (
            address _out
        )
    {
        return address(uint160(uint256(_in)));
    }

    function fromAddress(
        address _in
    )
        internal
        pure
        returns (
            bytes32 _out
        )
    {
        return bytes32(uint256(_in));
    }

    function removeLeadingZeros(
        bytes32 _in
    )
        internal
        pure
        returns (
            bytes memory _out
        )
    {
        bytes memory out;

        assembly {
            // Figure out how many leading zero bytes to remove.
            let shift := 0
            for { let i := 0 } and(lt(i, 32), eq(byte(i, _in), 0)) { i := add(i, 1) } {
                shift := add(shift, 1)
            }

            // Reserve some space for our output and fix the free memory pointer.
            out := mload(0x40)
            mstore(0x40, add(out, 0x40))

            // Shift the value and store it into the output bytes.
            mstore(add(out, 0x20), shl(mul(shift, 8), _in))

            // Store the new size (with leading zero bytes removed) in the output byte size.
            mstore(out, sub(32, shift))
        }

        return out;
    }
}
