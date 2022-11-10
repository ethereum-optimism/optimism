// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { IProxyAdapter } from "./adapters/IProxyAdapter.sol";
import { StandardProxyAdapter } from "./adapters/StandardProxyAdapter.sol";

/**
 * @title ProxyAdmin
 * @notice This is an auxiliary contract meant to be assigned as the admin of an ERC1967 Proxy,
 *         based on the OpenZeppelin implementation. It has backwards compatibility logic to work
 *         with the various types of proxies that have been deployed by Optimism in the past.
 */
contract ProxyAdmin is Ownable {
    /**
     * @notice A mapping of proxy addresses to their respective adapters.
     */
    mapping(address => address) public adapters;

    /**
     * @custom:legacy
     * @notice A legacy upgrading indicator used by the old Chugsplash Proxy. We permanently set
     *         this value to false because there are other ways to pause the contract that are more
     *         consistent with the rest of the codebase.
     */
    bool public isUpgrading = false;

    /**
     * @notice Standard adapter is deployed once and used for all proxies that are not explicitly
     *         marked as needing a custom adapter. Reduces deployment complexity.
     */
    StandardProxyAdapter public immutable STANDARD_PROXY_ADAPTER;

    /**
     * @param _owner Address of the initial owner of this contract.
     */
    constructor(address _owner) Ownable() {
        STANDARD_PROXY_ADAPTER = new StandardProxyAdapter();
        _transferOwnership(_owner);
    }

    /**
     * @notice Sets the adpater for a given proxy.
     *
     * @param _address Address of the proxy.
     * @param _adapter Adapter for the proxy.
     */
    function setProxyAdapter(address _address, address _adapter) external onlyOwner {
        adapters[_address] = _adapter;
    }

    /**
     * @notice Gets the proxy adapter for a given proxy.
     *
     * @param _address Proxy address to get an adapter for.
     *
     * @return Address of the adapter for the given proxy.
     */
    function getProxyAdapter(address _address) public view returns (address) {
        address adapter = adapters[_address];
        if (adapter == address(0)) {
            adapter = address(STANDARD_PROXY_ADAPTER);
        }

        // Prevent calling adapters with no code, which would silently fail in delegatecall.
        require(
            adapter.code.length > 0,
            "ProxyAdmin: adapter has no code, this is potentially dangerous"
        );

        return adapter;
    }

    /**
     * @notice Returns the implementation of the given proxy address.
     *
     * @param _proxy Address of the proxy to get the implementation of.
     *
     * @return Address of the implementation of the proxy.
     */
    function getProxyImplementation(address _proxy) external view returns (address) {
        address adapter = getProxyAdapter(_proxy);
        adapter.delegatecall(
            abi.encodeCall(
                IProxyAdapter.getProxyImplementation.selector,
                (
                    _proxy
                )
            )
        );
    }

    /**
     * @notice Returns the admin of the given proxy address.
     *
     * @param _proxy Address of the proxy to get the admin of.
     *
     * @return Address of the admin of the proxy.
     */
    function getProxyAdmin(address payable _proxy) external view returns (address) {
        address adapter = getProxyAdapter(_proxy);
        adapter.delegatecall(
            abi.encodeCall(
                IProxyAdapter.getProxyAdmin.selector,
                (
                    _proxy
                )
            )
        );
    }

    /**
     * @notice Updates the admin of the given proxy address.
     *
     * @param _proxy    Address of the proxy to update.
     * @param _newAdmin Address of the new proxy admin.
     */
    function changeProxyAdmin(address payable _proxy, address _newAdmin) external onlyOwner {
        address adapter = getProxyAdapter(_proxy);
        adapter.delegatecall(
            abi.encodeCall(
                IProxyAdapter.changeProxyAdmin.selector,
                (
                    _proxy,
                    _newAdmin
                )
            )
        );
    }

    /**
     * @notice Changes a proxy's implementation contract.
     *
     * @param _proxy          Address of the proxy to upgrade.
     * @param _implementation Address of the new implementation address.
     */
    function upgrade(address payable _proxy, address _implementation) public onlyOwner {
        address adapter = getProxyAdapter(_proxy);
        adapter.delegatecall(
            abi.encodeCall(
                IProxyAdapter.upgrade.selector,
                (
                    _proxy,
                    _implementation
                )
            )
        );
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
        address payable _proxy,
        address _implementation,
        bytes memory _data
    ) external payable onlyOwner {
        address adapter = getProxyAdapter(_proxy);
        adapter.delegatecall{value:msg.value}(
            abi.encodeCall(
                IProxyAdapter.upgradeAndCall.selector,
                (
                    _proxy,
                    _implementation,
                    _data
                )
            )
        );
    }
}
