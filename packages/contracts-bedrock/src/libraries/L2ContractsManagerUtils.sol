// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { L2ContractsManagerTypes } from "src/libraries/L2ContractsManagerTypes.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Types } from "src/libraries/Types.sol";

// Contracts
import { L2ProxyAdmin } from "src/L2/L2ProxyAdmin.sol";

// Interfaces
import { IStorageSetter } from "interfaces/universal/IStorageSetter.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

/// @title L2ContractsManagerUtils
/// @notice L2ContractsManagerUtils is a library that provides utility functions for the L2ContractsManager system.
/// @dev Upgrade functions revert if the target predeploy is not upgradeable (e.g., not proxied).
///      Callers must guard calls for conditionally-deployed predeploys.
library L2ContractsManagerUtils {
    /// @notice Thrown when a user attempts to downgrade a contract.
    /// @param _target The address of the contract that was attempted to be downgraded.
    error L2ContractsManager_DowngradeNotAllowed(address _target);

    /// @notice Thrown when a contract is in the process of being initialized during an upgrade.
    error L2ContractsManager_InitializingDuringUpgrade();

    /// @notice Thrown when a predeploy is not upgradeable.
    /// @param _target The address of the non-upgradeable predeploy.
    error L2ContractsManager_NotUpgradeable(address _target);

    /// @notice Thrown when a v5 slot is passed with a non-zero offset.
    error L2ContractsManager_InvalidV5Offset();

    /// @notice Upgrades a predeploy to a new implementation without calling an initializer.
    ///         Reverts if the predeploy is not upgradeable.
    /// @param _proxy The proxy address of the predeploy.
    /// @param _implementation The new implementation address.
    function upgradeTo(address _proxy, address _implementation) internal {
        if (!Predeploys.isUpgradeable(_proxy)) revert L2ContractsManager_NotUpgradeable(_proxy);

        // We skip checking the version for those predeploys that have no code. This would be the case for newly added
        // predeploys that are being introduced on this particular upgrade.
        address implementation = L2ProxyAdmin(Predeploys.PROXY_ADMIN).getProxyImplementation(_proxy);

        // We avoid downgrading Predeploys
        if (
            implementation.code.length != 0
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
        // TODO(#19600): Remove withdrawalNetwork reading as part of revenue sharing deprecation.
        // Try to read the withdrawal network from the FeeVault. If it fails, use the default value.
        Types.WithdrawalNetwork withdrawalNetwork;
        // eip150-safe
        try IFeeVault(payable(_feeVault)).WITHDRAWAL_NETWORK() returns (Types.WithdrawalNetwork withdrawalNetwork_) {
            withdrawalNetwork = withdrawalNetwork_;
        } catch {
            // Previous FeeVault implementations hardcoded L1 withdrawals (via L2StandardBridge.bridgeETHTo)
            // and did not expose a WITHDRAWAL_NETWORK() function. We preserve this L1 behavior as the default.
            // Modifying this configuration requires explicit migration steps outside the L2ContractsManager upgrade
            // flow.
            withdrawalNetwork = Types.WithdrawalNetwork.L1;
        }

        // Note: We are intentionally using legacy deprecated getters for this 1.0.0 version of the L2ContractsManager.
        // Subsequent versions should use the new getters as L2ContractsManager should ensure that the new current
        // version of the FeeVault is used.
        IFeeVault feeVault = IFeeVault(payable(_feeVault));
        config_ = L2ContractsManagerTypes.FeeVaultConfig({
            recipient: feeVault.RECIPIENT(),
            minWithdrawalAmount: feeVault.MIN_WITHDRAWAL_AMOUNT(),
            withdrawalNetwork: withdrawalNetwork
        });
    }

    /// @notice Upgrades an initializable Predeploy's implementation to _implementation by resetting the initialized
    ///         slot and calling upgradeToAndCall with _data. Reverts if the predeploy is not upgradeable.
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
        if (!Predeploys.isUpgradeable(_proxy)) revert L2ContractsManager_NotUpgradeable(_proxy);

        // We skip checking the version for those predeploys that have no code. This would be the case for newly added
        // predeploys that are being introduced on this particular upgrade.
        address implementation = L2ProxyAdmin(Predeploys.PROXY_ADMIN).getProxyImplementation(_proxy);

        if (
            implementation.code.length != 0
                && SemverComp.gt(ISemver(_proxy).version(), ISemver(_implementation).version())
        ) {
            revert L2ContractsManager_DowngradeNotAllowed(address(_proxy));
        }

        // Upgrade to StorageSetter.
        IProxy(payable(_proxy)).upgradeTo(_storageSetterImpl);

        // OZ v5 ERC-7201 Initializable namespaced slot. For v4 contracts this slot is all zeros.
        // Slot derivation (ERC-7201):
        //   keccak256(abi.encode(uint256(keccak256("openzeppelin.storage.Initializable")) - 1)) &
        // ~bytes32(uint256(0xff))
        // Ref:
        // https://github.com/OpenZeppelin/openzeppelin-contracts/blob/6b55a93e/contracts/proxy/utils/Initializable.sol#L77
        bytes32 v5Slot = bytes32(uint256(0xf0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a00));

        // V5 contracts use a fixed layout with _initialized at offset 0. A non-zero offset
        // would misalign the clearing mask and corrupt the slot.
        if (_slot == v5Slot && _offset != 0) {
            revert L2ContractsManager_InvalidV5Offset();
        }

        // OZ v4: check `_initializing` and clear `_initialized` byte.
        // Only applies when `_slot` differs from the v5 namespaced slot, to avoid
        // misreading v5's uint64 `_initialized` field as the v4 `_initializing` flag.
        if (_slot != v5Slot) {
            bytes32 v4Value = IStorageSetter(_proxy).getBytes32(_slot);
            if ((uint256(v4Value) >> (uint256(_offset + 1) * 8)) & 0xFF != 0) {
                revert L2ContractsManager_InitializingDuringUpgrade();
            }
            uint256 v4Mask = ~(uint256(0xff) << (uint256(_offset) * 8));
            IStorageSetter(_proxy).setBytes32(_slot, bytes32(uint256(v4Value) & v4Mask));
        }

        // OZ v5: check `_initializing` and clear `_initialized` uint64.
        // OZ v5 stores `_initialized` as uint64 in the low 8 bytes and `_initializing` as
        // bool at byte offset 8 of the ERC-7201 namespaced slot.
        // For v4 contracts this slot is all zeros, making this a no-op.
        uint256 v5Value = uint256(IStorageSetter(_proxy).getBytes32(v5Slot));
        if ((v5Value >> 64) & 0xFF != 0) {
            revert L2ContractsManager_InitializingDuringUpgrade();
        }
        IStorageSetter(_proxy).setBytes32(v5Slot, bytes32(v5Value & ~uint256(0xFFFFFFFFFFFFFFFF)));

        // Upgrade to the implementation and call the initializer.
        IProxy(payable(_proxy)).upgradeToAndCall(_implementation, _data);
    }
}
