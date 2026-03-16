// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Scripts
import { Deployer } from "scripts/deploy/Deployer.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { console2 as console } from "forge-std/console2.sol";

// Interfaces
import { IL1Block } from "interfaces/L2/IL1Block.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

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

        // Detect interop by checking if CrossL2Inbox implementation has code
        address crossL2InboxImpl = EIP1967Helper.getImplementation(Predeploys.CROSS_L2_INBOX);
        isInteropEnabled = crossL2InboxImpl.code.length > 0;
        if (isInteropEnabled) {
            console.log("ForkL2Live: Interop features detected");
        }

        // Log contract versions for key predeploys
        _logPredeployVersion("CrossL2Inbox", Predeploys.CROSS_L2_INBOX);
        _logPredeployVersion("L1Block", Predeploys.L1_BLOCK_ATTRIBUTES);
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
