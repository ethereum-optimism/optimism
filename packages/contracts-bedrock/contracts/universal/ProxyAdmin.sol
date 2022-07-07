// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { Owned } from "@rari-capital/solmate/src/auth/Owned.sol";
import { Proxy } from "./Proxy.sol";
import { AddressManager } from "../legacy/AddressManager.sol";
import { L1ChugSplashProxy } from "../legacy/L1ChugSplashProxy.sol";

/**
 * @title IStaticERC1967Proxy
 * @notice IStaticERC1967Proxy is a static version of the ERC1967 proxy interface.
 */
interface IStaticERC1967Proxy {
    function implementation() external view returns (address);

    function admin() external view returns (address);
}

/**
 * @title IStaticL1ChugSplashProxy
 * @notice IStaticL1ChugSplashProxy is a static version of the ChugSplash proxy interface.
 */
interface IStaticL1ChugSplashProxy {
    function getImplementation() external view returns (address);

    function getOwner() external view returns (address);
}

/**
 * @title ProxyAdmin
 * @notice This is an auxiliary contract meant to be assigned as the admin of an ERC1967 Proxy,
 *         based on the OpenZeppelin implementation. It has backwards compatibility logic to work
 *         with the various types of proxies that have been deployed by Optimism in the past.
 */
contract ProxyAdmin is Owned {
    /**
     * @notice The proxy types that the ProxyAdmin can manage.
     *
     * @custom:value ERC1967          Represents an ERC1967 compliant transparent proxy interface.
     * @custom:value Chugsplash       Represents the Chugsplash proxy interface (legacy).
     * @custom:value ResolvedDelegate Represents the ResolvedDelegate proxy (legacy).
     */
    enum ProxyType {
        ERC1967,
        Chugsplash,
        ResolvedDelegate
    }

    /**
     * @custom:legacy
     * @notice A mapping of proxy types, used for backwards compatibility.
     */
    mapping(address => ProxyType) public proxyType;

    /**
     * @custom:legacy
     * @notice A reverse mapping of addresses to names held in the AddressManager. This must be
     *         manually kept up to date with changes in the AddressManager for this contract
     *         to be able to work as an admin for the ResolvedDelegateProxy type.
     */
    mapping(address => string) public implementationName;

    /**
     * @custom:legacy
     * @notice The address of the address manager, this is required to manage the
     *         ResolvedDelegateProxy type.
     */
    AddressManager public addressManager;

    /**
     * @custom:legacy
     * @notice A legacy upgrading indicator used by the old Chugsplash Proxy.
     */
    bool internal upgrading = false;

    /**
     * @param _owner Address of the initial owner of this contract.
     */
    constructor(address _owner) Owned(_owner) {}

    /**
     * @notice Sets the proxy type for a given address. Only required for non-standard (legacy)
     *         proxy types.
     *
     * @param _address Address of the proxy.
     * @param _type    Type of the proxy.
     */
    function setProxyType(address _address, ProxyType _type) external onlyOwner {
        proxyType[_address] = _type;
    }

    /**
     * @notice Sets the implementation name for a given address. Only required for
     *         ResolvedDelegateProxy type proxies that have an implementation name.
     *
     * @param _address Address of the ResolvedDelegateProxy.
     * @param _name    Name of the implementation for the proxy.
     */
    function setImplementationName(address _address, string memory _name) external onlyOwner {
        implementationName[_address] = _name;
    }

    /**
     * @notice Set the address of the AddressManager. This is required to manage legacy
     *         ResolvedDelegateProxy type proxy contracts.
     *
     * @param _address Address of the AddressManager.
     */
    function setAddressManager(AddressManager _address) external onlyOwner {
        addressManager = _address;
    }

    /**
     * @custom:legacy
     * @notice Set an address in the address manager. Since only the owner of the AddressManager
     *         can directly modify addresses and the ProxyAdmin will own the AddressManager, this
     *         gives the owner of the ProxyAdmin the ability to modify addresses directly.
     *
     * @param _name    Name to set within the AddressManager.
     * @param _address Address to attach to the given name.
     */
    function setAddress(string memory _name, address _address) external onlyOwner {
        addressManager.setAddress(_name, _address);
    }

    /**
     * @custom:legacy
     * @notice Legacy function used to tell ChugSplashProxy contracts if an upgrade is happening.
     *
     * @return Whether or not there is an upgrade going on. May not actually tell you whether an
     *         upgrade is going on, since we don't currently plan to use this variable for anything
     *         other than a legacy indicator to fix a UX bug in the ChugSplash proxy.
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
     * @notice Returns the implementation of the given proxy address.
     *
     * @param _proxy Address of the proxy to get the implementation of.
     *
     * @return Address of the implementation of the proxy.
     */
    function getProxyImplementation(Proxy _proxy) external view returns (address) {
        ProxyType proxyType = proxyType[address(_proxy)];

        if (proxyType == ProxyType.ERC1967) {
            return IStaticERC1967Proxy(address(_proxy)).implementation();
        } else if (proxyType == ProxyType.Chugsplash) {
            return IStaticL1ChugSplashProxy(address(_proxy)).getImplementation();
        } else if (proxyType == ProxyType.ResolvedDelegate) {
            return addressManager.getAddress(implementationName[address(_proxy)]);
        } else {
            revert("ProxyAdmin: unknown proxy type");
        }
    }

    /**
     * @notice Returns the admin of the given proxy address.
     *
     * @param _proxy Address of the proxy to get the admin of.
     *
     * @return Address of the admin of the proxy.
     */
    function getProxyAdmin(Proxy _proxy) external view returns (address) {
        ProxyType proxyType = proxyType[address(_proxy)];

        if (proxyType == ProxyType.ERC1967) {
            return IStaticERC1967Proxy(address(_proxy)).admin();
        } else if (proxyType == ProxyType.Chugsplash) {
            return IStaticL1ChugSplashProxy(address(_proxy)).getOwner();
        } else if (proxyType == ProxyType.ResolvedDelegate) {
            return addressManager.owner();
        } else {
            revert("ProxyAdmin: unknown proxy type");
        }
    }

    /**
     * @notice Updates the admin of the given proxy address.
     *
     * @param _proxy    Address of the proxy to update.
     * @param _newAdmin Address of the new proxy admin.
     */
    function changeProxyAdmin(Proxy _proxy, address _newAdmin) external onlyOwner {
        ProxyType proxyType = proxyType[address(_proxy)];

        if (proxyType == ProxyType.ERC1967) {
            _proxy.changeAdmin(_newAdmin);
        } else if (proxyType == ProxyType.Chugsplash) {
            L1ChugSplashProxy(payable(_proxy)).setOwner(_newAdmin);
        } else if (proxyType == ProxyType.ResolvedDelegate) {
            addressManager.transferOwnership(_newAdmin);
        } else {
            revert("ProxyAdmin: unknown proxy type");
        }
    }

    /**
     * @notice Changes a proxy's implementation contract.
     *
     * @param _proxy          Address of the proxy to upgrade.
     * @param _implementation Address of the new implementation address.
     */
    function upgrade(Proxy _proxy, address _implementation) public onlyOwner {
        ProxyType proxyType = proxyType[address(_proxy)];

        if (proxyType == ProxyType.ERC1967) {
            _proxy.upgradeTo(_implementation);
        } else if (proxyType == ProxyType.Chugsplash) {
            L1ChugSplashProxy(payable(_proxy)).setStorage(
                0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc,
                bytes32(uint256(uint160(_implementation)))
            );
        } else if (proxyType == ProxyType.ResolvedDelegate) {
            string memory name = implementationName[address(_proxy)];
            addressManager.setAddress(name, _implementation);
        } else {
            revert("ProxyAdmin: unknown proxy type");
        }
    }

    /**
     * @notice Changes a proxy's implementation contract and delegatecalls the new implementation
     *         with some given data. Useful for atomic upgrade-and-initialize calls.
     *
     * @param _proxy          Address of the proxy to upgrade.
     * @param _implementation Address of the new implementation address.
     * @param _data           Data to trigger the new implementation with.
     */
    function upgradeAndCall(
        Proxy _proxy,
        address _implementation,
        bytes memory _data
    ) external payable onlyOwner {
        ProxyType proxyType = proxyType[address(_proxy)];

        if (proxyType == ProxyType.ERC1967) {
            _proxy.upgradeToAndCall{ value: msg.value }(_implementation, _data);
        } else {
            // reverts if proxy type is unknown
            upgrade(_proxy, _implementation);
            (bool success, ) = address(_proxy).call{ value: msg.value }(_data);
            require(success);
        }
    }
}
