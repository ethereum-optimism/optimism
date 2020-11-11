// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../optimistic-ethereum/libraries/codec/Lib_OVMCodec.sol";

/**
 * @title TestLib_OVMCodec
 */
contract TestLib_OVMCodec {

    function decodeEIP155Transaction(
        bytes memory _transaction,
        bool _isEthSignedMessage
    )
        public
        pure
        returns (
            Lib_OVMCodec.EIP155Transaction memory _decoded
        )
    {
        return Lib_OVMCodec.decodeEIP155Transaction(_transaction, _isEthSignedMessage);
    }

    function encodeTransaction(
        Lib_OVMCodec.Transaction memory _transaction
    )
        public
        pure
        returns (
            bytes memory _encoded
        )
    {
        return Lib_OVMCodec.encodeTransaction(_transaction);
    }

    function hashTransaction(
        Lib_OVMCodec.Transaction memory _transaction
    )
        public
        pure
        returns (
            bytes32 _hash
        )
    {
        return Lib_OVMCodec.hashTransaction(_transaction);
    }

    function decompressEIP155Transaction(
        bytes memory _transaction
    )
        public
        pure
        returns (
            Lib_OVMCodec.EIP155Transaction memory _decompressed
        )
    {
        return Lib_OVMCodec.decompressEIP155Transaction(_transaction);
    }
}
