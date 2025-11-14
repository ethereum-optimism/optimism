// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ICreate2Deployer } from "interfaces/preinstalls/ICreate2Deployer.sol";
import { Constants } from "src/libraries/Constants.sol";

/// @title L2ImplementationsDeployer
/// @notice Intermediary contract for deploying predeploy implementations during network upgrades.
contract L2ImplementationsDeployer {
    /// @notice Address of the Create2Deployer preinstall.
    address payable private constant CREATE2_DEPLOYER = payable(0x13b0D85CcB8bf860b6b79AF3029fCA081AE9beF2);

    /// @notice Emitted when an implementation is deployed.
    /// @param implementation The address of the deployed implementation.
    /// @param salt The salt used for deployment.
    event ImplementationDeployed(address indexed implementation, bytes32 salt);

    /// @notice Emitted when deployment is skipped because implementation already exists.
    /// @param implementation The address of the existing implementation.
    event ImplementationExists(address indexed implementation);

    /// @notice Error thrown when caller is not authorized.
    error UnauthorizedCaller();

    /// @notice Modifier to restrict access to depositor account or address(0).
    modifier onlyAuthorized() {
        if (msg.sender != Constants.DEPOSITOR_ACCOUNT && msg.sender != address(0)) {
            revert UnauthorizedCaller();
        }
        _;
    }

    /// @notice Deploys an implementation using CREATE2 if it doesn't already exist.
    /// @param value The amount of ETH to send with the deployment.
    /// @param salt The salt to use for CREATE2 deployment.
    /// @param code The initialization code for the contract.
    /// @return implementation The address of the deployed or existing implementation.
    function deploy(uint256 value, bytes32 salt, bytes memory code) external onlyAuthorized returns (address) {
        // Compute the address where the contract will be deployed
        bytes32 codeHash = keccak256(code);
        address implementation = ICreate2Deployer(CREATE2_DEPLOYER).computeAddress(salt, codeHash);

        // Check if implementation already exists
        if (implementation.code.length != 0) {
            emit ImplementationExists(implementation);
            return implementation;
        }

        // Deploy the implementation
        ICreate2Deployer(CREATE2_DEPLOYER).deploy(value, salt, code);

        emit ImplementationDeployed(implementation, salt);
        return implementation;
    }
}
