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
            raw[6] = RLPWriter.encodeUint(108);
            raw[7] = RLPWriter.encodeBytes(bytes(''));
            raw[8] = RLPWriter.encodeBytes(bytes(''));

            return RLPWriter.encodeList(raw);
        }
    }

    function decodeEOATransaction(
        bytes memory _transaction
    )
        internal
        pure
        returns (
            DataTypes.EOATransaction memory
        )
    {
        bytes32 nonce = BytesLib.toBytes32(BytesLib.slice(_transaction, 0, 2)) >> 30 * 8;
        bytes32 gasLimit = BytesLib.toBytes32(BytesLib.slice(_transaction, 2, 3)) >> 29 * 8;
        bytes32 gasPrice = BytesLib.toBytes32(BytesLib.slice(_transaction, 5, 1)) >> 31 * 8;
        address to = BytesLib.toAddress(BytesLib.slice(_transaction, 6, 20));
        bytes memory data = BytesLib.slice(_transaction, 26);

        return DataTypes.EOATransaction({
            nonce: uint256(nonce),
            gasLimit: uint256(gasLimit),
            gasPrice: uint256(gasPrice),
            to: to,
            data: data
        });
    }
}