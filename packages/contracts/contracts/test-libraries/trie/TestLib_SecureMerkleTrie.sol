// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_SecureMerkleTrie } from "../../optimistic-ethereum/libraries/trie/Lib_SecureMerkleTrie.sol";

/**
 * @title TestLib_SecureMerkleTrie
 */
library TestLib_SecureMerkleTrie {

    function verifyInclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    )
        public
        view
        returns (
            bool
        )
    {
        return Lib_SecureMerkleTrie.verifyInclusionProof(
            _key,
            _value,
            _proof,
            _root
        );
    }

    function verifyExclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    )
        public
        view
        returns (
            bool
        )
    {
        return Lib_SecureMerkleTrie.verifyExclusionProof(
            _key,
            _value,
            _proof,
            _root
        );
    }

    function update(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    )
        public
        view
        returns (
            bytes32
        )
    {
        return Lib_SecureMerkleTrie.update(
            _key,
            _value,
            _proof,
            _root
        );
    }

    function get(
        bytes memory _key,
        bytes memory _proof,
        bytes32 _root
    )
        public
        view
        returns (
            bool,
            bytes memory
        )
    {
        return Lib_SecureMerkleTrie.get(
            _key,
            _proof,
            _root
        );
    }

    function getSingleNodeRootHash(
        bytes memory _key,
        bytes memory _value
    )
        public
        view
        returns (
            bytes32
        )
    {
        return Lib_SecureMerkleTrie.getSingleNodeRootHash(
            _key,
            _value
        );
    }
}
