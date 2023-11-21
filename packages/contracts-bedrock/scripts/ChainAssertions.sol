// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { DeployConfig } from "scripts/DeployConfig.s.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Types } from "scripts/Types.sol";
import { AbiSpec } from "scripts/AbiSpec.s.sol";
import { Vm } from "forge-std/Vm.sol";
import { console2 as console } from "forge-std/console2.sol";

library ChainAssertions {
    /// @notice Asserts the correctness of an L1 deployment
    function postDeployAssertions(
        AbiSpec _spec,
        Types.ContractSet memory _prox,
        DeployConfig _cfg,
        uint256 _l2OutputOracleStartingTimestamp,
        Vm _vm
    )
        internal
        view
    {
        ResourceMetering.ResourceConfig memory rcfg = abi.decode(
            methodCall(_spec, _prox.SystemConfig, "SystemConfig", "resourceConfig()"), (ResourceMetering.ResourceConfig)
        );
        ResourceMetering.ResourceConfig memory dflt = Constants.DEFAULT_RESOURCE_CONFIG();
        require(keccak256(abi.encode(rcfg)) == keccak256(abi.encode(dflt)));

        checkSystemConfig(_spec, _prox, _cfg);
        checkL1CrossDomainMessenger(_spec, _vm, _prox);
        checkL1StandardBridge(_spec, _prox);
        checkL2OutputOracle(_spec, _vm, _prox, _cfg, _l2OutputOracleStartingTimestamp);
        checkOptimismMintableERC20Factory(_spec, _prox);
        checkL1ERC721Bridge(_spec, _prox);
        checkOptimismPortal(_spec, _vm, _prox, _cfg);
        checkProtocolVersions(_spec, _prox, _cfg);
    }

    /// @notice Asserts that the SystemConfig is setup correctly
    function checkSystemConfig(AbiSpec _spec, Types.ContractSet memory _proxies, DeployConfig _cfg) internal view {
        require(
            toAddress(methodCall(_spec, _proxies.SystemConfig, "SystemConfig", "owner()")) == _cfg.finalSystemOwner()
        );
        require(
            toUint256(methodCall(_spec, _proxies.SystemConfig, "SystemConfig", "overhead()"))
                == _cfg.gasPriceOracleOverhead()
        );
        require(
            toUint256(methodCall(_spec, _proxies.SystemConfig, "SystemConfig", "scalar()"))
                == _cfg.gasPriceOracleScalar()
        );
        require(
            bytes32(methodCall(_spec, _proxies.SystemConfig, "SystemConfig", "batcherHash()"))
                == bytes32(uint256(uint160(_cfg.batchSenderAddress())))
        );
        require(
            toAddress(methodCall(_spec, _proxies.SystemConfig, "SystemConfig", "unsafeBlockSigner()"))
                == _cfg.p2pSequencerAddress()
        );
        ResourceMetering.ResourceConfig memory rconfig = Constants.DEFAULT_RESOURCE_CONFIG();
        ResourceMetering.ResourceConfig memory resourceConfig = abi.decode(
            methodCall(_spec, _proxies.SystemConfig, "SystemConfig", "resourceConfig()"),
            (ResourceMetering.ResourceConfig)
        );
        require(resourceConfig.maxResourceLimit == rconfig.maxResourceLimit);
        require(resourceConfig.elasticityMultiplier == rconfig.elasticityMultiplier);
        require(resourceConfig.baseFeeMaxChangeDenominator == rconfig.baseFeeMaxChangeDenominator);
        require(resourceConfig.systemTxMaxGas == rconfig.systemTxMaxGas);
        require(resourceConfig.minimumBaseFee == rconfig.minimumBaseFee);
        require(resourceConfig.maximumBaseFee == rconfig.maximumBaseFee);
    }

    /// @notice Asserts that the L1CrossDomainMessenger is setup correctly
    function checkL1CrossDomainMessenger(AbiSpec _spec, Vm _vm, Types.ContractSet memory _proxies) internal view {
        require(
            toAddress(methodCall(_spec, _proxies.L1CrossDomainMessenger, "L1CrossDomainMessenger", "portal()"))
                == _proxies.OptimismPortal
        );
        require(
            toAddress(methodCall(_spec, _proxies.L1CrossDomainMessenger, "L1CrossDomainMessenger", "PORTAL()"))
                == _proxies.OptimismPortal
        );
        require(
            loadAddress(_spec, _vm, _proxies.L1CrossDomainMessenger, "L1CrossDomainMessenger", "xDomainMsgSender")
                == Constants.DEFAULT_L2_SENDER
        );
    }

    /// @notice Asserts that the L1StandardBridge is setup correctly
    function checkL1StandardBridge(AbiSpec _spec, Types.ContractSet memory _proxies) internal view {
        require(
            toAddress(methodCall(_spec, _proxies.L1StandardBridge, "L1StandardBridge", "MESSENGER()"))
                == _proxies.L1CrossDomainMessenger
        );
        require(
            toAddress(methodCall(_spec, _proxies.L1StandardBridge, "L1StandardBridge", "OTHER_BRIDGE()"))
                == Predeploys.L2_STANDARD_BRIDGE
        );
        require(
            toAddress(methodCall(_spec, _proxies.L1StandardBridge, "L1StandardBridge", "otherBridge()"))
                == Predeploys.L2_STANDARD_BRIDGE
        );
    }

    /// @notice Asserts that the L2OutputOracle is setup correctly
    function checkL2OutputOracle(
        AbiSpec _spec,
        Vm _vm,
        Types.ContractSet memory _proxies,
        DeployConfig _cfg,
        uint256 _l2OutputOracleStartingTimestamp
    )
        internal
        view
    {
        require(
            toUint256(methodCall(_spec, _proxies.L2OutputOracle, "L2OutputOracle", "SUBMISSION_INTERVAL()"))
                == _cfg.l2OutputOracleSubmissionInterval()
        );
        require(
            toUint256(methodCall(_spec, _proxies.L2OutputOracle, "L2OutputOracle", "submissionInterval()"))
                == _cfg.l2OutputOracleSubmissionInterval()
        );
        require(
            toUint256(methodCall(_spec, _proxies.L2OutputOracle, "L2OutputOracle", "L2_BLOCK_TIME()"))
                == _cfg.l2BlockTime()
        );
        require(
            toUint256(methodCall(_spec, _proxies.L2OutputOracle, "L2OutputOracle", "l2BlockTime()"))
                == _cfg.l2BlockTime()
        );
        require(
            toAddress(methodCall(_spec, _proxies.L2OutputOracle, "L2OutputOracle", "PROPOSER()"))
                == _cfg.l2OutputOracleProposer()
        );
        require(
            toAddress(methodCall(_spec, _proxies.L2OutputOracle, "L2OutputOracle", "proposer()"))
                == _cfg.l2OutputOracleProposer()
        );
        require(
            toAddress(methodCall(_spec, _proxies.L2OutputOracle, "L2OutputOracle", "CHALLENGER()"))
                == _cfg.l2OutputOracleChallenger()
        );
        require(
            toAddress(methodCall(_spec, _proxies.L2OutputOracle, "L2OutputOracle", "challenger()"))
                == _cfg.l2OutputOracleChallenger()
        );
        require(
            toUint256(methodCall(_spec, _proxies.L2OutputOracle, "L2OutputOracle", "FINALIZATION_PERIOD_SECONDS()"))
                == _cfg.finalizationPeriodSeconds()
        );
        require(
            toUint256(methodCall(_spec, _proxies.L2OutputOracle, "L2OutputOracle", "finalizationPeriodSeconds()"))
                == _cfg.finalizationPeriodSeconds()
        );
        require(
            toUint256(methodCall(_spec, _proxies.L2OutputOracle, "L2OutputOracle", "startingBlockNumber()"))
                == _cfg.l2OutputOracleStartingBlockNumber()
        );
        require(
            loadUint256(_spec, _vm, _proxies.L2OutputOracle, "L2OutputOracle", "startingBlockNumber")
                == _cfg.l2OutputOracleStartingBlockNumber()
        );
        require(
            toUint256(methodCall(_spec, _proxies.L2OutputOracle, "L2OutputOracle", "startingTimestamp()"))
                == _l2OutputOracleStartingTimestamp
        );
        require(
            loadUint256(_spec, _vm, _proxies.L2OutputOracle, "L2OutputOracle", "startingTimestamp")
                == _l2OutputOracleStartingTimestamp
        );
    }

    /// @notice Asserts that the OptimismMintableERC20Factory is setup correctly
    function checkOptimismMintableERC20Factory(AbiSpec _spec, Types.ContractSet memory _proxies) internal view {
        require(
            toAddress(
                methodCall(_spec, _proxies.OptimismMintableERC20Factory, "OptimismMintableERC20Factory", "BRIDGE()")
            ) == _proxies.L1StandardBridge
        );
        require(
            toAddress(
                methodCall(_spec, _proxies.OptimismMintableERC20Factory, "OptimismMintableERC20Factory", "bridge()")
            ) == _proxies.L1StandardBridge
        );
    }

    /// @notice Asserts that the L1ERC721Bridge is setup correctly
    function checkL1ERC721Bridge(AbiSpec _spec, Types.ContractSet memory _proxies) internal view {
        require(
            toAddress(methodCall(_spec, _proxies.L1ERC721Bridge, "L1ERC721Bridge", "MESSENGER()"))
                == _proxies.L1CrossDomainMessenger
        );
        require(
            toAddress(methodCall(_spec, _proxies.L1ERC721Bridge, "L1ERC721Bridge", "OTHER_BRIDGE()"))
                == Predeploys.L2_ERC721_BRIDGE
        );
        require(
            toAddress(methodCall(_spec, _proxies.L1ERC721Bridge, "L1ERC721Bridge", "otherBridge()"))
                == Predeploys.L2_ERC721_BRIDGE
        );
    }

    /// @notice Asserts the OptimismPortal is setup correctly
    function checkOptimismPortal(
        AbiSpec _spec,
        Vm _vm,
        Types.ContractSet memory _proxies,
        DeployConfig _cfg
    )
        internal
        view
    {
        address guardian = _cfg.portalGuardian();
        if (guardian.code.length == 0) {
            console.log("Portal guardian has no code: %s", guardian);
        }

        require(
            toAddress(methodCall(_spec, _proxies.OptimismPortal, "OptimismPortal", "L2_ORACLE()"))
                == _proxies.L2OutputOracle
        );
        require(
            toAddress(methodCall(_spec, _proxies.OptimismPortal, "OptimismPortal", "l2Oracle()"))
                == _proxies.L2OutputOracle
        );
        require(
            toAddress(methodCall(_spec, _proxies.OptimismPortal, "OptimismPortal", "GUARDIAN()"))
                == _cfg.portalGuardian()
        );
        require(
            toAddress(methodCall(_spec, _proxies.OptimismPortal, "OptimismPortal", "guardian()"))
                == _cfg.portalGuardian()
        );
        require(
            toAddress(methodCall(_spec, _proxies.OptimismPortal, "OptimismPortal", "SYSTEM_CONFIG()"))
                == _proxies.SystemConfig
        );
        require(
            toAddress(methodCall(_spec, _proxies.OptimismPortal, "OptimismPortal", "systemConfig()"))
                == _proxies.SystemConfig
        );
        require(toUint256(methodCall(_spec, _proxies.OptimismPortal, "OptimismPortal", "paused()")) == 0);
        require(loadUint8(_spec, _vm, _proxies.OptimismPortal, "OptimismPortal", "paused") == 0);
    }

    /// @notice Asserts that the ProtocolVersions is setup correctly
    function checkProtocolVersions(AbiSpec _spec, Types.ContractSet memory _proxies, DeployConfig _cfg) internal view {
        require(
            toAddress(methodCall(_spec, _proxies.ProtocolVersions, "ProtocolVersions", "owner()"))
                == _cfg.finalSystemOwner()
        );
        require(
            toUint256(methodCall(_spec, _proxies.ProtocolVersions, "ProtocolVersions", "required()"))
                == _cfg.requiredProtocolVersion()
        );
        require(
            toUint256(methodCall(_spec, _proxies.ProtocolVersions, "ProtocolVersions", "recommended()"))
                == _cfg.recommendedProtocolVersion()
        );
    }

    /// @notice Reads the slot of a contract per ABI spec
    function loadSlot(
        AbiSpec _spec,
        Vm _vm,
        address _contractAddress,
        string memory _contractName,
        string memory _identifier
    )
        internal
        view
        returns (bytes32 val_)
    {
        (uint256 slot, uint8 offset) = _spec.slot(_contractName, _identifier);
        val_ = _vm.load(_contractAddress, bytes32(slot));
        val_ = val_ >> (offset * 8);
    }

    /// @notice Reads the slot of a contract per ABI spec
    function loadAddress(
        AbiSpec _spec,
        Vm _vm,
        address _contractAddress,
        string memory _contractName,
        string memory _identifier
    )
        internal
        view
        returns (address val_)
    {
        // TODO(inphi): Assert width of slot. require(specSlot.width == 20)
        bytes32 slot = loadSlot(_spec, _vm, _contractAddress, _contractName, _identifier);
        val_ = address(uint160(uint256(slot)));
    }

    /// @notice Reads the slot of a contract per ABI spec
    function loadUint256(
        AbiSpec _spec,
        Vm _vm,
        address _contractAddress,
        string memory _contractName,
        string memory _identifier
    )
        internal
        view
        returns (uint256 val_)
    {
        bytes32 slot = loadSlot(_spec, _vm, _contractAddress, _contractName, _identifier);
        val_ = uint256(slot);
    }

    /// @notice Reads the slot of a contract per ABI spec
    function loadUint8(
        AbiSpec _spec,
        Vm _vm,
        address _contractAddress,
        string memory _contractName,
        string memory _identifier
    )
        internal
        view
        returns (uint8 val_)
    {
        // TODO(inphi): Assert width of slot. require(specSlot.width == 1)
        bytes32 slot = loadSlot(_spec, _vm, _contractAddress, _contractName, _identifier);
        val_ = uint8(uint256(slot));
    }

    /// @notice Retrieves the return value of method per ABI spec
    function methodCall(
        AbiSpec _spec,
        address _contractAddress,
        string memory _contractName,
        string memory _identifier
    )
        internal
        view
        returns (bytes memory out_)
    {
        bytes4 methodId = _spec.method(_contractName, _identifier);
        bool ok;
        (ok, out_) = _contractAddress.staticcall(abi.encodeWithSelector(methodId));
        require(ok, "ChainAssertions: spec method call failed");
    }

    /// @notice Converts the bytes to an address and asserts its width
    function toAddress(bytes memory _b) internal pure returns (address out_) {
        require(_b.length >= 20, "ChainAssertions: invalid address length");
        out_ = address(uint160(uint256(bytes32(_b))));
    }

    /// @notice Converts the bytes to a uint256 and asserts its width
    function toUint256(bytes memory _b) internal pure returns (uint256 out_) {
        require(_b.length == 32, "ChainAssertions: invalid uint256 length");
        out_ = uint256(bytes32(_b));
    }
}
