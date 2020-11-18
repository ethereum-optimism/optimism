// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_MerkleUtils } from "../../optimistic-ethereum/libraries/utils/Lib_MerkleUtils.sol";

/**
 * @title TestLib_MerkleUtils
 */
contract TestLib_MerkleUtils {

    function getMerkleRoot(
        bytes32[] memory _hashes
    )
        public
        view
        returns (
            bytes32 _root
        )
    {
        return Lib_MerkleUtils.getMerkleRoot(
            _hashes
        );
    }

    function getMerkleRoot(
        bytes[] memory _elements
    )
        public
        view
        returns (
            bytes32 _root
        )
    {
        return Lib_MerkleUtils.getMerkleRoot(
            _elements
        );
    }

    function verify(
        bytes32 _root,
        bytes memory _leaf,
        uint256 _path,
        bytes32[] memory _siblings
    )
        public
        pure
        returns (
            bool _verified
        )
    {
        return Lib_MerkleUtils.verify(
            _root,
            _leaf,
            _path,
            _siblings
        );
    }
}
