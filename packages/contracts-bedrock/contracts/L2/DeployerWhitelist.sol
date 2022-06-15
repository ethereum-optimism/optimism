// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/**
 * @custom:legacy
 * @custom:proxied
 * @custom:predeploy 0x4200000000000000000000000000000000000002
 *
 * @title DeployerWhitelist
 * @notice The DeployerWhitelist was a legacy predeployed contract used to provide a whitelist of
 *         addresses allowed to deploy smart contracts to Optimism. The DeployerWhitelist was
 *         permanently disabled on the Optimism mainnet shortly after the 2021-11-11 EVM
 *         Equivalence update. After the Bedrock upgrade, the DeployerWhitelist will no longer be
 *         functional at all and will only exist for posterity.
 */
contract DeployerWhitelist {
    /**
     * @notice Emitted whenever the owner of this address is modified.
     *
     * @param oldOwner Old owner of this address.
     * @param newOwner New owner of this address.
     */
    event OwnerChanged(address oldOwner, address newOwner);

    /**
     * @notice Emitted when the whitelist status of a deployer is modified.
     *
     * @param deployer    Address of the deployer.
     * @param whitelisted Whether the deployer is whitelisted.
     */
    event WhitelistStatusChanged(address deployer, bool whitelisted);

    /**
     * @notice Emitted when the whitelist is disabled.
     *
     * @param oldOwner Old owner of the whitelist.
     */
    event WhitelistDisabled(address oldOwner);

    /**
     * @notice Address of the owner of this contract. When this address is address(0) then the
     *         whitelist is fully disabled and cannot be re-enabled.
     */
    address public owner;

    /**
     * @notice Mapping of deployer addresses to their whitelist status.
     */
    mapping(address => bool) public whitelist;

    /**
     * @notice Blocks functions to anyone except the contract owner.
     */
    modifier onlyOwner() {
        require(
            msg.sender == owner,
            "DeployerWhitelist: function can only be called by the owner of this contract."
        );
        _;
    }

    /**
     * @notice Adds or removes an address from the deployment whitelist.
     *
     * @param _deployer      Address to update permissions for.
     * @param _isWhitelisted Whether or not the address is whitelisted.
     */
    function setWhitelistedDeployer(address _deployer, bool _isWhitelisted) external onlyOwner {
        whitelist[_deployer] = _isWhitelisted;
        emit WhitelistStatusChanged(_deployer, _isWhitelisted);
    }

    /**
     * @notice Updates the owner of this contract.
     *
     * @param _owner Address of the new owner.
     */
    function setOwner(address _owner) external onlyOwner {
        // Prevent users from setting the whitelist owner to address(0) except via
        // enableArbitraryContractDeployment. If you want to burn the whitelist owner, send it to
        // any other address that doesn't have a corresponding knowable private key.
        require(
            _owner != address(0),
            "OVM_DeployerWhitelist: can only be disabled via enableArbitraryContractDeployment"
        );

        emit OwnerChanged(owner, _owner);
        owner = _owner;
    }

    /**
     * @notice Permanently enables arbitrary contract deployment and deletes the owner.
     */
    function enableArbitraryContractDeployment() external onlyOwner {
        emit WhitelistDisabled(owner);
        owner = address(0);
    }

    /**
     * @notice Checks whether an address is allowed to deploy contracts.
     *
     * @param _deployer Address to check.
     *
     * @return Whether or not the address can deploy contracts.
     */
    function isDeployerAllowed(address _deployer) external view returns (bool) {
        return (owner == address(0) || whitelist[_deployer]);
    }
}
