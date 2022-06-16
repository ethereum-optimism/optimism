// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { Proxy } from "./Proxy.sol";
import { Owned } from "@rari-capital/solmate/src/auth/Owned.sol";
import { Lib_AddressManager } from "../legacy/Lib_AddressManager.sol";
import { L1ChugSplashProxy } from "../legacy/L1ChugSplashProxy.sol";

// Define static interfaces of these proxies so that we can easily
// use staticcall on the getters we need.
interface IStatic_ERC1967Proxy {
    function implementation() external view returns (address);

    function admin() external view returns (address);
}

interface IStatic_L1ChugSplashProxy {
    function getImplementation() external view returns (address);

    function getOwner() external view returns (address);
}

/**
 * @title ProxyAdmin
 * @dev This is an auxiliary contract meant to be assigned as the admin of an ERC1967 Proxy,
 *      based on the OpenZeppelin implementation. It has backwards compatibility logic to work with
 *      the various types of proxies that have been deployed by Optimism.
 */
contract ProxyAdmin is Owned {
    /**
     * @notice The proxy types that the ProxyAdmin can manage.
     *
     * @custom:value ERC1967          Represents an ERC1967 compliant transparent proxy
     *                                interface, this is the default.
     * @custom:value Chugsplash       Represents the Chugsplash proxy interface,
     *                                this is legacy.
     * @custom:value ResolvedDelegate Represents the ResolvedDelegate proxy
     *                                interface, this is legacy.
     */
    enum ProxyType {
        ERC1967,
        Chugsplash,
        ResolvedDelegate
    }

    /**
     * @custom:legacy
     * @notice         A mapping of proxy types, used for backwards compatibility.
     */
    mapping(address => ProxyType) public proxyType;

    /**
     * @custom:legacy
     * @notice A reverse mapping of addresses to names held in the AddressManager. This must be
     *         manually kept up to date with changes in the AddressManager for this contract
     *         to be able to work as an admin for the Lib_ResolvedDelegateProxy type.
     */
    mapping(address => string) public implementationName;

    /**
     * @custom:legacy
     * @notice The address of the address manager, this is required to manage the
     *         Lib_ResolvedDelegateProxy type.
     */
    Lib_AddressManager public addressManager;

    /**
     * @custom:legacy
     * @notice A legacy upgrading indicator used by the old Chugsplash Proxy.
     */
    bool internal upgrading = false;

    /**
     * @notice Set the owner of the ProxyAdmin via constructor argument.
     */
    constructor(address owner) Owned(owner) {}

    /**
     * @notice
     *
     * @param _address   The address of the proxy.
     * @param _type The type of the proxy.
     */
    function setProxyType(address _address, ProxyType _type) external onlyOwner {
        proxyType[_address] = _type;
    }

    /**
     * @notice Set the proxy type in the mapping. This needs to be kept up to date by the owner of
     *         the contract.
     *
     * @param _address The address to be named.
     * @param _name    The name of the address.
     */
    function setImplementationName(address _address, string memory _name) external onlyOwner {
        implementationName[_address] = _name;
    }

    /**
     * @notice Set the address of the address manager. This is required to manage the legacy
     *         `Lib_ResolvedDelegateProxy`.
     *
     * @param _address The address of the address manager.
     */
    function setAddressManager(Lib_AddressManager _address) external onlyOwner {
        addressManager = _address;
    }

    /**
     * @custom:legacy
     * @notice Set an address in the address manager. This is required because only the owner of
     *         the AddressManager can set the addresses in it.
     *
     * @param _name    The name of the address to set in the address manager.
     * @param _address The address to set in the address manager.
     */
    function setAddress(string memory _name, address _address) external onlyOwner {
        addressManager.setAddress(_name, _address);
    }

    /**
     * @custom:legacy
     * @notice Legacy function used by the old Chugsplash proxy to determine if an upgrade is
     *         happening.
     *
     * @return Whether or not there is an upgrade going on
     */
    function isUpgrading() external view returns (bool) {
        return upgrading;
    }

    /**
     * @custom:legacy
     * @notice Set the upgrading status for the Chugsplash proxy type.
     *
     * @param _upgrading Whether or not the system is upgrading.
     */
    function setUpgrading(bool _upgrading) external onlyOwner {
        upgrading = _upgrading;
    }

    /**
     * @dev Returns the current implementation of `proxy`.
     *      This contract must be the admin of `proxy`.
     *
     * @param proxy The Proxy to return the implementation of.
     * @return The address of the implementation.
     */
    function getProxyImplementation(Proxy proxy) external view returns (address) {
        ProxyType proxyType = proxyType[address(proxy)];

        if (proxyType == ProxyType.ERC1967) {
            return IStatic_ERC1967Proxy(address(proxy)).implementation();
        } else if (proxyType == ProxyType.Chugsplash) {
            return IStatic_L1ChugSplashProxy(address(proxy)).getImplementation();
        } else if (proxyType == ProxyType.ResolvedDelegate) {
            return addressManager.getAddress(implementationName[address(proxy)]);
        } else {
            revert("ProxyAdmin: unknown proxy type");
        }
    }

    /**
     * @dev Returns the current admin of `proxy`.
     *      This contract must be the admin of `proxy`.
     *
     * @param proxy The Proxy to return the admin of.
     * @return The address of the admin.
     */
    function getProxyAdmin(Proxy proxy) external view returns (address) {
        ProxyType proxyType = proxyType[address(proxy)];

        if (proxyType == ProxyType.ERC1967) {
            return IStatic_ERC1967Proxy(address(proxy)).admin();
        } else if (proxyType == ProxyType.Chugsplash) {
            return IStatic_L1ChugSplashProxy(address(proxy)).getOwner();
        } else if (proxyType == ProxyType.ResolvedDelegate) {
            return addressManager.owner();
        } else {
            revert("ProxyAdmin: unknown proxy type");
        }
    }

    /**
     * @dev Changes the admin of `proxy` to `newAdmin`. This contract must be the current admin
     *      of `proxy`.
     *
     * @param proxy    The proxy that will have its admin updated.
     * @param newAdmin The address of the admin to update to.
     */
    function changeProxyAdmin(Proxy proxy, address newAdmin) external onlyOwner {
        ProxyType proxyType = proxyType[address(proxy)];

        if (proxyType == ProxyType.ERC1967) {
            proxy.changeAdmin(newAdmin);
        } else if (proxyType == ProxyType.Chugsplash) {
            L1ChugSplashProxy(payable(proxy)).setOwner(newAdmin);
        } else if (proxyType == ProxyType.ResolvedDelegate) {
            addressManager.transferOwnership(newAdmin);
        } else {
            revert("ProxyAdmin: unknown proxy type");
        }
    }

    /**
     * @dev Upgrades `proxy` to `implementation`. This contract must be the admin of `proxy`.
     *
     * @param proxy          The address of the proxy.
     * @param implementation The address of the implementation.
     */
    function upgrade(Proxy proxy, address implementation) public onlyOwner {
        ProxyType proxyType = proxyType[address(proxy)];

        if (proxyType == ProxyType.ERC1967) {
            proxy.upgradeTo(implementation);
        } else if (proxyType == ProxyType.Chugsplash) {
            L1ChugSplashProxy(payable(proxy)).setStorage(
                0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc,
                bytes32(uint256(uint160(implementation)))
            );
        } else if (proxyType == ProxyType.ResolvedDelegate) {
            string memory name = implementationName[address(proxy)];
            addressManager.setAddress(name, implementation);
        } else {
            revert("ProxyAdmin: unknown proxy type");
        }
    }

    /**
     * @dev Upgrades `proxy` to `implementation` and calls a function on the new implementation.
     *      This contract must be the admin of `proxy`.
     *
     * @param proxy           The proxy to call.
     * @param implementation  The implementation to upgrade the proxy to.
     * @param data            The calldata to pass to the implementation.
     */
    function upgradeAndCall(
        Proxy proxy,
        address implementation,
        bytes memory data
    ) external payable onlyOwner {
        ProxyType proxyType = proxyType[address(proxy)];

        if (proxyType == ProxyType.ERC1967) {
            proxy.upgradeToAndCall{ value: msg.value }(implementation, data);
        } else {
            // reverts if proxy type is unknown
            upgrade(proxy, implementation);
            (bool success, ) = address(proxy).call{ value: msg.value }(data);
            require(success);
        }
    }
}
