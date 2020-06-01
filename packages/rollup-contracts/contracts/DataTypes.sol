pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title DataTypes
 * @notice Main data structures which to be used in rollup smart contracts.
 */
contract DataTypes {
    struct L2ToL1Message {
        address ovmSender;
        bytes callData;
    }

    struct Transaction {
        address ovmEntrypoint;
        bytes ovmCalldata;
    }

    struct StorageElement {
        address ovmContractAddress;
        bytes32 ovmStorageSlot;
        bytes32 ovmStorageValue;
    }

    struct ExecutionContext {
        bool inStaticContext;
        uint chainId;
        uint timestamp;
        uint queueOrigin;
        uint gasLimit;
        address ovmActiveContract;
        address ovmMsgSender;
        address ovmTxOrigin;
        address l1MessageSender;
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
