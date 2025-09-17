// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Enum } from "safe-contracts/common/Enum.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { Guard as IGuard } from "safe-contracts/base/GuardManager.sol";


/// @title ITimelockGuard
/// @notice Interface for the TimelockGuard Safe guard.
interface ITimelockGuard is IGuard, ISemver {
    // Errors
    error TimelockGuard_GuardNotConfigured();
    error TimelockGuard_GuardNotEnabled();
    error TimelockGuard_GuardStillEnabled();
    error TimelockGuard_InvalidTimelockDelay();

    // Events
    event GuardCleared(address indexed safe);
    event GuardConfigured(address indexed safe, uint256 timelockDelay);

    struct GuardConfig {
        uint256 timelockDelay;
        bool configured;
    }

    // Views
    function version() external view returns (string memory);

    function viewTimelockGuardConfiguration(address _safe) external view returns (GuardConfig memory);

    function cancellationThreshold(address _safe) external view returns (uint256 cancellationThreshold_);

    function safeConfigs(address)
        external
        view
        returns (uint256 timelockDelay);

    // Admin
    function configureTimelockGuard(uint256 _timelockDelay) external;

    function clearTimelockGuard() external;

    // Scheduling API (placeholders until fully implemented in the guard)
    function scheduleTransaction(
        address _safe,
        address _to,
        uint256 _value,
        bytes memory _data,
        Enum.Operation _operation,
        uint256 _safeTxGas,
        uint256 _baseGas,
        uint256 _gasPrice,
        address _gasToken,
        address payable _refundReceiver,
        bytes memory _signatures
    )
        external
        pure;

    function checkPendingTransactions(address _safe) external pure returns (bytes32[] memory pendingTxs_);

    function cancelTransaction(address _safe, bytes32 _txHash) external pure;
}

