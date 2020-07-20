pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import { MerkleTrie } from "./MerkleTrie.sol";

/**
 * @notice Wrapper around MerkleTrie that hashes keys before they're passed to
 * underlying functions. Necessary for compatibility with Ethereum.
 */
contract SecureMerkleTrie is MerkleTrie {
    /*
     * Public Functions
     */

    function verifyInclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    ) public pure returns (bool) {
        bytes memory key = getSecureKey(_key);
        return super.verifyInclusionProof(key, _value, _proof, _root);
    }

    function verifyExclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    ) public pure returns (bool) {
        bytes memory key = getSecureKey(_key);
        return super.verifyExclusionProof(key, _value, _proof, _root);
    }

    function update(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    ) public pure returns (bytes32) {
        bytes memory key = getSecureKey(_key);
        return super.update(key, _value, _proof, _root);
    }

    function get(
        bytes memory _key,
        bytes memory _proof,
        bytes32 _root
    ) public pure returns (bool, bytes memory) {
        bytes memory key = getSecureKey(_key);
        return super.get(key, _proof, _root);
    }


    /*
     * Internal Functions
     */

    function getSecureKey(
        bytes memory _key
    ) internal pure returns (bytes memory) {
        return abi.encodePacked(keccak256(_key));
    }
}