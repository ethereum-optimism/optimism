// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { ISemver } from "src/universal/ISemver.sol";

import "src/libraries/DisputeTypes.sol";

/// @title DeputyGuardianModule
/// @notice This module is intended to be enabled on the Security Council Safe, which will own the Guardian role in the
///         SuperchainConfig contract. The DeputyGuardianModule should allow a Deputy Guardian to administer any of the
///         actions that the Guardian is authorized to take. The security council can revoke the Deputy Guardian's
///         authorization at any time by disabling this module.
contract DeputyGuardianModule is ISemver {
    /// @notice The Safe contract instance
    Safe internal immutable SAFE;

    /// @notice The SuperchainConfig's address
    SuperchainConfig internal immutable SUPERCHAIN_CONFIG;

    /// @notice The deputy guardian's address
    address internal immutable DEPUTY_GUARDIAN;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    // Constructor to initialize the Safe and baseModule instances
    constructor(Safe _safe, SuperchainConfig _superchainConfig, address _deputyGuardian) {
        SAFE = _safe;
        SUPERCHAIN_CONFIG = _superchainConfig;
        DEPUTY_GUARDIAN = _deputyGuardian;
    }

    /// @notice Getter function for the Safe contract instance
    /// @return safe_ The Safe contract instance
    function safe() public view returns (Safe safe_) {
        safe_ = SAFE;
    }

    /// @notice Getter function for the SuperchainConfig's address
    /// @return superchainConfig_ The SuperchainConfig's address
    function superchainConfig() public view returns (SuperchainConfig superchainConfig_) {
        superchainConfig_ = SUPERCHAIN_CONFIG;
    }

    /// @notice Getter function for the deputy guardian's address
    /// @return deputyGuardian_ The deputy guardian's address
    function deputyGuardian() public view returns (address deputyGuardian_) {
        deputyGuardian_ = DEPUTY_GUARDIAN;
    }

    /// @notice Calls to the Security Council's `execTransactionFromModule()`, with the arguments
    ///      necessary to call `pause()` on the `SuperchainConfig` contract.
    ///      Only the deputy guardian can call this function.
    function pause() external {
        require(msg.sender == DEPUTY_GUARDIAN, "DeputyGuardianModule: Only the deputy guardian can pause.");
        bytes memory data = abi.encodeWithSelector(SUPERCHAIN_CONFIG.pause.selector, "");

        (bool success, bytes memory returnData) =
            SAFE.execTransactionFromModuleReturnData(address(SUPERCHAIN_CONFIG), 0, data, Enum.Operation.Call);
        require(success, string(returnData));
    }

    /// @notice Calls to the Security Council's `execTransactionFromModule()`, with the arguments
    ///      necessary to call `unpause()` on the `SuperchainConfig` contract.
    ///      Only the deputy guardian can call this function.
    function unpause() external {
        require(msg.sender == DEPUTY_GUARDIAN, "DeputyGuardianModule: Only the deputy guardian can unpause.");
        bytes memory data = abi.encodeWithSelector(SUPERCHAIN_CONFIG.unpause.selector);

        (bool success, bytes memory returnData) =
            SAFE.execTransactionFromModuleReturnData(address(SUPERCHAIN_CONFIG), 0, data, Enum.Operation.Call);
        require(success, string(returnData));
    }

    /// @notice When called, this function will call to the Security Council's `execTransactionFromModule()`
    ///      with the arguments necessary to call `blacklistDisputeGame()` on the `OptimismPortal2` contract.
    ///      Only the deputy guardian can call this function.
    /// @param _portal The `OptimismPortal2` contract instance.
    /// @param _game The `IDisputeGame` contract instance.
    function blacklistDisputeGame(OptimismPortal2 _portal, IDisputeGame _game) external {
        require(
            msg.sender == DEPUTY_GUARDIAN, "DeputyGuardianModule: Only the deputy guardian can blacklist dispute games."
        );
        bytes memory data = abi.encodeWithSelector(OptimismPortal2.blacklistDisputeGame.selector, address(_game));

        (bool success, bytes memory returnData) =
            SAFE.execTransactionFromModuleReturnData(address(_portal), 0, data, Enum.Operation.Call);
        require(success, string(returnData));
    }

    /// @notice When called, this function will call to the Security Council's `execTransactionFromModule()`
    ///      with the arguments necessary to call `setRespectedGameType()` on the `OptimismPortal2` contract.
    ///      Only the deputy guardian can call this function.
    /// @param _portal The `OptimismPortal2` contract instance.
    /// @param _gameType The `GameType` to set as the respected game type.
    function setRespectedGameType(OptimismPortal2 _portal, GameType _gameType) external {
        require(
            msg.sender == DEPUTY_GUARDIAN,
            "DeputyGuardianModule: Only the deputy guardian can set the respected game type."
        );
        bytes memory data = abi.encodeWithSelector(OptimismPortal2.setRespectedGameType.selector, _gameType);
        (bool success, bytes memory returnData) =
            SAFE.execTransactionFromModuleReturnData(address(_portal), 0, data, Enum.Operation.Call);
        require(success, string(returnData));
    }
}
