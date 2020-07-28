pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { RLPWriter } from "../RLPWriter.sol";

contract MockRLPWriter {
    function encodeBytes(
        bytes memory self
    )
        public
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeBytes(self);
    }

    function encodeList(
        bytes[] memory self
    )
        public
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeList(self);
    }

    function encodeString(
        string memory self
    )
        public
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeString(self);
    }

    function encodeAddress(
        address self
    )
        public
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeAddress(self);
    }

    function encodeUint(
        uint self
    )
        public
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeUint(self);
    }

    function encodeInt(
        int self
    )
        public
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeInt(self);
    }

    function encodeBool(
        bool self
    )
        public
        pure
        returns (bytes memory)
    {
        return RLPWriter.encodeBool(self);
    }
}