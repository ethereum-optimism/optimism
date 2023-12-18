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
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
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
        Types.ContractSet memory _prox,
        DeployConfig _cfg,
        uint256 _l2OutputOracleStartingTimestamp,
        Vm _vm
    )
        internal
        view
    {
        console.log("Running post-deploy assertions");
        ResourceMetering.ResourceConfig memory rcfg = SystemConfig(_prox.SystemConfig).resourceConfig();
        ResourceMetering.ResourceConfig memory dflt = Constants.DEFAULT_RESOURCE_CONFIG();
        require(keccak256(abi.encode(rcfg)) == keccak256(abi.encode(dflt)));

        checkSystemConfig({ _contracts: _prox, _cfg: _cfg, _isProxy: true });
        checkL1CrossDomainMessenger({ _contracts: _prox, _vm: _vm, _isProxy: true });
        checkL1StandardBridge({ _contracts: _prox, _isProxy: true });
        checkL2OutputOracle({
            _contracts: _prox,
            _cfg: _cfg,
            _l2OutputOracleStartingTimestamp: _l2OutputOracleStartingTimestamp,
            _isProxy: true
        });
        checkOptimismMintableERC20Factory({ _contracts: _prox, _isProxy: true });
        checkL1ERC721Bridge({ _contracts: _prox, _isProxy: true });
        checkOptimismPortal({ _contracts: _prox, _cfg: _cfg, _isProxy: true });
        checkProtocolVersions({ _contracts: _prox, _cfg: _cfg, _isProxy: true });
    }

    /// @notice Asserts that the SystemConfig is setup correctly
    function checkSystemConfig(Types.ContractSet memory _contracts, DeployConfig _cfg, bool _isProxy) internal view {
        console.log("Running chain assertions on the SystemConfig");
        ISystemConfigV0 config = ISystemConfigV0(_contracts.SystemConfig);

        if (_isProxy) {
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
    function checkL1CrossDomainMessenger(Types.ContractSet memory _contracts, Vm _vm, bool _isProxy) internal view {
        console.log("Running chain assertions on the L1CrossDomainMessenger");
        L1CrossDomainMessenger messenger = L1CrossDomainMessenger(_contracts.L1CrossDomainMessenger);

        require(address(messenger.OTHER_MESSENGER()) == Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        require(address(messenger.otherMessenger()) == Predeploys.L2_CROSS_DOMAIN_MESSENGER);

        if (_isProxy) {
            require(address(messenger.PORTAL()) == _contracts.OptimismPortal);
            require(address(messenger.portal()) == _contracts.OptimismPortal);
            require(address(messenger.superchainConfig()) == _contracts.SuperchainConfig);
            bytes32 xdmSenderSlot = _vm.load(address(messenger), bytes32(uint256(204)));
            require(address(uint160(uint256(xdmSenderSlot))) == Constants.DEFAULT_L2_SENDER);
        } else {
            require(address(messenger.PORTAL()) == address(0));
            require(address(messenger.portal()) == address(0));
            require(address(messenger.superchainConfig()) == address(0));
        }
    }

    /// @notice Asserts that the L1StandardBridge is setup correctly
    function checkL1StandardBridge(Types.ContractSet memory _contracts, bool _isProxy) internal view {
        console.log("Running chain assertions on the L1StandardBridge");
        L1StandardBridge bridge = L1StandardBridge(payable(_contracts.L1StandardBridge));
        require(address(bridge.MESSENGER()) == _contracts.L1CrossDomainMessenger);
        require(address(bridge.messenger()) == _contracts.L1CrossDomainMessenger);
        require(address(bridge.OTHER_BRIDGE()) == Predeploys.L2_STANDARD_BRIDGE);
        require(address(bridge.otherBridge()) == Predeploys.L2_STANDARD_BRIDGE);
        if (_isProxy) {
            require(address(bridge.superchainConfig()) == _contracts.SuperchainConfig);
        } else {
            require(address(bridge.superchainConfig()) == address(0));
        }
    }

    /// @notice Asserts that the L2OutputOracle is setup correctly
    function checkL2OutputOracle(
        Types.ContractSet memory _contracts,
        DeployConfig _cfg,
        uint256 _l2OutputOracleStartingTimestamp,
        bool _isProxy
    )
        internal
        view
    {
        console.log("Running chain assertions on the L2OutputOracle");
        L2OutputOracle oracle = L2OutputOracle(_contracts.L2OutputOracle);

        if (_isProxy) {
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
            require(oracle.startingBlockNumber() == _cfg.l2OutputOracleStartingBlockNumber());
            require(oracle.startingTimestamp() == _l2OutputOracleStartingTimestamp);
        } else {
            require(oracle.SUBMISSION_INTERVAL() == 1);
            require(oracle.submissionInterval() == 1);
            require(oracle.L2_BLOCK_TIME() == 1);
            require(oracle.l2BlockTime() == 1);
            require(oracle.PROPOSER() == address(0));
            require(oracle.proposer() == address(0));
            require(oracle.CHALLENGER() == address(0));
            require(oracle.challenger() == address(0));
            require(oracle.FINALIZATION_PERIOD_SECONDS() == 0);
            require(oracle.finalizationPeriodSeconds() == 0);
            require(oracle.startingBlockNumber() == 0);
            require(oracle.startingTimestamp() == 0);
        }
    }

    /// @notice Asserts that the OptimismMintableERC20Factory is setup correctly
    function checkOptimismMintableERC20Factory(Types.ContractSet memory _contracts, bool _isProxy) internal view {
        console.log("Running chain assertions on the OptimismMintableERC20Factory");
        OptimismMintableERC20Factory factory = OptimismMintableERC20Factory(_contracts.OptimismMintableERC20Factory);

        if (_isProxy) {
            require(factory.BRIDGE() == _contracts.L1StandardBridge);
            require(factory.bridge() == _contracts.L1StandardBridge);
        } else {
            require(factory.BRIDGE() == address(0));
            require(factory.bridge() == address(0));
        }
    }

    /// @notice Asserts that the L1ERC721Bridge is setup correctly
    function checkL1ERC721Bridge(
        Types.ContractSet memory _contracts,
        bool _isProxy,
        bool _isInitialized
    )
        internal
        view
    {
        console.log("Running chain assertions on the L1ERC721Bridge");
        L1ERC721Bridge bridge = L1ERC721Bridge(_contracts.L1ERC721Bridge);

        if (!_isInitialized) {
            require(address(bridge.MESSENGER()) == address(0));
            require(address(bridge.messenger()) == address(0));
            require(bridge.OTHER_BRIDGE() == Predeploys.L2_ERC721_BRIDGE);
            require(bridge.otherBridge() == Predeploys.L2_ERC721_BRIDGE);
            require(address(bridge.superchainConfig()) == address(0));
        } else {
            require(address(bridge.MESSENGER()) == _contracts.L1CrossDomainMessenger);
            require(address(bridge.messenger()) == _contracts.L1CrossDomainMessenger);
            require(bridge.OTHER_BRIDGE() == Predeploys.L2_ERC721_BRIDGE);
            require(bridge.otherBridge() == Predeploys.L2_ERC721_BRIDGE);
            if (_isProxy) {
                require(address(bridge.superchainConfig()) == _contracts.SuperchainConfig);
            } else {
                require(address(bridge.superchainConfig()) == address(0));
            }
        }
    }

    /// @notice Asserts the OptimismPortal is setup correctly
    function checkOptimismPortal(Types.ContractSet memory _contracts, DeployConfig _cfg, bool _isProxy) internal view {
        console.log("Running chain assertions on the OptimismPortal");

        OptimismPortal portal = OptimismPortal(payable(_contracts.OptimismPortal));

        address guardian = _cfg.superchainConfigGuardian();
        if (guardian.code.length == 0) {
            console.log("Guardian has no code: %s", guardian);
        }

        if (_isProxy) {
            require(address(portal.L2_ORACLE()) == _contracts.L2OutputOracle);
            require(address(portal.l2Oracle()) == _contracts.L2OutputOracle);
            require(address(portal.SYSTEM_CONFIG()) == _contracts.SystemConfig);
            require(address(portal.systemConfig()) == _contracts.SystemConfig);

            require(portal.GUARDIAN() == _cfg.superchainConfigGuardian());
            require(portal.guardian() == _cfg.superchainConfigGuardian());
            require(address(portal.superchainConfig()) == address(_contracts.SuperchainConfig));
            require(portal.paused() == SuperchainConfig(_contracts.SuperchainConfig).paused());
        } else {
            require(address(portal.L2_ORACLE()) == address(0));
            require(address(portal.l2Oracle()) == address(0));
            require(address(portal.SYSTEM_CONFIG()) == address(0));
            require(address(portal.systemConfig()) == address(0));
            require(address(portal.superchainConfig()) == address(0));
        }
    }

    /// @notice Asserts that the ProtocolVersions is setup correctly
    function checkProtocolVersions(
        Types.ContractSet memory _contracts,
        DeployConfig _cfg,
        bool _isProxy
    )
        internal
        view
    {
        console.log("Running chain assertions on the ProtocolVersions");
        ProtocolVersions versions = ProtocolVersions(_contracts.ProtocolVersions);
        if (_isProxy) {
            require(versions.owner() == _cfg.finalSystemOwner());
            require(ProtocolVersion.unwrap(versions.required()) == _cfg.requiredProtocolVersion());
            require(ProtocolVersion.unwrap(versions.recommended()) == _cfg.recommendedProtocolVersion());
        } else {
            require(versions.owner() == address(0xdead));
            require(ProtocolVersion.unwrap(versions.required()) == 0);
            require(ProtocolVersion.unwrap(versions.recommended()) == 0);
        }
    }

    /// @notice Asserts that the SuperchainConfig is setup correctly
    function checkSuperchainConfig(
        Types.ContractSet memory _contracts,
        DeployConfig _cfg,
        bool _isPaused
    )
        internal
        view
    {
        console.log("Running chain assertions on the SuperchainConfig");
        SuperchainConfig superchainConfig = SuperchainConfig(_contracts.SuperchainConfig);
        require(superchainConfig.guardian() == _cfg.superchainConfigGuardian());
        require(superchainConfig.paused() == _isPaused);
    }
}
