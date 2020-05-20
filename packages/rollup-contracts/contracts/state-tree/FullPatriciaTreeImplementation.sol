pragma solidity >=0.5.0 <0.6.0;
pragma experimental ABIEncoderV2;

import {D} from "./DataTypes.sol";
import {FullPatriciaTree} from "./FullPatriciaTree.sol";

contract FullPatriciaTreeImplementation {
    using FullPatriciaTree for FullPatriciaTree.Tree;
    FullPatriciaTree.Tree tree;

    constructor () public {
    }

    function insert(bytes32 key, bytes memory value) public {
        tree.insert(key, value);
    }

    function get(bytes32 key) public view returns (bytes memory) {
        return tree.get(key);
    }

    function safeGet(bytes32 key) public view returns (bytes memory) {
        return tree.safeGet(key);
    }

    function doesInclude(bytes memory key) public view returns (bool) {
        return tree.doesInclude(key);
    }

    function getValue(bytes32 hash) public view returns (bytes memory) {
        return tree.values[hash];
    }

    function getRootHash() public view returns (bytes32) {
        return tree.getRootHash();
    }

    function getNode(bytes32 hash) public view returns (uint, bytes32, bytes32, uint, bytes32, bytes32) {
        return tree.getNode(hash);
    }

    function getRootEdge() public view returns (uint, bytes32, bytes32) {
        return tree.getRootEdge();
    }

    function getProof(bytes32 key) public view returns (uint branchMask, bytes32[] memory _siblings) {
        return tree.getProof(key);
    }

    // todo naming -- these arent always leaves
    function getNonInclusionProof(bytes32 key) public view returns (
        D.Label memory potentialSiblingCumulativeLabel,
        bytes32 potentialSiblingValue,
        uint branchMask,
        bytes32[] memory _siblings
    ) {
        return tree.getNonInclusionProof(key);
    }

    function verifyProof(bytes32 rootHash, bytes32 key, bytes memory value, uint branchMask, bytes32[] memory siblings) public pure {
        FullPatriciaTree.verifyProof(rootHash, key, value, branchMask, siblings);
    }

    function verifyNonInclusionProof(
        bytes32 rootHash,
        bytes32 key,
        bytes32 conflictingEdgeFullLabelData,
        uint conflictingEdgeFullLabelLength,
        bytes32 conflictingEdgeCommitment,
        uint branchMask,
        bytes32[] memory siblings
    ) public pure {
        FullPatriciaTree.verifyNonInclusionProof(rootHash, key, conflictingEdgeFullLabelData, conflictingEdgeFullLabelLength, conflictingEdgeCommitment, branchMask, siblings);
    }

}