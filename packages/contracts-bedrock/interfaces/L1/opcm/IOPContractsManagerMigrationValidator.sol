// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IStandardValidatorUtils } from "interfaces/L1/opcm/IStandardValidatorUtils.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

interface IOPContractsManagerMigrationValidator {
    error InvalidGameArgsLength();

    struct MigrationValidationInput {
        IDisputeGameFactory dgf;
        ISystemConfig[] chainSystemConfigs;
        bytes32 cannonPrestate;
        bytes32 cannonKonaPrestate;
        address proposer;
    }

    /// @notice Shared implementation addresses used when validating proxy → impl pairings.
    ///         Includes the StandardValidatorUtils helper used for super-game drill-down checks.
    struct SharedImplementations {
        address disputeGameFactoryImpl;
        address anchorStateRegistryImpl;
        address ethLockboxImpl;
        address delayedWETHImpl;
        address mipsImpl;
        address superFaultDisputeGameImpl;
        address superPermissionedDisputeGameImpl;
        IStandardValidatorUtils standardValidatorUtils;
    }

    /// @notice Shared roles and config values used during migration validation.
    struct SharedConfig {
        address l1PAOMultisig;
        address challenger;
        uint256 withdrawalDelaySeconds;
    }

    function validateMigration(
        MigrationValidationInput memory _input,
        bool _allowFailure,
        SharedImplementations memory _impls,
        SharedConfig memory _cfg
    )
        external
        view
        returns (string memory);

    function validateMigrationWithOverrides(
        MigrationValidationInput memory _input,
        bool _allowFailure,
        IOPContractsManagerStandardValidator.ValidationOverrides memory _overrides,
        SharedImplementations memory _impls,
        SharedConfig memory _cfg
    )
        external
        view
        returns (string memory);

    function __constructor__() external;
}
