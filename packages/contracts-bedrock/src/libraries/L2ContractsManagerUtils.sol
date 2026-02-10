// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { XForkL2CMTypes } from "src/libraries/XForkL2CMTypes.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IStorageSetter } from "interfaces/universal/IStorageSetter.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";

/// @title L2ContractsManagerUtils
/// @notice L2ContractsManagerUtils is a library that provides utility functions for the L2ContractsManager system.
library L2ContractsManagerUtils {
    /// @notice Upgrades a predeploy to a new implementation without calling an initializer.
    /// @param _proxy The proxy address of the predeploy.
    /// @param _implementation The new implementation address.
    function upgradeTo(address _proxy, address _implementation) internal {
        IProxy(payable(_proxy)).upgradeTo(_implementation);
    }

    /// @notice Reads the configuration from a FeeVault predeploy.
    /// @param _feeVault The address of the FeeVault predeploy.
    /// @return config_ The FeeVault configuration.
    function readFeeVaultConfig(address _feeVault)
        internal
        view
        returns (XForkL2CMTypes.FeeVaultConfig memory config_)
    {
        IFeeVault feeVault = IFeeVault(payable(_feeVault));
        config_ = XForkL2CMTypes.FeeVaultConfig({
            recipient: feeVault.recipient(),
            minWithdrawalAmount: feeVault.minWithdrawalAmount(),
            withdrawalNetwork: feeVault.withdrawalNetwork()
        });
    }

    /// @notice Upgrades an initializable Predeploy's implementation to _implementation by resetting the initialized
    ///         slot and calling upgradeToAndCall with _data.
    /// @dev It's important to make sure that only initializable Predeploys are upgraded to this way.
    /// @param _proxy The proxy of the contract.
    /// @param _implementation The new implementation of the contract.
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
