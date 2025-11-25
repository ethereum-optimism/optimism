// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { LibString } from "@solady/utils/LibString.sol";
import { Blueprint } from "src/libraries/Blueprint.sol";
import { Claim, GameType, GameTypes, Proposal } from "src/dispute/lib/Types.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Features } from "src/libraries/Features.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { IOptimismPortalInterop } from "interfaces/L1/IOptimismPortalInterop.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1ERC721Bridge } from "interfaces/L1/IL1ERC721Bridge.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IStorageSetter } from "interfaces/universal/IStorageSetter.sol";
import { IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerContainer.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";

/// @title OPContractsManagerV2
/// @notice OPContractsManagerV2 is an enhanced version of OPContractsManager. OPContractsManagerV2
///         provides a simplified, minimized way of handling upgrades and deployments of OP Stack
///         chains. Each official release of the OP Stack contracts is packaged with its own unique
///         instance of OPContractsManagerV2 that handles the state transition for that particular
///         release.
/// @dev When adding a new dispute game type, if your dispute game requires configuration that
///      differs from configuration used by other dispute game types, you will need to add a new
///      configuration struct and then add parsing logic for that struct in the _makeGameArgs
///      function. You will also need to return the correct game implementation in _getGameImpl.
/// @dev When adding a net-new input, simply add the input to FullConfig and add the corresponding
///      logic for loading that input in _loadFullConfig. NOTE that when adding a completely new
///      input, users upgrading an existing chain will need to supply that input in the form of an
///      override as part of the UpgradeInput struct.
/// @dev If you were going to build a V3 of OPCM, you probably want to make this look a lot more
///      like Terraform. The V2 design is trending in the direction of being Terraform-like, but it
///      doesn't quite get there yet in an attempt to be a more incremental improvement over the V1
///      design. Look at _apply, squint, and imagine that it can output an upgrade plan rather than
///      actually executing the upgrade, and then you'll see how it can be improved.
contract OPContractsManagerV2 is ISemver {
    /// @notice Configuration struct for the FaultDisputeGame.
    struct FaultDisputeGameConfig {
        Claim absolutePrestate;
    }

    /// @notice Configuration struct for the PermissionedDisputeGame.
    struct PermissionedDisputeGameConfig {
        Claim absolutePrestate;
        address proposer;
        address challenger;
    }

    /// @notice Generic dispute game configuration data.
    struct DisputeGameConfig {
        bool enabled;
        uint256 initBond;
        GameType gameType;
        bytes gameArgs;
    }

    /// @notice Contracts that represent the Superchain system.
    struct SuperchainContracts {
        ISuperchainConfig superchainConfig;
    }

    /// @notice Contracts that represent the full chain system.
    struct ChainContracts {
        ISystemConfig systemConfig;
        IProxyAdmin proxyAdmin;
        IAddressManager addressManager;
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

    /// @notice Struct that represents an additional instruction for an upgrade. Each upgrade has
    ///         its own set of extra upgrade instructions that may or may not be required. We use
    ///         this struct to keep the upgrade interface the same each time.
    struct ExtraInstruction {
        string key;
        bytes data;
    }

    /// @notice Full chain management configuration.
    struct FullConfig {
        // Basic deployment configuration.
        string saltMixer;
        ISuperchainConfig superchainConfig;
        // System role configuration.
        address proxyAdminOwner;
        address systemConfigOwner;
        address unsafeBlockSigner;
        address batcher;
        // Anchor state configuration.
        Proposal startingAnchorRoot;
        GameType startingRespectedGameType;
        // L2 system configuration.
        uint32 basefeeScalar;
        uint32 blobBasefeeScalar;
        uint64 gasLimit;
        uint256 l2ChainId;
        IResourceMetering.ResourceConfig resourceConfig;
        // Dispute game configuration.
        DisputeGameConfig[] disputeGameConfigs;
    }

    /// @notice Partial input required for an upgrade.
    struct UpgradeInput {
        ISystemConfig systemConfig;
        DisputeGameConfig[] disputeGameConfigs;
        ExtraInstruction[] extraInstructions;
    }

    /// @notice Input for upgrading Superchain contracts.
    struct SuperchainUpgradeInput {
        ISuperchainConfig superchainConfig;
        ExtraInstruction[] extraInstructions;
    }

    /// @notice Helper struct for deploying proxies, keeps code cleaner.
    struct ProxyDeployArgs {
        IProxyAdmin proxyAdmin;
        IAddressManager addressManager;
        uint256 l2ChainId;
        string saltMixer;
    }

    /// @notice Emitted when a proxy is created by this contract.
    /// @param name  The name of the proxy.
    /// @param proxy The address of the proxy.
    event ProxyCreation(string name, address proxy);

    /// @notice Thrown when the SuperchainConfig needs to be upgraded.
    error OPContractsManagerV2_SuperchainConfigNeedsUpgrade();

    /// @notice Thrown when an unsupported game type is provided.
    error OPContractsManagerV2_UnsupportedGameType();

    /// @notice Thrown when an invalid game config is provided.
    error OPContractsManagerV2_InvalidGameConfigs();

    /// @notice Thrown when an invalid upgrade input is provided.
    error OPContractsManagerV2_InvalidUpgradeInput();

    /// @notice Thrown when a proxy must be loaded but couldn't be.
    error OPContractsManagerV2_ProxyMustLoad(string _name);

    /// @notice Thrown when user attempts to downgrade a contract.
    error OPContractsManagerV2_DowngradeNotAllowed(address _contract);

    /// @notice Thrown when an invalid upgrade instruction is provided.
    error OPContractsManagerV2_InvalidUpgradeInstruction();

    /// @notice Thrown when a config load fails.
    error OPContractsManagerV2_ConfigLoadFailed(string _name);

    /// @notice Container of blueprint and implementation contract addresses.
    IOPContractsManagerContainer public immutable contractsContainer;

    /// @notice Address of the Standard Validator for this OPCM release.
    IOPContractsManagerStandardValidator public immutable standardValidator;

    /// @notice The version of the OPCM contract.
    string public constant version = "6.0.0";

    /// @notice Special constant key for the PermittedProxyDeployment instruction.
    string internal constant PERMITTED_PROXY_DEPLOYMENT_KEY = "PermittedProxyDeployment";

    /// @notice Special constant value for the PermittedProxyDeployment instruction to permit all
    ///         contracts to be deployed. Only to be used for deployments.
    bytes internal constant PERMIT_ALL_CONTRACTS_INSTRUCTION = bytes("ALL");

    /// @param _contractsContainer The container of blueprint and implementation contract addresses.
    /// @param _standardValidator The standard validator for this OPCM release.
    constructor(
        IOPContractsManagerContainer _contractsContainer,
        IOPContractsManagerStandardValidator _standardValidator
    ) {
        contractsContainer = _contractsContainer;
        standardValidator = _standardValidator;
    }

    ///////////////////////////////////////////////////////////////////////////
    //                   PUBLIC CHAIN MANAGEMENT FUNCTIONS                   //
    ///////////////////////////////////////////////////////////////////////////

    /// @notice Upgrades the Superchain contracts. Currently this is limited to the
    ///         SuperchainConfig contract, but may eventually expand to include other
    ///         Superchain-wide contracts.
    /// @param _inp The input for the Superchain upgrade.
    function upgradeSuperchain(SuperchainUpgradeInput memory _inp) external returns (SuperchainContracts memory) {
        // NOTE: Since this function is very minimal and only upgrades the SuperchainConfig
        // contract, not bothering to fully follow the pattern of the normal chain upgrade flow.
        // If we expand the scope of this function to add other Superchain-wide contracts, we'll
        // probably want to start following a similar pattern to the chain upgrade flow.

        // Upgrade the SuperchainConfig if it has changed.
        _upgrade(
            IProxyAdmin(_inp.superchainConfig.proxyAdmin()),
            address(_inp.superchainConfig),
            implementations().superchainConfigImpl,
            abi.encodeCall(ISuperchainConfig.initialize, (_inp.superchainConfig.guardian()))
        );

        // Return the Superchain contracts.
        return SuperchainContracts({ superchainConfig: _inp.superchainConfig });
    }

    /// @notice Deploys a new chain from full config.
    /// @param _cfg The full chain deployment configuration.
    /// @return The chain contracts.
    function deploy(FullConfig memory _cfg) external returns (ChainContracts memory) {
        // Deploy is the ONLY place where we allow the "ALL" permission for proxy deployment.
        ExtraInstruction[] memory instructions = new ExtraInstruction[](1);
        instructions[0] =
            ExtraInstruction({ key: PERMITTED_PROXY_DEPLOYMENT_KEY, data: PERMIT_ALL_CONTRACTS_INSTRUCTION });

        // Load the chain contracts.
        ChainContracts memory cts =
            _loadChainContracts(ISystemConfig(address(0)), _cfg.l2ChainId, _cfg.saltMixer, instructions);

        // Execute the deployment.
        return _apply(_cfg, cts, true);
    }

    /// @notice Upgrades a chain based on the upgrade input.
    /// @param _inp The chain upgrade input.
    /// @return The upgraded chain contracts.
    function upgrade(UpgradeInput memory _inp) external returns (ChainContracts memory) {
        // Assert that the upgrade instructions are valid.
        // NOTE for developers: We use the concept of upgrade instructions to help maintain the
        // principle that OPCM should be updated at the time that the feature is being developed
        // and not again later for "maintenance" work. For example, if you are adding a net-new
        // input to the SystemConfig contract, OPCMv1 would require that you also modify the
        // UpgradeInput struct to include that input. You would then later need to go back and
        // remove the input from the struct in some later upgrade. With OPCMv2, you can simply
        // update the _loadFullConfig function to include your new input and have users supply an
        // override for that particular upgrade (the upgrade won't work without the override)
        // without any need to later come back and remove the input from the struct or ever even
        // change the interface of OPCMv2 in the first place.
        _assertValidUpgradeInstructions(_inp.extraInstructions);

        // Load the chain contracts.
        ChainContracts memory cts =
            _loadChainContracts(_inp.systemConfig, _inp.systemConfig.l2ChainId(), "salt mixer", _inp.extraInstructions);

        // Load the full config.
        FullConfig memory cfg = _loadFullConfig(_inp, cts);

        // Execute the upgrade.
        return _apply(cfg, cts, false);
    }

    ///////////////////////////////////////////////////////////////////////////
    //                  INTERNAL CHAIN MANAGEMENT FUNCTIONS                  //
    ///////////////////////////////////////////////////////////////////////////

    /// @notice Asserts that the upgrade instructions array is valid.
    /// @param _extraInstructions The extra upgrade instructions for the chain.
    function _assertValidUpgradeInstructions(ExtraInstruction[] memory _extraInstructions) internal pure {
        for (uint256 i = 0; i < _extraInstructions.length; i++) {
            if (
                LibString.eq(_extraInstructions[i].key, PERMITTED_PROXY_DEPLOYMENT_KEY)
                    && LibString.eq(string(_extraInstructions[i].data), "DelayedWETH")
            ) {
                // Unified DelayedWETH is being deployed for the first time.
                // TODO:(#?????): Remove this allowance after unified DelayedWETH is deployed.
            } else {
                revert OPContractsManagerV2_InvalidUpgradeInstruction();
            }
        }
    }

    /// @notice Loads (or builds) the chain contracts from whatever exists.
    /// @param _systemConfig The SystemConfig contract.
    /// @param _l2ChainId The L2 chain ID.
    /// @param _saltMixer The salt mixer for creating new proxies if needed.
    /// @param _extraInstructions The extra upgrade instructions for the chain.
    /// @return The chain contracts.
    function _loadChainContracts(
        ISystemConfig _systemConfig,
        uint256 _l2ChainId,
        string memory _saltMixer,
        ExtraInstruction[] memory _extraInstructions
    )
        internal
        returns (ChainContracts memory)
    {
        // If the systemConfig is not initialized, we assume that the entire chain is new.
        bool isInitialDeployment = address(_systemConfig) == address(0);

        // ProxyAdmin, AddressManager, and SystemConfig are the three special cases where we handle
        // them differently than everything else because they're fundamental. Without these three
        // contracts we can't get anything else.
        IProxyAdmin proxyAdmin;
        IAddressManager addressManager;
        ISystemConfig systemConfig;
        if (isInitialDeployment) {
            // Deploy the ProxyAdmin.
            proxyAdmin = IProxyAdmin(
                Blueprint.deployFrom(
                    blueprints().proxyAdmin,
                    _computeSalt(_l2ChainId, _saltMixer, "ProxyAdmin"),
                    abi.encode(address(this))
                )
            );

            // Deploy the AddressManager.
            addressManager = IAddressManager(
                Blueprint.deployFrom(
                    blueprints().addressManager, _computeSalt(_l2ChainId, _saltMixer, "AddressManager"), abi.encode()
                )
            );

            // Set the AddressManager on the ProxyAdmin.
            proxyAdmin.setAddressManager(addressManager);

            // Transfer ownership of the AddressManager to the ProxyAdmin.
            addressManager.transferOwnership(address(proxyAdmin));

            // Deploy the SystemConfig.
            systemConfig = ISystemConfig(
                Blueprint.deployFrom(
                    blueprints().proxy,
                    _computeSalt(_l2ChainId, _saltMixer, "SystemConfig"),
                    abi.encode(address(proxyAdmin))
                )
            );
        } else {
            // Load-or-deploy pattern just generally doesn't make a lot of sense here. You could
            // theoretically do it, but not worth the complexity. Having this special handling for
            // how we load these three contracts is just cleaner/simpler.
            proxyAdmin = _systemConfig.proxyAdmin();
            addressManager = proxyAdmin.addressManager();
            systemConfig = _systemConfig;
        }

        // Set up the deploy args once, keeps the code cleaner.
        ProxyDeployArgs memory proxyDeployArgs = ProxyDeployArgs({
            proxyAdmin: proxyAdmin,
            addressManager: addressManager,
            l2ChainId: _l2ChainId,
            saltMixer: _saltMixer
        });

        // Now also load the portal, which contains the last few contract references. We do this
        // before we set up the rest of the struct so we can reference it.
        IOptimismPortal optimismPortal = IOptimismPortal(
            _loadOrDeployProxy(
                address(systemConfig),
                systemConfig.optimismPortal.selector,
                proxyDeployArgs,
                "OptimismPortal",
                _extraInstructions
            )
        );

        // ETHLockbox is a special case. It's only to be used or deployed if the ETH_LOCKBOX
        // feature is enabled. If this is an initial deployment, we'll deploy a proxy for it
        // largely because the legacy code expects this proxy to be deployed on initial deployment
        // though this doesn't mean we actually have to set it up and initialize it. If this is an
        // upgrade, we'll load/deploy the proxy only if the system feature is set.
        // NOTE: It's important that we don't try to load the proxy here if we're upgrading a chain
        // that doesn't have the feature enabled. Chains that don't have the feature enabled will
        // return address(0) for optimismPortal.ethLockbox(). If we try to load the proxy here, we
        // will revert because the contract returns the zero address (reverting is the safe thing
        // to do, so we want to revert, but that would break the upgrade flow).
        IETHLockbox ethLockbox;
        if (isInitialDeployment || systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX)) {
            ethLockbox = IETHLockbox(
                _loadOrDeployProxy(
                    address(optimismPortal),
                    optimismPortal.ethLockbox.selector,
                    proxyDeployArgs,
                    "ETHLockbox",
                    _extraInstructions
                )
            );
        }

        // For every other contract, we load-or-build the proxy. Each contract has a theoretical
        // source where the address would be found. If the address isn't found there, we assume the
        // address needs to be constructed.
        // NOTE: We call _loadOrDeployProxy for each contract (rather than iterating over some sort
        // of array) because (1) it's far easier to implement in Solidity and (2) it makes the code
        // easier to understand.
        return ChainContracts({
            systemConfig: systemConfig,
            proxyAdmin: proxyAdmin,
            addressManager: addressManager,
            optimismPortal: optimismPortal,
            ethLockbox: ethLockbox,
            l1CrossDomainMessenger: IL1CrossDomainMessenger(
                _loadOrDeployProxy(
                    address(systemConfig),
                    systemConfig.l1CrossDomainMessenger.selector,
                    proxyDeployArgs,
                    "L1CrossDomainMessenger",
                    _extraInstructions
                )
            ),
            l1ERC721Bridge: IL1ERC721Bridge(
                _loadOrDeployProxy(
                    address(systemConfig),
                    systemConfig.l1ERC721Bridge.selector,
                    proxyDeployArgs,
                    "L1ERC721Bridge",
                    _extraInstructions
                )
            ),
            l1StandardBridge: IL1StandardBridge(
                _loadOrDeployProxy(
                    address(systemConfig),
                    systemConfig.l1StandardBridge.selector,
                    proxyDeployArgs,
                    "L1StandardBridge",
                    _extraInstructions
                )
            ),
            optimismMintableERC20Factory: IOptimismMintableERC20Factory(
                _loadOrDeployProxy(
                    address(systemConfig),
                    systemConfig.optimismMintableERC20Factory.selector,
                    proxyDeployArgs,
                    "OptimismMintableERC20Factory",
                    _extraInstructions
                )
            ),
            disputeGameFactory: IDisputeGameFactory(
                _loadOrDeployProxy(
                    address(systemConfig),
                    systemConfig.disputeGameFactory.selector,
                    proxyDeployArgs,
                    "DisputeGameFactory",
                    _extraInstructions
                )
            ),
            anchorStateRegistry: IAnchorStateRegistry(
                _loadOrDeployProxy(
                    address(optimismPortal),
                    optimismPortal.anchorStateRegistry.selector,
                    proxyDeployArgs,
                    "AnchorStateRegistry",
                    _extraInstructions
                )
            ),
            delayedWETH: IDelayedWETH(
                _loadOrDeployProxy(
                    address(systemConfig),
                    systemConfig.delayedWETH.selector,
                    proxyDeployArgs,
                    "DelayedWETH",
                    _extraInstructions
                )
            )
        });
    }

    /// @notice Loads the full config from the upgrade input.
    /// @param _upgradeInput The upgrade input.
    /// @param _chainContracts The chain contracts.
    /// @return The full config.
    function _loadFullConfig(
        UpgradeInput memory _upgradeInput,
        ChainContracts memory _chainContracts
    )
        internal
        view
        returns (FullConfig memory)
    {
        // Load the full config.
        return FullConfig({
            saltMixer: string(bytes.concat(bytes32(uint256(uint160(address(_chainContracts.systemConfig)))))),
            superchainConfig: abi.decode(
                _loadBytes(
                    address(_chainContracts.systemConfig),
                    _chainContracts.systemConfig.superchainConfig.selector,
                    "overrides.cfg.superchainConfig",
                    _upgradeInput.extraInstructions
                ),
                (ISuperchainConfig)
            ),
            proxyAdminOwner: abi.decode(
                _loadBytes(
                    address(_chainContracts.optimismPortal),
                    _chainContracts.optimismPortal.proxyAdminOwner.selector,
                    "overrides.cfg.proxyAdminOwner",
                    _upgradeInput.extraInstructions
                ),
                (address)
            ),
            systemConfigOwner: abi.decode(
                _loadBytes(
                    address(_chainContracts.systemConfig),
                    _chainContracts.systemConfig.owner.selector,
                    "overrides.cfg.systemConfigOwner",
                    _upgradeInput.extraInstructions
                ),
                (address)
            ),
            unsafeBlockSigner: abi.decode(
                _loadBytes(
                    address(_chainContracts.systemConfig),
                    _chainContracts.systemConfig.unsafeBlockSigner.selector,
                    "overrides.cfg.unsafeBlockSigner",
                    _upgradeInput.extraInstructions
                ),
                (address)
            ),
            batcher: abi.decode(
                _loadBytes(
                    address(_chainContracts.systemConfig),
                    _chainContracts.systemConfig.batcherHash.selector,
                    "overrides.cfg.batcher",
                    _upgradeInput.extraInstructions
                ),
                (address)
            ),
            basefeeScalar: abi.decode(
                _loadBytes(
                    address(_chainContracts.systemConfig),
                    _chainContracts.systemConfig.basefeeScalar.selector,
                    "overrides.cfg.basefeeScalar",
                    _upgradeInput.extraInstructions
                ),
                (uint32)
            ),
            blobBasefeeScalar: abi.decode(
                _loadBytes(
                    address(_chainContracts.systemConfig),
                    _chainContracts.systemConfig.blobbasefeeScalar.selector,
                    "overrides.cfg.blobBasefeeScalar",
                    _upgradeInput.extraInstructions
                ),
                (uint32)
            ),
            gasLimit: abi.decode(
                _loadBytes(
                    address(_chainContracts.systemConfig),
                    _chainContracts.systemConfig.gasLimit.selector,
                    "overrides.cfg.gasLimit",
                    _upgradeInput.extraInstructions
                ),
                (uint64)
            ),
            l2ChainId: abi.decode(
                _loadBytes(
                    address(_chainContracts.systemConfig),
                    _chainContracts.systemConfig.l2ChainId.selector,
                    "overrides.cfg.l2ChainId",
                    _upgradeInput.extraInstructions
                ),
                (uint256)
            ),
            resourceConfig: abi.decode(
                _loadBytes(
                    address(_chainContracts.systemConfig),
                    _chainContracts.systemConfig.resourceConfig.selector,
                    "overrides.cfg.resourceConfig",
                    _upgradeInput.extraInstructions
                ),
                (IResourceMetering.ResourceConfig)
            ),
            startingAnchorRoot: abi.decode(
                _loadBytes(
                    address(_chainContracts.anchorStateRegistry),
                    _chainContracts.anchorStateRegistry.getAnchorRoot.selector,
                    "overrides.cfg.startingAnchorRoot",
                    _upgradeInput.extraInstructions
                ),
                (Proposal)
            ),
            startingRespectedGameType: abi.decode(
                _loadBytes(
                    address(_chainContracts.anchorStateRegistry),
                    _chainContracts.anchorStateRegistry.respectedGameType.selector,
                    "overrides.cfg.startingRespectedGameType",
                    _upgradeInput.extraInstructions
                ),
                (GameType)
            ),
            disputeGameConfigs: _upgradeInput.disputeGameConfigs
        });
    }

    /// @notice Validates the deployment/upgrade config.
    /// @param _cfg The full config.
    function _assertValidFullConfig(FullConfig memory _cfg) internal pure {
        // Start validating the dispute game configs. Put allowed game types here.
        GameType[] memory validGameTypes = new GameType[](3);
        validGameTypes[0] = GameTypes.CANNON;
        validGameTypes[1] = GameTypes.PERMISSIONED_CANNON;
        validGameTypes[2] = GameTypes.CANNON_KONA;

        // We must have a config for each valid game type.
        if (_cfg.disputeGameConfigs.length != validGameTypes.length) {
            revert OPContractsManagerV2_InvalidGameConfigs();
        }

        // Simplest possible check, iterate over each provided config and confirm that it matches
        // the game type array. This places a requirement on the user to order the configs properly
        // but that's probably a good thing, keeps the config consistent.
        for (uint256 i = 0; i < _cfg.disputeGameConfigs.length; i++) {
            if (_cfg.disputeGameConfigs[i].gameType.raw() != validGameTypes[i].raw()) {
                revert OPContractsManagerV2_InvalidGameConfigs();
            }

            // If the game is disabled, we must have a 0 init bond.
            if (!_cfg.disputeGameConfigs[i].enabled && _cfg.disputeGameConfigs[i].initBond != 0) {
                revert OPContractsManagerV2_InvalidGameConfigs();
            }
        }

        // We currently REQUIRE that the PermissionedDisputeGame is enabled. We may be able to
        // remove this check at some point in the future if we stop making this assumption, but for
        // now we explicitly assert that it is enabled.
        if (!_cfg.disputeGameConfigs[1].enabled) {
            revert OPContractsManagerV2_InvalidGameConfigs();
        }
    }

    /// @notice Executes the deployment/upgrade action.
    /// @param _cfg The full config.
    /// @param _cts The chain contracts.
    /// @param _isInitialDeployment Whether or not this is an initial deployment.
    /// @return The chain contracts.
    function _apply(
        FullConfig memory _cfg,
        ChainContracts memory _cts,
        bool _isInitialDeployment
    )
        internal
        returns (ChainContracts memory)
    {
        // Validate the config.
        _assertValidFullConfig(_cfg);

        // Load the implementations.
        IOPContractsManagerContainer.Implementations memory impls = implementations();

        // Make sure the provided SuperchainConfig is up to date.
        if (SemverComp.lt(_cfg.superchainConfig.version(), ISuperchainConfig(impls.superchainConfigImpl).version())) {
            revert OPContractsManagerV2_SuperchainConfigNeedsUpgrade();
        }

        // Update the SystemConfig.
        // SystemConfig initializer is the only one large enough to require a separate function to
        // avoid stack-too-deep errors.
        _upgrade(
            _cts.proxyAdmin, address(_cts.systemConfig), impls.systemConfigImpl, _makeSystemConfigInitArgs(_cfg, _cts)
        );

        // Update the OptimismPortal.
        if (isDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP)) {
            _upgrade(
                _cts.proxyAdmin,
                address(_cts.optimismPortal),
                impls.optimismPortalInteropImpl,
                abi.encodeCall(
                    IOptimismPortalInterop.initialize, (_cts.systemConfig, _cts.anchorStateRegistry, _cts.ethLockbox)
                )
            );
        } else {
            _upgrade(
                _cts.proxyAdmin,
                address(_cts.optimismPortal),
                impls.optimismPortalImpl,
                abi.encodeCall(IOptimismPortal.initialize, (_cts.systemConfig, _cts.anchorStateRegistry))
            );
        }

        // NOTE: Same general pattern, we call _upgrade for each contract rather than
        // iterating over some sort of array because it's easier to implement and understand.

        // We upgrade/initialize the ETHLockbox if this is an initial deployment or if it's an
        // upgrade and the ETH_LOCKBOX feature is enabled.
        if (_isInitialDeployment || _cts.systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX)) {
            IOptimismPortal[] memory portals = new IOptimismPortal[](1);
            portals[0] = _cts.optimismPortal;
            _upgrade(
                _cts.proxyAdmin,
                address(_cts.ethLockbox),
                impls.ethLockboxImpl,
                abi.encodeCall(IETHLockbox.initialize, (_cts.systemConfig, portals))
            );
        }

        // If interop was requested, also set the ETHLockbox feature and migrate liquidity into the
        // ETHLockbox contract.
        if (isDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP)) {
            // If we haven't already enabled the ETHLockbox, enable it.
            if (!_cts.systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX)) {
                _cts.systemConfig.setFeature(Features.ETH_LOCKBOX, true);
            }

            // Migrate any ETH into the ETHLockbox.
            IOptimismPortalInterop(payable(_cts.optimismPortal)).migrateLiquidity();
        }

        // Update the L1CrossDomainMessenger.
        // NOTE: L1CrossDomainMessenger initializer is at slot 0, offset 20.
        _upgrade(
            _cts.proxyAdmin,
            address(_cts.l1CrossDomainMessenger),
            impls.l1CrossDomainMessengerImpl,
            abi.encodeCall(IL1CrossDomainMessenger.initialize, (_cts.systemConfig, _cts.optimismPortal)),
            bytes32(0),
            20
        );

        // Update the L1StandardBridge.
        _upgrade(
            _cts.proxyAdmin,
            address(_cts.l1StandardBridge),
            impls.l1StandardBridgeImpl,
            abi.encodeCall(IL1StandardBridge.initialize, (_cts.l1CrossDomainMessenger, _cts.systemConfig))
        );

        // Update the L1ERC721Bridge.
        _upgrade(
            _cts.proxyAdmin,
            address(_cts.l1ERC721Bridge),
            impls.l1ERC721BridgeImpl,
            abi.encodeCall(IL1ERC721Bridge.initialize, (_cts.l1CrossDomainMessenger, _cts.systemConfig))
        );

        // Update the OptimismMintableERC20Factory.
        _upgrade(
            _cts.proxyAdmin,
            address(_cts.optimismMintableERC20Factory),
            impls.optimismMintableERC20FactoryImpl,
            abi.encodeCall(IOptimismMintableERC20Factory.initialize, (address(_cts.l1StandardBridge)))
        );

        // Update the DisputeGameFactory.
        _upgrade(
            _cts.proxyAdmin,
            address(_cts.disputeGameFactory),
            impls.disputeGameFactoryImpl,
            abi.encodeCall(IDisputeGameFactory.initialize, (address(this)))
        );

        // Update the DelayedWETH.
        _upgrade(
            _cts.proxyAdmin,
            address(_cts.delayedWETH),
            impls.delayedWETHImpl,
            abi.encodeCall(IDelayedWETH.initialize, (_cts.systemConfig))
        );

        // Update the AnchorStateRegistry.
        _upgrade(
            _cts.proxyAdmin,
            address(_cts.anchorStateRegistry),
            impls.anchorStateRegistryImpl,
            abi.encodeCall(
                IAnchorStateRegistry.initialize,
                (_cts.systemConfig, _cts.disputeGameFactory, _cfg.startingAnchorRoot, _cfg.startingRespectedGameType)
            )
        );

        // Update the DisputeGame config and implementations.
        // NOTE: We assert in _assertValidFullConfig that we have a configuration for all valid game
        // types so we can be confident that we're setting/unsetting everything we care about.
        for (uint256 i = 0; i < _cfg.disputeGameConfigs.length; i++) {
            // Game implementation and arguments default to empty values. If the game is disabled,
            // we'll use these empty values to unset the game in the factory.
            IDisputeGame gameImpl = IDisputeGame(address(0));
            bytes memory gameArgs = bytes("");

            // If the game is enabled, grab the implementation and craft the game arguments.
            if (_cfg.disputeGameConfigs[i].enabled) {
                gameImpl = _getGameImpl(_cfg.disputeGameConfigs[i].gameType);
                gameArgs = _makeGameArgs(_cfg, _cts, _cfg.disputeGameConfigs[i]);
            }

            // Set the game implementation and arguments.
            // NOTE: If the game is disabled, we'll set the implementation to address(0) and the
            // arguments to bytes(""), disabling the game.
            _cts.disputeGameFactory.setImplementation(_cfg.disputeGameConfigs[i].gameType, gameImpl, gameArgs);
            _cts.disputeGameFactory.setInitBond(
                _cfg.disputeGameConfigs[i].gameType, _cfg.disputeGameConfigs[i].initBond
            );
        }

        // If critical transfer is allowed, tranfer ownership of the DisputeGameFactory and
        // ProxyAdmin to the PAO. During deployments, this means transferring ownership from the
        // OPCM contract to the target PAO. During upgrades, this would theoretically mean
        // transferring ownership from the existing PAO to itself, which would be a no-op. In an
        // abundance of caution to prevent accidental unexpected transfers of ownership, we use a
        // boolean flag to control whether this transfer is allowed which should ONLY be used for
        // the initial deployment and no other management/upgrade action.
        if (_isInitialDeployment) {
            // Transfer ownership of the DisputeGameFactory to the proxyAdminOwner.
            _cts.disputeGameFactory.transferOwnership(address(_cfg.proxyAdminOwner));

            // Transfer ownership of the ProxyAdmin to the proxyAdminOwner.
            _cts.proxyAdmin.transferOwnership(_cfg.proxyAdminOwner);
        }

        // Return contracts as the execution output.
        return _cts;
    }

    /// @notice Helper for making the SystemConfig initializer arguments. This is the only
    ///         initializer that needs a helper function because we get stack-too-deep.
    /// @param _cfg The full config.
    /// @param _cts The chain contracts.
    /// @return The SystemConfig initializer arguments.
    function _makeSystemConfigInitArgs(
        FullConfig memory _cfg,
        ChainContracts memory _cts
    )
        internal
        pure
        returns (bytes memory)
    {
        // Generate the SystemConfig addresses input.
        ISystemConfig.Addresses memory addrs = ISystemConfig.Addresses({
            l1CrossDomainMessenger: address(_cts.l1CrossDomainMessenger),
            l1ERC721Bridge: address(_cts.l1ERC721Bridge),
            l1StandardBridge: address(_cts.l1StandardBridge),
            optimismPortal: address(_cts.optimismPortal),
            optimismMintableERC20Factory: address(_cts.optimismMintableERC20Factory),
            delayedWETH: address(_cts.delayedWETH)
        });

        // Generate the initializer arguments.
        return abi.encodeCall(
            ISystemConfig.initialize,
            (
                _cfg.systemConfigOwner,
                _cfg.basefeeScalar,
                _cfg.blobBasefeeScalar,
                bytes32(uint256(uint160(_cfg.batcher))),
                _cfg.gasLimit,
                _cfg.unsafeBlockSigner,
                _cfg.resourceConfig,
                _chainIdToBatchInboxAddress(_cfg.l2ChainId),
                addrs,
                _cfg.l2ChainId,
                _cfg.superchainConfig
            )
        );
    }

    /// @notice Helper for retrieving dispute game implementations.
    /// @param _gameType The game type to retrieve the implementation for.
    /// @return The dispute game implementation.
    function _getGameImpl(GameType _gameType) internal view returns (IDisputeGame) {
        IOPContractsManagerContainer.Implementations memory impls = implementations();
        if (_gameType.raw() == GameTypes.CANNON.raw()) {
            return IDisputeGame(impls.faultDisputeGameV2Impl);
        } else if (_gameType.raw() == GameTypes.PERMISSIONED_CANNON.raw()) {
            return IDisputeGame(impls.permissionedDisputeGameV2Impl);
        } else if (_gameType.raw() == GameTypes.CANNON_KONA.raw()) {
            return IDisputeGame(impls.faultDisputeGameV2Impl);
        } else {
            // Since we assert in _assertValidFullConfig that we only have valid configs, this
            // should never happen, but we'll be defensive and revert if it does.
            revert OPContractsManagerV2_UnsupportedGameType();
        }
    }

    /// @notice Helper for creating game constructor arguments.
    /// @param _cfg Full chain config.
    /// @param _cts Chain contracts.
    /// @param _gcfg Configuration for the dispute game to create.
    /// @return The game constructor arguments.
    function _makeGameArgs(
        FullConfig memory _cfg,
        ChainContracts memory _cts,
        DisputeGameConfig memory _gcfg
    )
        internal
        view
        returns (bytes memory)
    {
        IOPContractsManagerContainer.Implementations memory impls = implementations();
        if (_gcfg.gameType.raw() == GameTypes.CANNON.raw() || _gcfg.gameType.raw() == GameTypes.CANNON_KONA.raw()) {
            FaultDisputeGameConfig memory parsedInputArgs = abi.decode(_gcfg.gameArgs, (FaultDisputeGameConfig));
            return abi.encodePacked(
                parsedInputArgs.absolutePrestate,
                impls.mipsImpl,
                address(_cts.anchorStateRegistry),
                address(_cts.delayedWETH),
                _cfg.l2ChainId
            );
        } else if (_gcfg.gameType.raw() == GameTypes.PERMISSIONED_CANNON.raw()) {
            PermissionedDisputeGameConfig memory parsedInputArgs =
                abi.decode(_gcfg.gameArgs, (PermissionedDisputeGameConfig));
            return abi.encodePacked(
                parsedInputArgs.absolutePrestate,
                impls.mipsImpl,
                address(_cts.anchorStateRegistry),
                address(_cts.delayedWETH),
                _cfg.l2ChainId,
                parsedInputArgs.proposer,
                parsedInputArgs.challenger
            );
        } else {
            // Since we assert in _assertValidFullConfig that we only have valid configs, this
            // should never happen, but we'll be defensive and revert if it does.
            revert OPContractsManagerV2_UnsupportedGameType();
        }
    }

    ///////////////////////////////////////////////////////////////////////////
    //                        PUBLIC UTILITY FUNCTIONS                       //
    ///////////////////////////////////////////////////////////////////////////

    /// @notice Returns the blueprint contract addresses.
    function blueprints() public view returns (IOPContractsManagerContainer.Blueprints memory) {
        return contractsContainer.blueprints();
    }

    /// @notice Returns the implementation contract addresses.
    function implementations() public view returns (IOPContractsManagerContainer.Implementations memory) {
        return contractsContainer.implementations();
    }

    /// @notice Returns the status of a development feature.
    /// @param _feature The feature to check.
    /// @return True if the feature is enabled, false otherwise.
    function isDevFeatureEnabled(bytes32 _feature) public view returns (bool) {
        return contractsContainer.isDevFeatureEnabled(_feature);
    }

    ///////////////////////////////////////////////////////////////////////////
    //                       INTERNAL UTILITY FUNCTIONS                      //
    ///////////////////////////////////////////////////////////////////////////

    /// @notice Maps an L2 chain ID to an L1 batch inbox address as defined by the standard
    ///         configuration's convention. This convention is
    ///         `versionByte || keccak256(bytes32(chainId))[:19]`, where || denotes concatenation,
    ///         versionByte is 0x00, and chainId is a uint256.
    ///         https://specs.optimism.io/protocol/configurability.html#consensus-parameters
    /// @param _l2ChainId The L2 chain ID to map to an L1 batch inbox address.
    /// @return Chain ID mapped to an L1 batch inbox address.
    function _chainIdToBatchInboxAddress(uint256 _l2ChainId) internal pure returns (address) {
        bytes1 versionByte = 0x00;
        bytes32 hashedChainId = keccak256(bytes.concat(bytes32(_l2ChainId)));
        bytes19 first19Bytes = bytes19(hashedChainId);
        return address(uint160(bytes20(bytes.concat(versionByte, first19Bytes))));
    }

    /// @notice Computes a unique salt for a contract deployment.
    /// @param _l2ChainId The L2 chain ID of the chain being deployed to.
    /// @param _saltMixer The salt mixer to use for the deployment.
    /// @param _contractName The name of the contract to deploy.
    /// @return The computed salt.
    function _computeSalt(
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

    /// @notice Helper function to check if a given instruction is present in a list of extra
    ///         upgrade instructions.
    /// @param _instructions The list of extra upgrade instructions.
    /// @param _key The key of the instruction to check for.
    /// @param _data The data of the instruction to check for.
    /// @return True if the instruction is present, false otherwise.
    function _hasInstruction(
        ExtraInstruction[] memory _instructions,
        string memory _key,
        bytes memory _data
    )
        internal
        pure
        returns (bool)
    {
        for (uint256 i = 0; i < _instructions.length; i++) {
            if (LibString.eq(_instructions[i].key, _key) && LibString.eq(string(_instructions[i].data), string(_data)))
            {
                return true;
            }
        }
        return false;
    }

    /// @notice Helper function to get an instruction by key.
    /// @param _instructions The list of extra upgrade instructions.
    /// @param _key The key of the instruction to get.
    /// @return The instruction, or an empty instruction if the instruction is not found.
    function _getInstructionByKey(
        ExtraInstruction[] memory _instructions,
        string memory _key
    )
        internal
        pure
        returns (ExtraInstruction memory)
    {
        for (uint256 i = 0; i < _instructions.length; i++) {
            if (LibString.eq(_instructions[i].key, _key)) {
                return _instructions[i];
            }
        }
        return ExtraInstruction({ key: "", data: bytes("") });
    }

    /// @notice Helper function to load data from a source contract as bytes.
    /// @param _source The source contract to load the data from.
    /// @param _selector The selector of the function to call on the source contract.
    /// @param _name The name of the field to load.
    /// @param _instructions The extra upgrade instructions for the data load.
    /// @return Data retrieved from the source contract.
    function _loadBytes(
        address _source,
        bytes4 _selector,
        string memory _name,
        ExtraInstruction[] memory _instructions
    )
        internal
        view
        returns (bytes memory)
    {
        // If an override exists for this load, return the override data.
        ExtraInstruction memory overrideInstruction = _getInstructionByKey(_instructions, _name);
        if (bytes(overrideInstruction.key).length > 0) {
            return overrideInstruction.data;
        }

        // Otherwise, load the data from the source contract.
        (bool success, bytes memory result) = address(_source).staticcall(abi.encodePacked(_selector));
        if (!success) {
            revert OPContractsManagerV2_ConfigLoadFailed(_name);
        }

        // Return the loaded data.
        return result;
    }

    /// @notice Attempts to load a proxy from a source function where the proxy should be found. If
    ///         the proxy isn't found at the source, or the call to the source fails, we build a
    ///         new proxy instead. Calls to source contracts MUST NOT fail under any circumstances
    ///         other than the function not existing (which can happen in an upgrade scenario).
    /// @param _source The source contract to load the proxy from.
    /// @param _selector The selector of the function to call on the source contract.
    /// @param _args The basic arguments for the proxy deployment.
    /// @param _contractName The name of the contract to deploy.
    /// @param _instructions The extra upgrade instructions for the proxy deployment.
    /// @return The address of the loaded or built proxy.
    function _loadOrDeployProxy(
        address _source,
        bytes4 _selector,
        ProxyDeployArgs memory _args,
        string memory _contractName,
        ExtraInstruction[] memory _instructions
    )
        internal
        returns (address payable)
    {
        // Loads are allowed to fail ONLY if the user explicitly permitted it (or if this is a
        // deployment and the "ALL" permission is set).
        bool loadCanFail = _hasInstruction(_instructions, PERMITTED_PROXY_DEPLOYMENT_KEY, bytes(_contractName))
            || _hasInstruction(_instructions, PERMITTED_PROXY_DEPLOYMENT_KEY, PERMIT_ALL_CONTRACTS_INSTRUCTION);

        // Try to load the proxy from the source.
        (bool success, bytes memory result) = address(_source).staticcall(abi.encodePacked(_selector));

        // If the load succeeded and the result is not a zero address, return the result.
        if (success && abi.decode(result, (address)) != address(0)) {
            return payable(abi.decode(result, (address)));
        } else if (!loadCanFail) {
            // Load not permitted to fail but did, revert.
            revert OPContractsManagerV2_ProxyMustLoad(_contractName);
        }

        // We've failed to load, but we allowed that failure.
        // Deploy the right proxy depending on the contract name.
        address ret;
        if (LibString.eq(_contractName, "L1StandardBridge")) {
            // L1StandardBridge is a special case ChugSplashProxy (legacy).
            ret = Blueprint.deployFrom(
                blueprints().l1ChugSplashProxy,
                _computeSalt(_args.l2ChainId, _args.saltMixer, "L1StandardBridge"),
                abi.encode(_args.proxyAdmin)
            );

            // ChugSplashProxy requires setting the proxy type on the ProxyAdmin.
            _args.proxyAdmin.setProxyType(ret, IProxyAdmin.ProxyType.CHUGSPLASH);
        } else if (LibString.eq(_contractName, "L1CrossDomainMessenger")) {
            // L1CrossDomainMessenger is a special case ResolvedDelegateProxy (legacy).
            string memory l1XdmName = "OVM_L1CrossDomainMessenger";
            ret = Blueprint.deployFrom(
                blueprints().resolvedDelegateProxy,
                _computeSalt(_args.l2ChainId, _args.saltMixer, "L1CrossDomainMessenger"),
                abi.encode(_args.addressManager, l1XdmName)
            );

            // ResolvedDelegateProxy requires setting the proxy type on the ProxyAdmin.
            _args.proxyAdmin.setProxyType(ret, IProxyAdmin.ProxyType.RESOLVED);
            _args.proxyAdmin.setImplementationName(ret, l1XdmName);
        } else {
            // Otherwise this is a normal proxy.
            ret = Blueprint.deployFrom(
                blueprints().proxy,
                _computeSalt(_args.l2ChainId, _args.saltMixer, _contractName),
                abi.encode(_args.proxyAdmin)
            );
        }

        // Emit the proxy creation event.
        emit ProxyCreation(_contractName, ret);

        // Return the final deployment result.
        return payable(ret);
    }

    /// @notice Upgrades a contract by resetting the initialized slot and calling the initializer.
    /// @param _proxyAdmin The proxy admin of the contract.
    /// @param _target The target of the contract.
    /// @param _implementation The implementation of the contract.
    /// @param _data The data to call the initializer with.
    function _upgrade(IProxyAdmin _proxyAdmin, address _target, address _implementation, bytes memory _data) internal {
        _upgrade(_proxyAdmin, _target, _implementation, _data, bytes32(0), 0);
    }

    /// @notice Upgrades a contract by resetting the initialized slot and calling the initializer.
    /// @param _proxyAdmin The proxy admin of the contract.
    /// @param _target The target of the contract.
    /// @param _implementation The implementation of the contract.
    /// @param _data The data to call the initializer with.
    /// @param _slot The slot where the initialized value is located.
    /// @param _offset The offset of the initializer value in the slot.
    function _upgrade(
        IProxyAdmin _proxyAdmin,
        address _target,
        address _implementation,
        bytes memory _data,
        bytes32 _slot,
        uint8 _offset
    )
        internal
    {
        // Check to make sure that we're not downgrading. Downgrades aren't inherently dangerous
        // but we also don't test for them so we don't really know if a specific downgrade will be
        // dangerous or not. It's easier to just revert instead.
        // NOTE: We DO allow upgrades to the same version, which makes it possible to use this
        //       function to both upgrade and then later perform management actions like changing
        //       the prestate for the fault dispute games.
        if (
            _proxyAdmin.getProxyImplementation(payable(_target)) != address(0)
                && SemverComp.gt(ISemver(_target).version(), ISemver(_implementation).version())
        ) {
            revert OPContractsManagerV2_DowngradeNotAllowed(address(_target));
        }

        // Upgrade to StorageSetter.
        _proxyAdmin.upgrade(payable(_target), address(implementations().storageSetterImpl));

        // Otherwise, we need to reset the initialized slot and call the initializer.
        // Reset the initialized slot by zeroing the single byte at `_offset` (from the right).
        bytes32 current = IStorageSetter(_target).getBytes32(_slot);
        uint256 mask = ~(uint256(0xff) << (uint256(_offset) * 8));
        IStorageSetter(_target).setBytes32(_slot, bytes32(uint256(current) & mask));

        // Upgrade to the implementation and call the initializer.
        _proxyAdmin.upgradeAndCall(payable(address(_target)), _implementation, _data);
    }
}
