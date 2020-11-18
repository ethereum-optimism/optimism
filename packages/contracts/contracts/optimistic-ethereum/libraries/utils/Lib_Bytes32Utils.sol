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
        return bytes32(bytes20(_in));
    }
}
