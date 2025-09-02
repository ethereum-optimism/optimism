// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { SafeSend } from "src/universal/SafeSend.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

// Errors
import { Unauthorized, InvalidAmount } from "src/libraries/errors/CommonErrors.sol";

/// @custom:predeploy 0x4200000000000000000000000000000000000029
/// @title NativeAssetLiquidity
/// @notice The NativeAssetLiquidity contract allows other contracts to access native asset liquidity
contract NativeAssetLiquidity is ISemver {
    /// @notice Emitted when an address withdraws native asset liquidity.
    event LiquidityWithdrawn(address indexed caller, uint256 value);

    /// @notice Emitted when an address deposits native asset liquidity.
    event LiquidityDeposited(address indexed caller, uint256 value);

    /// @notice Emitted when funds are received.
    event LiquidityFunded(address indexed funder, uint256 value);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Allows an address to lock native asset liquidity into this contract.
    function deposit() external payable {
        if (msg.sender != Predeploys.LIQUIDITY_CONTROLLER) revert Unauthorized();

        emit LiquidityDeposited(msg.sender, msg.value);
    }

    /// @notice Allows an address to unlock native asset liquidity from this contract.
    /// @param _amount The amount of liquidity to unlock.
    function withdraw(uint256 _amount) external {
        if (msg.sender != Predeploys.LIQUIDITY_CONTROLLER) revert Unauthorized();

        new SafeSend{ value: _amount }(payable(msg.sender));

        emit LiquidityWithdrawn(msg.sender, _amount);
    }

    /// @notice Fund the contract by sending native asset.
    /// @dev The function is payable to accept native asset.
    function fund() external payable {
        if (msg.value == 0) revert InvalidAmount();

        emit LiquidityFunded(msg.sender, msg.value);
    }
}
