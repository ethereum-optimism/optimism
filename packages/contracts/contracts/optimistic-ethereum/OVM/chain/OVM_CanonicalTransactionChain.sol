// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_MerkleUtils } from "../../../libraries/utils/Lib_MerkleUtils.sol";

/* Interface Imports */
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";
import { iOVM_BaseQueue } from "../../iOVM/queue/iOVM_BaseQueue.sol";

/* Contract Imports */
import { OVM_BaseChain } from "./OVM_BaseChain.sol";

contract OVM_CanonicalTransactionChain is iOVM_CanonicalTransactionChain, OVM_BaseChain {
    iOVM_BaseQueue internal ovmL1ToL2TransactionQueue;
    address internal sequencer;
    uint256 internal forceInclusionPeriodSeconds;
    uint256 internal lastOVMTimestamp;

    constructor(
        address _ovmL1ToL2TransactionQueue,
        address _sequencer,
        uint256 _forceInclusionPeriodSeconds,
    )
        Lib_ContractProxyResolver(_libContractProxyManager)
    {
        ovmL1ToL2TransactionQueue = iOVM_BaseQueue(_ovmL1ToL2TransactionQueue);
        sequencer = _sequencer;
        forceInclusionPeriodSeconds = _forceInclusionPeriodSeconds;
    }

    modifier onlySequencer() {
        require(
            msg.sender == sequencer,
            "Function can only be called by the sequencer."
        );
        _;
    }

    function appendQueueBatch()
        override
        public
    {
        require(
            ovmL1ToL2TransactionQueue.size() > 0 == true,
            "No batches are currently queued to be appended."
        );

        Lib_OVMCodec.QueueElement memory queueElement = ovmL1ToL2TransactionQueue.peek();
        
        require(
            queueElement.timestamp + forceInclusionPeriodSeconds <= block.timestamp,
            "Cannot append until the inclusion delay period has elapsed."
        );

        _appendQueueBatch(queueElement, 1);
        ovmL1ToL2TransactionQueue.dequeue();
    }

    function appendSequencerBatch(
        bytes[] memory _batch,
        uint256 _timestamp
    )
        override
        public
        onlySequencer
    {
        require(
            _timestamp >= lastOVMTimestamp,
            "Batch timestamp must be later than the last OVM timestamp."
        );

        if (ovmL1ToL2TransactionQueue.size() > 0) {
            require(
                _timestamp <= ovmL1ToL2TransactionQueue.peek().timestamp,
                "Older queue batches must be processed before a newer sequencer batch."
            );
        }

        Lib_OVMCodec.QueueElement memory queueElement = Lib_OVMCodec.QueueElement({
            timestamp: _timestamp,
            batchRoot: Lib_MerkleUtils.getMerkleRoot(_batch),
            isL1ToL2Batch: false
        });
        _appendQueueBatch(queueElement, _batch.length);
    }

    function _appendQueueBatch(
        Lib_OVMCodec.QueueElement memory _queueElement,
        uint256 _batchSize
    )
        internal
    {
        Lib_OVMCodec.ChainBatchHeader memory batchHeader = Lib_OVMCodec.ChainBatchHeader({
            batchIndex: getTotalBatches(),
            batchRoot: _queueElement.batchRoot,
            batchSize: _batchSize,
            prevTotalElements: getTotalElements(),
            extraData: abi.encodePacked(
                _queueElement.timestamp,
                _queueElement.isL1ToL2Batch
            )
        });

        _appendBatch(batchHeader);
        lastOVMTimestamp = _queueElement.timestamp;
    }
}
