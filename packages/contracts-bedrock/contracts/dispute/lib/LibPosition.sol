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
     * @custom:attribution Solady <https://github.com/Vectorized/Solady>
     */
    function depth(Position _position) internal pure returns (uint64 depth_) {
        // Return the most significant bit position
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
     * @notice Pulls the `indexAtDepth` out of a packed `Position` type.
     * @param _position The position to get the `indexAtDepth` of.
     * @return indexAtDepth_ The `indexAtDepth` of the `position`.
     */
    function indexAtDepth(Position _position) internal pure returns (uint64 indexAtDepth_) {
        // Return bits p_{msb-1}...p_{0}
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
    function rightIndex(
        Position _position,
        uint256 _maxDepth
    ) internal pure returns (uint64 rightIndex_) {
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
            rightIndex_ := sub(rightIndex_, shl(_maxDepth, 1))
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
