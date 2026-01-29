// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { console2 as console } from "forge-std/console2.sol";
import { Vm } from "forge-std/Vm.sol";

// Libraries
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Features } from "src/libraries/Features.sol";
import { Config } from "scripts/libraries/Config.sol";

// Interfaces
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

/// @notice FeatureFlags manages the feature bitmap by either direct user input or via environment
///         variables.
abstract contract FeatureFlags {
    /// @notice The address of the foundry Vm contract.
    Vm private constant vm = Vm(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    /// @notice The development feature bitmap.
    bytes32 internal devFeatureBitmap;

    /// @notice The address of the SystemConfig contract.
    ISystemConfig internal sysCfg;

    /// @notice Thrown when an unknown feature is provided.
    error FeatureFlags_UnknownFeature(bytes32);

    /// @notice Sets the address of the SystemConfig contract.
    /// @param _sysCfg The address of the SystemConfig contract.
    function setSystemConfig(ISystemConfig _sysCfg) public {
        sysCfg = _sysCfg;
    }

    /// @notice Resolves the development feature bitmap.
    /// @dev When updating this function, make sure to also update the getFeatureName function.
    function resolveFeaturesFromEnv() public {
        if (Config.devFeatureInterop()) {
            console.log("Setup: DEV_FEATURE__OPTIMISM_PORTAL_INTEROP is enabled");
            devFeatureBitmap |= DevFeatures.OPTIMISM_PORTAL_INTEROP;
        }
        if (Config.devFeatureOpcmV2()) {
            console.log("Setup: DEV_FEATURE__OPCM_V2 is enabled");
            devFeatureBitmap |= DevFeatures.OPCM_V2;
        }
    }

    /// @notice Returns the string name of a feature.
    /// @param _feature The feature to get the name of.
    /// @return The name of the feature.
    function getFeatureName(bytes32 _feature) public pure returns (string memory) {
        if (_feature == DevFeatures.OPTIMISM_PORTAL_INTEROP) {
            return "DEV_FEATURE__OPTIMISM_PORTAL_INTEROP";
        } else if (_feature == DevFeatures.OPCM_V2) {
            return "DEV_FEATURE__OPCM_V2";
        } else if (_feature == Features.CUSTOM_GAS_TOKEN) {
            return "SYS_FEATURE__CUSTOM_GAS_TOKEN";
        } else if (_feature == Features.ETH_LOCKBOX) {
            return "SYS_FEATURE__ETH_LOCKBOX";
        } else {
            // NOTE: We error out here so that developers remember to actually name their features
            //       above. Solidity doesn't have anything like reflection that could do this.
            revert FeatureFlags_UnknownFeature(_feature);
        }
    }

    /// @notice Enables a feature.
    /// @param _feature The feature to set.
    function setDevFeatureEnabled(bytes32 _feature) public {
        devFeatureBitmap |= _feature;
    }

    /// @notice Disables a feature.
    /// @param _feature The feature to set.
    function setDevFeatureDisabled(bytes32 _feature) public {
        devFeatureBitmap &= ~_feature;
    }

    /// @notice Checks if a system feature is enabled.
    /// @param _feature The feature to check.
    /// @return True if the feature is enabled, false otherwise.
    function isSysFeatureEnabled(bytes32 _feature) public view returns (bool) {
        return sysCfg.isFeatureEnabled(_feature);
    }

    /// @notice Checks if a development feature is enabled.
    /// @param _feature The feature to check.
    /// @return True if the feature is enabled, false otherwise.
    function isDevFeatureEnabled(bytes32 _feature) public view returns (bool) {
        return DevFeatures.isDevFeatureEnabled(devFeatureBitmap, _feature);
    }

    /// @notice Skips tests when the provided system feature is enabled.
    /// @param _feature The feature to check.
    function skipIfSysFeatureEnabled(bytes32 _feature) public {
        if (isSysFeatureEnabled(_feature)) {
            vm.skip(true, string.concat("Skipping test because ", getFeatureName(_feature), " is enabled"));
        }
    }

    /// @notice Skips tests when the provided system feature is disabled.
    /// @param _feature The feature to check.
    function skipIfSysFeatureDisabled(bytes32 _feature) public {
        if (!isSysFeatureEnabled(_feature)) {
            vm.skip(true, string.concat("Skipping test because ", getFeatureName(_feature), " is disabled"));
        }
    }

    /// @notice Skips tests when the provided development feature is enabled.
    /// @param _feature The feature to check.
    function skipIfDevFeatureEnabled(bytes32 _feature) public {
        if (isDevFeatureEnabled(_feature)) {
            vm.skip(true, string.concat("Skipping test because ", getFeatureName(_feature), " is enabled"));
        }
    }

    /// @notice Skips tests when the provided development feature is disabled.
    /// @param _feature The feature to check.
    function skipIfDevFeatureDisabled(bytes32 _feature) public {
        if (!isDevFeatureEnabled(_feature)) {
            vm.skip(true, string.concat("Skipping test because ", getFeatureName(_feature), " is disabled"));
        }
    }
}
