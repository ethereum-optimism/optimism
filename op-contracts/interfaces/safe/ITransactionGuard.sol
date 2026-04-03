// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Enum } from "safe-contracts/common/Enum.sol";
import { IERC165 } from "safe-contracts/interfaces/IERC165.sol";

/// @title ITransactionGuard Interface
interface ITransactionGuard is IERC165 {
    /// @notice Checks the transaction details.
    /// @dev The function needs to implement transaction validation logic.
    /// @param to The address to which the transaction is intended.
    /// @param value The native token value of the transaction in Wei.
    /// @param data The transaction data.
    /// @param operation Operation type (0 for `CALL`, 1 for `DELEGATECALL`).
    /// @param safeTxGas Gas used for the transaction.
    /// @param baseGas The base gas for the transaction.
    /// @param gasPrice The price of gas in Wei for the transaction.
    /// @param gasToken The token used to pay for gas.
    /// @param refundReceiver The address which should receive the refund.
    /// @param signatures The signatures of the transaction.
    /// @param msgSender The address of the message sender.
    function checkTransaction(
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation,
        uint256 safeTxGas,
        uint256 baseGas,
        uint256 gasPrice,
        address gasToken,
        address payable refundReceiver,
        bytes memory signatures,
        address msgSender
    ) external;

    /// @notice Checks after execution of the transaction.
    /// @dev The function needs to implement a check after the execution of the transaction.
    /// @param hash The hash of the executed transaction.
    /// @param success The status of the transaction execution.
    function checkAfterExecution(bytes32 hash, bool success) external;
}
