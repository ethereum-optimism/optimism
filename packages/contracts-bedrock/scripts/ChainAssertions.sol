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
import { console2 as console } from "forge-std/console2.sol";

library ChainAssertions {
    /// @notice Asserts the correctness of an L1 deployment. This function expects that all contracts
    ///         within the `prox` ContractSet are proxies that have been setup and initialized.
    function postDeployAssertions(
        Types.ContractSet memory prox,
        DeployConfig cfg,
        uint256 l2OutputOracleStartingBlockNumber,
        uint256 l2OutputOracleStartingTimestamp,
        Vm vm
    )
        internal
        view
    {
        ResourceMetering.ResourceConfig memory rcfg = SystemConfig(prox.SystemConfig).resourceConfig();
        ResourceMetering.ResourceConfig memory dflt = Constants.DEFAULT_RESOURCE_CONFIG();
        require(keccak256(abi.encode(rcfg)) == keccak256(abi.encode(dflt)));

        checkSystemConfig(prox, cfg, true);
        checkL1CrossDomainMessenger(prox, vm);
        checkL1StandardBridge(prox);
        checkL2OutputOracle(prox, cfg, l2OutputOracleStartingTimestamp, l2OutputOracleStartingBlockNumber);
        checkOptimismMintableERC20Factory(prox);
        checkL1ERC721Bridge(prox);
        checkOptimismPortal(prox, cfg, false);
        checkProtocolVersions(prox, cfg, true);
    }

    /// @notice Asserts that the SystemConfig is setup correctly
    function checkSystemConfig(Types.ContractSet memory _contracts, DeployConfig _cfg, bool _proxy) internal view {
        ISystemConfigV0 config = ISystemConfigV0(_contracts.SystemConfig);
        ResourceMetering.ResourceConfig memory rconfig = Constants.DEFAULT_RESOURCE_CONFIG();

        if (_proxy) {
            require(config.owner() == _cfg.finalSystemOwner());
            require(config.overhead() == _cfg.gasPriceOracleOverhead());
            require(config.scalar() == _cfg.gasPriceOracleScalar());
            require(config.batcherHash() == bytes32(uint256(uint160(_cfg.batchSenderAddress()))));
            require(config.unsafeBlockSigner() == _cfg.p2pSequencerAddress());
        } else {
            require(config.owner() == address(0xdead));
            require(config.overhead() == 0);
            require(config.scalar() == 0);
            require(config.batcherHash() == bytes32(0));
            require(config.unsafeBlockSigner() == address(0));
        }

        ResourceMetering.ResourceConfig memory resourceConfig = config.resourceConfig();
        require(resourceConfig.maxResourceLimit == rconfig.maxResourceLimit);
        require(resourceConfig.elasticityMultiplier == rconfig.elasticityMultiplier);
        require(resourceConfig.baseFeeMaxChangeDenominator == rconfig.baseFeeMaxChangeDenominator);
        require(resourceConfig.systemTxMaxGas == rconfig.systemTxMaxGas);
        require(resourceConfig.minimumBaseFee == rconfig.minimumBaseFee);
        require(resourceConfig.maximumBaseFee == rconfig.maximumBaseFee);
    }

    /// @notice Asserts that the L1CrossDomainMessenger is setup correctly
    function checkL1CrossDomainMessenger(Types.ContractSet memory _contracts, Vm _vm) internal view {
        L1CrossDomainMessenger messenger = L1CrossDomainMessenger(_contracts.L1CrossDomainMessenger);
        require(address(messenger.portal()) == _contracts.OptimismPortal);
        require(address(messenger.PORTAL()) == _contracts.OptimismPortal);
        bytes32 xdmSenderSlot = _vm.load(address(messenger), bytes32(uint256(204)));
        require(address(uint160(uint256(xdmSenderSlot))) == Constants.DEFAULT_L2_SENDER);
    }

    /// @notice Asserts that the L1StandardBridge is setup correctly
    function checkL1StandardBridge(Types.ContractSet memory _contracts) internal view {
        L1StandardBridge bridge = L1StandardBridge(payable(_contracts.L1StandardBridge));
        require(address(bridge.MESSENGER()) == _contracts.L1CrossDomainMessenger);
        require(address(bridge.messenger()) == _contracts.L1CrossDomainMessenger);
        require(address(bridge.OTHER_BRIDGE()) == Predeploys.L2_STANDARD_BRIDGE);
        require(address(bridge.otherBridge()) == Predeploys.L2_STANDARD_BRIDGE);
    }

    /// @notice Asserts that the L2OutputOracle is setup correctly
    function checkL2OutputOracle(
        Types.ContractSet memory _contracts,
        DeployConfig _cfg,
        uint256 _l2OutputOracleStartingBlockNumber,
        uint256 _l2OutputOracleStartingTimestamp
    )
        internal
        view
    {
        L2OutputOracle oracle = L2OutputOracle(_contracts.L2OutputOracle);
        require(oracle.SUBMISSION_INTERVAL() == _cfg.l2OutputOracleSubmissionInterval());
        require(oracle.submissionInterval() == _cfg.l2OutputOracleSubmissionInterval());
        require(oracle.L2_BLOCK_TIME() == _cfg.l2BlockTime());
        require(oracle.l2BlockTime() == _cfg.l2BlockTime());
        require(oracle.PROPOSER() == _cfg.l2OutputOracleProposer());
        require(oracle.proposer() == _cfg.l2OutputOracleProposer());
        require(oracle.CHALLENGER() == _cfg.l2OutputOracleChallenger());
        require(oracle.challenger() == _cfg.l2OutputOracleChallenger());
        require(oracle.FINALIZATION_PERIOD_SECONDS() == _cfg.finalizationPeriodSeconds());
        require(oracle.finalizationPeriodSeconds() == _cfg.finalizationPeriodSeconds());
        require(oracle.startingBlockNumber() == _l2OutputOracleStartingBlockNumber);
        require(oracle.startingTimestamp() == _l2OutputOracleStartingTimestamp);
    }

    /// @notice Asserts that the OptimismMintableERC20Factory is setup correctly
    function checkOptimismMintableERC20Factory(Types.ContractSet memory _contracts) internal view {
        OptimismMintableERC20Factory factory = OptimismMintableERC20Factory(_contracts.OptimismMintableERC20Factory);
        require(factory.BRIDGE() == _contracts.L1StandardBridge);
        require(factory.bridge() == _contracts.L1StandardBridge);
    }

    /// @notice Asserts that the L1ERC721Bridge is setup correctly
    function checkL1ERC721Bridge(Types.ContractSet memory _contracts) internal view {
        L1ERC721Bridge bridge = L1ERC721Bridge(_contracts.L1ERC721Bridge);
        require(address(bridge.MESSENGER()) == _contracts.L1CrossDomainMessenger);
        require(address(bridge.messenger()) == _contracts.L1CrossDomainMessenger);
        require(bridge.OTHER_BRIDGE() == Predeploys.L2_ERC721_BRIDGE);
        require(bridge.otherBridge() == Predeploys.L2_ERC721_BRIDGE);
    }

    /// @notice Asserts the OptimismPortal is setup correctly
    function checkOptimismPortal(
        Types.ContractSet memory _contracts,
        DeployConfig _cfg,
        bool _isPaused
    )
        internal
        view
    {
        OptimismPortal portal = OptimismPortal(payable(_contracts.OptimismPortal));

        address guardian = _cfg.portalGuardian();
        if (guardian.code.length == 0) {
            console.log("Portal guardian has no code: %s", guardian);
        }

        require(address(portal.L2_ORACLE()) == _contracts.L2OutputOracle);
        require(address(portal.l2Oracle()) == _contracts.L2OutputOracle);
        require(portal.GUARDIAN() == _cfg.portalGuardian());
        require(portal.guardian() == _cfg.portalGuardian());
        require(address(portal.SYSTEM_CONFIG()) == _contracts.SystemConfig);
        require(address(portal.systemConfig()) == _contracts.SystemConfig);
        require(portal.paused() == _isPaused);
    }

    /// @notice Asserts that the ProtocolVersions is setup correctly
    function checkProtocolVersions(Types.ContractSet memory _proxies, DeployConfig _cfg, bool _proxy) internal view {
        ProtocolVersions versions = ProtocolVersions(_proxies.ProtocolVersions);
        if (_proxy) {
            require(versions.owner() == _cfg.finalSystemOwner());
            require(ProtocolVersion.unwrap(versions.required()) == _cfg.requiredProtocolVersion());
            require(ProtocolVersion.unwrap(versions.recommended()) == _cfg.recommendedProtocolVersion());
        } else {
            require(versions.owner() == address(0xdead));
            require(ProtocolVersion.unwrap(versions.required()) == 0);
            require(ProtocolVersion.unwrap(versions.recommended()) == 0);
        }
    }
}
