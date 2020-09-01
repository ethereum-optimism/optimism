pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { CanonicalTransactionChain } from "./CanonicalTransactionChain.sol";
import { FraudVerifier } from "../ovm/FraudVerifier.sol";

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { DataTypes } from "../utils/libraries/DataTypes.sol";
import { RollupMerkleUtils } from "../utils/libraries/RollupMerkleUtils.sol";

/**
 * @title StateCommitmentChain
 */
contract StateCommitmentChain is ContractResolver {
    /*
     * Events
     */

    event StateBatchAppended(bytes32 _batchHeaderHash);


    /*
    * Contract Variables
    */

    uint public cumulativeNumElements;
    bytes32[] public batches;


    /*
    * Constructor
    */

    /**
     * @param _addressResolver Address of the AddressResolver contract.
     */
    constructor(
        address _addressResolver
    )
        public
        ContractResolver(_addressResolver)
    {}


    /*
    * Public Functions
    */

    /**
     * @return Total number of published state batches.
     */
    function getBatchesLength()
        public
        view
        returns (uint)
    {
        return batches.length;
    }

    /**
     * Computes the hash of a batch header.
     * @param _batchHeader Header to hash.
     * @return Hash of the provided header.
     */
    function hashBatchHeader(
        DataTypes.StateChainBatchHeader memory _batchHeader
    )
        public
        pure
        returns (bytes32)
    {
        return keccak256(abi.encodePacked(
            _batchHeader.elementsMerkleRoot,
            _batchHeader.numElementsInBatch,
            _batchHeader.cumulativePrevElements
        ));
    }

    /**
     * Attempts to append a state batch.
     * @param _stateBatch Batch of ordered root hashes to append.
     * @param _startsAtRootIndex The absolute index in the state root chain of the first state root in this batch.
     */
    function appendStateBatch(
        bytes32[] memory _stateBatch,
        uint _startsAtRootIndex
    )
        public
    {
        CanonicalTransactionChain canonicalTransactionChain = resolveCanonicalTransactionChain();
        RollupMerkleUtils merkleUtils = resolveRollupMerkleUtils();

        require(
            cumulativeNumElements + _stateBatch.length <= canonicalTransactionChain.cumulativeNumElements(),
            "Cannot append more state commitments than total number of transactions in CanonicalTransactionChain"
        );

        require(
            _stateBatch.length > 0,
            "Cannot submit an empty state commitment batch"
        );

        if (_startsAtRootIndex + _stateBatch.length >= cumulativeNumElements) {
            // This means all the roots in this batch were already appended. Don't fail, but don't change state.
            return;
        }

        bytes32[] memory batchToAppend;
        if (_startsAtRootIndex < cumulativeNumElements) {
            uint elementsToSkip = cumulativeNumElements - _startsAtRootIndex;
            batchToAppend = new bytes32[](_stateBatch.length - elementsToSkip);
            for (uint i = 0; i < batchToAppend.length; i++) {
                batchToAppend[i] = _stateBatch[elementsToSkip + i];
            }
        } else {
            batchToAppend = _stateBatch;
        }

        bytes32 batchHeaderHash = keccak256(abi.encodePacked(
            merkleUtils.getMerkleRootFrom32ByteLeafData(batchToAppend), // elementsMerkleRoot
            batchToAppend.length, // numElementsInBatch
            cumulativeNumElements // cumulativeNumElements
        ));

        batches.push(batchHeaderHash);
        cumulativeNumElements += batchToAppend.length;
        emit StateBatchAppended(batchHeaderHash);
    }

    /**
     * Checks that an element is included within a published batch.
     * @param _element Element to prove within the batch.
     * @param _position Index of the element within the batch.
     * @param _inclusionProof Inclusion proof for the element/batch.
     */
    function verifyElement(
        bytes memory _element,
        uint _position,
        DataTypes.StateElementInclusionProof memory _inclusionProof
    )
        public
        view
        returns (bool)
    {
        DataTypes.StateChainBatchHeader memory batchHeader = _inclusionProof.batchHeader;
        if (_position != _inclusionProof.indexInBatch +
            batchHeader.cumulativePrevElements) {
            return false;
        }

        RollupMerkleUtils merkleUtils = resolveRollupMerkleUtils();
        if (!merkleUtils.verify(
            batchHeader.elementsMerkleRoot,
            _element,
            _inclusionProof.indexInBatch,
            _inclusionProof.siblings
        )) {
            return false;
        }

        //compare computed batch header with the batch header in the list.
        return hashBatchHeader(batchHeader) == batches[_inclusionProof.batchIndex];
    }

    /**
     * Deletes all state batches after and including the given batch index.
     * Can only be called by the FraudVerifier contract.
     * @param _batchIndex Index of the batch to start deletion from.
     * @param _batchHeader Header of batch at the given index.
     */
    function deleteAfterInclusive(
        uint _batchIndex,
        DataTypes.StateChainBatchHeader memory _batchHeader
    )
        public
    {
        FraudVerifier fraudVerifier = resolveFraudVerifier();

        require(
            msg.sender == address(fraudVerifier),
            "Only FraudVerifier has permission to delete state batches"
        );

        require(
            _batchIndex < batches.length,
            "Cannot delete batches outside of valid range"
        );

        bytes32 calculatedBatchHeaderHash = hashBatchHeader(_batchHeader);
        require(
            calculatedBatchHeaderHash == batches[_batchIndex],
            "Calculated batch header is different than expected batch header"
        );

        batches.length = _batchIndex;
        cumulativeNumElements = _batchHeader.cumulativePrevElements;
    }


    /*
     * Contract Resolution
     */

    function resolveCanonicalTransactionChain()
        internal
        view
        returns (CanonicalTransactionChain)
    {
        return CanonicalTransactionChain(resolveContract("CanonicalTransactionChain"));
    }

    function resolveFraudVerifier()
        internal
        view
        returns (FraudVerifier)
    {
        return FraudVerifier(resolveContract("FraudVerifier"));
    }

    function resolveRollupMerkleUtils()
        internal
        view
        returns (RollupMerkleUtils)
    {
        return RollupMerkleUtils(resolveContract("RollupMerkleUtils"));
    }
}
