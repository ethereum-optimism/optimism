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

    /// @notice Thrown when a chain's SystemConfig does not have Features.INTEROP enabled.
    error OPContractsManagerMigrator_InteropFeatureNotEnabled();

    /// @notice Thrown when a chain's SystemConfig does not have Features.ETH_LOCKBOX enabled.
    error OPContractsManagerMigrator_EthLockboxFeatureNotEnabled();

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
