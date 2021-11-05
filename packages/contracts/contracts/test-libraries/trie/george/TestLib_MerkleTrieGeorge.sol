// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_MerkleTrieGeorge } from "../../../optimistic-ethereum/libraries/trie/george/Lib_MerkleTrieGeorge.sol";

/**
 * @title TestLib_MerkleTrieGeorge
 */
contract TestLib_MerkleTrieGeorge {

    // emitting this is how we get the returned root for testing
    event GeorgeHash(bytes32 indexed hash);

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
        bytes32 out = Lib_MerkleTrieGeorge.update(
            _key,
            _value,
            _root
        );
        emit GeorgeHash(out);
        return out;
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
        bytes32 out = Lib_MerkleTrieGeorge.getSingleNodeRootHash(
            _key,
            _value
        );
        emit GeorgeHash(out);
        return out;
    }
}
