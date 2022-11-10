// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IProxyAdapter } from "./IProxyAdapter.sol";
import { AddressManager } from "../../legacy/AddressManager.sol";

contract ResolvedDelegateProxyAdapter is IProxyAdapter {
    AddressManager public immutable ADDRESS_MANAGER;

    constructor(
        AddressManager _addressManager
    ) {
        ADDRESS_MANAGER = _addressManager;
    }

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
