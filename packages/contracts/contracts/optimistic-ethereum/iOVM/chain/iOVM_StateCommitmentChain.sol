// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

/**
 * @title iOVM_StateCommitmentChain
 */
interface iOVM_StateCommitmentChain {

    /********************
     * Public Functions *
     ********************/

    /**
     * Retrieves the total number of elements submitted.
     * @return _totalElements Total submitted elements.
     */
    function getTotalElements()
        external
        view
        returns (
            uint256 _totalElements
        );

    /**
     * Retrieves the total number of batches submitted.
     * @return _totalBatches Total submitted batches.
     */
    function getTotalBatches()
        external
        view
        returns (
            uint256 _totalBatches
        );

    /**
     * Appends a batch of state roots to the chain.
     * @param _batch Batch of state roots.
     */
    function appendStateBatch(
        bytes32[] calldata _batch
    )
        external;

    /**
     * Deletes all state roots after (and including) a given batch.
     * @param _batchHeader Header of the batch to start deleting from.
     */
    function deleteStateBatch(
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader
    )
        external;

    /**
     * Verifies a batch inclusion proof.
     * @param _element Hash of the element to verify a proof for.
     * @param _batchHeader Header of the batch in which the element was included.
     * @param _proof Merkle inclusion proof for the element.
     */
    function verifyStateCommitment(
        bytes32 _element,
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _proof
    )
        external
        view
        returns (
            bool _verified
        );

    /**
     * Checks whether a given batch is still inside its fraud proof window.
     * @param _batchHeader Header of the batch to check.
     * @return _inside Whether or not the batch is inside the fraud proof window.
     */
    function insideFraudProofWindow(
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader
    )
        external
        view
        returns (
            bool _inside
        );
    
    /**
     * Sets the last batch index that can be deleted.
     * @param _stateBatchHeader Proposed batch header that can be deleted.
     * @param _transaction Transaction to verify.
     * @param _txChainElement Transaction chain element corresponding to the transaction.
     * @param _txBatchHeader Header of the batch the transaction was included in.
     * @param _txInclusionProof Inclusion proof for the provided transaction chain element.
     */
    function setLastDeletableIndex(
        Lib_OVMCodec.ChainBatchHeader memory _stateBatchHeader,
        Lib_OVMCodec.Transaction memory _transaction,
        Lib_OVMCodec.TransactionChainElement memory _txChainElement,
        Lib_OVMCodec.ChainBatchHeader memory _txBatchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _txInclusionProof
    )
        external;
}
