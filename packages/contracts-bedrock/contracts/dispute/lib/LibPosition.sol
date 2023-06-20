// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../libraries/DisputeTypes.sol";

/**
 * @title LibPosition
 * @notice This library contains helper functions for working with the `Position` type.
 */
library LibPosition {
    /**
     * @notice Computes a generalized index (2^{depth} + indexAtDepth).
     * @param _depth The depth of the position.
     * @param _indexAtDepth The index at the depth of the position.
     * @return position_ The computed generalized index.
     */
    function wrap(uint64 _depth, uint64 _indexAtDepth) internal pure returns (Position position_) {
        assembly {
            // gindex = 2^{_depth} + _indexAtDepth
            position_ := add(shl(_depth, 1), _indexAtDepth)
        }
    }

    /**
     * @notice Pulls the `depth` out of a `Position` type.
     * @param _position The generalized index to get the `depth` of.
     * @return depth_ The `depth` of the `position` gindex.
     * @custom:attribution Solady <https://github.com/Vectorized/Solady>
     */
    function depth(Position _position) internal pure returns (uint64 depth_) {
        // Return the most significant bit offset, which signifies the depth of the gindex.
        assembly {
            depth_ := or(depth_, shl(6, lt(0xffffffffffffffff, shr(depth_, _position))))
            depth_ := or(depth_, shl(5, lt(0xffffffff, shr(depth_, _position))))

            // For the remaining 32 bits, use a De Bruijn lookup.
            _position := shr(depth_, _position)
            _position := or(_position, shr(1, _position))
            _position := or(_position, shr(2, _position))
            _position := or(_position, shr(4, _position))
            _position := or(_position, shr(8, _position))
            _position := or(_position, shr(16, _position))

            depth_ := or(
                depth_,
                byte(
                    shr(251, mul(_position, shl(224, 0x07c4acdd))),
                    0x0009010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f
                )
            )
        }
    }

    /**
     * @notice Pulls the `indexAtDepth` out of a `Position` type.
     *         The `indexAtDepth` is the left/right index of a position at a specific depth within
     *         the binary tree, starting from index 0. For example, at gindex 2, the `depth` = 1
     *         and the `indexAtDepth` = 0.
     * @param _position The generalized index to get the `indexAtDepth` of.
     * @return indexAtDepth_ The `indexAtDepth` of the `position` gindex.
     */
    function indexAtDepth(Position _position) internal pure returns (uint64 indexAtDepth_) {
        // Return bits p_{msb-1}...p_{0}. This effectively pulls the 2^{depth} out of the gindex,
        // leaving only the `indexAtDepth`.
        uint256 msb = depth(_position);
        assembly {
            indexAtDepth_ := sub(_position, shl(msb, 1))
        }
    }

    /**
     * @notice Get the left child of `_position`.
     * @param _position The position to get the left position of.
     * @return left_ The position to the left of `position`.
     */
    function left(Position _position) internal pure returns (Position left_) {
        assembly {
            left_ := shl(1, _position)
        }
    }

    /**
     * @notice Get the right child of `_position`
     * @param _position The position to get the right position of.
     * @return right_ The position to the right of `position`.
     */
    function right(Position _position) internal pure returns (Position right_) {
        assembly {
            right_ := or(1, shl(1, _position))
        }
    }

    /**
     * @notice Get the parent position of `_position`.
     * @param _position The position to get the parent position of.
     * @return parent_ The parent position of `position`.
     */
    function parent(Position _position) internal pure returns (Position parent_) {
        assembly {
            parent_ := shr(1, _position)
        }
    }

    /**
     * @notice Get the deepest, right most gindex relative to the `position`. This is equivalent to
     *         calling `right` on a position until the maximum depth is reached.
     * @param _position The position to get the relative deepest, right most gindex of.
     * @param _maxDepth The maximum depth of the game.
     * @return rightIndex_ The deepest, right most gindex relative to the `position`.
     */
    function rightIndex(
        Position _position,
        uint256 _maxDepth
    ) internal pure returns (Position rightIndex_) {
        uint256 msb = depth(_position);
        assembly {
            switch eq(msb, _maxDepth)
            case true {
                rightIndex_ := _position
            }
            default {
                let remaining := sub(_maxDepth, msb)
                rightIndex_ := or(shl(remaining, _position), sub(shl(remaining, 1), 1))
            }
        }
    }

    /**
     * @notice Get the attack position relative to `position`. The attack position is the next
     *         logical point of bisection if the parent claim is disagreed with, which is the
     *         midway point of the trace that the attacked node commits to.
     * @param _position The position to get the relative attack position of.
     * @return attack_ The attack position relative to `position`.
     */
    function attack(Position _position) internal pure returns (Position attack_) {
        // Move: Left
        return left(_position);
    }

    /**
     * @notice Get the defense position relative to `position`. The defense position is the next
     *         logical point of bisection if the parent claim and the grandparent claim are agreed
     *         with, which is at the midway point of the trace that the defended node's right
     *         sibling commits to.
     * @param _position The position to get the relative defense position of.
     * @return defense_ The defense position relative to `position`.
     */
    function defend(Position _position) internal pure returns (Position defense_) {
        assembly {
            // Move: Parent -> Right -> Left
            defense_ := shl(1, or(1, _position))
        }
    }
}
