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

    function update(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    ) public pure returns (bytes32) {
        TrieNode[] memory proof = parseProof(_proof);
        (uint256 pathLength, bytes memory keyRemainder, ) = getNodePath(proof, _key, _root);

        TrieNode[] memory newPath = getNewPath(proof, pathLength, keyRemainder, _value);

        return getUpdatedTrieRoot(newPath, _key);
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
    ) internal pure returns (bool) {
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
                bytes memory pathRemainder = BytesLib.slice(path, offset);
                bytes memory keyRemainder = BytesLib.slice(key, currentKeyIndex);
                uint256 sharedNibbleLength = getSharedNibbleLength(pathRemainder, keyRemainder);

                if (prefix == PREFIX_LEAF_EVEN || prefix == PREFIX_LEAF_ODD) {
                    if (pathRemainder.length == sharedNibbleLength && keyRemainder.length == sharedNibbleLength) {
                        currentKeyIndex += sharedNibbleLength;
                    }
                    currentNodeID = bytes32(RLP_NULL);
                    break;
                } else if (prefix == PREFIX_EXTENSION_EVEN || prefix == PREFIX_EXTENSION_ODD) {
                    if (sharedNibbleLength == 0) {
                        currentNodeID = bytes32(RLP_NULL);
                        break;
                    } else {
                        currentNodeID = getNodeID(currentNode.decoded[1]);
                        currentKeyIncrement = sharedNibbleLength;
                        continue;
                    }
                }
            }
        }

        bool isFinalNode = currentNodeID == bytes32(RLP_NULL);
        return (pathLength, BytesLib.slice(key, currentKeyIndex), isFinalNode);
    }

    function getNewPath(
        TrieNode[] memory _path,
        uint256 _pathLength,
        bytes memory _keyRemainder,
        bytes memory _value
    ) internal pure returns (TrieNode[] memory) {
        bytes memory keyRemainder = _keyRemainder;

        TrieNode memory lastNode = _path[_pathLength - 1];
        NodeType lastNodeType = getNodeType(lastNode);

        TrieNode[] memory newNodes = new TrieNode[](3);
        uint256 totalNewNodes = 0;

        if (keyRemainder.length == 0 && lastNodeType == NodeType.LeafNode) {
            newNodes[totalNewNodes] = makeLeafNode(getNodeKey(lastNode), _value);
            totalNewNodes += 1;
        } else if (lastNodeType == NodeType.BranchNode) {
            if (keyRemainder.length == 0) {
                newNodes[totalNewNodes] = editBranchValue(lastNode, _value);
                totalNewNodes += 1;
            } else {
                newNodes[totalNewNodes] = lastNode;
                totalNewNodes += 1;
                newNodes[totalNewNodes] = makeLeafNode(BytesLib.slice(keyRemainder, 1), _value);
                totalNewNodes += 1;
            }
        } else {
            bytes memory lastNodeKey = getNodeKey(lastNode);
            uint256 sharedNibbleLength = getSharedNibbleLength(lastNodeKey, keyRemainder);

            if (sharedNibbleLength != 0) {
                bytes memory nextNodeKey = BytesLib.slice(lastNodeKey, 0, sharedNibbleLength);
                newNodes[totalNewNodes] = makeExtensionNode(nextNodeKey, getNodeHash(_value));
                totalNewNodes += 1;
                lastNodeKey = BytesLib.slice(lastNodeKey, sharedNibbleLength);
                keyRemainder = BytesLib.slice(keyRemainder, sharedNibbleLength);
            }

            TrieNode memory newBranch = makeEmptyBranchNode();

            if (lastNodeKey.length == 0) {
                newBranch = editBranchValue(newBranch, getNodeValue(lastNode));
            } else {
                uint8 branchKey = uint8(lastNodeKey[0]);
                lastNodeKey = BytesLib.slice(lastNodeKey, 1);

                if (lastNodeKey.length != 0 || lastNodeType == NodeType.LeafNode) {
                    TrieNode memory modifiedLastNode = makeLeafNode(lastNodeKey, getNodeValue(lastNode));
                    newBranch = editBranchIndex(newBranch, branchKey, getNodeHash(modifiedLastNode.encoded));
                } else {
                    newBranch = editBranchIndex(newBranch, branchKey, getNodeValue(lastNode));
                }
            }

            if (keyRemainder.length == 0) {
                newBranch = editBranchValue(newBranch, _value);
                newNodes[totalNewNodes] = newBranch;
                totalNewNodes += 1;
            } else {
                keyRemainder = BytesLib.slice(keyRemainder, 1);
                newNodes[totalNewNodes] = newBranch;
                totalNewNodes += 1;
                newNodes[totalNewNodes] = makeLeafNode(keyRemainder, _value);
                totalNewNodes += 1;
            }
        }

        return concatNodes(_path, _pathLength - 1, newNodes, totalNewNodes);
    }

    function getUpdatedTrieRoot(
        TrieNode[] memory _nodes,
        bytes memory _key
    ) internal pure returns (bytes32) {
        bytes memory key = BytesLib.toNibbles(_key);

        TrieNode memory currentNode;
        NodeType currentNodeType;
        bytes memory previousNodeHash;

        for (uint256 i = _nodes.length; i > 0; i--) {
            currentNode = _nodes[i - 1];
            currentNodeType = getNodeType(currentNode);

            if (currentNodeType == NodeType.LeafNode) {
                bytes memory nodeKey = getNodeKey(currentNode);
                key = BytesLib.slice(key, 0, key.length - nodeKey.length);
            } else if (currentNodeType == NodeType.ExtensionNode) {
                bytes memory nodeKey = getNodeKey(currentNode);
                key = BytesLib.slice(key, 0, key.length - nodeKey.length);

                if (previousNodeHash.length > 0) {
                    currentNode = makeExtensionNode(nodeKey, previousNodeHash);
                }
            } else if (currentNodeType == NodeType.BranchNode) {
                if (previousNodeHash.length > 0) {
                    uint8 branchKey = uint8(key[key.length - 1]);
                    key = BytesLib.slice(key, 0, key.length - 1);
                    currentNode = editBranchIndex(currentNode, branchKey, previousNodeHash);
                }
            }

            previousNodeHash = getNodeHash(currentNode.encoded);
        }

        return keccak256(currentNode.encoded);
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

    function getNodeKey(
        TrieNode memory _node
    ) internal pure returns (bytes memory) {
        return removeHexPrefix(getNodePath(_node));
    }

    function getNodeHash(
        bytes memory _encoded
    ) internal pure returns (bytes memory) {
        if (_encoded.length < 32) {
            return _encoded;
        } else {
            return abi.encodePacked(keccak256(_encoded));
        }
    }

    function getNodeType(
        TrieNode memory _node
    ) internal pure returns (NodeType) {
        if (_node.decoded.length == BRANCH_NODE_LENGTH) {
            return NodeType.BranchNode;
        } else {
            bytes memory path = getNodePath(_node);
            uint8 prefix = uint8(path[0]);
            if (prefix == PREFIX_LEAF_EVEN || prefix == PREFIX_LEAF_ODD) {
                return NodeType.LeafNode;
            } else if (prefix == PREFIX_EXTENSION_EVEN || prefix == PREFIX_EXTENSION_ODD) {
                return NodeType.ExtensionNode;
            }
        }

        revert("Invalid node type");
    }

    function getSharedNibbleLength(bytes memory _a, bytes memory _b) internal pure returns (uint256) {
        uint256 i = 0;
        while (_a.length > i && _b.length > i && _a[i] == _b[i]) {
            i++;
        }
        return i;
    }

    function makeNode(
        bytes[] memory _raw
    ) internal pure returns (TrieNode memory) {
        bytes memory encoded = RLPWriter.encodeList(_raw);

        return TrieNode({
            encoded: encoded,
            decoded: RLPReader.toList(RLPReader.toRlpItem(encoded))
        });
    }

    function makeNode(
        RLPReader.RLPItem[] memory _items
    ) internal pure returns (TrieNode memory) {
        bytes[] memory raw = new bytes[](_items.length);
        for (uint256 i = 0; i < _items.length; i++) {
            raw[i] = RLPReader.toRlpBytes(_items[i]);
        }
        return makeNode(raw);
    }

    function makeExtensionNode(
        bytes memory _key,
        bytes memory _value
    ) internal pure returns (TrieNode memory) {
        bytes[] memory raw = new bytes[](2);
        bytes memory key = addHexPrefix(_key, false);
        raw[0] = RLPWriter.encodeBytes(BytesLib.fromNibbles(key));
        raw[1] = RLPWriter.encodeBytes(_value);
        return makeNode(raw);
    }

    function makeLeafNode(
        bytes memory _key,
        bytes memory _value
    ) internal pure returns (TrieNode memory) {
        bytes[] memory raw = new bytes[](2);
        bytes memory key = addHexPrefix(_key, true);
        raw[0] = RLPWriter.encodeBytes(BytesLib.fromNibbles(key));
        raw[1] = RLPWriter.encodeBytes(_value);
        return makeNode(raw);
    }

    function makeEmptyBranchNode() internal pure returns (TrieNode memory) {
        bytes[] memory raw = new bytes[](BRANCH_NODE_LENGTH);
        for (uint256 i = 0; i < raw.length; i++) {
            raw[i] = hex'80';
        }
        return makeNode(raw);
    }

    function editBranchValue(
        TrieNode memory _branch,
        bytes memory _value
    ) internal pure returns (TrieNode memory) {
        bytes memory encoded = RLPWriter.encodeBytes(_value);
        _branch.decoded[_branch.decoded.length - 1] = RLPReader.toRlpItem(encoded);
        return makeNode(_branch.decoded);
    }

    function editBranchIndex(
        TrieNode memory _branch,
        uint8 _index,
        bytes memory _value
    ) internal pure returns (TrieNode memory) {
        bytes memory encoded = _value.length < 32 ? _value : RLPWriter.encodeBytes(_value);
        _branch.decoded[_index] = RLPReader.toRlpItem(encoded);
        return makeNode(_branch.decoded);
    }

    function concatNodes(
        TrieNode[] memory _a,
        uint256 _aLength,
        TrieNode[] memory _b,
        uint256 _bLength
    ) internal pure returns (TrieNode[] memory) {
        TrieNode[] memory ret = new TrieNode[](_aLength + _bLength);

        for (uint256 i = 0; i < _aLength; i++) {
            ret[i] = _a[i];
        }

        for (uint256 i = 0; i < _bLength; i++) {
            ret[i + _aLength] = _b[i];
        }

        return ret;
    }

    function addHexPrefix(
        bytes memory _path,
        bool _isLeaf
    ) internal pure returns (bytes memory) {
        uint8 prefix = _isLeaf ? uint8(0x02) : uint8(0x00);
        uint8 offset = uint8(_path.length % 2);
        bytes memory prefixed = new bytes(2 - offset);
        prefixed[0] = bytes1(prefix + offset);
        return BytesLib.concat(prefixed, _path);
    }

    function removeHexPrefix(
        bytes memory _path
    ) internal pure returns (bytes memory) {
        if (uint8(_path[0]) % 2 == 0) {
            return BytesLib.slice(_path, 2);
        } else {
            return BytesLib.slice(_path, 1);
        }
    }
}