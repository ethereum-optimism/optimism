pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title DeployerWhitelist
 */
contract DeployerWhitelist {
    mapping(address=>bool) public whitelistedDeployers;
    address public owner;
    bool public allowArbitraryDeployment;

    constructor(address _owner, bool _allowArbitraryDeployment)
        public
    {
        owner = _owner; 
        allowArbitraryDeployment = _allowArbitraryDeployment; 
    }

    /*
     * Modifiers
     */
    // Source: https://solidity.readthedocs.io/en/v0.5.3/contracts.html
    modifier onlyOwner {
        require(
            msg.sender == owner,
            "Only owner can call this function."
        );
        _;
    }

    /*
     * Public Functions
     */

    /**
     * Sets a whitelisted deployer.
     */
    function setWhitelistedDeployer(
        address _deployerAddress,
        bool _isWhitelisted
    )
        external
        onlyOwner
    {
        whitelistedDeployers[_deployerAddress] = _isWhitelisted;
    }

    /**
     * Set owner of the contract.
     */
    function setOwner(
        address _newOwner
    )
        external
        onlyOwner
    {
        owner = _newOwner;
    }

    /**
     * Set allowArbitraryDeployment which if enabled allows anyone to deploy.
     */
    function setAllowArbitraryDeployment(
        bool _allowArbitraryDeployment
    )
        external
        onlyOwner
    {
        allowArbitraryDeployment = _allowArbitraryDeployment;
    }

    /**
     * Enables arbitrary contract deployment.
     * This cannot be undone!
     */
    function enableArbitraryContractDeployment()
        external
        onlyOwner
    {
        // Allow anyone to deploy and then burn the owner address!
        allowArbitraryDeployment = true;
        owner = address(0);
    }

    /**
     * Returns whether or not the deployer address is allowed to deploy new contracts.
     */
    function isDeployerAllowed(
        address _deployerAddress
    )
        external
        view
        returns(bool)
    {
        return allowArbitraryDeployment || whitelistedDeployers[_deployerAddress];
    }
}
