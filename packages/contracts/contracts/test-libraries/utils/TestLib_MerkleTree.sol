// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_MerkleTree } from "../../optimistic-ethereum/libraries/utils/Lib_MerkleTree.sol";

/**
 * @title TestLib_MerkleTree
 */
contract TestLib_MerkleTree {

    function getMerkleRoot(
        bytes32[] memory _elements
    )
        public
       pure
        returns (
            bytes32
        )
    {
        return Lib_MerkleTree.getMerkleRoot(
            _elements
        );
    }

    function verify(
        bytes32 _root,
        bytes32 _leaf,
        uint256 _index,
        bytes32[] memory _siblings,
        uint256 _totalLeaves
    )
        public
        pure
        returns (
            bool
        )
    {
        return Lib_MerkleTree.verify(
            _root,
            _leaf,
            _index,
            _siblings,
            _totalLeaves
        );
    }
}
