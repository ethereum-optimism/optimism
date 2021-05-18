// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_Buffer } from "../../optimistic-ethereum/libraries/utils/Lib_Buffer.sol";

/**
 * @title TestLib_Buffer
 */
contract TestLib_Buffer {
    using Lib_Buffer for Lib_Buffer.Buffer;

    Lib_Buffer.Buffer internal buf;

    function push(
        bytes32 _value,
        bytes27 _extraData
    )
        public
    {
        buf.push(
            _value,
            _extraData
        );
    }

    function get(
        uint256 _index
    )
        public
        view
        returns (
            bytes32
        )
    {
        return buf.get(_index);
    }

    function deleteElementsAfterInclusive(
        uint40 _index,
        bytes27 _extraData
    )
        public
    {
        return buf.deleteElementsAfterInclusive(
            _index,
            _extraData
        );
    }

    function getLength()
        public
        view
        returns (
            uint40
        )
    {
        return buf.getLength();
    }

    function getExtraData()
        public
        view
        returns (
            bytes27
        )
    {
        return buf.getExtraData();
    }
}
