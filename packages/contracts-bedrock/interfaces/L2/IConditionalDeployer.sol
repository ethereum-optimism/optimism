// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title IConditionalDeployer
/// @notice Interface for the ConditionalDeployer contract.
interface IConditionalDeployer is ISemver {
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

    /// @notice Deploys an implementation using CREATE2 if it doesn't already exist.
    /// @param _salt The salt to use for CREATE2 deployment.
    /// @param _code The initialization code for the contract.
    /// @return implementation_ The address of the deployed or existing implementation.
    function deploy(bytes32 _salt, bytes memory _code) external returns (address implementation_);

    /// @notice Address of the Arachnid's DeterministicDeploymentProxy.
    /// @return deterministicDeploymentProxy_ The address of the Arachnid's DeterministicDeploymentProxy.
    function deterministicDeploymentProxy() external pure returns (address deterministicDeploymentProxy_);
}
