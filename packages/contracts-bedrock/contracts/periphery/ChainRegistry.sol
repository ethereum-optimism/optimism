// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/**
 * @title  ChainRegistry
 * @notice ChainRegistry acts as a way to register and query
 *         contract addresses for an OP Stack chain
 */
contract ChainRegistry {

    /**
     * @notice Only the deployment admin can claim a deployment
     */
    error OnlyDeploymentAdmin();

    /**
     * @notice Emitted any time a deployment is claimed
     *
     * @param deployment The name of the deployment claimed
     * @param admin The admin of the deployment claimed
     */
    event DeploymentClaimed(string deployment, address admin);

    /**
     * @notice Emitted any time the admin transfers ownership of a
     *         deployment to a new admin address
     *
     * @param oldAdmin The former admin who made the change
     * @param newAdmin The new admin
     */
    event AdminChanged(address oldAdmin, address newAdmin);

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
     * @param _deployment The deployment to claim
     */
    function claimDeployment(string calldata _deployment, address _admin) public {
        deployments[_deployment] = _admin;

        emit DeploymentClaimed(_deployment, _admin);
    }

    /**
     * @notice Transfers ownership of a deployment to a new admin
     *
     * @param _deployment The deployment to transfer ownership of
     * @param _newAdmin The new admin to transfer ownership to
     */
    function transferAdmin(string calldata _deployment, address _newAdmin) public {
        if (msg.sender != deployments[_deployment]) revert OnlyDeploymentAdmin();
        deployments[_deployment] = _newAdmin;

        emit AdminChanged(msg.sender, _newAdmin);
    }

    /**
     * @notice Registers entries in a deployment
     *
     * @param _deployment The deployment to register entries in
     * @param _entries An array of entries to register
     */
    function register(string calldata _deployment, DeploymentEntry[] calldata _entries) public {
        if (msg.sender != deployments[_deployment]) revert OnlyDeploymentAdmin();
        for (uint i = 0; i < _entries.length; i++) {
            registry[_deployment][_entries[i].entryName] = _entries[i].entryAddress;
        }
    }

    /**
     * @notice Queries the chain registry for a list of deployment addresses
     *
     * @param _deployment The deployment to query
     * @param _names An array of names to query the addresses for
     *
     * @return An array of contract addresses for the names queried
     */
    function query(string calldata _deployment, string[] calldata _names) public view returns (address[] memory) {
        address[] memory addresses = new address[](_names.length);
        for (uint i = 0; i < _names.length; i++) {
            addresses[i] = registry[_deployment][_names[i]];
        }
        return addresses;
    }
}
