// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { OPContractsManagerStandardValidator } from "src/L1/OPContractsManagerStandardValidator.sol";
import { StorageSetter } from "src/universal/StorageSetter.sol";

// Libraries
import { Blueprint } from "src/libraries/Blueprint.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Bytes } from "src/libraries/Bytes.sol";
import { Claim, Duration, GameType, GameTypes, Proposal, Hash } from "src/dispute/lib/Types.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Features } from "src/libraries/Features.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { IBigStepper } from "interfaces/dispute/IBigStepper.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { ISuperFaultDisputeGame } from "interfaces/dispute/ISuperFaultDisputeGame.sol";
import { ISuperPermissionedDisputeGame } from "interfaces/dispute/ISuperPermissionedDisputeGame.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProtocolVersions } from "interfaces/L1/IProtocolVersions.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { IOptimismPortalInterop } from "interfaces/L1/IOptimismPortalInterop.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1ERC721Bridge } from "interfaces/L1/IL1ERC721Bridge.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";

contract OPContractsManager is ISemver {
    struct Blueprints {
        address addressManager;
        address proxy;
        address proxyAdmin;
        address l1ChugSplashProxy;
        address resolvedDelegateProxy;
    }

    struct Implementations {
        address superchainConfigImpl;
        address protocolVersionsImpl;
        address l1ERC721BridgeImpl;
        address optimismPortalImpl;
        address optimismPortalInteropImpl;
        address ethLockboxImpl;
        address systemConfigImpl;
        address optimismMintableERC20FactoryImpl;
        address l1CrossDomainMessengerImpl;
        address l1StandardBridgeImpl;
        address disputeGameFactoryImpl;
        address anchorStateRegistryImpl;
        address delayedWETHImpl;
        address mipsImpl;
        address faultDisputeGameImpl;
        address permissionedDisputeGameImpl;
        address superFaultDisputeGameImpl;
        address superPermissionedDisputeGameImpl;
    }

    struct FaultDisputeGameConfig {
        Claim absolutePrestate;
    }

    struct PermissionedDisputeGameConfig {
        Claim absolutePrestate;
        address proposer;
        address challenger;
    }

    struct DisputeGameConfig {
        GameType gameType;
        bytes gameArgs;
    }

    struct SystemRoles {
        address proxyAdminOwner;
        address systemConfigOwner;
        address unsafeBlockSigner;
        bytes32 batcherHash;
    }

    struct L2SystemConfig {
        uint32 basefeeScalar;
        uint32 blobBasefeeScalar;
        uint64 gasLimit;
        uint256 l2ChainId;
        IResourceMetering.ResourceConfig resourceConfig;
    }

    struct AnchorStateConfig {
        bytes startingAnchorRoot;
        GameType startingRespectedGameType;
    }

    struct UpgradeInput {
        ISystemConfig systemConfigProxy;
        DisputeGameConfig[] disputeGameConfigs;
    }

    struct DisputeGameContracts {
        IDisputeGame disputeGame;
        IDelayedWETH delayedWETH;
    }

    struct ChainContracts {
        ISystemConfig systemConfig;
        IProxyAdmin proxyAdmin;
        IAddressManager addressManager;
        ISuperchainConfig superchainConfig;
        IL1CrossDomainMessenger l1CrossDomainMessenger;
        IL1ERC721Bridge l1ERC721Bridge;
        IL1StandardBridge l1StandardBridge;
        IOptimismPortal optimismPortal;
        IETHLockbox ethLockbox;
        IOptimismMintableERC20Factory optimismMintableERC20Factory;
        IDisputeGameFactory disputeGameFactory;
        IAnchorStateRegistry anchorStateRegistry;
        IDelayedWETH delayedWETH;
    }

    struct FullConfig {
        string saltMixer;
        SystemRoles roles;
        L2SystemConfig l2SystemConfig;
        DisputeGameConfig[] disputeGameConfigs;
        AnchorStateConfig anchorStateConfig;
    }

    struct ExecutionInput {
        ChainContracts cts;
        FullConfig cfg;
    }

    struct ExecutionOutput {
        ChainContracts cts;
    }

    struct ProxyDeployArgs {
        uint256 l2ChainId;
        IProxyAdmin proxyAdmin;
        string saltMixer;
    }

    error OPContractsManager_SuperchainConfigNeedsUpgrade();
    error OPContractsManager_UnknownGameType();

    Blueprints public bps;

    Implementations public impls;

    /// @custom:semver 3.3.0
    function version() public pure virtual returns (string memory) {
        return "3.3.0";
    }

    constructor(Blueprints memory _bps, Implementations memory _impls) {
        bps = _bps;
        impls = _impls;
    }

    function deploy(FullConfig memory _input) external returns (ExecutionOutput memory) {
        // Start building the execution input.
        ExecutionInput memory inp;

        // Deploy already has the full config, use it.
        inp.cfg = _input;

        // ProxyAdmin and AddressManager are special cases, not deployed as proxies.
        inp.cts.proxyAdmin = IProxyAdmin(
            Blueprint.deployFrom(
                bps.proxyAdmin,
                _makeSalt(inp.cfg.l2SystemConfig.l2ChainId, inp.cfg.saltMixer, "ProxyAdmin"),
                abi.encode(address(this))
            )
        );
        inp.cts.addressManager = IAddressManager(
            Blueprint.deployFrom(
                bps.addressManager,
                _makeSalt(inp.cfg.l2SystemConfig.l2ChainId, inp.cfg.saltMixer, "AddressManager"),
                abi.encode()
            )
        );

        // L1CrossDomainMessenger is a special case ResolvedDelegateProxy (legacy).
        string memory l1XdmName = "OVM_L1CrossDomainMessenger";
        inp.cts.l1CrossDomainMessenger = IL1CrossDomainMessenger(
            Blueprint.deployFrom(
                bps.resolvedDelegateProxy,
                _makeSalt(inp.cfg.l2SystemConfig.l2ChainId, inp.cfg.saltMixer, "L1CrossDomainMessenger"),
                abi.encode(inp.cts.addressManager, l1XdmName)
            )
        );

        // ResolvedDelegateProxy requires setting the proxy type on the ProxyAdmin.
        inp.cts.proxyAdmin.setProxyType(address(inp.cts.l1CrossDomainMessenger), IProxyAdmin.ProxyType.RESOLVED);
        inp.cts.proxyAdmin.setImplementationName(address(inp.cts.l1CrossDomainMessenger), l1XdmName);

        // L1StandardBridge is a special case ChugSplashProxy (legacy).
        inp.cts.l1StandardBridge = IL1StandardBridge(
            payable(
                Blueprint.deployFrom(
                    bps.l1ChugSplashProxy,
                    _makeSalt(inp.cfg.l2SystemConfig.l2ChainId, inp.cfg.saltMixer, "L1StandardBridge"),
                    abi.encode(inp.cts.proxyAdmin)
                )
            )
        );

        // ChugSplashProxy requires setting the proxy type on the ProxyAdmin.
        inp.cts.proxyAdmin.setProxyType(address(inp.cts.l1StandardBridge), IProxyAdmin.ProxyType.CHUGSPLASH);

        // Everything else is deployed as a proxy.
        // Set up the deploy args once, keeps the code cleaner.
        ProxyDeployArgs memory proxyDeployArgs = ProxyDeployArgs({
            l2ChainId: inp.cfg.l2SystemConfig.l2ChainId,
            proxyAdmin: inp.cts.proxyAdmin,
            saltMixer: inp.cfg.saltMixer
        });

        // Deploy all of system proxies.
        inp.cts.systemConfig = ISystemConfig(_makeProxy(proxyDeployArgs, "SystemConfig"));
        inp.cts.l1ERC721Bridge = IL1ERC721Bridge(_makeProxy(proxyDeployArgs, "L1ERC721Bridge"));
        inp.cts.optimismPortal = IOptimismPortal(payable(_makeProxy(proxyDeployArgs, "OptimismPortal")));
        inp.cts.ethLockbox = IETHLockbox(_makeProxy(proxyDeployArgs, "ETHLockbox"));
        inp.cts.disputeGameFactory = IDisputeGameFactory(_makeProxy(proxyDeployArgs, "DisputeGameFactory"));
        inp.cts.anchorStateRegistry = IAnchorStateRegistry(_makeProxy(proxyDeployArgs, "AnchorStateRegistry"));
        inp.cts.delayedWETH = IDelayedWETH(payable(_makeProxy(proxyDeployArgs, "DelayedWETH")));
        inp.cts.optimismMintableERC20Factory =
            IOptimismMintableERC20Factory(_makeProxy(proxyDeployArgs, "OptimismMintableERC20Factory"));

        // Execute the deployment.
        return _execute(inp);
    }

    function upgrade(UpgradeInput memory _input) external returns (ExecutionOutput memory) {
        ExecutionInput memory inp;

        // Grab all of the easy contracts.
        inp.cts.systemConfig = _input.systemConfigProxy;
        inp.cts.proxyAdmin = inp.cts.systemConfig.proxyAdmin();
        inp.cts.addressManager = inp.cts.proxyAdmin.addressManager();
        inp.cts.superchainConfig = inp.cts.systemConfig.superchainConfig();
        inp.cts.l1CrossDomainMessenger = IL1CrossDomainMessenger(inp.cts.systemConfig.l1CrossDomainMessenger());
        inp.cts.l1ERC721Bridge = IL1ERC721Bridge(inp.cts.systemConfig.l1ERC721Bridge());
        inp.cts.l1StandardBridge = IL1StandardBridge(payable(inp.cts.systemConfig.l1StandardBridge()));
        inp.cts.optimismPortal = IOptimismPortal(payable(inp.cts.systemConfig.optimismPortal()));
        inp.cts.ethLockbox = inp.cts.optimismPortal.ethLockbox();
        inp.cts.disputeGameFactory = inp.cts.optimismPortal.disputeGameFactory();
        inp.cts.anchorStateRegistry = inp.cts.optimismPortal.anchorStateRegistry();
        inp.cts.delayedWETH = IDelayedWETH(payable(address(0))); // TODO
        inp.cts.optimismMintableERC20Factory =
            IOptimismMintableERC20Factory(inp.cts.systemConfig.optimismMintableERC20Factory());

        // Extract system roles.
        inp.cfg.roles.proxyAdminOwner = inp.cts.optimismPortal.proxyAdminOwner();
        inp.cfg.roles.systemConfigOwner = inp.cts.systemConfig.owner();
        inp.cfg.roles.batcherHash = inp.cts.systemConfig.batcherHash();
        inp.cfg.roles.unsafeBlockSigner = inp.cts.systemConfig.unsafeBlockSigner();

        // Extract system config.
        inp.cfg.l2SystemConfig.basefeeScalar = inp.cts.systemConfig.basefeeScalar();
        inp.cfg.l2SystemConfig.blobBasefeeScalar = inp.cts.systemConfig.blobbasefeeScalar();
        inp.cfg.l2SystemConfig.gasLimit = inp.cts.systemConfig.gasLimit();
        inp.cfg.l2SystemConfig.l2ChainId = inp.cts.systemConfig.l2ChainId();
        inp.cfg.l2SystemConfig.resourceConfig = inp.cts.systemConfig.resourceConfig();

        // Extract AnchorStateRegistry parameters.
        (Hash root, uint256 l2SequenceNumber) = inp.cts.anchorStateRegistry.getAnchorRoot();
        inp.cfg.anchorStateConfig.startingAnchorRoot = abi.encode(root, l2SequenceNumber);
        inp.cfg.anchorStateConfig.startingRespectedGameType = inp.cts.anchorStateRegistry.respectedGameType();

        // Generate a salt mixer based on the SystemConfig address.
        inp.cfg.saltMixer = string(bytes.concat(bytes32(uint256(uint160(address(inp.cts.systemConfig))))));

        // Execute the upgrade.
        return _execute(inp);
    }

    function _execute(ExecutionInput memory _input) internal returns (ExecutionOutput memory output_) {
        // Make sure the provided SuperchainConfig is up to date.
        if (
            SemverComp.lt(
                _input.cts.superchainConfig.version(), ISuperchainConfig(impls.superchainConfigImpl).version()
            )
        ) {
            revert OPContractsManager_SuperchainConfigNeedsUpgrade();
        }

        // Update the SystemConfig.
        _resetAndInitialize(
            _input.cts.proxyAdmin,
            address(_input.cts.systemConfig),
            impls.systemConfigImpl,
            abi.encodeCall(
                ISystemConfig.initialize,
                (
                    _input.cfg.roles.systemConfigOwner,
                    _input.cfg.l2SystemConfig.basefeeScalar,
                    _input.cfg.l2SystemConfig.blobBasefeeScalar,
                    _input.cfg.roles.batcherHash,
                    _input.cfg.l2SystemConfig.gasLimit,
                    _input.cfg.roles.unsafeBlockSigner,
                    _input.cfg.l2SystemConfig.resourceConfig,
                    _makeBatchInboxAddress(_input.cfg.l2SystemConfig.l2ChainId),
                    ISystemConfig.Addresses({
                        l1CrossDomainMessenger: address(_input.cts.l1CrossDomainMessenger),
                        l1ERC721Bridge: address(_input.cts.l1ERC721Bridge),
                        l1StandardBridge: address(_input.cts.l1StandardBridge),
                        optimismPortal: address(_input.cts.optimismPortal),
                        optimismMintableERC20Factory: address(_input.cts.optimismMintableERC20Factory)
                    }),
                    _input.cfg.l2SystemConfig.l2ChainId,
                    _input.cts.superchainConfig
                )
            )
        );

        // Update the OptimismPortal.
        if (_isDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP)) {
            _resetAndInitialize(
                _input.cts.proxyAdmin,
                address(_input.cts.optimismPortal),
                impls.optimismPortalInteropImpl,
                abi.encodeCall(
                    IOptimismPortalInterop.initialize,
                    (_input.cts.systemConfig, _input.cts.anchorStateRegistry, _input.cts.ethLockbox)
                )
            );
        } else {
            _resetAndInitialize(
                _input.cts.proxyAdmin,
                address(_input.cts.optimismPortal),
                impls.optimismPortalImpl,
                abi.encodeCall(IOptimismPortal.initialize, (_input.cts.systemConfig, _input.cts.anchorStateRegistry))
            );
        }

        // Update the ETHLockbox.
        IOptimismPortal[] memory portals = new IOptimismPortal[](1);
        portals[0] = _input.cts.optimismPortal;
        _resetAndInitialize(
            _input.cts.proxyAdmin,
            address(_input.cts.ethLockbox),
            impls.ethLockboxImpl,
            abi.encodeCall(IETHLockbox.initialize, (_input.cts.systemConfig, portals))
        );

        // Update the L1CrossDomainMessenger.
        _resetAndInitialize(
            _input.cts.proxyAdmin,
            address(_input.cts.l1CrossDomainMessenger),
            impls.l1CrossDomainMessengerImpl,
            abi.encodeCall(IL1CrossDomainMessenger.initialize, (_input.cts.systemConfig, _input.cts.optimismPortal))
        );

        // Update the L1StandardBridge.
        _resetAndInitialize(
            _input.cts.proxyAdmin,
            address(_input.cts.l1StandardBridge),
            impls.l1StandardBridgeImpl,
            abi.encodeCall(IL1StandardBridge.initialize, (_input.cts.l1CrossDomainMessenger, _input.cts.systemConfig))
        );

        // Update the L1ERC721Bridge.
        _resetAndInitialize(
            _input.cts.proxyAdmin,
            address(_input.cts.l1ERC721Bridge),
            impls.l1ERC721BridgeImpl,
            abi.encodeCall(IL1ERC721Bridge.initialize, (_input.cts.l1CrossDomainMessenger, _input.cts.systemConfig))
        );

        // Update the OptimismMintableERC20Factory.
        _resetAndInitialize(
            _input.cts.proxyAdmin,
            address(_input.cts.optimismMintableERC20Factory),
            impls.optimismMintableERC20FactoryImpl,
            abi.encodeCall(IOptimismMintableERC20Factory.initialize, (address(_input.cts.l1StandardBridge)))
        );

        // Update the DisputeGameFactory.
        _resetAndInitialize(
            _input.cts.proxyAdmin,
            address(_input.cts.disputeGameFactory),
            impls.disputeGameFactoryImpl,
            abi.encodeCall(IDisputeGameFactory.initialize, (address(this)))
        );

        // Update the DelayedWETH.
        _resetAndInitialize(
            _input.cts.proxyAdmin,
            address(_input.cts.delayedWETH),
            impls.delayedWETHImpl,
            abi.encodeCall(IDelayedWETH.initialize, (_input.cts.systemConfig))
        );

        // Update the AnchorStateRegistry.
        _resetAndInitialize(
            _input.cts.proxyAdmin,
            address(_input.cts.anchorStateRegistry),
            impls.anchorStateRegistryImpl,
            abi.encodeCall(
                IAnchorStateRegistry.initialize,
                (
                    _input.cts.systemConfig,
                    _input.cts.disputeGameFactory,
                    abi.decode(_input.cfg.anchorStateConfig.startingAnchorRoot, (Proposal)),
                    _input.cfg.anchorStateConfig.startingRespectedGameType
                )
            )
        );

        // Update the DisputeGame config and implementations.
        for (uint256 i = 0; i < _input.cfg.disputeGameConfigs.length; i++) {
            _input.cts.disputeGameFactory.setImplementation(
                _input.cfg.disputeGameConfigs[i].gameType,
                IDisputeGame(_getGameImpl(_input.cfg.disputeGameConfigs[i].gameType)),
                _makeGameArgs(_input, _input.cfg.disputeGameConfigs[i])
            );
        }

        // Transfer ownership of the DisputeGameFactory to the proxyAdminOwner.
        _input.cts.disputeGameFactory.transferOwnership(address(_input.cfg.roles.proxyAdminOwner));

        // Transfer ownership of the ProxyAdmin to the proxyAdminOwner.
        _input.cts.proxyAdmin.transferOwnership(_input.cfg.roles.proxyAdminOwner);

        // Return contracts as the execution output.
        return ExecutionOutput({ cts: _input.cts });
    }

    /// @notice Makes a salt for a contract deployment.
    /// @param _l2ChainId The L2 chain ID.
    /// @param _saltMixer The salt mixer.
    /// @param _contractName The name of the contract to deploy.
    /// @return The salt for the contract deployment.
    function _makeSalt(
        uint256 _l2ChainId,
        string memory _saltMixer,
        string memory _contractName
    )
        internal
        pure
        returns (bytes32)
    {
        return keccak256(abi.encode(_l2ChainId, _saltMixer, _contractName));
    }

    /// @notice Helper function for deploying a proxy with nicer arguments than deployProxy.
    /// @param _args The arguments for the proxy deployment.
    /// @param _contractName The name of the contract to deploy.
    /// @return The address of the deployed proxy.
    function _makeProxy(ProxyDeployArgs memory _args, string memory _contractName) internal returns (address) {
        bytes32 salt = _makeSalt(_args.l2ChainId, _args.saltMixer, _contractName);
        return Blueprint.deployFrom(bps.proxy, salt, abi.encode(_args.proxyAdmin));
    }

    /// @notice Generates a batch inbox address for a given L2 chain ID.
    /// @param _l2ChainId The L2 chain ID.
    /// @return The batch inbox address.
    function _makeBatchInboxAddress(uint256 _l2ChainId) internal pure returns (address) {
        bytes1 versionByte = 0x00;
        bytes32 hashedChainId = keccak256(bytes.concat(bytes32(_l2ChainId)));
        bytes19 first19Bytes = bytes19(hashedChainId);
        return address(uint160(bytes20(bytes.concat(versionByte, first19Bytes))));
    }

    /// @notice Checks if a development feature is enabled.
    /// @param _feature The feature to check.
    /// @return True if the feature is enabled, false otherwise.
    function _isDevFeatureEnabled(bytes32 _feature) internal view returns (bool) {
        return false;
    }

    /// @notice Resets the initialized slot for a contract and then initializes it.
    /// @param _proxyAdmin The proxy admin of the contract.
    /// @param _target The target of the contract.
    /// @param _implementation The implementation of the contract.
    /// @param _data The data to call the initializer with.
    function _resetAndInitialize(
        IProxyAdmin _proxyAdmin,
        address _target,
        address _implementation,
        bytes memory _data
    )
        internal
    {
        // TODO: Don't upgrade if implementation doesn't change.

        // Upgrade to StorageSetter.
        // TODO: Use a universal storage setter.
        _proxyAdmin.upgrade(payable(_target), address(new StorageSetter()));

        // Reset the initialized slot.
        // TODO: Support other than slot 0.
        StorageSetter(_target).setBytes32(bytes32(0), bytes32(0));

        // Upgrade to the implementation and call the initializer.
        _proxyAdmin.upgradeAndCall(payable(address(_target)), _implementation, _data);
    }

    /// @notice Returns the implementation contract address for a given game type.
    /// @param _gameType The game type to get the implementation for.
    /// @return The implementation contract address for the game type.
    function _getGameImpl(GameType _gameType) internal view returns (address) {
        if (_gameType.raw() == GameTypes.CANNON.raw()) {
            return impls.faultDisputeGameImpl;
        } else if (_gameType.raw() == GameTypes.PERMISSIONED_CANNON.raw()) {
            return impls.permissionedDisputeGameImpl;
        } else if (_gameType.raw() == GameTypes.SUPER_CANNON.raw()) {
            return impls.superFaultDisputeGameImpl;
        } else if (_gameType.raw() == GameTypes.SUPER_PERMISSIONED_CANNON.raw()) {
            return impls.superPermissionedDisputeGameImpl;
        } else {
            // TODO: Support custom game types.
            revert OPContractsManager_UnknownGameType();
        }
    }

    function _makeGameArgs(
        ExecutionInput memory _input,
        DisputeGameConfig memory _cfg
    )
        internal
        view
        returns (bytes memory)
    {
        if (_cfg.gameType.raw() == GameTypes.CANNON.raw() || _cfg.gameType.raw() == GameTypes.SUPER_CANNON.raw()) {
            FaultDisputeGameConfig memory parsedCfg = abi.decode(_cfg.gameArgs, (FaultDisputeGameConfig));
            return abi.encodePacked(
                parsedCfg.absolutePrestate,
                impls.mipsImpl,
                _input.cts.anchorStateRegistry,
                _input.cts.delayedWETH,
                _input.cfg.l2SystemConfig.l2ChainId
            );
        } else if (
            _cfg.gameType.raw() == GameTypes.PERMISSIONED_CANNON.raw()
                || _cfg.gameType.raw() == GameTypes.SUPER_PERMISSIONED_CANNON.raw()
        ) {
            PermissionedDisputeGameConfig memory parsedCfg = abi.decode(_cfg.gameArgs, (PermissionedDisputeGameConfig));
            return abi.encodePacked(
                parsedCfg.absolutePrestate,
                impls.mipsImpl,
                _input.cts.anchorStateRegistry,
                _input.cts.delayedWETH,
                _input.cfg.l2SystemConfig.l2ChainId,
                parsedCfg.proposer,
                parsedCfg.challenger
            );
        } else {
            // TODO: Support custom data if a dev flag is enabled.
            revert OPContractsManager_UnknownGameType();
        }
    }
}
