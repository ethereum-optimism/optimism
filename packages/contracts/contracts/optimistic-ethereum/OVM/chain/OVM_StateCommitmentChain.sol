// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";
import { Lib_MerkleUtils } from "../../libraries/utils/Lib_MerkleUtils.sol";
import { Lib_RingBuffer, iRingBufferOverwriter } from "../../libraries/utils/Lib_RingBuffer.sol";

/* Interface Imports */
import { iOVM_FraudVerifier } from "../../iOVM/verification/iOVM_FraudVerifier.sol";
import { iOVM_StateCommitmentChain } from "../../iOVM/chain/iOVM_StateCommitmentChain.sol";
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";
import { iOVM_BondManager } from "../../iOVM/verification/iOVM_BondManager.sol";


/**
 * @title OVM_StateCommitmentChain
 */
contract OVM_StateCommitmentChain is iOVM_StateCommitmentChain, iRingBufferOverwriter, Lib_AddressResolver {
    using Lib_RingBuffer for Lib_RingBuffer.RingBuffer;


    /*************
     * Constants *
     *************/

    uint256 constant public FRAUD_PROOF_WINDOW = 7 days;


    /*************
     * Variables *
     *************/

    uint256 internal lastDeletableIndex;
    uint256 internal lastDeletableTimestamp;
    Lib_RingBuffer.RingBuffer internal batches;
    iOVM_CanonicalTransactionChain internal ovmCanonicalTransactionChain;
    iOVM_FraudVerifier internal ovmFraudVerifier;
    iOVM_BondManager internal ovmBondManager;

    /***************
     * Constructor *
     ***************/

    /**
     * @param _libAddressManager Address of the Address Manager.
     */
    constructor(
        address _libAddressManager
    )
        Lib_AddressResolver(_libAddressManager)
    {
        ovmCanonicalTransactionChain = iOVM_CanonicalTransactionChain(resolve("OVM_CanonicalTransactionChain"));
        ovmFraudVerifier = iOVM_FraudVerifier(resolve("OVM_FraudVerifier"));
        ovmBondManager = iOVM_BondManager(resolve("OVM_BondManager"));

        batches.init(
            16,
            Lib_OVMCodec.RING_BUFFER_SCC_BATCHES,
            iRingBufferOverwriter(address(this))
        );
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * @inheritdoc iOVM_StateCommitmentChain
     */
    function getTotalElements()
        override
        public
        view
        returns (
            uint256 _totalElements
        )
    {
        return uint256(uint216(batches.getExtraData()));
    }

    /**
     * @inheritdoc iOVM_StateCommitmentChain
     */
    function getTotalBatches()
        override
        public
        view
        returns (
            uint256 _totalBatches
        )
    {
        return uint256(batches.getLength());
    }

    /**
     * @inheritdoc iOVM_StateCommitmentChain
     */
    function appendStateBatch(
        bytes32[] memory _batch
    )
        override
        public
    {
        // Proposers must have previously staked at the BondManager
        require(
            ovmBondManager.isCollateralized(msg.sender),
            "Proposer does not have enough collateral posted"
        );

        require(
            _batch.length > 0,
            "Cannot submit an empty state batch."
        );

        require(
            getTotalElements() + _batch.length <= ovmCanonicalTransactionChain.getTotalElements(),
            "Number of state roots cannot exceed the number of canonical transactions."
        );

        bytes[] memory elements = new bytes[](_batch.length);
        for (uint256 i = 0; i < _batch.length; i++) {
            elements[i] = abi.encodePacked(_batch[i]);
        }

        // Pass the block's timestamp and the publisher of the data
        // to be used in the fraud proofs
        _appendBatch(
            elements,
            abi.encode(block.timestamp, msg.sender)
        );
    }

    /**
     * @inheritdoc iOVM_StateCommitmentChain
     */
    function deleteStateBatch(
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader
    )
        override
        public
    {
        require(
            msg.sender == address(ovmFraudVerifier),
            "State batches can only be deleted by the OVM_FraudVerifier."
        );

        require(
            Lib_OVMCodec.hashBatchHeader(_batchHeader) == batches.get(uint32(_batchHeader.batchIndex)),
            "Invalid batch header."
        );

        require(
            insideFraudProofWindow(_batchHeader),
            "State batches can only be deleted within the fraud proof window."
        );

        _deleteBatch(_batchHeader);
    }

    /**
     * @inheritdoc iOVM_StateCommitmentChain
     */
    function verifyStateCommitment(
        bytes32 _element,
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _proof
    )
        override
        public
        view
        returns (
            bool
        )
    {
        require(
            Lib_OVMCodec.hashBatchHeader(_batchHeader) == batches.get(uint32(_batchHeader.batchIndex)),
            "Invalid batch header."
        );

        require(
            Lib_MerkleUtils.verify(
                _batchHeader.batchRoot,
                _element,
                _proof.index,
                _proof.siblings
            ),
            "Invalid inclusion proof."
        );

        return true;
    }

    /**
     * @inheritdoc iOVM_StateCommitmentChain
     */
    function insideFraudProofWindow(
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader
    )
        override
        public
        view
        returns (
            bool _inside
        )
    {
        (uint256 timestamp,) = abi.decode(
            _batchHeader.extraData,
            (uint256, address)
        );

        require(
            timestamp != 0,
            "Batch header timestamp cannot be zero"
        );

        return timestamp + FRAUD_PROOF_WINDOW > block.timestamp;
    }

    /**
     * @inheritdoc iOVM_StateCommitmentChain
     */
    function setLastDeletableIndex(
        Lib_OVMCodec.ChainBatchHeader memory _stateBatchHeader,
        Lib_OVMCodec.Transaction memory _transaction,
        Lib_OVMCodec.TransactionChainElement memory _txChainElement,
        Lib_OVMCodec.ChainBatchHeader memory _txBatchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _txInclusionProof
    )
        override
        public
    {
        require(
            Lib_OVMCodec.hashBatchHeader(_stateBatchHeader) == batches.get(uint32(_stateBatchHeader.batchIndex)),
            "Invalid batch header."
        );

        require(
            insideFraudProofWindow(_stateBatchHeader) == false,
            "Batch header must be outside of fraud proof window to be deletable."
        );

        require(
            _stateBatchHeader.batchIndex > lastDeletableIndex,
            "Batch index must be greater than last deletable index."
        );

        require(
            ovmCanonicalTransactionChain.verifyTransaction(
                _transaction,
                _txChainElement,
                _txBatchHeader,
                _txInclusionProof
            ),
            "Invalid transaction proof."
        );

        lastDeletableIndex = _stateBatchHeader.batchIndex;
        lastDeletableTimestamp = _transaction.timestamp;
    }

    /**
     * @inheritdoc iRingBufferOverwriter
     */
    function canOverwrite(
        bytes32 _id,
        uint256 _index
    )
        override
        public
        view
        returns (
            bool
        )
    {
        if (_id == Lib_OVMCodec.RING_BUFFER_CTC_QUEUE) {
            return ovmCanonicalTransactionChain.getQueueElement(_index / 2).timestamp < lastDeletableTimestamp;
        } else {
            return _index < lastDeletableIndex;
        }
    }


    /**********************
     * Internal Functions *
     **********************/

    /**
     * Appends a batch to the chain.
     * @param _batchHeader Batch header to append.
     */
    function _appendBatch(
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader
    )
        internal
    {
        batches.push(
            Lib_OVMCodec.hashBatchHeader(_batchHeader),
            bytes27(uint216(getTotalElements() + _batchHeader.batchSize))
        );
    }

    /**
     * Appends a batch to the chain.
     * @param _elements Elements within the batch.
     * @param _extraData Any extra data to append to the batch.
     */
    function _appendBatch(
        bytes[] memory _elements,
        bytes memory _extraData
    )
        internal
    {
        Lib_OVMCodec.ChainBatchHeader memory batchHeader = Lib_OVMCodec.ChainBatchHeader({
            batchIndex: uint256(batches.getLength()),
            batchRoot: Lib_MerkleUtils.getMerkleRoot(_elements),
            batchSize: _elements.length,
            prevTotalElements: getTotalElements(),
            extraData: _extraData
        });

        _appendBatch(batchHeader);
    }

    /**
     * Removes a batch from the chain.
     * @param _batchHeader Header of the batch to remove.
     */
    function _deleteBatch(
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader
    )
        internal
    {
        require(
            _batchHeader.batchIndex < batches.getLength(),
            "Invalid batch index."
        );

        require(
            Lib_OVMCodec.hashBatchHeader(_batchHeader) == batches.get(uint32(_batchHeader.batchIndex)),
            "Invalid batch header."
        );

        batches.deleteElementsAfterInclusive(
            uint40(_batchHeader.batchIndex),
            bytes27(uint216(_batchHeader.prevTotalElements))
        );
    }
}
