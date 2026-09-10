// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Features } from "src/libraries/Features.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IL1Block } from "interfaces/L2/IL1Block.sol";

/// @notice Contract-level authorization for private-chain public projections.
library PrivateProjection {
    /// @notice Only the current batcher may execute projection protocol calls.
    error PrivateProjection_NotBatcher();

    /// @notice Enforces batcher authorization when the projection feature is enabled.
    /// @dev This checks the immediate caller, including for calls forwarded by other contracts.
    ///      It does not distinguish transaction types or alter deposit execution semantics.
    function requireBatcher() internal view {
        IL1Block attributes = IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES);
        if (
            attributes.isFeatureEnabled(Features.PRIVATE_PROJECTION)
                && bytes32(uint256(uint160(msg.sender))) != attributes.batcherHash()
        ) {
            revert PrivateProjection_NotBatcher();
        }
    }
}
