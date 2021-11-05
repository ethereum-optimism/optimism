// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_MerkleTrieGeorge } from "../../../optimistic-ethereum/libraries/trie/george/Lib_MerkleTrieGeorge.sol";

/**
 * @title TestLib_MerkleTrieGeorge
 */
contract TestLib_MerkleTrieGeorge {

    function update(
        bytes memory _key,
        bytes memory _value,
        bytes32 _root
    )
        public
        returns (
            bytes32
        )
    {
        return Lib_MerkleTrieGeorge.update(
            _key,
            _value,
            _root
        );
    }

    function get(
        bytes memory _key,
        bytes32 _root
    )
        public
        view
        returns (
            bool,
            bytes memory
        )
    {
        return Lib_MerkleTrieGeorge.get(
            _key,
            _root
        );
    }

    function getSingleNodeRootHash(
        bytes memory _key,
        bytes memory _value
    )
        public
        returns (
            bytes32
        )
    {
        return Lib_MerkleTrieGeorge.getSingleNodeRootHash(
            _key,
            _value
        );
    }
}
