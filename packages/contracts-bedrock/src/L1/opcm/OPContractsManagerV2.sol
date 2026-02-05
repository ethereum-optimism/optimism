// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { OPContractsManagerUtilsCaller } from "src/L1/opcm/OPContractsManagerUtilsCaller.sol";
import { IOPContractsManagerMigrator } from "interfaces/L1/opcm/IOPContractsManagerMigrator.sol";

// Libraries
import { Blueprint } from "src/libraries/Blueprint.sol";
import { GameType, GameTypes, Proposal } from "src/dispute/lib/Types.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Features } from "src/libraries/Features.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Constants } from "src/libraries/Constants.sol";

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
import { IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerContainer.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";

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
contract OPContractsManagerV2 is ISemver, OPContractsManagerUtilsCaller {
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
        IOPContractsManagerUtils.DisputeGameConfig[] disputeGameConfigs;
        // CGT
        bool useCustomGasToken;
    }

    /// @notice Partial input required for an upgrade.
    struct UpgradeInput {
        ISystemConfig systemConfig;
        IOPContractsManagerUtils.DisputeGameConfig[] disputeGameConfigs;
        IOPContractsManagerUtils.ExtraInstruction[] extraInstructions;
    }

    /// @notice Input for upgrading Superchain contracts.
    struct SuperchainUpgradeInput {
        ISuperchainConfig superchainConfig;
        IOPContractsManagerUtils.ExtraInstruction[] extraInstructions;
    }

    /// @notice Thrown when the SuperchainConfig needs to be upgraded.
    error OPContractsManagerV2_SuperchainConfigNeedsUpgrade();

    /// @notice Thrown when an invalid game config is provided.
    error OPContractsManagerV2_InvalidGameConfigs();

    /// @notice Thrown when an invalid upgrade input is provided.
    error OPContractsManagerV2_InvalidUpgradeInput();

    /// @notice Thrown when an invalid upgrade instruction is provided.
    error OPContractsManagerV2_InvalidUpgradeInstruction(string _key);

    /// @notice Thrown when a chain attempts to upgrade to custom gas token after initial deployment.
    error OPContractsManagerV2_CannotUpgradeToCustomGasToken();

    /// @notice Thrown when an invalid upgrade sequence is provided.
    error OPContractsManagerV2_InvalidUpgradeSequence(string _lastVersion, string _thisVersion);

    /// @notice Address of the Standard Validator for this OPCM release.
    IOPContractsManagerStandardValidator public immutable opcmStandardValidator;

    /// @notice Address of the Migrator contract for this OPCM release.
    IOPContractsManagerMigrator public immutable opcmMigrator;

    /// @notice Immutable reference to this OPCM contract so that the address of this contract can
    ///         be used when this contract is DELEGATECALLed.
    OPContractsManagerV2 public immutable opcmV2;

    /// @notice The version of the OPCM contract.
    ///         WARNING: OPCM versioning rules differ from other contracts:
    ///         - Major bump: New required sequential upgrade
    ///         - Minor bump: Replacement OPCM for same upgrade
    ///         - Patch bump: Development changes (expected for normal dev work)
    /// @custom:semver 7.0.8
    function version() public pure returns (string memory) {
        return "7.0.8";
    }

    /// @param _standardValidator The standard validator for this OPCM release.
    /// @param _migrator The migrator contract for this OPCM release.
    /// @param _utils The utility functions for the OPContractsManager.
    constructor(
        IOPContractsManagerStandardValidator _standardValidator,
        IOPContractsManagerMigrator _migrator,
        IOPContractsManagerUtils _utils
    )
        OPContractsManagerUtilsCaller(_utils)
    {
        opcmStandardValidator = _standardValidator;
        opcmMigrator = _migrator;
        opcmV2 = this;
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

        // Upgrade the SuperchainConfig.
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
        IOPContractsManagerUtils.ExtraInstruction[] memory instructions =
            new IOPContractsManagerUtils.ExtraInstruction[](1);
        instructions[0] = IOPContractsManagerUtils.ExtraInstruction({
            key: Constants.PERMITTED_PROXY_DEPLOYMENT_KEY,
            data: Constants.PERMIT_ALL_CONTRACTS_INSTRUCTION
        });

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
        // Sanity check that the SystemConfig isn't address(0). We use address(0) as a special
        // value to indicate that this is an initial deployment, so we definitely don't want to
        // allow it here.
        if (address(_inp.systemConfig) == address(0)) {
            revert OPContractsManagerV2_InvalidUpgradeInput();
        }

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

    /// @notice Migrates one or more OP Stack chains to use the Super Root dispute games and shared
    ///         dispute game contracts.
    /// @dev WARNING: This is a one-way operation. You cannot easily undo this operation without a
    ///      smart contract upgrade. Do not call this function unless you are 100% confident that
    ///      you know what you're doing and that you are prepared to fully execute this migration.
    ///      You SHOULD NOT CALL THIS FUNCTION IN PRODUCTION unless you are absolutely sure that
    ///      you know what you are doing.
    /// @dev WARNING: Executing this function WILL result in all prior withdrawal proofs being
    ///      invalidated. Users will have to submit new proofs for their withdrawals in the
    ///      OptimismPortal contract. THIS IS EXPECTED BEHAVIOR.
    /// @dev NOTE: Unlike other functions in OPCM, this is a one-off function used to serve the
    ///      temporary need to support the interop migration action. It will likely be removed in
    ///      the near future once interop support is baked more directly into OPCM. It does NOT
    ///      look or function like all of the other functions in OPCMv2.
    /// @param _input The input parameters for the migration.
    function migrate(IOPContractsManagerMigrator.MigrateInput calldata _input) public {
        // Delegatecall to the migrator contract.
        (bool success, bytes memory result) =
            address(opcmMigrator).delegatecall(abi.encodeCall(IOPContractsManagerMigrator.migrate, (_input)));
        if (!success) {
            assembly {
                revert(add(result, 0x20), mload(result))
            }
        }
    }

    ///////////////////////////////////////////////////////////////////////////
    //                  INTERNAL CHAIN MANAGEMENT FUNCTIONS                  //
    ///////////////////////////////////////////////////////////////////////////

    /// @notice Asserts that the upgrade instructions array is valid.
    /// @dev Developers don't need to touch this function, modify _isPermittedInstruction instead.
    /// @param _extraInstructions The extra upgrade instructions for the chain.
    function _assertValidUpgradeInstructions(IOPContractsManagerUtils.ExtraInstruction[] memory _extraInstructions)
        internal
        view
    {
        for (uint256 i = 0; i < _extraInstructions.length; i++) {
            if (!_isPermittedInstruction(_extraInstructions[i])) {
                revert OPContractsManagerV2_InvalidUpgradeInstruction(_extraInstructions[i].key);
            }
        }
    }

    /// @notice Checks if an upgrade instruction is permitted.
    /// @param _instruction The upgrade instruction to check.
    /// @return True if the instruction is permitted, false otherwise.
    function _isPermittedInstruction(IOPContractsManagerUtils.ExtraInstruction memory _instruction)
        internal
        view
        returns (bool)
    {
        // NOTE (IMPORTANT FOR DEVELOPERS): You MAY need to allow permitted instructions here for
        // your specific upgrade. For example, if you are adding a new contract that needs to be
        // deployed you will need to add an allowance so that the proxy can be deployed.
        // Allowances MUST always be restricted to one specific upgrade. Here we maintain this
        // restriction by checking that the version is less than the NEXT release version. Once
        // developers start working on the next release this will automatically become false so
        // even if the code is somehow forgotten it will not actually apply to the deployment. Make
        // sure to REMOVE the allowance once the upgrade is complete.
        if (SemverComp.lt(_version(), "8.0.0")) {
            // Unified DelayedWETH is being deployed for the first time.
            // TODO:(#18382): Remove this allowance after unified DelayedWETH is deployed.
            if (_isMatchingInstruction(_instruction, Constants.PERMITTED_PROXY_DEPLOYMENT_KEY, "DelayedWETH")) {
                return true;
            }
        }

        // Always return false by default.
        return false;
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
        IOPContractsManagerUtils.ExtraInstruction[] memory _extraInstructions
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
        IOPContractsManagerUtils.ProxyDeployArgs memory proxyDeployArgs = IOPContractsManagerUtils.ProxyDeployArgs({
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
            disputeGameConfigs: _upgradeInput.disputeGameConfigs,
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
                    _chainContracts.anchorStateRegistry.getStartingAnchorRoot.selector,
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
            useCustomGasToken: abi.decode(
                _loadBytes(
                    address(_chainContracts.systemConfig),
                    _chainContracts.systemConfig.isCustomGasToken.selector,
                    "overrides.cfg.useCustomGasToken",
                    _upgradeInput.extraInstructions
                ),
                (bool)
            )
        });
    }

    /// @notice Validates the deployment/upgrade config.
    /// @param _cfg The full config.
    function _assertValidFullConfig(FullConfig memory _cfg, bool _isInitialDeployment) internal pure {
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

            // During initial deployment, only PERMISSIONED_CANNON can be enabled, because no prestate exists for
            // permissionless games.
            if (
                _isInitialDeployment && (validGameTypes[i].raw() != GameTypes.PERMISSIONED_CANNON.raw())
                    && _cfg.disputeGameConfigs[i].enabled
            ) {
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
        _assertValidFullConfig(_cfg, _isInitialDeployment);

        // Load the implementations.
        IOPContractsManagerContainer.Implementations memory impls = implementations();

        // Make sure the provided SuperchainConfig is up to date.
        if (SemverComp.lt(_cfg.superchainConfig.version(), ISuperchainConfig(impls.superchainConfigImpl).version())) {
            revert OPContractsManagerV2_SuperchainConfigNeedsUpgrade();
        }

        // Chains prior to OPCMv2 don't yet have a functional lastUsedOPCMVersion function on the
        // SystemConfig contract. The first deployment of OPCMv2 will make this function available
        // and subsequent deployments will then use this function to verify the system is
        // progressing from one OPCM to the next. We only care about this for upgrades, you can
        // perform an initial deployment from any OPCM.
        if (!_isInitialDeployment && !isPermittedUpgradeSequence(_cts.systemConfig)) {
            revert OPContractsManagerV2_InvalidUpgradeSequence(_cts.systemConfig.lastUsedOPCMVersion(), _version());
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
                gameArgs = _makeGameArgs(
                    _cfg.l2ChainId, _cts.anchorStateRegistry, _cts.delayedWETH, _cfg.disputeGameConfigs[i]
                );
            }

            // Set the game implementation and arguments.
            // NOTE: If the game is disabled, we'll set the implementation to address(0) and the
            // arguments to bytes(""), disabling the game.
            _cts.disputeGameFactory.setImplementation(_cfg.disputeGameConfigs[i].gameType, gameImpl, gameArgs);
            _cts.disputeGameFactory.setInitBond(
                _cfg.disputeGameConfigs[i].gameType, _cfg.disputeGameConfigs[i].initBond
            );
        }

        // If the custom gas token feature was requested, enable it in the SystemConfig.
        // If the cgt is enabled, we skip this step.
        if (_cfg.useCustomGasToken && !_cts.systemConfig.isCustomGasToken()) {
            // NOTE: Enabling the custom gas token feature is only allowed during initial deployment to prevent
            // chains from enabling it during upgrades. Passing in true for this flag during an upgrade is considered an
            // error and will revert.
            // Revert only if trying to upgrade from CGT disabled to CGT enabled.
            if (!_isInitialDeployment) {
                revert OPContractsManagerV2_CannotUpgradeToCustomGasToken();
            }
            _cts.systemConfig.setFeature(Features.CUSTOM_GAS_TOKEN, true);
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
        view
        returns (bytes memory)
    {
        // Generate the SystemConfig addresses input.
        ISystemConfig.Addresses memory addrs = ISystemConfig.Addresses({
            l1CrossDomainMessenger: address(_cts.l1CrossDomainMessenger),
            l1ERC721Bridge: address(_cts.l1ERC721Bridge),
            l1StandardBridge: address(_cts.l1StandardBridge),
            optimismPortal: address(_cts.optimismPortal),
            optimismMintableERC20Factory: address(_cts.optimismMintableERC20Factory),
            delayedWETH: address(_cts.delayedWETH),
            opcm: address(opcmV2)
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

    ///////////////////////////////////////////////////////////////////////////
    //                        PUBLIC UTILITY FUNCTIONS                       //
    ///////////////////////////////////////////////////////////////////////////

    /// @notice Checks if the upgrade sequence from the last used OPCM to this OPCM is permitted.
    ///         This function is public to allow tests to verify upgrade sequence logic directly
    ///         by mocking the OPCM version and calling this function, rather than running the full
    ///         upgrade flow. This avoids the need for special testing flags that bypass validation.
    /// @param _systemConfig The SystemConfig contract to check the upgrade sequence for.
    /// @return True if the upgrade sequence is permitted, false otherwise.
    function isPermittedUpgradeSequence(ISystemConfig _systemConfig) public view returns (bool) {
        // If the SystemConfig is not initialized, this is an initial deployment, which is always
        // permitted. Initial deployments can use any OPCM version.
        if (address(_systemConfig) == address(0)) {
            return true;
        }

        // Chains prior to OPCMv2 (version 7.0.0) don't have a functional lastUsedOPCM function on
        // the SystemConfig contract. The first deployment of OPCMv2 which makes this available is
        // version 7.0.0. We need to skip the check for 7.x.x OPCM versions because they can't
        // guarantee that the lastUsedOPCM function will be available on the incoming SystemConfig.
        // 8.0.0 and later will always have this function available.
        if (SemverComp.lt(_version(), "8.0.0")) {
            return true;
        }

        ISemver lastUsedOPCM = ISemver(address(_systemConfig.lastUsedOPCM()));
        SemverComp.Semver memory lastUsedSemver = SemverComp.parse(lastUsedOPCM.version());
        SemverComp.Semver memory thisSemver = SemverComp.parse(_version());

        // We have three permitted cases:
        // 1. Address of the last used OPCM is identical to the address of this OPCM (re-running).
        // 2. This OPCM version is the same major version but a greater minor version (patch).
        // 3. This OPCM version is the next major version (sequential upgrade).
        bool isSameOPCM = address(lastUsedOPCM) == address(opcmV2);
        bool isNextMajor = thisSemver.major == lastUsedSemver.major + 1;
        bool isSameMajorHigherMinor =
            thisSemver.major == lastUsedSemver.major && thisSemver.minor > lastUsedSemver.minor;

        return isSameOPCM || isSameMajorHigherMinor || isNextMajor;
    }

    /// @notice Returns the blueprint contract addresses.
    function blueprints() public view returns (IOPContractsManagerContainer.Blueprints memory) {
        return contractsContainer().blueprints();
    }

    /// @notice Returns the implementation contract addresses.
    function implementations() public view returns (IOPContractsManagerContainer.Implementations memory) {
        return contractsContainer().implementations();
    }

    /// @notice Returns the status of a development feature.
    /// @param _feature The feature to check.
    /// @return True if the feature is enabled, false otherwise.
    function isDevFeatureEnabled(bytes32 _feature) public view returns (bool) {
        return contractsContainer().isDevFeatureEnabled(_feature);
    }

    ///////////////////////////////////////////////////////////////////////////
    //                       INTERNAL UTILITY FUNCTIONS                      //
    ///////////////////////////////////////////////////////////////////////////

    /// @notice Helper for retrieving the version of the OPCM contract.
    /// @dev We use opcmV2.version() because it allows us to properly mock the version function
    ///      in tests without running into issues because this contract is being DELEGATECALLed.
    /// @return The version of the OPCM contract.
    function _version() internal view returns (string memory) {
        return opcmV2.version();
    }

    /// @notice Returns the contracts container.
    /// @return The contracts container.
    function contractsContainer() public view returns (IOPContractsManagerContainer) {
        return opcmUtils.contractsContainer();
    }

    /// @notice Returns the development feature bitmap.
    /// @return The development feature bitmap.
    function devFeatureBitmap() public view returns (bytes32) {
        return contractsContainer().devFeatureBitmap();
    }
}
