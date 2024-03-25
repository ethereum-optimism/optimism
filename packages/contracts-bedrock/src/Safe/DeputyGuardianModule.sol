// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ISemver } from "src/universal/ISemver.sol";

/// @title DeputyGuardianModule
/// @notice This module is intended to be enabled on th
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

    /// @notice Getter function for the deputy guardian's address
    /// @return deputyGuardian_ The deputy guardian's address
    function deputyGuardian() public view returns (address deputyGuardian_) {
        deputyGuardian_ = DEPUTY_GUARDIAN;
    }

    /// @notice Getter function for the SuperchainConfig's address
    /// @return superchainConfig_ The SuperchainConfig's address
    function superchainConfig() public view returns (SuperchainConfig superchainConfig_) {
        superchainConfig_ = SUPERCHAIN_CONFIG;
    }

    /// @dev Calls to the Security Council's `execTransactionFromModule()`, with the arguments
    ///      necessary to call `pause()` on the `SuperchainConfig` contract.
    ///     Only the deputy guardian can call this function.
    function pause() external {
        require(msg.sender == DEPUTY_GUARDIAN, "DeputyGuardianModule: Only the deputy guardian can call this function");
        bytes memory data = abi.encodeWithSelector(SUPERCHAIN_CONFIG.pause.selector, "");
        SAFE.execTransactionFromModule(address(SUPERCHAIN_CONFIG), 0, data, Enum.Operation.Call);
    }

    /// @dev Calls to the Security Council's `execTransactionFromModule()`, with the arguments
    ///      necessary to call `unpause()` on the `SuperchainConfig` contract.
    ///     Only the deputy guardian can call this function.
    function unpause() external {
        require(msg.sender == DEPUTY_GUARDIAN, "DeputyGuardianModule: Only the deputy guardian can call this function");
        bytes memory data = abi.encodeWithSelector(SUPERCHAIN_CONFIG.unpause.selector);
        SAFE.execTransactionFromModule(address(SUPERCHAIN_CONFIG), 0, data, Enum.Operation.Call);
    }

    /// @dev When called, this function will call to the Security Council's `execTransactionFromModule()`
    ///      with the arguments necessary to call `blacklistDisputeGame()` on the `DisputeGameFactory` contract.
    ////     Only the deputy guardian can call this function.
    function blacklistDisputeGame(address) external { }

    /// @dev When called, this function will call to the Security Council's `execTransactionFromModule()`
    ///      with the arguments necessary to call `setRespectedGameType()` on the `DisputeGameFactory` contract.
    ////     Only the deputy guardian can call this function.
    function setRespectedGameType(uint32) external { }
}
