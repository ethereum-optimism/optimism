// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IWETH } from "src/dispute/interfaces/IWETH.sol";

/// @title IDelayedWETH
/// @notice Interface for the DelayedWETH contract.
interface IDelayedWETH is IWETH {
    /// @notice Represents a withdrawal request.
    struct WithdrawalRequest {
        uint256 amount;
        uint256 timestamp;
    }

    /// @notice Emitted when an unwrap is started.
    /// @param src The address that started the unwrap.
    /// @param wad The amount of WETH that was unwrapped.
    event Unwrap(address indexed src, uint256 wad);

    /// @notice Returns the withdrawal delay in seconds.
    /// @return The withdrawal delay in seconds.
    function delay() external view returns (uint256);

    /// @notice Returns a withdrawal request for the given address.
    /// @param _owner The address to query the withdrawal request of.
    /// @param _guy Sub-account to query the withdrawal request of.
    /// @return The withdrawal request for the given address-subaccount pair.
    function withdrawals(address _owner, address _guy) external view returns (uint256, uint256);

    /// @notice Unlocks withdrawals for the sender's account, after a time delay.
    /// @param _guy Sub-account to unlock.
    /// @param _wad The amount of WETH to unlock.
    function unlock(address _guy, uint256 _wad) external;

    /// @notice Extension to withdrawal, must provide a sub-account to withdraw from.
    /// @param _guy Sub-account to withdraw from.
    /// @param _wad The amount of WETH to withdraw.
    function withdraw(address _guy, uint256 _wad) external;

    /// @notice Allows the owner to recover from error cases by pulling ETH out of the contract.
    /// @param _wad The amount of WETH to recover.
    function recover(uint256 _wad) external;

    /// @notice Allows the owner to recover from error cases by pulling ETH from a specific owner.
    /// @param _guy The address to recover the WETH from.
    /// @param _wad The amount of WETH to recover.
    function hold(address _guy, uint256 _wad) external;
}
