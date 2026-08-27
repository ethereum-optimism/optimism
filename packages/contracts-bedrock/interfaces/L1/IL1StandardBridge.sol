// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

interface IL1StandardBridge is IStandardBridge, IProxyAdminOwnedBase {
    error WithdrawalThrottle_TimestampOverflow();
    error ReinitializableBase_ZeroInitVersion();
    error L1StandardBridge_InvalidWithdrawalThrottleToken();
    error L1StandardBridge_InvalidWithdrawalThrottleBps();
    error L1StandardBridge_UnsupportedWithdrawalThrottleToken();
    error L1StandardBridge_InvalidWithdrawalThrottlePeriod();
    error L1StandardBridge_WithdrawalThrottleNotEnabled();
    error L1StandardBridge_WithdrawalThrottled(
        address token,
        uint256 requestedAmount,
        uint256 availableCapacity,
        uint256 totalCapacity
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

    event ERC20DepositInitiated(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256 amount,
        bytes extraData
    );
    event ERC20WithdrawalFinalized(
        address indexed l1Token,
        address indexed l2Token,
        address indexed from,
        address to,
        uint256 amount,
        bytes extraData
    );
    event ETHDepositInitiated(address indexed from, address indexed to, uint256 amount, bytes extraData);
    event ETHWithdrawalFinalized(address indexed from, address indexed to, uint256 amount, bytes extraData);
    event WithdrawalThrottleConfigured(
        address indexed token,
        uint16 maxBps,
        uint64 refillPeriod,
        uint256 stockSnapshot,
        uint256 capacity,
        uint256 available
    );
    event WithdrawalThrottleRefreshed(
        address indexed token, uint256 stockSnapshot, uint256 capacity, uint256 available
    );
    event WithdrawalThrottleDisabled(address indexed token);
    event WithdrawalThrottleCapacityConsumed(address indexed token, uint256 amount, uint256 remaining);
    event WithdrawalThrottleCapacityExhausted(address indexed token);

    function initVersion() external view returns (uint8);
    function depositERC20(
        address _l1Token,
        address _l2Token,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes memory _extraData
    )
        external;
    function depositERC20To(
        address _l1Token,
        address _l2Token,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes memory _extraData
    )
        external;
    function depositETH(uint32 _minGasLimit, bytes memory _extraData) external payable;
    function depositETHTo(address _to, uint32 _minGasLimit, bytes memory _extraData) external payable;
    function finalizeERC20Withdrawal(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes memory _extraData
    )
        external;
    function finalizeETHWithdrawal(
        address _from,
        address _to,
        uint256 _amount,
        bytes memory _extraData
    )
        external
        payable;
    function initialize(ICrossDomainMessenger _messenger, ISystemConfig _systemConfig) external;
    function l2TokenBridge() external view returns (address);
    function setWithdrawalThrottle(address _token, uint16 _maxBps, uint64 _refillPeriod) external;
    function refreshWithdrawalThrottle(address _token) external;
    function disableWithdrawalThrottle(address _token) external;
    function withdrawalThrottle(address _token) external view returns (WithdrawalThrottleConfig memory);
    function availableWithdrawalCapacity(address _token) external view returns (uint256);
    function systemConfig() external view returns (ISystemConfig);
    function version() external view returns (string memory);
    function superchainConfig() external view returns (ISuperchainConfig);

    function __constructor__() external;
}
