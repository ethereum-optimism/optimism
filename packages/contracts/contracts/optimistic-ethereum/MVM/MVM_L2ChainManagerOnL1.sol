// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_AddressResolver } from "../libraries/resolver/Lib_AddressResolver.sol";

/* Interface Imports */

/* External Imports */

/**
 * @title MVM_L2ChainManagerOnL1
 * @dev if want support multi l2 chain on l1,it should add a manager to desc 
 * how many l2 chain now ,and dispatch the l2 chain id to make it is unique.
 *
 * Compiler used: solc
 * Runtime target: EVM
 */
contract MVM_L2ChainManagerOnL1 is Lib_AddressResolver {

    /*************
     * Constants *
     *************/
    string constant public CONFIG_OWNER_KEY = "owner";
    

    /*************
     * Variables *
     *************/
    struct L2Config {  
        address owner;
        mapping(string=>bytes) data;
    }
  
  
    string internal owner;
    uint256 internal l2ChainIdBase;
    mapping (uint256 => L2Config) internal l2Configs;
    mapping (address => uint256) public l2chainIds;
    uint256 internal totalL2Config;

    /***************
     * Constructor *
     ***************/

    /**
     * @param _libAddressManager Address of the Address Manager.
     */
    constructor(
        address _libAddressManager,
        string memory _owner
    )
        public
        Lib_AddressResolver(_libAddressManager)
    {
        owner = _owner;
        l2ChainIdBase = 500;
        totalL2Config = 0;
        /*default two l2 chain for testing*/
        _applyL2ChainId(420);
        _applyL2ChainId(421);
    }
    
    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyOwner() {
        require(
            msg.sender == resolve(owner),
            "MVM_L2ChainManagerOnL1: Function can only be called by the owner."
        );
        _;
    }

    /********************
     * Public Functions *
     ********************/
    /**
     * 
     */
    function applyL2ChainId()
        public
        returns (
            uint256 _chainId
        )
    {
        uint256 chainId=l2ChainIdBase++;
        return _applyL2ChainId(chainId);
    }
    
    /**
     * 
     */
    function _applyL2ChainId(uint256 chainId)
        internal
        returns (
            uint256 _chainId
        )
    {
        require(
            l2Configs[chainId].owner==address(0),
            "The l2Configs must be null."
        );
        l2Configs[chainId].owner=msg.sender;
        l2chainIds[msg.sender]=chainId;
        totalL2Config++;
        return chainId;
    }
    

    /**
     * 
     */
    function getTotalL2Chains()
        public
        view
        returns (
            uint256 _totalChains
        )
    {
        return totalL2Config;
    }


     /**
     * Inserts an Config value into the state.
     * @param _chainId Address of the Config to insert.
     * @param _key Key to insert for the given address.
     * @param _value Value to insert for the given key.
     */
    function putL2Config(
        uint256 _chainId,
        string memory _key,
        bytes memory _value
    )
        public
    {
        require(
            l2Configs[_chainId].owner!=address(0),
            "The l2Configs can not be null."
        );
        if(_chainId>=500){
            require(
                l2Configs[_chainId].owner==msg.sender,
                "The updater must be the owner of the config."
            );
        }
        l2Configs[_chainId].data[_key] = _value;
    }

    
    /**
     * Retrieves an Config from the state.
     * @param _chainId Address of the Config to retrieve.
     * @param _key Key of the Config value to retrieve.
     * @return _value Value for the given address and key.
     */
    function getL2ConfigByKey(
            uint256 _chainId,
            string memory _key)
        public
        view
        returns (
            bytes memory _value
        )
    {
        require(
            l2Configs[_chainId].owner!=address(0),
            "The l2Configs must be not null."
        );
        return l2Configs[_chainId].data[_key];
    }
    
    /**
     * Retrieves an Config from the state.
     * @param _chainId Address of the Config to retrieve.
     * @return _value Value for the given address and key.
     */
    function getL2ConfigOwner(uint256 _chainId)
        public
        view
        returns (
            address _value
        )
    {
        require(
            l2Configs[_chainId].owner!=address(0),
            "The l2Configs must be not null."
        );
        return l2Configs[_chainId].owner;
    }
}
