// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IProxyAdapter } from "./IProxyAdapter.sol";
import { Proxy } from "../Proxy.sol";

/**
 * @title IStaticERC1967Proxy
 * @notice IStaticERC1967Proxy is a static version of the ERC1967 proxy interface.
 */
interface IStaticERC1967Proxy {
    function implementation() external view returns (address);

    function admin() external view returns (address);
}

contract StandardProxyAdapter is IProxyAdapter {
    function getProxyImplementation(address payable proxy) external view returns (address) {
        return IStaticERC1967Proxy(proxy).implementation();
    }

    function getProxyAdmin(address payable _proxy) external view returns (address) {
        return IStaticERC1967Proxy(_proxy).admin();
    }

    function changeProxyAdmin(address payable _proxy, address _newAdmin) external {
        Proxy(_proxy).changeAdmin(_newAdmin);
    }

    function upgrade(address payable _proxy, address _implementation) external {
        Proxy(_proxy).upgradeTo(_implementation);
    }

    function upgradeAndCall(address payable _proxy, address _implementation, bytes calldata _data) external payable {
        Proxy(_proxy).upgradeToAndCall{ value: msg.value }(_implementation, _data);
    }
}
