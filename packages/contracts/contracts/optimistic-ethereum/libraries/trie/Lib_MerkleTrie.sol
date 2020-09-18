// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/* Library Imports */
import { Lib_ByteUtils } from "../utils/Lib_ByteUtils.sol";
import { Lib_RLPReader } from "../rlp/Lib_RLPReader.sol";
import { Lib_RLPWriter } from "../rlp/Lib_RLPWriter.sol";

/**
 * @title Lib_MerkleTrie
 */
library Lib_MerkleTrie {

    /*******************
     * Data Structures *
     *******************/

    enum NodeType {
        BranchNode,
        ExtensionNode,
        LeafNode
    }

    struct TrieNode {
        bytes encoded;
        Lib_RLPReader.RLPItem[] decoded;
    }


    /**********************
     * Contract Constants *
     **********************/

    // TREE_RADIX determines the number of elements per branch node.
    uint256 constant TREE_RADIX = 16;
    // Branch nodes have TREE_RADIX elements plus an additional `value` slot.
    uint256 constant BRANCH_NODE_LENGTH = TREE_RADIX + 1;
    // Leaf nodes and extension nodes always have two elements, a `path` and a `value`.
    uint256 constant LEAF_OR_EXTENSION_NODE_LENGTH = 2;

    // Prefixes are prepended to the `path` within a leaf or extension node and
    // allow us to differentiate between the two node types. `ODD` or `EVEN` is
    // determined by the number of nibbles within the unprefixed `path`. If the
    // number of nibbles if even, we need to insert an extra padding nibble so
    // the resulting prefixed `path` has an even number of nibbles.
    uint8 constant PREFIX_EXTENSION_EVEN = 0;
    uint8 constant PREFIX_EXTENSION_ODD = 1;
    uint8 constant PREFIX_LEAF_EVEN = 2;
    uint8 constant PREFIX_LEAF_ODD = 3;

    // Just a utility constant. RLP represents `NULL` as 0x80.
    bytes1 constant RLP_NULL = bytes1(0x80);
    bytes constant RLP_NULL_BYTES = hex'80';


    /**********************
     * Internal Functions *
     **********************/

    /**
     * @notice Verifies a proof that a given key/value pair is present in the
     * Merkle trie.
     * @param _key Key of the node to search for, as a hex string.
     * @param _value Value of the node to search for, as a hex string.
     * @param _proof Merkle trie inclusion proof for the desired node. Unlike
     * traditional Merkle trees, this proof is executed top-down and consists
     * of a list of RLP-encoded nodes that make a path down to the target node.
     * @param _root Known root of the Merkle trie. Used to verify that the
     * included proof is correctly constructed.
     * @return `true` if the k/v pair exists in the trie, `false` otherwise.
     */
    function verifyInclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    )
        internal
        view
        returns (
            bool
        )
    {
        return _verifyProof(_key, _value, _proof, _root, true);
    }

    /**
     * @notice Verifies a proof that a given key/value pair is *not* present in
     * the Merkle trie.
     * @param _key Key of the node to search for, as a hex string.
     * @param _value Value of the node to search for, as a hex string.
     * @param _proof Merkle trie inclusion proof for the node *nearest* the
     * target node. We effectively need to show that either the key exists and
     * its value differs, or the key does not exist at all.
     * @param _root Known root of the Merkle trie. Used to verify that the
     * included proof is correctly constructed.
     * @return `true` if the k/v pair is absent in the trie, `false` otherwise.
     */
    function verifyExclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    )
        internal
        view
        returns (
            bool
        )
    {
        return _verifyProof(_key, _value, _proof, _root, false);
    }

    /**
     * @notice Updates a Merkle trie and returns a new root hash.
     * @param _key Key of the node to update, as a hex string.
     * @param _value Value of the node to update, as a hex string.
     * @param _proof Merkle trie inclusion proof for the node *nearest* the
     * target node. If the key exists, we can simply update the value.
     * Otherwise, we need to modify the trie to handle the new k/v pair.
     * @param _root Known root of the Merkle trie. Used to verify that the
     * included proof is correctly constructed.
     * @return Root hash of the newly constructed trie.
     */
    function update(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    )
        internal
        view
        returns (
            bytes32
        )
    {
        TrieNode[] memory proof = _parseProof(_proof);
        (uint256 pathLength, bytes memory keyRemainder, ) = _walkNodePath(proof, _key, _root);

        TrieNode[] memory newPath = _getNewPath(proof, pathLength, keyRemainder, _value);

        return _getUpdatedTrieRoot(newPath, _key);
    }

    /**
     * @notice Retrieves the value associated with a given key.
     * @param _key Key to search for, as hex bytes.
     * @param _proof Merkle trie inclusion proof for the key.
     * @param _root Known root of the Merkle trie.
     * @return Whether the node exists, value associated with the key if so.
     */
    function get(
        bytes memory _key,
        bytes memory _proof,
        bytes32 _root
    )
        internal
        view
        returns (
            bool,
            bytes memory
        )
    {
        TrieNode[] memory proof = _parseProof(_proof);
        (uint256 pathLength, bytes memory keyRemainder, ) = _walkNodePath(proof, _key, _root);

        bool exists = keyRemainder.length == 0;
        bytes memory value = exists ? _getNodeValue(proof[pathLength - 1]) : bytes('');

        return (
            exists,
            value
        );
    }

    /**
     * Computes the root hash for a trie with a single node.
     * @param _key Key for the single node.
     * @param _value Value for the single node.
     * @return Hash of the trie.
     */
    function getSingleNodeRootHash(
        bytes memory _key,
        bytes memory _value
    )
        internal
        view
        returns (
            bytes32
        )
    {
        return keccak256(_makeLeafNode(
            _key,
            _value
        ).encoded);
    }


    /*********************
     * Private Functions *
     *********************/

    /**
     * @notice Utility function that handles verification of inclusion or
     * exclusion proofs. Since the verification methods are almost identical,
     * it's easier to shove this into a single function.
     * @param _key Key of the node to search for, as a hex string.
     * @param _value Value of the node to search for, as a hex string.
     * @param _proof Merkle trie inclusion proof for the node *nearest* the
     * target node. If we're proving explicit inclusion, the nearest node
     * should be the target node.
     * @param _root Known root of the Merkle trie. Used to verify that the
     * included proof is correctly constructed.
     * @param _inclusion Whether to check for inclusion or exclusion.
     * @return `true` if the k/v pair is (in/not in) the trie, `false` otherwise.
     */
    function _verifyProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root,
        bool _inclusion
    )
        private
        view
        returns (
            bool
        )
    {
        TrieNode[] memory proof = _parseProof(_proof);
        (uint256 pathLength, bytes memory keyRemainder, bool isFinalNode) = _walkNodePath(proof, _key, _root);

        if (_inclusion) {
            // Included leaf nodes should have no key remainder, values should match.
            return (
                keyRemainder.length == 0 &&
                Lib_ByteUtils.equal(_getNodeValue(proof[pathLength - 1]), _value)
            );
        } else {
            // If there's no key remainder then a leaf with the given key exists and the value should differ.
            // Otherwise, we need to make sure that we've hit a dead end.
            return (
                (keyRemainder.length == 0 && !Lib_ByteUtils.equal(_getNodeValue(proof[pathLength - 1]), _value)) ||
                (keyRemainder.length != 0 && isFinalNode)
            );
        }
    }

    /**
     * @notice Walks through a proof using a provided key.
     * @param _proof Inclusion proof to walk through.
     * @param _key Key to use for the walk.
     * @param _root Known root of the trie.
     * @return (
     *     Length of the final path;
     *     Portion of the key remaining after the walk;
     *     Whether or not we've hit a dead end;
     * )
     */
    function _walkNodePath(
        TrieNode[] memory _proof,
        bytes memory _key,
        bytes32 _root
    )
        private
        view
        returns (
            uint256,
            bytes memory,
            bool
        )
    {
        uint256 pathLength = 0;
        bytes memory key = Lib_ByteUtils.toNibbles(_key);

        bytes32 currentNodeID = _root;
        uint256 currentKeyIndex = 0;
        uint256 currentKeyIncrement = 0;
        TrieNode memory currentNode;

        // Proof is top-down, so we start at the first element (root).
        for (uint256 i = 0; i < _proof.length; i++) {
            currentNode = _proof[i];
            currentKeyIndex += currentKeyIncrement;

            // Keep track of the proof elements we actually need.
            // It's expensive to resize arrays, so this simply reduces gas costs.
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
                    Lib_ByteUtils.toBytes32(currentNode.encoded) == currentNodeID,
                    "Invalid internal node hash"
                );
            }

            if (currentNode.decoded.length == BRANCH_NODE_LENGTH) {
                if (currentKeyIndex == key.length) {
                    // We've hit the end of the key, meaning the value should be within this branch node.
                    break;
                } else {
                    // We're not at the end of the key yet.
                    // Figure out what the next node ID should be and continue.
                    uint8 branchKey = uint8(key[currentKeyIndex]);
                    Lib_RLPReader.RLPItem memory nextNode = currentNode.decoded[branchKey];
                    currentNodeID = _getNodeID(nextNode);
                    currentKeyIncrement = 1;
                    continue;
                }
            } else if (currentNode.decoded.length == LEAF_OR_EXTENSION_NODE_LENGTH) {
                bytes memory path = _getNodePath(currentNode);
                uint8 prefix = uint8(path[0]);
                uint8 offset = 2 - prefix % 2;
                bytes memory pathRemainder = Lib_ByteUtils.slice(path, offset);
                bytes memory keyRemainder = Lib_ByteUtils.slice(key, currentKeyIndex);
                uint256 sharedNibbleLength = _getSharedNibbleLength(pathRemainder, keyRemainder);

                if (prefix == PREFIX_LEAF_EVEN || prefix == PREFIX_LEAF_ODD) {
                    if (
                        pathRemainder.length == sharedNibbleLength &&
                        keyRemainder.length == sharedNibbleLength
                    ) {
                        // The key within this leaf matches our key exactly.
                        // Increment the key index to reflect that we have no remainder.
                        currentKeyIndex += sharedNibbleLength;
                    }

                    // We've hit a leaf node, so our next node should be NULL.
                    currentNodeID = bytes32(RLP_NULL);
                    break;
                } else if (prefix == PREFIX_EXTENSION_EVEN || prefix == PREFIX_EXTENSION_ODD) {
                    if (sharedNibbleLength == 0) {
                        // Our extension node doesn't share any part of our key.
                        // We've hit the end of this path, updates will need to modify this extension.
                        currentNodeID = bytes32(RLP_NULL);
                        break;
                    } else {
                        // Our extension shares some nibbles.
                        // Carry on to the next node.
                        currentNodeID = _getNodeID(currentNode.decoded[1]);
                        currentKeyIncrement = sharedNibbleLength;
                        continue;
                    }
                }
            }
        }

        // If our node ID is NULL, then we're at a dead end.
        bool isFinalNode = currentNodeID == bytes32(RLP_NULL);
        return (pathLength, Lib_ByteUtils.slice(key, currentKeyIndex), isFinalNode);
    }

    /**
     * @notice Creates new nodes to support a k/v pair insertion into a given
     * Merkle trie path.
     * @param _path Path to the node nearest the k/v pair.
     * @param _pathLength Length of the path. Necessary because the provided
     * path may include additional nodes (e.g., it comes directly from a proof)
     * and we can't resize in-memory arrays without costly duplication.
     * @param _keyRemainder Portion of the initial key that must be inserted
     * into the trie.
     * @param _value Value to insert at the given key.
     * @return A new path with the inserted k/v pair and extra supporting nodes.
     */
    function _getNewPath(
        TrieNode[] memory _path,
        uint256 _pathLength,
        bytes memory _keyRemainder,
        bytes memory _value
    )
        private
        view
        returns (
            TrieNode[] memory
        )
    {
        bytes memory keyRemainder = _keyRemainder;

        // Most of our logic depends on the status of the last node in the path.
        TrieNode memory lastNode = _path[_pathLength - 1];
        NodeType lastNodeType = _getNodeType(lastNode);

        // Create an array for newly created nodes.
        // We need up to three new nodes, depending on the contents of the last node.
        // Since array resizing is expensive, we'll keep track of the size manually.
        // We're using an explicit `totalNewNodes += 1` after insertions for clarity.
        TrieNode[] memory newNodes = new TrieNode[](3);
        uint256 totalNewNodes = 0;

        if (keyRemainder.length == 0 && lastNodeType == NodeType.LeafNode) {
            // We've found a leaf node with the given key.
            // Simply need to update the value of the node to match.
            newNodes[totalNewNodes] = _makeLeafNode(_getNodeKey(lastNode), _value);
            totalNewNodes += 1;
        } else if (lastNodeType == NodeType.BranchNode) {
            if (keyRemainder.length == 0) {
                // We've found a branch node with the given key.
                // Simply need to update the value of the node to match.
                newNodes[totalNewNodes] = _editBranchValue(lastNode, _value);
                totalNewNodes += 1;
            } else {
                // We've found a branch node, but it doesn't contain our key.
                // Reinsert the old branch for now.
                newNodes[totalNewNodes] = lastNode;
                totalNewNodes += 1;
                // Create a new leaf node, slicing our remainder since the first byte points
                // to our branch node.
                newNodes[totalNewNodes] = _makeLeafNode(Lib_ByteUtils.slice(keyRemainder, 1), _value);
                totalNewNodes += 1;
            }
        } else {
            // Our last node is either an extension node or a leaf node with a different key.
            bytes memory lastNodeKey = _getNodeKey(lastNode);
            uint256 sharedNibbleLength = _getSharedNibbleLength(lastNodeKey, keyRemainder);

            if (sharedNibbleLength != 0) {
                // We've got some shared nibbles between the last node and our key remainder.
                // We'll need to insert an extension node that covers these shared nibbles.
                bytes memory nextNodeKey = Lib_ByteUtils.slice(lastNodeKey, 0, sharedNibbleLength);
                newNodes[totalNewNodes] = _makeExtensionNode(nextNodeKey, _getNodeHash(_value));
                totalNewNodes += 1;

                // Cut down the keys since we've just covered these shared nibbles.
                lastNodeKey = Lib_ByteUtils.slice(lastNodeKey, sharedNibbleLength);
                keyRemainder = Lib_ByteUtils.slice(keyRemainder, sharedNibbleLength);
            }

            // Create an empty branch to fill in.
            TrieNode memory newBranch = _makeEmptyBranchNode();

            if (lastNodeKey.length == 0) {
                // Key remainder was larger than the key for our last node.
                // The value within our last node is therefore going to be shifted into
                // a branch value slot.
                newBranch = _editBranchValue(newBranch, _getNodeValue(lastNode));
            } else {
                // Last node key was larger than the key remainder.
                // We're going to modify some index of our branch.
                uint8 branchKey = uint8(lastNodeKey[0]);
                // Move on to the next nibble.
                lastNodeKey = Lib_ByteUtils.slice(lastNodeKey, 1);

                if (lastNodeType == NodeType.LeafNode) {
                    // We're dealing with a leaf node.
                    // We'll modify the key and insert the old leaf node into the branch index.
                    TrieNode memory modifiedLastNode = _makeLeafNode(lastNodeKey, _getNodeValue(lastNode));
                    newBranch = _editBranchIndex(newBranch, branchKey, _getNodeHash(modifiedLastNode.encoded));
                } else if (lastNodeKey.length != 0) {
                    // We're dealing with a shrinking extension node.
                    // We need to modify the node to decrease the size of the key.
                    TrieNode memory modifiedLastNode = _makeExtensionNode(lastNodeKey, _getNodeValue(lastNode));
                    newBranch = _editBranchIndex(newBranch, branchKey, _getNodeHash(modifiedLastNode.encoded));
                } else {
                    // We're dealing with an unnecessary extension node.
                    // We're going to delete the node entirely.
                    // Simply insert its current value into the branch index.
                    newBranch = _editBranchIndex(newBranch, branchKey, _getNodeValue(lastNode));
                }
            }

            if (keyRemainder.length == 0) {
                // We've got nothing left in the key remainder.
                // Simply insert the value into the branch value slot.
                newBranch = _editBranchValue(newBranch, _value);
                // Push the branch into the list of new nodes.
                newNodes[totalNewNodes] = newBranch;
                totalNewNodes += 1;
            } else {
                // We've got some key remainder to work with.
                // We'll be inserting a leaf node into the trie.
                // First, move on to the next nibble.
                keyRemainder = Lib_ByteUtils.slice(keyRemainder, 1);
                // Push the branch into the list of new nodes.
                newNodes[totalNewNodes] = newBranch;
                totalNewNodes += 1;
                // Push a new leaf node for our k/v pair.
                newNodes[totalNewNodes] = _makeLeafNode(keyRemainder, _value);
                totalNewNodes += 1;
            }
        }

        // Finally, join the old path with our newly created nodes.
        // Since we're overwriting the last node in the path, we use `_pathLength - 1`.
        return _joinNodeArrays(_path, _pathLength - 1, newNodes, totalNewNodes);
    }

    /**
     * @notice Computes the trie root from a given path.
     * @param _nodes Path to some k/v pair.
     * @param _key Key for the k/v pair.
     * @return Root hash for the updated trie.
     */
    function _getUpdatedTrieRoot(
        TrieNode[] memory _nodes,
        bytes memory _key
    )
        private
        view
        returns (
            bytes32
        )
    {
        bytes memory key = Lib_ByteUtils.toNibbles(_key);

        // Some variables to keep track of during iteration.
        TrieNode memory currentNode;
        NodeType currentNodeType;
        bytes memory previousNodeHash;

        // Run through the path backwards to rebuild our root hash.
        for (uint256 i = _nodes.length; i > 0; i--) {
            // Pick out the current node.
            currentNode = _nodes[i - 1];
            currentNodeType = _getNodeType(currentNode);

            if (currentNodeType == NodeType.LeafNode) {
                // Leaf nodes are already correctly encoded.
                // Shift the key over to account for the nodes key.
                bytes memory nodeKey = _getNodeKey(currentNode);
                key = Lib_ByteUtils.slice(key, 0, key.length - nodeKey.length);
            } else if (currentNodeType == NodeType.ExtensionNode) {
                // Shift the key over to account for the nodes key.
                bytes memory nodeKey = _getNodeKey(currentNode);
                key = Lib_ByteUtils.slice(key, 0, key.length - nodeKey.length);

                // If this node is the last element in the path, it'll be correctly encoded
                // and we can skip this part.
                if (previousNodeHash.length > 0) {
                    // Re-encode the node based on the previous node.
                    currentNode = _makeExtensionNode(nodeKey, previousNodeHash);
                }
            } else if (currentNodeType == NodeType.BranchNode) {
                // If this node is the last element in the path, it'll be correctly encoded
                // and we can skip this part.
                if (previousNodeHash.length > 0) {
                    // Re-encode the node based on the previous node.
                    uint8 branchKey = uint8(key[key.length - 1]);
                    key = Lib_ByteUtils.slice(key, 0, key.length - 1);
                    currentNode = _editBranchIndex(currentNode, branchKey, previousNodeHash);
                }
            }

            // Compute the node hash for the next iteration.
            previousNodeHash = _getNodeHash(currentNode.encoded);
        }

        // Current node should be the root at this point.
        // Simply return the hash of its encoding.
        return keccak256(currentNode.encoded);
    }

    /**
     * @notice Parses an RLP-encoded proof into something more useful.
     * @param _proof RLP-encoded proof to parse.
     * @return Proof parsed into easily accessible structs.
     */
    function _parseProof(
        bytes memory _proof
    )
        private
        view
        returns (
            TrieNode[] memory
        )
    {
        Lib_RLPReader.RLPItem[] memory nodes = Lib_RLPReader.toList(Lib_RLPReader.toRlpItem(_proof));
        TrieNode[] memory proof = new TrieNode[](nodes.length);

        for (uint256 i = 0; i < nodes.length; i++) {
            bytes memory encoded = Lib_RLPReader.toBytes(nodes[i]);
            proof[i] = TrieNode({
                encoded: encoded,
                decoded: Lib_RLPReader.toList(Lib_RLPReader.toRlpItem(encoded))
            });
        }

        return proof;
    }

    /**
     * @notice Picks out the ID for a node. Node ID is referred to as the
     * "hash" within the specification, but nodes < 32 bytes are not actually
     * hashed.
     * @param _node Node to pull an ID for.
     * @return ID for the node, depending on the size of its contents.
     */
    function _getNodeID(
        Lib_RLPReader.RLPItem memory _node
    )
        private
        view
        returns (
            bytes32
        )
    {
        bytes memory nodeID;

        if (_node.len < 32) {
            // Nodes smaller than 32 bytes are RLP encoded.
            nodeID = Lib_RLPReader.toRlpBytes(_node);
        } else {
            // Nodes 32 bytes or larger are hashed.
            nodeID = Lib_RLPReader.toBytes(_node);
        }

        return Lib_ByteUtils.toBytes32(nodeID);
    }

    /**
     * @notice Gets the path for a leaf or extension node.
     * @param _node Node to get a path for.
     * @return Node path, converted to an array of nibbles.
     */
    function _getNodePath(
        TrieNode memory _node
    )
        private
        view
        returns (
            bytes memory
        )
    {
        return Lib_ByteUtils.toNibbles(Lib_RLPReader.toBytes(_node.decoded[0]));
    }

    /**
     * @notice Gets the key for a leaf or extension node. Keys are essentially
     * just paths without any prefix.
     * @param _node Node to get a key for.
     * @return Node key, converted to an array of nibbles.
     */
    function _getNodeKey(
        TrieNode memory _node
    )
        private
        view
        returns (
            bytes memory
        )
    {
        return _removeHexPrefix(_getNodePath(_node));
    }

    /**
     * @notice Gets the path for a node.
     * @param _node Node to get a value for.
     * @return Node value, as hex bytes.
     */
    function _getNodeValue(
        TrieNode memory _node
    )
        private
        view
        returns (
            bytes memory
        )
    {
        return Lib_RLPReader.toBytes(_node.decoded[_node.decoded.length - 1]);
    }

    /**
     * @notice Computes the node hash for an encoded node. Nodes < 32 bytes
     * are not hashed, all others are keccak256 hashed.
     * @param _encoded Encoded node to hash.
     * @return Hash of the encoded node. Simply the input if < 32 bytes.
     */
    function _getNodeHash(
        bytes memory _encoded
    )
        private
        pure
        returns (
            bytes memory
        )
    {
        if (_encoded.length < 32) {
            return _encoded;
        } else {
            return abi.encodePacked(keccak256(_encoded));
        }
    }

    /**
     * @notice Determines the type for a given node.
     * @param _node Node to determine a type for.
     * @return Type of the node; BranchNode/ExtensionNode/LeafNode.
     */
    function _getNodeType(
        TrieNode memory _node
    )
        private
        view
        returns (
            NodeType
        )
    {
        if (_node.decoded.length == BRANCH_NODE_LENGTH) {
            return NodeType.BranchNode;
        } else if (_node.decoded.length == LEAF_OR_EXTENSION_NODE_LENGTH) {
            bytes memory path = _getNodePath(_node);
            uint8 prefix = uint8(path[0]);

            if (prefix == PREFIX_LEAF_EVEN || prefix == PREFIX_LEAF_ODD) {
                return NodeType.LeafNode;
            } else if (prefix == PREFIX_EXTENSION_EVEN || prefix == PREFIX_EXTENSION_ODD) {
                return NodeType.ExtensionNode;
            }
        }

        revert("Invalid node type");
    }

    /**
     * @notice Utility; determines the number of nibbles shared between two
     * nibble arrays.
     * @param _a First nibble array.
     * @param _b Second nibble array.
     * @return Number of shared nibbles.
     */
    function _getSharedNibbleLength(
        bytes memory _a,
        bytes memory _b
    )
        private
        view
        returns (
            uint256
        )
    {
        uint256 i = 0;
        while (_a.length > i && _b.length > i && _a[i] == _b[i]) {
            i++;
        }
        return i;
    }

    /**
     * @notice Utility; converts an RLP-encoded node into our nice struct.
     * @param _raw RLP-encoded node to convert.
     * @return Node as a TrieNode struct.
     */
    function _makeNode(
        bytes[] memory _raw
    )
        private
        view
        returns (
            TrieNode memory
        )
    {
        bytes memory encoded = Lib_RLPWriter.encodeList(_raw);

        return TrieNode({
            encoded: encoded,
            decoded: Lib_RLPReader.toList(Lib_RLPReader.toRlpItem(encoded))
        });
    }

    /**
     * @notice Utility; converts an RLP-decoded node into our nice struct.
     * @param _items RLP-decoded node to convert.
     * @return Node as a TrieNode struct.
     */
    function _makeNode(
        Lib_RLPReader.RLPItem[] memory _items
    )
        private
        view
        returns (
            TrieNode memory
        )
    {
        bytes[] memory raw = new bytes[](_items.length);
        for (uint256 i = 0; i < _items.length; i++) {
            raw[i] = Lib_RLPReader.toRlpBytes(_items[i]);
        }
        return _makeNode(raw);
    }

    /**
     * @notice Creates a new extension node.
     * @param _key Key for the extension node, unprefixed.
     * @param _value Value for the extension node.
     * @return New extension node with the given k/v pair.
     */
    function _makeExtensionNode(
        bytes memory _key,
        bytes memory _value
    )
        private
        view
        returns (
            TrieNode memory
        )
    {
        bytes[] memory raw = new bytes[](2);
        bytes memory key = _addHexPrefix(_key, false);
        raw[0] = Lib_RLPWriter.encodeBytes(Lib_ByteUtils.fromNibbles(key));
        raw[1] = Lib_RLPWriter.encodeBytes(_value);
        return _makeNode(raw);
    }

    /**
     * @notice Creates a new leaf node.
     * @dev This function is essentially identical to `_makeExtensionNode`.
     * Although we could route both to a single method with a flag, it's
     * more gas efficient to keep them separate and duplicate the logic.
     * @param _key Key for the leaf node, unprefixed.
     * @param _value Value for the leaf node.
     * @return New leaf node with the given k/v pair.
     */
    function _makeLeafNode(
        bytes memory _key,
        bytes memory _value
    )
        private
        view
        returns (
            TrieNode memory
        )
    {
        bytes[] memory raw = new bytes[](2);
        bytes memory key = _addHexPrefix(_key, true);
        raw[0] = Lib_RLPWriter.encodeBytes(Lib_ByteUtils.fromNibbles(key));
        raw[1] = Lib_RLPWriter.encodeBytes(_value);
        return _makeNode(raw);
    }

    /**
     * @notice Creates an empty branch node.
     * @return Empty branch node as a TrieNode stuct.
     */
    function _makeEmptyBranchNode()
        private
        view
        returns (
            TrieNode memory
        )
    {
        bytes[] memory raw = new bytes[](BRANCH_NODE_LENGTH);
        for (uint256 i = 0; i < raw.length; i++) {
            raw[i] = RLP_NULL_BYTES;
        }
        return _makeNode(raw);
    }

    /**
     * @notice Modifies the value slot for a given branch.
     * @param _branch Branch node to modify.
     * @param _value Value to insert into the branch.
     * @return Modified branch node.
     */
    function _editBranchValue(
        TrieNode memory _branch,
        bytes memory _value
    )
        private
        view
        returns (
            TrieNode memory
        )
    {
        bytes memory encoded = Lib_RLPWriter.encodeBytes(_value);
        _branch.decoded[_branch.decoded.length - 1] = Lib_RLPReader.toRlpItem(encoded);
        return _makeNode(_branch.decoded);
    }

    /**
     * @notice Modifies a slot at an index for a given branch.
     * @param _branch Branch node to modify.
     * @param _index Slot index to modify.
     * @param _value Value to insert into the slot.
     * @return Modified branch node.
     */
    function _editBranchIndex(
        TrieNode memory _branch,
        uint8 _index,
        bytes memory _value
    )
        private
        view
        returns (
            TrieNode memory
        )
    {
        bytes memory encoded = _value.length < 32 ? _value : Lib_RLPWriter.encodeBytes(_value);
        _branch.decoded[_index] = Lib_RLPReader.toRlpItem(encoded);
        return _makeNode(_branch.decoded);
    }

    /**
     * @notice Utility; adds a prefix to a key.
     * @param _key Key to prefix.
     * @param _isLeaf Whether or not the key belongs to a leaf.
     * @return Prefixed key.
     */
    function _addHexPrefix(
        bytes memory _key,
        bool _isLeaf
    )
        private
        view
        returns (
            bytes memory
        )
    {
        uint8 prefix = _isLeaf ? uint8(0x02) : uint8(0x00);
        uint8 offset = uint8(_key.length % 2);
        bytes memory prefixed = new bytes(2 - offset);
        prefixed[0] = bytes1(prefix + offset);
        return Lib_ByteUtils.concat(prefixed, _key);
    }

    /**
     * @notice Utility; removes a prefix from a path.
     * @param _path Path to remove the prefix from.
     * @return Unprefixed key.
     */
    function _removeHexPrefix(
        bytes memory _path
    )
        private
        view
        returns (
            bytes memory
        )
    {
        if (uint8(_path[0]) % 2 == 0) {
            return Lib_ByteUtils.slice(_path, 2);
        } else {
            return Lib_ByteUtils.slice(_path, 1);
        }
    }

    /**
     * @notice Utility; combines two node arrays. Array lengths are required
     * because the actual lengths may be longer than the filled lengths.
     * Array resizing is extremely costly and should be avoided.
     * @param _a First array to join.
     * @param _aLength Length of the first array.
     * @param _b Second array to join.
     * @param _bLength Length of the second array.
     * @return Combined node array.
     */
    function _joinNodeArrays(
        TrieNode[] memory _a,
        uint256 _aLength,
        TrieNode[] memory _b,
        uint256 _bLength
    )
        private
        pure
        returns (
            TrieNode[] memory
        )
    {
        TrieNode[] memory ret = new TrieNode[](_aLength + _bLength);

        // Copy elements from the first array.
        for (uint256 i = 0; i < _aLength; i++) {
            ret[i] = _a[i];
        }

        // Copy elements from the second array.
        for (uint256 i = 0; i < _bLength; i++) {
            ret[i + _aLength] = _b[i];
        }

        return ret;
    }
}
