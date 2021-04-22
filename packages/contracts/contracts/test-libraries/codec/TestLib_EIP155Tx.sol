// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_EIP155Tx } from "../../optimistic-ethereum/libraries/codec/Lib_EIP155Tx.sol";

/**
 * @title TestLib_EIP155Tx
 */
contract TestLib_EIP155Tx {
    function decode(
        bytes memory _encoded,
        uint256 _chainId
    )
        public
        pure
        returns (
            Lib_EIP155Tx.EIP155Tx memory
        )
    {
        return Lib_EIP155Tx.decode(
            _encoded,
            _chainId
        );
    }

    function encode(
        Lib_EIP155Tx.EIP155Tx memory _transaction,
        bool _includeSignature
    )
        public
        pure
        returns (
            bytes memory
        )
    {
        return Lib_EIP155Tx.encode(
            _transaction,
            _includeSignature
        );
    }

    function hash(
        Lib_EIP155Tx.EIP155Tx memory _transaction
    )
        public
        pure
        returns (
            bytes32
        )
    {
        return Lib_EIP155Tx.hash(
            _transaction
        );
    }

    function sender(
        Lib_EIP155Tx.EIP155Tx memory _transaction
    )
        public
        pure
        returns (
            address
        )
    {
        return Lib_EIP155Tx.sender(
            _transaction
        );
    }
}
