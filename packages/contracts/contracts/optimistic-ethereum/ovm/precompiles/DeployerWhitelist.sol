pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title DeployerWhitelist
 */
contract DeployerWhitelist {
    // mapping(address=>bool) public whitelistedDeployers;
    bytes32 constant INITIALIZED_KEY = 0x0000000000000000000000000000000000000000000000000000000000000010;
    bytes32 constant OWNER_KEY = 0x0000000000000000000000000000000000000000000000000000000000000011;
    bytes32 constant ALLOW_ARBITRARY_DEPLOYMENT = 0x0000000000000000000000000000000000000000000000000000000000000012;

    constructor (address _owner, bool _allowArbitraryDeployment) public {}

    function initialize(address _owner, bool _allowArbitraryDeployment)
        public
    {
        bytes32 alreadyInitialized = doOvmSLOAD(INITIALIZED_KEY);
        if (alreadyInitialized != bytes32(0)) {
            return;
        }

        doOvmSSTORE(INITIALIZED_KEY, bytes32(uint(1)));
        doOvmSSTORE(OWNER_KEY, bytes32(bytes20(_owner)));
        uint allowArbitraryDeployment = _allowArbitraryDeployment ? 1 : 0;
        doOvmSSTORE(ALLOW_ARBITRARY_DEPLOYMENT, bytes32(allowArbitraryDeployment));
    }

    /*
     * Modifiers
     */
    // Source: https://solidity.readthedocs.io/en/v0.5.3/contracts.html
    modifier onlyOwner {
        bytes32 owner = doOvmSLOAD(OWNER_KEY);
        bytes32 ovmCALLER = doOvmCALLER();
        require(
            ovmCALLER == owner,
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
        uint isWhitelisted = _isWhitelisted ? 1 : 0;
        doOvmSSTORE(bytes32(bytes20(_deployerAddress)), bytes32(isWhitelisted));
    }

    /**
     * Set owner of this contract.
     */
    function setOwner(
        address _newOwner
    )
        external
        onlyOwner
    {
        doOvmSSTORE(OWNER_KEY, bytes32(bytes20(_newOwner)));
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
        uint allowArbitraryDeployment = _allowArbitraryDeployment ? 1 : 0;
        doOvmSSTORE(ALLOW_ARBITRARY_DEPLOYMENT, bytes32(allowArbitraryDeployment));
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
        doOvmSSTORE(ALLOW_ARBITRARY_DEPLOYMENT, bytes32(uint(1)));
        doOvmSSTORE(OWNER_KEY, bytes32(bytes20(address(0))));
    }

    /**
     * Returns whether or not the deployer address is allowed to deploy new contracts.
     */
    function isDeployerAllowed(
        address _deployerAddress
    )
        external
        returns(bool)
    {
        bool allowArbitraryDeployment = uint(doOvmSLOAD(ALLOW_ARBITRARY_DEPLOYMENT)) == 1;
        bool isWhitelistedDeployer = uint(doOvmSLOAD(bytes32(bytes20(_deployerAddress)))) == 1;
        bool isInitialized = uint(doOvmSLOAD(INITIALIZED_KEY)) != 0;
        
        return allowArbitraryDeployment || isWhitelistedDeployer || !isInitialized;
    }

    /**
     * Sets storage
     */
    function doOvmSSTORE(
        bytes32 _key,
        bytes32 _value
    )
        internal
    {
        bytes4 methodId = bytes4(keccak256("ovmSSTORE()"));
        msg.sender.call(abi.encodeWithSelector(methodId, _key, _value));
    }

    /**
     * Gets storage
     */
    function doOvmSLOAD(
        bytes32 _key
    )
        public
        returns(bytes32)
    {
        bytes4 methodId = bytes4(keccak256("ovmSLOAD()"));
        (, bytes memory result) = msg.sender.call(abi.encodeWithSelector(methodId, _key));
        bytes32 value;
        assembly {
            value := mload(add(result, 0x20))
        }
        return value;
    }

    function doOvmCALLER() internal returns(bytes32) {
        bytes4 methodId = bytes4(keccak256("ovmCALLER()"));
        (, bytes memory result) = msg.sender.call(abi.encodeWithSelector(methodId)); 
        
        bytes32 value;
        assembly {
            value := mload(add(result, 0x20))
        }
        return value << 96;
    }
}
