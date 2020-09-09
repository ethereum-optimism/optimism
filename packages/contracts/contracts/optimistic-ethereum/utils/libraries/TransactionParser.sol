pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { BytesLib } from "./BytesLib.sol";
import { DataTypes } from "./DataTypes.sol";
import { RLPWriter } from "./RLPWriter.sol";

/**
 * @title TransactionParser
 */
library TransactionParser {
    /*
     * Internal Functions
     */

    /**
     * Utility; computes the hash of a given transaction.
     * @param _transaction OVM transaction to hash.
     * @return Hash of the provided transaction.
     */
    function getTransactionHash(
        DataTypes.OVMTransactionData memory _transaction
    )
        internal
        pure
        returns (bytes32)
    {
        bytes memory encodedTransaction = encodeTransactionData(_transaction);
        return keccak256(encodedTransaction);
    }

    /**
     * Utility; RLP encodes an OVMTransactionData struct.
     * @dev Likely to be changed (if not moved to another contract). Currently
     * remaining here as to avoid modifying CanonicalTransactionChain. Unclear
     * whether or not this is the correct transaction structure, but it should
     * work for the meantime.
     * @param _transactionData Transaction data to encode.
     * @return RLP encoded transaction data.
     */
    function encodeTransactionData(
        DataTypes.OVMTransactionData memory _transactionData
    )
        internal
        pure
        returns (bytes memory)
    {
        bytes[] memory raw = new bytes[](8);

        raw[0] = RLPWriter.encodeUint(_transactionData.timestamp);
        raw[1] = RLPWriter.encodeUint(_transactionData.queueOrigin);
        raw[2] = RLPWriter.encodeAddress(_transactionData.ovmEntrypoint);
        raw[3] = RLPWriter.encodeBytes(_transactionData.callBytes);
        raw[4] = RLPWriter.encodeAddress(_transactionData.fromAddress);
        raw[5] = RLPWriter.encodeAddress(_transactionData.l1MsgSenderAddress);
        raw[6] = RLPWriter.encodeUint(_transactionData.gasLimit);
        raw[7] = RLPWriter.encodeBool(_transactionData.allowRevert);

        return RLPWriter.encodeList(raw);
    }

    /**
     * Encodes an EOA transaction back into the original transaction.
     * @param _transaction EOA transaction to encode.
     * @param _isEthSignedMessage Whether or not this was an eth signed message.
     * @return Encoded transaction.
     */
    function encodeEOATransaction(
        DataTypes.EOATransaction memory _transaction,
        bool _isEthSignedMessage
    )
        internal
        pure
        returns (
            bytes memory
        )
    {
        if (_isEthSignedMessage) {
            return abi.encode(
                _transaction.nonce,
                _transaction.gasLimit,
                _transaction.gasPrice,
                _transaction.chainId,
                _transaction.to,
                _transaction.data
            );
        } else {
            bytes[] memory raw = new bytes[](9);

            raw[0] = RLPWriter.encodeUint(_transaction.nonce);
            raw[1] = RLPWriter.encodeUint(_transaction.gasPrice);
            raw[2] = RLPWriter.encodeUint(_transaction.gasLimit);
            raw[3] = RLPWriter.encodeAddress(_transaction.to);
            raw[4] = RLPWriter.encodeUint(0);
            raw[5] = RLPWriter.encodeBytes(_transaction.data);
            raw[6] = RLPWriter.encodeUint(_transaction.chainId);
            raw[7] = RLPWriter.encodeBytes(bytes(''));
            raw[8] = RLPWriter.encodeBytes(bytes(''));

            return RLPWriter.encodeList(raw);
        }
    }

    /**
     * Decodes and then re-encodes an EOA transaction.
     * @param _transaction Compactly encoded EOA transaction.
     * @param _isEthSignedMessage Whether or not this is an eth signed message.
     * @return Transaction with original encoding.
     */
    function encodeEOATransaction(
        bytes memory _transaction,
        bool _isEthSignedMessage
    )
        internal
        pure
        returns (
            bytes memory
        )
    {
        return encodeEOATransaction(
            decodeEOATransaction(_transaction),
            _isEthSignedMessage
        );
    }

    /**
     * Decodes a compactly encoded EOA transaction.
     * @param _transaction Compactly encoded transaction.
     * @return Transaction as a convenient struct.
     */
    function decodeEOATransaction(
        bytes memory _transaction
    )
        internal
        pure
        returns (
            DataTypes.EOATransaction memory
        )
    {
        uint256 nonce = BytesLib.toUintN(BytesLib.slice(_transaction, 0, 2));
        uint256 gasLimit = BytesLib.toUintN(BytesLib.slice(_transaction, 2, 3));
        uint256 gasPrice = BytesLib.toUintN(BytesLib.slice(_transaction, 5, 1));
        uint256 chainId = BytesLib.toUintN(BytesLib.slice(_transaction, 6, 4));
        address to = BytesLib.toAddress(BytesLib.slice(_transaction, 10, 20));
        bytes memory data = BytesLib.slice(_transaction, 30);

        return DataTypes.EOATransaction({
            nonce: nonce,
            gasLimit: gasLimit,
            gasPrice: gasPrice,
            chainId: chainId,
            to: to,
            data: data
        });
    }
}