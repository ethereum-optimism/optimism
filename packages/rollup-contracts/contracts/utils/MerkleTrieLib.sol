pragma solidity ^0.5.0;

import './BytesLib.sol';
import './RLPReader.sol';

/**
 * @notice Library for dealing with Merkle tries.
 */
contract MerkleTrieLib {
    using BytesLib for bytes;

    uint256 constant TREE_RADIX = 16;

    uint8 constant PREFIX_EVEN_EXTENSION = 0;
    uint8 constant PREFIX_ODD_EXTENSION = 1;
    uint8 constant PREFIX_EVEN_LEAF = 2;
    uint8 constant PREFIX_ODD_LEAF = 3;

    struct ProofElement {
        bytes encoded;
        RLPReader.RLPItem[] decoded;
    }

    /**
     * @notice Checks a trie inclusion proof.
     * @param _key Key of the node to verify.
     * @param _value Value of the node to verify.
     * @param _root Root of the trie.
     * @param _proof Encoded proof.
     * @return `true` if the node is in the trie, `false` otherwise.
     */
    function verifyInclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes32 _root,
        bytes memory _proof
    ) public pure returns (bool) {
        RLPReader.RLPItem[] memory proof = RLPReader.toList(RLPReader.toRlpItem(_proof));

        // Convert the key into a series of half-packed bytes.
        bytes memory key = _key.toNibbles();

        bytes32 currentHash = _root;
        uint256 keyIndex = 0;
        for (uint256 i = 0; i < proof.length; i++) {
            ProofElement memory node = ProofElement({
                encoded: RLPReader.toBytes(proof[i]),
                decoded: RLPReader.toList(RLPReader.toRlpItem(RLPReader.toBytes(proof[i])))
            });

            if (keyIndex == 0) {
                // First proof element is always the root node.
                require(
                    keccak256(node.encoded) == currentHash,
                    "Invalid root hash"
                );
            } else if (node.encoded.length >= 32) {
                // Nodes 32 bytes or larger are hashed inside branch nodes.
                require(
                    keccak256(node.encoded) == currentHash,
                    "Invalid large internal hash"
                );
            } else {
                // Nodes smaller than 31 bytes aren't hashed.
                require(
                    node.encoded.toBytes32() == currentHash,
                    "Invalid internal node hash"
                );
            }

            // Nodes with `TREE_RADIX + 1` elements are branches.
            if (node.decoded.length == TREE_RADIX + 1) {
                if (keyIndex >= key.length) {
                    // Value may sometimes be included at a branch node.
                    return (
                        RLPReader.toBytes(node.decoded[node.decoded.length-1]).equal(_value)
                    );
                } else {
                    // Find the next node within the branch node and repeat.
                    RLPReader.RLPItem memory next = node.decoded[uint8(key[keyIndex])];
                    currentHash = getCorrectBytes(next).toBytes32();
                    keyIndex++;
                    continue;
                }
            }

            // Nodes with two elements are either leaves or extensions.
            if (node.decoded.length == 2) {
                // Throw this step into a new function to avoid `STACK_TOO_DEEP`.
                bool done;
                (done, currentHash, keyIndex) = checkNonBranchNode(
                    node,
                    key,
                    _value,
                    keyIndex
                );

                if (done) {
                    return true;
                } else {
                    continue;
                }
            }
        }

        return false;
    }

    function checkNonBranchNode(
        ProofElement memory _node,
        bytes memory _key,
        bytes memory _value,
        uint256 _keyIndex
    ) private pure returns (bool, bytes32, uint256) {
        // First element of the node is its path.
        bytes memory path = RLPReader.toBytes(_node.decoded[0]).toNibbles();
        // First nibble of the path is its prefix.
        uint8 prefix = uint8(path[0]);
        // Even prefixes include an extra nibble.
        uint8 offset = 2 - prefix % 2;

        if (prefix == PREFIX_EVEN_LEAF || prefix == PREFIX_ODD_LEAF) {
            bytes memory value = RLPReader.toBytes(_node.decoded[1]);

            require (
                path.slice(offset).equal(_key.slice(_keyIndex)) &&
                value.equal(_value),
                "Invalid leaf node"
            );

            // Return "done" and fill the rest with empty values.
            return (true, bytes32(0), 0);
        } else if (prefix == PREFIX_EVEN_EXTENSION || prefix == PREFIX_ODD_EXTENSION) {
            bytes memory value = getCorrectBytes(_node.decoded[1]);
            bytes memory shared = path.slice(offset);
            uint256 extension = shared.length;

            require (
                shared.equal(_key.slice(_keyIndex, extension)),
                "Invalid extension node"
            );

            // Return "not done", set the next value, increment the key index.
            return (false, value.toBytes32(), _keyIndex + extension);
        }

        revert("Bad prefix");
    }

    function getCorrectBytes(
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
}
