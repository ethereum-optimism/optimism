// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Deployer } from "./Deployer.sol";
import { DeployConfig } from "./DeployConfig.s.sol";
import { console2 as console } from "forge-std/console2.sol";

import { EAS } from "../src/EAS/EAS.sol";
import { SchemaRegistry } from "../src/EAS/SchemaRegistry.sol";
import { ISchemaRegistry } from "../src/EAS/ISchemaRegistry.sol";
import { Predeploys } from "../src/libraries/Predeploys.sol";

/// @title DeployL2
/// @notice Script used to deploy predeploy implementations to L2.
contract DeployL2 is Deployer {
    DeployConfig cfg;

    /// @notice The name of the script, used to ensure the right deploy artifacts
    ///         are used.
    function name() public pure override returns (string memory) {
        return "DeployL2";
    }

    function setUp() public override {
        super.setUp();

        string memory path = string.concat(vm.projectRoot(), "/deploy-config/", deploymentContext, ".json");
        cfg = new DeployConfig(path);

        console.log("Deploying from %s", deployScript);
        console.log("Deployment context: %s", deploymentContext);
    }

    /// @notice Modifier that wraps a function in broadcasting.
    modifier broadcast() {
        vm.startBroadcast();
        _;
        vm.stopBroadcast();
    }

    /// @notice Deploy the EAS implementation.
    function deployEAS() public broadcast returns (address) {
        EAS eas = new EAS();

        ISchemaRegistry registry = eas.getSchemaRegistry();
        require(address(registry) == Predeploys.SCHEMA_REGISTRY, "EAS: invalid SchemaRegistry address");

        save("EAS", address(eas));
        console.log("EAS deployed at %s", address(eas));

        string memory version = eas.version();
        console.log("EAS version: %s", version);

        return address(eas);
    }

    /// @notice Deploy the SchemaManager implementation.
    function deploySchemaRegistry() public broadcast returns (address) {
        SchemaRegistry registry = new SchemaRegistry();

        save("SchemaRegistry", address(registry));
        console.log("SchemaRegistry deployed at %s", address(registry));

        string memory version = registry.version();
        console.log("SchemaRegistry version: %s", version);

        return address(registry);
    }
}
