// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Forge
import { Script } from "forge-std/Script.sol";

// Scripts
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { ISaferSafes } from "interfaces/safe/ISaferSafes.sol";

// Libraries
import { SemverComp } from "src/libraries/SemverComp.sol";

/// @title DeploySaferSafes
/// @notice Deploys the SaferSafes singleton contract using CREATE2 with the default salt.
contract DeploySaferSafes is Script {
    struct Output {
        ISaferSafes saferSafesSingleton;
    }

    /// @notice Deploys SaferSafes and returns the output struct.
    function run() public returns (Output memory output_) {
        output_ = _deploy();
        assertValidOutput(output_);
    }

    /// @notice Deploys SaferSafes without broadcasting (for use by other scripts).
    function _deploy() internal returns (Output memory output_) {
        output_.saferSafesSingleton = ISaferSafes(
            DeployUtils.createDeterministic({
                _name: "SaferSafes",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(ISaferSafes.__constructor__, ())),
                _salt: DeployUtils.DEFAULT_SALT
            })
        );
        vm.label(address(output_.saferSafesSingleton), "SaferSafesSingleton");
    }

    /// @notice Validates the deployment output.
    function assertValidOutput(Output memory _output) public view {
        DeployUtils.assertValidContractAddress(address(_output.saferSafesSingleton));

        require(SemverComp.eq(_output.saferSafesSingleton.version(), "1.10.1"), "DeploySaferSafes: unexpected version");
    }
}
