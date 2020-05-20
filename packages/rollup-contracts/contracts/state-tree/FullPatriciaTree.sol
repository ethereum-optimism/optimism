pragma solidity >=0.5.0 <0.6.0;
pragma experimental ABIEncoderV2;

import {D} from "./DataTypes.sol";
import {Utils} from "./BinaryUtils.sol";

/**
 MIT License
 Original author: chriseth
 Rewritten by: Wanseob Lim
 */

library FullPatriciaTree {
    struct Tree {
        // Mapping of hash of key to value
        mapping(bytes32 => bytes) values;

        // Particia tree nodes (hash to decoded contents)
        mapping(bytes32 => D.Node) nodes;
        // The current root hash, keccak256(node(path_M('')), path_M(''))
        bytes32 root;
        D.Edge rootEdge;
    }

    function get(Tree storage tree, bytes32 key) internal view returns (bytes memory) {
        return getValue(tree, _findNode(tree, key));
    }

    function safeGet(Tree storage tree, bytes32 key) internal view returns (bytes memory value) {
        bytes32 valueHash = _findNode(tree, key);
        require(valueHash != bytes32(0));
        value = getValue(tree, valueHash);
        require(valueHash == keccak256(value));
    }

    function doesInclude(Tree storage tree, bytes memory key) internal view returns (bool) {
        return doesIncludeHashedKey(tree, keccak256(key));
    }

    function doesIncludeHashedKey(Tree storage tree, bytes32 hashedKey) internal view returns (bool) {
        bytes32 valueHash = _findNodeWithHashedKey(tree, hashedKey);
        return (valueHash != bytes32(0));
    }

    function getValue(Tree storage tree, bytes32 valueHash) internal view returns (bytes memory) {
        return tree.values[valueHash];
    }

    function getRootHash(Tree storage tree) internal view returns (bytes32) {
        return tree.root;
    }


    function getNode(Tree storage tree, bytes32 hash) internal view returns (uint, bytes32, bytes32, uint, bytes32, bytes32) {
        D.Node storage n = tree.nodes[hash];
        return (
        n.children[0].label.length, n.children[0].label.data, n.children[0].node,
        n.children[1].label.length, n.children[1].label.data, n.children[1].node
        );
    }

    function getRootEdge(Tree storage tree) internal view returns (uint, bytes32, bytes32) {
        return (tree.rootEdge.label.length, tree.rootEdge.label.data, tree.rootEdge.node);
    }

    function edgeHash(D.Edge memory e) internal pure returns (bytes32) {
        return keccak256(abi.encode(e.node, e.label.length, e.label.data));
    }

    // Returns the hash of the encoding of a node.
    function hash(D.Node memory n) internal pure returns (bytes32) {
        return keccak256(abi.encode(edgeHash(n.children[0]), edgeHash(n.children[1])));
    }

    // Returns the Merkle-proof for the given key
    // Proof format should be:
    //  - uint branchMask - bitmask with high bits at the positions in the key
    //                    where we have branch nodes (bit in key denotes direction)
    //  - bytes32[] hashes - hashes of sibling edges
    
    function getProof(Tree storage tree, bytes32 key) public view returns (uint branchMask, bytes32[] memory _siblings) {
        // We will progressively "eat" into the key from the left as we traverse.
        D.Label memory remaining;
        // We initialize to the full key
        remaining = D.Label(key, 256);
        // Keeps track of how much we have "eaten" into the key.
        // It should always hold that bitsTraversed + remaining.length == keyLength (256) at the end of each loop iteration
        uint bitsTraversed = 0;
        // Current edge the traversal is processing.
        // Each loop iteration will chose the right or left child as the new current edge.
        D.Edge memory currentEdge;
        // Start traversal at the root
        currentEdge = tree.rootEdge;

        // Proof to return along with branch bitmask
        bytes32[256] memory siblings;
        uint numSiblings = 0;
        while (true) {
            // Figure out the common prefix between the current edge and remaning bits in the traversal.
            // If the requested key has indeed been set, the current edge should be a prefix of the remaining.
            D.Label memory prefix;
            D.Label memory suffix;
            (prefix, suffix) = Utils.splitCommonPrefix(remaining, currentEdge.label);
            require(
                prefix.length == currentEdge.label.length,
                'Reached an edge in traversal whose label is not a strict prefix of the remaining part of key.  This indicates that the requested key has not been set.'
            );
            if (suffix.length == 0) {
                // Found a match!
                break;
            }
            // Now that we are traversing this edge, add its length to bitsTraversed.
            bitsTraversed += prefix.length;
            // The next bit in the key determines whether to branch left or right.
            // So, this sets the bitsTraversed'th bit in the branch mask to 1.
            branchMask |= uint(1) << (255 - bitsTraversed);

            // As explained in the last line, we traverse left or right based on the next bit.
            uint head;
            D.Label memory tail;
            (head, tail) = Utils.chopFirstBit(suffix);
            // head, either 0 or 1, tells us which child edge is to traverse to, and which is the sibling to supply in our proof.
            uint siblingIndex = 1 - head;
            siblings[numSiblings++] = edgeHash(
                tree.nodes[currentEdge.node].children[siblingIndex]
            );
            // Now, update the current edge to be processed in next iteration
            currentEdge = tree.nodes[currentEdge.node].children[head];
            // Account for having processed another bit by choosing left or right.
            bitsTraversed += 1;
            remaining = tail;
        }
        if (numSiblings > 0)
        {
            _siblings = new bytes32[](numSiblings);
            for (uint i = 0; i < numSiblings; i++)
                _siblings[i] = siblings[i];
        }
    }

    function getNonInclusionProof(Tree storage tree, bytes32 key) internal view returns (
        D.Label memory potentialSiblingCumulativeLabel,
        bytes32 potentialSiblingValue,
        uint branchMask,
        bytes32[] memory _siblings
    ){
        uint length;
        uint numSiblings;

        D.Label memory cumulativeKeyLabel = D.Label(key, 256);

        // Start from root edge
        D.Label memory label = cumulativeKeyLabel;
        D.Edge memory e = tree.rootEdge;
        bytes32[256] memory siblings;

        while (true) {
            // Find at edge
            require(label.length >= e.label.length);
            D.Label memory prefix;
            D.Label memory suffix;
            (prefix, suffix) = Utils.splitCommonPrefix(label, e.label);

            // suffix.length == 0 means that the key exists. Thus the length of the suffix should be not zero
            require(suffix.length != 0);

            if (prefix.length >= e.label.length) {
                // Partial matched, keep finding
                length += prefix.length;
                branchMask |= uint(1) << (255 - length);
                length += 1;
                uint head;
                (head, label) = Utils.chopFirstBit(suffix);
                siblings[numSiblings++] = edgeHash(tree.nodes[e.node].children[1 - head]);
                e = tree.nodes[e.node].children[head];
            } else {
                // Found the potential sibling. Set data to return
                (D.Label memory sharedCumulativePrefix, ) = Utils.splitAt(cumulativeKeyLabel, length);
                potentialSiblingCumulativeLabel = Utils.combineLabels(sharedCumulativePrefix, e.label);
                potentialSiblingValue = e.node;
                break;
            }
        }
        if (numSiblings > 0)
        {
            _siblings = new bytes32[](numSiblings);
            for (uint i = 0; i < numSiblings; i++)
                _siblings[i] = siblings[i];
        }
    }

    // TODO comment/explain these args
    function verifyEdgeInclusionProof(
        bytes32 rootHash,
        bytes32 edgeCommittment,
        D.Label memory fullEdgeLabel,
        uint branchMask,
        bytes32[] memory siblings
    ) internal pure {
        // We will progressively "eat" into the label from the right as we hash up to the root.
        D.Label memory remaining = fullEdgeLabel;
        // We will progressively hash the current edge up with its siblings until we get the root edge.
        D.Edge memory currentEdge;
        // To start, this is the edge we are verifying so it's the edgeHash which was input
        currentEdge.node = edgeCommittment;
        // Iterate over each set bit in the branch mask to build parent edges, starting from the right.
        for (uint i = 0; branchMask != 0; i++) {
            // Find the lowest index nonzero bit in the mask, where rightmost == index 0
            uint bitSet = Utils.lowestBitSet(branchMask);
            // Remove from bitmask as we are about to process it
            branchMask &= ~(uint(1) << bitSet);
            // The label for the current edge is the suffix of the remaining label proceeeding the set bit
            (remaining, currentEdge.label) = Utils.splitAt(remaining, 255 - bitSet); // (255 - bitSet) since bitset indexes from the right
            // The bitSet'th bit in the key determines whether the sibling is left or right.
            uint bit;
            // chop this bit off the label, it is implicit in the merkle path so will not be included in a label
            (bit, currentEdge.label) = Utils.chopFirstBit(currentEdge.label);
            bytes32[2] memory edgeHashes;
            edgeHashes[bit] = edgeHash(currentEdge);
            edgeHashes[1 - bit] = siblings[siblings.length - i - 1];
            currentEdge.node = keccak256(abi.encode(edgeHashes[0], edgeHashes[1]));
        }
        // no more branching, so the remaining label is the root edge's label
        currentEdge.label = remaining;
        require(rootHash == edgeHash(currentEdge), 'Edge inclusion proof verification failed: root hashes do not match.');
    }

    function verifyProof(
        bytes32 rootHash,
        bytes32 key,
        bytes memory value,
        uint branchMask,
        bytes32[] memory siblings
    ) public pure {
        // The edge above a leaf commits to the leaf value (i.e. what was actually set) 
        bytes32 edgeCommittment = keccak256(value);
        // The full "label" for a leaf node is the entirety of the key.
        D.Label memory fullLabel = D.Label(key, 256);

        verifyEdgeInclusionProof(
            rootHash,
            edgeCommittment,
            fullLabel,
            branchMask,
            siblings
        );
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
        // first, verify there is a conflict between the key and given edge
        require(conflictingEdgeFullLabelLength <= 256, 'invalid label specified');
        D.Label memory fullConflictingEdgeLabel = D.Label(conflictingEdgeFullLabelData, conflictingEdgeFullLabelLength);
        D.Label memory fullLeafLabel = D.Label(key, 256);
        uint indexOfConflict = Utils.commonPrefix(fullConflictingEdgeLabel, fullLeafLabel);
        uint doesBranchAtConflict = branchMask & (1 << 255 - indexOfConflict);
        
        require(doesBranchAtConflict == 0, 'The provided conflicting edge is not actually conflicting.');
        verifyEdgeInclusionProof(
            rootHash,
            conflictingEdgeCommitment,
            fullConflictingEdgeLabel,
            branchMask,
            siblings
        );
    }

    function insert(Tree storage tree, bytes32 key, bytes memory value) internal {
        D.Label memory k = D.Label(key, 256);
        bytes32 valueHash = keccak256(value);
        tree.values[valueHash] = value;
        // keys.push(key);
        D.Edge memory e;
        if (tree.rootEdge.node == 0 && tree.rootEdge.label.length == 0)
        {
            // Empty Trie
            e.label = k;
            e.node = valueHash;
        }
        else
        {
            e = _insertAtEdge(tree, tree.rootEdge, k, valueHash);
        }
        tree.root = edgeHash(e);
        tree.rootEdge = e;
    }

    function _insertAtNode(
        Tree storage tree, 
        bytes32 nodeHash, 
        D.Label memory key, 
        bytes32 value
    ) private returns (bytes32) {
        require(key.length > 1, "Bad key");
        D.Node memory n = tree.nodes[nodeHash];
        (uint256 head, D.Label memory tail) = Utils.chopFirstBit(key);
        n.children[head] = _insertAtEdge(tree, n.children[head], tail, value);
        return _replaceNode(tree, nodeHash, n);
    }

    function _insertAtEdge(
        Tree storage tree, 
        D.Edge memory e, 
        D.Label memory key, bytes32 value
    ) private returns (D.Edge memory) {
        require(key.length >= e.label.length, "Key lenght mismatch label lenght");
        (D.Label memory prefix, D.Label memory suffix) = Utils.splitCommonPrefix(key, e.label);
        bytes32 newNodeHash;
        if (suffix.length == 0) {
            // Full match with the key, update operation
            newNodeHash = value;
        } else if (prefix.length >= e.label.length) {
            // Partial match, just follow the path
            newNodeHash = _insertAtNode(tree, e.node, suffix, value);
        } else {
            // Mismatch, so let us create a new branch node.
            (uint256 head, D.Label memory tail) = Utils.chopFirstBit(suffix);
            D.Node memory branchNode;
            branchNode.children[head] = D.Edge(value, tail);
            branchNode.children[1 - head] = D.Edge(e.node, Utils.removePrefix(e.label, prefix.length + 1));
            newNodeHash = _insertNode(tree, branchNode);
        }
        return D.Edge(newNodeHash, prefix);
    }

    function _insertNode(Tree storage tree, D.Node memory n) private returns (bytes32 newHash) {
        bytes32 h = hash(n);
        tree.nodes[h].children[0] = n.children[0];
        tree.nodes[h].children[1] = n.children[1];
        return h;
    }

    function _replaceNode(
        Tree storage tree, 
        bytes32 oldHash, 
        D.Node memory n
    ) private returns (bytes32 newHash) {
        delete tree.nodes[oldHash];
        return _insertNode(tree, n);
    }

// todo remove this
    function _findNode(Tree storage tree, bytes32 key) private view returns (bytes32) {
        return _findNodeWithHashedKey(tree, key);
    }

    function _findNodeWithHashedKey(Tree storage tree, bytes32 hashedKey) private view returns (bytes32) {
        if (tree.rootEdge.node == 0 && tree.rootEdge.label.length == 0) {
            return 0;
        } else {
            D.Label memory k = D.Label(hashedKey, 256);
            return _findAtEdge(tree, tree.rootEdge, k);
        }
    }

    function _findAtNode(Tree storage tree, bytes32 nodeHash, D.Label memory key) private view returns (bytes32) {
        require(key.length > 1);
        D.Node memory n = tree.nodes[nodeHash];
        (uint head, D.Label memory tail) = Utils.chopFirstBit(key);
        return _findAtEdge(tree, n.children[head], tail);
    }

    function _findAtEdge(Tree storage tree, D.Edge memory e, D.Label memory key) private view returns (bytes32){
        require(key.length >= e.label.length);
        (D.Label memory prefix, D.Label memory suffix) = Utils.splitCommonPrefix(key, e.label);
        if (suffix.length == 0) {
            // Full match with the key, update operation
            return e.node;
        } else if (prefix.length >= e.label.length) {
            // Partial match, just follow the path
            return _findAtNode(tree, e.node, suffix);
        } else {
            // Mismatch, return empty bytes
            return bytes32(0);
        }
    }
}

