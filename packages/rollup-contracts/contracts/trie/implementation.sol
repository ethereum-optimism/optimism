pragma solidity >=0.5.0 <0.6.0;

import {D} from "./data.sol";
import {PatriciaTree} from "./tree.sol";

contract PatriciaTreeImplementation {
    using PatriciaTree for PatriciaTree.Tree;
    PatriciaTree.Tree tree;

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
        bytes32 leafLabel,
        bytes32 leafNode,
        uint branchMask,
        bytes32[] memory _siblings,
        uint leafLength
    ) {
        return tree.getNonInclusionProof(key);
    }

    function verifyProof(bytes32 rootHash, bytes32 key, bytes memory value, uint branchMask, bytes32[] memory siblings) public pure {
        PatriciaTree.verifyProof(rootHash, key, value, branchMask, siblings);
    }

    function verifyNonInclusionProof(bytes32 rootHash, bytes32 key, bytes32 leafLabel, bytes32 leafNode, uint branchMask, bytes32[] memory siblings) public pure {
        PatriciaTree.verifyNonInclusionProof(rootHash, key, leafLabel, leafNode, branchMask, siblings);
    }

    function verifyNonInclusionProof2(
        bytes32 rootHash,
        bytes32 key,
        bytes32 conflictingEdgeFullLabelData,
        uint conflictingEdgeFullLabelLength,
        bytes32 conflictingEdgeCommitment,
        uint branchMask,
        bytes32[] memory siblings
    ) public pure {
        PatriciaTree.verifyNonInclusionProof2(rootHash, key, conflictingEdgeFullLabelData, conflictingEdgeFullLabelLength, conflictingEdgeCommitment, branchMask, siblings);
    }
    // temp

    function getHash(bytes memory pre) public pure returns(bytes32) {return keccak256(pre);}

}

// contract PatriciaTreeMerkleProof {
//     using PatriciaTree for PatriciaTree.Tree;
//     PatriciaTree.Tree tree;

//     enum Status {OPENED, ONGOING, SUCCESS, FAILURE}

//     event OnChangeStatus(Status s);

//     modifier onlyFor(Status _status) {
//         require(status == _status);
//         _;
//     }

//     mapping(bytes32 => bool) committedValues;

//     Status public status;
//     D.Edge originalRootEdge;
//     bytes32 originalRoot;
//     D.Edge targetRootEdge;
//     bytes32 targetRoot;

//     constructor() public {
//         // Init status
//         status = Status.OPENED;
//     }

//     function commitOriginalEdge(
//         uint _originalLabelLength,
//         bytes32 _originalLabel,
//         bytes32 _originalValue
//     ) public onlyFor(Status.OPENED) {
//         // Init original root edge
//         originalRootEdge.label = D.Label(_originalLabel, _originalLabelLength);
//         originalRootEdge.node = _originalValue;
//         originalRoot = PatriciaTree.edgeHash(originalRootEdge);
//     }

//     function commitTargetEdge(
//         uint _targetLabelLength,
//         bytes32 _targetLabel,
//         bytes32 _targetValue
//     ) public onlyFor(Status.OPENED) {
//         // Init target root edge
//         targetRootEdge.label = D.Label(_targetLabel, _targetLabelLength);
//         targetRootEdge.node = _targetValue;
//         targetRoot = PatriciaTree.edgeHash(targetRootEdge);
//     }

//     function insert(bytes memory key, bytes memory value) public {
//         bytes32 k = keccak256(value);
//         committedValues[k] = true;
//         tree.insert(key, value);
//     }

//     function commitNode(
//         bytes32 nodeHash,
//         uint firstEdgeLabelLength,
//         bytes32 firstEdgeLabel,
//         bytes32 firstEdgeValue,
//         uint secondEdgeLabelLength,
//         bytes32 secondEdgeLabel,
//         bytes32 secondEdgeValue
//     ) public onlyFor(Status.OPENED) {
//         D.Label memory k0 = D.Label(firstEdgeLabel, firstEdgeLabelLength);
//         D.Edge memory e0 = D.Edge(firstEdgeValue, k0);
//         D.Label memory k1 = D.Label(secondEdgeLabel, secondEdgeLabelLength);
//         D.Edge memory e1 = D.Edge(secondEdgeValue, k1);
//         require(tree.nodes[nodeHash].children[0].node == 0);
//         require(tree.nodes[nodeHash].children[1].node == 0);
//         require(nodeHash == keccak256(
//             abi.encodePacked(PatriciaTree.edgeHash(e0), PatriciaTree.edgeHash(e1)))
//         );
//         tree.nodes[nodeHash].children[0] = e0;
//         tree.nodes[nodeHash].children[1] = e1;
//     }

//     function commitValue(bytes memory value) public onlyFor(Status.OPENED) {
//         bytes32 k = keccak256(value);
//         committedValues[k] = true;
//         tree.values[k] = value;
//     }

//     function seal() public onlyFor(Status.OPENED) {
//         //        require(_verifyEdge(originalRootEdge));
//         tree.rootEdge = originalRootEdge;
//         tree.root = PatriciaTree.edgeHash(tree.rootEdge);
//         _changeStatus(Status.ONGOING);
//     }

//     function proof() public onlyFor(Status.ONGOING) {
//         require(targetRootEdge.node == tree.rootEdge.node);
//         require(targetRootEdge.label.length == tree.rootEdge.label.length);
//         require(targetRootEdge.label.data == tree.rootEdge.label.data);
//         require(_verifyEdge(tree.rootEdge));
//         _changeStatus(Status.SUCCESS);
//     }

//     function getRootHash() public view returns (bytes32) {
//         return tree.getRootHash();
//     }

//     function _verifyEdge(D.Edge memory _edge) internal view returns (bool) {
//         if (_edge.node == 0) {
//             // Empty. Return true because there is nothing to verify
//             return true;
//         } else if (_isLeaf(_edge)) {
//             // check stored value of the leaf node
//             require(_hasValue(_edge.node));
//         } else {
//             D.Edge[2] memory children = tree.nodes[_edge.node].children;
//             // its node value should be the hashed value of its child nodes
//             require(_edge.node == keccak256(
//                 abi.encodePacked(PatriciaTree.edgeHash(children[0]), PatriciaTree.edgeHash(children[1]))
//             ));
//             // check children recursively
//             require(_verifyEdge(children[0]));
//             require(_verifyEdge(children[1]));
//         }
//         return true;
//     }

//     function _isLeaf(D.Edge memory _edge) internal view returns (bool) {
//         return (tree.nodes[_edge.node].children[0].node == 0 && tree.nodes[_edge.node].children[1].node == 0);
//     }

//     function _hasValue(bytes32 valHash) internal view returns (bool) {
//         return committedValues[valHash];
//     }

//     function _changeStatus(Status _status) internal {
//         require(status < _status);
//         // unidirectional
//         status = _status;
//         emit OnChangeStatus(status);
//     }
// }
