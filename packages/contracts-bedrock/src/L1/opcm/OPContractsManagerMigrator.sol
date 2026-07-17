// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { OPContractsManagerUtilsCaller } from "src/L1/opcm/OPContractsManagerUtilsCaller.sol";

// Libraries
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { GameTypes } from "src/dispute/lib/Types.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Features } from "src/libraries/Features.sol";

// Interfaces
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerContainer.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { GameType, Proposal } from "src/dispute/lib/Types.sol";

/// @title OPContractsManagerMigrator
/// @notice OPContractsManagerMigrator is a contract that provides the migration functionality for
///         migrating one or more OP Stack chains to use the Super Root dispute games and shared
///         dispute game contracts.
contract OPContractsManagerMigrator is OPContractsManagerUtilsCaller {
    /// @notice Input for migrating one or more OP Stack chains to use the Super Root dispute games
    ///         and shared dispute game contracts.
    struct MigrateInput {
        ISystemConfig[] chainSystemConfigs;
        IOPContractsManagerUtils.DisputeGameConfig[] disputeGameConfigs;
        Proposal startingAnchorRoot;
        GameType startingRespectedGameType;
    }

    /// @notice Thrown when a chain's ProxyAdmin owner does not match the other chains.
    error OPContractsManagerMigrator_ProxyAdminOwnerMismatch();

    /// @notice Thrown when a chain's SuperchainConfig does not match the other chains.
    error OPContractsManagerMigrator_SuperchainConfigMismatch();

    /// @notice Thrown when the starting respected game type is not a valid super game type.
    error OPContractsManagerMigrator_InvalidStartingRespectedGameType();

    /// @notice Thrown when attempting to migrate a CGT chain.
    error OPContractsManagerMigrator_CustomGasTokenNotSupported();

    /// @notice Thrown when the chainSystemConfigs array is empty.
    error OPContractsManagerMigrator_NoChains();

    /// @notice Thrown when the OPTIMISM_PORTAL_INTEROP dev feature is not enabled.
    error OPContractsManagerMigrator_InteropNotEnabled();

    /// @notice Thrown when a chain is paused before migration mutates its portal.
    error OPContractsManagerMigrator_SystemPaused();

    /// @notice Thrown when a chain's SystemConfig reports an l2ChainId of zero.
    error OPContractsManagerMigrator_ZeroL2ChainId();

    /// @notice Thrown when two chains share the same l2ChainId.
    error OPContractsManagerMigrator_DuplicateL2ChainId();

    /// @notice Thrown when chainSystemConfigs are not provided in ascending order by l2ChainId.
    error OPContractsManagerMigrator_ChainIdsNotAscending();

    /// @notice Thrown when the ZK_DISPUTE_GAME dev feature is not enabled.
    error OPContractsManagerMigrator_ZKDisputeGameNotEnabled();

    /// @notice Thrown when the supplied chains do not all share the same AnchorStateRegistry, i.e.
    ///         they are not a single already-interop set.
    error OPContractsManagerMigrator_NotSharedInteropSet();

    /// @notice Thrown when the new respected game type does not resolve to a registered
    ///         implementation on the shared DisputeGameFactory after the dispute games are swapped.
    error OPContractsManagerMigrator_RespectedGameTypeNotRegistered();

    /// @param _utils The utility functions for the OPContractsManager.
    constructor(IOPContractsManagerUtils _utils) OPContractsManagerUtilsCaller(_utils) { }

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
    /// @dev NOTE: This function is designed exclusively for the case of N independent pre-interop
    ///      chains merging into a single interop set. It does NOT support partial migration (i.e.,
    ///      migrating a subset of chains that share a lockbox), re-migration of already-migrated
    ///      chains, or any other migration scenario. Re-calling this function on already-migrated
    ///      portals will corrupt the shared DisputeGameFactory used by all migrated chains.
    /// @dev NOTE: Unlike deploy/upgrade, this function does not enforce a SuperchainConfig
    ///      version floor. The caller is responsible for ensuring the SuperchainConfig is
    ///      upgraded to the current OPCM release version before calling migrate.
    /// @dev NOTE: OPContractsManagerV2.upgrade() only performs standard chain upgrades. This
    ///      function performs the one-off interop activation by enabling required features,
    ///      connecting each portal to the shared ETHLockbox, migrating liquidity, and moving each
    ///      portal to the shared dispute game contracts.
    /// @param _input The input parameters for the migration.
    function migrate(MigrateInput calldata _input) public {
        // Check that at least one chain is being migrated.
        if (_input.chainSystemConfigs.length == 0) {
            revert OPContractsManagerMigrator_NoChains();
        }

        // Check that the OPTIMISM_PORTAL_INTEROP dev feature is enabled.
        if (!contractsContainer().isDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP)) {
            revert OPContractsManagerMigrator_InteropNotEnabled();
        }

        // Check that the starting respected game type is a valid super game type.
        // SUPER_CANNON is retired in favor of SUPER_CANNON_KONA.
        if (
            _input.startingRespectedGameType.raw() != GameTypes.SUPER_CANNON_KONA.raw()
                && _input.startingRespectedGameType.raw() != GameTypes.SUPER_PERMISSIONED.raw()
        ) {
            revert OPContractsManagerMigrator_InvalidStartingRespectedGameType();
        }

        // Check that all of the chains have the same core contracts, that no chain reports a
        // zero l2ChainId, that no two chains share the same l2ChainId, and that l2ChainIds are
        // provided in ascending order.
        _validateChainSystemConfigs(_input.chainSystemConfigs);

        // NOTE: Interop doesn't have a real chain ID, and the chain ID provided here is ONLY used
        // as a salt mixer, so we just use the block.timestamp instead. It really doesn't matter
        // what we use here.
        IOPContractsManagerUtils.ProxyDeployArgs memory proxyDeployArgs = IOPContractsManagerUtils.ProxyDeployArgs({
            proxyAdmin: _input.chainSystemConfigs[0].proxyAdmin(),
            addressManager: _input.chainSystemConfigs[0].proxyAdmin().addressManager(),
            l2ChainId: block.timestamp,
            saltMixer: "interop salt mixer"
        });

        // Set up the extra instructions to allow all proxy deployments.
        IOPContractsManagerUtils.ExtraInstruction[] memory extraInstructions =
            new IOPContractsManagerUtils.ExtraInstruction[](1);
        extraInstructions[0] = IOPContractsManagerUtils.ExtraInstruction({
            key: Constants.PERMITTED_PROXY_DEPLOYMENT_KEY,
            data: bytes(Constants.PERMIT_ALL_CONTRACTS_INSTRUCTION)
        });

        // Deploy the new ETHLockbox.
        IETHLockbox ethLockbox = IETHLockbox(
            _loadOrDeployProxy(
                address(0), // Source from address(0) so we always deploy a new proxy.
                bytes4(0),
                proxyDeployArgs,
                "ETHLockbox",
                extraInstructions
            )
        );

        // Deploy the new DisputeGameFactory.
        IDisputeGameFactory disputeGameFactory = IDisputeGameFactory(
            _loadOrDeployProxy(
                address(0), // Source from address(0) so we always deploy a new proxy.
                bytes4(0),
                proxyDeployArgs,
                "DisputeGameFactory",
                extraInstructions
            )
        );

        // Deploy the new AnchorStateRegistry.
        IAnchorStateRegistry anchorStateRegistry = IAnchorStateRegistry(
            _loadOrDeployProxy(
                address(0), // Source from address(0) so we always deploy a new proxy.
                bytes4(0),
                proxyDeployArgs,
                "AnchorStateRegistry",
                extraInstructions
            )
        );

        // Reuse the existing DelayedWETH from chainSystemConfigs[0] rather than deploying a
        // new one. The migrated chains share a SystemConfig, and by extension share its
        // DelayedWETH. Deploying a new one would create a divergence — SystemConfig would
        // still point to the old one, and future upgrades (which load DelayedWETH from
        // SystemConfig) would reference a different DelayedWETH than the shared DGF games.
        IDelayedWETH delayedWETH = IDelayedWETH(payable(_input.chainSystemConfigs[0].delayedWETH()));

        // Separate context to avoid stack too deep (isolate the implementations variable).
        {
            // Grab the implementations.
            IOPContractsManagerContainer.Implementations memory impls = contractsContainer().implementations();

            // Initialize the new ETHLockbox.
            // NOTE: Shared contracts (ETHLockbox, AnchorStateRegistry, DelayedWETH) are
            // intentionally initialized with chainSystemConfigs[0]. All chains are validated to
            // share the same ProxyAdmin owner and SuperchainConfig, so the first chain's
            // SystemConfig is used as the canonical governance reference for shared contracts.
            _upgrade(
                proxyDeployArgs.proxyAdmin,
                address(ethLockbox),
                impls.ethLockboxImpl,
                abi.encodeCall(IETHLockbox.initialize, (_input.chainSystemConfigs[0], new IOptimismPortal[](0)))
            );

            // Initialize the new DisputeGameFactory.
            _upgrade(
                proxyDeployArgs.proxyAdmin,
                address(disputeGameFactory),
                impls.disputeGameFactoryImpl,
                abi.encodeCall(IDisputeGameFactory.initialize, (proxyDeployArgs.proxyAdmin.owner()))
            );

            // Initialize the new AnchorStateRegistry.
            _upgrade(
                proxyDeployArgs.proxyAdmin,
                address(anchorStateRegistry),
                impls.anchorStateRegistryImpl,
                abi.encodeCall(
                    IAnchorStateRegistry.initialize,
                    (
                        _input.chainSystemConfigs[0],
                        disputeGameFactory,
                        _input.startingAnchorRoot,
                        _input.startingRespectedGameType
                    )
                )
            );

            // Migrate each portal to the new ETHLockbox and AnchorStateRegistry.
            for (uint256 i = 0; i < _input.chainSystemConfigs.length; i++) {
                _updateSystemConfigDelayedWETH(_input.chainSystemConfigs[i], delayedWETH, impls.systemConfigImpl);
                _migratePortal(_input.chainSystemConfigs[i], ethLockbox, anchorStateRegistry);
            }
        }

        // Set up the dispute games in the new DisputeGameFactory.
        // NOTE: Unlike deploy/upgrade, migration does not perform full game config
        // validation. This is intentional:
        // 1. Migration is a privileged, one-off admin action by the ProxyAdmin owner
        // 2. getGameImpl() rejects unrecognized game types
        // 3. Only super game types are meaningful here — non-super types would have
        //    l2ChainId=0, causing FaultDisputeGame to revert on chain ID mismatch
        // 4. All supplied configs are registered regardless of the enabled flag —
        //    callers must only include configs they want active
        for (uint256 i = 0; i < _input.disputeGameConfigs.length; i++) {
            disputeGameFactory.setImplementation(
                _input.disputeGameConfigs[i].gameType,
                _getGameImpl(_input.disputeGameConfigs[i].gameType),
                _makeGameArgs(0, anchorStateRegistry, delayedWETH, _input.disputeGameConfigs[i])
            );
            disputeGameFactory.setInitBond(_input.disputeGameConfigs[i].gameType, _input.disputeGameConfigs[i].initBond);
        }
    }

    /// @notice Re-points the shared dispute games of an ALREADY-interop set to a new respected
    ///         super game, running shared-infra steps exactly once. This is general across super
    ///         game types: the caller chooses the source game(s) to retire and the target game to
    ///         enable via `disputeGameConfigs`, and the new respected type via
    ///         `startingRespectedGameType`. Unlike migrate(), it does NOT deploy new shared infra
    ///         and does NOT touch per-chain portals: the chains already share an
    ///         AnchorStateRegistry, DisputeGameFactory and ETHLockbox. It re-initializes the shared
    ///         AnchorStateRegistry with the new respected game type and anchor root, then sets the
    ///         dispute game implementations on the shared DisputeGameFactory.
    /// @dev NOTE: Gated on OPTIMISM_PORTAL_INTEROP. It is additionally gated on the ZK_DISPUTE_GAME
    ///      dev feature while ZK is the in-development target super game.
    /// @dev NOTE: In-flight games can keep finalizing since wasRespectedGameTypeWhenCreated remains
    ///      true for the retired game.
    /// @dev NOTE: The shared AnchorStateRegistry is re-initialized, which re-seeds the anchor root.
    ///      The caller MUST supply an honest anchor root in `startingAnchorRoot`; if it differs
    ///      from the current one, its l2SequenceNumber MUST be strictly greater than the current
    ///      anchor's or AnchorStateRegistry.initialize reverts (same constraint as migrate()).
    /// @dev NOTE: Unlike migrate(), this does NOT invalidate already-proven withdrawals: the
    ///      AnchorStateRegistry retirement timestamp is preserved. Retired games can therefore still
    ///      advance the anchor unless they are blacklisted separately.
    /// @param _input The input parameters. `chainSystemConfigs` is the existing interop set;
    ///        `disputeGameConfigs` enables the target game and disables the retired one;
    ///        `startingRespectedGameType` is the new (super) respected game type.
    function setInteropDisputeGames(MigrateInput calldata _input) public {
        // Check that at least one chain is supplied.
        if (_input.chainSystemConfigs.length == 0) {
            revert OPContractsManagerMigrator_NoChains();
        }

        if (!contractsContainer().isDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP)) {
            revert OPContractsManagerMigrator_InteropNotEnabled();
        }
        // TODO(#21529): remove this ZK_DISPUTE_GAME gate once the ZK game becomes the default super game post-interop.
        if (!contractsContainer().isDevFeatureEnabled(DevFeatures.ZK_DISPUTE_GAME)) {
            revert OPContractsManagerMigrator_ZKDisputeGameNotEnabled();
        }

        // The new respected game type must be a super game type.
        if (!GameTypes.isSuperGame(_input.startingRespectedGameType)) {
            revert OPContractsManagerMigrator_InvalidStartingRespectedGameType();
        }

        // All chains must share the same core contracts. Validate the set, then resolve the shared
        // infra from the first chain as the canonical reference (same convention as migrate()).
        _validateChainSystemConfigs(_input.chainSystemConfigs);
        ISystemConfig sysCfg = _input.chainSystemConfigs[0];

        // Resolve the existing shared infra.
        IAnchorStateRegistry anchorStateRegistry =
            IOptimismPortal(payable(sysCfg.optimismPortal())).anchorStateRegistry();
        IDisputeGameFactory disputeGameFactory = anchorStateRegistry.disputeGameFactory();
        IDelayedWETH delayedWETH = IDelayedWETH(payable(sysCfg.delayedWETH()));

        // Assert every chain is actually part of this interop set: each portal must resolve to the
        // same shared AnchorStateRegistry. Starts at 0 to mirror _validateChainSystemConfigs
        for (uint256 i = 0; i < _input.chainSystemConfigs.length; i++) {
            IOptimismPortal portal = IOptimismPortal(payable(_input.chainSystemConfigs[i].optimismPortal()));
            if (portal.anchorStateRegistry() != anchorStateRegistry) {
                revert OPContractsManagerMigrator_NotSharedInteropSet();
            }
        }

        IOPContractsManagerContainer.Implementations memory impls = contractsContainer().implementations();

        // Re-initialize the shared AnchorStateRegistry with the new respected game type and anchor
        // root. _upgrade resets the initialized slot, re-seeding the anchor for the new game.
        _upgrade(
            sysCfg.proxyAdmin(),
            address(anchorStateRegistry),
            impls.anchorStateRegistryImpl,
            abi.encodeCall(
                IAnchorStateRegistry.initialize,
                (sysCfg, disputeGameFactory, _input.startingAnchorRoot, _input.startingRespectedGameType)
            )
        );

        // Swap the dispute games on the shared DisputeGameFactory. An enabled config registers the
        // new game. A disabled config clears the retired one by setting the implementation to address(0).
        for (uint256 i = 0; i < _input.disputeGameConfigs.length; i++) {
            IDisputeGame gameImpl = IDisputeGame(address(0));
            bytes memory gameArgs = bytes("");
            if (_input.disputeGameConfigs[i].enabled) {
                gameImpl = _getGameImpl(_input.disputeGameConfigs[i].gameType);
                gameArgs = _makeGameArgs(0, anchorStateRegistry, delayedWETH, _input.disputeGameConfigs[i]);
            }
            disputeGameFactory.setImplementation(_input.disputeGameConfigs[i].gameType, gameImpl, gameArgs);
            disputeGameFactory.setInitBond(_input.disputeGameConfigs[i].gameType, _input.disputeGameConfigs[i].initBond);
        }

        // The new respected game type must resolve to a registered implementation after the swap.
        if (address(disputeGameFactory.gameImpls(_input.startingRespectedGameType)) == address(0)) {
            revert OPContractsManagerMigrator_RespectedGameTypeNotRegistered();
        }
    }

    /// @notice Validates the per-chain SystemConfig array supplied to migrate(). All chains must
    ///         share a ProxyAdmin owner and SuperchainConfig, and their l2ChainIds must be
    ///         non-zero, distinct, and provided in ascending order.
    /// @param _chainSystemConfigs The chain system configs to validate.
    function _validateChainSystemConfigs(ISystemConfig[] calldata _chainSystemConfigs) internal view {
        uint256 prevL2ChainId;
        for (uint256 i = 0; i < _chainSystemConfigs.length; i++) {
            // Different chains might actually have different ProxyAdmin contracts, but it's fine
            // as long as the owner of all of those contracts is the same.
            if (_chainSystemConfigs[i].proxyAdmin().owner() != _chainSystemConfigs[0].proxyAdmin().owner()) {
                revert OPContractsManagerMigrator_ProxyAdminOwnerMismatch();
            }

            // Each chain must have the same SuperchainConfig.
            if (_chainSystemConfigs[i].superchainConfig() != _chainSystemConfigs[0].superchainConfig()) {
                revert OPContractsManagerMigrator_SuperchainConfigMismatch();
            }

            // The shared super-root dispute game system keys output roots by l2ChainId, so a
            // zero or duplicate l2ChainId across migrated portals would let the same withdrawal
            // be finalized through multiple portals.
            uint256 l2ChainId = _chainSystemConfigs[i].l2ChainId();
            if (i == 0) {
                if (l2ChainId == 0) {
                    revert OPContractsManagerMigrator_ZeroL2ChainId();
                }
            } else if (l2ChainId == prevL2ChainId) {
                revert OPContractsManagerMigrator_DuplicateL2ChainId();
            } else if (l2ChainId < prevL2ChainId) {
                revert OPContractsManagerMigrator_ChainIdsNotAscending();
            }
            prevL2ChainId = l2ChainId;
        }
    }

    /// @notice Updates a chain's SystemConfig to point at the shared DelayedWETH while preserving
    ///         all other per-chain configuration values.
    /// @param _systemConfig The system config for the chain being migrated.
    /// @param _delayedWETH The shared DelayedWETH to store in the SystemConfig.
    /// @param _systemConfigImpl The SystemConfig implementation to reinitialize with.
    function _updateSystemConfigDelayedWETH(
        ISystemConfig _systemConfig,
        IDelayedWETH _delayedWETH,
        address _systemConfigImpl
    )
        internal
    {
        ISystemConfig.Addresses memory addrs = _systemConfig.getAddresses();
        addrs.delayedWETH = address(_delayedWETH);
        addrs.opcm = _systemConfig.lastUsedOPCM();

        _upgrade(
            _systemConfig.proxyAdmin(),
            address(_systemConfig),
            _systemConfigImpl,
            _makeSystemConfigInitArgs(_systemConfig, addrs)
        );
    }

    /// @notice Builds SystemConfig initialize calldata from the chain's current values.
    /// @dev Kept separate from _updateSystemConfigDelayedWETH to avoid stack-too-deep errors.
    /// @param _systemConfig The system config to read existing values from.
    /// @param _addrs The L1 contract address set to write.
    /// @return Calldata for SystemConfig.initialize.
    function _makeSystemConfigInitArgs(
        ISystemConfig _systemConfig,
        ISystemConfig.Addresses memory _addrs
    )
        internal
        view
        returns (bytes memory)
    {
        return abi.encodeCall(
            ISystemConfig.initialize,
            (
                _systemConfig.owner(),
                _systemConfig.basefeeScalar(),
                _systemConfig.blobbasefeeScalar(),
                _systemConfig.batcherHash(),
                _systemConfig.gasLimit(),
                _systemConfig.unsafeBlockSigner(),
                _systemConfig.resourceConfig(),
                _systemConfig.batchInbox(),
                _addrs,
                _systemConfig.l2ChainId(),
                _systemConfig.superchainConfig()
            )
        );
    }

    /// @notice Migrates a single portal to the new ETHLockbox and AnchorStateRegistry.
    /// @param _systemConfig The system config for the chain being migrated.
    /// @param _newLockbox The new ETHLockbox.
    /// @param _newASR The new AnchorStateRegistry.
    function _migratePortal(
        ISystemConfig _systemConfig,
        IETHLockbox _newLockbox,
        IAnchorStateRegistry _newASR
    )
        internal
    {
        // CGT chains must not be migrated — prevents incorrect pooling into shared ETHLockbox.
        if (_systemConfig.isCustomGasToken()) {
            revert OPContractsManagerMigrator_CustomGasTokenNotSupported();
        }

        // Convert portal to interop portal interface, and grab existing migration state.
        IOptimismPortal portal = IOptimismPortal(payable(_systemConfig.optimismPortal()));
        IETHLockbox oldLockbox = IETHLockbox(payable(address(portal.ethLockbox())));
        IAnchorStateRegistry oldASR = portal.anchorStateRegistry();
        IDisputeGameFactory existingDGF = IDisputeGameFactory(payable(address(portal.disputeGameFactory())));

        // Check the current pause state before mutating the portal's lockbox. For chains that
        // already use a per-chain lockbox, SystemConfig.paused() keys the local pause against the
        // portal's current lockbox. Reinitializing the portal first would switch the pause
        // identifier to the shared lockbox and could hide an active old-lockbox pause.
        if (_systemConfig.paused()) {
            revert OPContractsManagerMigrator_SystemPaused();
        }

        // Authorize the portal on the new ETHLockbox.
        _newLockbox.authorizePortal(portal);

        // Enable the features required by portal liquidity migration and shared game migration.
        // ETH_LOCKBOX must be on so SystemConfig.paused() keys against the portal's lockbox; INTEROP
        // must be on for the post-migration cross-chain message paths. Both are idempotent.
        if (!_systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX)) {
            _systemConfig.setFeature(Features.ETH_LOCKBOX, true);
        }
        if (!_systemConfig.isFeatureEnabled(Features.INTEROP)) {
            _systemConfig.setFeature(Features.INTEROP, true);
        }

        // Attach the portal directly to the shared ETHLockbox before migrating portal-held ETH.
        _upgrade(
            _systemConfig.proxyAdmin(),
            address(portal),
            contractsContainer().implementations().optimismPortalImpl,
            abi.encodeCall(IOptimismPortal.initialize, (_systemConfig, oldASR, _newLockbox))
        );

        // Migrate ETH held directly by the portal into the shared ETHLockbox.
        portal.migrateLiquidity();

        // Sweep any pre-existing per-chain lockbox liquidity into the shared ETHLockbox. Fresh
        // chains may have no old lockbox, while already-lockbox-enabled chains can have ETH there
        // that would be stranded after the portal starts pointing at the shared lockbox.
        if (address(oldLockbox) != address(0) && address(oldLockbox) != address(_newLockbox)) {
            // The shared lockbox must authorize the old lockbox before receiveLiquidity() will
            // accept ETH from oldLockbox.migrateLiquidity().
            _newLockbox.authorizeLockbox(oldLockbox);
            oldLockbox.migrateLiquidity(_newLockbox);
        }

        // Clear out any implementations that might exist in the old DisputeGameFactory proxy.
        // We clear out all potential game types to be safe. These game types are intentionally
        // hardcoded rather than sourced from a shared utility. When new game types are added,
        // this list and the corresponding list in OPCMv2's _assertValidFullConfig must both
        // be updated.
        existingDGF.setImplementation(GameTypes.CANNON, IDisputeGame(address(0)), hex"");
        existingDGF.setImplementation(GameTypes.SUPER_CANNON, IDisputeGame(address(0)), hex"");
        existingDGF.setImplementation(GameTypes.PERMISSIONED_CANNON, IDisputeGame(address(0)), hex"");
        existingDGF.setImplementation(GameTypes.SUPER_PERMISSIONED, IDisputeGame(address(0)), hex"");
        existingDGF.setImplementation(GameTypes.CANNON_KONA, IDisputeGame(address(0)), hex"");
        existingDGF.setImplementation(GameTypes.SUPER_CANNON_KONA, IDisputeGame(address(0)), hex"");
        existingDGF.setImplementation(GameTypes.ZK_DISPUTE_GAME, IDisputeGame(address(0)), hex"");

        // Migrate the portal to the new ETHLockbox and AnchorStateRegistry.
        portal.migrateToSharedDisputeGame(_newLockbox, _newASR);
    }

    /// @notice Returns the contracts container.
    /// @return The contracts container.
    function contractsContainer() public view returns (IOPContractsManagerContainer) {
        return opcmUtils.contractsContainer();
    }
}
