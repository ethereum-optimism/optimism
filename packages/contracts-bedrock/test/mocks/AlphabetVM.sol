// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { IBigStepper, IPreimageOracle } from "src/dispute/interfaces/IBigStepper.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import "src/libraries/DisputeTypes.sol";

/// @title AlphabetVM
/// @dev A mock VM for the purpose of testing the dispute game infrastructure.
contract AlphabetVM is IBigStepper {
    Claim internal immutable ABSOLUTE_PRESTATE;
    IPreimageOracle public oracle;

    constructor(Claim _absolutePrestate) {
        ABSOLUTE_PRESTATE = _absolutePrestate;
        oracle = new PreimageOracle();
    }

    /// @inheritdoc IBigStepper
    function step(bytes calldata _stateData, bytes calldata, bytes32) external view returns (bytes32 postState_) {
        uint256 traceIndex;
        uint256 claim;
        if ((keccak256(_stateData) << 8) == (Claim.unwrap(ABSOLUTE_PRESTATE) << 8)) {
            // If the state data is empty, then the absolute prestate is the claim.
            traceIndex = 0;
            (claim) = abi.decode(_stateData, (uint256));
        } else {
            // Otherwise, decode the state data.
            (traceIndex, claim) = abi.decode(_stateData, (uint256, uint256));
            traceIndex++;
        }

        // STF: n -> n + 1
        postState_ = keccak256(abi.encode(traceIndex, claim + 1));
        assembly {
            postState_ := or(and(postState_, not(shl(248, 0xFF))), shl(248, 1))
        }
    }
}
