pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { DataTypes } from "../utils/libraries/DataTypes.sol";
import { RollupMerkleUtils } from "../utils/libraries/RollupMerkleUtils.sol";
import { CanonicalTransactionChain } from "./CanonicalTransactionChain.sol";
import { FraudVerifier } from "../ovm/FraudVerifier.sol";

contract StateCommitmentChain is ContractResolver {
    /*
    * Contract Variables
    */

    uint public cumulativeNumElements;
    bytes32[] public batches;

    /*
     * Events
     */

    event StateBatchAppended(bytes32 _batchHeaderHash);

    /*
    * Constructor
    */

    constructor(address _addressResolver) public ContractResolver(_addressResolver) {}

    /*
    * Public Functions
    */

    function getBatchesLength() public view returns (uint) {
        return batches.length;
    }

    function hashBatchHeader(
        DataTypes.StateChainBatchHeader memory _batchHeader
    ) public pure returns (bytes32) {
        return keccak256(abi.encodePacked(
            _batchHeader.elementsMerkleRoot,
            _batchHeader.numElementsInBatch,
            _batchHeader.cumulativePrevElements
        ));
    }

    function appendStateBatch(
        bytes[] memory _stateBatch
    ) public {
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

        bytes32 batchHeaderHash = keccak256(abi.encodePacked(
            merkleUtils.getMerkleRoot(_stateBatch), // elementsMerkleRoot
            _stateBatch.length, // numElementsInBatch
            cumulativeNumElements // cumulativeNumElements
        ));

        batches.push(batchHeaderHash);
        cumulativeNumElements += _stateBatch.length;
        emit StateBatchAppended(batchHeaderHash);
    }

    // verifies an element is in the current list at the given position
    function verifyElement(
        bytes memory _element, // the element of the list being proven
        uint _position, // the position in the list of the element being proven
        DataTypes.StateElementInclusionProof memory _inclusionProof
    ) public view returns (bool) {
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

    function deleteAfterInclusive(
        uint _batchIndex,
        DataTypes.StateChainBatchHeader memory _batchHeader
    ) public {
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

    function resolveCanonicalTransactionChain() internal view returns (CanonicalTransactionChain) {
        return CanonicalTransactionChain(resolveContract("CanonicalTransactionChain"));
    }

    function resolveFraudVerifier() internal view returns (FraudVerifier) {
        return FraudVerifier(resolveContract("FraudVerifier"));
    }

    function resolveRollupMerkleUtils() internal view returns (RollupMerkleUtils) {
        return RollupMerkleUtils(resolveContract("RollupMerkleUtils"));
    }
}
