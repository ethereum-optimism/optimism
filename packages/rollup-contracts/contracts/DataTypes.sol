pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title DataTypes
 * @notice TODO
 */
contract DataTypes {
    struct L2ToL1Message {
        address ovmSender;
        bytes callData;
    }

    struct ElementInclusionProof {
       uint batchIndex; // index in batches array (first batch has batchNumber of 0)
       TxChainBatchHeader batchHeader;
       uint indexInBatch; // used to verify inclusion of the element in elementsMerkleRoot
       bytes32[] siblings; // used to verify inclusion of the element in elementsMerkleRoot
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
