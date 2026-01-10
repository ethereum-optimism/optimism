// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Constants } from "src/libraries/Constants.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";
import { ChainAssertions } from "scripts/deploy/ChainAssertions.sol";
import { Constants as ScriptConstants } from "scripts/libraries/Constants.sol";
import { Types } from "scripts/libraries/Types.sol";

import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1ERC721Bridge } from "interfaces/L1/IL1ERC721Bridge.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { GameTypes } from "src/dispute/lib/Types.sol";

contract DeployOPChain is Script {
    struct Output {
        IProxyAdmin opChainProxyAdmin;
        IAddressManager addressManager;
        IL1ERC721Bridge l1ERC721BridgeProxy;
        ISystemConfig systemConfigProxy;
        IOptimismMintableERC20Factory optimismMintableERC20FactoryProxy;
        IL1StandardBridge l1StandardBridgeProxy;
        IL1CrossDomainMessenger l1CrossDomainMessengerProxy;
        IOptimismPortal optimismPortalProxy;
        IETHLockbox ethLockboxProxy;
        IDisputeGameFactory disputeGameFactoryProxy;
        IAnchorStateRegistry anchorStateRegistryProxy;
        IFaultDisputeGame faultDisputeGame;
        IPermissionedDisputeGame permissionedDisputeGame;
        IDelayedWETH delayedWETHPermissionedGameProxy;
        IDelayedWETH delayedWETHPermissionlessGameProxy;
    }

    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        Types.DeployOPChainInput memory input = abi.decode(_input, (Types.DeployOPChainInput));
        Output memory output_ = run(input);
        return abi.encode(output_);
    }

    function run(Types.DeployOPChainInput memory _input) public returns (Output memory output_) {
        checkInput(_input);

        // Check if OPCM v2 should be used.
        bool useV2 = isDevFeatureOpcmV2Enabled(_input.opcm);

        if (useV2) {
            IOPContractsManagerV2 opcmV2 = IOPContractsManagerV2(_input.opcm);
            IOPContractsManagerV2.FullConfig memory config = toOPCMV2DeployInput(_input);

            vm.broadcast(msg.sender);
            IOPContractsManagerV2.ChainContracts memory chainContracts = opcmV2.deploy(config);
            output_ = fromOPCMV2OutputToOutput(chainContracts);
        } else {
            IOPContractsManager opcm = IOPContractsManager(_input.opcm);
            IOPContractsManager.DeployInput memory deployInput = toOPCMV1DeployInput(_input);

            vm.broadcast(msg.sender);
            IOPContractsManager.DeployOutput memory deployOutput = opcm.deploy(deployInput);

            output_ = fromOPCMV1OutputToOutput(deployOutput);
        }

        checkOutput(_input, output_);

        vm.label(address(output_.opChainProxyAdmin), "opChainProxyAdmin");
        vm.label(address(output_.addressManager), "addressManager");
        vm.label(address(output_.l1ERC721BridgeProxy), "l1ERC721BridgeProxy");
        vm.label(address(output_.systemConfigProxy), "systemConfigProxy");
        vm.label(address(output_.optimismMintableERC20FactoryProxy), "optimismMintableERC20FactoryProxy");
        vm.label(address(output_.l1StandardBridgeProxy), "l1StandardBridgeProxy");
        vm.label(address(output_.l1CrossDomainMessengerProxy), "l1CrossDomainMessengerProxy");
        vm.label(address(output_.optimismPortalProxy), "optimismPortalProxy");
        vm.label(address(output_.ethLockboxProxy), "ethLockboxProxy");
        vm.label(address(output_.disputeGameFactoryProxy), "disputeGameFactoryProxy");
        vm.label(address(output_.anchorStateRegistryProxy), "anchorStateRegistryProxy");
        vm.label(address(output_.delayedWETHPermissionedGameProxy), "delayedWETHPermissionedGameProxy");
        // TODO: Eventually switch from Permissioned to Permissionless.
        // vm.label(address(output_.faultDisputeGame), "faultDisputeGame");
        // vm.label(address(output_.delayedWETHPermissionlessGameProxy), "delayedWETHPermissionlessGameProxy");
    }

    // -------- Features --------

    /// @notice Checks if OPCM v2 dev feature flag is enabled from the contract's dev feature bitmap.
    function isDevFeatureOpcmV2Enabled(address _opcmAddr) internal view returns (bool) {
        // Both v1 and v2 share the same interface for this function.
        return IOPContractsManager(_opcmAddr).isDevFeatureEnabled(DevFeatures.OPCM_V2);
    }

    function isDevFeatureV2DisputeGamesEnabled(address _opcmAddr) internal view returns (bool) {
        IOPContractsManager opcm = IOPContractsManager(_opcmAddr);
        return DevFeatures.isDevFeatureEnabled(opcm.devFeatureBitmap(), DevFeatures.DEPLOY_V2_DISPUTE_GAMES);
    }

    /// @notice Converts Types.DeployOPChainInput to IOPContractsManager.DeployInput.
    /// @param _input The input parameters.
    /// @return deployInput_ The deployed input parameters.
    function toOPCMV1DeployInput(Types.DeployOPChainInput memory _input)
        internal
        pure
        returns (IOPContractsManager.DeployInput memory deployInput_)
    {
        IOPContractsManager.Roles memory roles = IOPContractsManager.Roles({
            opChainProxyAdminOwner: _input.opChainProxyAdminOwner,
            systemConfigOwner: _input.systemConfigOwner,
            batcher: _input.batcher,
            unsafeBlockSigner: _input.unsafeBlockSigner,
            proposer: _input.proposer,
            challenger: _input.challenger
        });
        deployInput_ = IOPContractsManager.DeployInput({
            roles: roles,
            basefeeScalar: _input.basefeeScalar,
            blobBasefeeScalar: _input.blobBaseFeeScalar,
            l2ChainId: _input.l2ChainId,
            startingAnchorRoot: startingAnchorRoot(),
            saltMixer: _input.saltMixer,
            gasLimit: _input.gasLimit,
            disputeGameType: _input.disputeGameType,
            disputeAbsolutePrestate: _input.disputeAbsolutePrestate,
            disputeMaxGameDepth: _input.disputeMaxGameDepth,
            disputeSplitDepth: _input.disputeSplitDepth,
            disputeClockExtension: _input.disputeClockExtension,
            disputeMaxClockDuration: _input.disputeMaxClockDuration,
            useCustomGasToken: _input.useCustomGasToken
        });
    }

    /// @notice Converts Types.DeployOPChainInput to IOPContractsManagerV2.FullConfig.
    /// @param _input The input parameters.
    /// @return config_ The deployed input parameters.
    function toOPCMV2DeployInput(Types.DeployOPChainInput memory _input)
        internal
        pure
        returns (IOPContractsManagerV2.FullConfig memory config_)
    {
        // Build dispute game configs - OPCMV2 requires exactly 3 configs: CANNON, PERMISSIONED_CANNON, CANNON_KONA
        IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](3);

        // Determine which games should be enabled based on the starting respected game type
        bool cannonEnabled = _input.disputeGameType.raw() == GameTypes.CANNON.raw();
        bool permissionedCannonEnabled = true; // PERMISSIONED_CANNON must always be enabled
        bool cannonKonaEnabled = _input.disputeGameType.raw() == GameTypes.CANNON_KONA.raw();

        // Config 0: CANNON
        IOPContractsManagerUtils.FaultDisputeGameConfig memory cannonConfig =
            IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: _input.disputeAbsolutePrestate });

        disputeGameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: cannonEnabled,
            initBond: cannonEnabled ? 0.08 ether : 0, // Standard init bond if enabled
            gameType: GameTypes.CANNON,
            gameArgs: abi.encode(cannonConfig)
        });

        // Config 1: PERMISSIONED_CANNON (must be enabled)
        IOPContractsManagerUtils.PermissionedDisputeGameConfig memory pdgConfig = IOPContractsManagerUtils
            .PermissionedDisputeGameConfig({
            absolutePrestate: _input.disputeAbsolutePrestate,
            proposer: _input.proposer,
            challenger: _input.challenger
        });

        disputeGameConfigs[1] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: permissionedCannonEnabled,
            initBond: 0.08 ether, // Standard init bond
            gameType: GameTypes.PERMISSIONED_CANNON,
            gameArgs: abi.encode(pdgConfig)
        });

        // Config 2: CANNON_KONA
        IOPContractsManagerUtils.FaultDisputeGameConfig memory cannonKonaConfig =
            IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: _input.disputeAbsolutePrestate });

        disputeGameConfigs[2] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: cannonKonaEnabled,
            initBond: cannonKonaEnabled ? 0.08 ether : 0, // Standard init bond if enabled
            gameType: GameTypes.CANNON_KONA,
            gameArgs: abi.encode(cannonKonaConfig)
        });

        config_ = IOPContractsManagerV2.FullConfig({
            saltMixer: _input.saltMixer,
            superchainConfig: _input.superchainConfig,
            proxyAdminOwner: _input.opChainProxyAdminOwner,
            systemConfigOwner: _input.systemConfigOwner,
            unsafeBlockSigner: _input.unsafeBlockSigner,
            batcher: _input.batcher,
            startingAnchorRoot: ScriptConstants.DEFAULT_OUTPUT_ROOT(),
            startingRespectedGameType: _input.disputeGameType,
            basefeeScalar: _input.basefeeScalar,
            blobBasefeeScalar: _input.blobBaseFeeScalar,
            gasLimit: _input.gasLimit,
            l2ChainId: _input.l2ChainId,
            resourceConfig: Constants.DEFAULT_RESOURCE_CONFIG(),
            disputeGameConfigs: disputeGameConfigs,
            useCustomGasToken: _input.useCustomGasToken
        });
    }

    /// @notice Converts IOPContractsManagerV2.ChainContracts to Output.
    /// @param _chainContracts The chain contracts.
    /// @return output_ The output parameters.
    function fromOPCMV2OutputToOutput(IOPContractsManagerV2.ChainContracts memory _chainContracts)
        internal
        pure
        returns (Output memory output_)
    {
        output_ = Output({
            opChainProxyAdmin: _chainContracts.proxyAdmin,
            addressManager: _chainContracts.addressManager,
            l1ERC721BridgeProxy: _chainContracts.l1ERC721Bridge,
            systemConfigProxy: _chainContracts.systemConfig,
            optimismMintableERC20FactoryProxy: _chainContracts.optimismMintableERC20Factory,
            l1StandardBridgeProxy: _chainContracts.l1StandardBridge,
            l1CrossDomainMessengerProxy: _chainContracts.l1CrossDomainMessenger,
            optimismPortalProxy: _chainContracts.optimismPortal,
            ethLockboxProxy: _chainContracts.ethLockbox,
            disputeGameFactoryProxy: _chainContracts.disputeGameFactory,
            anchorStateRegistryProxy: _chainContracts.anchorStateRegistry,
            faultDisputeGame: IFaultDisputeGame(address(0)),
            permissionedDisputeGame: IPermissionedDisputeGame(address(0)),
            delayedWETHPermissionedGameProxy: _chainContracts.delayedWETH,
            delayedWETHPermissionlessGameProxy: IDelayedWETH(payable(address(0)))
        });
    }

    /// @notice Converts IOPContractsManager.DeployOutput to Output.
    /// @param _deployOutput The deploy output.
    /// @return output_ The output parameters.
    function fromOPCMV1OutputToOutput(IOPContractsManager.DeployOutput memory _deployOutput)
        internal
        pure
        returns (Output memory output_)
    {
        output_ = Output({
            opChainProxyAdmin: _deployOutput.opChainProxyAdmin,
            addressManager: _deployOutput.addressManager,
            l1ERC721BridgeProxy: _deployOutput.l1ERC721BridgeProxy,
            systemConfigProxy: _deployOutput.systemConfigProxy,
            optimismMintableERC20FactoryProxy: _deployOutput.optimismMintableERC20FactoryProxy,
            l1StandardBridgeProxy: _deployOutput.l1StandardBridgeProxy,
            l1CrossDomainMessengerProxy: _deployOutput.l1CrossDomainMessengerProxy,
            optimismPortalProxy: _deployOutput.optimismPortalProxy,
            ethLockboxProxy: _deployOutput.ethLockboxProxy,
            disputeGameFactoryProxy: _deployOutput.disputeGameFactoryProxy,
            anchorStateRegistryProxy: _deployOutput.anchorStateRegistryProxy,
            faultDisputeGame: _deployOutput.faultDisputeGame,
            permissionedDisputeGame: _deployOutput.permissionedDisputeGame,
            delayedWETHPermissionedGameProxy: _deployOutput.delayedWETHPermissionedGameProxy,
            delayedWETHPermissionlessGameProxy: _deployOutput.delayedWETHPermissionlessGameProxy
        });
    }

    // -------- Validations --------

    function checkInput(Types.DeployOPChainInput memory _i) public view {
        require(_i.opChainProxyAdminOwner != address(0), "DeployOPChainInput: opChainProxyAdminOwner not set");
        require(_i.systemConfigOwner != address(0), "DeployOPChainInput: systemConfigOwner not set");
        require(_i.batcher != address(0), "DeployOPChainInput: batcher not set");
        require(_i.unsafeBlockSigner != address(0), "DeployOPChainInput: unsafeBlockSigner not set");
        require(_i.proposer != address(0), "DeployOPChainInput: proposer not set");
        require(_i.challenger != address(0), "DeployOPChainInput: challenger not set");

        require(_i.blobBaseFeeScalar != 0, "DeployOPChainInput: blobBaseFeeScalar not set");
        require(_i.basefeeScalar != 0, "DeployOPChainInput: basefeeScalar not set");
        require(_i.gasLimit != 0, "DeployOPChainInput: gasLimit not set");

        require(_i.l2ChainId != 0, "DeployOPChainInput: l2ChainId not set");
        require(_i.l2ChainId != block.chainid, "DeployOPChainInput: l2ChainId matches block.chainid");

        require(_i.opcm != address(0), "DeployOPChainInput: opcm not set");
        DeployUtils.assertValidContractAddress(_i.opcm);

        require(_i.disputeMaxGameDepth != 0, "DeployOPChainInput: disputeMaxGameDepth not set");
        require(_i.disputeSplitDepth != 0, "DeployOPChainInput: disputeSplitDepth not set");
        require(_i.disputeMaxClockDuration.raw() != 0, "DeployOPChainInput: disputeMaxClockDuration not set");
        require(_i.disputeAbsolutePrestate.raw() != bytes32(0), "DeployOPChainInput: disputeAbsolutePrestate not set");
    }

    function checkOutput(Types.DeployOPChainInput memory _i, Output memory _o) public {
        // With 16 addresses, we'd get a stack too deep error if we tried to do this inline as a
        // single call to `Solarray.addresses`. So we split it into two calls.
        address[] memory addrs1 = Solarray.addresses(
            address(_o.opChainProxyAdmin),
            address(_o.addressManager),
            address(_o.l1ERC721BridgeProxy),
            address(_o.systemConfigProxy),
            address(_o.optimismMintableERC20FactoryProxy),
            address(_o.l1StandardBridgeProxy),
            address(_o.l1CrossDomainMessengerProxy)
        );
        address[] memory addrs2 = Solarray.addresses(
            address(_o.optimismPortalProxy),
            address(_o.disputeGameFactoryProxy),
            address(_o.anchorStateRegistryProxy),
            address(_o.delayedWETHPermissionedGameProxy),
            address(_o.ethLockboxProxy)
        );

        DeployUtils.assertValidContractAddresses(Solarray.extend(addrs1, addrs2));
        _assertValidDeploy(_i, _o);
    }

    function _assertValidDeploy(Types.DeployOPChainInput memory _i, Output memory _o) internal {
        Types.ContractSet memory proxies = Types.ContractSet({
            L1CrossDomainMessenger: address(_o.l1CrossDomainMessengerProxy),
            L1StandardBridge: address(_o.l1StandardBridgeProxy),
            L2OutputOracle: address(0),
            DisputeGameFactory: address(_o.disputeGameFactoryProxy),
            DelayedWETH: address(_o.delayedWETHPermissionlessGameProxy),
            PermissionedDelayedWETH: address(_o.delayedWETHPermissionedGameProxy),
            AnchorStateRegistry: address(_o.anchorStateRegistryProxy),
            OptimismMintableERC20Factory: address(_o.optimismMintableERC20FactoryProxy),
            OptimismPortal: address(_o.optimismPortalProxy),
            ETHLockbox: address(_o.ethLockboxProxy),
            SystemConfig: address(_o.systemConfigProxy),
            L1ERC721Bridge: address(_o.l1ERC721BridgeProxy),
            ProtocolVersions: address(0),
            SuperchainConfig: address(_i.superchainConfig)
        });

        // Check dispute games and get superchain config
        address expectedPDGImpl = address(_o.permissionedDisputeGame);

        if (isDevFeatureOpcmV2Enabled(_i.opcm)) {
            // OPCM v2: use implementations from v2 contract
            IOPContractsManagerV2 opcmV2 = IOPContractsManagerV2(_i.opcm);
            expectedPDGImpl = opcmV2.implementations().permissionedDisputeGameV2Impl;
        } else {
            // OPCM v1: use implementations from v1 contract
            IOPContractsManager opcm = IOPContractsManager(_i.opcm);
            // With v2 game contracts enabled, we use the predeployed pdg implementation
            expectedPDGImpl = opcm.implementations().permissionedDisputeGameV2Impl;
        }

        ChainAssertions.checkDisputeGameFactory(
            _o.disputeGameFactoryProxy, _i.opChainProxyAdminOwner, expectedPDGImpl, true
        );

        ChainAssertions.checkAnchorStateRegistryProxy(_o.anchorStateRegistryProxy, true);
        ChainAssertions.checkL1CrossDomainMessenger(_o.l1CrossDomainMessengerProxy, vm, true);
        ChainAssertions.checkOptimismPortal2({
            _contracts: proxies,
            _superchainConfig: _i.superchainConfig,
            _opChainProxyAdminOwner: _i.opChainProxyAdminOwner,
            _isProxy: true
        });
        ChainAssertions.checkSystemConfigProxies(proxies, _i);

        DeployUtils.assertValidContractAddress(address(_o.l1CrossDomainMessengerProxy));
        DeployUtils.assertResolvedDelegateProxyImplementationSet("OVM_L1CrossDomainMessenger", _o.addressManager);

        // Proxies initialized checks
        DeployUtils.assertInitialized({
            _contractAddress: address(_o.l1ERC721BridgeProxy),
            _isProxy: true,
            _slot: 0,
            _offset: 0
        });
        DeployUtils.assertInitialized({
            _contractAddress: address(_o.l1StandardBridgeProxy),
            _isProxy: true,
            _slot: 0,
            _offset: 0
        });
        DeployUtils.assertInitialized({
            _contractAddress: address(_o.optimismMintableERC20FactoryProxy),
            _isProxy: true,
            _slot: 0,
            _offset: 0
        });
        DeployUtils.assertInitialized({
            _contractAddress: address(_o.ethLockboxProxy),
            _isProxy: true,
            _slot: 0,
            _offset: 0
        });

        require(_o.addressManager.owner() == address(_o.opChainProxyAdmin), "AM-10");
        assertValidOPChainProxyAdmin(_i, _o);
    }

    function assertValidOPChainProxyAdmin(Types.DeployOPChainInput memory _doi, Output memory _doo) internal {
        IProxyAdmin admin = _doo.opChainProxyAdmin;
        require(admin.owner() == _doi.opChainProxyAdminOwner, "OPCPA-10");
        require(
            admin.getProxyImplementation(address(_doo.l1CrossDomainMessengerProxy))
                == DeployUtils.assertResolvedDelegateProxyImplementationSet(
                    "OVM_L1CrossDomainMessenger", _doo.addressManager
                ),
            "OPCPA-20"
        );
        require(address(admin.addressManager()) == address(_doo.addressManager), "OPCPA-30");
        require(
            admin.getProxyImplementation(address(_doo.l1StandardBridgeProxy))
                == DeployUtils.assertL1ChugSplashImplementationSet(address(_doo.l1StandardBridgeProxy)),
            "OPCPA-40"
        );
        require(
            admin.getProxyImplementation(address(_doo.l1ERC721BridgeProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.l1ERC721BridgeProxy)),
            "OPCPA-50"
        );
        require(
            admin.getProxyImplementation(address(_doo.optimismPortalProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.optimismPortalProxy)),
            "OPCPA-60"
        );
        require(
            admin.getProxyImplementation(address(_doo.systemConfigProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.systemConfigProxy)),
            "OPCPA-70"
        );
        require(
            admin.getProxyImplementation(address(_doo.optimismMintableERC20FactoryProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.optimismMintableERC20FactoryProxy)),
            "OPCPA-80"
        );
        require(
            admin.getProxyImplementation(address(_doo.disputeGameFactoryProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.disputeGameFactoryProxy)),
            "OPCPA-90"
        );
        require(
            admin.getProxyImplementation(address(_doo.delayedWETHPermissionedGameProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.delayedWETHPermissionedGameProxy)),
            "OPCPA-100"
        );
        require(
            admin.getProxyImplementation(address(_doo.anchorStateRegistryProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.anchorStateRegistryProxy)),
            "OPCPA-110"
        );
        require(
            admin.getProxyImplementation(address(_doo.ethLockboxProxy))
                == DeployUtils.assertERC1967ImplementationSet(address(_doo.ethLockboxProxy)),
            "OPCPA-120"
        );
    }

    function startingAnchorRoot() public pure returns (bytes memory) {
        // WARNING: For now always hardcode the starting permissioned game anchor root to 0xdead,
        // and we do not set anything for the permissioned game. This is because we currently only
        // support deploying straight to permissioned games, and the starting root does not
        // matter for that, as long as it is non-zero, since no games will be played. We do not
        // deploy the permissionless game (and therefore do not set a starting root for it here)
        // because to to update to the permissionless game, we will need to update its starting
        // anchor root and deploy a new permissioned dispute game contract anyway.
        //
        // You can `console.logBytes(abi.encode(ScriptConstants.DEFAULT_OUTPUT_ROOT()))` to get the bytes that
        // are hardcoded into `op-chain-ops/deployer/opcm/opchain.go`

        return abi.encode(ScriptConstants.DEFAULT_OUTPUT_ROOT());
    }
}
