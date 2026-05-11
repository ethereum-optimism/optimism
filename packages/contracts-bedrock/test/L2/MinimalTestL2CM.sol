// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

interface IProxy {
    function upgradeTo(address _implementation) external;
}

/// @title MinimalTestL2CM
/// @notice Test-only minimal L2ContractsManager that upgrades a single predeploy proxy.
///         Designed to be called via DELEGATECALL from L2ProxyAdmin.upgradePredeploys():
///         in that context address(this) == L2ProxyAdmin, which is the proxy admin, so
///         the upgradeTo call is authorized.
contract MinimalTestL2CM {
    address public immutable proxy;
    address public immutable newImpl;

    constructor(address _proxy, address _newImpl) {
        proxy = _proxy;
        newImpl = _newImpl;
    }

    function upgrade() external {
        IProxy(proxy).upgradeTo(newImpl);
    }
}
