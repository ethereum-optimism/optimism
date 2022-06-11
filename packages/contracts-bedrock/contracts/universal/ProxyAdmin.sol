// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { Proxy } from "./Proxy.sol";
import { Owned } from "@rari-capital/solmate/src/auth/Owned.sol";
import { Lib_AddressManager } from "../legacy/Lib_AddressManager.sol";
import { L1ChugSplashProxy } from "../legacy/L1ChugSplashProxy.sol";
import { Bytes32AddressLib } from "@rari-capital/solmate/src/utils/Bytes32AddressLib.sol";


/**
 * @title ProxyAdmin
 * @dev This is an auxiliary contract meant to be assigned as the admin of a
        Proxy, based on the OpenZeppelin implementation.
 */
contract ProxyAdmin is Owned {

   enum ProxyType {
       OpenZeppelin,
       Chugsplash,
       ResolvedDelegate
   }

   mapping(address => ProxyType) internal _proxyType;
   mapping(address => string) internal _proxyName;

   Lib_AddressManager addressManager;

    /**
     * @notice A legacy upgrading indicator used by the old Chugsplash Proxy
     * @custom:legacy
     */
    bool internal upgrading = false;

    /**
     * @notice Set the owner of the ProxyAdmin via constructor argument.
     */

    constructor(address owner) Owned(owner) {}

    /**
     * @notice
     *
     * @param _address   The address of the proxy
     * @param _type The type of the proxy
     */
    function setProxyType(address _address, ProxyType _type) public onlyOwner {
        _proxyType[_address] = _type;
    }

    function getProxyType(address _address) public view returns (ProxyType) {
        return _proxyType[_address];
    }

    function setProxyName(address _address, string memory _name) public onlyOwner {
        _proxyName[_address] = _name;
    }

    function getProxyName(address _address) public view returns (string memory) {
        return _proxyName[_address];
    }

    function setAddressManager(address _address) external onlyOwner {
        addressManager = Lib_AddressManager(_address);
    }

    /**
     * @notice
     *
     * @param _upgrading
     */
    function setIsUpgrading(bool _upgrading) external onlyOwner {
        upgrading = _upgrading;
    }

    /**
     * @dev Returns the current implementation of `proxy`.
     *      This contract must be the admin of `proxy`.
     *
     * @param proxy The Proxy to return the implementation of.
     * @return The address of the implementation
     */
    function getProxyImplementation(Proxy proxy) external view returns (address) {
        ProxyType proxyType = getProxyType(address(proxy));

        // We need to manually run the static call since the getter cannot be flagged as view
        if (proxyType == ProxyType.OpenZeppelin) {
            // bytes4(keccak256("implementation()")) == 0x5c60da1b
            (bool success, bytes memory returndata) = address(proxy).staticcall(hex"5c60da1b");
            require(success);
            return abi.decode(returndata, (address));
        } else if (proxyType == ProxyType.Chugsplash) {
            // bytes4(keccak256("getImplementation()")) == 0xaaf10f42
            (bool success, bytes memory returndata) = address(proxy).staticcall(hex"aaf10f42");
            require(success);
            return abi.decode(returndata, (address));
        } else if (proxyType == ProxyType.ResolvedDelegate) {
            string memory name = getProxyName(address(proxy));
            return addressManager.getAddress(name);
        }
    }

    /**
     * @dev Returns the current admin of `proxy`.
     *      This contract must be the admin of `proxy`.
     *
     * @param proxy The Proxy to return the admin of.
     * @return The address of the admin
     */
    function getProxyAdmin(Proxy proxy) external view returns (address) {
        ProxyType proxyType = getProxyType(address(proxy));

        // We need to manually run the static call since the getter cannot be flagged as view
        if (proxyType == ProxyType.OpenZeppelin) {
            // bytes4(keccak256("admin()")) == 0xf851a440
            (bool success, bytes memory returndata) = address(proxy).staticcall(hex"f851a440");
            require(success);
            return abi.decode(returndata, (address));
        } else if (proxyType == ProxyType.Chugsplash) {
            // bytes4(keccak256("getOwner()")) == 0x
            (bool success, bytes memory returndata) = address(proxy).staticcall(hex"893d20e8");
            require(success);
            return abi.decode(returndata, (address));
        } else if (proxyType == ProxyType.ResolvedDelegate) {
            return addressManager.owner();
        }
    }

    /**
     * @dev Changes the admin of `proxy` to `newAdmin`.
     *      This contract must be the current admin of `proxy`.
     *
     * @param proxy    The proxy that will have its admin updated
     * @param newAdmin The address of the admin to update to
     */
    function changeProxyAdmin(Proxy proxy, address newAdmin) external onlyOwner {
        ProxyType proxyType = getProxyType(address(proxy));

        if (proxyType == ProxyType.OpenZeppelin) {
            proxy.changeAdmin(newAdmin);
        } else if (proxyType == ProxyType.Chugsplash) {
            L1ChugSplashProxy(payable(proxy)).setOwner(newAdmin);
        } else if (proxyType == ProxyType.ResolvedDelegate) {
            Lib_AddressManager(address(proxy)).transferOwnership(newAdmin);
        }
    }

    /**
     * @dev Upgrades `proxy` to `implementation`.
     *      This contract must be the admin of `proxy`.
     *
     * @param proxy          The address of the proxy
     * @param implementation The address of the implementation
     */
    function upgrade(Proxy proxy, address implementation) public onlyOwner {
        ProxyType proxyType = getProxyType(address(proxy));

        if (proxyType == ProxyType.OpenZeppelin) {
            proxy.upgradeTo(implementation);
        } else if (proxyType == ProxyType.Chugsplash) {
            bytes memory code = address(proxy).code;
            L1ChugSplashProxy(payable(proxy)).setStorage(
                0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc,
                Bytes32AddressLib.fillLast12Bytes(implementation)
            );
        } else if (proxyType == ProxyType.ResolvedDelegate) {
            string memory name = getProxyName(address(proxy));
            Lib_AddressManager(address(proxy)).setAddress(name, implementation);
        }
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
    ) external payable onlyOwner {
        upgrade(proxy, implementation);
        (bool success,) = address(proxy).delegatecall(data);
        require(success);
    }

    /**
     * @dev Legacy function used by the old Chugsplash proxy
     *      to determine if an upgrade is happening.
     * @custom:legacy
     *
     * @return Whether or not there is an upgrade going on
     */
    function isUpgrading() external view returns (bool) {
        return upgrading;
    }
}
