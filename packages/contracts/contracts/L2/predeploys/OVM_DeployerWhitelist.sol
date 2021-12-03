// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/**
 * @title OVM_DeployerWhitelist
 * @dev The Deployer Whitelist is a temporary predeploy used to provide additional safety during the
 * initial phases of our mainnet roll out. It is owned by the Optimism team, and defines accounts
 * which are allowed to deploy contracts on Layer2. The Execution Manager will only allow an
 * ovmCREATE or ovmCREATE2 operation to proceed if the deployer's address whitelisted.
 */
contract OVM_DeployerWhitelist {
    /**********
     * Events *
     **********/

    event OwnerChanged(address oldOwner, address newOwner);
    event WhitelistStatusChanged(address deployer, bool whitelisted);
    event WhitelistDisabled(address oldOwner);

    /**********************
     * Contract Constants *
     **********************/

    // WARNING: When owner is set to address(0), the whitelist is disabled.
    address public owner;
    mapping(address => bool) public whitelist;

    /**********************
     * Function Modifiers *
     **********************/

    /**
     * Blocks functions to anyone except the contract owner.
     */
    modifier onlyOwner() {
        require(msg.sender == owner, "Function can only be called by the owner of this contract.");
        _;
    }

    /********************
     * Public Functions *
     ********************/

    /**
     * Adds or removes an address from the deployment whitelist.
     * @param _deployer Address to update permissions for.
     * @param _isWhitelisted Whether or not the address is whitelisted.
     */
    function setWhitelistedDeployer(address _deployer, bool _isWhitelisted) external onlyOwner {
        whitelist[_deployer] = _isWhitelisted;
        emit WhitelistStatusChanged(_deployer, _isWhitelisted);
    }

    /**
     * Updates the owner of this contract.
     * @param _owner Address of the new owner.
     */
    // slither-disable-next-line external-function
    function setOwner(address _owner) public onlyOwner {
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
     * Permanently enables arbitrary contract deployment and deletes the owner.
     */
    function enableArbitraryContractDeployment() external onlyOwner {
        emit WhitelistDisabled(owner);
        owner = address(0);
    }

    /**
     * Checks whether an address is allowed to deploy contracts.
     * @param _deployer Address to check.
     * @return _allowed Whether or not the address can deploy contracts.
     */
    function isDeployerAllowed(address _deployer) external view returns (bool) {
        return (owner == address(0) || whitelist[_deployer]);
    }
}
