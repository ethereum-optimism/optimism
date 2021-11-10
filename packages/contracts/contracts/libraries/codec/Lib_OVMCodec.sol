// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { Lib_RLPReader } from "../rlp/Lib_RLPReader.sol";
import { Lib_RLPWriter } from "../rlp/Lib_RLPWriter.sol";
import { Lib_BytesUtils } from "../utils/Lib_BytesUtils.sol";
import { Lib_Bytes32Utils } from "../utils/Lib_Bytes32Utils.sol";

/**
 * @title Lib_OVMCodec
 */
library Lib_OVMCodec {
    /*********
     * Enums *
     *********/

    enum QueueOrigin {
        SEQUENCER_QUEUE,
        L1TOL2_QUEUE
    }

    /***********
     * Structs *
     ***********/

    struct EVMAccount {
        uint256 nonce;
        uint256 balance;
        bytes32 storageRoot;
        bytes32 codeHash;
    }

    struct ChainBatchHeader {
        uint256 batchIndex;
        bytes32 batchRoot;
        uint256 batchSize;
        uint256 prevTotalElements;
        bytes extraData;
    }

    struct ChainInclusionProof {
        uint256 index;
        bytes32[] siblings;
    }

    struct Transaction {
        uint256 timestamp;
        uint256 blockNumber;
        QueueOrigin l1QueueOrigin;
        address l1TxOrigin;
        address entrypoint;
        uint256 gasLimit;
        bytes data;
    }

    struct TransactionChainElement {
        bool isSequenced;
        uint256 queueIndex; // QUEUED TX ONLY
        uint256 timestamp; // SEQUENCER TX ONLY
        uint256 blockNumber; // SEQUENCER TX ONLY
        bytes txData; // SEQUENCER TX ONLY
    }

    struct QueueElement {
        bytes32 transactionHash;
        uint40 timestamp;
        uint40 blockNumber;
    }

    /**********************
     * Internal Functions *
     **********************/

    /**
     * Encodes a standard OVM transaction.
     * @param _transaction OVM transaction to encode.
     * @return Encoded transaction bytes.
     */
    function encodeTransaction(Transaction memory _transaction)
        internal
        pure
        returns (bytes memory)
    {
        return
            abi.encodePacked(
                _transaction.timestamp,
                _transaction.blockNumber,
                _transaction.l1QueueOrigin,
                _transaction.l1TxOrigin,
                _transaction.entrypoint,
                _transaction.gasLimit,
                _transaction.data
            );
    }

    /**
     * Hashes a standard OVM transaction.
     * @param _transaction OVM transaction to encode.
     * @return Hashed transaction
     */
    function hashTransaction(Transaction memory _transaction) internal pure returns (bytes32) {
        return keccak256(encodeTransaction(_transaction));
    }

    /**
     * @notice Decodes an RLP-encoded account state into a useful struct.
     * @param _encoded RLP-encoded account state.
     * @return Account state struct.
     */
    function decodeEVMAccount(bytes memory _encoded) internal pure returns (EVMAccount memory) {
        Lib_RLPReader.RLPItem[] memory accountState = Lib_RLPReader.readList(_encoded);

        return
            EVMAccount({
                nonce: Lib_RLPReader.readUint256(accountState[0]),
                balance: Lib_RLPReader.readUint256(accountState[1]),
                storageRoot: Lib_RLPReader.readBytes32(accountState[2]),
                codeHash: Lib_RLPReader.readBytes32(accountState[3])
            });
    }

    /**
     * Calculates a hash for a given batch header.
     * @param _batchHeader Header to hash.
     * @return Hash of the header.
     */
    function hashBatchHeader(Lib_OVMCodec.ChainBatchHeader memory _batchHeader)
        internal
        pure
        returns (bytes32)
    {
        return
            keccak256(
                abi.encode(
                    _batchHeader.batchRoot,
                    _batchHeader.batchSize,
                    _batchHeader.prevTotalElements,
                    _batchHeader.extraData
                )
            );
    }
}
