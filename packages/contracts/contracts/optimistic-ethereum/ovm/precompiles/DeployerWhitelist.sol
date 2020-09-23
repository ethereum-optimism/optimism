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
        public
        onlyOwner
    {
        whitelistedDeployers[_deployerAddress] = _isWhitelisted;
    }

    /**
     * Enables arbitrary contract deployment.
     * This cannot be undone!
     */
    function enableArbitraryContractDeployment()
        public
        onlyOwner
    {
        allowArbitraryDeployment = true;
    }

    /**
     * Returns whether or not the deployer address is allowed to deploy new contracts.
     */
    function isDeployerAllowed(
        address _deployerAddress
    )
        public returns(bool)
    {
        return allowArbitraryDeployment || whitelistedDeployers[_deployerAddress];
    }
}
