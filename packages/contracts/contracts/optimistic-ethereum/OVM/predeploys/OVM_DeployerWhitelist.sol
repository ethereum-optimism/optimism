// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Interface Imports */
import { iOVM_DeployerWhitelist } from "../../iOVM/predeploys/iOVM_DeployerWhitelist.sol";

/**
 * @title OVM_DeployerWhitelist
 * @dev The Deployer Whitelist is a temporary predeploy used to provide additional safety during the
 * initial phases of our mainnet roll out. It is owned by the Optimism team, and defines accounts
 * which are allowed to deploy contracts on Layer2. The Execution Manager will only allow an
 * ovmCREATE or ovmCREATE2 operation to proceed if the deployer's address whitelisted.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_DeployerWhitelist is iOVM_DeployerWhitelist {

    /**********************
     * Contract Constants *
     **********************/

    bool public initialized;
    bool public allowArbitraryDeployment;
    address override public owner;
    mapping (address => bool) public whitelist;


    /**********************
     * Function Modifiers *
     **********************/

    /**
     * Blocks functions to anyone except the contract owner.
     */
    modifier onlyOwner() {
        require(
            msg.sender == owner,
            "Function can only be called by the owner of this contract."
        );
        _;
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * Initializes the whitelist.
     * @param _owner Address of the owner for this contract.
     * @param _allowArbitraryDeployment Whether or not to allow arbitrary contract deployment.
     */
    function initialize(
        address _owner,
        bool _allowArbitraryDeployment
    )
        override
        external
    {
        if (initialized == true) {
            return;
        }

        initialized = true;
        allowArbitraryDeployment = _allowArbitraryDeployment;
        owner = _owner;
    }

    /**
     * Adds or removes an address from the deployment whitelist.
     * @param _deployer Address to update permissions for.
     * @param _isWhitelisted Whether or not the address is whitelisted.
     */
    function setWhitelistedDeployer(
        address _deployer,
        bool _isWhitelisted
    )
        override
        external
        onlyOwner
    {
        whitelist[_deployer] = _isWhitelisted;
    }

    /**
     * Updates the owner of this contract.
     * @param _owner Address of the new owner.
     */
    function setOwner(
        address _owner
    )
        override
        public
        onlyOwner
    {
        owner = _owner;
    }

    /**
     * Updates the arbitrary deployment flag.
     * @param _allowArbitraryDeployment Whether or not to allow arbitrary contract deployment.
     */
    function setAllowArbitraryDeployment(
        bool _allowArbitraryDeployment
    )
        override
        public
        onlyOwner
    {
        allowArbitraryDeployment = _allowArbitraryDeployment;
    }

    /**
     * Permanently enables arbitrary contract deployment and deletes the owner.
     */
    function enableArbitraryContractDeployment()
        override
        external
        onlyOwner
    {
        setAllowArbitraryDeployment(true);
        setOwner(address(0));
    }

    /**
     * Checks whether an address is allowed to deploy contracts.
     * @param _deployer Address to check.
     * @return _allowed Whether or not the address can deploy contracts.
     */
    function isDeployerAllowed(
        address _deployer
    )
        override
        external
        returns (
            bool
        )
    {
        return (
            initialized == false
            || allowArbitraryDeployment == true
            || whitelist[_deployer]
        );
    }
}
