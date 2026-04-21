// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

interface IOPContractsManagerMigrationValidator {
    error InvalidGameArgsLength();

    struct MigrationValidationInput {
        IDisputeGameFactory dgf;
        ISystemConfig[] chainSystemConfigs;
        bytes32 cannonPrestate;
        bytes32 cannonKonaPrestate;
        address proposer;
        address challenger;
    }

    struct SharedImplementations {
        address disputeGameFactoryImpl;
        address anchorStateRegistryImpl;
        address ethLockboxImpl;
        address delayedWETHImpl;
        address mipsImpl;
        address l1PAOMultisig;
        uint256 withdrawalDelaySeconds;
    }

    function validateMigration(
        MigrationValidationInput memory _input,
        bool _allowFailure,
        SharedImplementations memory _refs
    )
        external
        view
        returns (string memory);

    function validateMigrationWithOverrides(
        MigrationValidationInput memory _input,
        bool _allowFailure,
        IOPContractsManagerStandardValidator.ValidationOverrides memory _overrides,
        SharedImplementations memory _refs
    )
        external
        view
        returns (string memory);

    function __constructor__() external;
}
