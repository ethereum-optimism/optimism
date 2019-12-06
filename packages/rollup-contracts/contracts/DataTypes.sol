pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title DataTypes
 * @notice TODO
 */
contract DataTypes {
    struct ElementInclusionProof {
       uint blockIndex; // index in blocks array (first block has blockNumber of 0)
       BlockHeader blockHeader;
       uint indexInBlock; // used to verify inclusion of the element in elementsMerkleRoot
       bytes32[] siblings; // used to verify inclusion of the element in elementsMerkleRoot
    }

    struct BlockHeader {
       uint ethBlockNumber;
       bytes32 elementsMerkleRoot;
       uint numElementsInBlock;
       uint cumulativePrevElements;
    }
}
