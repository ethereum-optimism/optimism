// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Safe
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";

// Contracts
import { ReentrancyGuard } from "@openzeppelin/contracts-v5/utils/ReentrancyGuard.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title UnorderedExecutionModule
/// @notice Safe Module that allows for unordered execution of transactions on a Safe, by leveraging
///         the Safe's checkSignatures function. Replay prevention is handled on this module using
///         keccak256 hashes in place of the Safe's nonce. This ensures that any signatures submitted
///         to the safe will have a nonce which is so high that it is impossible for the Safe's nonce
///         to ever collide with a nonce used by this module.
/// @dev This module is intended to be installed on multiple Safe contracts, in order to avoid
///      the need to deploy distinct modules for each Safe.
contract UnorderedExecutionModule is ISemver, ReentrancyGuard {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Arguments to ModuleManager.execTransactionFromModule()
    struct ExecTransactionFromModuleParams {
        address to;
        uint256 value;
        bytes data;
        Enum.Operation operation;
    }

    /// @notice Used for replay protection
    mapping(bytes32 => bool) public executedTransactions;

    /// @notice Emitted when a transaction is successfully executed
    event TransactionExecuted(
        address indexed safe, bytes32 indexed txHash, address to, uint256 value, bytes data, Enum.Operation operation
    );

    /// @notice Error thrown when a transaction has already been executed
    error TransactionAlreadyExecuted();

    /// @notice Error thrown when signature validation fails
    error InvalidSignature();

    /// @notice Error thrown when module execution fails
    error ModuleExecutionFailed();

    /// @notice Allows for unordered execution of transactions on a Safe, by leveraging the Safe's
    /// checkSignatures function. Replay prevention is handled on this module using keccak256 hashes
    /// in place of the Safe's nonce.
    /// @param safe The Safe contract to execute the transaction on
    /// @param params The transaction parameters
    /// @param signatures Signature data that should be verified
    /// @return success Whether the transaction was successful
    function execTransactionOnSafe(
        Safe safe,
        ExecTransactionFromModuleParams memory params,
        bytes memory signatures
    )
        public
        payable
        nonReentrant
        returns (bool success)
    {
        // Generate a deterministic hash based on safe address, chain ID, and transaction params
        // Including block.chainid and block.timestamp for additional collision resistance
        bytes32 perSafeTxHash = keccak256(
            abi.encode(safe, block.chainid, params.to, params.value, params.data, params.operation, block.timestamp)
        );

        if (executedTransactions[perSafeTxHash]) {
            revert TransactionAlreadyExecuted();
        }
        executedTransactions[perSafeTxHash] = true;

        // Encode transaction data using the hash as nonce
        // We use a very high nonce value to avoid collision with regular Safe transactions
        uint256 pseudoNonce = uint256(perSafeTxHash) | (1 << 255); // Set MSB to ensure high value

        bytes memory txHashData = safe.encodeTransactionData(
            params.to,
            params.value,
            params.data,
            params.operation,
            0, // safeTxGas
            0, // baseGas
            0, // gasPrice
            address(0), // gasToken
            address(0), // refundReceiver
            pseudoNonce
        );

        bytes32 txHash = keccak256(txHashData);

        // Verify signatures using Safe's signature checking
        try safe.checkSignatures(txHash, txHashData, signatures) {
            // Signature validation succeeded
        } catch {
            revert InvalidSignature();
        }

        // Execute transaction through Safe's module system
        success = safe.execTransactionFromModule(params.to, params.value, params.data, params.operation);

        if (!success) {
            revert ModuleExecutionFailed();
        }

        emit TransactionExecuted(address(safe), txHash, params.to, params.value, params.data, params.operation);
    }

    /// @notice Returns whether nonceless execution is enabled (always true for this module)
    /// @return enabled Always returns true
    function isNoncelessEnabled() external pure returns (bool enabled) {
        return true;
    }

    /// @notice Utility function to check if this module is enabled on a given Safe
    /// @param safe The Safe to check
    /// @return enabled Whether this module is enabled on the Safe
    function isEnabledOnSafe(Safe safe) external view returns (bool enabled) {
        return ModuleManager(address(safe)).isModuleEnabled(address(this));
    }

    /// @notice Generates the transaction hash that would be used for a given transaction
    /// @param safe The Safe contract
    /// @param params The transaction parameters
    /// @return txHash The transaction hash
    /// @return perSafeTxHash The deterministic hash used for replay protection
    function getTransactionHashes(
        Safe safe,
        ExecTransactionFromModuleParams memory params
    )
        external
        view
        returns (bytes32 txHash, bytes32 perSafeTxHash)
    {
        perSafeTxHash = keccak256(
            abi.encode(safe, block.chainid, params.to, params.value, params.data, params.operation, block.timestamp)
        );

        uint256 pseudoNonce = uint256(perSafeTxHash) | (1 << 255);

        bytes memory txHashData = safe.encodeTransactionData(
            params.to,
            params.value,
            params.data,
            params.operation,
            0, // safeTxGas
            0, // baseGas
            0, // gasPrice
            address(0), // gasToken
            address(0), // refundReceiver
            pseudoNonce
        );

        txHash = keccak256(txHashData);
    }
}
