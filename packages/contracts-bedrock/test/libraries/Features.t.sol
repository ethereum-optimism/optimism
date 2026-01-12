// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Test } from "test/setup/Test.sol";
import { Features } from "src/libraries/Features.sol";

/// @title Features_ValidateCompatibility_Test
/// @notice Tests for the Features library validateCompatibility function
contract Features_ValidateCompatibility_Test is Test {
    /// @notice Tests that validateCompatibility passes for all valid feature combinations
    function test_validateCompatibility_succeeds() external view {
        // Both features disabled
        this._validateCompatibility(false, false);

        // Only custom gas token enabled
        this._validateCompatibility(true, false);

        // Only revenue share enabled
        this._validateCompatibility(false, true);
    }

    /// @notice Tests that validateCompatibility reverts when both features are enabled
    function test_validateCompatibility_bothEnabled_reverts() external {
        vm.expectRevert(Features.Features_CustomGasTokenAndRevenueShareIncompatible.selector);
        this._validateCompatibility(true, true);
    }

    /// @notice External helper function to call Features.validateCompatibility (needed for expectRevert)
    function _validateCompatibility(bool _useCustomGasToken, bool _useRevenueShare) external pure {
        Features.validateCompatibility(_useCustomGasToken, _useRevenueShare);
    }
}
