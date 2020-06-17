// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.5.0;

import './BytesLib.sol';
import './RLPReader.sol';
import './RLPWriter.sol';

contract MerkleTrie {
    uint256 constant TREE_RADIX = 16;
    uint256 constant BRANCH_NODE_LENGTH = TREE_RADIX + 1;
    uint256 constant LEAF_OR_EXTENSION_NODE_LENGTH = 2;

    uint8 constant PREFIX_EXTENSION_EVEN = 0;
    uint8 constant PREFIX_EXTENSION_ODD = 1;
    uint8 constant PREFIX_LEAF_EVEN = 2;
    uint8 constant PREFIX_LEAF_ODD = 3;

    bytes1 constant RLP_NULL = bytes1(0x80);

    enum NodeType {
        BranchNode,
        ExtensionNode,
        LeafNode
    }

    struct TrieNode {
        bytes encoded;
        RLPReader.RLPItem[] decoded;
    }


    /*
     * Public Functions
     */

    function verifyInclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    ) public pure returns (bool) {
        return verifyProof(_key, _value, _proof, _root, true);
    }

    function verifyExclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    ) public pure returns (bool) {
        return verifyProof(_key, _value, _proof, _root, false);
    }


    /*
     * Internal Functions
     */

    function verifyProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root,
        bool _inclusion
    ) public pure returns (bool) {
        TrieNode[] memory proof = parseProof(_proof);
        (uint256 pathLength, bytes memory keyRemainder, bool isFinalNode) = getNodePath(proof, _key, _root);

        if (_inclusion) {
            return (
                keyRemainder.length == 0 &&
                BytesLib.equal(getNodeValue(proof[pathLength - 1]), _value)
            );
        } else {
            return (
                (keyRemainder.length == 0 && !BytesLib.equal(getNodeValue(proof[pathLength - 1]), _value)) ||
                (keyRemainder.length != 0 && isFinalNode)
            );
        }
    }

    function getNodePath(
        TrieNode[] memory _proof,
        bytes memory _key,
        bytes32 _root
    ) internal pure returns (
        uint256,
        bytes memory,
        bool
    ) {
        uint256 pathLength = 0;
        bytes memory key = BytesLib.toNibbles(_key);

        bytes32 currentNodeID = _root;
        uint256 currentKeyIndex = 0;
        uint256 currentKeyIncrement = 0;
        TrieNode memory currentNode;

        for (uint256 i = 0; i < _proof.length; i++) {
            currentNode = _proof[i];
            currentKeyIndex += currentKeyIncrement;
            pathLength += 1;

            if (currentKeyIndex == 0) {
                // First proof element is always the root node.
                require(
                    keccak256(currentNode.encoded) == currentNodeID,
                    "Invalid root hash"
                );
            } else if (currentNode.encoded.length >= 32) {
                // Nodes 32 bytes or larger are hashed inside branch nodes.
                require(
                    keccak256(currentNode.encoded) == currentNodeID,
                    "Invalid large internal hash"
                );
            } else {
                // Nodes smaller than 31 bytes aren't hashed.
                require(
                    BytesLib.toBytes32(currentNode.encoded) == currentNodeID,
                    "Invalid internal node hash"
                );
            }

            if (currentNode.decoded.length == BRANCH_NODE_LENGTH) {
                if (currentKeyIndex == key.length) {
                    break;
                } else {
                    uint8 branchKey = uint8(key[currentKeyIndex]);
                    RLPReader.RLPItem memory nextNode = currentNode.decoded[branchKey];
                    currentNodeID = getNodeID(nextNode);
                    currentKeyIncrement = 1;
                    continue;
                }
            } else if (currentNode.decoded.length == LEAF_OR_EXTENSION_NODE_LENGTH) {
                bytes memory path = getNodePath(currentNode);
                uint8 prefix = uint8(path[0]);
                uint8 offset = 2 - prefix % 2;
                bytes memory sharedNibbles = BytesLib.slice(path, offset);

                if (prefix == PREFIX_LEAF_EVEN || prefix == PREFIX_LEAF_ODD) {
                    currentKeyIndex += sharedNibbles.length;
                    currentNodeID = bytes32(RLP_NULL);
                    break;
                } else if (prefix == PREFIX_EXTENSION_EVEN || prefix == PREFIX_EXTENSION_ODD) {
                    if (sharedNibbles.length == 0) {
                        break;
                    } else {
                        require (
                            BytesLib.equal(
                                sharedNibbles,
                                BytesLib.slice(key, currentKeyIndex, sharedNibbles.length)
                            ),
                            "Invalid extension node in provided path."
                        );

                        currentNodeID = getNodeID(currentNode.decoded[1]);
                        currentKeyIncrement = sharedNibbles.length;
                        continue;
                    }
                }
            }
        }

        bool isFinalNode = currentNodeID == bytes32(RLP_NULL);
        return (pathLength, BytesLib.slice(key, currentKeyIndex), isFinalNode);
    }

    function parseProof(
        bytes memory _proof
    ) internal pure returns (TrieNode[] memory) {
        RLPReader.RLPItem[] memory nodes = RLPReader.toList(RLPReader.toRlpItem(_proof));
        TrieNode[] memory proof = new TrieNode[](nodes.length);

        for (uint256 i = 0; i < nodes.length; i++) {
            bytes memory encoded = RLPReader.toBytes(nodes[i]);
            proof[i] = TrieNode({
                encoded: encoded,
                decoded: RLPReader.toList(RLPReader.toRlpItem(encoded))
            });
        }

        return proof;
    }

    function getNodeID(
        RLPReader.RLPItem memory _node
    ) internal pure returns (bytes32) {
        bytes memory nodeID;

        if (_node.len < 32) {
            // Nodes smaller than 32 bytes are RLP encoded.
            nodeID = RLPReader.toRlpBytes(_node);
        } else {
            // Nodes 32 bytes or larger are hashed.
            nodeID = RLPReader.toBytes(_node);
        }

        return BytesLib.toBytes32(nodeID);
    }

    function getNodePath(
        TrieNode memory _node
    ) internal pure returns (bytes memory) {
        return BytesLib.toNibbles(RLPReader.toBytes(_node.decoded[0]));
    }

    function getNodeValue(
        TrieNode memory _node
    ) internal pure returns (bytes memory) {
        return RLPReader.toBytes(_node.decoded[_node.decoded.length - 1]);
    }
}