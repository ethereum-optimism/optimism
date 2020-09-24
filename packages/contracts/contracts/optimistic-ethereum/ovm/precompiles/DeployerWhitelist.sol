pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;
import { console } from "@nomiclabs/buidler/console.sol";

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
        bytes32 alreadyInitialized = get(INITIALIZED_KEY);
        if (alreadyInitialized != bytes32(0)) {
            return;
        }

        console.log("Setting values");
        set(INITIALIZED_KEY, bytes32(uint(1)));
        console.log("Just set initialized");
        uint allowArbitraryDeployment = _allowArbitraryDeployment ? 1 : 0;
        console.log("Just set allowArbitraryDeployment");
        set(OWNER_KEY, bytes32(bytes20(_owner)));
        console.log("Just set owner key");
        set(ALLOW_ARBITRARY_DEPLOYMENT, bytes32(allowArbitraryDeployment));
        console.log("Just set allow arbitrary");
        console.log("Initialized!");
        console.log("Arbitrary Deployment!");
        console.log(allowArbitraryDeployment);
        console.log("Owner");
        console.log(_owner);
    }

    /*
     * Modifiers
     */
    // Source: https://solidity.readthedocs.io/en/v0.5.3/contracts.html
    modifier onlyOwner {
        address owner = address(bytes20(get(OWNER_KEY)));
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
        uint isWhitelisted = _isWhitelisted ? 1 : 0;
        set(bytes32(bytes20(_deployerAddress)), bytes32(isWhitelisted));
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
        set(OWNER_KEY, bytes32(bytes20(_newOwner)));
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
        set(ALLOW_ARBITRARY_DEPLOYMENT, bytes32(allowArbitraryDeployment));
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
        set(ALLOW_ARBITRARY_DEPLOYMENT, bytes32(uint(1)));
        set(OWNER_KEY, bytes32(bytes20(address(0))));
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
        bool allowArbitraryDeployment = uint(get(ALLOW_ARBITRARY_DEPLOYMENT)) == 1;
        bool isWhitelistedDeployer = uint(get(bytes32(bytes20(_deployerAddress)))) == 1;
        bool isInitialized = uint(get(INITIALIZED_KEY)) != 0;
        
        console.log("Allow arbitrary");
        console.log(allowArbitraryDeployment);
        console.log("isWhitelistedDeployer");
        console.log(isWhitelistedDeployer);
        console.log("isInitialized");
        console.log(isInitialized);

        return allowArbitraryDeployment || isWhitelistedDeployer || !isInitialized;
    }

    /**
     * Sets storage
     */
    function set(
        bytes32 _key,
        bytes32 _value
    )
        public
    {
        bytes4 methodId = bytes4(keccak256("ovmSSTORE()"));
        msg.sender.call(abi.encodeWithSelector(methodId, _key, _value));
        console.log("SET key");
        console.logBytes32(_key);
        console.log("SET value");
        console.logBytes32(_value);
    }

    /**
     * Gets storage
     */
    function get(
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
        console.log("GET key");
        console.logBytes32(_key);
        console.log("GET returned value");
        console.logBytes32(value);
        return value;
    }
}
