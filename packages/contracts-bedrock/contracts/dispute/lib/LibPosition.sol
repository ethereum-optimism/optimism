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
            position_ := add(shl(_depth, 1), _indexAtDepth)
        }
    }

    /**
     * @notice Pulls the `depth` out of a packed `Position` type.
     * @param _position The position to get the `depth` of.
     * @return depth_ The `depth` of the `position`.
     */
    function depth(Position _position) internal pure returns (uint64 depth_) {
        // Return the most significant bit position
        assembly {
            for { } gt(_position, 1) { } {
                depth_ := add(depth_, 1)
                _position := shr(1, _position)
            }
        }
    }

    /**
     * @notice Pulls the `indexAtDepth` out of a packed `Position` type.
     * @param _position The position to get the `indexAtDepth` of.
     * @return indexAtDepth_ The `indexAtDepth` of the `position`.
     */
    function indexAtDepth(Position _position) internal pure returns (uint64 indexAtDepth_) {
        // Return bits p_{msb}...p_{0}
        uint256 msb = depth(_position);
        assembly {
            indexAtDepth_ := sub(_position, shl(msb, 1))
        }
    }

    /**
     * @notice Get the position to the left of `position`.
     * @param _position The position to get the left position of.
     * @return left_ The position to the left of `position`.
     */
    function left(Position _position) internal pure returns (Position left_) {
        assembly {
            left_ := shl(1, _position)
        }
    }

    /**
     * @notice Get the position to the right of `position`.
     * @param _position The position to get the right position of.
     * @return right_ The position to the right of `position`.
     */
    function right(Position _position) internal pure returns (Position right_) {
        assembly {
            right_ := add(1, shl(1, _position))
        }
    }

    /**
     * @notice Get the parent position of `position`.
     * @param _position The position to get the parent position of.
     * @return parent_ The parent position of `position`.
     */
    function parent(Position _position) internal pure returns (Position parent_) {
        assembly {
            parent_ := shr(1, _position)
        }
    }

    /**
     * @notice Get the deepest, right most index relative to the `position`.
     * @param _position The position to get the relative deepest, right most index of.
     * @param _maxDepth The maximum depth of the game.
     * @return rightIndex_ The deepest, right most index relative to the `position`.
     */
    function rightIndex(Position _position, uint256 _maxDepth) internal pure returns (uint64 rightIndex_) {
        uint256 msb = depth(_position);
        assembly {
            let descent := sub(_maxDepth, msb)
            for { let i := 0 } lt(i, descent) { i := add(i, 1) } {
                _position := add(1, shl(1, _position))
            }
            let mask := sub(shl(_maxDepth, 1), 1)
            rightIndex_ := and(_position, mask)
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
        assembly {
            defense_ := shl(1, add(1, shl(1, shr(1, _position))))
        }
    }
}
