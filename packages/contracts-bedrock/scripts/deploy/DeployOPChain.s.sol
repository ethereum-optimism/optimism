// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { Constants } from "src/libraries/Constants.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";
import { ChainAssertions } from "scripts/deploy/ChainAssertions.sol";
import { Constants as ScriptConstants } from "scripts/libraries/Constants.sol";
import { Types } from "scripts/libraries/Types.sol";

import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerContainer.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
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
import { GameType, GameTypes } from "src/dispute/lib/Types.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

contract DeployOPChain is Script {
    /// @notice The default init bond for the dispute games.
    uint256 public constant DEFAULT_INIT_BOND = 0.08 ether;

    /// @notice Whether the OPCM has SUPER_ROOT_GAMES_MIGRATION enabled.
    bool public isSuperRoot;

    /// @notice The output of the DeployOPChain script. This is the same as the DeployOPChainOutput type in the
    /// op-deployer package.
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

    /// @notice Runs the DeployOPChain script with the given input.
    /// @param _input The input to the script.
    /// @return output_ The output of the script.
    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        require(_input.length > 0, "DeployOPChain: input cannot be empty");
        Types.DeployOPChainInput memory input = abi.decode(_input, (Types.DeployOPChainInput));
        Output memory output_ = run(input);
        return abi.encode(output_);
    }

    /// @notice Runs the DeployOPChain script with the given input.
    /// @param _input The input to the script.
    /// @return output_ The output of the script.
    function run(Types.DeployOPChainInput memory _input) public returns (Output memory output_) {
        checkInput(_input);

        require(address(_input.opcm).code.length > 0, "DeployOPChain: OPCM address has no code");

        IOPContractsManagerV2 opcmV2 = IOPContractsManagerV2(_input.opcm);
        isSuperRoot = _isSuperRootEnabled(opcmV2);
        IOPContractsManagerV2.FullConfig memory config = _toOPCMV2DeployInput(_input);

        vm.broadcast(msg.sender);
        IOPContractsManagerV2.ChainContracts memory chainContracts = opcmV2.deploy(config);
        output_ = _fromOPCMV2OutputToOutput(chainContracts);

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
        // TODO: OPCMV2 uses one shared DelayedWETH for all game types; both Output WETH fields
        // alias it, so only one label is set above. Revisit the WETH label and field naming in
        // a follow-up PR.
        if (address(output_.faultDisputeGame) != address(0)) {
            vm.label(address(output_.faultDisputeGame), "faultDisputeGame");
        }
    }

    // -------- Features --------

    /// @notice Converts Types.DeployOPChainInput to IOPContractsManagerV2.FullConfig.
    /// @param _input The input parameters.
    /// @return config_ The deployed input parameters.
    function _toOPCMV2DeployInput(Types.DeployOPChainInput memory _input)
        internal
        view
        returns (IOPContractsManagerV2.FullConfig memory config_)
    {
        (bool permissionless, GameType respectedGameType) =
            _initialDeployGameSelection(_input.disputeGameType, isSuperRoot);

        bool enableCannonKona = _input.disputeGameType.raw() == GameTypes.CANNON_KONA.raw();
        bool enableSuperCannonKona = _input.disputeGameType.raw() == GameTypes.SUPER_CANNON_KONA.raw();
        // Register the CANNON_KONA guardian fallback. OPCMV2 does not require it.
        bool enablePermissionedCannon = enableCannonKona || (!isSuperRoot && !permissionless);
        bool enableSuperPermissioned = enableSuperCannonKona || (isSuperRoot && !permissionless);
        // Build dispute game configs - OPCMV2 requires all 6 game type configs.
        // Order must match validGameTypes in OPContractsManagerV2._assertValidFullConfig().
        IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](6);

        // Config 0: legacy CANNON slot, disabled after U19 and kept to satisfy OPCMV2's 6-config shape.
        disputeGameConfigs[0] = _createGameConfig(false, 0, GameTypes.CANNON, bytes(""));

        // Config 1: PERMISSIONED_CANNON
        disputeGameConfigs[1] = _createGameConfig(
            enablePermissionedCannon,
            DEFAULT_INIT_BOND,
            GameTypes.PERMISSIONED_CANNON,
            abi.encode(
                IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                    absolutePrestate: enableCannonKona ? _input.cannonAbsolutePrestate : _input.disputeAbsolutePrestate,
                    proposer: _input.proposer,
                    challenger: _input.challenger
                })
            )
        );

        // Config 2: CANNON_KONA
        disputeGameConfigs[2] = _createGameConfig(
            enableCannonKona,
            DEFAULT_INIT_BOND,
            GameTypes.CANNON_KONA,
            abi.encode(
                IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: _input.disputeAbsolutePrestate })
            )
        );

        // Config 3: SUPER_PERMISSIONED
        disputeGameConfigs[3] = _createGameConfig(
            enableSuperPermissioned,
            0,
            GameTypes.SUPER_PERMISSIONED,
            abi.encode(IOPContractsManagerUtils.SuperPermissionedDisputeGameConfig({ proposer: _input.proposer }))
        );

        // Config 4: SUPER_CANNON_KONA
        disputeGameConfigs[4] = _createGameConfig(
            enableSuperCannonKona,
            DEFAULT_INIT_BOND,
            GameTypes.SUPER_CANNON_KONA,
            abi.encode(
                IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: _input.disputeAbsolutePrestate })
            )
        );

        // Config 5: ZK_DISPUTE_GAME (disabled for initial deployment)
        disputeGameConfigs[5] = _createGameConfig(false, 0, GameTypes.ZK_DISPUTE_GAME, bytes(""));

        config_ = IOPContractsManagerV2.FullConfig({
            saltMixer: _input.saltMixer,
            superchainConfig: _input.superchainConfig,
            proxyAdminOwner: _input.opChainProxyAdminOwner,
            systemConfigOwner: _input.systemConfigOwner,
            unsafeBlockSigner: _input.unsafeBlockSigner,
            batcher: _input.batcher,
            startingAnchorRoot: _input.startingAnchorRoot,
            startingRespectedGameType: respectedGameType,
            basefeeScalar: _input.basefeeScalar,
            blobBasefeeScalar: _input.blobBaseFeeScalar,
            gasLimit: _input.gasLimit,
            l2ChainId: _input.l2ChainId,
            resourceConfig: _resourceConfigForGasLimit(_input.gasLimit),
            disputeGameConfigs: disputeGameConfigs,
            useCustomGasToken: _input.useCustomGasToken
        });
    }

    /// @notice Converts IOPContractsManagerV2.ChainContracts to Output.
    /// @param _chainContracts The chain contracts.
    /// @return output_ The output parameters.
    function _fromOPCMV2OutputToOutput(IOPContractsManagerV2.ChainContracts memory _chainContracts)
        internal
        view
        returns (Output memory output_)
    {
        GameType permGameType = isSuperRoot ? GameTypes.SUPER_PERMISSIONED : GameTypes.PERMISSIONED_CANNON;
        GameType faultGameType = isSuperRoot ? GameTypes.SUPER_CANNON_KONA : GameTypes.CANNON_KONA;
        address permissionedDgImpl = address(_chainContracts.disputeGameFactory.gameImpls(permGameType));
        address faultDgImpl = address(_chainContracts.disputeGameFactory.gameImpls(faultGameType));

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
            faultDisputeGame: IFaultDisputeGame(faultDgImpl),
            permissionedDisputeGame: IPermissionedDisputeGame(permissionedDgImpl),
            delayedWETHPermissionedGameProxy: _chainContracts.delayedWETH,
            delayedWETHPermissionlessGameProxy: IDelayedWETH(payable(_chainContracts.delayedWETH))
        });
    }

    /// @notice Derives a ResourceConfig sized to fit the requested L2 gas limit.
    ///
    ///         For gasLimit >= the default's reserved gas (maxResourceLimit + systemTxMaxGas),
    ///         returns DEFAULT_RESOURCE_CONFIG unchanged. Every existing production chain takes
    ///         this branch.
    ///
    ///         For smaller gasLimits, shrinks maxResourceLimit to fit while preserving
    ///         systemTxMaxGas and the EIP-1559 parameters, so the L1 attributes deposit
    ///         reservation stays constant and deposit throughput scales with chain size.
    ///         maxResourceLimit is rounded down to a multiple of elasticityMultiplier to
    ///         satisfy SystemConfig._setResourceConfig's precision-loss check.
    /// @param _gasLimit The requested L2 gas limit.
    /// @return cfg_ A ResourceConfig that satisfies maxResourceLimit + systemTxMaxGas <= _gasLimit.
    function _resourceConfigForGasLimit(uint64 _gasLimit)
        internal
        pure
        returns (IResourceMetering.ResourceConfig memory cfg_)
    {
        cfg_ = Constants.DEFAULT_RESOURCE_CONFIG();
        uint64 reserved = uint64(cfg_.maxResourceLimit) + uint64(cfg_.systemTxMaxGas);
        if (_gasLimit >= reserved) {
            return cfg_;
        }

        require(_gasLimit > uint64(cfg_.systemTxMaxGas), "DeployOPChain: gasLimit must exceed systemTxMaxGas");
        // Branch is only reached for _gasLimit < reserved (~21M for the current default), well
        // within uint32, so the downcast on assignment cannot truncate.
        uint64 available = _gasLimit - uint64(cfg_.systemTxMaxGas);
        uint64 mult = uint64(cfg_.elasticityMultiplier);

        // Satisfy the SystemConfig._setResourceConfig requirement that the maxResourceLimit is
        // a multiple of the elasticityMultiplier.
        cfg_.maxResourceLimit = uint32((available / mult) * mult);
        require(cfg_.maxResourceLimit > 0, "DeployOPChain: gasLimit too small for any deposit budget");
    }

    /// @notice Returns the permissionless mode and respected game type for an initial deployment.
    /// @dev CANNON_KONA and SUPER_CANNON_KONA prestates are not interchangeable, so the type must be explicit.
    ///      PERMISSIONED_CANNON selects the permissioned game for the OPCM mode; SUPER_PERMISSIONED has no prestate.
    function _initialDeployGameSelection(
        GameType _disputeGameType,
        bool _isSuperRoot
    )
        internal
        pure
        returns (bool permissionless_, GameType respectedGameType_)
    {
        permissionless_ = _disputeGameType.raw() == GameTypes.CANNON_KONA.raw()
            || _disputeGameType.raw() == GameTypes.SUPER_CANNON_KONA.raw();

        // PERMISSIONED_CANNON is the only **permissioned** type supported for an initial deploy.
        require(
            permissionless_ || _disputeGameType.raw() == GameTypes.PERMISSIONED_CANNON.raw(),
            "DeployOPChain: unsupported dispute game type"
        );

        respectedGameType_ = permissionless_
            ? _disputeGameType
            : (_isSuperRoot ? GameTypes.SUPER_PERMISSIONED : GameTypes.PERMISSIONED_CANNON);
    }

    /// @notice Returns whether the given OPCM has the SUPER_ROOT_GAMES_MIGRATION dev feature enabled.
    /// @param _opcm The OPCM to check.
    /// @return Whether SUPER_ROOT_GAMES_MIGRATION is enabled.
    function _isSuperRootEnabled(IOPContractsManagerV2 _opcm) internal view returns (bool) {
        return DevFeatures.isDevFeatureEnabled(_opcm.devFeatureBitmap(), DevFeatures.SUPER_ROOT_GAMES_MIGRATION);
    }

    /// @notice Creates a game config, clearing its bond and arguments when disabled.
    function _createGameConfig(
        bool _enabled,
        uint256 _initBond,
        GameType _gameType,
        bytes memory _enabledArgs
    )
        internal
        pure
        returns (IOPContractsManagerUtils.DisputeGameConfig memory)
    {
        return IOPContractsManagerUtils.DisputeGameConfig({
            enabled: _enabled,
            initBond: _enabled ? _initBond : 0,
            gameType: _gameType,
            gameArgs: _enabled ? _enabledArgs : bytes("")
        });
    }

    // -------- Validations --------

    /// @notice Checks if the input is valid.
    /// @param _i The input to check.
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
        bool superRoot = _isSuperRootEnabled(IOPContractsManagerV2(_i.opcm));
        (bool permissionless,) = _initialDeployGameSelection(_i.disputeGameType, superRoot);
        if (permissionless) {
            GameType expectedGameType = superRoot ? GameTypes.SUPER_CANNON_KONA : GameTypes.CANNON_KONA;
            require(
                _i.disputeGameType.raw() == expectedGameType.raw(),
                "DeployOPChainInput: dispute game type does not match OPCM mode"
            );
        }

        require(_i.disputeMaxGameDepth != 0, "DeployOPChainInput: disputeMaxGameDepth not set");
        require(_i.disputeSplitDepth != 0, "DeployOPChainInput: disputeSplitDepth not set");
        require(_i.disputeMaxClockDuration.raw() != 0, "DeployOPChainInput: disputeMaxClockDuration not set");
        require(_i.disputeAbsolutePrestate.raw() != bytes32(0), "DeployOPChainInput: disputeAbsolutePrestate not set");
        require(_i.startingAnchorRoot.root.raw() != bytes32(0), "DeployOPChainInput: startingAnchorRoot not set");
        require(
            _i.startingAnchorRoot.l2SequenceNumber < type(uint64).max,
            "DeployOPChainInput: startingAnchorRoot.l2SequenceNumber too large"
        );

        if (_i.disputeGameType.raw() == GameTypes.CANNON_KONA.raw()) {
            require(_i.cannonAbsolutePrestate.raw() != bytes32(0), "DeployOPChainInput: cannonAbsolutePrestate not set");
            // The two prestates commit to different fault-proof programs (op-program vs Kona),
            // so equal values always indicate a misconfigured producer.
            require(
                _i.cannonAbsolutePrestate.raw() != _i.disputeAbsolutePrestate.raw(),
                "DeployOPChainInput: cannonAbsolutePrestate must differ from disputeAbsolutePrestate"
            );
        }

        if (permissionless) {
            require(
                _i.startingAnchorRoot.root.raw() != ScriptConstants.DEFAULT_OUTPUT_ROOT().root.raw(),
                "DeployOPChainInput: permissionless startingAnchorRoot cannot be placeholder"
            );
        }
    }

    /// @notice Checks if the output is valid.
    /// @param _i The input to check.
    /// @param _o The output to check.
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

    /// @notice Asserts that the deploy is valid.
    /// @param _i The input to check.
    /// @param _o The output to check.
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
            SuperchainConfig: address(_i.superchainConfig)
        });

        // Check dispute games and get superchain config
        IOPContractsManagerV2 opcmV2 = IOPContractsManagerV2(_i.opcm);
        bool superRoot = _isSuperRootEnabled(opcmV2);
        IOPContractsManagerContainer.Implementations memory implementations = opcmV2.implementations();

        (bool permissionless, GameType respectedGameType) = _initialDeployGameSelection(_i.disputeGameType, superRoot);
        address expectedPermissionedDGImpl =
            superRoot ? implementations.superPermissionedDisputeGameImpl : implementations.permissionedDisputeGameImpl;
        address expectedFaultDGImpl =
            superRoot ? implementations.superFaultDisputeGameImpl : implementations.faultDisputeGameImpl;
        address expectedRespectedDGImpl = permissionless ? expectedFaultDGImpl : expectedPermissionedDGImpl;
        ChainAssertions.checkDisputeGameFactory(
            _o.disputeGameFactoryProxy, _i.opChainProxyAdminOwner, expectedRespectedDGImpl, true, respectedGameType
        );
        require(
            address(_o.faultDisputeGame) == (permissionless ? expectedRespectedDGImpl : address(0)),
            "DeployOPChain: faultDisputeGame output mismatch"
        );
        require(
            address(_o.permissionedDisputeGame) == expectedPermissionedDGImpl,
            "DeployOPChain: permissionedDisputeGame output mismatch"
        );
        ChainAssertions.checkAnchorStateRegistryProxy(
            _o.anchorStateRegistryProxy, true, respectedGameType, _i.startingAnchorRoot
        );
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
        _assertValidOPChainProxyAdmin(_i, _o);
    }

    /// @notice Asserts that the OPChainProxyAdmin is valid based on the input and output of the deployment.
    /// @param _doi The input to the deployment.
    /// @param _doo The output of the deployment.
    function _assertValidOPChainProxyAdmin(Types.DeployOPChainInput memory _doi, Output memory _doo) internal {
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
}
