// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Testing
import { Vm } from "forge-std/Vm.sol";
import { console2 as console } from "forge-std/console2.sol";

// Scripts
import { DeployConfig } from "scripts/deploy/DeployConfig.s.sol";
import { DeployOPChainInput } from "scripts/deploy/DeployOPChain.s.sol";
import { DeployImplementations } from "scripts/deploy/DeployImplementations.s.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Types } from "scripts/libraries/Types.sol";
import { Blueprint } from "src/libraries/Blueprint.sol";
import { GameTypes } from "src/dispute/lib/Types.sol";

// Interfaces
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { IL1ERC721Bridge } from "interfaces/L1/IL1ERC721Bridge.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { ProtocolVersion, IProtocolVersions } from "interfaces/L1/IProtocolVersions.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IMIPS } from "interfaces/cannon/IMIPS.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IProxyAdminOwnedBase } from "interfaces/L1/IProxyAdminOwnedBase.sol";

library ChainAssertions {
    Vm internal constant vm = Vm(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    /// @notice Checks that a call to the proxyAdmin function on a contract that follows the ProxyAdminOwnedBase
    /// interface fails.
    /// @dev This is used to check that the proxyAdmin is not set on the contract. E.g Implementation contracts.
    /// @param _contract The address of the contract that follows the ProxyAdminOwnedBase interface.
    /// @param _errorSelector The error selector to check for.
    /// @return true if the call fails with the error selector, false otherwise.
    function checkProxyAdminCallFails(address _contract, bytes4 _errorSelector) internal view returns (bool) {
        (bool success, bytes memory data) =
            address(_contract).staticcall(abi.encodeCall(IProxyAdminOwnedBase.proxyAdmin, ()));
        return (!success && data.length == 4 && bytes4(data) == _errorSelector);
    }

    /// @notice Asserts that the SystemConfig is setup correctly
    function checkSystemConfig(
        Types.ContractSet memory _contracts,
        DeployOPChainInput _doi,
        bool _isProxy
    )
        internal
        view
    {
        ISystemConfig config = ISystemConfig(_contracts.SystemConfig);
        console.log(
            "Running chain assertions on the SystemConfig %s at %s",
            _isProxy ? "proxy" : "implementation",
            address(config)
        );

        // Check that the contract is initialized
        DeployUtils.assertInitialized({ _contractAddress: address(config), _isProxy: _isProxy, _slot: 0, _offset: 0 });

        IResourceMetering.ResourceConfig memory resourceConfig = config.resourceConfig();

        if (_isProxy) {
            require(config.owner() == _doi.systemConfigOwner(), "CHECK-SCFG-10");
            require(config.basefeeScalar() == _doi.basefeeScalar(), "CHECK-SCFG-20");
            require(config.blobbasefeeScalar() == _doi.blobBaseFeeScalar(), "CHECK-SCFG-30");
            require(config.batcherHash() == bytes32(uint256(uint160(_doi.batcher()))), "CHECK-SCFG-40");
            require(config.gasLimit() == uint64(_doi.gasLimit()), "CHECK-SCFG-50");
            require(config.unsafeBlockSigner() == _doi.unsafeBlockSigner(), "CHECK-SCFG-60");
            require(config.scalar() >> 248 == 1, "CHECK-SCFG-70");
            // Depends on start block being set to 0 in `initialize`
            require(config.startBlock() == block.number, "CHECK-SCFG-140");
            require(config.batchInbox() == _doi.opcm().chainIdToBatchInboxAddress(_doi.l2ChainId()), "CHECK-SCFG-150");
            // Check _addresses
            require(config.l1CrossDomainMessenger() == _contracts.L1CrossDomainMessenger, "CHECK-SCFG-160");
            require(config.l1ERC721Bridge() == _contracts.L1ERC721Bridge, "CHECK-SCFG-170");
            require(config.l1StandardBridge() == _contracts.L1StandardBridge, "CHECK-SCFG-180");
            require(config.optimismPortal() == _contracts.OptimismPortal, "CHECK-SCFG-200");
            require(config.optimismMintableERC20Factory() == _contracts.OptimismMintableERC20Factory, "CHECK-SCFG-210");
        } else {
            require(config.owner() == address(0), "CHECK-SCFG-220");
            require(config.overhead() == 0, "CHECK-SCFG-230");
            require(config.scalar() == 0, "CHECK-SCFG-240"); // version 1
            require(config.basefeeScalar() == 0, "CHECK-SCFG-250");
            require(config.blobbasefeeScalar() == 0, "CHECK-SCFG-260");
            require(config.batcherHash() == bytes32(0), "CHECK-SCFG-270");
            require(config.gasLimit() == 0, "CHECK-SCFG-280");
            require(config.unsafeBlockSigner() == address(0), "CHECK-SCFG-290");
            // Check _config
            require(resourceConfig.maxResourceLimit == 0, "CHECK-SCFG-300");
            require(resourceConfig.elasticityMultiplier == 0, "CHECK-SCFG-310");
            require(resourceConfig.baseFeeMaxChangeDenominator == 0, "CHECK-SCFG-320");
            require(resourceConfig.systemTxMaxGas == 0, "CHECK-SCFG-330");
            require(resourceConfig.minimumBaseFee == 0, "CHECK-SCFG-340");
            require(resourceConfig.maximumBaseFee == 0, "CHECK-SCFG-350");
            // Check _addresses
            require(config.startBlock() == type(uint256).max, "CHECK-SCFG-360");
            require(config.batchInbox() == address(0), "CHECK-SCFG-370");
            require(config.l1CrossDomainMessenger() == address(0), "CHECK-SCFG-380");
            require(config.l1ERC721Bridge() == address(0), "CHECK-SCFG-390");
            require(config.l1StandardBridge() == address(0), "CHECK-SCFG-400");
            require(config.optimismPortal() == address(0), "CHECK-SCFG-420");
            require(config.optimismMintableERC20Factory() == address(0), "CHECK-SCFG-430");
        }
    }

    /// @notice Asserts that the L1CrossDomainMessenger is setup correctly
    function checkL1CrossDomainMessenger(IL1CrossDomainMessenger _messenger, Vm _vm, bool _isProxy) internal view {
        console.log(
            "Running chain assertions on the L1CrossDomainMessenger %s at %s",
            _isProxy ? "proxy" : "implementation",
            address(_messenger)
        );
        require(address(_messenger) != address(0), "CHECK-L1XDM-10");

        // Check that the contract is initialized
        DeployUtils.assertInitialized({
            _contractAddress: address(_messenger),
            _isProxy: _isProxy,
            _slot: 0,
            _offset: 20
        });

        if (_isProxy) {
            bytes32 xdmSenderSlot = _vm.load(address(_messenger), bytes32(uint256(204)));
            require(address(uint160(uint256(xdmSenderSlot))) == Constants.DEFAULT_L2_SENDER, "CHECK-L1XDM-70");
        } else {
            require(address(_messenger.OTHER_MESSENGER()) == address(0), "CHECK-L1XDM-80");
            require(address(_messenger.otherMessenger()) == address(0), "CHECK-L1XDM-90");
            require(address(_messenger.PORTAL()) == address(0), "CHECK-L1XDM-100");
            require(address(_messenger.portal()) == address(0), "CHECK-L1XDM-110");
            require(address(_messenger.systemConfig()) == address(0), "CHECK-L1XDM-120");
            require(
                checkProxyAdminCallFails(
                    address(_messenger), IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotResolvedDelegateProxy.selector
                ),
                "CHECK-L1XDM-130"
            );
        }
    }

    /// @notice Asserts that the L1StandardBridge is setup correctly
    function checkL1StandardBridgeImpl(IL1StandardBridge _bridge) internal view {
        console.log("Running chain assertions on the L1StandardBridge implementation at %s", address(_bridge));
        require(address(_bridge) != address(0), "CHECK-L1SB-10");

        // Check that the contract is initialized
        DeployUtils.assertInitialized({ _contractAddress: address(_bridge), _isProxy: false, _slot: 0, _offset: 0 });

        require(address(_bridge.MESSENGER()) == address(0), "CHECK-L1SB-70");
        require(address(_bridge.messenger()) == address(0), "CHECK-L1SB-80");
        require(address(_bridge.OTHER_BRIDGE()) == address(0), "CHECK-L1SB-90");
        require(address(_bridge.otherBridge()) == address(0), "CHECK-L1SB-100");
        require(address(_bridge.systemConfig()) == address(0), "CHECK-L1SB-110");
    }

    /// @notice Asserts that the DisputeGameFactory is setup correctly
    function checkDisputeGameFactory(
        IDisputeGameFactory _factory,
        address _expectedOwner,
        address _permissionedDisputeGame,
        bool _isProxy
    )
        internal
        view
    {
        console.log(
            "Running chain assertions on the DisputeGameFactory %s at %s",
            _isProxy ? "proxy" : "implementation",
            address(_factory)
        );
        require(address(_factory) != address(0), "CHECK-DG-10");

        // Check that the contract is initialized
        DeployUtils.assertInitialized({ _contractAddress: address(_factory), _isProxy: _isProxy, _slot: 0, _offset: 0 });

        if (_isProxy) {
            require(
                address(_factory.gameImpls(GameTypes.PERMISSIONED_CANNON)) == _permissionedDisputeGame, "CHECK-DG-20"
            );
        } else {
            require(address(_factory.gameImpls(GameTypes.PERMISSIONED_CANNON)) == address(0), "CHECK-DG-20");
            // The same check is made for both proxy and implementation
            require(_factory.owner() == _expectedOwner, "CHECK-DG-30");
        }
    }

    /// @notice Asserts that the MIPs contract is setup correctly
    function checkMIPS(IMIPS _mips, IPreimageOracle _oracle) internal view {
        console.log("Running chain assertions on the MIPS at %s", address(_mips));
        require(address(_mips) != address(0), "CHECK-MIPS-10");

        require(_mips.oracle() == _oracle, "CHECK-MIPS-20");
    }

    /// @notice Asserts that the DelayedWETH is setup correctly
    function checkDelayedWETHImpl(IDelayedWETH _weth, uint256 _faultGameWithdrawalDelay) internal view {
        console.log("Running chain assertions on the DelayedWETH implementation at %s", address(_weth));
        require(address(_weth) != address(0), "CHECK-DWETH-10");

        // Check that the contract is initialized
        DeployUtils.assertInitialized({ _contractAddress: address(_weth), _isProxy: false, _slot: 0, _offset: 0 });

        require(_weth.delay() == _faultGameWithdrawalDelay, "CHECK-DWETH-50");
    }

    /// @notice Asserts that the OptimismMintableERC20Factory is setup correctly
    function checkOptimismMintableERC20Factory(Types.ContractSet memory _contracts, bool _isProxy) internal view {
        IOptimismMintableERC20Factory factory = IOptimismMintableERC20Factory(_contracts.OptimismMintableERC20Factory);
        console.log(
            "Running chain assertions on the OptimismMintableERC20Factory %s at %s",
            _isProxy ? "proxy" : "implementation",
            address(factory)
        );
        require(address(factory) != address(0), "CHECK-MERC20F-10");

        // Check that the contract is initialized
        DeployUtils.assertInitialized({ _contractAddress: address(factory), _isProxy: _isProxy, _slot: 0, _offset: 0 });

        if (_isProxy) {
            require(factory.BRIDGE() == _contracts.L1StandardBridge, "CHECK-MERC20F-10");
            require(factory.bridge() == _contracts.L1StandardBridge, "CHECK-MERC20F-20");
        } else {
            require(factory.BRIDGE() == address(0), "CHECK-MERC20F-30");
            require(factory.bridge() == address(0), "CHECK-MERC20F-40");
        }
    }

    /// @notice Asserts that the L1ERC721Bridge is setup correctly
    function checkL1ERC721BridgeImpl(IL1ERC721Bridge _bridge) internal view {
        console.log("Running chain assertions on the L1ERC721Bridge implementation at %s", address(_bridge));

        // Check that the contract is initialized
        DeployUtils.assertInitialized({ _contractAddress: address(_bridge), _isProxy: false, _slot: 0, _offset: 0 });

        require(address(_bridge.OTHER_BRIDGE()) == address(0), "CHECK-L1ERC721B-60");
        require(address(_bridge.otherBridge()) == address(0), "CHECK-L1ERC721B-70");
        require(address(_bridge.MESSENGER()) == address(0), "CHECK-L1ERC721B-80");
        require(address(_bridge.messenger()) == address(0), "CHECK-L1ERC721B-90");
        require(address(_bridge.systemConfig()) == address(0), "CHECK-L1ERC721B-100");
        require(
            checkProxyAdminCallFails(
                address(_bridge), IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotResolvedDelegateProxy.selector
            ),
            "CHECK-L1XDM-130"
        );
    }

    /// @notice Asserts the OptimismPortal is setup correctly
    function checkOptimismPortal2(
        Types.ContractSet memory _contracts,
        DeployConfig _cfg,
        bool _isProxy
    )
        internal
        view
    {
        IOptimismPortal portal = IOptimismPortal(payable(_contracts.OptimismPortal));
        console.log(
            "Running chain assertions on the OptimismPortal2 %s at %s",
            _isProxy ? "proxy" : "implementation",
            address(portal)
        );
        require(address(portal) != address(0), "CHECK-OP2-10");

        // Check that the contract is initialized
        DeployUtils.assertInitialized({ _contractAddress: address(portal), _isProxy: _isProxy, _slot: 0, _offset: 0 });

        address guardian = _cfg.superchainConfigGuardian();
        if (guardian.code.length == 0) {
            console.log("Guardian has no code: %s", guardian);
        }

        if (_isProxy) {
            require(address(portal.disputeGameFactory()) == _contracts.DisputeGameFactory, "CHECK-OP2-20");
            require(address(portal.anchorStateRegistry()) == _contracts.AnchorStateRegistry, "CHECK-OP2-25");
            require(address(portal.systemConfig()) == _contracts.SystemConfig, "CHECK-OP2-30");
            require(portal.guardian() == guardian, "CHECK-OP2-40");
            require(address(portal.systemConfig()) == address(_contracts.SystemConfig), "CHECK-OP2-50");
            require(portal.paused() == ISystemConfig(_contracts.SystemConfig).paused(), "CHECK-OP2-60");
            require(portal.l2Sender() == Constants.DEFAULT_L2_SENDER, "CHECK-OP2-70");
            require(address(portal.ethLockbox()) == _contracts.ETHLockbox, "CHECK-OP2-80");
        } else {
            require(address(portal.anchorStateRegistry()) == address(0), "CHECK-OP2-80");
            require(address(portal.systemConfig()) == address(0), "CHECK-OP2-90");
            require(address(portal.systemConfig()) == address(0), "CHECK-OP2-100");
            require(portal.l2Sender() == address(0), "CHECK-OP2-110");
            require(address(portal.ethLockbox()) == address(0), "CHECK-OP2-120");
        }
        // This slot is the custom gas token _balance and this check ensures
        // that it stays unset for forwards compatibility with custom gas token.
        require(vm.load(address(portal), bytes32(uint256(61))) == bytes32(0), "CHECK-OP2-130");
    }

    /// @notice Asserts that the ETHLockbox is setup correctly
    function checkETHLockboxImpl(IETHLockbox _ethLockbox, IOptimismPortal _portal) internal view {
        console.log("Running chain assertions on the ETHLockbox implementation at %s", address(_ethLockbox));

        // Check that the contract is initialized
        DeployUtils.assertInitialized({ _contractAddress: address(_ethLockbox), _isProxy: false, _slot: 0, _offset: 0 });

        require(address(_ethLockbox.systemConfig()) == address(0), "CHECK-ELB-50");
        require(_ethLockbox.authorizedPortals(_portal) == false, "CHECK-ELB-60");
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
        IProtocolVersions versions = IProtocolVersions(_contracts.ProtocolVersions);
        console.log(
            "Running chain assertions on the ProtocolVersions %s at %s",
            _isProxy ? "proxy" : "implementation",
            address(versions)
        );
        require(address(versions) != address(0), "CHECK-PV-10");

        // Check that the contract is initialized
        DeployUtils.assertInitialized({ _contractAddress: address(versions), _isProxy: _isProxy, _slot: 0, _offset: 0 });

        if (_isProxy) {
            require(versions.owner() == _cfg.finalSystemOwner(), "CHECK-PV-20");
            require(ProtocolVersion.unwrap(versions.required()) == _cfg.requiredProtocolVersion(), "CHECK-PV-30");
            require(ProtocolVersion.unwrap(versions.recommended()) == _cfg.recommendedProtocolVersion(), "CHECK-PV-40");
        } else {
            require(versions.owner() == address(0), "CHECK-PV-50");
            require(ProtocolVersion.unwrap(versions.required()) == 0, "CHECK-PV-60");
            require(ProtocolVersion.unwrap(versions.recommended()) == 0, "CHECK-PV-70");
        }
    }

    /// @notice Asserts that the SuperchainConfig is setup correctly
    function checkSuperchainConfig(
        Types.ContractSet memory _contracts,
        DeployConfig _cfg,
        bool _isProxy
    )
        internal
        view
    {
        ISuperchainConfig superchainConfig = ISuperchainConfig(_contracts.SuperchainConfig);
        console.log(
            "Running chain assertions on the SuperchainConfig %s at %s",
            _isProxy ? "proxy" : "implementation",
            address(superchainConfig)
        );
        require(address(superchainConfig) != address(0), "CHECK-SC-10");

        // Check that the contract is initialized
        DeployUtils.assertInitialized({
            _contractAddress: address(superchainConfig),
            _isProxy: _isProxy,
            _slot: 0,
            _offset: 0
        });

        if (_isProxy) {
            require(superchainConfig.guardian() == _cfg.superchainConfigGuardian(), "CHECK-SC-20");
        } else {
            require(superchainConfig.guardian() == address(0), "CHECK-SC-40");
        }
    }

    /// @notice Asserts that the OPContractsManager is setup correctly
    function checkOPContractsManager(
        Types.ContractSet memory _impls,
        Types.ContractSet memory _proxies,
        IOPContractsManager _opcm,
        IMIPS _mips,
        IProxyAdmin _superchainProxyAdmin
    )
        internal
        view
    {
        console.log("Running chain assertions on the OPContractsManager at %s", address(_opcm));
        require(address(_opcm) != address(0), "CHECK-OPCM-10");

        require(bytes(_opcm.version()).length > 0, "CHECK-OPCM-15");
        require(bytes(_opcm.l1ContractsRelease()).length > 0, "CHECK-OPCM-16");
        require(address(_opcm.protocolVersions()) == _proxies.ProtocolVersions, "CHECK-OPCM-17");
        require(address(_opcm.superchainProxyAdmin()) == address(_superchainProxyAdmin), "CHECK-OPCM-18");
        require(address(_opcm.superchainConfig()) == _proxies.SuperchainConfig, "CHECK-OPCM-19");

        // Ensure that the OPCM impls are correctly saved
        IOPContractsManager.Implementations memory impls = _opcm.implementations();
        require(impls.l1ERC721BridgeImpl == _impls.L1ERC721Bridge, "CHECK-OPCM-50");
        require(impls.optimismPortalImpl == _impls.OptimismPortal, "CHECK-OPCM-60");
        require(impls.systemConfigImpl == _impls.SystemConfig, "CHECK-OPCM-70");
        require(impls.optimismMintableERC20FactoryImpl == _impls.OptimismMintableERC20Factory, "CHECK-OPCM-80");
        require(impls.l1CrossDomainMessengerImpl == _impls.L1CrossDomainMessenger, "CHECK-OPCM-90");
        require(impls.l1StandardBridgeImpl == _impls.L1StandardBridge, "CHECK-OPCM-100");
        require(impls.disputeGameFactoryImpl == _impls.DisputeGameFactory, "CHECK-OPCM-110");
        require(impls.delayedWETHImpl == _impls.DelayedWETH, "CHECK-OPCM-120");
        require(impls.mipsImpl == address(_mips), "CHECK-OPCM-130");
        require(impls.superchainConfigImpl == _impls.SuperchainConfig, "CHECK-OPCM-140");
        require(impls.protocolVersionsImpl == _impls.ProtocolVersions, "CHECK-OPCM-150");

        // Verify that initCode is correctly set into the blueprints
        IOPContractsManager.Blueprints memory blueprints = _opcm.blueprints();
        Blueprint.Preamble memory addressManagerPreamble =
            Blueprint.parseBlueprintPreamble(address(blueprints.addressManager).code);
        require(keccak256(addressManagerPreamble.initcode) == keccak256(vm.getCode("AddressManager")), "CHECK-OPCM-160");

        Blueprint.Preamble memory proxyPreamble = Blueprint.parseBlueprintPreamble(address(blueprints.proxy).code);
        require(keccak256(proxyPreamble.initcode) == keccak256(vm.getCode("Proxy")), "CHECK-OPCM-170");

        Blueprint.Preamble memory proxyAdminPreamble =
            Blueprint.parseBlueprintPreamble(address(blueprints.proxyAdmin).code);
        require(keccak256(proxyAdminPreamble.initcode) == keccak256(vm.getCode("ProxyAdmin")), "CHECK-OPCM-180");

        Blueprint.Preamble memory l1ChugSplashProxyPreamble =
            Blueprint.parseBlueprintPreamble(address(blueprints.l1ChugSplashProxy).code);
        require(
            keccak256(l1ChugSplashProxyPreamble.initcode) == keccak256(vm.getCode("L1ChugSplashProxy")),
            "CHECK-OPCM-190"
        );

        Blueprint.Preamble memory rdProxyPreamble =
            Blueprint.parseBlueprintPreamble(address(blueprints.resolvedDelegateProxy).code);
        require(keccak256(rdProxyPreamble.initcode) == keccak256(vm.getCode("ResolvedDelegateProxy")), "CHECK-OPCM-200");

        Blueprint.Preamble memory pdg1Preamble =
            Blueprint.parseBlueprintPreamble(address(blueprints.permissionedDisputeGame1).code);
        Blueprint.Preamble memory pdg2Preamble =
            Blueprint.parseBlueprintPreamble(address(blueprints.permissionedDisputeGame2).code);
        // combine pdg1 and pdg2 initcodes
        bytes memory fullPermissionedDisputeGameInitcode =
            abi.encodePacked(pdg1Preamble.initcode, pdg2Preamble.initcode);
        require(
            keccak256(fullPermissionedDisputeGameInitcode) == keccak256(vm.getCode("PermissionedDisputeGame")),
            "CHECK-OPCM-210"
        );
    }

    /// @notice Converts variables needed from the DeployConfig to a DeployOPChainInput contract
    function dioToContractSet(DeployImplementations.Output memory _output)
        internal
        pure
        returns (Types.ContractSet memory)
    {
        return Types.ContractSet({
            L1CrossDomainMessenger: address(_output.l1CrossDomainMessengerImpl),
            L1StandardBridge: address(_output.l1StandardBridgeImpl),
            L2OutputOracle: address(0),
            DisputeGameFactory: address(_output.disputeGameFactoryImpl),
            DelayedWETH: address(_output.delayedWETHImpl),
            PermissionedDelayedWETH: address(_output.delayedWETHImpl),
            AnchorStateRegistry: address(_output.anchorStateRegistryImpl),
            OptimismMintableERC20Factory: address(_output.optimismMintableERC20FactoryImpl),
            OptimismPortal: address(_output.optimismPortalImpl),
            ETHLockbox: address(_output.ethLockboxImpl),
            SystemConfig: address(_output.systemConfigImpl),
            L1ERC721Bridge: address(_output.l1ERC721BridgeImpl),
            ProtocolVersions: address(_output.protocolVersionsImpl),
            SuperchainConfig: address(_output.superchainConfigImpl)
        });
    }
}
