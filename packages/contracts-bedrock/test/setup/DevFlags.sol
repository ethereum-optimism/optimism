// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { console2 as console } from "forge-std/console2.sol";
import { Vm, VmSafe } from "forge-std/Vm.sol";

// Scripts
import { Deploy } from "scripts/deploy/Deploy.s.sol";

// Libraries
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { LibString } from "@solady/utils/LibString.sol";

/// @notice DevFlags manages the development feature bitmap by either direct user input or via
///         environment variables.
contract DevFlags {
    /// @notice The address of the foundry Vm contract.
    Vm private constant vm = Vm(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    /// @notice The development feature bitmap.
    bytes32 internal devFeatureBitmap;

    /// @notice Resolves the development feature bitmap.
    function resolveFeaturesFromEnv() public {
        if (LibString.eq(vm.envOr("DEV_FEATURE__OPTIMISM_PORTAL_INTEROP", string("0")), "1")) {
            console.log("Setup: DEV_FEATURE__OPTIMISM_PORTAL_INTEROP is enabled");
            devFeatureBitmap |= DevFeatures.OPTIMISM_PORTAL_INTEROP;
        }
    }

    /// @notice Enables a feature.
    /// @param _feature The feature to set.
    function setFeatureEnabled(bytes32 _feature) public {
        devFeatureBitmap |= _feature;
    }

    /// @notice Disables a feature.
    /// @param _feature The feature to set.
    function setFeatureDisabled(bytes32 _feature) public {
        devFeatureBitmap &= ~_feature;
    }

    /// @notice Checks if a development feature is enabled.
    /// @param _feature The feature to check.
    /// @return True if the feature is enabled, false otherwise.
    function isDevFeatureEnabled(bytes32 _feature) public view returns (bool) {
        return DevFeatures.isDevFeatureEnabled(devFeatureBitmap, _feature);
    }

    /// @notice Skips tests when the provided development feature is enabled.
    /// @param _feature The feature to check.
    function skipIfDevFeatureEnabled(bytes32 _feature) public {
        if (isDevFeatureEnabled(_feature)) {
            vm.skip(true);
        }
    }

    /// @notice Skips tests when the provided development feature is disabled.
    /// @param _feature The feature to check.
    function skipIfDevFeatureDisabled(bytes32 _feature) public {
        if (!isDevFeatureEnabled(_feature)) {
            vm.skip(true);
        }
    }
}
