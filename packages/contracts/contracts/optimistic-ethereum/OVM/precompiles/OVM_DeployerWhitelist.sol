// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_Bytes32Utils } from "../../libraries/utils/Lib_Bytes32Utils.sol";

/* Interface Imports */
import { iOVM_DeployerWhitelist } from "../../iOVM/precompiles/iOVM_DeployerWhitelist.sol";
import { Lib_SafeExecutionManagerWrapper } from "../../libraries/wrappers/Lib_SafeExecutionManagerWrapper.sol";

/**
 * @title OVM_DeployerWhitelist
 * @dev The Deployer Whitelist is a temporary predeploy used to provide additional safety during the
 * initial phases of our mainnet roll out. It is owned by the Optimism team, and defines accounts
 * which are allowed to deploy contracts on Layer2. The Execution Manager will only allow an 
 * ovmCREATE or ovmCREATE2 operation to proceed if the deployer's address whitelisted.
 * 
 * Compiler used: solc
 * Runtime target: OVM
 */
contract OVM_DeployerWhitelist is iOVM_DeployerWhitelist {

    /**********************
     * Contract Constants *
     **********************/

    bytes32 internal constant KEY_INITIALIZED =                0x0000000000000000000000000000000000000000000000000000000000000010;
    bytes32 internal constant KEY_OWNER =                      0x0000000000000000000000000000000000000000000000000000000000000011;
    bytes32 internal constant KEY_ALLOW_ARBITRARY_DEPLOYMENT = 0x0000000000000000000000000000000000000000000000000000000000000012;


    /**********************
     * Function Modifiers *
     **********************/
    
    /**
     * Blocks functions to anyone except the contract owner.
     */
    modifier onlyOwner() {
        address owner = Lib_Bytes32Utils.toAddress(
            Lib_SafeExecutionManagerWrapper.safeSLOAD(
                KEY_OWNER
            )
        );

        Lib_SafeExecutionManagerWrapper.safeREQUIRE(
            Lib_SafeExecutionManagerWrapper.safeCALLER() == owner,
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
        public
    {
        bool initialized = Lib_Bytes32Utils.toBool(
            Lib_SafeExecutionManagerWrapper.safeSLOAD(KEY_INITIALIZED)
        );

        if (initialized == true) {
            return;
        }

        Lib_SafeExecutionManagerWrapper.safeSSTORE(
            KEY_INITIALIZED,
            Lib_Bytes32Utils.fromBool(true)
        );
        Lib_SafeExecutionManagerWrapper.safeSSTORE(
            KEY_OWNER,
            Lib_Bytes32Utils.fromAddress(_owner)
        );
        Lib_SafeExecutionManagerWrapper.safeSSTORE(
            KEY_ALLOW_ARBITRARY_DEPLOYMENT,
            Lib_Bytes32Utils.fromBool(_allowArbitraryDeployment)
        );
    }

    /**
     * Gets the owner of the whitelist.
     */
    function getOwner()
        override
        public
        returns(
            address
        )
    {
        return Lib_Bytes32Utils.toAddress(
            Lib_SafeExecutionManagerWrapper.safeSLOAD(
                KEY_OWNER
            )
        );
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
        public
        onlyOwner
    {
        Lib_SafeExecutionManagerWrapper.safeSSTORE(
            Lib_Bytes32Utils.fromAddress(_deployer),
            Lib_Bytes32Utils.fromBool(_isWhitelisted)
        );
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
        Lib_SafeExecutionManagerWrapper.safeSSTORE(
            KEY_OWNER,
            Lib_Bytes32Utils.fromAddress(_owner)
        );
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
        Lib_SafeExecutionManagerWrapper.safeSSTORE(
            KEY_ALLOW_ARBITRARY_DEPLOYMENT,
            Lib_Bytes32Utils.fromBool(_allowArbitraryDeployment)
        );
    }

    /**
     * Permanently enables arbitrary contract deployment and deletes the owner.
     */
    function enableArbitraryContractDeployment()
        override
        public
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
        public
        returns (
            bool _allowed
        )
    {
        bool initialized = Lib_Bytes32Utils.toBool(
            Lib_SafeExecutionManagerWrapper.safeSLOAD(KEY_INITIALIZED)
        );

        if (initialized == false) {
            return true;
        }

        bool allowArbitraryDeployment = Lib_Bytes32Utils.toBool(
            Lib_SafeExecutionManagerWrapper.safeSLOAD(KEY_ALLOW_ARBITRARY_DEPLOYMENT)
        );

        if (allowArbitraryDeployment == true) {
            return true;
        }

        bool isWhitelisted = Lib_Bytes32Utils.toBool(
            Lib_SafeExecutionManagerWrapper.safeSLOAD(
                Lib_Bytes32Utils.fromAddress(_deployer)
            )
        );

        return isWhitelisted;        
    }
}
