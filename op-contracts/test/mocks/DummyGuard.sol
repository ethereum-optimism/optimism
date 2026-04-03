// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { BaseGuard } from "safe-contracts/base/GuardManager.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

/// @title DummyGuard
/// @notice A minimal guard implementation for testing purposes that does nothing.
///         This guard implements the Guard interface with ERC165 support
///         but performs no actual validation. Useful for testing guard setup/removal.
contract DummyGuard is BaseGuard {
    /// @notice Emitted when checkTransaction is called
    event CheckedTransaction(address indexed safe, address to, uint256 value);

    /// @notice Emitted when checkAfterExecution is called
    event CheckedAfterExecution(bytes32 indexed txHash, bool success);

    /// @notice Pre-transaction check (no-op implementation)
    /// @dev This function does nothing and always succeeds
    function checkTransaction(
        address to,
        uint256 value,
        bytes memory, /* data */
        Enum.Operation, /* operation */
        uint256, /* safeTxGas */
        uint256, /* baseGas */
        uint256, /* gasPrice */
        address, /* gasToken */
        address payable, /* refundReceiver */
        bytes memory, /* signatures */
        address msgSender
    )
        external
        override
    {
        emit CheckedTransaction(msgSender, to, value);
    }

    /// @notice Post-transaction check (no-op implementation)
    /// @dev This function does nothing and always succeeds
    function checkAfterExecution(bytes32 hash, bool success) external override {
        emit CheckedAfterExecution(hash, success);
    }
}
