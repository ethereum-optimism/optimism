// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_MerkleTrie } from "./Lib_MerkleTrie.sol";

/**
 * @title Lib_SecureMerkleTrie
 */
library Lib_SecureMerkleTrie {

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
        public
        view
        returns (bool)
    {
        bytes memory key = _getSecureKey(_key);
        return Lib_MerkleTrie.verifyInclusionProof(key, _value, _proof, _root);
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
        public
        view
        returns (bool)
    {
        bytes memory key = _getSecureKey(_key);
        return Lib_MerkleTrie.verifyExclusionProof(key, _value, _proof, _root);
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
        public
        view
        returns (bytes32)
    {
        bytes memory key = _getSecureKey(_key);
        return Lib_MerkleTrie.update(key, _value, _proof, _root);
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
        public
        view
        returns (bool, bytes memory)
    {
        bytes memory key = _getSecureKey(_key);
        return Lib_MerkleTrie.get(key, _proof, _root);
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
        public
        view
        returns (bytes32)
    {
        bytes memory key = _getSecureKey(_key);
        return Lib_MerkleTrie.getSingleNodeRootHash(key, _value);
    }


    /*********************
     * Private Functions *
     *********************/

    function _getSecureKey(
        bytes memory _key
    ) private pure returns (bytes memory) {
        return abi.encodePacked(keccak256(_key));
    }
}