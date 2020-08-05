pragma solidity ^0.5.0;

/* Library Imports */
import { MerkleTrie } from "../MerkleTrie.sol";

contract MockMerkleTrie {
    function verifyInclusionProof(
        bytes memory _key,
        bytes memory _value,
        bytes memory _proof,
        bytes32 _root
    )
        public
        pure
        returns (bool)
    {
        return MerkleTrie.verifyInclusionProof(
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
        pure
        returns (bool)
    {
        return MerkleTrie.verifyExclusionProof(
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
        pure
        returns (bytes32)
    {
        return MerkleTrie.update(
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
        pure
        returns (bool, bytes memory)
    {
        return MerkleTrie.get(
            _key,
            _proof,
            _root
        );
    }
}