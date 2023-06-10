// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";
import { LibPosition } from "../dispute/lib/LibPosition.sol";
import "../libraries/DisputeTypes.sol";

/**
 * @notice Tests for `LibPosition`
 */
contract LibPosition_Test is Test {
    /**
     * @dev Assumes a MAX depth of 127 for the Position type. Any greater depth can cause overflows.
     * @dev At the lowest level of the tree, this allows for 2 ** 127 leaves. In reality, the max game depth
     *      will likely be much lower.
     */
    uint8 internal constant MAX_DEPTH = 127;

    /**
     * @notice Tests that the `depth` function correctly shifts out the `depth` from a packed `Position` type.
     */
    function testFuzz_depth_correctness(uint128 _depth, uint128 _indexAtDepth) public {
        Position position = LibPosition.wrap(_depth, _indexAtDepth);
        assertEq(LibPosition.depth(position), _depth);
    }

    /**
     * @notice Tests that the `indexAtDepth` function correctly shifts out the `indexAtDepth` from a packed `Position` type.
     */
    function testFuzz_indexAtDepth_correctness(uint128 _depth, uint128 _indexAtDepth) public {
        Position position = LibPosition.wrap(_depth, _indexAtDepth);
        assertEq(LibPosition.indexAtDepth(position), _indexAtDepth);
    }

    /**
     * @notice Tests that the `left` function correctly computes the position of the left child.
     */
    function testFuzz_left_correctness(uint8 _depth, uint128 _indexAtDepth) public {
        // Depth bound: [0, 127]
        _depth = uint8(bound(_depth, 0, MAX_DEPTH));
        // Index at depth bound: [0, 2 ** _depth]
        _indexAtDepth = uint128(bound(_indexAtDepth, 0, 2 ** _depth));

        Position position = LibPosition.wrap(_depth, _indexAtDepth);
        Position left = LibPosition.left(position);

        assertEq(LibPosition.depth(left), uint128(_depth) + 1);
        assertEq(LibPosition.indexAtDepth(left), _indexAtDepth * 2);
    }

    /**
     * @notice Tests that the `right` function correctly computes the position of the right child.
     */
    function testFuzz_right_correctness(uint8 _depth, uint128 _indexAtDepth) public {
        // Depth bound: [0, 127]
        _depth = uint8(bound(_depth, 0, MAX_DEPTH));
        // Index at depth bound: [0, 2 ** _depth]
        _indexAtDepth = uint128(bound(_indexAtDepth, 0, 2 ** _depth));

        Position position = LibPosition.wrap(_depth, _indexAtDepth);
        Position right = LibPosition.right(position);

        assertEq(LibPosition.depth(right), _depth + 1);
        assertEq(LibPosition.indexAtDepth(right), _indexAtDepth * 2 + 1);
    }

    /**
     * @notice Tests that the `parent` function correctly computes the position of the parent.
     */
    function testFuzz_parent_correctness(uint8 _depth, uint128 _indexAtDepth) public {
        // Depth bound: [1, 127]
        _depth = uint8(bound(_depth, 1, MAX_DEPTH));
        // Index at depth bound: [0, 2 ** _depth]
        _indexAtDepth = uint128(bound(_indexAtDepth, 0, 2 ** _depth));

        Position position = LibPosition.wrap(_depth, _indexAtDepth);
        Position parent = LibPosition.parent(position);

        assertEq(LibPosition.depth(parent), _depth - 1);
        assertEq(LibPosition.indexAtDepth(parent), _indexAtDepth / 2);
    }

    /**
     * @notice Tests that the `rightIndex` function correctly computes the deepest, right most index relative
     * to a given position.
     */
    function testFuzz_rightIndex_correctness(uint8 _maxDepth, uint8 _depth, uint128 _indexAtDepth) public {
        // Max depth bound: [1, 127]
        // The max game depth MUST be at least 1.
        _maxDepth = uint8(bound(_maxDepth, 1, MAX_DEPTH));
        // Depth bound: [0, _maxDepth]
        _depth = uint8(bound(_depth, 0, _maxDepth));
        // Index at depth bound: [0, 2 ** _depth]
        _indexAtDepth = uint128(bound(_indexAtDepth, 0, 2 ** _depth));

        Position position = LibPosition.wrap(_depth, _indexAtDepth);
        uint128 rightIndex = LibPosition.rightIndex(position, _maxDepth);

        // Find the deepest, rightmost index in Solidity rather than Yul
        for (uint256 i = _depth; i < _maxDepth - 1; ++i) {
            position = LibPosition.right(position);
        }
        uint128 _rightIndex = LibPosition.indexAtDepth(position);

        assertEq(rightIndex, _rightIndex);
    }

    /**
     * @notice Tests that the `attack` function correctly computes the position of the attack relative to
     * a given position.
     * @dev `attack` is an alias for `left`, but we test it separately for completeness.
     */
    function testFuzz_attack_correctness(uint8 _depth, uint128 _indexAtDepth) public {
        // Depth bound: [0, 127]
        _depth = uint8(bound(_depth, 0, MAX_DEPTH));
        // Index at depth bound: [0, 2 ** _depth]
        _indexAtDepth = uint128(bound(_indexAtDepth, 0, 2 ** _depth));

        Position position = LibPosition.wrap(_depth, _indexAtDepth);
        Position attack = LibPosition.attack(position);

        assertEq(LibPosition.depth(attack), _depth + 1);
        assertEq(LibPosition.indexAtDepth(attack), _indexAtDepth * 2);
    }

    /**
     * @notice Tests that the `defend` function correctly computes the position of the defense relative to
     * a given position.
     * @dev A defense can only be given if the position does not belong to the root claim, hence the bound of [1, 127]
     * on the depth.
     */
    function testFuzz_defend_correctness(uint8 _depth, uint128 _indexAtDepth) public {
        // Depth bound: [1, 127]
        _depth = uint8(bound(_depth, 1, MAX_DEPTH));
        // Index at depth bound: [0, 2 ** _depth]
        _indexAtDepth = uint128(bound(_indexAtDepth, 0, 2 ** _depth));

        Position position = LibPosition.wrap(_depth, _indexAtDepth);
        Position defend = LibPosition.defend(position);

        assertEq(LibPosition.depth(defend), _depth + 1);
        assertEq(LibPosition.indexAtDepth(defend), ((_indexAtDepth / 2) * 2 + 1) * 2);
    }
}
