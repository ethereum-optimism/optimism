// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Safe
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

/// @title IUnorderedExecutionModule
/// @notice Interface for the UnorderedExecutionModule contract
interface IUnorderedExecutionModule {
    /// @notice Arguments to ModuleManager.execTransactionFromModule()
    struct ExecTransactionFromModuleParams {
        address to;
        uint256 value;
        bytes data;
        Enum.Operation operation;
    }

    /// @notice Emitted when a transaction is successfully executed
    event TransactionExecuted(
        address indexed safe,
        bytes32 indexed txHash,
        address to,
        uint256 value,
        bytes data,
        Enum.Operation operation
    );

    /// @notice Error thrown when a transaction has already been executed
    error TransactionAlreadyExecuted();

    /// @notice Error thrown when signature validation fails
    error InvalidSignature();

    /// @notice Error thrown when module execution fails
    error ModuleExecutionFailed();

    /// @notice Allows for unordered execution of transactions on a Safe
    /// @param safe The Safe contract to execute the transaction on
    /// @param params The transaction parameters
    /// @param signatures Signature data that should be verified
    /// @return success Whether the transaction was successful
    function execTransactionOnSafe(
        Safe safe,
        ExecTransactionFromModuleParams memory params,
        bytes memory signatures
    ) external payable returns (bool success);

    /// @notice Returns whether nonceless execution is enabled
    /// @return enabled Whether nonceless execution is enabled
    function isNoncelessEnabled() external pure returns (bool enabled);

    /// @notice Utility function to check if this module is enabled on a given Safe
    /// @param safe The Safe to check
    /// @return enabled Whether this module is enabled on the Safe
    function isEnabledOnSafe(Safe safe) external view returns (bool enabled);

    /// @notice Generates the transaction hash that would be used for a given transaction
    /// @param safe The Safe contract
    /// @param params The transaction parameters
    /// @return txHash The transaction hash
    /// @return perSafeTxHash The deterministic hash used for replay protection
    function getTransactionHashes(
        Safe safe,
        ExecTransactionFromModuleParams memory params
    ) external view returns (bytes32 txHash, bytes32 perSafeTxHash);

    /// @notice Returns whether a transaction has been executed
    /// @param txHash The transaction hash to check
    /// @return executed Whether the transaction has been executed
    function executedTransactions(bytes32 txHash) external view returns (bool executed);
}