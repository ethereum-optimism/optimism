// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../libraries/DisputeTypes.sol";

/**
 * @title Hashing
 * @author clabby <https://github.com/clabby>
 * @notice This library contains all of the hashing utilities used in the Cannon contracts.
 */
library LibHashing {
   /**
    * @notice Hashes a claim and a position together.
    * @param claim A Claim type.
    * @param position The position of `claim`.
    * @return claimHash A hash of abi.encodePacked(claim, position);
    */
    function hashClaimPos(Claim claim, Position position) internal pure returns (ClaimHash claimHash) {
        assembly {
            mstore(0x00, claim)
            mstore(0x20, position)
            claimHash := keccak256(0x00, 0x40)
        }
    }
}
