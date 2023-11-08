// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { DeployConfig } from "scripts/DeployConfig.s.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { Constants } from "src/libraries/Constants.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { ProtocolVersion, ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Types } from "scripts/Types.sol";
import { Vm } from "forge-std/Vm.sol";
import { ISystemConfigV0 } from "scripts/interfaces/ISystemConfigV0.sol";

library ChainAssertions {
    /// @notice Asserts the correctness of an L1 deployment
    function postDeployAssertions(
        Types.ContractSet memory prox,
        DeployConfig cfg,
        uint256 l2OutputOracleStartingTimestamp,
        Vm vm
    )
        internal
        view
    {
        ResourceMetering.ResourceConfig memory rcfg = SystemConfig(prox.SystemConfig).resourceConfig();
        ResourceMetering.ResourceConfig memory dflt = Constants.DEFAULT_RESOURCE_CONFIG();
        require(keccak256(abi.encode(rcfg)) == keccak256(abi.encode(dflt)));

        checkSystemConfig(prox, cfg);
        checkL1CrossDomainMessenger(prox, vm);
        checkL1StandardBridge(prox, vm);
        checkL2OutputOracle(prox, cfg, l2OutputOracleStartingTimestamp);
        checkOptimismMintableERC20Factory(prox);
        checkL1ERC721Bridge(prox);
        checkOptimismPortal(prox, cfg);
        checkProtocolVersions(prox, cfg);
    }

    /// @notice Asserts that the SystemConfig is setup correctly
    function checkSystemConfig(Types.ContractSet memory proxies, DeployConfig cfg) internal view {
        ISystemConfigV0 config = ISystemConfigV0(proxies.SystemConfig);
        require(config.owner() == cfg.finalSystemOwner());
        require(config.overhead() == cfg.gasPriceOracleOverhead());
        require(config.scalar() == cfg.gasPriceOracleScalar());
        require(config.batcherHash() == bytes32(uint256(uint160(cfg.batchSenderAddress()))));
        require(config.unsafeBlockSigner() == cfg.p2pSequencerAddress());

        ResourceMetering.ResourceConfig memory rconfig = Constants.DEFAULT_RESOURCE_CONFIG();
        ResourceMetering.ResourceConfig memory resourceConfig = config.resourceConfig();
        require(resourceConfig.maxResourceLimit == rconfig.maxResourceLimit);
        require(resourceConfig.elasticityMultiplier == rconfig.elasticityMultiplier);
        require(resourceConfig.baseFeeMaxChangeDenominator == rconfig.baseFeeMaxChangeDenominator);
        require(resourceConfig.systemTxMaxGas == rconfig.systemTxMaxGas);
        require(resourceConfig.minimumBaseFee == rconfig.minimumBaseFee);
        require(resourceConfig.maximumBaseFee == rconfig.maximumBaseFee);
    }

    /// @notice Asserts that the L1CrossDomainMessenger is setup correctly
    function checkL1CrossDomainMessenger(Types.ContractSet memory proxies, Vm vm) internal view {
        L1CrossDomainMessenger messenger = L1CrossDomainMessenger(proxies.L1CrossDomainMessenger);
        require(address(messenger.portal()) == proxies.OptimismPortal);
        require(address(messenger.PORTAL()) == proxies.OptimismPortal);
        bytes32 xdmSenderSlot = vm.load(address(messenger), bytes32(uint256(204)));
        require(address(uint160(uint256(xdmSenderSlot))) == Constants.DEFAULT_L2_SENDER);
    }

    /// @notice Asserts that the L1StandardBridge is setup correctly
    function checkL1StandardBridge(Types.ContractSet memory proxies, Vm vm) internal view {
        L1StandardBridge bridge = L1StandardBridge(payable(proxies.L1StandardBridge));
        require(address(bridge.MESSENGER()) == proxies.L1CrossDomainMessenger);
        require(address(bridge.messenger()) == proxies.L1CrossDomainMessenger);
        require(address(bridge.OTHER_BRIDGE()) == Predeploys.L2_STANDARD_BRIDGE);
        require(address(bridge.otherBridge()) == Predeploys.L2_STANDARD_BRIDGE);
        // Ensures that the legacy slot is modified correctly. This will fail
        // during predeployment simulation on OP Mainnet if there is a bug.
        bytes32 slot0 = vm.load(address(bridge), bytes32(uint256(0)));
        require(slot0 == bytes32(uint256(Constants.INITIALIZER)));
    }

    /// @notice Asserts that the L2OutputOracle is setup correctly
    function checkL2OutputOracle(
        Types.ContractSet memory proxies,
        DeployConfig cfg,
        uint256 l2OutputOracleStartingTimestamp
    )
        internal
        view
    {
        L2OutputOracle oracle = L2OutputOracle(proxies.L2OutputOracle);
        require(oracle.SUBMISSION_INTERVAL() == cfg.l2OutputOracleSubmissionInterval());
        require(oracle.submissionInterval() == cfg.l2OutputOracleSubmissionInterval());
        require(oracle.L2_BLOCK_TIME() == cfg.l2BlockTime());
        require(oracle.l2BlockTime() == cfg.l2BlockTime());
        require(oracle.PROPOSER() == cfg.l2OutputOracleProposer());
        require(oracle.proposer() == cfg.l2OutputOracleProposer());
        require(oracle.CHALLENGER() == cfg.l2OutputOracleChallenger());
        require(oracle.challenger() == cfg.l2OutputOracleChallenger());
        require(oracle.FINALIZATION_PERIOD_SECONDS() == cfg.finalizationPeriodSeconds());
        require(oracle.finalizationPeriodSeconds() == cfg.finalizationPeriodSeconds());
        require(oracle.startingBlockNumber() == cfg.l2OutputOracleStartingBlockNumber());
        require(oracle.startingTimestamp() == l2OutputOracleStartingTimestamp);
    }

    /// @notice Asserts that the OptimismMintableERC20Factory is setup correctly
    function checkOptimismMintableERC20Factory(Types.ContractSet memory proxies) internal view {
        OptimismMintableERC20Factory factory = OptimismMintableERC20Factory(proxies.OptimismMintableERC20Factory);
        require(factory.BRIDGE() == proxies.L1StandardBridge);
        require(factory.bridge() == proxies.L1StandardBridge);
    }

    /// @notice Asserts that the L1ERC721Bridge is setup correctly
    function checkL1ERC721Bridge(Types.ContractSet memory proxies) internal view {
        L1ERC721Bridge bridge = L1ERC721Bridge(proxies.L1ERC721Bridge);
        require(address(bridge.MESSENGER()) == proxies.L1CrossDomainMessenger);
        require(bridge.OTHER_BRIDGE() == Predeploys.L2_ERC721_BRIDGE);
    }

    /// @notice Asserts the OptimismPortal is setup correctly
    function checkOptimismPortal(Types.ContractSet memory proxies, DeployConfig cfg) internal view {
        OptimismPortal portal = OptimismPortal(payable(proxies.OptimismPortal));
        require(address(portal.L2_ORACLE()) == proxies.L2OutputOracle);
        require(portal.GUARDIAN() == cfg.portalGuardian());
        require(address(portal.SYSTEM_CONFIG()) == proxies.SystemConfig);
        require(portal.paused() == false);
    }

    /// @notice Asserts that the ProtocolVersions is setup correctly
    function checkProtocolVersions(Types.ContractSet memory proxies, DeployConfig cfg) internal view {
        ProtocolVersions versions = ProtocolVersions(proxies.ProtocolVersions);
        require(versions.owner() == cfg.finalSystemOwner());
        require(ProtocolVersion.unwrap(versions.required()) == cfg.requiredProtocolVersion());
        require(ProtocolVersion.unwrap(versions.recommended()) == cfg.recommendedProtocolVersion());
    }
}
