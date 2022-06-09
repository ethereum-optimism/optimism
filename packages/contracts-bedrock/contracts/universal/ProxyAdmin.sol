// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { Proxy } from "./Proxy.sol";
import { Owned } from "@rari-capital/solmate/src/auth/Owned.sol";

/**
 * @title ProxyAdmin
 * @dev This is an auxiliary contract meant to be assigned as the admin of a
        Proxy, based on the OpenZeppelin implementation.
 */
contract ProxyAdmin is Owned {
    /**
     * @dev A legacy upgrading indicator used by the old Chugsplash Proxy
     * @custom:legacy
     */
    bool internal upgrading = false;

    /**
     * @notice Set the owner of the ProxyAdmin via constructor argument.
     */

    constructor(address owner) Owned(owner) {}

    /**
     * @dev Returns the current implementation of `proxy`.
     *      This contract must be the admin of `proxy`.
     *
     * @param proxy The Proxy to return the implementation of.
     * @return The address of the implementation
     */
    function getProxyImplementation(Proxy proxy) public view returns (address) {
        // We need to manually run the static call since the getter cannot be flagged as view
        // bytes4(keccak256("implementation()")) == 0x5c60da1b
        (bool success, bytes memory returndata) = address(proxy).staticcall(hex"5c60da1b");
        require(success);
        return abi.decode(returndata, (address));
    }

    /**
     * @dev Returns the current admin of `proxy`.
     *      This contract must be the admin of `proxy`.
     *
     * @param proxy The Proxy to return the admin of.
     * @return The address of the admin
     */
    function getProxyAdmin(Proxy proxy) public view returns (address) {
        // We need to manually run the static call since the getter cannot be flagged as view
        // bytes4(keccak256("admin()")) == 0xf851a440
        (bool success, bytes memory returndata) = address(proxy).staticcall(hex"f851a440");
        require(success);
        return abi.decode(returndata, (address));
    }

    /**
     * @dev Changes the admin of `proxy` to `newAdmin`.
     *      This contract must be the current admin of `proxy`.
     *
     * @param proxy    The proxy that will have its admin updated
     * @param newAdmin The address of the admin to update to
     */
    function changeProxyAdmin(Proxy proxy, address newAdmin) public onlyOwner {
        proxy.changeAdmin(newAdmin);
    }

    /**
     * @dev Upgrades `proxy` to `implementation`.
     *      This contract must be the admin of `proxy`.
     *
     * @param proxy          The address of the proxy
     * @param implementation The address of the implementation
     */
    function upgrade(Proxy proxy, address implementation) public onlyOwner {
        proxy.upgradeTo(implementation);
    }

    /**
     * @dev Upgrades `proxy` to `implementation` and calls a function on the new implementation.
     *      This contract must be the admin of `proxy`.
     *
     * @param proxy           The proxy to call
     * @param implementation  The implementation to upgrade the proxy to
     * @param data            The calldata to pass to the implementation
     */
    function upgradeAndCall(
        Proxy proxy,
        address implementation,
        bytes memory data
    ) public payable onlyOwner {
        proxy.upgradeToAndCall{ value: msg.value }(implementation, data);
    }

    /**
     * @dev Legacy function used by the old Chugsplash proxy
     *      to determine if an upgrade is happening.
     * @custom:legacy
     *
     * @return Whether or not there is an upgrade going on
     */
    function isUpgrading() public view returns (bool) {
        return upgrading;
    }
}
