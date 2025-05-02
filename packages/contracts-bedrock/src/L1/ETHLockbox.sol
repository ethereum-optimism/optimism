// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Contracts
import { ProxyAdminOwnedBase } from "src/L1/ProxyAdminOwnedBase.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";

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
contract ETHLockbox is ProxyAdminOwnedBase, Initializable, ISemver {
    /// @notice Thrown when the lockbox is paused.
    error ETHLockbox_Paused();

    /// @notice Thrown when the caller is not authorized.
    error ETHLockbox_Unauthorized();

    /// @notice Thrown when the value to unlock is greater than the balance of the lockbox.
    error ETHLockbox_InsufficientBalance();

    /// @notice Thrown when attempting to unlock ETH from the lockbox through a withdrawal transaction.
    error ETHLockbox_NoWithdrawalTransactions();

    /// @notice Thrown when the admin owner of the lockbox is different from the admin owner of the proxy admin.
    error ETHLockbox_DifferentProxyAdminOwner();

    /// @notice Thrown when any authorized portal has a different SuperchainConfig.
    error ETHLockbox_DifferentSuperchainConfig();

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

    /// @notice The address of the SystemConfig contract.
    ISystemConfig public systemConfig;

    /// @notice Mapping of authorized portals.
    mapping(IOptimismPortal => bool) public authorizedPortals;

    /// @notice Mapping of authorized lockboxes.
    mapping(IETHLockbox => bool) public authorizedLockboxes;

    /// @notice Semantic version.
    /// @custom:semver 1.1.1
    function version() public view virtual returns (string memory) {
        return "1.1.1";
    }

    /// @notice Constructs the ETHLockbox contract.
    constructor() {
        _disableInitializers();
    }

    /// @notice Initializer.
    /// @param _systemConfig The address of the SystemConfig contract.
    /// @param _portals The addresses of the portals to authorize.
    /// @dev Note: Multiple chains can share an ETHLockbox contract. In this case, all SystemConfig
    ///      contracts will point to the same pause identifier (the lockbox itself). Therefore, it
    ///      doesn't matter which SystemConfig is used here as long as it belongs to one of the
    ///      chains that share the lockbox.
    function initialize(ISystemConfig _systemConfig, IOptimismPortal[] calldata _portals) external initializer {
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

    /// @notice Authorizes a portal to lock and unlock ETH.
    /// @param _portal The address of the portal to authorize.
    function authorizePortal(IOptimismPortal _portal) external {
        // Check that the sender is the proxy admin owner.
        if (msg.sender != proxyAdminOwner()) revert ETHLockbox_Unauthorized();

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
        // Check that the sender is the proxy admin owner.
        if (msg.sender != proxyAdminOwner()) revert ETHLockbox_Unauthorized();

        // Check that the lockbox has the same proxy admin owner.
        if (!_sameProxyAdminOwner(address(_lockbox))) revert ETHLockbox_DifferentProxyAdminOwner();

        // Authorize the lockbox.
        authorizedLockboxes[_lockbox] = true;

        // Emit the event.
        emit LockboxAuthorized(_lockbox);
    }

    /// @notice Migrates liquidity from the current ETH lockbox to another.
    /// @dev    Must be called atomically with `OptimismPortal.migrateToSuperRoots()` in the same
    ///         transaction batch, or otherwise the OptimismPortal may not be able to unlock ETH
    ///         from the ETHLockbox on finalized withdrawals.
    /// @param _lockbox The address of the ETH lockbox to migrate liquidity to.
    function migrateLiquidity(IETHLockbox _lockbox) external {
        // Check that the sender is the proxy admin owner.
        if (msg.sender != proxyAdminOwner()) revert ETHLockbox_Unauthorized();

        // Check that the lockbox has the same proxy admin owner.
        if (!_sameProxyAdminOwner(address(_lockbox))) revert ETHLockbox_DifferentProxyAdminOwner();

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
        if (!_sameProxyAdminOwner(address(_portal))) revert ETHLockbox_DifferentProxyAdminOwner();

        // Check that the portal has the same superchain config.
        if (_portal.superchainConfig() != superchainConfig()) revert ETHLockbox_DifferentSuperchainConfig();

        // Authorize the portal.
        authorizedPortals[_portal] = true;

        // Emit the event.
        emit PortalAuthorized(_portal);
    }
}
