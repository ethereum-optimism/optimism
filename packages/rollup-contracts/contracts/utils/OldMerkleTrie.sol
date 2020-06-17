pragma solidity ^0.5.0;

import './BytesLib.sol';
import './RLPReader.sol';
import './RLPWriter.sol';

/**
 * @notice Contract for dealing with Merkle tries.
 */
contract OldMerkleTrie {
    using BytesLib for bytes;

    uint256 constant TREE_RADIX = 16;
    uint256 constant BRANCH_NODE_LENGTH = TREE_RADIX + 1;

    uint8 constant PREFIX_EVEN_EXTENSION = 0;
    uint8 constant PREFIX_ODD_EXTENSION = 1;
    uint8 constant PREFIX_EVEN_LEAF = 2;
    uint8 constant PREFIX_ODD_LEAF = 3;

    enum NodeType {
        BranchNode,
        ExtensionNode,
        LeafNode
    }

    struct TrieNode {
        bytes encoded;
        RLPReader.RLPItem[] decoded;
    }

    struct Trie {
        bytes32 root;
        TrieNode[] nodes;
    }


    /*
     * Public Functions
     */

    /**
     * @notice Checks a trie inclusion proof.
     * @param _key Key of the node to verify.
     * @param _value Value of the node to verify.
     * @param _proof Encoded proof.
     * @return `true` if the node is in the trie, `false` otherwise.
     */
    function verifyInclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof
    ) public pure returns (bool) {
        Trie memory trie = getTrieFromProof(_proof);
        (TrieNode memory target, bytes memory keyRemainder, , ) = getTrieNode(trie, _key);

        require(
            getNodeValue(target).equal(_value) &&
            keyRemainder.length == 0,
            "Invalid node value"
        );

        return true;
    }

    function updateTrieRoot(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof
    ) public pure returns (bytes32) {
        Trie memory trie = getTrieFromProof(_proof);
        (TrieNode memory target, bytes memory keyRemainder, TrieNode[] memory path, uint256 pathLength) = getTrieNode(trie, _key);

        TrieNode[] memory nodes = copyTrieNodes(path, pathLength, pathLength + 3);
        uint256 newNodes = 0;

        if (keyRemainder.length == 0 && getNodeType(target) == NodeType.LeafNode) {
            nodes[pathLength + newNodes] = encodeLeafNode(getNodeKey(target), _value);
            newNodes += 1;
        } else if (getNodeType(target) == NodeType.BranchNode) {
            if (keyRemainder.length == 0) {
                nodes[pathLength + newNodes] = encodeBranchValue(target, _value);
                newNodes += 1;
            } else {
                nodes[pathLength + newNodes] = target;
                newNodes += 1;
                nodes[pathLength + newNodes] = encodeLeafNode(keyRemainder.slice(1), _value);
                newNodes += 1;
            }
        } else {
            bytes memory lastKey = getNodeKey(target);
            uint256 matchingLength = getMatchingNibbleLength(lastKey, keyRemainder);

            if (matchingLength != 0) {
                bytes memory newKey = lastKey.slice(0, matchingLength);
                nodes[pathLength + newNodes] = encodeExtensionNode(newKey, getNodeHash(_value), false);
                newNodes += 1;
                lastKey = lastKey.slice(matchingLength);
                keyRemainder = keyRemainder.slice(matchingLength);
            }

            TrieNode memory branch = encodeEmptyBranch();

            if (lastKey.length == 0) {
                branch = encodeBranchValue(branch, getNodeValue(target));
            } else {
                bytes1 branchKey = lastKey[0];
                lastKey = lastKey.slice(1);

                if (lastKey.length != 0 || getNodeType(target) == NodeType.LeafNode) {
                    bytes memory encoded = encodeLeafNode(lastKey, getNodeValue(target)).encoded;
                    branch = encodeBranchIndex(branch, branchKey, RLPWriter.encodeBytes(getNodeHash(encoded)));
                } else {
                    branch = encodeBranchIndex(branch, branchKey, getNodeValue(target));
                }
            }

            if (keyRemainder.length == 0) {
                branch = encodeBranchValue(branch, _value);
                nodes[pathLength + newNodes] = branch;
                newNodes += 1;
            } else {
                keyRemainder = keyRemainder.slice(1);
                nodes[pathLength + newNodes] = branch;
                newNodes += 1;
                nodes[pathLength + newNodes] = encodeLeafNode(keyRemainder, _value);
                newNodes += 1;
            }
        }

        nodes = copyTrieNodes(nodes, pathLength + 3, pathLength + newNodes);

        return getUpdatedTrieRoot(nodes, _key, _value);
    }


    /*
     * Internal Functions
     */

    function getMatchingNibbleLength(bytes memory _a, bytes memory _b) internal pure returns (uint256) {
        uint256 i = 0;
        while (_a[i] == _b[i] && _a.length > i) {
            i++;
        }
        return i;
    }

    function getTrieFromProof(bytes memory _proof) internal pure returns (Trie memory) {
        RLPReader.RLPItem[] memory nodes = RLPReader.toList(RLPReader.toRlpItem(_proof));

        Trie memory trie;
        trie.nodes = new TrieNode[](nodes.length);
        for (uint256 i = 0; i < nodes.length; i++) {
            bytes memory encoded = RLPReader.toBytes(nodes[i]);
            trie.nodes[i] = TrieNode({
                encoded: encoded,
                decoded: RLPReader.toList(RLPReader.toRlpItem(encoded))
            });
        }

        trie.root = keccak256(trie.nodes[0].encoded);

        return trie;
    }

    function getUpdatedTrieRoot(
        TrieNode[] memory _nodes,
        bytes memory _key,
        bytes memory _value
    ) internal pure returns (bytes32) {
        bytes memory key = _key.toNibbles();

        TrieNode memory leaf = _nodes[_nodes.length - 1];
        bytes memory root = getNodeHash(leaf.encoded);
        bytes memory encoded;
        revert("Bad");

        for (uint256 i = _nodes.length - 1; i > 0; i--) {
            TrieNode memory node = _nodes[i - 1];
            if (node.decoded.length == BRANCH_NODE_LENGTH) {
                bytes1 branchKey = key[key.length - 1];
                key = key.slice(0, key.length - 1);
                encoded = encodeBranchIndex(node, branchKey, root).encoded;
            } else if (node.decoded.length == 2) {
                bytes memory nodeKey = i - 1 == 0 ? getNodePath(node) : removeHexPrefix(getNodePath(node));
                key = key.slice(0, key.length - nodeKey.length);
                encoded = encodeExtensionNode(nodeKey, root, i - 1 == 0).encoded;
            } else {
                revert("Invalid node");
            }

            root = getNodeHash(encoded);
        }

        return root.toBytes32();
    }

    function getTrieNode(
        Trie memory _trie,
        bytes memory _key
    ) internal pure returns (TrieNode memory, bytes memory, TrieNode[] memory, uint256) {
        // Convert the key into a series of half-packed bytes.
        bytes memory key = _key.toNibbles();

        TrieNode[] memory path = new TrieNode[](_trie.nodes.length);
        bytes32 currentNodeID = _trie.root;
        uint256 keyIndex = 0;
        uint256 prevKeyIndex = 0;
        uint256 pathLength = 0;
        for (uint256 i = 0; i < _trie.nodes.length; i++) {
            TrieNode memory node = _trie.nodes[i];
            pathLength = i;

            if (keyIndex == 0) {
                // First proof element is always the root node.
                require(
                    keccak256(node.encoded) == currentNodeID,
                    "Invalid root hash"
                );
            } else if (node.encoded.length >= 32) {
                // Nodes 32 bytes or larger are hashed inside branch nodes.
                require(
                    keccak256(node.encoded) == currentNodeID,
                    "Invalid large internal hash"
                );
            } else {
                // Nodes smaller than 31 bytes aren't hashed.
                require(
                    node.encoded.toBytes32() == currentNodeID,
                    "Invalid internal node hash"
                );
            }

            if (node.decoded.length == BRANCH_NODE_LENGTH) {
                if (keyIndex == key.length) {
                    // Value may sometimes be included at a branch node.
                    return (node, key.slice(keyIndex), path, i);
                } else {
                    // Find the next node within the branch node and repeat.
                    RLPReader.RLPItem memory next = node.decoded[uint8(key[keyIndex])];
                    currentNodeID = getNodeID(next).toBytes32();
                    prevKeyIndex = keyIndex;
                    keyIndex++;
                    path[i] = node;
                    continue;
                }
            }

            // Nodes with two elements are either leaves or extensions.
            if (node.decoded.length == 2) {
                // Throw this step into a new function to avoid `STACK_TOO_DEEP`.
                prevKeyIndex = keyIndex;
                bool done;
                (done, currentNodeID, keyIndex) = checkNonBranchNode(
                    node,
                    key,
                    keyIndex
                );

                if (done) {
                    return (node, key.slice(keyIndex), path, i);
                } else {
                    path[i] = node;
                    continue;
                }
            }
        }

        return (path[pathLength], key.slice(prevKeyIndex), path, pathLength);
    }

    function checkNonBranchNode(
        TrieNode memory _node,
        bytes memory _key,
        uint256 _keyIndex
    ) internal pure returns (bool, bytes32, uint256) {
        // First element of the node is its path.
        bytes memory path = getNodePath(_node);
        // First nibble of the path is its prefix.
        uint8 prefix = uint8(path[0]);
        // Even prefixes include an extra nibble.
        uint8 offset = 2 - prefix % 2;

        if (prefix == PREFIX_EVEN_LEAF || prefix == PREFIX_ODD_LEAF) {
            bytes memory value = getNodeID(_node.decoded[1]);
            bytes memory remainder = path.slice(offset);
            if (remainder.equal(_key.slice(_keyIndex, remainder.length))) {
                // Return "done" and fill the rest with empty values.
                return (true, bytes32(0), _keyIndex + remainder.length);
            } else {
                // Return "not done", set the next value, increment the key index.
                return (false, value.toBytes32(), _keyIndex + remainder.length);
            }
        } else if (prefix == PREFIX_EVEN_EXTENSION || prefix == PREFIX_ODD_EXTENSION) {
            bytes memory value = getNodeID(_node.decoded[1]);
            bytes memory shared = path.slice(offset);
            uint256 extension = shared.length;

            require (
                shared.equal(_key.slice(_keyIndex, extension)),
                "Invalid extension node"
            );

            if (extension == 0) {
                // Return "done" and fill the rest with empty values.
                return (true, bytes32(0), _keyIndex + extension);
            } else {
                // Return "not done", set the next value, increment the key index.
                return (false, value.toBytes32(), _keyIndex + extension);
            }
        }

        revert("Bad prefix");
    }

    function encodeNode(bytes memory _encoded) internal pure returns (TrieNode memory) {
        return TrieNode({
            encoded: _encoded,
            decoded: RLPReader.toList(RLPReader.toRlpItem(_encoded))
        });
    }

    function encodeEmptyBranch() internal pure returns (TrieNode memory) {
        bytes[] memory raw = new bytes[](BRANCH_NODE_LENGTH);
        for (uint256 i = 0; i < raw.length; i++) {
            raw[i] = hex'80';
        }
        return encodeNode(encodeByteList(raw));
    }

    function encodeLeafNode(
        bytes memory _path,
        bytes memory _value
    ) internal pure returns (TrieNode memory) {
        bytes[] memory raw = new bytes[](2);
        raw[0] = RLPWriter.encodeBytes(addHexPrefix(_path, true).fromNibbles());
        raw[1] = RLPWriter.encodeBytes(_value);
        return encodeNode(encodeByteList(raw));
    }

    function encodeExtensionNode(
        bytes memory _path,
        bytes memory _key,
        bool _isRoot
    ) internal pure returns (TrieNode memory) {
        bytes[] memory raw = new bytes[](2);
        bytes memory path = _isRoot ? _path : addHexPrefix(_path, false);
        raw[0] = RLPWriter.encodeBytes(path.fromNibbles());
        raw[1] = RLPWriter.encodeBytes(_key);
        return encodeNode(encodeByteList(raw));
    }

    function addHexPrefix(
        bytes memory _path,
        bool _terminator
    ) internal pure returns (bytes memory) {
        bytes1 prefix = _terminator ? bytes1(0x02) : bytes1(0x00);
        bytes memory prefixed = new bytes(2 - _path.length % 2);
        prefixed[0] = prefix;
        return prefixed.concat(_path);
    }

    function removeHexPrefix(
        bytes memory _path
    ) internal pure returns (bytes memory) {
        if (uint8(_path[0]) % 2 == 0) {
            return _path.slice(2);
        } else {
            return _path.slice(1);
        }
    }

    function encodeBranchIndex(
        TrieNode memory _branch,
        bytes1 _index,
        bytes memory _value
    ) internal pure returns (TrieNode memory) {
        bytes[] memory raw = new bytes[](_branch.decoded.length);
        raw[_branch.decoded.length - 1] = RLPWriter.encodeBytes(getNodeValue(_branch));
        for (uint256 i = 0; i < _branch.decoded.length - 1; i++) {
            raw[i] = RLPReader.toRlpBytes(_branch.decoded[i]);
        }
        raw[uint8(_index)] = _value;
        return encodeNode(encodeByteList(raw));
    }

    function encodeBranchValue(
        TrieNode memory _branch,
        bytes memory _value
    ) internal pure returns (TrieNode memory) {
        bytes[] memory raw = new bytes[](_branch.decoded.length);
        raw[_branch.decoded.length - 1] = RLPWriter.encodeBytes(_value);
        for (uint256 i = 0; i < _branch.decoded.length - 1; i++) {
            raw[i] = RLPReader.toRlpBytes(_branch.decoded[i]);
        }
        return encodeNode(encodeByteList(raw));
    }

    function encodeByteList(
        bytes[] memory _raw
    ) internal pure returns (bytes memory) {
        return RLPWriter.encodeList(_raw);
    }

    function getNodePath(
        TrieNode memory _node
    ) internal pure returns (bytes memory) {
        return RLPReader.toBytes(_node.decoded[0]).toNibbles();
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
            if (prefix == PREFIX_EVEN_LEAF || prefix == PREFIX_ODD_LEAF) {
                return NodeType.LeafNode;
            } else if (prefix == PREFIX_EVEN_EXTENSION || prefix == PREFIX_ODD_EXTENSION) {
                return NodeType.ExtensionNode;
            }
        }

        revert("Invalid node type");
    }

    function getNodeID(
        RLPReader.RLPItem memory _item
    ) internal pure returns (bytes memory) {
        if (_item.len < 32) {
            // Nodes smaller than 32 bytes are RLP encoded.
            return RLPReader.toRlpBytes(_item);
        } else {
            // Nodes 32 bytes or larger are hashed.
            return RLPReader.toBytes(_item);
        }
    }

    function copyTrieNodes(
        TrieNode[] memory _nodes,
        uint256 _currentLength,
        uint256 _newLength
    ) internal pure returns (TrieNode[] memory) {
        TrieNode[] memory copy = new TrieNode[](_newLength);

        uint256 smallest = _currentLength < _newLength ? _currentLength : _newLength;
        for (uint256 i = 0; i < smallest; i++) {
            copy[i] = _nodes[i];
        }

        return copy;
    }
}
