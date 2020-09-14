// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

interface iOVM_DataTypes {
    struct OVMAccount {
        uint256 nonce;
        uint256 balance;
        bytes32 storageRoot;
        bytes32 codeHash;
        address ethAddress;
    }

    struct EVMAccount {
        uint256 nonce;
        uint256 balance;
        bytes32 storageRoot;
        bytes32 codeHash;
    }

    struct OVMChainBatchHeader {
        uint256 batchIndex;
        bytes32 batchRoot;
        uint256 batchSize;
        uint256 prevTotalElements;
        bytes extraData;
    }

    struct OVMChainInclusionProof {
        uint256 index;
        bytes32[] siblings;
    }

    struct OVMTransactionData {
        uint256 timestamp;
        uint256 queueOrigin;
        address entrypoint;
        address origin;
        address msgSender;
        uint256 gasLimit;
        bytes data;
    }

    struct OVMProofMatrix {
        bool checkNonce;
        bool checkBalance;
        bool checkStorageRoot;
        bool checkCodeHash;
    }

    struct OVMQueueElement {
        uint256 timestamp;
        bytes32 batchRoot;
        bool isL1ToL2Batch;
    }
}
