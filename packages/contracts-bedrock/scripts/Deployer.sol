// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";
import { Artifacts } from "scripts/Artifacts.s.sol";
import { Config } from "scripts/Config.sol";
import { DeployConfig } from "scripts/DeployConfig.s.sol";
import { USE_FAULT_PROOFS_SLOT } from "scripts/DeployConfig.s.sol";

/// @title Deployer
/// @author tynes
/// @notice A contract that can make deploying and interacting with deployments easy.
abstract contract Deployer is Script, Artifacts {
    DeployConfig public constant cfg =
        DeployConfig(address(uint160(uint256(keccak256(abi.encode("optimism.deployconfig"))))));

    /// @notice Sets up the artifacts contract.
    function setUp() public virtual override {
        Artifacts.setUp();

        // Load the `useFaultProofs` slot value prior to etching the DeployConfig's bytecode and reading the deploy
        // config file. If this slot has already been set, it will override the preference in the deploy config.
        bytes32 useFaultProofsOverride = vm.load(address(cfg), USE_FAULT_PROOFS_SLOT);

        vm.etch(address(cfg), vm.getDeployedCode("DeployConfig.s.sol:DeployConfig"));
        vm.label(address(cfg), "DeployConfig");
        vm.allowCheatcodes(address(cfg));
        cfg.read(Config.deployConfigPath());

        if (useFaultProofsOverride != 0) {
            vm.store(address(cfg), USE_FAULT_PROOFS_SLOT, useFaultProofsOverride);
        }
    }

    /// @notice Returns the name of the deployment script. Children contracts
    ///         must implement this to ensure that the deploy artifacts can be found.
    ///         This should be the same as the name of the script and is used as the file
    ///         name inside of the `broadcast` directory when looking up deployment artifacts.
    function name() public pure virtual returns (string memory);
}
