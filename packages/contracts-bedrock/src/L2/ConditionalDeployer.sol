// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @custom:proxied true
/// @custom:predeploy 0x420000000000000000000000000000000000002C
/// @title ConditionalDeployer
/// @notice ConditionalDeployer is used to deploy implementations for predeploys during network upgrades.
///         It uses Arachnid's DeterministicDeploymentProxy to deploy the implementations.
contract ConditionalDeployer is ISemver {
    /// @notice Emitted when an implementation is deployed.
    /// @param implementation The address of the deployed implementation.
    /// @param salt The salt used for deployment.
    event ImplementationDeployed(address indexed implementation, bytes32 salt);

    /// @notice Emitted when deployment is skipped because implementation already exists.
    /// @param implementation The address of the existing implementation.
    event ImplementationExists(address indexed implementation);

    /// @notice Error thrown when deployment fails.
    /// @param data The data returned from the deployment call.
    error ConditionalDeployer_DeploymentFailed(bytes data);

    /// @notice Address of the Arachnid DeterministicDeploymentProxy.
    address payable internal constant DETERMINISTIC_DEPLOYMENT_PROXY =
        payable(0x4e59b44847b379578588920cA78FbF26c0B4956C);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Deploys an implementation using CREATE2 if it doesn't already exist.
    /// @dev Does not support deployments requiring ETH.
    /// @dev Reverts when the deployment call to the DeterministicDeploymentProxy fails.
    /// @param _salt The salt to use for CREATE2 deployment.
    /// @param _code The initialization code for the contract.
    /// @return implementation_ The address of the deployed or existing implementation.
    function deploy(bytes32 _salt, bytes memory _code) external returns (address implementation_) {
        // Compute the address where the contract will be deployed using CREATE2 formula
        bytes32 codeHash = keccak256(_code);
        address expectedImplementation = address(
            uint160(uint256(keccak256(abi.encodePacked(bytes1(0xff), DETERMINISTIC_DEPLOYMENT_PROXY, _salt, codeHash))))
        );

        // Check if implementation already exists
        if (expectedImplementation.code.length != 0) {
            emit ImplementationExists(expectedImplementation);
            return expectedImplementation;
        }

        // Deploy using Arachnid's DeterministicDeploymentProxy
        // Calldata format: salt + initcode
        // Returns: raw 20 bytes (deployed address, not ABI-encoded)
        (bool success, bytes memory data) = DETERMINISTIC_DEPLOYMENT_PROXY.call(abi.encodePacked(_salt, _code));

        // Decode the returned address (raw 20 bytes)
        implementation_ = address(bytes20(data));
        if (!success || implementation_ != expectedImplementation) {
            revert ConditionalDeployer_DeploymentFailed(data);
        }

        emit ImplementationDeployed(implementation_, _salt);
    }

    /// @notice Returns the address of the Arachnid's DeterministicDeploymentProxy.
    /// @return deterministicDeploymentProxy_ The address of the Arachnid's DeterministicDeploymentProxy.
    function deterministicDeploymentProxy() external pure returns (address deterministicDeploymentProxy_) {
        deterministicDeploymentProxy_ = DETERMINISTIC_DEPLOYMENT_PROXY;
    }
}
