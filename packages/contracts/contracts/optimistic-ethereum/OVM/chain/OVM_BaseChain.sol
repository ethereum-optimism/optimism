// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_BaseChain } from "../../iOVM/chain/iOVM_BaseChain.sol";

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_MerkleUtils } from "../../libraries/utils/Lib_MerkleUtils.sol";

/**
 * @title OVM_BaseChain
 */
contract OVM_BaseChain is iOVM_BaseChain {

    /*******************************
     * Contract Variables: Batches *
     *******************************/

    bytes32[] internal batches;
    uint256 internal totalBatches;
    uint256 internal totalElements;


    /*************************************
     * Public Functions: Batch Retrieval *
     *************************************/

    /**
     * Gets the total number of submitted elements.
     * @return _totalElements Total submitted elements.
     */
    function getTotalElements()
        override
        public
        view
        returns (
            uint256 _totalElements
        )
    {
        return totalElements;
    }

    /**
     * Gets the total number of submitted batches.
     * @return _totalBatches Total submitted batches.
     */
    function getTotalBatches()
        override
        public
        view
        returns (
            uint256 _totalBatches
        )
    {
        return totalBatches;
    }


    /****************************************
     * Public Functions: Batch Verification *
     ****************************************/

    /**
     * Verifies an inclusion proof for a given element.
     * @param _element Element to verify.
     * @param _batchHeader Header of the batch in which this element was included.
     * @param _proof Inclusion proof for the element.
     * @return _verified Whether or not the element was included in the batch.
     */
    function verifyElement(
        bytes calldata _element,
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _proof
    )
        override
        public
        view
        returns (
            bool _verified
        )
    {
        require(
            _hashBatchHeader(_batchHeader) == batches[_batchHeader.batchIndex],
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


    /******************************************
     * Internal Functions: Batch Modification *
     ******************************************/

    /**
     * Appends a batch to the chain.
     * @param _batchHeader Batch header to append.
     */
    function _appendBatch(
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader
    )
        internal
    {
        bytes32 batchHeaderHash = _hashBatchHeader(_batchHeader);
        batches.push(batchHeaderHash);
        totalBatches += 1;
        totalElements += _batchHeader.batchSize;
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
            batchIndex: batches.length,
            batchRoot: Lib_MerkleUtils.getMerkleRoot(_elements),
            batchSize: _elements.length,
            prevTotalElements: totalElements,
            timestamp: block.timestamp,
            extraData: _extraData
        });

        _appendBatch(batchHeader);
    }

    /**
     * Appends a batch to the chain.
     * @param _elements Elements within the batch.
     */
    function _appendBatch(
        bytes[] memory _elements
    )
        internal
    {
        _appendBatch(
            _elements,
            bytes('')
        );
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
            _batchHeader.batchIndex < batches.length,
            "Invalid batch index."
        );

        require(
            _hashBatchHeader(_batchHeader) == batches[_batchHeader.batchIndex],
            "Invalid batch header."
        );

        totalBatches = _batchHeader.batchIndex;
        totalElements = _batchHeader.prevTotalElements;
    }


    /*********************
     * Private Functions *
     *********************/

    /**
     * Calculates a hash for a given batch header.
     * @param _batchHeader Header to hash.
     * @return _hash Hash of the header.
     */
    function _hashBatchHeader(
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader
    )
        private
        pure
        returns (
            bytes32 _hash
        )
    {
        return keccak256(abi.encodePacked(
            _batchHeader.batchRoot,
            _batchHeader.batchSize,
            _batchHeader.prevTotalElements,
            _batchHeader.timestamp,
            _batchHeader.extraData
        ));
    }
}
