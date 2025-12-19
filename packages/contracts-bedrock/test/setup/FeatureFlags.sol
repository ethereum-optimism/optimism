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

    /// @notice Mapping of feature bits to their names.
    mapping(bytes32 => string) internal featureNames;

    /// @notice Sets the address of the SystemConfig contract.
    /// @param _sysCfg The address of the SystemConfig contract.
    function setSystemConfig(ISystemConfig _sysCfg) public {
        sysCfg = _sysCfg;
    }

    /// @notice Resolves the development feature bitmap.
    /// @dev When updating this function, make sure to also update the featureNames mapping.
    function resolveFeaturesFromEnv() public {
        if (Config.devFeatureInterop()) {
            console.log("Setup: DEV_FEATURE__OPTIMISM_PORTAL_INTEROP is enabled");
            devFeatureBitmap |= DevFeatures.OPTIMISM_PORTAL_INTEROP;
        }
        if (Config.devFeatureOpcmV2()) {
            console.log("Setup: DEV_FEATURE__OPCM_V2 is enabled");
            devFeatureBitmap |= DevFeatures.OPCM_V2;
        }

        // Map dev feature bits to their names.
        featureNames[DevFeatures.OPTIMISM_PORTAL_INTEROP] = "DEV_FEATURE__OPTIMISM_PORTAL_INTEROP";
        featureNames[DevFeatures.OPCM_V2] = "DEV_FEATURE__OPCM_V2";

        // Map sys feature bits to their names.
        featureNames[Features.CUSTOM_GAS_TOKEN] = "SYS_FEATURE__CUSTOM_GAS_TOKEN";
        featureNames[Features.ETH_LOCKBOX] = "SYS_FEATURE__ETH_LOCKBOX";
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
            vm.skip(true, string.concat("Skipping test because ", featureNames[_feature], " is enabled"));
        }
    }

    /// @notice Skips tests when the provided system feature is disabled.
    /// @param _feature The feature to check.
    function skipIfSysFeatureDisabled(bytes32 _feature) public {
        if (!isSysFeatureEnabled(_feature)) {
            vm.skip(true, string.concat("Skipping test because ", featureNames[_feature], " is disabled"));
        }
    }

    /// @notice Skips tests when the provided development feature is enabled.
    /// @param _feature The feature to check.
    function skipIfDevFeatureEnabled(bytes32 _feature) public {
        if (isDevFeatureEnabled(_feature)) {
            vm.skip(true, string.concat("Skipping test because ", featureNames[_feature], " is enabled"));
        }
    }

    /// @notice Skips tests when the provided development feature is disabled.
    /// @param _feature The feature to check.
    function skipIfDevFeatureDisabled(bytes32 _feature) public {
        if (!isDevFeatureEnabled(_feature)) {
            vm.skip(true, string.concat("Skipping test because ", featureNames[_feature], " is disabled"));
        }
    }
}
