pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { RLPWriter } from "../RLPWriter.sol";

contract MockRLPWriter {
    function encodeBytes(
        bytes memory self
    )
        internal
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeBytes(self);
    }

    function encodeList(
        bytes[] memory self
    )
        internal
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeList(self);
    }

    function encodeString(
        string memory self
    )
        internal
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeString(self);
    }

    function encodeAddress(
        address self
    )
        internal
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeAddress(self);
    }

    function encodeUint(
        uint self
    )
        internal
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeUint(self);
    }

    function encodeInt(
        int self
    )
        internal
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeInt(self);
    }

    function encodeBool(
        bool self
    )
        internal
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeBool(self);
    }
}