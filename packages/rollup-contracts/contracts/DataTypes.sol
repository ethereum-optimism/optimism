pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title DataTypes
 */
contract DataTypes {
    struct L2ToL1Message {
        address ovmSender;
        bytes callData;
    }

    struct TxElementInclusionProof {
       uint batchIndex;
       TxChainBatchHeader batchHeader;
       uint indexInBatch;
       bytes32[] siblings;
    }

    struct StateElementInclusionProof {
       uint batchIndex;
       StateChainBatchHeader batchHeader;
       uint indexInBatch;
       bytes32[] siblings;
    }

    struct StateChainBatchHeader {
       bytes32 elementsMerkleRoot;
       uint numElementsInBatch;
       uint cumulativePrevElements;
    }

    struct TxChainBatchHeader {
       uint timestamp;
       bool isL1ToL2Tx;
       bytes32 elementsMerkleRoot;
       uint numElementsInBatch;
       uint cumulativePrevElements;
    }

   struct TimestampedHash {
       uint timestamp;
       bytes32 txHash;
    }
}
