// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { Unauthorized } from "src/libraries/PortalErrors.sol";

import "src/libraries/DisputeTypes.sol";

/// @title DeputyGuardianModule
/// @notice This module is intended to be enabled on the Security Council Safe, which will own the Guardian role in the
///         SuperchainConfig contract. The DeputyGuardianModule should allow a Deputy Guardian to administer any of the
///         actions that the Guardian is authorized to take. The security council can revoke the Deputy Guardian's
///         authorization at any time by disabling this module.
contract DeputyGuardianModule is ISemver {
    /// @notice Error message for failed transaction execution
    error ExecutionFailed(string);

    /// @notice Emitted when the SuperchainConfig is paused
    event Paused(string identifier);

    /// @notice Emitted when the SuperchainConfig is unpaused
    event Unpaused();

    /// @notice Emitted when a DisputeGame is blacklisted
    event DisputeGameBlacklisted(IDisputeGame game);

    /// @notice Emitted when the respected game type is set
    event RespectedGameTypeSet(GameType gameType);

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

    /// @notice Internal function to ensure that only the deputy guardian can call certain functions.
    function _onlyDeputyGuardian() internal view {
        if (msg.sender != DEPUTY_GUARDIAN) {
            revert Unauthorized();
        }
    }

    /// @notice Calls the Security Council Safe's `execTransactionFromModuleReturnData()`, with the arguments
    ///      necessary to call `pause()` on the `SuperchainConfig` contract.
    ///      Only the deputy guardian can call this function.
    function pause() external {
        _onlyDeputyGuardian();

        bytes memory data = abi.encodeCall(SUPERCHAIN_CONFIG.pause, ("Deputy Guardian"));
        (bool success, bytes memory returnData) =
            SAFE.execTransactionFromModuleReturnData(address(SUPERCHAIN_CONFIG), 0, data, Enum.Operation.Call);
        if (!success) {
            revert ExecutionFailed(string(returnData));
        }
        emit Paused("Deputy Guardian");
    }

    /// @notice Calls the Security Council Safe's `execTransactionFromModuleReturnData()`, with the arguments
    ///      necessary to call `unpause()` on the `SuperchainConfig` contract.
    ///      Only the deputy guardian can call this function.
    function unpause() external {
        _onlyDeputyGuardian();

        bytes memory data = abi.encodeCall(SUPERCHAIN_CONFIG.unpause, ());
        (bool success, bytes memory returnData) =
            SAFE.execTransactionFromModuleReturnData(address(SUPERCHAIN_CONFIG), 0, data, Enum.Operation.Call);
        if (!success) {
            revert ExecutionFailed(string(returnData));
        }
        emit Unpaused();
    }

    /// @notice Calls the Security Council Safe's `execTransactionFromModuleReturnData()`, with the arguments
    ///      necessary to call `blacklistDisputeGame()` on the `OptimismPortal2` contract.
    ///      Only the deputy guardian can call this function.
    /// @param _portal The `OptimismPortal2` contract instance.
    /// @param _game The `IDisputeGame` contract instance.
    function blacklistDisputeGame(OptimismPortal2 _portal, IDisputeGame _game) external {
        _onlyDeputyGuardian();

        bytes memory data = abi.encodeCall(OptimismPortal2.blacklistDisputeGame, (_game));
        (bool success, bytes memory returnData) =
            SAFE.execTransactionFromModuleReturnData(address(_portal), 0, data, Enum.Operation.Call);
        if (!success) {
            revert ExecutionFailed(string(returnData));
        }
        emit DisputeGameBlacklisted(_game);
    }

    /// @notice Calls the Security Council Safe's `execTransactionFromModuleReturnData()`, with the arguments
    ///      necessary to call `setRespectedGameType()` on the `OptimismPortal2` contract.
    ///      Only the deputy guardian can call this function.
    /// @param _portal The `OptimismPortal2` contract instance.
    /// @param _gameType The `GameType` to set as the respected game type.
    function setRespectedGameType(OptimismPortal2 _portal, GameType _gameType) external {
        _onlyDeputyGuardian();

        bytes memory data = abi.encodeCall(OptimismPortal2.setRespectedGameType, (_gameType));
        (bool success, bytes memory returnData) =
            SAFE.execTransactionFromModuleReturnData(address(_portal), 0, data, Enum.Operation.Call);
        if (!success) {
            revert ExecutionFailed(string(returnData));
        }
        emit RespectedGameTypeSet(_gameType);
    }
}
