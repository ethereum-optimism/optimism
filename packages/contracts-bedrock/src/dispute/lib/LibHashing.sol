// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../libraries/DisputeTypes.sol";

/// @title Hashing
/// @notice This library contains all of the hashing utilities used in the Cannon contracts.
library LibHashing {

    /// @notice Hashes a claim and a position together.
    /// @param _claim A Claim type.
    /// @param _position The position of `claim`.
    /// @return claimHash_ A hash of abi.encodePacked(claim, position);
    function hashClaimPos(Claim _claim, Position _position) internal pure returns (ClaimHash claimHash_) {
        assembly {
            mstore(0x00, _claim)
            mstore(0x20, _position)
            claimHash_ := keccak256(0x00, 0x40)
        }
    }
}
