// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/* Proxy Imports */
import { Proxy_Manager } from "./Proxy_Manager.sol";

/**
 * @title Proxy_Forwarder
 */
contract Proxy_Forwarder {

    /*******************************************
     * Contract Variables: Contract References * 
     *******************************************/

    Proxy_Manager private proxyManager;


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

    /**
     * Forwards calls to the appropriate target.
     */
    fallback()
        external
    {
        address target = _getTarget();

        require(
            target != address(0),
            "Proxy does not have a target."
        );

        (bool success, bytes memory returndata) = target.call(msg.data);

        if (success == true) {
            assembly {
                return(add(returndata, 0x20), mload(returndata))
            }
        } else {
            assembly {
                revert(add(returndata, 0x20), mload(returndata))
            }
        }
    }


    /**********************
     * Internal Functions *
     **********************/

    /**
     * Determines the appropriate target.
     * @return _target Target to forward requests to.
     */
    function _getTarget()
        internal
        view
        returns (
            address _target
        )
    {
        address target;
        if (proxyManager.isProxy(msg.sender)) {
            target = proxyManager.getTarget(msg.sender);
        } else if (proxyManager.hasProxy(msg.sender)) {
            target = proxyManager.getProxy(msg.sender);
        } else {
            target = proxyManager.getTarget(address(this));
        }

        return target;
    }
}
