pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { MerkleTrie } from "./MerkleTrie.sol";

/**
* @title SecureMerkleTrie
 * Wrapper around MerkleTrie that hashes keys before they're passed to
 * underlying functions. Necessary for compatibility with Ethereum.
 */
library SecureMerkleTrie {
    /*
     * Internal Functions
     */

    function verifyInclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    )
        internal
        pure
        returns (bool)
    {
        bytes memory key = getSecureKey(_key);
        return MerkleTrie.verifyInclusionProof(key, _value, _proof, _root);
    }

    function verifyExclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    )
        internal
        pure
        returns (bool)
    {
        bytes memory key = getSecureKey(_key);
        return MerkleTrie.verifyExclusionProof(key, _value, _proof, _root);
    }

    function update(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    )
        internal
        pure
        returns (bytes32)
    {
        bytes memory key = getSecureKey(_key);
        return MerkleTrie.update(key, _value, _proof, _root);
    }

    function get(
        bytes memory _key,
        bytes memory _proof,
        bytes32 _root
    )
        internal
        pure
        returns (bool, bytes memory)
    {
        bytes memory key = getSecureKey(_key);
        return MerkleTrie.get(key, _proof, _root);
    }

    function getSingleNodeRootHash(
        bytes memory _key,
        bytes memory _value
    )
        internal
        pure
        returns (bytes32)
    {
        bytes memory key = getSecureKey(_key);
        return MerkleTrie.getSingleNodeRootHash(key, _value);
    }


    /*
     * Private Functions
     */

    function getSecureKey(
        bytes memory _key
    ) private pure returns (bytes memory) {
        return abi.encodePacked(keccak256(_key));
    }
}