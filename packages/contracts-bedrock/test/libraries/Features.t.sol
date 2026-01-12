// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Test } from "forge-std/Test.sol";
import { Features } from "src/libraries/Features.sol";

/// @title FeaturesWrapper
/// @notice Wrapper contract to test the Features library
contract FeaturesWrapper {
    function validateCompatibility(bool _useCustomGasToken, bool _useRevenueShare) external pure {
        Features.validateCompatibility(_useCustomGasToken, _useRevenueShare);
    }
}

/// @title FeaturesTest
/// @notice Tests for the Features library
contract FeaturesTest is Test {
    FeaturesWrapper internal wrapper;

    function setUp() public {
        wrapper = new FeaturesWrapper();
    }

    /// @notice Tests that validateCompatibility passes for all valid feature combinations
    function test_validateCompatibility_succeeds() external view {
        // Both features disabled
        wrapper.validateCompatibility(false, false);

        // Only custom gas token enabled
        wrapper.validateCompatibility(true, false);

        // Only revenue share enabled
        wrapper.validateCompatibility(false, true);
    }

    /// @notice Tests that validateCompatibility reverts when both features are enabled
    function test_validateCompatibility_bothEnabled_reverts() external {
        vm.expectRevert(Features.Features_CustomGasTokenAndRevenueShareIncompatible.selector);
        wrapper.validateCompatibility(true, true);
    }
}
