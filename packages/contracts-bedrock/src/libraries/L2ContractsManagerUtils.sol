// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { L2ContractsManagerTypes } from "src/libraries/L2ContractsManagerTypes.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IStorageSetter } from "interfaces/universal/IStorageSetter.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

/// @title L2ContractsManagerUtils
/// @notice L2ContractsManagerUtils is a library that provides utility functions for the L2ContractsManager system.
/// @dev Upgrade functions silently skip predeploys that are not upgradeable (i.e., not deployed on the chain).
///      This is intentional to support chains where certain predeploys are conditionally deployed, such as
///      CrossL2Inbox on non-interop chains or LiquidityController on non-custom-gas-token chains.
library L2ContractsManagerUtils {
    /// @notice Thrown when a user attempts to downgrade a contract.
    /// @param _target The address of the contract that was attempted to be downgraded.
    error L2ContractsManager_DowngradeNotAllowed(address _target);

    /// @notice Upgrades a predeploy to a new implementation without calling an initializer.
    ///         If the predeploy is not upgradeable, this function is a no-op.
    /// @param _proxy The proxy address of the predeploy.
    /// @param _implementation The new implementation address.
    function upgradeTo(address _proxy, address _implementation) internal {
        // Skip if the predeploy is not upgradeable (e.g., not deployed on this chain).
        if (!Predeploys.isUpgradeable(_proxy)) return;

        // We avoid downgrading Predeploys
        if (
            // TODO(#19195): Remove this code skipping the ProxyAdmin once version is implemented.
            _proxy != Predeploys.PROXY_ADMIN
                && ProxyAdmin(Predeploys.PROXY_ADMIN).getProxyImplementation(_proxy) != address(0)
                && SemverComp.gt(ISemver(_proxy).version(), ISemver(_implementation).version())
        ) {
            revert L2ContractsManager_DowngradeNotAllowed(address(_proxy));
        }

        IProxy(payable(_proxy)).upgradeTo(_implementation);
    }

    /// @notice Reads the configuration from a FeeVault predeploy.
    /// @param _feeVault The address of the FeeVault predeploy.
    /// @return config_ The FeeVault configuration.
    function readFeeVaultConfig(address _feeVault)
        internal
        view
        returns (L2ContractsManagerTypes.FeeVaultConfig memory config_)
    {
        // Note: We are intentionally using legacy deprecated getters for this 1.0.0 version of the L2ContractsManager.
        // Subsequent versions should use the new getters as L2ContractsManager should ensure that the new current
        // version of the FeeVault is used.
        IFeeVault feeVault = IFeeVault(payable(_feeVault));
        config_ = L2ContractsManagerTypes.FeeVaultConfig({
            recipient: feeVault.RECIPIENT(),
            minWithdrawalAmount: feeVault.MIN_WITHDRAWAL_AMOUNT(),
            withdrawalNetwork: feeVault.WITHDRAWAL_NETWORK()
        });
    }

    /// @notice Upgrades an initializable Predeploy's implementation to _implementation by resetting the initialized
    ///         slot and calling upgradeToAndCall with _data. If the predeploy is not upgradeable, this function
    ///         is a no-op.
    /// @dev It's important to make sure that only initializable Predeploys are upgraded this way.
    /// @param _proxy The proxy of the contract.
    /// @param _implementation The new implementation of the contract.
    /// @param _storageSetterImpl The address of the StorageSetter implementation.
    /// @param _data The data to call upgradeToAndCall with.
    /// @param _slot The slot where the initialized value is located.
    /// @param _offset The offset of the initializer value in the slot.
    function upgradeToAndCall(
        address _proxy,
        address _implementation,
        address _storageSetterImpl,
        bytes memory _data,
        bytes32 _slot,
        uint8 _offset
    )
        internal
    {
        // Skip if the predeploy is not upgradeable (e.g., not deployed on this chain).
        if (!Predeploys.isUpgradeable(_proxy)) return;

        if (
            // TODO(#19195): Remove this code skipping the ProxyAdmin once version is implemented.
            // This should never be the case, if you're trying to initialize the ProxyAdmin, it's probably a mistake.
            _proxy != Predeploys.PROXY_ADMIN
                && ProxyAdmin(Predeploys.PROXY_ADMIN).getProxyImplementation(_proxy) != address(0)
                && SemverComp.gt(ISemver(_proxy).version(), ISemver(_implementation).version())
        ) {
            revert L2ContractsManager_DowngradeNotAllowed(address(_proxy));
        }

        // Upgrade to StorageSetter.
        IProxy(payable(_proxy)).upgradeTo(_storageSetterImpl);

        // Reset the initialized slot by zeroing the single byte at `_offset` (from the right).
        bytes32 current = IStorageSetter(_proxy).getBytes32(_slot);
        uint256 mask = ~(uint256(0xff) << (uint256(_offset) * 8));
        IStorageSetter(_proxy).setBytes32(_slot, bytes32(uint256(current) & mask));

        // Upgrade to the implementation and call the initializer.
        IProxy(payable(_proxy)).upgradeToAndCall(_implementation, _data);
    }
}
