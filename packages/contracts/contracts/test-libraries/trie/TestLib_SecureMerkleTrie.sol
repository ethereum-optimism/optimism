// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { Lib_SecureMerkleTrie } from "../../libraries/trie/Lib_SecureMerkleTrie.sol";

/**
 * @title TestLib_SecureMerkleTrie
 */
contract TestLib_SecureMerkleTrie {
    function verifyInclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    ) public pure returns (bool) {
        return Lib_SecureMerkleTrie.verifyInclusionProof(_key, _value, _proof, _root);
    }

    function get(
        bytes memory _key,
        bytes memory _proof,
        bytes32 _root
    ) public pure returns (bool, bytes memory) {
        return Lib_SecureMerkleTrie.get(_key, _proof, _root);
    }
}
