// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { GameType, Proposal } from "src/dispute/lib/Types.sol";

// Interfaces
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOPContractsManagerContainer } from "interfaces/L1/opcm/IOPContractsManagerContainer.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";

interface IOPContractsManagerMigrator {
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

    /// @notice Thrown when a dispute game config has an init bond that its game type does not
    ///         allow: non-zero for SUPER_PERMISSIONED, which does not use bonds, or zero for any
    ///         other game type.
    error OPContractsManagerMigrator_InvalidInitBond();

    /// @notice Thrown when a permissionless fault game config has a zero absolute prestate.
    error OPContractsManagerMigrator_InvalidAbsolutePrestate();

    /// @notice Thrown when a dispute game config is for a game type that does not use super roots.
    error OPContractsManagerMigrator_InvalidGameType();

    /// @notice Thrown when a dispute game config is not enabled. Migration registers every config
    ///         it is given, so a disabled config would be registered anyway.
    error OPContractsManagerMigrator_DisputeGameNotEnabled();

    /// @notice Returns the container of blueprint and implementation contract addresses.
    function contractsContainer() external view returns (IOPContractsManagerContainer);

    /// @notice Returns the address of the OPContractsManagerUtils contract.
    function opcmUtils() external view returns (IOPContractsManagerUtils);

    /// @notice Migrates one or more OP Stack chains to use the Super Root dispute games and shared
    ///         dispute game contracts.
    /// @param _input The input parameters for the migration.
    function migrate(MigrateInput calldata _input) external;

    function __constructor__(IOPContractsManagerUtils _utils) external;
}
