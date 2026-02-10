// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { OPContractsManagerUtilsCaller } from "src/L1/opcm/OPContractsManagerUtilsCaller.sol";

// Libraries
import { GameTypes } from "src/dispute/lib/Types.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Features } from "src/libraries/Features.sol";

// Interfaces
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { IOptimismPortalInterop } from "interfaces/L1/IOptimismPortalInterop.sol";
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
    /// @param _input The input parameters for the migration.
    function migrate(MigrateInput calldata _input) public {
        // Check that the starting respected game type is a valid super game type.
        if (
            _input.startingRespectedGameType.raw() != GameTypes.SUPER_CANNON.raw()
                && _input.startingRespectedGameType.raw() != GameTypes.SUPER_PERMISSIONED_CANNON.raw()
        ) {
            revert OPContractsManagerMigrator_InvalidStartingRespectedGameType();
        }

        // Check that all of the chains have the same core contracts.
        for (uint256 i = 0; i < _input.chainSystemConfigs.length; i++) {
            // Different chains might actually have different ProxyAdmin contracts, but it's fine
            // as long as the owner of all of those contracts is the same.
            if (_input.chainSystemConfigs[i].proxyAdmin().owner() != _input.chainSystemConfigs[0].proxyAdmin().owner())
            {
                revert OPContractsManagerMigrator_ProxyAdminOwnerMismatch();
            }

            // Each chain must have the same SuperchainConfig.
            if (_input.chainSystemConfigs[i].superchainConfig() != _input.chainSystemConfigs[0].superchainConfig()) {
                revert OPContractsManagerMigrator_SuperchainConfigMismatch();
            }
        }

        // NOTE: Interop doesn't have a real chain ID, and the chain ID provided here is ONLY used
        // as a salt mixer, so we just use the block.timestamp instead. It really doesn't matter
        // what we use here.
        IOPContractsManagerUtils.ProxyDeployArgs memory proxyDeployArgs = IOPContractsManagerUtils.ProxyDeployArgs({
            proxyAdmin: _input.chainSystemConfigs[0].proxyAdmin(),
            addressManager: IAddressManager(address(0)), // AddressManager NOT needed for these proxies.
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

        // Deploy the new DelayedWETH.
        IDelayedWETH delayedWETH = IDelayedWETH(
            _loadOrDeployProxy(
                address(0), // Source from address(0) so we always deploy a new proxy.
                bytes4(0),
                proxyDeployArgs,
                "DelayedWETH",
                extraInstructions
            )
        );

        // Separate context to avoid stack too deep (isolate the implementations variable).
        {
            // Grab the implementations.
            IOPContractsManagerContainer.Implementations memory impls = contractsContainer().implementations();

            // Initialize the new ETHLockbox.
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

            // Initialize the new DelayedWETH.
            _upgrade(
                proxyDeployArgs.proxyAdmin,
                address(delayedWETH),
                impls.delayedWETHImpl,
                abi.encodeCall(IDelayedWETH.initialize, (_input.chainSystemConfigs[0]))
            );

            // Migrate each portal to the new ETHLockbox and AnchorStateRegistry.
            for (uint256 i = 0; i < _input.chainSystemConfigs.length; i++) {
                _migratePortal(_input.chainSystemConfigs[i], ethLockbox, anchorStateRegistry);
            }
        }

        // Set up the dispute games in the new DisputeGameFactory.
        for (uint256 i = 0; i < _input.disputeGameConfigs.length; i++) {
            disputeGameFactory.setImplementation(
                _input.disputeGameConfigs[i].gameType,
                _getGameImpl(_input.disputeGameConfigs[i].gameType),
                _makeGameArgs(0, anchorStateRegistry, delayedWETH, _input.disputeGameConfigs[i])
            );
            disputeGameFactory.setInitBond(_input.disputeGameConfigs[i].gameType, _input.disputeGameConfigs[i].initBond);
        }
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
        // Convert portal to interop portal interface, and grab existing ETHLockbox and DGF.
        IOptimismPortalInterop portal = IOptimismPortalInterop(payable(_systemConfig.optimismPortal()));
        IETHLockbox existingLockbox = IETHLockbox(payable(address(portal.ethLockbox())));
        IDisputeGameFactory existingDGF = IDisputeGameFactory(payable(address(portal.disputeGameFactory())));

        // Authorize the portal on the new ETHLockbox.
        _newLockbox.authorizePortal(IOptimismPortal(payable(address(portal))));

        // Authorize the existing ETHLockbox to use the new ETHLockbox.
        _newLockbox.authorizeLockbox(existingLockbox);

        // Migrate the existing ETHLockbox to the new ETHLockbox.
        existingLockbox.migrateLiquidity(_newLockbox);

        // Clear out any implementations that might exist in the old DisputeGameFactory proxy.
        // We clear out all potential game types to be safe.
        existingDGF.setImplementation(GameTypes.CANNON, IDisputeGame(address(0)), hex"");
        existingDGF.setImplementation(GameTypes.SUPER_CANNON, IDisputeGame(address(0)), hex"");
        existingDGF.setImplementation(GameTypes.PERMISSIONED_CANNON, IDisputeGame(address(0)), hex"");
        existingDGF.setImplementation(GameTypes.SUPER_PERMISSIONED_CANNON, IDisputeGame(address(0)), hex"");
        existingDGF.setImplementation(GameTypes.CANNON_KONA, IDisputeGame(address(0)), hex"");
        existingDGF.setImplementation(GameTypes.SUPER_CANNON_KONA, IDisputeGame(address(0)), hex"");

        // Enable the ETH lockbox feature on the SystemConfig if not already enabled.
        // This is needed for the SystemConfig's paused() function to use the correct identifier.
        if (!_systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX)) {
            _systemConfig.setFeature(Features.ETH_LOCKBOX, true);
        }

        // Migrate the portal to the new ETHLockbox and AnchorStateRegistry.
        // This also sets superRootsActive = true.
        // NOTE: This requires the portal to already be upgraded to the interop version
        // (OptimismPortalInterop). If the portal is not on the interop version, this call will
        // fail.
        portal.migrateToSuperRoots(_newLockbox, _newASR);
    }

    /// @notice Returns the contracts container.
    /// @return The contracts container.
    function contractsContainer() public view returns (IOPContractsManagerContainer) {
        return opcmUtils.contractsContainer();
    }
}
