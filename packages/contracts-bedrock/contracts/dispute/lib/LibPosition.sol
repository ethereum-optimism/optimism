// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../libraries/DisputeTypes.sol";

/**
 * @title LibPosition
 * @notice This library contains helper functions for working with the `Position` type.
 */
library LibPosition {
    function wrap(uint64 _depth, uint64 _indexAtDepth) internal pure returns (Position position_) {
        assembly {
            position_ := or(shl(0x40, _depth), _indexAtDepth)
        }
    }

    /**
     * @notice Pulls the `depth` out of a packed `Position` type.
     * @param _position The position to get the `depth` of.
     * @return depth_ The `depth` of the `position`.
     */
    function depth(Position _position) internal pure returns (uint64 depth_) {
        // Shift the high-order 64 bits into the low-order 64 bits, leaving only the `depth`.
        assembly {
            depth_ := shr(0x40, _position)
        }
    }

    /**
     * @notice Pulls the `indexAtDepth` out of a packed `Position` type.
     * @param _position The position to get the `indexAtDepth` of.
     * @return indexAtDepth_ The `indexAtDepth` of the `position`.
     */
    function indexAtDepth(Position _position) internal pure returns (uint64 indexAtDepth_) {
        // Clean the high-order 192 bits by shifting the position left and then right again, leaving
        // only the `indexAtDepth`.
        assembly {
            indexAtDepth_ := shr(0xC0, shl(0xC0, _position))
        }
    }

    /**
     * @notice Get the position to the left of `position`.
     * @param _position The position to get the left position of.
     * @return left_ The position to the left of `position`.
     */
    function left(Position _position) internal pure returns (Position left_) {
        uint64 _depth = depth(_position);
        uint64 _indexAtDepth = indexAtDepth(_position);

        // Left = { depth: position.depth + 1, indexAtDepth: position.indexAtDepth * 2 }
        assembly {
            left_ := or(shl(0x40, add(_depth, 0x01)), shl(0x01, _indexAtDepth))
        }
    }

    /**
     * @notice Get the position to the right of `position`.
     * @param _position The position to get the right position of.
     * @return right_ The position to the right of `position`.
     */
    function right(Position _position) internal pure returns (Position right_) {
        uint64 _depth = depth(_position);
        uint64 _indexAtDepth = indexAtDepth(_position);

        // Right = { depth: position.depth + 1, indexAtDepth: position.indexAtDepth * 2 + 1 }
        assembly {
            right_ := or(shl(0x40, add(_depth, 0x01)), add(shl(0x01, _indexAtDepth), 0x01))
        }
    }

    /**
     * @notice Get the parent position of `position`.
     * @param _position The position to get the parent position of.
     * @return parent_ The parent position of `position`.
     */
    function parent(Position _position) internal pure returns (Position parent_) {
        uint64 _depth = depth(_position);
        uint64 _indexAtDepth = indexAtDepth(_position);

        // Parent = { depth: position.depth - 1, indexAtDepth: position.indexAtDepth / 2 }
        assembly {
            parent_ := or(shl(0x40, sub(_depth, 0x01)), shr(0x01, _indexAtDepth))
        }
    }

    /**
     * @notice Get the deepest, right most index relative to the `position`.
     * @param _position The position to get the relative deepest, right most index of.
     * @param _maxDepth The maximum depth of the game.
     * @return rightIndex_ The deepest, right most index relative to the `position`.
     */
    function rightIndex(Position _position, uint256 _maxDepth) internal pure returns (uint64 rightIndex_) {
        assembly {
            rightIndex_ := shr(0xC0, shl(0xC0, _position))

            // Walk down to the max depth by moving right
            for { let i := shr(0x40, _position) } lt(i, sub(_maxDepth, 0x01)) { i := add(i, 0x01) } {
                rightIndex_ := add(0x01, shl(0x01, rightIndex_))
            }
        }
    }

    /**
     * @notice Get the attack position relative to `position`.
     * @param _position The position to get the relative attack position of.
     * @return attack_ The attack position relative to `position`.
     */
    function attack(Position _position) internal pure returns (Position attack_) {
        return left(_position);
    }

    /**
     * @notice Get the defense position relative to `position`.
     * @param _position The position to get the relative defense position of.
     * @return defense_ The defense position relative to `position`.
     */
    function defend(Position _position) internal pure returns (Position defense_) {
        uint64 _depth = depth(_position);
        uint64 _indexAtDepth = indexAtDepth(_position);

        // Defend = { depth: position.depth + 1, indexAtDepth: ((position.indexAtDepth / 2) * 2 + 1) * 2 }
        assembly {
            defense_ := or(shl(0x40, add(_depth, 0x01)), shl(0x01, add(0x01, shl(0x01, shr(0x01, _indexAtDepth)))))
        }
    }
}
