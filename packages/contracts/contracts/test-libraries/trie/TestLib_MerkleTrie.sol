// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { Lib_MerkleTrie } from "../../libraries/trie/Lib_MerkleTrie.sol";

/**
 * @title TestLib_MerkleTrie
 */
contract TestLib_MerkleTrie {
    function verifyInclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    ) public pure returns (bool) {
        return Lib_MerkleTrie.verifyInclusionProof(_key, _value, _proof, _root);
    }

    function get(
        bytes memory _key,
        bytes memory _proof,
        bytes32 _root
    ) public pure returns (bool, bytes memory) {
        return Lib_MerkleTrie.get(_key, _proof, _root);
    }
}
