// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { WithdrawalThrottle } from "src/libraries/WithdrawalThrottle.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

/// @custom:proxied true
/// @title ETHLockbox
/// @notice Manages ETH liquidity locking and unlocking for authorized OptimismPortals, enabling unified ETH liquidity
///         management across chains in the superchain cluster.
contract ETHLockbox is ProxyAdminOwnedBase, Initializable, ReinitializableBase, ISemver {
    /// @notice Thrown when the lockbox is paused.
    error ETHLockbox_Paused();

    /// @notice Thrown when the caller is not authorized.
    error ETHLockbox_Unauthorized();

    /// @notice Thrown when the value to unlock is greater than the balance of the lockbox.
    error ETHLockbox_InsufficientBalance();

    /// @notice Thrown when attempting to unlock ETH from the lockbox through a withdrawal transaction.
    error ETHLockbox_NoWithdrawalTransactions();

    /// @notice Thrown when any authorized portal has a different SuperchainConfig.
    error ETHLockbox_DifferentSuperchainConfig();

    /// @notice Thrown when a withdrawal throttle basis point value is zero or exceeds 100%.
    error ETHLockbox_InvalidWithdrawalThrottleBps();

    /// @notice Thrown when a withdrawal throttle refill period is zero.
    error ETHLockbox_InvalidWithdrawalThrottlePeriod();

    /// @notice Thrown when the withdrawal throttle is not enabled.
    error ETHLockbox_WithdrawalThrottleNotEnabled();

    /// @notice Thrown when an ETH withdrawal exceeds the available withdrawal capacity.
    error ETHLockbox_WithdrawalThrottled(uint256 requestedAmount, uint256 availableCapacity, uint256 totalCapacity);

    /// @notice Withdrawal throttle state for ETH held by the lockbox.
    struct WithdrawalThrottleConfig {
        uint256 capacity;
        uint256 available;
        uint64 refillPeriod;
        uint64 lastUpdated;
        uint64 refillRemainder;
        uint16 maxBps;
        bool enabled;
    }

    /// @notice Emitted when ETH is locked in the lockbox by an authorized portal.
    /// @param portal The address of the portal that locked the ETH.
    /// @param amount The amount of ETH locked.
    event ETHLocked(IOptimismPortal indexed portal, uint256 amount);

    /// @notice Emitted when ETH is unlocked from the lockbox by an authorized portal.
    /// @param portal The address of the portal that unlocked the ETH.
    /// @param amount The amount of ETH unlocked.
    event ETHUnlocked(IOptimismPortal indexed portal, uint256 amount);

    /// @notice Emitted when a portal is authorized to lock and unlock ETH.
    /// @param portal The address of the portal that was authorized.
    event PortalAuthorized(IOptimismPortal indexed portal);

    /// @notice Emitted when an ETH lockbox is authorized to migrate its liquidity to the current ETH lockbox.
    /// @param lockbox The address of the ETH lockbox that was authorized.
    event LockboxAuthorized(IETHLockbox indexed lockbox);

    /// @notice Emitted when ETH liquidity is migrated from the current ETH lockbox to another.
    /// @param lockbox The address of the ETH lockbox that was migrated.
    event LiquidityMigrated(IETHLockbox indexed lockbox, uint256 amount);

    /// @notice Emitted when ETH liquidity is received during an authorized lockbox migration.
    /// @param lockbox The address of the ETH lockbox that received the liquidity.
    /// @param amount The amount of ETH received.
    event LiquidityReceived(IETHLockbox indexed lockbox, uint256 amount);

    /// @notice Emitted when the ETH withdrawal throttle is configured.
    event WithdrawalThrottleConfigured(
        uint16 maxBps, uint64 refillPeriod, uint256 stockSnapshot, uint256 capacity, uint256 available
    );

    /// @notice Emitted when the ETH withdrawal throttle stock snapshot is refreshed.
    event WithdrawalThrottleRefreshed(uint256 stockSnapshot, uint256 capacity, uint256 available);

    /// @notice Emitted when the ETH withdrawal throttle is disabled.
    event WithdrawalThrottleDisabled();

    /// @notice Emitted when ETH withdrawal capacity is consumed.
    event WithdrawalThrottleCapacityConsumed(uint256 amount, uint256 remaining);

    /// @notice Emitted when all currently available ETH withdrawal capacity is consumed.
    event WithdrawalThrottleCapacityExhausted();

    /// @notice The address of the SystemConfig contract.
    ISystemConfig public systemConfig;

    /// @notice Mapping of authorized portals.
    mapping(IOptimismPortal => bool) public authorizedPortals;

    /// @notice Mapping of authorized lockboxes.
    mapping(IETHLockbox => bool) public authorizedLockboxes;

    /// @notice Withdrawal throttle state for ETH held by the lockbox.
    WithdrawalThrottleConfig internal _withdrawalThrottle;

    /// @notice Semantic version.
    /// @custom:semver 2.0.0
    function version() public view virtual returns (string memory) {
        return "2.0.0";
    }

    /// @notice Constructs the ETHLockbox contract.
    constructor() ReinitializableBase(1) {
        _disableInitializers();
    }

    /// @notice Initializer.
    /// @param _systemConfig The address of the SystemConfig contract.
    /// @param _portals The addresses of the portals to authorize.
    /// @dev Note: Multiple chains can share an ETHLockbox contract. In this case, all SystemConfig
    ///      contracts will point to the same pause identifier (the lockbox itself). Therefore, it
    ///      doesn't matter which SystemConfig is used here as long as it belongs to one of the
    ///      chains that share the lockbox.
    function initialize(
        ISystemConfig _systemConfig,
        IOptimismPortal[] calldata _portals
    )
        external
        reinitializer(initVersion())
    {
        // Initialization transactions must come from the ProxyAdmin or its owner.
        _assertOnlyProxyAdminOrProxyAdminOwner();

        // Now perform initialization logic.
        systemConfig = _systemConfig;
        for (uint256 i; i < _portals.length; i++) {
            _authorizePortal(_portals[i]);
        }
    }

    /// @notice Getter for the current paused status.
    function paused() public view returns (bool) {
        return systemConfig.paused();
    }

    /// @notice Returns the SuperchainConfig contract.
    /// @return ISuperchainConfig The SuperchainConfig contract.
    function superchainConfig() public view returns (ISuperchainConfig) {
        return systemConfig.superchainConfig();
    }

    /// @notice Configures ETH withdrawal capacity as a percentage of the current lockbox balance.
    ///         Reconfiguration preserves accrued capacity and clamps it to the new maximum.
    /// @param _maxBps       Maximum withdrawable stock in basis points.
    /// @param _refillPeriod Time in seconds for the bucket to refill from empty to full.
    function setWithdrawalThrottle(uint16 _maxBps, uint64 _refillPeriod) external {
        _assertOnlyProxyAdminOwner();

        if (_maxBps == 0 || _maxBps > WithdrawalThrottle.MAX_BPS) {
            revert ETHLockbox_InvalidWithdrawalThrottleBps();
        }
        if (_refillPeriod == 0) revert ETHLockbox_InvalidWithdrawalThrottlePeriod();

        WithdrawalThrottleConfig storage throttle = _withdrawalThrottle;
        uint256 available;
        if (throttle.enabled) (available,) = _availableWithdrawalCapacity();
        else available = type(uint256).max;
        uint256 stockSnapshot = address(this).balance;
        uint256 capacity = WithdrawalThrottle.capacity(stockSnapshot, _maxBps);
        if (available > capacity) available = capacity;

        throttle.capacity = capacity;
        throttle.available = available;
        throttle.refillPeriod = _refillPeriod;
        throttle.lastUpdated = uint64(block.timestamp);
        throttle.refillRemainder = 0;
        throttle.maxBps = _maxBps;
        throttle.enabled = true;

        emit WithdrawalThrottleConfigured(_maxBps, _refillPeriod, stockSnapshot, capacity, available);
    }

    /// @notice Recomputes ETH withdrawal capacity from the current lockbox balance without refilling it.
    function refreshWithdrawalThrottle() external {
        _assertOnlyProxyAdminOwner();

        WithdrawalThrottleConfig storage throttle = _withdrawalThrottle;
        if (!throttle.enabled) revert ETHLockbox_WithdrawalThrottleNotEnabled();

        (uint256 available, uint64 refillRemainder) = _availableWithdrawalCapacity();
        uint256 stockSnapshot = address(this).balance;
        uint256 capacity = WithdrawalThrottle.capacity(stockSnapshot, throttle.maxBps);
        if (available >= capacity) {
            available = capacity;
            refillRemainder = 0;
        }

        throttle.capacity = capacity;
        throttle.available = available;
        throttle.lastUpdated = uint64(block.timestamp);
        throttle.refillRemainder = refillRemainder;

        emit WithdrawalThrottleRefreshed(stockSnapshot, capacity, available);
    }

    /// @notice Disables the ETH withdrawal throttle.
    function disableWithdrawalThrottle() external {
        _assertOnlyProxyAdminOwner();

        if (!_withdrawalThrottle.enabled) revert ETHLockbox_WithdrawalThrottleNotEnabled();
        delete _withdrawalThrottle;

        emit WithdrawalThrottleDisabled();
    }

    /// @notice Returns the stored ETH withdrawal throttle state before pending refill is materialized.
    /// @return Withdrawal throttle state.
    function withdrawalThrottle() external view returns (WithdrawalThrottleConfig memory) {
        return _withdrawalThrottle;
    }

    /// @notice Returns currently available ETH withdrawal capacity.
    /// @return Available capacity, or the maximum uint256 value when throttling is disabled.
    function availableWithdrawalCapacity() external view returns (uint256) {
        if (!_withdrawalThrottle.enabled) return type(uint256).max;
        (,, uint256 available,) = _syncedWithdrawalCapacity();
        return available;
    }

    /// @notice Authorizes a portal to lock and unlock ETH.
    /// @param _portal The address of the portal to authorize.
    function authorizePortal(IOptimismPortal _portal) external {
        // Check that this transaction is coming from the ProxyAdmin owner.
        _assertOnlyProxyAdminOwner();

        // Authorize the portal.
        _authorizePortal(_portal);
    }

    /// @notice Receives the ETH liquidity migrated from an authorized lockbox.
    function receiveLiquidity() external payable {
        // Check that the sender is authorized to trigger this function.
        IETHLockbox sender = IETHLockbox(payable(msg.sender));
        if (!authorizedLockboxes[sender]) revert ETHLockbox_Unauthorized();

        // Emit the event.
        emit LiquidityReceived(sender, msg.value);
    }

    /// @notice Locks ETH in the lockbox.
    ///         Called by an authorized portal on a deposit to lock the ETH value.
    function lockETH() external payable {
        // Check that the sender is authorized to trigger this function.
        IOptimismPortal sender = IOptimismPortal(payable(msg.sender));
        if (!authorizedPortals[sender]) revert ETHLockbox_Unauthorized();

        // Emit the event.
        emit ETHLocked(sender, msg.value);
    }

    /// @notice Unlocks ETH from the lockbox.
    ///         Called by an authorized portal when finalizing a withdrawal that requires ETH.
    ///         Cannot be called if the lockbox is paused.
    /// @param _value The amount of ETH to unlock.
    function unlockETH(uint256 _value) external {
        // Unlocks are blocked when paused, locks are not.
        if (paused()) revert ETHLockbox_Paused();

        // Check that the sender is authorized to trigger this function.
        IOptimismPortal sender = IOptimismPortal(payable(msg.sender));
        if (!authorizedPortals[sender]) revert ETHLockbox_Unauthorized();

        // Check that we have enough balance to process the unlock.
        if (_value > address(this).balance) revert ETHLockbox_InsufficientBalance();

        // Check that the sender is not executing a withdrawal transaction.
        if (sender.l2Sender() != Constants.DEFAULT_L2_SENDER) {
            revert ETHLockbox_NoWithdrawalTransactions();
        }

        _consumeWithdrawalCapacity(_value);

        // Using donateETH to avoid triggering a deposit.
        sender.donateETH{ value: _value }();

        // Emit the event.
        emit ETHUnlocked(sender, _value);
    }

    /// @notice Authorizes an ETH lockbox to migrate its liquidity to the current ETH lockbox. We
    ///         allow this function to be called more than once for the same lockbox. A lockbox
    ///         cannot be removed from the authorized list once added.
    /// @param _lockbox The address of the ETH lockbox to authorize.
    function authorizeLockbox(IETHLockbox _lockbox) external {
        // Check that this transaction is coming from the ProxyAdmin owner.
        _assertOnlyProxyAdminOwner();

        // Check that the lockbox has the same proxy admin owner.
        _assertSharedProxyAdminOwner(address(_lockbox));

        // Authorize the lockbox.
        authorizedLockboxes[_lockbox] = true;

        // Emit the event.
        emit LockboxAuthorized(_lockbox);
    }

    /// @notice Migrates liquidity from the current ETH lockbox to another.
    /// @dev    Must be called atomically with `OptimismPortal.migrateToSharedDisputeGame()` in the same
    ///         transaction batch, or otherwise the OptimismPortal may not be able to unlock ETH
    ///         from the ETHLockbox on finalized withdrawals.
    /// @param _lockbox The address of the ETH lockbox to migrate liquidity to.
    function migrateLiquidity(IETHLockbox _lockbox) external {
        // Check that this transaction is coming from the ProxyAdmin owner.
        _assertOnlyProxyAdminOwner();

        // Check that the lockbox has the same proxy admin owner.
        _assertSharedProxyAdminOwner(address(_lockbox));

        // Receive the liquidity.
        uint256 balance = address(this).balance;
        IETHLockbox(_lockbox).receiveLiquidity{ value: balance }();

        // Emit the event.
        emit LiquidityMigrated(_lockbox, balance);
    }

    /// @notice Authorizes a portal to lock and unlock ETH.
    /// @param _portal The address of the portal to authorize.
    function _authorizePortal(IOptimismPortal _portal) internal {
        // Check that the portal has the same proxy admin owner.
        _assertSharedProxyAdminOwner(address(_portal));

        // Check that the portal has the same superchain config.
        if (_portal.superchainConfig() != superchainConfig()) revert ETHLockbox_DifferentSuperchainConfig();

        // Authorize the portal.
        authorizedPortals[_portal] = true;

        // Emit the event.
        emit PortalAuthorized(_portal);
    }

    /// @notice Consumes available ETH withdrawal capacity when the throttle is enabled.
    /// @param _amount Amount of ETH to withdraw.
    function _consumeWithdrawalCapacity(uint256 _amount) internal {
        WithdrawalThrottleConfig storage throttle = _withdrawalThrottle;
        if (!throttle.enabled || _amount == 0) return;

        (uint256 stockSnapshot, uint256 capacity, uint256 available, uint64 refillRemainder) =
            _syncedWithdrawalCapacity();
        if (_amount > available) {
            revert ETHLockbox_WithdrawalThrottled(_amount, available, capacity);
        }

        uint256 remaining = available - _amount;
        bool capacityChanged = capacity != throttle.capacity;
        throttle.capacity = capacity;
        throttle.available = remaining;
        throttle.lastUpdated = uint64(block.timestamp);
        throttle.refillRemainder = refillRemainder;

        if (capacityChanged) emit WithdrawalThrottleRefreshed(stockSnapshot, capacity, available);
        emit WithdrawalThrottleCapacityConsumed(_amount, remaining);
        if (remaining == 0) emit WithdrawalThrottleCapacityExhausted();
    }

    /// @notice Computes available ETH withdrawal capacity after its pending linear refill.
    /// @return available_ Available capacity at the current timestamp.
    /// @return remainder_ Fractional refill numerator to preserve when materializing the refill.
    function _availableWithdrawalCapacity() internal view returns (uint256 available_, uint64 remainder_) {
        WithdrawalThrottleConfig storage throttle = _withdrawalThrottle;
        return WithdrawalThrottle.available(
            throttle.capacity,
            throttle.available,
            throttle.refillRemainder,
            throttle.refillPeriod,
            throttle.lastUpdated,
            block.timestamp
        );
    }

    /// @notice Computes the effective bucket state against the lockbox's current ETH stock.
    /// @return stock_ Current lockbox balance.
    /// @return capacity_ Capacity derived from the current stock.
    /// @return available_ Available capacity after refill and stock synchronization.
    /// @return remainder_ Fractional refill numerator to preserve when materializing the refill.
    function _syncedWithdrawalCapacity()
        internal
        view
        returns (uint256 stock_, uint256 capacity_, uint256 available_, uint64 remainder_)
    {
        WithdrawalThrottleConfig storage throttle = _withdrawalThrottle;
        (available_, remainder_) = _availableWithdrawalCapacity();
        stock_ = address(this).balance;
        capacity_ = WithdrawalThrottle.capacity(stock_, throttle.maxBps);
        if (available_ >= capacity_) {
            available_ = capacity_;
            remainder_ = 0;
        }
    }
}
