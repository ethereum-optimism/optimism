pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";

/*
 * Merkle Tree Utilities for Rollup
 */
contract RollupMerkleUtils {
    /* Structs */
    // A partial merkle tree which can be updated with new nodes, recomputing the root
    struct SparseMerkleTree {
        // The root
        bytes32 root;
        uint height;
        mapping (bytes32 => bytes32) nodes;
    }

    /* Fields */
    // The default hashes
    bytes32[160] public defaultHashes;
    // A tree which is used in `update()` and `store()`
    SparseMerkleTree public tree;

    /**
     * @notice Initialize a new SparseMerkleUtils contract, computing the default hashes for the sparse merkle tree (SMT)
     */
    constructor() public {
        // Calculate & set the default hashes
        setDefaultHashes();
    }

    /* Methods */
    /**
     * @notice Set default hashes
     */
    function setDefaultHashes() private {
        // Set the initial default hash.
        defaultHashes[0] = keccak256(abi.encodePacked(uint(0)));
        for (uint i = 1; i < defaultHashes.length; i ++) {
            defaultHashes[i] = keccak256(abi.encodePacked(defaultHashes[i-1], defaultHashes[i-1]));
        }
    }

    /**
     * @notice Get the sparse merkle root computed from some set of data blocks.
     * @param _dataBlocks The data being used to generate the tree.
     * @return the sparse merkle tree root
     */
    function getMerkleRoot(bytes[] calldata _dataBlocks) external view returns(bytes32) {
        uint nextLevelLength = _dataBlocks.length;
        uint currentLevel = 0;
        bytes32[] memory nodes = new bytes32[](nextLevelLength + 1); // Add one in case we have an odd number of leaves
        // Generate the leaves
        for (uint i = 0; i < _dataBlocks.length; i++) {
            nodes[i] = keccak256(_dataBlocks[i]);
        }
        if (_dataBlocks.length == 1) {
            return nodes[0];
        }
        // Add a defaultNode if we've got an odd number of leaves
        if (nextLevelLength % 2 == 1) {
            nodes[nextLevelLength] = defaultHashes[currentLevel];
            nextLevelLength += 1;
        }

        // Now generate each level
        while (nextLevelLength > 1) {
            currentLevel += 1;
            // Calculate the nodes for the currentLevel
            for (uint i = 0; i < nextLevelLength / 2; i++) {
                nodes[i] = getParent(nodes[i*2], nodes[i*2 + 1]);
            }
            nextLevelLength = nextLevelLength / 2;
            // Check if we will need to add an extra node
            if (nextLevelLength % 2 == 1 && nextLevelLength != 1) {
                nodes[nextLevelLength] = defaultHashes[currentLevel];
                nextLevelLength += 1;
            }
        }

        // Alright! We should be left with a single node! Return it...
        return nodes[0];
    }

    /**
     * @notice Verify an inclusion proof.
     * @param _root The root of the tree we are verifying inclusion for.
     * @param _dataBlock The data block we're verifying inclusion for.
     * @param _path The path from the leaf to the root.
     * @param _siblings The sibling nodes along the way.
     * @return The next level of the tree
     */
    function verify(bytes32 _root, bytes memory _dataBlock, uint _path, bytes32[] memory _siblings) public pure returns (bool) {
        // First compute the leaf node
        bytes32 computedNode = keccak256(_dataBlock);
        for (uint i = 0; i < _siblings.length; i++) {
            bytes32 sibling = _siblings[i];
            uint8 isComputedRightSibling = getNthBitFromRight(_path, i);
            if (isComputedRightSibling == 0) {
                computedNode = getParent(computedNode, sibling);
            } else {
                computedNode = getParent(sibling, computedNode);
            }
        }
        // Check if the computed node (_root) is equal to the provided root
        return computedNode == _root;
    }

    /**
     * @notice Update the stored tree / root with a particular dataBlock at some path (no siblings needed)
     * @param _dataBlock The data block we're storing/verifying
     * @param _path The path from the leaf to the root / the index of the leaf.
     */
    function update(bytes memory _dataBlock, uint _path) public {
        bytes32[] memory siblings = getSiblings(_path);
        store(_dataBlock, _path, siblings);
    }

    /**
     * @notice Update the stored tree / root with a particular leaf hash at some path (no siblings needed)
     * @param _leaf The leaf we're storing/verifying
     * @param _path The path from the leaf to the root / the index of the leaf.
     */
    function updateLeaf(bytes32 _leaf, uint _path) public {
        bytes32[] memory siblings = getSiblings(_path);
        storeLeaf(_leaf, _path, siblings);
    }

    /**
     * @notice Store a particular merkle proof & verify that the root did not change.
     * @param _dataBlock The data block we're storing/verifying
     * @param _path The path from the leaf to the root / the index of the leaf.
     * @param _siblings The sibling nodes along the way.
     */
    function verifyAndStore(bytes memory _dataBlock, uint _path, bytes32[] memory _siblings) public {
        bytes32 oldRoot = tree.root;
        store(_dataBlock, _path, _siblings);
        require(tree.root == oldRoot, "Failed same root verification check! This was an inclusion proof for a different tree!");
    }

    /**
     * @notice Store a particular dataBlock & its intermediate nodes in the tree
     * @param _dataBlock The data block we're storing.
     * @param _path The path from the leaf to the root / the index of the leaf.
     * @param _siblings The sibling nodes along the way.
     */
    function store(bytes memory _dataBlock, uint _path, bytes32[] memory _siblings) public {
        // Compute the leaf node & store the leaf
        bytes32 leaf = keccak256(_dataBlock);
        storeLeaf(leaf, _path, _siblings);
    }

    /**
     * @notice Store a particular leaf hash & its intermediate nodes in the tree
     * @param _leaf The leaf we're storing.
     * @param _path The path from the leaf to the root / the index of the leaf.
     * @param _siblings The sibling nodes along the way.
     */
    function storeLeaf(bytes32 _leaf, uint _path, bytes32[] memory _siblings) public {
        // First compute the leaf node
        bytes32 computedNode = _leaf;
        for (uint i = 0; i < _siblings.length; i++) {
            bytes32 parent;
            bytes32 sibling = _siblings[i];
            uint8 isComputedRightSibling = getNthBitFromRight(_path, i);
            if (isComputedRightSibling == 0) {
                parent = getParent(computedNode, sibling);
                // Store the node!
                storeNode(parent, computedNode, sibling);
            } else {
                parent = getParent(sibling, computedNode);
                // Store the node!
                storeNode(parent, sibling, computedNode);
            }
            computedNode = parent;
        }
        // Store the new root
        tree.root = computedNode;
    }

    /**
     * @notice Get siblings for a leaf at a particular index of the tree.
     *         This is used for updates which don't include sibling nodes.
     * @param _path The path from the leaf to the root / the index of the leaf.
     * @return The sibling nodes along the way.
     */
    function getSiblings(uint _path) public view returns (bytes32[] memory) {
        bytes32[] memory siblings = new bytes32[](tree.height);
        bytes32 computedNode = tree.root;
        for(uint i = tree.height; i > 0; i--) {
            uint siblingIndex = i-1;
            (bytes32 leftChild, bytes32 rightChild) = getChildren(computedNode);
            if (getNthBitFromRight(_path, siblingIndex) == 0) {
                computedNode = leftChild;
                siblings[siblingIndex] = rightChild;
            } else {
                computedNode = rightChild;
                siblings[siblingIndex] = leftChild;
            }
        }
        // Now store everything
        return siblings;
    }

    /*********************
     * Utility Functions *
     ********************/
    /**
     * @notice Get our stored tree's root
     * @return The merkle root of the tree
     */
    function getRoot() public view returns(bytes32) {
        return tree.root;
    }

    /**
     * @notice Set the tree root and height of the stored tree
     * @param _root The merkle root of the tree
     * @param _height The height of the tree
     */
    function setMerkleRootAndHeight(bytes32 _root, uint _height) public {
        tree.root = _root;
        tree.height = _height;
    }

    /**
     * @notice Store node in the (in-storage) sparse merkle tree
     * @param _parent The parent node
     * @param _leftChild The left child of the parent in the tree
     * @param _rightChild The right child of the parent in the tree
     */
    function storeNode(bytes32 _parent, bytes32 _leftChild, bytes32 _rightChild) public {
        tree.nodes[getLeftSiblingKey(_parent)] = _leftChild;
        tree.nodes[getRightSiblingKey(_parent)] = _rightChild;
    }

    /**
     * @notice Get the parent of two children nodes in the tree
     * @param _left The left child
     * @param _right The right child
     * @return The parent node
     */
    function getParent(bytes32 _left, bytes32 _right) internal pure returns(bytes32) {
        return keccak256(abi.encodePacked(_left, _right));
    }

    /**
     * @notice get the n'th bit in a uint.
     *         For instance, if exampleUint=binary(11), getNth(exampleUint, 0) == 1, getNth(2, 1) == 1
     * @param _intVal The uint we are extracting a bit out of
     * @param _index The index of the bit we want to extract
     * @return The bit (1 or 0) in a uint8
     */
    function getNthBitFromRight(uint _intVal, uint _index) public pure returns (uint8) {
        return uint8(_intVal >> _index & 1);
    }

    /**
     * @notice Get the children of some parent in the tree
     * @param _parent The parent node
     * @return (rightChild, leftChild) -- the two children of the parent
     */
    function getChildren(bytes32 _parent) public view returns(bytes32, bytes32) {
        return (tree.nodes[getLeftSiblingKey(_parent)], tree.nodes[getRightSiblingKey(_parent)]);
    }

    /**
     * @notice Get the right sibling key. Note that these keys overwrite the first bit of the hash
               to signify if it is on the right side of the parent or on the left
     * @param _parent The parent node
     * @return the key for the left sibling (0 as the first bit)
     */
    function getLeftSiblingKey(bytes32 _parent) public pure returns(bytes32) {
        return _parent & 0x0111111111111111111111111111111111111111111111111111111111111111;
    }

    /**
     * @notice Get the right sibling key. Note that these keys overwrite the first bit of the hash
               to signify if it is on the right side of the parent or on the left
     * @param _parent The parent node
     * @return the key for the right sibling (1 as the first bit)
     */
    function getRightSiblingKey(bytes32 _parent) public pure returns(bytes32) {
        return _parent | 0x1000000000000000000000000000000000000000000000000000000000000000;
    }
}
