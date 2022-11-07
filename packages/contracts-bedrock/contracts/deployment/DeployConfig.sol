// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ProxyAdmin } from "../universal/ProxyAdmin.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { L1CrossDomainMessenger } from "../L1/L1CrossDomainMessenger.sol";
import { L1StandardBridge } from "../L1/L1StandardBridge.sol";
import { L1ERC721Bridge } from "../L1/L1ERC721Bridge.sol";
import { SystemConfig } from "../L1/SystemConfig.sol";
import { OptimismMintableERC20Factory } from "../universal/OptimismMintableERC20Factory.sol";
import { AddressManager } from "../legacy/AddressManager.sol";
import { PortalSender } from "./PortalSender.sol";

struct GlobalConfig {
    AddressManager addressManager;
    ProxyAdmin proxyAdmin;
    address controller;
    address finalOwner;
}

struct ProxyAddressConfig {
    address l2OutputOracleProxy;
    address optimismPortalProxy;
    address l1CrossDomainMessengerProxy;
    address l1StandardBridgeProxy;
    address optimismMintableERC20FactoryProxy;
    address l1ERC721BridgeProxy;
    address systemConfigProxy;
}

struct ImplementationAddressConfig {
    L2OutputOracle l2OutputOracleImpl;
    OptimismPortal optimismPortalImpl;
    L1CrossDomainMessenger l1CrossDomainMessengerImpl;
    L1StandardBridge l1StandardBridgeImpl;
    OptimismMintableERC20Factory optimismMintableERC20FactoryImpl;
    L1ERC721Bridge l1ERC721BridgeImpl;
    PortalSender portalSenderImpl;
    SystemConfig systemConfigImpl;
}

struct L2OutputOracleConfig {
    bytes32 l2OutputOracleGenesisL2Output;
    address l2OutputOracleProposer;
    address l2OutputOracleOwner;
}

struct SystemConfigConfig {
    address owner;
    uint256 overhead;
    uint256 scalar;
    bytes32 batcherHash;
}

struct DeployConfig {
    GlobalConfig globalConfig;
    ProxyAddressConfig proxyAddressConfig;
    ImplementationAddressConfig implementationAddressConfig;
    L2OutputOracleConfig l2OutputOracleConfig;
    SystemConfigConfig systemConfigConfig;
}
