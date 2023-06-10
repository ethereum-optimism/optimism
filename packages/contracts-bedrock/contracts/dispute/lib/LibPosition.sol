// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../libraries/DisputeTypes.sol";

/**
 * @title LibPosition
 * @author clabby <https://github.com/clabby>
 * @notice This library contains helper functions for working with the `Position` type.
 */
library LibPosition {
    function wrap(uint128 _depth, uint128 _indexAtDepth) internal pure returns (Position _position) {
        assembly {
            _position := or(shl(0x80, _depth), _indexAtDepth)
        }
    }

    /**
     * @notice Pulls the `depth` out of a packed `Position` type.
     */
    function depth(Position position) internal pure returns (uint128 _depth) {
        // Shift the high-order 128 bits into the low-order 128 bits, leaving only the `depth`.
        assembly {
            _depth := shr(0x80, position)
        }
    }

    /**
     * @notice Pulls the `indexAtDepth` out of a packed `Position` type.
     */
    function indexAtDepth(Position position) internal pure returns (uint128 _indexAtDepth) {
        // Clean the high-order 128 bits by shifting the position left and then right again, leaving
        // only the `indexAtDepth`.
        assembly {
            _indexAtDepth := shr(0x80, shl(0x80, position))
        }
    }

    /**
     * @notice Get the position to the left of `position`.
     * @param position The position to get the left position of.
     * @return _left The position to the left of `position`.
     */
    function left(Position position) internal pure returns (Position _left) {
        uint128 _depth = depth(position);
        uint128 _indexAtDepth = indexAtDepth(position);

        // Left = { depth: position.depth + 1, indexAtDepth: position.indexAtDepth * 2 }
        assembly {
            _left := or(shl(0x80, add(_depth, 0x01)), shl(0x01, _indexAtDepth))
        }
    }

    /**
     * @notice Get the position to the right of `position`.
     * @param position The position to get the right position of.
     * @return _right The position to the right of `position`.
     */
    function right(Position position) internal pure returns (Position _right) {
        uint128 _depth = depth(position);
        uint128 _indexAtDepth = indexAtDepth(position);

        // Right = { depth: position.depth + 1, indexAtDepth: position.indexAtDepth * 2 + 1 }
        assembly {
            _right := or(shl(0x80, add(_depth, 0x01)), add(shl(0x01, _indexAtDepth), 0x01))
        }
    }

    /**
     * @notice Get the parent position of `position`.
     * @param position The position to get the parent position of.
     * @return _parent The parent position of `position`.
     */
    function parent(Position position) internal pure returns (Position _parent) {
        uint128 _depth = depth(position);
        uint128 _indexAtDepth = indexAtDepth(position);

        // Parent = { depth: position.depth - 1, indexAtDepth: position.indexAtDepth / 2 }
        assembly {
            _parent := or(shl(0x80, sub(_depth, 0x01)), shr(0x01, _indexAtDepth))
        }
    }

    /**
     * @notice Get the deepest, right most index relative to the `position`.
     * @param position The position to get the relative deepest, right most index of.
     * @param maxDepth The maximum depth of the game.
     * @return _rightIndex The deepest, right most index relative to the `position`.
     * TODO: Optimize; No need to update the full position in the sub loop.
     */
    function rightIndex(Position position, uint256 maxDepth) internal pure returns (uint128 _rightIndex) {
        assembly {
            _rightIndex := shr(0x80, shl(0x80, position))

            // Walk down to the max depth by moving right
            for { let i := shr(0x80, position) } lt(i, sub(maxDepth, 0x01)) { i := add(i, 0x01) } {
                _rightIndex := add(0x01, shl(0x01, _rightIndex))
            }
        }
    }

    /**
     * @notice Get the attack position relative to `position`.
     */
    function attack(Position position) internal pure returns (Position _attack) {
        return left(position);
    }

    /**
     * @notice Get the defend position of `position`.
     */
    function defend(Position position) internal pure returns (Position _defend) {
        uint128 _depth = depth(position);
        uint128 _indexAtDepth = indexAtDepth(position);

        // Defend = { depth: position.depth + 1, indexAtDepth: ((position.indexAtDepth / 2) * 2 + 1) * 2 }
        assembly {
            _defend := or(shl(0x80, add(_depth, 0x01)), shl(0x01, add(0x01, shl(0x01, shr(0x01, _indexAtDepth)))))
        }
    }
}
