// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Scripts
import { Deployer } from "scripts/deploy/Deployer.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { console2 as console } from "forge-std/console2.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

// Interfaces
import { IL1Block } from "interfaces/L2/IL1Block.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IL2DevFeatureFlags } from "interfaces/L2/IL2DevFeatureFlags.sol";

/// @title ForkL2Live
/// @notice Sets up L2 fork tests by fetching config from the forked L2 chain.
contract ForkL2Live is Deployer {
    /// @notice Whether Custom Gas Token is detected on the forked chain.
    bool public isCustomGasToken;

    /// @notice Whether interop features are detected on the forked chain.
    bool public isInteropEnabled;

    /// @notice Main entry point for L2 fork setup.
    function run() public {
        console.log("ForkL2Live: Starting L2 fork setup");

        // Detect chain features from forked state
        _detectChainFeatures();

        console.log("ForkL2Live: L2 fork setup complete");
    }

    /// @notice Detects chain features from the forked L2 state.
    function _detectChainFeatures() internal {
        // Detect Custom Gas Token
        try IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).isCustomGasToken() returns (bool isCGT_) {
            isCustomGasToken = isCGT_;
            if (isCGT_) {
                console.log("ForkL2Live: Custom Gas Token detected");
            }
        } catch {
            isCustomGasToken = false;
        }

        try IL2DevFeatureFlags(Predeploys.L2_DEV_FEATURE_FLAGS).isDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP)
        returns (bool isInteropEnabled_) {
            isInteropEnabled = isInteropEnabled_;
            console.log("ForkL2Live: Interop features detected");
        } catch {
            isInteropEnabled = false;
        }

        _logPredeployVersion("L1Block", Predeploys.L1_BLOCK_ATTRIBUTES);
        _logPredeployVersion("L2DevFeatureFlags", Predeploys.L2_DEV_FEATURE_FLAGS);
    }

    /// @notice Logs the version of a predeploy contract if available.
    function _logPredeployVersion(string memory _name, address _proxy) internal view {
        // Try to call version() on the proxy
        (bool success, bytes memory data) = _proxy.staticcall(abi.encodeCall(ISemver.version, ()));
        if (success && data.length > 0) {
            string memory version = abi.decode(data, (string));
            console.log("ForkL2Live: %s version %s", _name, version);
        }
    }
}
