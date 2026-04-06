// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title OPContractsManagerMigrationValidator
/// @notice Validates the configuration of L1 contracts after an interop migration. Separated from
///         OPContractsManagerStandardValidator due to EIP-170 contract size limits.
contract OPContractsManagerMigrationValidator is ISemver {
    /// @notice The semantic version of the OPContractsManagerMigrationValidator contract.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Struct containing the input parameters for post-migration validation.
    struct MigrationValidationInput {
        IDisputeGameFactory dgf;
        ISystemConfig[] chainSystemConfigs;
        bytes32 cannonPrestate;
        bytes32 cannonKonaPrestate;
        address proposer;
    }

    /// @notice Reference to the standard validator for shared state (impl addresses, etc.).
    IOPContractsManagerStandardValidator public standardValidator;

    /// @notice Constructor for the OPContractsManagerMigrationValidator contract.
    /// @param _standardValidator The standard validator to read shared configuration from.
    constructor(IOPContractsManagerStandardValidator _standardValidator) {
        standardValidator = _standardValidator;
    }

    /// @notice Validates the configuration of all L1 contracts after an interop migration.
    function validateMigration(
        MigrationValidationInput memory _input,
        bool _allowFailure
    )
        public
        view
        returns (string memory)
    {
        return validateMigrationWithOverrides(
            _input,
            _allowFailure,
            IOPContractsManagerStandardValidator.ValidationOverrides({
                l1PAOMultisig: address(0),
                challenger: address(0)
            })
        );
    }

    /// @notice Validates the configuration of all L1 contracts after an interop migration.
    ///         Supports overrides of certain storage values denoted in the ValidationOverrides struct.
    function validateMigrationWithOverrides(
        MigrationValidationInput memory _input,
        bool _allowFailure,
        IOPContractsManagerStandardValidator.ValidationOverrides memory _overrides
    )
        public
        view
        returns (string memory)
    {
        // TODO: Implement shared infra + per-chain validation.
        (_input, _allowFailure, _overrides);
        return "";
    }
}
