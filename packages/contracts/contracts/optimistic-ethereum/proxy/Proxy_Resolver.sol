// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/* Proxy Imports */
import { Proxy_Manager } from "./Proxy_Manager.sol";

/**
 * @title Proxy_Resolver
 */
contract Proxy_Resolver {

    /*******************************************
     * Contract Variables: Contract References * 
     *******************************************/

    Proxy_Manager internal proxyManager;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _proxyManager Address of the Proxy_Manager.
     */
    constructor(
        address _proxyManager
    ) {
        proxyManager = Proxy_Manager(_proxyManager);
    }


    /********************
     * Public Functions *
     ********************/

    function resolve(
        string memory _name
    )
        public
        view
        returns (
            address _proxy
        )
    {
        return proxyManager.getProxy(_name);
    }

    function resolveTarget(
        string memory _name
    )
        public
        view
        returns (
            address _target
        )
    {
        return proxyManager.getTarget(_name);
    }
}
