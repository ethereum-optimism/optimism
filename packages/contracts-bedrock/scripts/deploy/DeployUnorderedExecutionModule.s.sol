// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Scripts
import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Contracts
import { UnorderedExecutionModule } from "src/safe/UnorderedExecutionModule.sol";

/// @notice Deploys UnorderedExecutionModule using CREATE2 for deterministic addresses
/// @dev This script deploys the UnorderedExecutionModule contract that allows for
///      unordered execution of transactions on Safe contracts using replay protection
///      via transaction hashes instead of sequential nonces.
contract DeployUnorderedExecutionModule is Script {
    /// @notice Salt for CREATE2 deployment - ensures deterministic addresses
    bytes32 internal constant SALT = keccak256("UnorderedExecutionModule-v1.0.0");

    /// @notice Main deployment function using CREATE2
    /// @dev Uses CREATE2 for deterministic deployment across different chains
    /// @return moduleAddr_ Address of the deployed UnorderedExecutionModule
    function run() external returns (address moduleAddr_) {
        return deployUnorderedExecutionModule();
    }

    /// @notice Deploy UnorderedExecutionModule using CREATE2
    /// @dev No constructor arguments needed for UnorderedExecutionModule
    /// @return moduleAddr_ Address of the deployed contract
    function deployUnorderedExecutionModule() public returns (address moduleAddr_) {
        // Pre-compute the address for logging
        bytes memory initCode = abi.encodePacked(vm.getCode("UnorderedExecutionModule"));
        address preComputedAddress = vm.computeCreate2Address(SALT, keccak256(initCode));

        console.log("Deploying UnorderedExecutionModule with CREATE2");
        console.log("Salt: %s", vm.toString(SALT));
        console.log("Expected address: %s", preComputedAddress);

        // Check if contract already exists at computed address
        if (preComputedAddress.code.length > 0) {
            console.log("UnorderedExecutionModule already deployed at %s", preComputedAddress);
            return preComputedAddress;
        }

        // Confirm that there is code at the Create2Deployer address (0x4e59b44847b379578588920cA78FbF26c0B4956C)
        address create2Deployer = 0x4e59b44847b379578588920cA78FbF26c0B4956C;
        require(create2Deployer.code.length > 0, "Create2Deployer not deployed");

        // Deploy using CREATE2
        vm.broadcast();
        moduleAddr_ = DeployUtils.create2(
            "UnorderedExecutionModule",
            "", // No constructor args needed
            SALT
        );

        console.log("UnorderedExecutionModule deployed at: %s", moduleAddr_);

        // Verify deployment
        require(moduleAddr_ == preComputedAddress, "Deployment address mismatch");
        require(moduleAddr_.code.length > 0, "Deployment failed - no code at address");

        console.log("Deployment successful and verified");
        return moduleAddr_;
    }

    /// @notice Helper function to compute the deterministic address
    /// @dev Can be called to check where the contract will be deployed
    /// @return Expected deployment address
    function getDeploymentAddress() external view returns (address) {
        bytes memory initCode = abi.encodePacked(vm.getCode("UnorderedExecutionModule"));
        return vm.computeCreate2Address(SALT, keccak256(initCode));
    }

    /// @notice Alternative deployment with custom salt
    /// @param _customSalt Custom salt for CREATE2 deployment
    /// @return moduleAddr_ Address of the deployed contract
    function deployWithCustomSalt(bytes32 _customSalt) external returns (address moduleAddr_) {
        console.log("Deploying UnorderedExecutionModule with custom salt: %s", vm.toString(_customSalt));

        vm.broadcast();
        moduleAddr_ = DeployUtils.create2(
            "UnorderedExecutionModule",
            "", // No constructor args needed
            _customSalt
        );

        console.log("UnorderedExecutionModule deployed at: %s", moduleAddr_);
        return moduleAddr_;
    }
}
