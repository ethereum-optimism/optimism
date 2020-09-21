// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/* Proxy Imports */
import { Proxy_Manager } from "./Proxy_Manager.sol";

/**
 * @title Proxy_Forwarder
 */
contract Proxy_Forwarder {
    Proxy_Manager private proxyManager;

    constructor(
        address _proxyManager
    ) {
        proxyManager = Proxy_Manager(_proxyManager);
    }

    fallback()
        external
    {
        address target = _getTarget();
        bytes memory data = msg.data;

        require(
            target != address(0),
            "Proxy does not have a target."
        );

        assembly {
            let success := call(
                gas(),
                target,
                0,
                add(data, 0x20),
                mload(data),
                0,
                0
            )

            let size := returndatasize()
            let returndata := mload(0x40)
            mstore(0x40, add(returndata, add(size, 0x20)))
            returndatacopy(add(returndata, 0x20), 0, size)
            
            if iszero(success) {
                revert(add(returndata, 0x20), size)
            }

            return(add(returndata, 0x20), size)
        }
    }

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
