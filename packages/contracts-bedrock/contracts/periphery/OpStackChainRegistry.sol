// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/**
 * @title  OpStackChainRegistry
 * @notice OpStackChainRegistry acts as a way to register and query
 *         contract addresses for an OP Stack chain
 */
contract OpStackChainRegistry {

    error OnlyDeploymentAdmin();

    /**
     * @notice Struct representing a deployment
     *         on an OP Stack chain
     */
    struct Deployment {
        string deploymentName;
        address deploymentAdmin;
    }

    /**
     * @notice Struct representing a deployment entry with
     *         a contract's name and associated address
     */
    struct DeploymentEntry {
        string entryName;
        address entryAddress;
    }

    /**
     * @notice Mapping of deployment names to deployment admins
     */
    mapping(string => address) public deployments;

    /**
     * @notice Mapping of deployments to their chain registry
     */
    mapping(string => mapping(string => address)) public registry;

    /**
     * @notice Claims a deployment
     *
     * @param deployment The deployment to claim
     */
    function claimDeployment(Deployment calldata deployment) public {
        deployments[deployment.deploymentName] = deployment.deploymentAdmin;
    }

    /**
     * @notice Registers entries in a deployment
     *
     * @param deployment The deployment to register entries in
     * @param entries An array of entries to register
     */
    function register(Deployment calldata deployment, DeploymentEntry[] calldata entries) public {
        if (msg.sender != deployments[deployment.deploymentName]) revert OnlyDeploymentAdmin();
        for (uint i = 0; i < entries.length; i++) {
            registry[deployment.deploymentName][entries[i].entryName] = entries[i].entryAddress;
        }
    }

    /**
     * @notice Queries the chain registry for a list of deployment addresses
     *
     * @param deployment The deployment to query
     * @param names An array of names to query the addresses for
     */
    function query(Deployment calldata deployment, string[] calldata names) public view returns (address[] memory) {
        address[] memory addresses = new address[](names.length);
        for (uint i = 0; i < names.length; i++) {
            addresses[i] = registry[deployment.deploymentName][names[i]];
        }
        return addresses;
    }
}
