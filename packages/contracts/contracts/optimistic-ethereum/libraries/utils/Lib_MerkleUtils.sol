// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/**
 * @title Lib_MerkleUtils
 */
library Lib_MerkleUtils {
    function getMerkleRoot(
        bytes32[] memory _hashes
    )
        internal
        pure
        returns (
            bytes32 _root
        )
    {
        require(
            _hashes.length > 0,
            "Must provide at least one leaf hash."
        );

        if (_hashes.length == 1) {
            return _hashes[0];
        }

        bytes32[] memory defaultHashes = _getDefaultHashes(_hashes.length);

        bytes32[] memory nodes = _hashes;
        if (_hashes.length % 2 == 1) {
            nodes = new bytes32[](_hashes.length + 1);
            for (uint256 i = 0; i < _hashes.length; i++) {
                nodes[i] = _hashes[i];
            }
        }

        uint256 currentLevel = 0;
        uint256 nextLevelSize = _hashes.length;
        
        if (nextLevelSize % 2 == 1) {
            nodes[nextLevelSize] = defaultHashes[currentLevel];
            nextLevelSize += 1;
        }

        while (nextLevelSize > 1) {
            currentLevel += 1;

            for (uint256 i = 0; i < nextLevelSize / 2; i++) {
                nodes[i] = _getParentHash(
                    nodes[i*2],
                    nodes[i*2 + 1]
                );
            }

            nextLevelSize = nextLevelSize / 2;

            if (nextLevelSize % 2 == 1 && nextLevelSize != 1) {
                nodes[nextLevelSize] = defaultHashes[currentLevel];
                nextLevelSize += 1;
            }
        }

        return nodes[0];
    }

    function getMerkleRoot(
        bytes[] memory _elements
    )
        internal
        view
        returns (
            bytes32 _root
        )
    {
        bytes32[] memory hashes = new bytes32[](_elements.length);
        for (uint256 i = 0; i < _elements.length; i++) {
            hashes[i] = keccak256(_elements[i]);
        }

        return getMerkleRoot(hashes);
    }

    function verify(
        bytes32 _root,
        bytes32 _leaf,
        uint256 _path,
        bytes32[] memory _siblings
    )
        internal
        pure
        returns (
            bool _verified
        )
    {
        bytes32 computedRoot = _leaf;

        for (uint256 i = 0; i < _siblings.length; i++) {
            bytes32 sibling = _siblings[i];
            bool isRightSibling = uint8(_path >> i & 1) == 0;

            if (isRightSibling) {
                computedRoot = _getParentHash(computedRoot, sibling);
            } else {
                computedRoot = _getParentHash(sibling, computedRoot);
            }
        }

        return computedRoot == _root;
    }

    function verify(
        bytes32 _root,
        bytes memory _leaf,
        uint256 _path,
        bytes32[] memory _siblings
    )
        internal
        pure
        returns (
            bool _verified
        )
    {
        return verify(
            _root,
            keccak256(_leaf),
            _path,
            _siblings
        );
    }

    function _getDefaultHashes(
        uint256 _length
    )
        private
        pure
        returns (
            bytes32[] memory _defaultHashes
        )
    {
        bytes32[] memory defaultHashes = new bytes32[](_length);

        defaultHashes[0] = keccak256(abi.encodePacked(uint256(0)));
        for (uint256 i = 1; i < defaultHashes.length; i++) {
            defaultHashes[i] = keccak256(abi.encodePacked(
                defaultHashes[i-1],
                defaultHashes[i-1]
            ));
        }

        return defaultHashes;
    }

    function _getParentHash(
        bytes32 _leftChildHash,
        bytes32 _rightChildHash
    )
        private
        pure
        returns (
            bytes32 _hash
        )
    {
        return keccak256(abi.encodePacked(_leftChildHash, _rightChildHash));
    }
}