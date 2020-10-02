// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_RLPWriter } from "../../optimistic-ethereum/libraries/rlp/Lib_RLPWriter.sol";

/**
 * @title TestLib_RLPWriter
 */
contract TestLib_RLPWriter {

    function encodeBytes(
        bytes memory _in
    )
        public
        pure
        returns (
            bytes memory _out
        )
    {
        return Lib_RLPWriter.encodeBytes(_in);
    }

    function encodeList(
        bytes[] memory _in
    )
        public
        pure
        returns (
            bytes memory _out
        )
    {
        return Lib_RLPWriter.encodeList(_in);
    }

    function encodeString(
        string memory _in
    )
        public
        pure
        returns (
            bytes memory _out
        )
    {
        return Lib_RLPWriter.encodeString(_in);
    }

    function encodeAddress(
        address _in
    )
        public
        pure
        returns (
            bytes memory _out
        )
    {
        return Lib_RLPWriter.encodeAddress(_in);
    }

    function encodeUint(
        uint _in
    )
        public
        pure
        returns (
            bytes memory _out
        )
    {
        return Lib_RLPWriter.encodeUint(_in);
    }

    function encodeInt(
        int _in
    )
        public
        pure
        returns (
            bytes memory _out
        )
    {
        return Lib_RLPWriter.encodeInt(_in);
    }

    function encodeBool(
        bool _in
    )
        public
        pure
        returns (
            bytes memory _out
        )
    {
        return Lib_RLPWriter.encodeBool(_in);
    }
}
