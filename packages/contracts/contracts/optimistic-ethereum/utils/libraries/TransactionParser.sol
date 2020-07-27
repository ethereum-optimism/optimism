pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
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
        bytes[] memory raw = new bytes[](7);

        raw[0] = RLPWriter.encodeUint(_transactionData.timestamp);
        raw[1] = RLPWriter.encodeUint(_transactionData.queueOrigin);
        raw[2] = RLPWriter.encodeAddress(_transactionData.ovmEntrypoint);
        raw[3] = RLPWriter.encodeBytes(_transactionData.callBytes);
        raw[4] = RLPWriter.encodeAddress(_transactionData.fromAddress);
        raw[5] = RLPWriter.encodeAddress(_transactionData.l1MsgSenderAddress);
        raw[6] = RLPWriter.encodeBool(_transactionData.allowRevert);

        return RLPWriter.encodeList(raw);
    }
}