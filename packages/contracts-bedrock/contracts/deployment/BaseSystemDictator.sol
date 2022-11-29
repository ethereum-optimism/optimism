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
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title BaseSystemDictator
 * @notice The BaseSystemDictator is a base contract for SystemDictator contracts.
 */
contract BaseSystemDictator is Ownable {
    /**
     * @notice Basic system configuration.
     */
    struct GlobalConfig {
        AddressManager addressManager;
        ProxyAdmin proxyAdmin;
        address controller;
        address finalOwner;
    }

    /**
     * @notice Set of proxy addresses.
     */
    struct ProxyAddressConfig {
        address l2OutputOracleProxy;
        address optimismPortalProxy;
        address l1CrossDomainMessengerProxy;
        address l1StandardBridgeProxy;
        address optimismMintableERC20FactoryProxy;
        address l1ERC721BridgeProxy;
        address systemConfigProxy;
    }

    /**
     * @notice Set of implementation addresses.
     */
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

    /**
     * @notice Dynamic L2OutputOracle config.
     */
    struct L2OutputOracleDynamicConfig {
        uint256 l2OutputOracleStartingBlockNumber;
        uint256 l2OutputOracleStartingTimestamp;
    }

    /**
     * @notice Values for the system config contract.
     */
    struct SystemConfigConfig {
        address owner;
        uint256 overhead;
        uint256 scalar;
        bytes32 batcherHash;
        uint64 gasLimit;
    }

    /**
     * @notice Combined system configuration.
     */
    struct DeployConfig {
        GlobalConfig globalConfig;
        ProxyAddressConfig proxyAddressConfig;
        ImplementationAddressConfig implementationAddressConfig;
        SystemConfigConfig systemConfigConfig;
    }

    /**
     * @notice System configuration.
     */
    DeployConfig public config;

    /**
     * @notice Dynamic configuration for the L2OutputOracle.
     */
    L2OutputOracleDynamicConfig public l2OutputOracleDynamicConfig;

    /**
     * @notice Current step;
     */
    uint8 public currentStep = 1;

    /**
     * @notice Whether or not dynamic config has been set.
     */
    bool public dynamicConfigSet;

    /**
     * @notice Checks that the current step is the expected step, then bumps the current step.
     *
     * @param _step Current step.
     */
    modifier step(uint8 _step) {
        require(currentStep == _step, "BaseSystemDictator: incorrect step");
        _;
        currentStep++;
    }

    /**
     * @param _config System configuration.
     */
    constructor(DeployConfig memory _config) Ownable() {
        config = _config;
        _transferOwnership(config.globalConfig.controller);
    }

    /**
     * @notice Allows the owner to update dynamic L2OutputOracle config.
     *
     * @param _l2OutputOracleDynamicConfig Dynamic L2OutputOracle config.
     */
    function updateL2OutputOracleDynamicConfig(
        L2OutputOracleDynamicConfig memory _l2OutputOracleDynamicConfig
    ) external onlyOwner {
        l2OutputOracleDynamicConfig = _l2OutputOracleDynamicConfig;
        dynamicConfigSet = true;
    }
}
