// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IReinitializableBase } from "interfaces/universal/IReinitializableBase.sol";

interface IETHLockbox is IProxyAdminOwnedBase, ISemver, IReinitializableBase {
    error WithdrawalThrottle_TimestampOverflow();
    error ETHLockbox_Unauthorized();
    error ETHLockbox_Paused();
    error ETHLockbox_InsufficientBalance();
    error ETHLockbox_NoWithdrawalTransactions();
    error ETHLockbox_DifferentSuperchainConfig();
    error ETHLockbox_InvalidWithdrawalThrottleBps();
    error ETHLockbox_InvalidWithdrawalThrottlePeriod();
    error ETHLockbox_WithdrawalThrottleNotEnabled();
    error ETHLockbox_WithdrawalThrottled(
        uint256 requestedAmount, uint256 availableCapacity, uint256 totalCapacity
    );

    /// @notice Stored withdrawal throttle state before pending refill is materialized.
    struct WithdrawalThrottleConfig {
        uint256 capacity;
        uint256 available;
        uint64 refillPeriod;
        uint64 lastUpdated;
        uint64 refillRemainder;
        uint16 maxBps;
        bool enabled;
    }

    event Initialized(uint8 version);
    event ETHLocked(IOptimismPortal2 indexed portal, uint256 amount);
    event ETHUnlocked(IOptimismPortal2 indexed portal, uint256 amount);
    event PortalAuthorized(IOptimismPortal2 indexed portal);
    event LockboxAuthorized(IETHLockbox indexed lockbox);
    event LiquidityMigrated(IETHLockbox indexed lockbox, uint256 amount);
    event LiquidityReceived(IETHLockbox indexed lockbox, uint256 amount);
    event WithdrawalThrottleConfigured(
        uint16 maxBps,
        uint64 refillPeriod,
        uint256 stockSnapshot,
        uint256 capacity,
        uint256 available
    );
    event WithdrawalThrottleRefreshed(uint256 stockSnapshot, uint256 capacity, uint256 available);
    event WithdrawalThrottleDisabled();
    event WithdrawalThrottleCapacityConsumed(uint256 amount, uint256 remaining);
    event WithdrawalThrottleCapacityExhausted();

    function initialize(ISystemConfig _systemConfig, IOptimismPortal2[] calldata _portals) external;
    function systemConfig() external view returns (ISystemConfig);
    function paused() external view returns (bool);
    function authorizedPortals(IOptimismPortal2) external view returns (bool);
    function authorizedLockboxes(IETHLockbox) external view returns (bool);
    function receiveLiquidity() external payable;
    function lockETH() external payable;
    function unlockETH(uint256 _value) external;
    function authorizePortal(IOptimismPortal2 _portal) external;
    function authorizeLockbox(IETHLockbox _lockbox) external;
    function migrateLiquidity(IETHLockbox _lockbox) external;
    function superchainConfig() external view returns (ISuperchainConfig);
    function setWithdrawalThrottle(uint16 _maxBps, uint64 _refillPeriod) external;
    function refreshWithdrawalThrottle() external;
    function disableWithdrawalThrottle() external;
    function withdrawalThrottle() external view returns (WithdrawalThrottleConfig memory);
    function availableWithdrawalCapacity() external view returns (uint256);

    function __constructor__() external;
}
